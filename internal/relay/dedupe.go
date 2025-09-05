package relay

import (
	"sync"
	"time"

	"go.uber.org/zap"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/PayRpc/Bitcoin-Sprint/internal/dedup"
)

// Enterprise deduplication metrics
var (
	duplicateBlocksSuppressed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "relay_duplicate_blocks_suppressed_total",
		Help: "Number of duplicate block announcements dropped by the deduper",
	}, []string{"network", "source", "tier"})
	
	deduplicationProcessingLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "relay_deduplication_processing_duration_seconds",
		Help:    "Time spent processing deduplication requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"network", "operation"})
	
	deduplicationCacheHitRate = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "relay_deduplication_cache_hit_rate",
		Help: "Cache hit rate for deduplication system",
	}, []string{"network"})
	
	deduplicationMemoryPressure = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "relay_deduplication_memory_pressure",
		Help: "Memory pressure of deduplication cache (0.0-1.0)",
	}, []string{"network"})
)

// BlockDeduper provides enterprise-grade block deduplication with advanced features
type BlockDeduper struct {
	// Core deduplication
	mu    sync.RWMutex
	set   map[string]time.Time
	order []string
	cap   int
	ttl   time.Duration
	
	// Enterprise features
	adaptive          *dedup.AdaptiveBlockDeduper
	enterpriseMode    bool
	logger            *zap.Logger
	tier              string
	
	// Performance tracking
	totalRequests     int64
	duplicatesFound   int64
	avgProcessingTime time.Duration
	
	// Network-specific optimizations
	networkConfigs    map[string]*NetworkConfig
	
	// Advanced features
	crossNetworkDedup bool
	intelligentEviction bool
	priorityHandling  bool
}

// NetworkConfig holds network-specific deduplication configuration
type NetworkConfig struct {
	TTL              time.Duration `json:"ttl"`
	Capacity         int           `json:"capacity"`
	Priority         int           `json:"priority"`
	OptimizationLevel int          `json:"optimization_level"`
	CrossNetworkEnabled bool       `json:"cross_network_enabled"`
}

// NewBlockDeduper creates a new enterprise-grade deduplication handler
func NewBlockDeduper(capacity int, ttl time.Duration) *BlockDeduper {
	return NewBlockDeduperWithOptions(capacity, ttl, nil, "FREE", false)
}

// NewBlockDeduperWithOptions creates a new deduper with enterprise options
func NewBlockDeduperWithOptions(capacity int, ttl time.Duration, logger *zap.Logger, tier string, enterpriseMode bool) *BlockDeduper {
	if capacity <= 0 {
		capacity = getTierCapacity(tier)
	}
	if ttl <= 0 {
		ttl = getTierTTL(tier)
	}
	
	bd := &BlockDeduper{
		set:                 make(map[string]time.Time, capacity),
		order:               make([]string, 0, capacity),
		cap:                 capacity,
		ttl:                 ttl,
		enterpriseMode:      enterpriseMode,
		logger:              logger,
		tier:                tier,
		networkConfigs:      make(map[string]*NetworkConfig),
		crossNetworkDedup:   tier == "ENTERPRISE",
		intelligentEviction: tier != "FREE",
		priorityHandling:    tier == "ENTERPRISE" || tier == "BUSINESS",
	}
	
	// Initialize adaptive deduper for enterprise mode
	if enterpriseMode && (tier == "ENTERPRISE" || tier == "BUSINESS") {
		bd.adaptive = dedup.NewAdaptiveBlockDeduper(capacity*2, ttl, logger)
		
		// Configure performance mode based on tier
		switch tier {
		case "ENTERPRISE":
			bd.adaptive.SetPerformanceMode(dedup.PerformanceModeHighPerformance)
			bd.adaptive.EnableMLOptimization(true)
		case "BUSINESS":
			bd.adaptive.SetPerformanceMode(dedup.PerformanceModeStandard)
			bd.adaptive.EnableMLOptimization(false)
		}
	}
	
	// Initialize network configurations
	bd.initializeNetworkConfigs()
	
	if logger != nil {
		logger.Info("Enterprise Block Deduper initialized",
			zap.Int("capacity", capacity),
			zap.Duration("ttl", ttl),
			zap.String("tier", tier),
			zap.Bool("enterprise_mode", enterpriseMode),
			zap.Bool("cross_network", bd.crossNetworkDedup))
	}
	
	return bd
}

// getTierCapacity returns appropriate capacity based on service tier
func getTierCapacity(tier string) int {
	switch tier {
	case "FREE":
		return 2048
	case "BUSINESS":
		return 8192
	case "ENTERPRISE":
		return 20480
	default:
		return 4096
	}
}

// getTierTTL returns appropriate TTL based on service tier
func getTierTTL(tier string) time.Duration {
	switch tier {
	case "FREE":
		return 5 * time.Minute
	case "BUSINESS":
		return 10 * time.Minute
	case "ENTERPRISE":
		return 15 * time.Minute
	default:
		return 10 * time.Minute
	}
}

// initializeNetworkConfigs sets up network-specific configurations
func (bd *BlockDeduper) initializeNetworkConfigs() {
	networks := []string{"bitcoin", "ethereum", "solana", "polygon", "avalanche", "bsc"}
	
	for _, network := range networks {
		bd.networkConfigs[network] = &NetworkConfig{
			TTL:                 bd.getTTLForNetwork(network),
			Capacity:            bd.cap / len(networks),
			Priority:            bd.getPriorityForNetwork(network),
			OptimizationLevel:   1,
			CrossNetworkEnabled: bd.crossNetworkDedup,
		}
	}
}

// getTTLForNetwork returns network-specific TTL
func (bd *BlockDeduper) getTTLForNetwork(network string) time.Duration {
	baseTTL := bd.ttl
	
	// Network-specific adjustments
	switch network {
	case "bitcoin":
		return baseTTL * 2 // Bitcoin blocks are slower
	case "ethereum":
		return baseTTL
	case "solana":
		return baseTTL / 3 // Solana blocks are much faster
	case "polygon":
		return baseTTL / 2 // Polygon is faster than Ethereum
	default:
		return baseTTL
	}
}

// getPriorityForNetwork returns network-specific priority
func (bd *BlockDeduper) getPriorityForNetwork(network string) int {
	switch network {
	case "bitcoin":
		return 10 // Highest priority
	case "ethereum":
		return 8
	case "solana":
		return 6
	case "polygon":
		return 4
	default:
		return 1
	}
}

// Seen returns true if hash is already seen within TTL with enterprise features
func (bd *BlockDeduper) Seen(hash string, now time.Time, network string, options ...dedup.DedupeOption) bool {
	start := time.Now()
	defer func() {
		bd.avgProcessingTime = time.Since(start)
		deduplicationProcessingLatency.WithLabelValues(network, "seen").Observe(time.Since(start).Seconds())
	}()
	
	// Input validation
	if bd == nil {
		return false // If no deduper, never consider it a duplicate
	}
	
	if hash == "" {
		return false // Empty hashes are never considered duplicates
	}
	
	bd.totalRequests++
	
	// Use adaptive deduplication for enterprise/business tiers
	if bd.enterpriseMode && bd.adaptive != nil {
		isDuplicate := bd.adaptive.Seen(hash, now, network, options...)
		if isDuplicate {
			bd.duplicatesFound++
			duplicateBlocksSuppressed.WithLabelValues(network, bd.getSourceFromOptions(options), bd.tier).Inc()
		}
		
		// Update cache hit rate
		hitRate := float64(bd.duplicatesFound) / float64(bd.totalRequests)
		deduplicationCacheHitRate.WithLabelValues(network).Set(hitRate)
		
		return isDuplicate
	}
	
	// Legacy mode with enhanced features
	return bd.seenLegacy(hash, now, network, options)
}

// seenLegacy provides enhanced legacy deduplication with network awareness
func (bd *BlockDeduper) seenLegacy(hash string, now time.Time, network string, options []dedup.DedupeOption) bool {
	bd.mu.Lock()
	defer bd.mu.Unlock()
	
	// Get network-specific configuration
	netConfig := bd.networkConfigs[network]
	if netConfig == nil {
		netConfig = &NetworkConfig{
			TTL:      bd.ttl,
			Capacity: bd.cap,
			Priority: 1,
		}
	}
	
	// Generate key (with cross-network support)
	key := bd.generateKey(hash, network, netConfig)
	
	if ts, ok := bd.set[key]; ok {
		if now.Sub(ts) <= netConfig.TTL {
			bd.duplicatesFound++
			duplicateBlocksSuppressed.WithLabelValues(network, bd.getSourceFromOptions(options), bd.tier).Inc()
			return true
		}
		// Expired - treat as new (below will refresh ts)
	}
	
	// Record new entry
	bd.set[key] = now
	bd.order = append(bd.order, key)
	
	// Intelligent eviction based on priority and tier
	if len(bd.order) > bd.cap {
		bd.evictOldest(network)
	}
	
	// Update memory pressure metric
	pressure := float64(len(bd.set)) / float64(bd.cap)
	deduplicationMemoryPressure.WithLabelValues(network).Set(pressure)
	
	return false
}

// generateKey creates appropriate keys based on configuration
func (bd *BlockDeduper) generateKey(hash, network string, config *NetworkConfig) string {
	if config.CrossNetworkEnabled && bd.crossNetworkDedup {
		return hash // Cross-network deduplication
	}
	return network + ":" + hash // Network-specific
}

// evictOldest implements intelligent eviction based on priority and age
func (bd *BlockDeduper) evictOldest(currentNetwork string) {
	if len(bd.order) == 0 {
		return
	}
	
	if bd.intelligentEviction && bd.priorityHandling {
		// Find lowest priority entry to evict
		bd.evictByPriority(currentNetwork)
	} else {
		// Simple FIFO eviction
		oldKey := bd.order[0]
		bd.order = bd.order[1:]
		delete(bd.set, oldKey)
	}
}

// evictByPriority implements priority-based eviction
func (bd *BlockDeduper) evictByPriority(currentNetwork string) {
	currentPriority := bd.getPriorityForNetwork(currentNetwork)
	
	// Find entries with lower priority than current network
	for i, key := range bd.order {
		network := bd.extractNetworkFromKey(key)
		if bd.getPriorityForNetwork(network) < currentPriority {
			// Remove this lower-priority entry
			bd.order = append(bd.order[:i], bd.order[i+1:]...)
			delete(bd.set, key)
			return
		}
	}
	
	// If no lower priority found, remove oldest
	oldKey := bd.order[0]
	bd.order = bd.order[1:]
	delete(bd.set, oldKey)
}

// extractNetworkFromKey extracts network from a composite key
func (bd *BlockDeduper) extractNetworkFromKey(key string) string {
	if bd.crossNetworkDedup {
		return "cross-network"
	}
	
	for network := range bd.networkConfigs {
		if len(key) > len(network)+1 && key[:len(network)] == network && key[len(network)] == ':' {
			return network
		}
	}
	return "unknown"
}

// getSourceFromOptions extracts source from deduplication options
func (bd *BlockDeduper) getSourceFromOptions(options []dedup.DedupeOption) string {
	opts := &dedup.DedupeOptions{}
	for _, opt := range options {
		opt(opts)
	}
	if opts.Source != "" {
		return opts.Source
	}
	return "unknown"
}

// Cleanup removes expired entries with enhanced intelligence
func (bd *BlockDeduper) Cleanup() {
	if bd == nil {
		return
	}
	
	start := time.Now()
	defer func() {
		deduplicationProcessingLatency.WithLabelValues("all", "cleanup").Observe(time.Since(start).Seconds())
	}()
	
	// Use adaptive cleanup for enterprise mode
	if bd.enterpriseMode && bd.adaptive != nil {
		// Adaptive deduper handles its own cleanup
		return
	}
	
	// Enhanced legacy cleanup
	bd.cleanupLegacy()
}

// cleanupLegacy performs enhanced cleanup for legacy mode
func (bd *BlockDeduper) cleanupLegacy() {
	now := time.Now()
	bd.mu.Lock()
	defer bd.mu.Unlock()
	
	if len(bd.order) == 0 {
		return
	}
	
	// Network-aware cleanup
	w := 0
	for _, key := range bd.order {
		if ts, ok := bd.set[key]; ok {
			network := bd.extractNetworkFromKey(key)
			netConfig := bd.networkConfigs[network]
			ttl := bd.ttl
			if netConfig != nil {
				ttl = netConfig.TTL
			}
			
			if now.Sub(ts) <= ttl {
				bd.order[w] = key
				w++
				continue
			}
		}
		delete(bd.set, key)
	}
	bd.order = bd.order[:w]
	
	if bd.logger != nil && w < len(bd.order) {
		bd.logger.Debug("Legacy cleanup completed",
			zap.Int("removed", len(bd.order)-w),
			zap.Int("remaining", w))
	}
}

// GetStats returns comprehensive deduplication statistics
func (bd *BlockDeduper) GetStats() map[string]interface{} {
	if bd.enterpriseMode && bd.adaptive != nil {
		stats := bd.adaptive.GetStats()
		// Add relay-specific stats
		stats["tier"] = bd.tier
		stats["total_requests"] = bd.totalRequests
		stats["duplicates_found"] = bd.duplicatesFound
		stats["avg_processing_time_ms"] = bd.avgProcessingTime.Milliseconds()
		return stats
	}
	
	// Legacy stats
	bd.mu.RLock()
	defer bd.mu.RUnlock()
	
	hitRate := 0.0
	if bd.totalRequests > 0 {
		hitRate = float64(bd.duplicatesFound) / float64(bd.totalRequests)
	}
	
	return map[string]interface{}{
		"mode":                    "legacy",
		"tier":                    bd.tier,
		"total_cached":            len(bd.set),
		"capacity":                bd.cap,
		"ttl_seconds":             bd.ttl.Seconds(),
		"total_requests":          bd.totalRequests,
		"duplicates_found":        bd.duplicatesFound,
		"hit_rate":                hitRate,
		"avg_processing_time_ms":  bd.avgProcessingTime.Milliseconds(),
		"cross_network_enabled":   bd.crossNetworkDedup,
		"intelligent_eviction":    bd.intelligentEviction,
		"priority_handling":       bd.priorityHandling,
	}
}

// SetTier updates the service tier and reconfigures accordingly
func (bd *BlockDeduper) SetTier(tier string) {
	bd.mu.Lock()
	defer bd.mu.Unlock()
	
	oldTier := bd.tier
	bd.tier = tier
	
	// Update tier-dependent features
	bd.crossNetworkDedup = tier == "ENTERPRISE"
	bd.intelligentEviction = tier != "FREE"
	bd.priorityHandling = tier == "ENTERPRISE" || tier == "BUSINESS"
	
	// Update capacity if needed
	newCap := getTierCapacity(tier)
	if newCap != bd.cap {
		bd.cap = newCap
		// Trigger cleanup if over capacity
		if len(bd.set) > bd.cap {
			bd.enforceCapacity()
		}
	}
	
	// Update adaptive deduper if exists
	if bd.adaptive != nil {
		switch tier {
		case "ENTERPRISE":
			bd.adaptive.SetPerformanceMode(dedup.PerformanceModeHighPerformance)
			bd.adaptive.EnableMLOptimization(true)
		case "BUSINESS":
			bd.adaptive.SetPerformanceMode(dedup.PerformanceModeStandard)
			bd.adaptive.EnableMLOptimization(false)
		case "FREE":
			bd.adaptive.SetPerformanceMode(dedup.PerformanceModeMemoryOptimized)
			bd.adaptive.EnableMLOptimization(false)
		}
	}
	
	if bd.logger != nil {
		bd.logger.Info("Service tier updated",
			zap.String("old_tier", oldTier),
			zap.String("new_tier", tier),
			zap.Int("new_capacity", newCap))
	}
}

// enforceCapacity reduces cache size to fit within capacity limits
func (bd *BlockDeduper) enforceCapacity() {
	for len(bd.set) > bd.cap && len(bd.order) > 0 {
		oldKey := bd.order[0]
		bd.order = bd.order[1:]
		delete(bd.set, oldKey)
	}
}

// UpdateNetworkConfig updates configuration for a specific network
func (bd *BlockDeduper) UpdateNetworkConfig(network string, config *NetworkConfig) {
	bd.mu.Lock()
	defer bd.mu.Unlock()
	
	bd.networkConfigs[network] = config
	
	if bd.logger != nil {
		bd.logger.Info("Network configuration updated",
			zap.String("network", network),
			zap.Duration("ttl", config.TTL),
			zap.Int("priority", config.Priority))
	}
}

// GetNetworkConfig returns configuration for a specific network
func (bd *BlockDeduper) GetNetworkConfig(network string) *NetworkConfig {
	bd.mu.RLock()
	defer bd.mu.RUnlock()
	
	if config, exists := bd.networkConfigs[network]; exists {
		// Return a copy to prevent external modification
		return &NetworkConfig{
			TTL:                 config.TTL,
			Capacity:            config.Capacity,
			Priority:            config.Priority,
			OptimizationLevel:   config.OptimizationLevel,
			CrossNetworkEnabled: config.CrossNetworkEnabled,
		}
	}
	
	return nil
}

// Close gracefully shuts down the deduper
func (bd *BlockDeduper) Close() error {
	if bd.adaptive != nil {
		return bd.adaptive.Close()
	}
	return nil
}
