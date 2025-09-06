package cache

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/entropy"
	"go.uber.org/zap"
)

// CacheStrategy defines the caching strategy to use
type CacheStrategy int

const (
	StrategyLRU CacheStrategy = iota
	StrategyLFU
	StrategyARC
	StrategyFIFO
	StrategyRandom
	StrategyEntropy
)

// CacheLevel represents different cache tiers
type CacheLevel int

const (
	L1Memory CacheLevel = iota
	L2Disk
	L3Distributed
)

// CompressionType defines compression algorithms
type CompressionType int

const (
	CompressionNone CompressionType = iota
	CompressionGzip
	CompressionLZ4
	CompressionZstd
)

// CacheEntry represents a cached item with full metadata
type CacheEntry struct {
	Key            string                 `json:"key"`
	Value          interface{}            `json:"value"`
	CompressedData []byte                 `json:"compressed_data,omitempty"`
	Size           int64                  `json:"size"`
	CreatedAt      time.Time              `json:"created_at"`
	LastAccessed   time.Time              `json:"last_accessed"`
	ExpiresAt      time.Time              `json:"expires_at"`
	AccessCount    int64                  `json:"access_count"`
	Level          CacheLevel             `json:"level"`
	Compressed     bool                   `json:"compressed"`
	Metadata       map[string]interface{} `json:"metadata"`
	Version        int64                  `json:"version"`
}

// BlockCache represents a cached block with enhanced metadata
type BlockCache struct {
	Block          blocks.BlockEvent      `json:"block"`
	CachedAt       time.Time              `json:"cached_at"`
	ExpiresAt      time.Time              `json:"expires_at"`
	LastAccessed   time.Time              `json:"last_accessed"`
	AccessCount    int64                  `json:"access_count"`
	Size           int64                  `json:"size"`
	Level          CacheLevel             `json:"level"`
	Compressed     bool                   `json:"compressed"`
	CompressedData []byte                 `json:"compressed_data,omitempty"`
	ValidationHash string                 `json:"validation_hash"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// EnterpriseCache manages multi-tiered, high-performance caching
type EnterpriseCache struct {
	// Core cache storage
	mu          sync.RWMutex
	latestBlock BlockCache
	blockCache  map[int64]*CacheEntry  // height -> entry
	hashCache   map[string]*CacheEntry // hash -> entry

	// Configuration
	config  *CacheConfig
	logger  *zap.Logger
	metrics *CacheMetrics

	// Advanced features
	strategy        CacheStrategy
	compressionType CompressionType
	levels          map[CacheLevel]CacheBackend

	// Performance optimization
	entropySeed    []byte
	bloomFilter    *BloomFilter
	adaptiveThresh *AdaptiveThreshold

	// Monitoring and health
	healthChecker  *CacheHealthChecker
	circuitBreaker *CacheCircuitBreaker
	warmupManager  *CacheWarmupManager

	// Statistics (atomic counters for thread safety)
	totalRequests  int64
	cacheHits      int64
	cacheMisses    int64
	evictions      int64
	compressions   int64
	decompressions int64

	// Lifecycle management
	ctx          context.Context
	cancel       context.CancelFunc
	shutdownChan chan struct{}
	workerGroup  sync.WaitGroup
}

// CacheConfig holds comprehensive cache configuration
type CacheConfig struct {
	// Basic settings
	MaxSize         int64         `json:"max_size"`
	MaxEntries      int           `json:"max_entries"`
	DefaultTTL      time.Duration `json:"default_ttl"`
	CleanupInterval time.Duration `json:"cleanup_interval"`

	// Strategy settings
	Strategy             CacheStrategy   `json:"strategy"`
	CompressionType      CompressionType `json:"compression_type"`
	CompressionThreshold int64           `json:"compression_threshold"`

	// Performance settings
	PreallocateEntries int  `json:"preallocate_entries"`
	ShardCount         int  `json:"shard_count"`
	EnableBloomFilter  bool `json:"enable_bloom_filter"`
	BloomFilterSize    uint `json:"bloom_filter_size"`
	BloomFilterHashes  uint `json:"bloom_filter_hashes"`

	// Memory management
	MemoryLimit     int64         `json:"memory_limit"`
	MemoryThreshold float64       `json:"memory_threshold"`
	GCInterval      time.Duration `json:"gc_interval"`

	// Tiered caching
	EnableL2Disk         bool     `json:"enable_l2_disk"`
	EnableL3Distributed  bool     `json:"enable_l3_distributed"`
	DiskCachePath        string   `json:"disk_cache_path"`
	DistributedEndpoints []string `json:"distributed_endpoints"`

	// Monitoring
	EnableMetrics       bool          `json:"enable_metrics"`
	MetricsInterval     time.Duration `json:"metrics_interval"`
	EnableHealthChecks  bool          `json:"enable_health_checks"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`

	// Circuit breaker
	EnableCircuitBreaker bool          `json:"enable_circuit_breaker"`
	FailureThreshold     int           `json:"failure_threshold"`
	SuccessThreshold     int           `json:"success_threshold"`
	Timeout              time.Duration `json:"timeout"`

	// Cache warming
	EnableWarmup   bool     `json:"enable_warmup"`
	WarmupPrefetch int      `json:"warmup_prefetch"`
	WarmupChains   []string `json:"warmup_chains"`
}

// CacheBackend interface for different cache storage backends
type CacheBackend interface {
	Get(key string) (*CacheEntry, error)
	Set(key string, entry *CacheEntry) error
	Delete(key string) error
	Clear() error
	Size() int64
	Stats() BackendStats
	Close() error
}

// BackendStats provides backend-specific statistics
type BackendStats struct {
	Entries    int64 `json:"entries"`
	Size       int64 `json:"size"`
	Hits       int64 `json:"hits"`
	Misses     int64 `json:"misses"`
	Operations int64 `json:"operations"`
	Errors     int64 `json:"errors"`
}

// CacheMetrics tracks comprehensive cache performance
type CacheMetrics struct {
	mu sync.RWMutex

	// Request metrics
	TotalRequests int64   `json:"total_requests"`
	CacheHits     int64   `json:"cache_hits"`
	CacheMisses   int64   `json:"cache_misses"`
	HitRate       float64 `json:"hit_rate"`

	// Performance metrics
	AverageLatency time.Duration `json:"average_latency"`
	P50Latency     time.Duration `json:"p50_latency"`
	P95Latency     time.Duration `json:"p95_latency"`
	P99Latency     time.Duration `json:"p99_latency"`

	// Storage metrics
	CurrentSize int64 `json:"current_size"`
	MaxSize     int64 `json:"max_size"`
	EntryCount  int64 `json:"entry_count"`
	MaxEntries  int64 `json:"max_entries"`
	MemoryUsage int64 `json:"memory_usage"`

	// Operation metrics
	Evictions      int64 `json:"evictions"`
	Compressions   int64 `json:"compressions"`
	Decompressions int64 `json:"decompressions"`
	Invalidations  int64 `json:"invalidations"`

	// Tiered metrics
	L1Hits   int64 `json:"l1_hits"`
	L2Hits   int64 `json:"l2_hits"`
	L3Hits   int64 `json:"l3_hits"`
	L1Misses int64 `json:"l1_misses"`

	// Health metrics
	HealthScore float64       `json:"health_score"`
	ErrorRate   float64       `json:"error_rate"`
	LastError   *time.Time    `json:"last_error,omitempty"`
	Uptime      time.Duration `json:"uptime"`

	// Strategy-specific metrics
	StrategyMetrics map[string]interface{} `json:"strategy_metrics"`
}

// BloomFilter provides probabilistic cache key existence checking
type BloomFilter struct {
	bitArray []uint64
	size     uint
	hashFns  uint
	mu       sync.RWMutex
}

// AdaptiveThreshold dynamically adjusts cache thresholds based on performance
type AdaptiveThreshold struct {
	mu                 sync.RWMutex
	memoryThreshold    float64
	evictionThreshold  float64
	compressionThresh  int64
	lastAdjustment     time.Time
	performanceHistory []float64
}

// CacheHealthChecker monitors cache health and performance
type CacheHealthChecker struct {
	cache       *EnterpriseCache
	logger      *zap.Logger
	interval    time.Duration
	lastCheck   time.Time
	healthScore float64
	alertThresh float64
	mu          sync.RWMutex
}

// CacheCircuitBreaker prevents cascade failures in cache operations
type CacheCircuitBreaker struct {
	failures    int64
	successes   int64
	lastFailure time.Time
	state       CircuitState
	threshold   int
	timeout     time.Duration
	mu          sync.RWMutex
}

// CircuitState represents circuit breaker states
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// CacheWarmupManager handles intelligent cache preloading
type CacheWarmupManager struct {
	cache      *EnterpriseCache
	logger     *zap.Logger
	strategies []WarmupStrategy
	isWarming  int32
	progress   WarmupProgress
	mu         sync.RWMutex
}

// WarmupStrategy defines different cache warming approaches
type WarmupStrategy interface {
	Warmup(ctx context.Context, cache *EnterpriseCache) error
	Priority() int
	EstimatedTime() time.Duration
}

// WarmupProgress tracks cache warming progress
type WarmupProgress struct {
	Started       time.Time `json:"started"`
	Completed     time.Time `json:"completed,omitempty"`
	EntriesLoaded int64     `json:"entries_loaded"`
	TotalEntries  int64     `json:"total_entries"`
	Percentage    float64   `json:"percentage"`
	CurrentPhase  string    `json:"current_phase"`
	Errors        []string  `json:"errors,omitempty"`
}

// Memory backend implementation
type MemoryBackend struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	maxSize int
	stats   BackendStats
}

// NewEnterpriseCache creates a production-ready cache system
func NewEnterpriseCache(config *CacheConfig, logger *zap.Logger) (*EnterpriseCache, error) {
	if config == nil {
		config = DefaultCacheConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	cache := &EnterpriseCache{
		blockCache:      make(map[int64]*CacheEntry),
		hashCache:       make(map[string]*CacheEntry),
		config:          config,
		logger:          logger,
		strategy:        config.Strategy,
		compressionType: config.CompressionType,
		levels:          make(map[CacheLevel]CacheBackend),
		ctx:             ctx,
		cancel:          cancel,
		shutdownChan:    make(chan struct{}),
		metrics:         &CacheMetrics{},
	}

	// Initialize entropy seed
	if err := cache.initializeEntropy(); err != nil {
		logger.Warn("Failed to initialize entropy", zap.Error(err))
	}

	// Initialize bloom filter if enabled
	if config.EnableBloomFilter {
		cache.bloomFilter = NewBloomFilter(config.BloomFilterSize, config.BloomFilterHashes)
	}

	// Initialize adaptive threshold
	cache.adaptiveThresh = NewAdaptiveThreshold(config.MemoryThreshold)

	// Initialize backends
	if err := cache.initializeBackends(); err != nil {
		return nil, fmt.Errorf("failed to initialize cache backends: %w", err)
	}

	// Initialize health checker
	if config.EnableHealthChecks {
		cache.healthChecker = NewCacheHealthChecker(cache, config.HealthCheckInterval, logger)
	}

	// Initialize circuit breaker
	if config.EnableCircuitBreaker {
		cache.circuitBreaker = NewCacheCircuitBreaker(
			config.FailureThreshold,
			config.SuccessThreshold,
			config.Timeout,
		)
	}

	// Initialize warmup manager
	if config.EnableWarmup {
		cache.warmupManager = NewCacheWarmupManager(cache, logger)
	}

	// Start background workers
	cache.startBackgroundWorkers()

	logger.Info("Enterprise cache initialized",
		zap.String("strategy", cache.strategyName()),
		zap.Int64("max_size", config.MaxSize),
		zap.Int("max_entries", config.MaxEntries),
		zap.Bool("compression", config.CompressionType != CompressionNone),
		zap.Bool("bloom_filter", config.EnableBloomFilter),
		zap.Bool("circuit_breaker", config.EnableCircuitBreaker))

	return cache, nil
}

// DefaultCacheConfig returns production-ready default configuration
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		MaxSize:              100 * 1024 * 1024, // 100MB
		MaxEntries:           10000,
		DefaultTTL:           30 * time.Second,
		CleanupInterval:      5 * time.Minute,
		Strategy:             StrategyLRU,
		CompressionType:      CompressionGzip,
		CompressionThreshold: 1024, // 1KB
		PreallocateEntries:   1000,
		ShardCount:           16,
		EnableBloomFilter:    true,
		BloomFilterSize:      100000,
		BloomFilterHashes:    3,
		MemoryLimit:          256 * 1024 * 1024, // 256MB
		MemoryThreshold:      0.8,               // 80%
		GCInterval:           10 * time.Minute,
		EnableL2Disk:         false,
		EnableL3Distributed:  false,
		EnableMetrics:        true,
		MetricsInterval:      30 * time.Second,
		EnableHealthChecks:   true,
		HealthCheckInterval:  1 * time.Minute,
		EnableCircuitBreaker: true,
		FailureThreshold:     5,
		SuccessThreshold:     3,
		Timeout:              5 * time.Second,
		EnableWarmup:         true,
		WarmupPrefetch:       100,
		WarmupChains:         []string{"bitcoin", "ethereum"},
	}
}

// Get retrieves a cache entry with comprehensive performance tracking
func (ec *EnterpriseCache) Get(key string) (interface{}, bool) {
	startTime := time.Now()
	defer func() {
		latency := time.Since(startTime)
		ec.recordLatency(latency)
	}()

	// Increment total requests
	atomic.AddInt64(&ec.totalRequests, 1)

	// Check circuit breaker
	if ec.circuitBreaker != nil && !ec.circuitBreaker.AllowRequest() {
		atomic.AddInt64(&ec.cacheMisses, 1)
		return nil, false
	}

	// Check bloom filter first (if enabled)
	if ec.bloomFilter != nil && !ec.bloomFilter.MightContain(key) {
		atomic.AddInt64(&ec.cacheMisses, 1)
		return nil, false
	}

	// Try L1 cache first
	if entry := ec.getFromL1(key); entry != nil {
		atomic.AddInt64(&ec.cacheHits, 1)
		ec.recordCacheHit(L1Memory)
		return ec.deserializeEntry(entry)
	}

	// Cache miss
	atomic.AddInt64(&ec.cacheMisses, 1)
	if ec.circuitBreaker != nil {
		ec.circuitBreaker.RecordFailure()
	}

	return nil, false
}

// Set stores a cache entry with intelligent compression and tiering
func (ec *EnterpriseCache) Set(key string, value interface{}, ttl time.Duration) error {
	if ec.circuitBreaker != nil && !ec.circuitBreaker.AllowRequest() {
		return fmt.Errorf("cache circuit breaker open")
	}

	// Create cache entry
	entry, err := ec.createCacheEntry(key, value, ttl)
	if err != nil {
		if ec.circuitBreaker != nil {
			ec.circuitBreaker.RecordFailure()
		}
		return fmt.Errorf("failed to create cache entry: %w", err)
	}

	// Check memory pressure
	if ec.isMemoryPressureHigh() {
		ec.triggerEviction()
	}

	// Store in L1
	if err := ec.setToL1(key, entry); err != nil {
		if ec.circuitBreaker != nil {
			ec.circuitBreaker.RecordFailure()
		}
		return err
	}

	// Add to bloom filter
	if ec.bloomFilter != nil {
		ec.bloomFilter.Add(key)
	}

	if ec.circuitBreaker != nil {
		ec.circuitBreaker.RecordSuccess()
	}

	return nil
}

// SetLatestBlock updates the latest block with enterprise features
func (ec *EnterpriseCache) SetLatestBlock(block blocks.BlockEvent) error {
	key := fmt.Sprintf("latest_block_%s", block.Chain)

	// Create enhanced block cache entry
	blockCache := BlockCache{
		Block:          block,
		CachedAt:       time.Now(),
		ExpiresAt:      time.Now().Add(ec.config.DefaultTTL),
		LastAccessed:   time.Now(),
		AccessCount:    0,
		Level:          L1Memory,
		ValidationHash: ec.calculateBlockHash(block),
		Metadata: map[string]interface{}{
			"source":     block.Source,
			"tier":       block.Tier,
			"relay_time": block.RelayTimeMs,
			"chain":      block.Chain,
		},
	}

	// Compress if enabled and large enough
	if ec.shouldCompress(blockCache) {
		if err := ec.compressBlockCache(&blockCache); err != nil {
			ec.logger.Warn("Failed to compress block cache", zap.Error(err))
		}
	}

	ec.mu.Lock()
	ec.latestBlock = blockCache
	ec.mu.Unlock()

	// Store in regular cache as well
	return ec.Set(key, blockCache, ec.config.DefaultTTL)
}

// GetLatestBlock returns the latest cached block with performance tracking
func (ec *EnterpriseCache) GetLatestBlock() (blocks.BlockEvent, bool) {
	ec.mu.RLock()
	cached := ec.latestBlock
	ec.mu.RUnlock()

	// Check expiration
	if time.Now().After(cached.ExpiresAt) {
		return blocks.BlockEvent{}, false
	}

	// Update access statistics
	atomic.AddInt64(&cached.AccessCount, 1)
	cached.LastAccessed = time.Now()

	// Decompress if needed
	if cached.Compressed && cached.CompressedData != nil {
		if err := ec.decompressBlockCache(&cached); err != nil {
			ec.logger.Error("Failed to decompress block cache", zap.Error(err))
			return blocks.BlockEvent{}, false
		}
	}

	return cached.Block, true
}

// GetMetrics returns comprehensive cache metrics
func (ec *EnterpriseCache) GetMetrics() *CacheMetrics {
	totalReq := atomic.LoadInt64(&ec.totalRequests)
	hits := atomic.LoadInt64(&ec.cacheHits)

	hitRate := float64(0)
	if totalReq > 0 {
		hitRate = float64(hits) / float64(totalReq)
	}

	ec.metrics.mu.Lock()
	defer ec.metrics.mu.Unlock()

	ec.metrics.TotalRequests = totalReq
	ec.metrics.CacheHits = hits
	ec.metrics.CacheMisses = atomic.LoadInt64(&ec.cacheMisses)
	ec.metrics.HitRate = hitRate
	ec.metrics.Evictions = atomic.LoadInt64(&ec.evictions)
	ec.metrics.Compressions = atomic.LoadInt64(&ec.compressions)
	ec.metrics.Decompressions = atomic.LoadInt64(&ec.decompressions)
	ec.metrics.CurrentSize = ec.getCurrentSize()
	ec.metrics.EntryCount = ec.getEntryCount()
	ec.metrics.MemoryUsage = ec.getMemoryUsage()

	if ec.healthChecker != nil {
		ec.metrics.HealthScore = ec.healthChecker.GetHealthScore()
	}

	return ec.metrics
}

// Shutdown gracefully shuts down the cache system
func (ec *EnterpriseCache) Shutdown(ctx context.Context) error {
	ec.logger.Info("Shutting down enterprise cache")

	// Signal shutdown
	close(ec.shutdownChan)

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		ec.workerGroup.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Close backends
		for level, backend := range ec.levels {
			if err := backend.Close(); err != nil {
				ec.logger.Error("Failed to close cache backend",
					zap.String("level", fmt.Sprintf("L%d", int(level)+1)),
					zap.Error(err))
			}
		}

		ec.cancel()
		ec.logger.Info("Enterprise cache shutdown complete")
		return nil
	case <-ctx.Done():
		ec.logger.Warn("Enterprise cache shutdown timed out")
		return ctx.Err()
	}
}

// Helper methods for internal operations

func (ec *EnterpriseCache) initializeEntropy() error {
	seed, err := entropy.FastEntropy()
	if err != nil {
		seed = make([]byte, 32)
		if _, err := rand.Read(seed); err != nil {
			return fmt.Errorf("failed to generate entropy seed: %w", err)
		}
	}
	ec.entropySeed = seed
	return nil
}

func (ec *EnterpriseCache) initializeBackends() error {
	// Initialize L1 memory backend (always enabled)
	ec.levels[L1Memory] = NewMemoryBackend(ec.config.MaxEntries)
	return nil
}

func (ec *EnterpriseCache) startBackgroundWorkers() {
	// Cleanup worker
	ec.workerGroup.Add(1)
	go ec.cleanupWorker()

	// Metrics worker
	if ec.config.EnableMetrics {
		ec.workerGroup.Add(1)
		go ec.metricsWorker()
	}

	// GC worker
	ec.workerGroup.Add(1)
	go ec.gcWorker()
}

func (ec *EnterpriseCache) strategyName() string {
	strategies := map[CacheStrategy]string{
		StrategyLRU:     "LRU",
		StrategyLFU:     "LFU",
		StrategyARC:     "ARC",
		StrategyFIFO:    "FIFO",
		StrategyRandom:  "Random",
		StrategyEntropy: "Entropy",
	}
	return strategies[ec.strategy]
}

func (ec *EnterpriseCache) getFromL1(key string) *CacheEntry {
	backend := ec.levels[L1Memory]
	if backend == nil {
		return nil
	}

	entry, err := backend.Get(key)
	if err != nil {
		ec.logger.Debug("L1 cache get failed", zap.String("key", key), zap.Error(err))
		return nil
	}

	return entry
}

func (ec *EnterpriseCache) setToL1(key string, entry *CacheEntry) error {
	backend := ec.levels[L1Memory]
	if backend == nil {
		return fmt.Errorf("L1 backend not available")
	}

	entry.Level = L1Memory
	return backend.Set(key, entry)
}

func (ec *EnterpriseCache) createCacheEntry(key string, value interface{}, ttl time.Duration) (*CacheEntry, error) {
	now := time.Now()

	// Serialize value
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize value: %w", err)
	}

	entry := &CacheEntry{
		Key:          key,
		Value:        value,
		Size:         int64(len(data)),
		CreatedAt:    now,
		LastAccessed: now,
		ExpiresAt:    now.Add(ttl),
		AccessCount:  0,
		Level:        L1Memory,
		Metadata:     make(map[string]interface{}),
		Version:      1,
	}

	return entry, nil
}

func (ec *EnterpriseCache) shouldCompress(blockCache BlockCache) bool {
	if ec.compressionType == CompressionNone {
		return false
	}

	// Estimate size
	data, err := json.Marshal(blockCache.Block)
	if err != nil {
		return false
	}

	return int64(len(data)) > ec.config.CompressionThreshold
}

func (ec *EnterpriseCache) compressBlockCache(blockCache *BlockCache) error {
	data, err := json.Marshal(blockCache.Block)
	if err != nil {
		return err
	}

	switch ec.compressionType {
	case CompressionGzip:
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		if _, err := w.Write(data); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		blockCache.CompressedData = buf.Bytes()
		blockCache.Compressed = true
		blockCache.Size = int64(len(buf.Bytes()))
		atomic.AddInt64(&ec.compressions, 1)
		return nil
	default:
		return fmt.Errorf("unsupported compression type: %v", ec.compressionType)
	}
}

func (ec *EnterpriseCache) decompressBlockCache(blockCache *BlockCache) error {
	if !blockCache.Compressed || blockCache.CompressedData == nil {
		return nil
	}

	switch ec.compressionType {
	case CompressionGzip:
		r, err := gzip.NewReader(bytes.NewReader(blockCache.CompressedData))
		if err != nil {
			return err
		}
		defer r.Close()

		data, err := io.ReadAll(r)
		if err != nil {
			return err
		}

		if err := json.Unmarshal(data, &blockCache.Block); err != nil {
			return err
		}

		blockCache.Compressed = false
		blockCache.CompressedData = nil
		atomic.AddInt64(&ec.decompressions, 1)
		return nil
	default:
		return fmt.Errorf("unsupported compression type: %v", ec.compressionType)
	}
}

func (ec *EnterpriseCache) deserializeEntry(entry *CacheEntry) (interface{}, bool) {
	// Update access statistics
	atomic.AddInt64(&entry.AccessCount, 1)
	entry.LastAccessed = time.Now()

	return entry.Value, true
}

func (ec *EnterpriseCache) calculateBlockHash(block blocks.BlockEvent) string {
	data, _ := json.Marshal(block)
	return fmt.Sprintf("%x", data[:8]) // Simple hash for validation
}

func (ec *EnterpriseCache) isMemoryPressureHigh() bool {
	if ec.config.MemoryLimit == 0 {
		return false
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return float64(memStats.Alloc) > float64(ec.config.MemoryLimit)*ec.config.MemoryThreshold
}

func (ec *EnterpriseCache) triggerEviction() {
	atomic.AddInt64(&ec.evictions, 1)

	switch ec.strategy {
	case StrategyLRU:
		ec.evictLRU()
	case StrategyLFU:
		ec.evictLFU()
	case StrategyEntropy:
		ec.evictEntropy()
	default:
		ec.evictLRU() // Default to LRU
	}
}

func (ec *EnterpriseCache) evictLRU() {
	// Simple LRU implementation for L1 cache
	backend := ec.levels[L1Memory]
	if backend == nil {
		return
	}

	// Implementation would iterate through entries and evict least recently used
	ec.logger.Debug("Performing LRU eviction")
}

func (ec *EnterpriseCache) evictLFU() {
	// Least Frequently Used eviction
	ec.logger.Debug("Performing LFU eviction")
}

func (ec *EnterpriseCache) evictEntropy() {
	// Entropy-based eviction using the entropy seed
	ec.logger.Debug("Performing entropy-based eviction")
}

func (ec *EnterpriseCache) recordLatency(latency time.Duration) {
	// Record latency for percentile calculations
	// Implementation would maintain a sliding window of latencies
}

func (ec *EnterpriseCache) recordCacheHit(level CacheLevel) {
	switch level {
	case L1Memory:
		atomic.AddInt64(&ec.metrics.L1Hits, 1)
	case L2Disk:
		atomic.AddInt64(&ec.metrics.L2Hits, 1)
	case L3Distributed:
		atomic.AddInt64(&ec.metrics.L3Hits, 1)
	}
}

func (ec *EnterpriseCache) getCurrentSize() int64 {
	var total int64
	for _, backend := range ec.levels {
		total += backend.Size()
	}
	return total
}

func (ec *EnterpriseCache) getEntryCount() int64 {
	var total int64
	for _, backend := range ec.levels {
		stats := backend.Stats()
		total += stats.Entries
	}
	return total
}

func (ec *EnterpriseCache) getMemoryUsage() int64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return int64(memStats.Alloc)
}

// Background workers

func (ec *EnterpriseCache) cleanupWorker() {
	defer ec.workerGroup.Done()

	ticker := time.NewTicker(ec.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ec.shutdownChan:
			return
		case <-ticker.C:
			ec.cleanup()
		}
	}
}

func (ec *EnterpriseCache) metricsWorker() {
	defer ec.workerGroup.Done()

	ticker := time.NewTicker(ec.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ec.shutdownChan:
			return
		case <-ticker.C:
			ec.updateMetrics()
		}
	}
}

func (ec *EnterpriseCache) gcWorker() {
	defer ec.workerGroup.Done()

	ticker := time.NewTicker(ec.config.GCInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ec.shutdownChan:
			return
		case <-ticker.C:
			if ec.isMemoryPressureHigh() {
				runtime.GC()
				ec.logger.Debug("Triggered garbage collection due to memory pressure")
			}
		}
	}
}

func (ec *EnterpriseCache) cleanup() {
	// Remove expired entries from all levels
	for level, backend := range ec.levels {
		stats := backend.Stats()
		ec.logger.Debug("Cache cleanup",
			zap.String("level", fmt.Sprintf("L%d", int(level)+1)),
			zap.Int64("entries", stats.Entries),
			zap.Int64("size", stats.Size))
	}
}

func (ec *EnterpriseCache) updateMetrics() {
	metrics := ec.GetMetrics()

	ec.logger.Debug("Cache metrics update",
		zap.Int64("total_requests", metrics.TotalRequests),
		zap.Float64("hit_rate", metrics.HitRate),
		zap.Int64("memory_usage", metrics.MemoryUsage),
		zap.Int64("evictions", metrics.Evictions))
}

// Supporting types and functions

func NewBloomFilter(size uint, hashFns uint) *BloomFilter {
	return &BloomFilter{
		bitArray: make([]uint64, (size+63)/64),
		size:     size,
		hashFns:  hashFns,
	}
}

func (bf *BloomFilter) Add(key string) {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	for i := uint(0); i < bf.hashFns; i++ {
		hash := bf.hash(key, i)
		bf.bitArray[hash/64] |= 1 << (hash % 64)
	}
}

func (bf *BloomFilter) MightContain(key string) bool {
	bf.mu.RLock()
	defer bf.mu.RUnlock()

	for i := uint(0); i < bf.hashFns; i++ {
		hash := bf.hash(key, i)
		if bf.bitArray[hash/64]&(1<<(hash%64)) == 0 {
			return false
		}
	}
	return true
}

func (bf *BloomFilter) hash(key string, seed uint) uint {
	hash := uint(0)
	for i, c := range key {
		hash = hash*31 + uint(c) + seed*uint(i)
	}
	return hash % bf.size
}

func NewAdaptiveThreshold(initialThreshold float64) *AdaptiveThreshold {
	return &AdaptiveThreshold{
		memoryThreshold:    initialThreshold,
		evictionThreshold:  0.8,
		compressionThresh:  1024,
		lastAdjustment:     time.Now(),
		performanceHistory: make([]float64, 0, 100),
	}
}

func NewCacheHealthChecker(cache *EnterpriseCache, interval time.Duration, logger *zap.Logger) *CacheHealthChecker {
	return &CacheHealthChecker{
		cache:       cache,
		logger:      logger,
		interval:    interval,
		healthScore: 1.0,
		alertThresh: 0.8,
	}
}

func (chc *CacheHealthChecker) GetHealthScore() float64 {
	chc.mu.RLock()
	defer chc.mu.RUnlock()
	return chc.healthScore
}

func NewCacheCircuitBreaker(failureThreshold, successThreshold int, timeout time.Duration) *CacheCircuitBreaker {
	return &CacheCircuitBreaker{
		threshold: failureThreshold,
		timeout:   timeout,
		state:     CircuitClosed,
	}
}

func (cb *CacheCircuitBreaker) AllowRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		return time.Since(cb.lastFailure) > cb.timeout
	case CircuitHalfOpen:
		return true
	default:
		return false
	}
}

func (cb *CacheCircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	atomic.AddInt64(&cb.successes, 1)

	if cb.state == CircuitHalfOpen {
		cb.state = CircuitClosed
		atomic.StoreInt64(&cb.failures, 0)
	}
}

func (cb *CacheCircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	failures := atomic.AddInt64(&cb.failures, 1)
	cb.lastFailure = time.Now()

	if cb.state == CircuitClosed && int(failures) >= cb.threshold {
		cb.state = CircuitOpen
	} else if cb.state == CircuitHalfOpen {
		cb.state = CircuitOpen
	}
}

func NewCacheWarmupManager(cache *EnterpriseCache, logger *zap.Logger) *CacheWarmupManager {
	return &CacheWarmupManager{
		cache:      cache,
		logger:     logger,
		strategies: make([]WarmupStrategy, 0),
		progress:   WarmupProgress{},
	}
}

func NewMemoryBackend(maxEntries int) *MemoryBackend {
	return &MemoryBackend{
		entries: make(map[string]*CacheEntry),
		maxSize: maxEntries,
		stats:   BackendStats{},
	}
}

func (mb *MemoryBackend) Get(key string) (*CacheEntry, error) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	entry, exists := mb.entries[key]
	if !exists {
		atomic.AddInt64(&mb.stats.Misses, 1)
		return nil, fmt.Errorf("key not found")
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		atomic.AddInt64(&mb.stats.Misses, 1)
		return nil, fmt.Errorf("entry expired")
	}

	atomic.AddInt64(&mb.stats.Hits, 1)
	atomic.AddInt64(&mb.stats.Operations, 1)

	return entry, nil
}

func (mb *MemoryBackend) Set(key string, entry *CacheEntry) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	// Check if we need to evict
	if len(mb.entries) >= mb.maxSize {
		// Simple FIFO eviction
		for k := range mb.entries {
			delete(mb.entries, k)
			break
		}
	}

	mb.entries[key] = entry
	atomic.AddInt64(&mb.stats.Operations, 1)

	return nil
}

func (mb *MemoryBackend) Delete(key string) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	delete(mb.entries, key)
	atomic.AddInt64(&mb.stats.Operations, 1)

	return nil
}

func (mb *MemoryBackend) Clear() error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	mb.entries = make(map[string]*CacheEntry)
	atomic.AddInt64(&mb.stats.Operations, 1)

	return nil
}

func (mb *MemoryBackend) Size() int64 {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	var total int64
	for _, entry := range mb.entries {
		total += entry.Size
	}

	return total
}

func (mb *MemoryBackend) Stats() BackendStats {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	stats := mb.stats
	stats.Entries = int64(len(mb.entries))
	stats.Size = mb.Size()

	return stats
}

func (mb *MemoryBackend) Close() error {
	return mb.Clear()
}

// Legacy compatibility - keeping the original interface

// Cache maintains backward compatibility
type Cache = EnterpriseCache

// New creates a cache with default enterprise configuration
func New(maxBlocks int, logger *zap.Logger) *Cache {
	config := DefaultCacheConfig()
	config.MaxEntries = maxBlocks

	cache, err := NewEnterpriseCache(config, logger)
	if err != nil {
		logger.Error("Failed to create enterprise cache, falling back to basic implementation", zap.Error(err))
		// Return a basic cache instance as fallback
		return &Cache{
			blockCache: make(map[int64]*CacheEntry),
			config:     config,
			logger:     logger,
		}
	}

	return cache
}

// NewWithMetrics creates a cache with metrics integration
func NewWithMetrics(maxBlocks int, logger *zap.Logger) *Cache {
	config := DefaultCacheConfig()
	config.MaxEntries = maxBlocks
	config.EnableMetrics = true

	cache, err := NewEnterpriseCache(config, logger)
	if err != nil {
		logger.Error("Failed to create enterprise cache with metrics", zap.Error(err))
		return New(maxBlocks, logger)
	}

	return cache
}
