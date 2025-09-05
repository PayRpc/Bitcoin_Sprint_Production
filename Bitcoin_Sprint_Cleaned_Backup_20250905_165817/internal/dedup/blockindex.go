// internal/dedup/blockindex.go
package dedup

import (
	"sync"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/metrics"
)

type BlockIndex struct {
	ttl   time.Duration

	mu   sync.RWMutex
	seen map[string]time.Time

	// per-hash locks so only one goroutine processes a given block at a time
	lockMu sync.Mutex
	locks  map[string]*sync.Mutex

	// janitor
	stop chan struct{}
}

func NewBlockIndex(ttl time.Duration) *BlockIndex {
	bi := &BlockIndex{
		ttl:   ttl,
		seen:  make(map[string]time.Time),
		locks: make(map[string]*sync.Mutex),
		stop:  make(chan struct{}),
	}
	go bi.janitor()
	return bi
}

func (bi *BlockIndex) Close() { close(bi.stop) }

// TryBegin obtains the per-hash lock, then checks the TTL cache.
// It returns (end, ok). If ok==false, caller MUST NOT process.
// Call end(processed=true) when you really did work so we stamp "seen".
func (bi *BlockIndex) TryBegin(hash string) (end func(processed bool), ok bool) {
	// 1) per-hash lock
	mu := bi.getLock(hash)
	mu.Lock()

	// 2) check recent-seen while holding the lock to avoid races
	if bi.isRecent(hash) {
		// Update metrics for duplicate detection
		// Note: source will be passed by caller when incrementing metrics
		// release and say "nope"
		mu.Unlock()
		return func(bool) {}, false
	}

	// 3) hand back an end() that stamps seen only if actual processing happened
	return func(processed bool) {
		if processed {
			bi.mu.Lock()
			bi.seen[hash] = time.Now()
			bi.mu.Unlock()
			// Update cache size metric
			bi.mu.RLock()
			metrics.DeduplicationCacheSize.Set(float64(len(bi.seen)))
			bi.mu.RUnlock()
		}
		mu.Unlock()
		// Optionally drop the lock handle to keep map small
		bi.lockMu.Lock()
		delete(bi.locks, hash)
		bi.lockMu.Unlock()
	}, true
}

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

func (bi *BlockIndex) isRecent(hash string) bool {
	now := time.Now()
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
			// no need to sweep locks: they're dropped in end()
		case <-bi.stop:
			return
		}
	}
}
