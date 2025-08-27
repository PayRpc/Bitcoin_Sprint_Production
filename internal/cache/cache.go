package cache

import (
	"context"
	"sync"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
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
}

// New creates a new cache instance
func New(maxBlocks int, logger *zap.Logger) *Cache {
	return &Cache{
		blockCache: make(map[int64]BlockCache),
		maxBlocks:   maxBlocks,
		logger:      logger,
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

// cleanupOldBlocks removes oldest blocks when cache is full
func (c *Cache) cleanupOldBlocks() {
	if len(c.blockCache) <= c.maxBlocks {
		return
	}

	// Find the oldest block
	var oldestHeight int64
	var oldestTime time.Time = time.Now()

	for height, cached := range c.blockCache {
		if cached.CachedAt.Before(oldestTime) {
			oldestTime = cached.CachedAt
			oldestHeight = height
		}
	}

	delete(c.blockCache, oldestHeight)
	c.logger.Debug("Cleaned up old block from cache", zap.Int64("height", oldestHeight))
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
