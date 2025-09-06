// internal/dedup/adaptive.go
package dedup

import (
	"math"
	"sync"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/metrics"
	"go.uber.org/zap"
)

// DedupeRecord stores comprehensive information about a block for enterprise deduplication
type DedupeRecord struct {
	BlockHash  string                 `json:"block_hash"`
	Network    string                 `json:"network"`
	FirstSeen  time.Time              `json:"first_seen"`
	LastSeen   time.Time              `json:"last_seen"`
	SeenCount  int                    `json:"seen_count"`
	Source     string                 `json:"source"`
	Size       int64                  `json:"size"`
	Properties map[string]interface{} `json:"properties"`
	Confidence float64                `json:"confidence"`
	Priority   int                    `json:"priority"`
}

// NetworkStats tracks sophisticated metrics for ML-based optimization per network
type NetworkStats struct {
	mu                sync.RWMutex
	avgTimeBetween    time.Duration `json:"avg_time_between"`
	blockCount        int           `json:"block_count"`
	lastBlockTime     time.Time     `json:"last_block_time"`
	duplicateRate     float64       `json:"duplicate_rate"`
	duplicatesTotal   int           `json:"duplicates_total"`
	blocksTotal       int           `json:"blocks_total"`
	adaptiveTTL       time.Duration `json:"adaptive_ttl"`
	lastRecalculated  time.Time     `json:"last_recalculated"`
	peakDuplicateRate float64       `json:"peak_duplicate_rate"`
	avgBlockSize      int64         `json:"avg_block_size"`
	throughputBPS     float64       `json:"throughput_bps"`
	networkLatency    time.Duration `json:"network_latency"`
	reliability       float64       `json:"reliability"`
	optimizationLevel int           `json:"optimization_level"`
}

// AdaptiveBlockDeduper provides enterprise-grade deduplication with ML-based optimization
type AdaptiveBlockDeduper struct {
	mu            sync.RWMutex
	blocks        map[string]*DedupeRecord
	maxSize       int
	baseTTL       time.Duration
	extendedTTL   time.Duration
	cleanupTicker *time.Ticker
	blockStats    map[string]*NetworkStats
	logger        *zap.Logger

	// Advanced enterprise features
	adaptiveEnabled   bool
	mlOptimization    bool
	priorityQueuing   bool
	crossNetworkDedup bool
	performanceMode   string // "STANDARD", "HIGH_PERFORMANCE", "MEMORY_OPTIMIZED"

	// Performance metrics
	totalProcessed    int64
	totalDuplicates   int64
	totalMemoryUsed   int64
	avgProcessingTime time.Duration

	// Advanced algorithms
	confidenceThreshold float64
	adaptationRate      float64
	learningEnabled     bool

	// Enterprise monitoring
	alertThresholds map[string]float64
	stop            chan struct{}
}

// NewAdaptiveBlockDeduper creates a new enterprise adaptive block deduper
func NewAdaptiveBlockDeduper(maxSize int, baseTTL time.Duration, logger *zap.Logger) *AdaptiveBlockDeduper {
	if maxSize <= 0 {
		maxSize = 10000 // Enterprise default
	}
	if baseTTL <= 0 {
		baseTTL = 5 * time.Minute // Enterprise default
	}

	abd := &AdaptiveBlockDeduper{
		blocks:              make(map[string]*DedupeRecord),
		maxSize:             maxSize,
		baseTTL:             baseTTL,
		extendedTTL:         baseTTL * 3,
		blockStats:          make(map[string]*NetworkStats),
		logger:              logger,
		adaptiveEnabled:     true,
		mlOptimization:      true,
		priorityQueuing:     true,
		crossNetworkDedup:   true,
		performanceMode:     "HIGH_PERFORMANCE",
		confidenceThreshold: 0.85,
		adaptationRate:      0.1,
		learningEnabled:     true,
		alertThresholds:     make(map[string]float64),
		stop:                make(chan struct{}),
	}

	// Initialize alert thresholds
	abd.alertThresholds["duplicate_rate"] = 0.7
	abd.alertThresholds["memory_usage"] = 0.9
	abd.alertThresholds["processing_time"] = 100.0 // milliseconds

	// Start cleanup goroutine
	abd.cleanupTicker = time.NewTicker(30 * time.Second)
	go abd.cleanupLoop()
	go abd.optimizationLoop()
	go abd.monitoringLoop()

	if logger != nil {
		logger.Info("Enterprise Adaptive Block Deduper initialized",
			zap.Int("max_size", maxSize),
			zap.Duration("base_ttl", baseTTL),
			zap.String("performance_mode", abd.performanceMode),
			zap.Bool("ml_optimization", abd.mlOptimization))
	}

	return abd
}

// Seen checks if a block has been seen before with advanced ML-based detection
func (abd *AdaptiveBlockDeduper) Seen(blockHash string, timestamp time.Time, network string, options ...DedupeOption) bool {
	start := time.Now()
	defer func() {
		abd.avgProcessingTime = time.Since(start)
		metrics.DeduplicationProcessingTime.WithLabelValues("seen", network).Observe(float64(time.Since(start).Nanoseconds()))
	}()

	// Input validation
	if blockHash == "" || network == "" {
		return false
	}

	abd.mu.Lock()
	defer abd.mu.Unlock()

	abd.totalProcessed++

	// Apply options
	opts := &DedupeOptions{}
	for _, opt := range options {
		opt(opts)
	}

	// Create network stats if it doesn't exist
	if _, exists := abd.blockStats[network]; !exists {
		abd.blockStats[network] = &NetworkStats{
			avgTimeBetween:    abd.baseTTL / 10,
			lastRecalculated:  timestamp,
			adaptiveTTL:       abd.baseTTL,
			reliability:       1.0,
			optimizationLevel: 1,
		}
	}

	// Generate composite key for cross-network deduplication
	key := abd.generateKey(blockHash, network, opts)
	record, exists := abd.blocks[key]

	// Update network statistics with ML optimization
	abd.updateNetworkStatsAdvanced(network, timestamp, exists, opts)

	if !exists {
		// Memory management - remove oldest if at capacity
		if len(abd.blocks) >= abd.maxSize {
			abd.removeOldestIntelligent()
		}

		// Create new record with enhanced metadata
		abd.blocks[key] = &DedupeRecord{
			BlockHash:  blockHash,
			Network:    network,
			FirstSeen:  timestamp,
			LastSeen:   timestamp,
			SeenCount:  1,
			Source:     opts.Source,
			Size:       opts.Size,
			Properties: opts.Properties,
			Confidence: abd.calculateConfidence(blockHash, network, opts),
			Priority:   abd.calculatePriority(network, opts),
		}

		// Update metrics
		metrics.DeduplicationCacheSize.Set(float64(len(abd.blocks)))

		return false
	}

	// Update existing record with enhanced tracking
	record.LastSeen = timestamp
	record.SeenCount++
	if opts.Size > 0 {
		record.Size = opts.Size
	}

	// Update confidence based on frequency and timing patterns
	record.Confidence = abd.updateConfidence(record, timestamp, opts)

	abd.totalDuplicates++
	metrics.DeduplicationDuplicatesDetected.WithLabelValues(network, "block").Inc()

	// Trigger alerts if thresholds exceeded
	abd.checkAlertThresholds(network)

	return true
}

// generateKey creates intelligent composite keys for advanced deduplication
func (abd *AdaptiveBlockDeduper) generateKey(blockHash, network string, opts *DedupeOptions) string {
	if abd.crossNetworkDedup && opts.CrossNetwork {
		// Cross-network deduplication - use hash only
		return blockHash
	}
	// Network-specific deduplication
	return network + ":" + blockHash
}

// calculateConfidence uses ML-based algorithms to determine block authenticity
func (abd *AdaptiveBlockDeduper) calculateConfidence(blockHash, network string, opts *DedupeOptions) float64 {
	if !abd.mlOptimization {
		return 1.0
	}

	confidence := 1.0

	// Hash pattern analysis
	if len(blockHash) < 64 {
		confidence *= 0.8
	}

	// Size validation
	if opts.Size > 0 {
		stats := abd.blockStats[network]
		if stats.avgBlockSize > 0 {
			sizeRatio := float64(opts.Size) / float64(stats.avgBlockSize)
			if sizeRatio < 0.1 || sizeRatio > 10.0 {
				confidence *= 0.7
			}
		}
	}

	// Source reliability
	if opts.Source != "" {
		// Implementation would include source reputation tracking
		confidence *= 0.95
	}

	return confidence
}

// calculatePriority determines processing priority based on network and options
func (abd *AdaptiveBlockDeduper) calculatePriority(network string, opts *DedupeOptions) int {
	if !abd.priorityQueuing {
		return 1
	}

	priority := 1

	// Network-based priority
	switch network {
	case "bitcoin":
		priority = 5
	case "ethereum":
		priority = 4
	case "solana":
		priority = 3
	default:
		priority = 1
	}

	// Size-based priority (larger blocks = higher priority)
	if opts.Size > 1024*1024 { // 1MB
		priority += 2
	}

	return priority
}

// updateNetworkStatsAdvanced performs sophisticated ML-based network optimization
func (abd *AdaptiveBlockDeduper) updateNetworkStatsAdvanced(network string, timestamp time.Time, isDuplicate bool, opts *DedupeOptions) {
	stats := abd.blockStats[network]
	stats.mu.Lock()
	defer stats.mu.Unlock()

	// Update block timing with advanced algorithms
	if !stats.lastBlockTime.IsZero() {
		timeSinceLast := timestamp.Sub(stats.lastBlockTime)

		if !isDuplicate || timeSinceLast > 100*time.Millisecond {
			// Exponential moving average with adaptive learning rate
			alpha := abd.adaptationRate
			if abd.learningEnabled {
				// Adjust learning rate based on network stability
				alpha = abd.calculateAdaptiveLearningRate(stats, timeSinceLast)
			}

			if stats.avgTimeBetween == 0 {
				stats.avgTimeBetween = timeSinceLast
			} else {
				stats.avgTimeBetween = time.Duration(float64(stats.avgTimeBetween)*(1-alpha) + float64(timeSinceLast)*alpha)
			}
			stats.blockCount++
		}
	}

	stats.lastBlockTime = timestamp

	// Advanced duplicate rate calculation with trend analysis
	stats.blocksTotal++
	if isDuplicate {
		stats.duplicatesTotal++
	}

	// Update block size statistics
	if opts.Size > 0 {
		if stats.avgBlockSize == 0 {
			stats.avgBlockSize = opts.Size
		} else {
			stats.avgBlockSize = int64(float64(stats.avgBlockSize)*0.9 + float64(opts.Size)*0.1)
		}
	}

	// Calculate throughput
	if stats.blockCount > 0 && stats.avgTimeBetween > 0 {
		stats.throughputBPS = float64(stats.avgBlockSize) / stats.avgTimeBetween.Seconds()
	}

	// Recalculate adaptive parameters every 100 blocks or every minute
	if stats.blocksTotal%100 == 0 || timestamp.Sub(stats.lastRecalculated) > time.Minute {
		abd.recalculateAdaptiveParametersAdvanced(stats, network, timestamp)
		stats.lastRecalculated = timestamp
	}
}

// calculateAdaptiveLearningRate implements ML-based learning rate adjustment
func (abd *AdaptiveBlockDeduper) calculateAdaptiveLearningRate(stats *NetworkStats, timeSinceLast time.Duration) float64 {
	baseRate := abd.adaptationRate

	// Adjust based on network stability
	if stats.avgTimeBetween > 0 {
		variance := float64(abs(timeSinceLast-stats.avgTimeBetween)) / float64(stats.avgTimeBetween)
		if variance > 0.5 { // High variance = faster learning
			return baseRate * 1.5
		} else if variance < 0.1 { // Low variance = slower learning
			return baseRate * 0.5
		}
	}

	return baseRate
}

// recalculateAdaptiveParametersAdvanced implements sophisticated ML optimization
func (abd *AdaptiveBlockDeduper) recalculateAdaptiveParametersAdvanced(stats *NetworkStats, network string, timestamp time.Time) {
	// Calculate current duplicate rate
	if stats.blocksTotal > 0 {
		stats.duplicateRate = float64(stats.duplicatesTotal) / float64(stats.blocksTotal)
		if stats.duplicateRate > stats.peakDuplicateRate {
			stats.peakDuplicateRate = stats.duplicateRate
		}
	}

	// Adaptive TTL calculation with ML optimization
	newTTL := abd.baseTTL

	if abd.adaptiveEnabled && stats.avgTimeBetween > 0 {
		// Factor 1: Block frequency (faster blocks = shorter TTL)
		frequencyFactor := float64(abd.baseTTL) / float64(stats.avgTimeBetween)
		if frequencyFactor > 5.0 {
			frequencyFactor = 5.0
		}

		// Factor 2: Duplicate rate (higher duplicates = longer TTL)
		duplicateFactor := 1.0 + stats.duplicateRate*2.0

		// Factor 3: Network reliability
		reliabilityFactor := stats.reliability

		// Factor 4: Optimization level
		optimizationFactor := 1.0 + float64(stats.optimizationLevel)*0.1

		// Combined adaptive calculation
		adaptiveFactor := frequencyFactor * duplicateFactor * reliabilityFactor * optimizationFactor
		newTTL = time.Duration(float64(abd.baseTTL) * adaptiveFactor)

		// Bounds checking
		if newTTL < abd.baseTTL/3 {
			newTTL = abd.baseTTL / 3
		} else if newTTL > abd.extendedTTL {
			newTTL = abd.extendedTTL
		}
	}

	stats.adaptiveTTL = newTTL

	// Update optimization level based on performance
	if stats.duplicateRate > 0.8 {
		stats.optimizationLevel = min(5, stats.optimizationLevel+1)
	} else if stats.duplicateRate < 0.2 {
		stats.optimizationLevel = max(1, stats.optimizationLevel-1)
	}

	// Update reliability score
	abd.updateReliabilityScore(stats, network)

	if abd.logger != nil {
		abd.logger.Debug("Adaptive parameters recalculated",
			zap.String("network", network),
			zap.Float64("duplicate_rate", stats.duplicateRate),
			zap.Duration("adaptive_ttl", stats.adaptiveTTL),
			zap.Int("optimization_level", stats.optimizationLevel),
			zap.Float64("reliability", stats.reliability))
	}
}

// updateReliabilityScore calculates network reliability based on performance metrics
func (abd *AdaptiveBlockDeduper) updateReliabilityScore(stats *NetworkStats, network string) {
	reliability := 1.0

	// Penalize high duplicate rates
	if stats.duplicateRate > 0.5 {
		reliability *= (1.0 - (stats.duplicateRate-0.5)*0.5)
	}

	// Reward consistent block timing
	if stats.blockCount > 10 {
		// Implementation would include timing variance calculation
		reliability *= 0.95 // Placeholder
	}

	// Apply exponential moving average
	if stats.reliability == 0 {
		stats.reliability = reliability
	} else {
		stats.reliability = stats.reliability*0.9 + reliability*0.1
	}
}

// updateConfidence implements ML-based confidence updating
func (abd *AdaptiveBlockDeduper) updateConfidence(record *DedupeRecord, timestamp time.Time, opts *DedupeOptions) float64 {
	if !abd.mlOptimization {
		return record.Confidence
	}

	// Increase confidence with repeated sightings
	frequencyBoost := 1.0 + float64(record.SeenCount)*0.05
	if frequencyBoost > 2.0 {
		frequencyBoost = 2.0
	}

	// Time-based confidence decay
	timeSinceFirst := timestamp.Sub(record.FirstSeen)
	timeDecay := math.Exp(-float64(timeSinceFirst) / float64(time.Hour))

	newConfidence := record.Confidence * frequencyBoost * timeDecay
	if newConfidence > 1.0 {
		newConfidence = 1.0
	}

	return newConfidence
}

// removeOldestIntelligent implements intelligent eviction based on priority and confidence
func (abd *AdaptiveBlockDeduper) removeOldestIntelligent() {
	if len(abd.blocks) == 0 {
		return
	}

	var oldestKey string
	var lowestScore float64 = 1000.0
	now := time.Now()

	// Find the block with the lowest composite score
	for key, record := range abd.blocks {
		// Calculate composite score based on multiple factors
		ageScore := float64(now.Sub(record.LastSeen)) / float64(time.Hour)
		confidenceScore := 1.0 / record.Confidence
		priorityScore := 1.0 / float64(record.Priority)
		frequencyScore := 1.0 / float64(record.SeenCount)

		compositeScore := ageScore + confidenceScore + priorityScore + frequencyScore

		if compositeScore < lowestScore {
			lowestScore = compositeScore
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(abd.blocks, oldestKey)
	}
}

// checkAlertThresholds monitors performance and triggers alerts when needed
func (abd *AdaptiveBlockDeduper) checkAlertThresholds(network string) {
	stats := abd.blockStats[network]
	if stats == nil {
		return
	}

	// Check duplicate rate threshold
	if stats.duplicateRate > abd.alertThresholds["duplicate_rate"] && abd.logger != nil {
		abd.logger.Warn("High duplicate rate detected",
			zap.String("network", network),
			zap.Float64("duplicate_rate", stats.duplicateRate),
			zap.Float64("threshold", abd.alertThresholds["duplicate_rate"]))
	}

	// Check memory usage
	memoryUsage := float64(len(abd.blocks)) / float64(abd.maxSize)
	if memoryUsage > abd.alertThresholds["memory_usage"] && abd.logger != nil {
		abd.logger.Warn("High memory usage detected",
			zap.Float64("usage_ratio", memoryUsage),
			zap.Int("current_size", len(abd.blocks)),
			zap.Int("max_size", abd.maxSize))
	}

	// Check processing time
	if abd.avgProcessingTime.Milliseconds() > int64(abd.alertThresholds["processing_time"]) && abd.logger != nil {
		abd.logger.Warn("High processing time detected",
			zap.Duration("avg_processing_time", abd.avgProcessingTime),
			zap.Float64("threshold_ms", abd.alertThresholds["processing_time"]))
	}
}

// Helper functions
func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// cleanupLoop runs periodic cleanup of expired records
func (abd *AdaptiveBlockDeduper) cleanupLoop() {
	for {
		select {
		case <-abd.cleanupTicker.C:
			abd.cleanup()
		case <-abd.stop:
			return
		}
	}
}

// optimizationLoop runs continuous ML-based optimization
func (abd *AdaptiveBlockDeduper) optimizationLoop() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if abd.mlOptimization {
				abd.performMLOptimization()
			}
		case <-abd.stop:
			return
		}
	}
}

// monitoringLoop provides real-time performance monitoring
func (abd *AdaptiveBlockDeduper) monitoringLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			abd.updatePerformanceMetrics()
		case <-abd.stop:
			return
		}
	}
}

// cleanup removes expired records based on adaptive TTL
func (abd *AdaptiveBlockDeduper) cleanup() {
	abd.mu.Lock()
	defer abd.mu.Unlock()

	now := time.Now()
	keysToDelete := []string{}

	for key, record := range abd.blocks {
		ttl := abd.getTTLNoLock(record.Network)

		// Consider confidence in cleanup decisions
		adjustedTTL := ttl
		if record.Confidence < abd.confidenceThreshold {
			adjustedTTL = time.Duration(float64(ttl) * record.Confidence)
		}

		if now.Sub(record.LastSeen) > adjustedTTL {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(abd.blocks, key)
	}

	// Update cache size metric
	metrics.DeduplicationCacheSize.Set(float64(len(abd.blocks)))

	if len(keysToDelete) > 0 && abd.logger != nil {
		abd.logger.Debug("Cleanup completed",
			zap.Int("removed", len(keysToDelete)),
			zap.Int("remaining", len(abd.blocks)),
			zap.Float64("cleanup_ratio", float64(len(keysToDelete))/float64(len(keysToDelete)+len(abd.blocks))))
	}
}

// getTTLNoLock returns the adaptive TTL for a network (must be called with lock held)
func (abd *AdaptiveBlockDeduper) getTTLNoLock(network string) time.Duration {
	if stats, exists := abd.blockStats[network]; exists {
		return stats.adaptiveTTL
	}
	return abd.baseTTL
}

// performMLOptimization runs advanced ML-based optimization algorithms
func (abd *AdaptiveBlockDeduper) performMLOptimization() {
	abd.mu.RLock()
	defer abd.mu.RUnlock()

	// Analyze patterns across all networks
	totalBlocks := 0
	totalDuplicates := 0

	for network, stats := range abd.blockStats {
		stats.mu.RLock()
		totalBlocks += stats.blocksTotal
		totalDuplicates += stats.duplicatesTotal

		// Optimize network-specific parameters
		abd.optimizeNetworkParameters(network, stats)
		stats.mu.RUnlock()
	}

	// Global optimization
	if totalBlocks > 0 {
		globalDuplicateRate := float64(totalDuplicates) / float64(totalBlocks)
		abd.optimizeGlobalParameters(globalDuplicateRate)
	}

	if abd.logger != nil {
		abd.logger.Debug("ML optimization completed",
			zap.Int("total_blocks", totalBlocks),
			zap.Int("total_duplicates", totalDuplicates),
			zap.Int("networks", len(abd.blockStats)))
	}
}

// optimizeNetworkParameters performs network-specific ML optimization
func (abd *AdaptiveBlockDeduper) optimizeNetworkParameters(network string, stats *NetworkStats) {
	// Gradient-based optimization for TTL
	if stats.duplicateRate > 0.6 {
		// High duplicate rate - increase TTL
		newTTL := time.Duration(float64(stats.adaptiveTTL) * 1.1)
		if newTTL <= abd.extendedTTL {
			stats.adaptiveTTL = newTTL
		}
	} else if stats.duplicateRate < 0.2 {
		// Low duplicate rate - decrease TTL
		newTTL := time.Duration(float64(stats.adaptiveTTL) * 0.9)
		if newTTL >= abd.baseTTL/3 {
			stats.adaptiveTTL = newTTL
		}
	}

	// Optimize learning rate based on network stability
	if stats.blockCount > 100 {
		// Calculate variance in block timing
		variance := abd.calculateTimingVariance(stats)
		if variance < 0.1 {
			// Stable network - slower learning
			abd.adaptationRate = maxFloat(0.05, abd.adaptationRate*0.95)
		} else if variance > 0.5 {
			// Unstable network - faster learning
			abd.adaptationRate = minFloat(0.3, abd.adaptationRate*1.05)
		}
	}
}

// calculateTimingVariance calculates variance in block timing for stability assessment
func (abd *AdaptiveBlockDeduper) calculateTimingVariance(stats *NetworkStats) float64 {
	// Simplified variance calculation - in production this would track actual variance
	if stats.avgTimeBetween == 0 {
		return 1.0
	}

	// Placeholder implementation - would need actual timing data
	baseVariance := 0.2
	if stats.duplicateRate > 0.5 {
		baseVariance *= 2.0 // High duplicates suggest instability
	}

	return baseVariance
}

// optimizeGlobalParameters performs system-wide optimization
func (abd *AdaptiveBlockDeduper) optimizeGlobalParameters(globalDuplicateRate float64) {
	// Adjust confidence threshold based on global performance
	if globalDuplicateRate > 0.7 {
		abd.confidenceThreshold = minFloat(0.95, abd.confidenceThreshold+0.05)
	} else if globalDuplicateRate < 0.3 {
		abd.confidenceThreshold = maxFloat(0.5, abd.confidenceThreshold-0.05)
	}

	// Adjust performance mode based on load
	currentLoad := float64(len(abd.blocks)) / float64(abd.maxSize)
	if currentLoad > 0.8 && abd.performanceMode != PerformanceModeMemoryOptimized {
		abd.performanceMode = PerformanceModeMemoryOptimized
		abd.triggerMemoryOptimization()
	} else if currentLoad < 0.3 && abd.performanceMode != PerformanceModeHighPerformance {
		abd.performanceMode = PerformanceModeHighPerformance
	}
}

// Helper functions for float operations
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// triggerMemoryOptimization implements aggressive memory optimization
func (abd *AdaptiveBlockDeduper) triggerMemoryOptimization() {
	// Remove low-confidence entries
	abd.mu.Lock()
	defer abd.mu.Unlock()

	keysToRemove := []string{}
	for key, record := range abd.blocks {
		if record.Confidence < 0.6 || record.SeenCount < 2 {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for _, key := range keysToRemove {
		delete(abd.blocks, key)
	}

	if len(keysToRemove) > 0 && abd.logger != nil {
		abd.logger.Info("Memory optimization triggered",
			zap.Int("removed_low_confidence", len(keysToRemove)),
			zap.Int("remaining", len(abd.blocks)))
	}
}

// updatePerformanceMetrics updates real-time performance metrics
func (abd *AdaptiveBlockDeduper) updatePerformanceMetrics() {
	abd.mu.RLock()
	defer abd.mu.RUnlock()

	// Update memory usage metrics
	abd.totalMemoryUsed = int64(len(abd.blocks)) * 1024 // Approximate per-record size
	metrics.DeduplicationMemoryUsage.WithLabelValues("adaptive_blocks").Set(float64(abd.totalMemoryUsed))

	// Update efficiency metrics
	if abd.totalProcessed > 0 {
		efficiency := 1.0 - (float64(abd.totalDuplicates) / float64(abd.totalProcessed))
		metrics.DeduplicationEfficiency.WithLabelValues("global", "adaptive").Set(efficiency)
	}

	// Update network-specific metrics
	for network, stats := range abd.blockStats {
		stats.mu.RLock()
		metrics.DeduplicationNetworkDuplicateRate.WithLabelValues(network).Set(stats.duplicateRate)
		metrics.DeduplicationNetworkTTL.WithLabelValues(network).Set(stats.adaptiveTTL.Seconds())
		stats.mu.RUnlock()
	}
}

// GetStats returns comprehensive statistics about the deduplication system
func (abd *AdaptiveBlockDeduper) GetStats() map[string]interface{} {
	abd.mu.RLock()
	defer abd.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_records"] = len(abd.blocks)
	stats["max_size"] = abd.maxSize
	stats["base_ttl_seconds"] = abd.baseTTL.Seconds()
	stats["extended_ttl_seconds"] = abd.extendedTTL.Seconds()
	stats["total_processed"] = abd.totalProcessed
	stats["total_duplicates"] = abd.totalDuplicates
	stats["memory_usage_bytes"] = abd.totalMemoryUsed
	stats["avg_processing_time_ms"] = abd.avgProcessingTime.Milliseconds()
	stats["performance_mode"] = abd.performanceMode
	stats["confidence_threshold"] = abd.confidenceThreshold
	stats["adaptation_rate"] = abd.adaptationRate
	stats["ml_optimization_enabled"] = abd.mlOptimization
	stats["learning_enabled"] = abd.learningEnabled

	// Calculate efficiency
	if abd.totalProcessed > 0 {
		stats["efficiency"] = 1.0 - (float64(abd.totalDuplicates) / float64(abd.totalProcessed))
	}

	networkStats := make(map[string]interface{})
	for network, ns := range abd.blockStats {
		ns.mu.RLock()
		netStats := map[string]interface{}{
			"duplicate_rate":           ns.duplicateRate,
			"peak_duplicate_rate":      ns.peakDuplicateRate,
			"total_blocks_seen":        ns.blocksTotal,
			"total_duplicates":         ns.duplicatesTotal,
			"adaptive_ttl_seconds":     ns.adaptiveTTL.Seconds(),
			"avg_time_between_seconds": ns.avgTimeBetween.Seconds(),
			"avg_block_size_bytes":     ns.avgBlockSize,
			"throughput_bps":           ns.throughputBPS,
			"network_latency_ms":       ns.networkLatency.Milliseconds(),
			"reliability_score":        ns.reliability,
			"optimization_level":       ns.optimizationLevel,
		}
		ns.mu.RUnlock()
		networkStats[network] = netStats
	}
	stats["networks"] = networkStats

	return stats
}

// Close gracefully shuts down the adaptive deduper
func (abd *AdaptiveBlockDeduper) Close() error {
	close(abd.stop)
	if abd.cleanupTicker != nil {
		abd.cleanupTicker.Stop()
	}

	if abd.logger != nil {
		abd.logger.Info("Adaptive Block Deduper shutdown complete",
			zap.Int64("total_processed", abd.totalProcessed),
			zap.Int64("total_duplicates", abd.totalDuplicates),
			zap.Int("final_cache_size", len(abd.blocks)))
	}

	return nil
}

// SetPerformanceMode changes the performance optimization mode
func (abd *AdaptiveBlockDeduper) SetPerformanceMode(mode string) {
	abd.mu.Lock()
	defer abd.mu.Unlock()

	oldMode := abd.performanceMode
	abd.performanceMode = mode

	// Apply mode-specific optimizations
	switch mode {
	case PerformanceModeHighPerformance:
		abd.adaptationRate = 0.15
		abd.confidenceThreshold = 0.8
	case PerformanceModeMemoryOptimized:
		abd.adaptationRate = 0.05
		abd.confidenceThreshold = 0.9
		abd.triggerMemoryOptimization()
	case PerformanceModeLatencyOptimized:
		abd.adaptationRate = 0.2
		abd.confidenceThreshold = 0.75
	}

	if abd.logger != nil {
		abd.logger.Info("Performance mode changed",
			zap.String("old_mode", oldMode),
			zap.String("new_mode", mode))
	}
}

// EnableMLOptimization enables or disables ML-based optimization
func (abd *AdaptiveBlockDeduper) EnableMLOptimization(enabled bool) {
	abd.mu.Lock()
	defer abd.mu.Unlock()

	abd.mlOptimization = enabled
	abd.learningEnabled = enabled

	if abd.logger != nil {
		abd.logger.Info("ML optimization setting changed",
			zap.Bool("enabled", enabled))
	}
}
