// internal/dedup/blockindex.go
package dedup

import (
	"sync"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/metrics"
	"go.uber.org/zap"
)

// BlockIndex provides enterprise-grade block deduplication with adaptive algorithms
type BlockIndex struct {
	// Legacy simple mode support
	ttl  time.Duration
	mu   sync.RWMutex
	seen map[string]time.Time

	// Enterprise features
	adaptive       *AdaptiveBlockDeduper
	enterpriseMode bool
	logger         *zap.Logger

	// Per-hash locks for concurrent processing
	lockMu sync.Mutex
	locks  map[string]*sync.Mutex

	// Cleanup management
	stop chan struct{}
}

// NewBlockIndex creates a new enterprise block index with adaptive deduplication
func NewBlockIndex(ttl time.Duration) *BlockIndex {
	return NewBlockIndexWithOptions(ttl, nil, false)
}

// NewBlockIndexWithOptions creates a new block index with enterprise options
func NewBlockIndexWithOptions(ttl time.Duration, logger *zap.Logger, enterpriseMode bool) *BlockIndex {
	bi := &BlockIndex{
		ttl:            ttl,
		seen:           make(map[string]time.Time),
		locks:          make(map[string]*sync.Mutex),
		enterpriseMode: enterpriseMode,
		logger:         logger,
		stop:           make(chan struct{}),
	}

	// Initialize adaptive deduper for enterprise mode
	if enterpriseMode {
		bi.adaptive = NewAdaptiveBlockDeduper(DefaultMaxSize, ttl, logger)
		if logger != nil {
			logger.Info("Enterprise Block Index initialized with adaptive deduplication",
				zap.Duration("base_ttl", ttl),
				zap.Bool("enterprise_mode", enterpriseMode))
		}
	} else {
		// Legacy mode cleanup
		go bi.janitor()
		if logger != nil {
			logger.Info("Basic Block Index initialized",
				zap.Duration("ttl", ttl))
		}
	}

	return bi
}

// Close gracefully shuts down the block index
func (bi *BlockIndex) Close() {
	close(bi.stop)
	if bi.adaptive != nil {
		bi.adaptive.Close()
	}
}

// TryBegin obtains the per-hash lock, then checks the deduplication cache
// Returns (end, ok). If ok==false, caller MUST NOT process.
// Call end(processed=true) when you really did work so we stamp "seen".
func (bi *BlockIndex) TryBegin(hash string) (end func(processed bool), ok bool) {
	return bi.TryBeginWithOptions(hash, time.Now(), "unknown")
}

// TryBeginWithOptions provides enterprise deduplication with network and timing context
func (bi *BlockIndex) TryBeginWithOptions(hash string, timestamp time.Time, network string, options ...DedupeOption) (end func(processed bool), ok bool) {
	if hash == "" {
		return func(bool) {}, false
	}

	// Enterprise mode uses adaptive deduplication
	if bi.enterpriseMode && bi.adaptive != nil {
		isDuplicate := bi.adaptive.Seen(hash, timestamp, network, options...)

		// Return appropriate end function
		return func(processed bool) {
			if processed && bi.logger != nil {
				bi.logger.Debug("Block processed",
					zap.String("hash", hash[:min(len(hash), 16)]),
					zap.String("network", network),
					zap.Bool("was_duplicate", isDuplicate))
			}
		}, !isDuplicate
	}

	// Legacy mode implementation
	mu := bi.getLock(hash)
	mu.Lock()

	// Check recent-seen while holding the lock to avoid races
	if bi.isRecent(hash, timestamp) {
		mu.Unlock()
		return func(bool) {}, false
	}

	// Hand back an end() that stamps seen only if actual processing happened
	return func(processed bool) {
		if processed {
			bi.mu.Lock()
			bi.seen[hash] = timestamp
			bi.mu.Unlock()
			// Update cache size metric
			bi.mu.RLock()
			metrics.DeduplicationCacheSize.Set(float64(len(bi.seen)))
			bi.mu.RUnlock()
		}
		mu.Unlock()
		// Drop the lock handle to keep map small
		bi.lockMu.Lock()
		delete(bi.locks, hash)
		bi.lockMu.Unlock()
	}, true
}

// Seen provides a simple interface for checking if a block hash has been seen
func (bi *BlockIndex) Seen(hash string, network string, options ...DedupeOption) bool {
	if bi.enterpriseMode && bi.adaptive != nil {
		return bi.adaptive.Seen(hash, time.Now(), network, options...)
	}

	// Legacy mode
	bi.mu.RLock()
	defer bi.mu.RUnlock()

	ts, ok := bi.seen[hash]
	if !ok {
		return false
	}

	return time.Now().Sub(ts) < bi.ttl
}

// GetStats returns comprehensive deduplication statistics
func (bi *BlockIndex) GetStats() map[string]interface{} {
	if bi.enterpriseMode && bi.adaptive != nil {
		return bi.adaptive.GetStats()
	}

	// Legacy stats
	bi.mu.RLock()
	defer bi.mu.RUnlock()

	return map[string]interface{}{
		"mode":         "legacy",
		"total_cached": len(bi.seen),
		"ttl_seconds":  bi.ttl.Seconds(),
	}
}

// SetPerformanceMode changes the performance optimization mode (enterprise only)
func (bi *BlockIndex) SetPerformanceMode(mode string) {
	if bi.enterpriseMode && bi.adaptive != nil {
		bi.adaptive.SetPerformanceMode(mode)
	}
}

// EnableMLOptimization enables or disables ML optimization (enterprise only)
func (bi *BlockIndex) EnableMLOptimization(enabled bool) {
	if bi.enterpriseMode && bi.adaptive != nil {
		bi.adaptive.EnableMLOptimization(enabled)
	}
}

// Legacy support methods
func (bi *BlockIndex) getLock(hash string) *sync.Mutex {
	bi.lockMu.Lock()
	defer bi.lockMu.Unlock()
	if m, ok := bi.locks[hash]; ok {
		return m
	}
	m := &sync.Mutex{}
	bi.locks[hash] = m
	return m
}

func (bi *BlockIndex) isRecent(hash string, now time.Time) bool {
	bi.mu.RLock()
	ts, ok := bi.seen[hash]
	bi.mu.RUnlock()
	return ok && now.Sub(ts) < bi.ttl
}

func (bi *BlockIndex) janitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cut := time.Now().Add(-bi.ttl)
			bi.mu.Lock()
			for h, ts := range bi.seen {
				if ts.Before(cut) {
					delete(bi.seen, h)
				}
			}
			after := len(bi.seen)
			bi.mu.Unlock()
			// Update cache size metric after cleanup
			metrics.DeduplicationCacheSize.Set(float64(after))
		case <-bi.stop:
			return
		}
	}
}
