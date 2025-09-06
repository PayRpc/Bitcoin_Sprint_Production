package mempool

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// MempoolMetrics represents Prometheus metrics for mempool operations
type MempoolMetrics struct {
	TotalTransactions   prometheus.Counter
	ActiveTransactions  prometheus.Gauge
	ExpiredTransactions prometheus.Counter
	AddDuration         prometheus.Histogram
	CleanupDuration     prometheus.Histogram
	MemoryUsage         prometheus.Gauge
}

// NewMempoolMetrics creates new mempool metrics
func NewMempoolMetrics(reg prometheus.Registerer) *MempoolMetrics {
	m := &MempoolMetrics{
		TotalTransactions: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "mempool_total_transactions",
			Help: "Total number of transactions added to mempool",
		}),
		ActiveTransactions: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mempool_active_transactions",
			Help: "Current number of active transactions in mempool",
		}),
		ExpiredTransactions: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "mempool_expired_transactions",
			Help: "Total number of expired transactions removed from mempool",
		}),
		AddDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "mempool_add_duration_seconds",
			Help:    "Time taken to add transaction to mempool",
			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 10),
		}),
		CleanupDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "mempool_cleanup_duration_seconds",
			Help:    "Time taken for mempool cleanup operations",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
		}),
		MemoryUsage: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mempool_memory_usage_bytes",
			Help: "Estimated memory usage of mempool",
		}),
	}

	if reg != nil {
		reg.MustRegister(m.TotalTransactions, m.ActiveTransactions, m.ExpiredTransactions,
			m.AddDuration, m.CleanupDuration, m.MemoryUsage)
	}

	return m
}

// Config represents mempool configuration
type Config struct {
	MaxSize         int           `yaml:"max_size" json:"max_size"`
	ExpiryTime      time.Duration `yaml:"expiry_time" json:"expiry_time"`
	CleanupInterval time.Duration `yaml:"cleanup_interval" json:"cleanup_interval"`
	ShardCount      int           `yaml:"shard_count" json:"shard_count"`
}

// DefaultConfig returns default mempool configuration
func DefaultConfig() Config {
	return Config{
		MaxSize:         100000,
		ExpiryTime:      5 * time.Minute,
		CleanupInterval: 30 * time.Second,
		ShardCount:      16,
	}
}

// TransactionEntry represents a transaction in the mempool
type TransactionEntry struct {
	TxID       string    `json:"txid"`
	AddedAt    time.Time `json:"added_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Size       int       `json:"size"`
	Priority   int       `json:"priority"`
	FeeRate    float64   `json:"fee_rate"`
}

// Shard represents a single shard of the mempool for concurrent access
type Shard struct {
	mu    sync.RWMutex
	items map[string]*TransactionEntry
}

// Mempool represents an enterprise-grade transaction mempool with sharding and metrics
type Mempool struct {
	config     Config
	shards     []*Shard
	shardCount int
	size       int64
	metrics    *MempoolMetrics
	logger     *zap.Logger
	
	// Lifecycle management
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	stopped    int32
}

// New creates a new mempool with default configuration
func New() *Mempool {
	return NewWithConfig(DefaultConfig())
}

// NewWithConfig creates a new mempool with specified configuration
func NewWithConfig(config Config) *Mempool {
	return NewWithMetricsAndConfig(config, nil)
}

// NewWithMetricsAndConfig creates a new mempool with configuration and metrics
func NewWithMetricsAndConfig(config Config, metrics *MempoolMetrics) *Mempool {
	if config.ShardCount <= 0 {
		config.ShardCount = 16
	}
	
	if config.MaxSize <= 0 {
		config.MaxSize = 100000
	}
	
	if config.ExpiryTime <= 0 {
		config.ExpiryTime = 5 * time.Minute
	}
	
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 30 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	shards := make([]*Shard, config.ShardCount)
	for i := range shards {
		shards[i] = &Shard{
			items: make(map[string]*TransactionEntry),
		}
	}

	logger, _ := zap.NewProduction()
	
	m := &Mempool{
		config:     config,
		shards:     shards,
		shardCount: config.ShardCount,
		size:       0,
		metrics:    metrics,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}

	// Only start cleanup goroutine if cleanup interval is reasonable
	if config.CleanupInterval < 10*time.Minute {
		m.wg.Add(1)
		go m.cleanupLoop()
	}

	return m
}

// getShard returns the shard for a given transaction ID
func (m *Mempool) getShard(txid string) *Shard {
	// Simple hash function for shard selection
	hash := uint32(0)
	for _, b := range []byte(txid) {
		hash = hash*31 + uint32(b)
	}
	return m.shards[hash%uint32(m.shardCount)]
}

// Add adds a transaction to the mempool
func (m *Mempool) Add(txid string) {
	m.AddWithDetails(txid, 0, 0, 0.0)
}

// AddWithDetails adds a transaction with additional details to the mempool
func (m *Mempool) AddWithDetails(txid string, size int, priority int, feeRate float64) {
	if atomic.LoadInt32(&m.stopped) == 1 {
		return
	}

	start := time.Now()
	defer func() {
		if m.metrics != nil {
			m.metrics.AddDuration.Observe(time.Since(start).Seconds())
		}
	}()

	shard := m.getShard(txid)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	now := time.Now()
	entry := &TransactionEntry{
		TxID:      txid,
		AddedAt:   now,
		ExpiresAt: now.Add(m.config.ExpiryTime),
		Size:      size,
		Priority:  priority,
		FeeRate:   feeRate,
	}

	// Check if already exists
	if _, exists := shard.items[txid]; !exists {
		// Check size limit
		currentSize := atomic.LoadInt64(&m.size)
		if currentSize >= int64(m.config.MaxSize) {
			m.logger.Warn("Mempool at capacity, rejecting transaction",
				zap.String("txid", txid),
				zap.Int64("current_size", currentSize),
				zap.Int("max_size", m.config.MaxSize))
			return
		}

		atomic.AddInt64(&m.size, 1)
		if m.metrics != nil {
			m.metrics.TotalTransactions.Inc()
			m.metrics.ActiveTransactions.Set(float64(atomic.LoadInt64(&m.size)))
			m.metrics.MemoryUsage.Add(float64(size + len(txid)*2 + 64)) // Approximate memory usage
		}
	}

	shard.items[txid] = entry
}

// Contains checks if a transaction exists in the mempool
func (m *Mempool) Contains(txid string) bool {
	shard := m.getShard(txid)
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	entry, exists := shard.items[txid]
	if !exists {
		return false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		return false
	}

	return true
}

// Get retrieves a transaction entry from the mempool
func (m *Mempool) Get(txid string) (*TransactionEntry, bool) {
	shard := m.getShard(txid)
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	entry, exists := shard.items[txid]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}

	// Return a copy to avoid race conditions
	entryCopy := *entry
	return &entryCopy, true
}

// Remove removes a transaction from the mempool
func (m *Mempool) Remove(txid string) bool {
	shard := m.getShard(txid)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	entry, exists := shard.items[txid]
	if !exists {
		return false
	}

	delete(shard.items, txid)
	atomic.AddInt64(&m.size, -1)

	if m.metrics != nil {
		m.metrics.ActiveTransactions.Set(float64(atomic.LoadInt64(&m.size)))
		m.metrics.MemoryUsage.Sub(float64(entry.Size + len(txid)*2 + 64))
	}

	return true
}

// All returns all transaction IDs in the mempool
func (m *Mempool) All() []string {
	var txids []string
	now := time.Now()

	for _, shard := range m.shards {
		shard.mu.RLock()
		for _, entry := range shard.items {
			if now.Before(entry.ExpiresAt) {
				txids = append(txids, entry.TxID)
			}
		}
		shard.mu.RUnlock()
	}

	return txids
}

// AllEntries returns all transaction entries in the mempool
func (m *Mempool) AllEntries() []*TransactionEntry {
	var entries []*TransactionEntry
	now := time.Now()

	for _, shard := range m.shards {
		shard.mu.RLock()
		for _, entry := range shard.items {
			if now.Before(entry.ExpiresAt) {
				entryCopy := *entry
				entries = append(entries, &entryCopy)
			}
		}
		shard.mu.RUnlock()
	}

	return entries
}

// Size returns the current number of transactions in the mempool
func (m *Mempool) Size() int {
	return int(atomic.LoadInt64(&m.size))
}

// Stats returns mempool statistics
func (m *Mempool) Stats() map[string]interface{} {
	stats := map[string]interface{}{
		"size":             m.Size(),
		"max_size":         m.config.MaxSize,
		"shard_count":      m.shardCount,
		"expiry_time":      m.config.ExpiryTime.String(),
		"cleanup_interval": m.config.CleanupInterval.String(),
	}

	// Add per-shard statistics
	shardStats := make([]map[string]interface{}, m.shardCount)
	for i, shard := range m.shards {
		shard.mu.RLock()
		shardStats[i] = map[string]interface{}{
			"shard_id": i,
			"size":     len(shard.items),
		}
		shard.mu.RUnlock()
	}
	stats["shards"] = shardStats

	return stats
}

// cleanupLoop runs periodic cleanup of expired transactions
func (m *Mempool) cleanupLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.cleanup()
		}
	}
}

// cleanup removes expired transactions from all shards
func (m *Mempool) cleanup() {
	start := time.Now()
	defer func() {
		if m.metrics != nil {
			m.metrics.CleanupDuration.Observe(time.Since(start).Seconds())
		}
	}()

	now := time.Now()
	totalExpired := 0

	for _, shard := range m.shards {
		shard.mu.Lock()
		expired := make([]string, 0)

		for txid, entry := range shard.items {
			if now.After(entry.ExpiresAt) {
				expired = append(expired, txid)
			}
		}

		for _, txid := range expired {
			entry := shard.items[txid]
			delete(shard.items, txid)
			totalExpired++

			if m.metrics != nil {
				m.metrics.MemoryUsage.Sub(float64(entry.Size + len(txid)*2 + 64))
			}
		}

		shard.mu.Unlock()
	}

	if totalExpired > 0 {
		atomic.AddInt64(&m.size, -int64(totalExpired))
		if m.metrics != nil {
			m.metrics.ExpiredTransactions.Add(float64(totalExpired))
			m.metrics.ActiveTransactions.Set(float64(atomic.LoadInt64(&m.size)))
		}

		m.logger.Debug("Mempool cleanup completed",
			zap.Int("expired_count", totalExpired),
			zap.Duration("duration", time.Since(start)),
			zap.Int("current_size", m.Size()))
	}
}

// Clear removes all transactions from the mempool
func (m *Mempool) Clear() {
	for _, shard := range m.shards {
		shard.mu.Lock()
		for txid, entry := range shard.items {
			if m.metrics != nil {
				m.metrics.MemoryUsage.Sub(float64(entry.Size + len(txid)*2 + 64))
			}
		}
		shard.items = make(map[string]*TransactionEntry)
		shard.mu.Unlock()
	}

	atomic.StoreInt64(&m.size, 0)
	if m.metrics != nil {
		m.metrics.ActiveTransactions.Set(0)
	}

	m.logger.Info("Mempool cleared")
}

// Stop gracefully stops the mempool
func (m *Mempool) Stop() error {
	if !atomic.CompareAndSwapInt32(&m.stopped, 0, 1) {
		return fmt.Errorf("mempool already stopped")
	}

	m.cancel()
	m.wg.Wait()

	m.logger.Info("Mempool stopped gracefully")
	return nil
}
