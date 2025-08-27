package cache

import (
	"context"
	"crypto/rand"
	"sync"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/entropy"
	"go.uber.org/zap"
)

// BlockCache represents a cached block with metadata
type BlockCache struct {
	Block     blocks.BlockEvent `json:"block"`
	CachedAt  time.Time         `json:"cached_at"`
	ExpiresAt time.Time         `json:"expires_at"`
}

// Cache manages in-memory storage of blockchain data
type Cache struct {
	mu          sync.RWMutex
	latestBlock BlockCache
	blockCache  map[int64]BlockCache // height -> block
	maxBlocks   int
	logger      *zap.Logger

	// Entropy-seeded cache eviction for flat latency
	entropySeed     []byte
	evictionCounter int64
	hitCounter      int64
	missCounter     int64
}

// EntropyEvictionConfig configures entropy-based cache eviction
type EntropyEvictionConfig struct {
	SeedLength       int           `json:"seed_length"`
	UpdateInterval   time.Duration `json:"update_interval"`
	EvictionThreshold float64      `json:"eviction_threshold"` // 0.0-1.0
}

// New creates a new cache instance with turbo-optimized settings
func New(maxBlocks int, logger *zap.Logger) *Cache {
	// Initialize entropy seed for cache eviction
	seed, err := entropy.FastEntropy()
	if err != nil {
		logger.Warn("Failed to generate entropy seed for cache, using random seed", zap.Error(err))
		seed = make([]byte, 32)
		if _, err := rand.Read(seed); err != nil {
			logger.Error("Failed to generate random seed", zap.Error(err))
		}
	}

	// Increase default cache size for turbo mode
	if maxBlocks < 1000 {
		maxBlocks = 1000
	}

	return &Cache{
		blockCache:  make(map[int64]BlockCache),
		maxBlocks:   maxBlocks,
		logger:      logger,
		entropySeed: seed,
	}
}

// SetLatestBlock updates the latest block in cache
func (c *Cache) SetLatestBlock(block blocks.BlockEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expiresAt := now.Add(30 * time.Second) // Cache for 30 seconds

	cachedBlock := BlockCache{
		Block:     block,
		CachedAt:  now,
		ExpiresAt: expiresAt,
	}

	c.latestBlock = cachedBlock

	// Also store by height for historical access
	c.blockCache[int64(block.Height)] = cachedBlock

	// Clean up old blocks if we exceed maxBlocks
	if len(c.blockCache) > c.maxBlocks {
		c.cleanupOldBlocks()
	}

	c.logger.Debug("Block cached",
		zap.Uint32("height", block.Height),
		zap.String("hash", block.Hash),
		zap.Time("cached_at", now))
}

// GetLatestBlock returns the latest cached block
func (c *Cache) GetLatestBlock() (blocks.BlockEvent, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if cache is expired
	if time.Now().After(c.latestBlock.ExpiresAt) {
		return blocks.BlockEvent{}, false
	}

	return c.latestBlock.Block, true
}

// GetBlockByHeight returns a block by height if cached
func (c *Cache) GetBlockByHeight(height int64) (blocks.BlockEvent, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, exists := c.blockCache[height]
	if !exists || time.Now().After(cached.ExpiresAt) {
		return blocks.BlockEvent{}, false
	}

	return cached.Block, true
}

// IsStale returns true if the cache is stale (no recent data)
func (c *Cache) IsStale() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return time.Now().After(c.latestBlock.ExpiresAt)
}

// CacheStatus represents the current status of the cache
type CacheStatus struct {
	Enabled         bool      `json:"enabled"`
	CachedBlocks    int       `json:"cached_blocks"`
	MaxBlocks       int       `json:"max_blocks"`
	LatestHeight    int64     `json:"latest_height"`
	LatestHash      string    `json:"latest_hash"`
	LatestCachedAt  time.Time `json:"latest_cached_at"`
	IsStale         bool      `json:"is_stale"`
	StaleSeconds    float64   `json:"stale_seconds"`
	CacheHitRate    float64   `json:"cache_hit_rate"`
	TotalRequests   int64     `json:"total_requests"`
	CacheHits       int64     `json:"cache_hits"`
	CacheMisses     int64     `json:"cache_misses"`
}

// GetStatus returns the current cache status
func (c *Cache) GetStatus() CacheStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := c.GetCacheStats()

	return CacheStatus{
		Enabled:        true,
		CachedBlocks:   stats["cached_blocks"].(int),
		MaxBlocks:      stats["max_blocks"].(int),
		LatestHeight:   stats["latest_height"].(int64),
		LatestHash:     c.latestBlock.Block.Hash,
		LatestCachedAt: stats["latest_cached_at"].(time.Time),
		IsStale:        stats["is_stale"].(bool),
		StaleSeconds:   stats["stale_seconds"].(float64),
		CacheHitRate:   0.0, // TODO: implement hit rate tracking
		TotalRequests:  0,   // TODO: implement request tracking
		CacheHits:      0,   // TODO: implement hit tracking
		CacheMisses:    0,   // TODO: implement miss tracking
	}
}

// GetCacheStats returns cache statistics
func (c *Cache) GetCacheStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"cached_blocks":     len(c.blockCache),
		"max_blocks":        c.maxBlocks,
		"latest_height":     int64(c.latestBlock.Block.Height),
		"latest_cached_at":  c.latestBlock.CachedAt,
		"is_stale":          c.IsStale(),
		"stale_seconds":     time.Since(c.latestBlock.ExpiresAt).Seconds(),
	}
}

// cleanupOldBlocks removes blocks using entropy-seeded eviction for flat latency
func (c *Cache) cleanupOldBlocks() {
	c.entropySeededEviction()
	c.evictionCounter++
}

// PrefetchWorker continuously updates cache from block channel
type PrefetchWorker struct {
	cache     *Cache
	blockChan chan blocks.BlockEvent
	logger    *zap.Logger
	stopCh    chan struct{}
}

// NewPrefetchWorker creates a new prefetch worker
func NewPrefetchWorker(cache *Cache, blockChan chan blocks.BlockEvent, logger *zap.Logger) *PrefetchWorker {
	return &PrefetchWorker{
		cache:     cache,
		blockChan: blockChan,
		logger:    logger,
		stopCh:    make(chan struct{}),
	}
}

// Start begins the prefetch worker
func (w *PrefetchWorker) Start(ctx context.Context) {
	w.logger.Info("Starting prefetch worker")

	go func() {
		for {
			select {
			case <-ctx.Done():
				w.logger.Info("Prefetch worker stopped by context")
				return
			case <-w.stopCh:
				w.logger.Info("Prefetch worker stopped")
				return
			case block := <-w.blockChan:
				w.cache.SetLatestBlock(block)
				w.logger.Debug("Prefetch worker processed block",
					zap.Uint32("height", block.Height),
					zap.String("hash", block.Hash))
			}
		}
	}()
}

// Stop stops the prefetch worker
func (w *PrefetchWorker) Stop() {
	close(w.stopCh)
}

// entropySeededEviction performs entropy-seeded cache eviction for flat latency
func (c *Cache) entropySeededEviction() {
	if len(c.blockCache) <= c.maxBlocks {
		return
	}

	// Generate entropy for eviction decision
	entropyBytes, err := entropy.FastEntropy()
	if err != nil {
		// Fallback to simple random eviction
		c.simpleEviction()
		return
	}

	// Use entropy to create unpredictable eviction pattern
	heights := make([]int64, 0, len(c.blockCache))
	for height := range c.blockCache {
		heights = append(heights, height)
	}

	// Sort heights using entropy-seeded comparison
	c.entropySeededSort(heights, entropyBytes)

	// Evict blocks based on entropy-seeded priority
	evictCount := len(c.blockCache) - c.maxBlocks
	for i := 0; i < evictCount && i < len(heights); i++ {
		height := heights[i]
		delete(c.blockCache, height)
		c.logger.Debug("Entropy-seeded eviction",
			zap.Int64("evicted_height", height),
			zap.Int("remaining_blocks", len(c.blockCache)))
	}
}

// entropySeededSort sorts heights using entropy for unpredictable ordering
func (c *Cache) entropySeededSort(heights []int64, entropySeed []byte) {
	if len(heights) <= 1 {
		return
	}

	// Fisher-Yates shuffle using entropy seed
	for i := len(heights) - 1; i > 0; i-- {
		// Use entropy seed to generate random index
		seedIndex := int(entropySeed[i%len(entropySeed)]) % (i + 1)
		heights[i], heights[seedIndex] = heights[seedIndex], heights[i]
	}
}

// simpleEviction performs simple LRU-style eviction as fallback
func (c *Cache) simpleEviction() {
	if len(c.blockCache) <= c.maxBlocks {
		return
	}

	// Find oldest blocks to evict
	type heightTime struct {
		height int64
		time   time.Time
	}

	var candidates []heightTime
	for height, cached := range c.blockCache {
		candidates = append(candidates, heightTime{height: height, time: cached.CachedAt})
	}

	// Sort by cache time (oldest first)
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].time.Before(candidates[i].time) {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	// Evict oldest blocks
	evictCount := len(c.blockCache) - c.maxBlocks
	for i := 0; i < evictCount && i < len(candidates); i++ {
		height := candidates[i].height
		delete(c.blockCache, height)
		c.logger.Debug("Simple eviction",
			zap.Int64("evicted_height", height),
			zap.Int("remaining_blocks", len(c.blockCache)))
	}
}

// updateEntropySeed refreshes the entropy seed for cache eviction
func (c *Cache) updateEntropySeed() {
	newSeed, err := entropy.FastEntropy()
	if err != nil {
		c.logger.Warn("Failed to update entropy seed", zap.Error(err))
		return
	}
	c.entropySeed = newSeed
	c.logger.Debug("Updated entropy seed for cache eviction")
}

// GetDetailedCacheStats returns detailed cache statistics with entropy metrics
func (c *Cache) GetDetailedCacheStats() CacheStats {
	totalRequests := c.hitCounter + c.missCounter
	hitRate := float64(0)
	if totalRequests > 0 {
		hitRate = float64(c.hitCounter) / float64(totalRequests)
	}

	return CacheStats{
		TotalRequests:   totalRequests,
		CacheHits:       c.hitCounter,
		CacheMisses:     c.missCounter,
		CacheHitRate:    hitRate,
		EvictionCounter: c.evictionCounter,
		EntropySeedAge:  time.Duration(c.evictionCounter) * time.Millisecond, // Approximation
	}
}

// CacheStats represents detailed cache statistics
type CacheStats struct {
	TotalRequests   int64         `json:"total_requests"`
	CacheHits       int64         `json:"cache_hits"`
	CacheMisses     int64         `json:"cache_misses"`
	CacheHitRate    float64       `json:"cache_hit_rate"`
	EvictionCounter int64         `json:"eviction_counter"`
	EntropySeedAge  time.Duration `json:"entropy_seed_age"`
}
