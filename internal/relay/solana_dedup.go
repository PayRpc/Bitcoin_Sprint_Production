package relay

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/dedup"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

// Solana-specific deduplication metrics
var (
	solanaDuplicatesSuppressed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "solana_relay_duplicates_suppressed_total",
		Help: "Number of duplicate Solana blocks/slots suppressed",
	}, []string{"type", "tier"})

	solanaTTLAdjustments = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "solana_relay_ttl_adjustments_total",
		Help: "Number of TTL adjustments made for Solana deduplication",
	}, []string{"direction", "tier"})

	solanaAdaptiveTTL = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "solana_relay_adaptive_ttl_seconds",
		Help: "Current adaptive TTL for Solana deduplication",
	}, []string{"tier"})

	solanaDuplicateRate = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "solana_relay_duplicate_rate",
		Help: "Current duplicate rate for Solana blocks/slots",
	}, []string{"type", "tier"})
)

// SolanaDeduper provides enterprise-grade Solana-specific deduplication
type SolanaDeduper struct {
	// Core deduplication
	mu   sync.RWMutex
	seen map[string]*SolanaEntry

	// Adaptive TTL management
	ttl         time.Duration
	minTTL      time.Duration
	maxTTL      time.Duration
	dupCount    int64
	totalCount  int64
	lastAdjust  time.Time
	adjustEvery time.Duration

	// Enterprise features
	logger   *zap.Logger
	tier     string
	capacity int
	order    []string

	// Solana-specific optimizations
	slotDedup        bool
	blockDedup       bool
	txDedup          bool
	crossSlotDedup   bool
	velocityTracking bool

	// Performance tracking
	slotVelocity  float64 // slots per second
	blockVelocity float64 // blocks per second
	avgSlotTime   time.Duration
	lastSlotTime  time.Time

	// Advanced algorithms
	adaptiveLearning  bool
	confidenceScoring bool
	priorityQueuing   bool

	// ML-based optimization
	learningRate        float64
	confidenceThreshold float64
	velocityPredictor   float64

	// Statistics tracking
	typeStats map[string]*SolanaTypeStats
}

// SolanaEntry represents a deduplicated Solana entry with metadata
type SolanaEntry struct {
	Hash        string                 `json:"hash"`
	Type        string                 `json:"type"` // "slot", "block", "transaction"
	FirstSeen   time.Time              `json:"first_seen"`
	LastSeen    time.Time              `json:"last_seen"`
	SeenCount   int                    `json:"seen_count"`
	Confidence  float64                `json:"confidence"`
	Priority    int                    `json:"priority"`
	SlotNumber  uint64                 `json:"slot_number,omitempty"`
	BlockHeight uint64                 `json:"block_height,omitempty"`
	Source      string                 `json:"source"`
	Properties  map[string]interface{} `json:"properties"`
}

// SolanaTypeStats tracks statistics for different Solana data types
type SolanaTypeStats struct {
	TotalSeen      int64         `json:"total_seen"`
	Duplicates     int64         `json:"duplicates"`
	DuplicateRate  float64       `json:"duplicate_rate"`
	AdaptiveTTL    time.Duration `json:"adaptive_ttl"`
	AvgTimeBetween time.Duration `json:"avg_time_between"`
	LastSeen       time.Time     `json:"last_seen"`
	Velocity       float64       `json:"velocity"`
}

// NewSolanaDeduper creates a new enterprise-grade Solana deduplicator
func NewSolanaDeduper(tier string, logger *zap.Logger) *SolanaDeduper {
	capacity := getSolanaCapacityForTier(tier)

	sd := &SolanaDeduper{
		seen:        make(map[string]*SolanaEntry, capacity),
		order:       make([]string, 0, capacity),
		capacity:    capacity,
		ttl:         getSolanaBaseTTL(tier),
		minTTL:      5 * time.Second,
		maxTTL:      5 * time.Minute,
		adjustEvery: 30 * time.Second,
		lastAdjust:  time.Now(),
		logger:      logger,
		tier:        tier,

		// Enable features based on tier
		slotDedup:         true,
		blockDedup:        true,
		txDedup:           tier == "ENTERPRISE" || tier == "BUSINESS",
		crossSlotDedup:    tier == "ENTERPRISE",
		velocityTracking:  tier != "FREE",
		adaptiveLearning:  tier == "ENTERPRISE",
		confidenceScoring: tier == "ENTERPRISE" || tier == "BUSINESS",
		priorityQueuing:   tier == "ENTERPRISE",

		// ML parameters
		learningRate:        0.1,
		confidenceThreshold: 0.8,

		// Statistics
		typeStats: make(map[string]*SolanaTypeStats),
	}

	// Initialize type statistics
	types := []string{"slot", "block", "transaction"}
	for _, t := range types {
		sd.typeStats[t] = &SolanaTypeStats{
			AdaptiveTTL: sd.ttl,
		}
	}

	if logger != nil {
		logger.Info("Enterprise Solana Deduper initialized",
			zap.String("tier", tier),
			zap.Int("capacity", capacity),
			zap.Duration("base_ttl", sd.ttl),
			zap.Bool("adaptive_learning", sd.adaptiveLearning),
			zap.Bool("velocity_tracking", sd.velocityTracking))
	}

	return sd
}

// getSolanaCapacityForTier returns appropriate capacity for service tier
func getSolanaCapacityForTier(tier string) int {
	switch tier {
	case "FREE":
		return 2048
	case "BUSINESS":
		return 8192
	case "ENTERPRISE":
		return 16384
	default:
		return 4096
	}
}

// getSolanaBaseTTL returns base TTL optimized for Solana's fast block times
func getSolanaBaseTTL(tier string) time.Duration {
	switch tier {
	case "FREE":
		return 20 * time.Second
	case "BUSINESS":
		return 30 * time.Second
	case "ENTERPRISE":
		return 45 * time.Second
	default:
		return 25 * time.Second
	}
}

// IsDuplicate checks if a Solana item is a duplicate with enterprise features
func (sd *SolanaDeduper) IsDuplicate(hash, itemType string, options ...dedup.DedupeOption) bool {
	if hash == "" || itemType == "" {
		return false
	}

	// Apply options
	opts := &dedup.DedupeOptions{}
	for _, opt := range options {
		opt(opts)
	}

	now := time.Now()
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.totalCount++

	// Update type statistics
	if sd.typeStats[itemType] == nil {
		sd.typeStats[itemType] = &SolanaTypeStats{
			AdaptiveTTL: sd.ttl,
		}
	}
	typeStats := sd.typeStats[itemType]
	typeStats.TotalSeen++

	// Generate composite key for cross-slot deduplication
	key := sd.generateKey(hash, itemType, opts)

	if entry, exists := sd.seen[key]; exists {
		// Calculate type-specific TTL
		currentTTL := sd.getAdaptiveTTL(itemType, typeStats)

		if now.Sub(entry.LastSeen) <= currentTTL {
			// It's a duplicate
			sd.dupCount++
			typeStats.Duplicates++
			entry.LastSeen = now
			entry.SeenCount++

			// Update confidence if enabled
			if sd.confidenceScoring {
				entry.Confidence = sd.updateConfidence(entry, now)
			}

			// Update metrics
			solanaDuplicatesSuppressed.WithLabelValues(itemType, sd.tier).Inc()

			// Track velocity for active items
			if sd.velocityTracking {
				sd.updateVelocityTracking(itemType, now, typeStats)
			}

			return true
		}

		// Entry expired, update it
		entry.LastSeen = now
		entry.SeenCount = 1 // Reset count for expired entry
		if sd.confidenceScoring {
			entry.Confidence = sd.calculateInitialConfidence(hash, itemType, opts)
		}
	} else {
		// New entry
		if len(sd.seen) >= sd.capacity {
			sd.evictOldest()
		}

		entry := &SolanaEntry{
			Hash:       hash,
			Type:       itemType,
			FirstSeen:  now,
			LastSeen:   now,
			SeenCount:  1,
			Source:     opts.Source,
			Priority:   sd.calculatePriority(itemType, opts),
			Properties: opts.Properties,
		}

		if sd.confidenceScoring {
			entry.Confidence = sd.calculateInitialConfidence(hash, itemType, opts)
		}

		// Add Solana-specific metadata
		if opts.Properties != nil {
			if slotNum, ok := opts.Properties["slot_number"].(uint64); ok {
				entry.SlotNumber = slotNum
			}
			if blockHeight, ok := opts.Properties["block_height"].(uint64); ok {
				entry.BlockHeight = blockHeight
			}
		}

		sd.seen[key] = entry
		sd.order = append(sd.order, key)
	}

	// Update velocity tracking
	if sd.velocityTracking {
		sd.updateVelocityTracking(itemType, now, typeStats)
	}

	// Periodic TTL adjustment with ML optimization
	if now.Sub(sd.lastAdjust) >= sd.adjustEvery {
		sd.adjustTTLWithML()
		sd.lastAdjust = now
	}

	return false
}

// generateKey creates appropriate keys based on configuration
func (sd *SolanaDeduper) generateKey(hash, itemType string, opts *dedup.DedupeOptions) string {
	if sd.crossSlotDedup && itemType != "slot" {
		// Cross-slot deduplication for blocks and transactions
		return hash
	}

	// Slot-specific deduplication
	slotPrefix := "unknown"
	if opts.Properties != nil {
		if slotNum, ok := opts.Properties["slot_number"].(uint64); ok {
			slotPrefix = fmt.Sprintf("slot_%d", slotNum)
		}
	}

	return fmt.Sprintf("%s:%s:%s", slotPrefix, itemType, hash)
}

// getAdaptiveTTL returns adaptive TTL for a specific type
func (sd *SolanaDeduper) getAdaptiveTTL(itemType string, typeStats *SolanaTypeStats) time.Duration {
	if typeStats.AdaptiveTTL > 0 {
		return typeStats.AdaptiveTTL
	}
	return sd.ttl
}

// calculateInitialConfidence calculates initial confidence for new entries
func (sd *SolanaDeduper) calculateInitialConfidence(hash, itemType string, opts *dedup.DedupeOptions) float64 {
	if !sd.confidenceScoring {
		return 1.0
	}

	confidence := 0.8 // Base confidence

	// Hash quality assessment
	if len(hash) >= 64 {
		confidence += 0.1
	}

	// Source reliability
	if opts.Source != "" {
		confidence += 0.05
	}

	// Type-specific adjustments
	switch itemType {
	case "slot":
		confidence += 0.05 // Slots are usually reliable
	case "block":
		confidence += 0.03
	case "transaction":
		confidence -= 0.02 // Transactions can be more variable
	}

	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// updateConfidence updates confidence based on observation patterns
func (sd *SolanaDeduper) updateConfidence(entry *SolanaEntry, now time.Time) float64 {
	if !sd.confidenceScoring {
		return entry.Confidence
	}

	// Increase confidence with repeated sightings
	frequencyBoost := 1.0 + float64(entry.SeenCount)*0.02
	if frequencyBoost > 1.5 {
		frequencyBoost = 1.5
	}

	// Time-based confidence decay
	timeSinceFirst := now.Sub(entry.FirstSeen)
	timeDecay := math.Exp(-float64(timeSinceFirst) / float64(time.Hour))

	newConfidence := entry.Confidence * frequencyBoost * timeDecay
	if newConfidence > 1.0 {
		newConfidence = 1.0
	}

	return newConfidence
}

// calculatePriority determines priority based on type and options
func (sd *SolanaDeduper) calculatePriority(itemType string, opts *dedup.DedupeOptions) int {
	if !sd.priorityQueuing {
		return 1
	}

	priority := 1

	// Type-based priority
	switch itemType {
	case "slot":
		priority = 5 // Highest priority
	case "block":
		priority = 3
	case "transaction":
		priority = 1
	}

	// Option-based adjustments
	if opts.Priority > 0 {
		priority = opts.Priority
	}

	return priority
}

// updateVelocityTracking updates velocity statistics for performance optimization
func (sd *SolanaDeduper) updateVelocityTracking(itemType string, now time.Time, typeStats *SolanaTypeStats) {
	if !typeStats.LastSeen.IsZero() {
		timeSinceLast := now.Sub(typeStats.LastSeen)

		// Exponential moving average for time between items
		alpha := sd.learningRate
		if typeStats.AvgTimeBetween == 0 {
			typeStats.AvgTimeBetween = timeSinceLast
		} else {
			typeStats.AvgTimeBetween = time.Duration(float64(typeStats.AvgTimeBetween)*(1-alpha) + float64(timeSinceLast)*alpha)
		}

		// Calculate velocity (items per second)
		if typeStats.AvgTimeBetween > 0 {
			typeStats.Velocity = 1.0 / typeStats.AvgTimeBetween.Seconds()
		}
	}

	typeStats.LastSeen = now
}

// adjustTTLWithML performs ML-based TTL adjustment
func (sd *SolanaDeduper) adjustTTLWithML() {
	if sd.totalCount < 50 {
		return
	}

	// Calculate global duplicate rate
	globalRate := float64(sd.dupCount) / float64(sd.totalCount)

	// ML-based TTL adjustment
	if sd.adaptiveLearning {
		sd.adjustTTLWithAdvancedML(globalRate)
	} else {
		sd.adjustTTLBasic(globalRate)
	}

	// Update metrics
	solanaAdaptiveTTL.WithLabelValues(sd.tier).Set(sd.ttl.Seconds())
	solanaDuplicateRate.WithLabelValues("global", sd.tier).Set(globalRate)

	// Reset counters with partial decay
	sd.totalCount = sd.totalCount / 2
	sd.dupCount = sd.dupCount / 2
}

// adjustTTLWithAdvancedML implements advanced ML-based TTL optimization
func (sd *SolanaDeduper) adjustTTLWithAdvancedML(globalRate float64) {
	// Calculate velocity-adjusted learning rate
	adaptiveLearningRate := sd.learningRate
	if sd.velocityTracking && sd.slotVelocity > 0 {
		// Adjust learning rate based on network velocity
		velocityFactor := math.Min(2.0, sd.slotVelocity/2.0) // Normalize around 2 slots/sec
		adaptiveLearningRate *= velocityFactor
	}

	// Multi-factor TTL adjustment
	targetRate := 0.3 // Target 30% duplicate rate for optimal performance
	rateDelta := globalRate - targetRate

	// Calculate TTL adjustment factor
	adjustmentFactor := 1.0 + (rateDelta * adaptiveLearningRate * 2.0)

	// Velocity-based adjustment
	if sd.velocityTracking {
		velocityAdjustment := 1.0
		if sd.slotVelocity > 3.0 { // High velocity - shorter TTL
			velocityAdjustment = 0.8
		} else if sd.slotVelocity < 1.0 { // Low velocity - longer TTL
			velocityAdjustment = 1.2
		}
		adjustmentFactor *= velocityAdjustment
	}

	// Apply adjustment
	newTTL := time.Duration(float64(sd.ttl) * adjustmentFactor)

	// Bounds checking
	if newTTL < sd.minTTL {
		newTTL = sd.minTTL
	} else if newTTL > sd.maxTTL {
		newTTL = sd.maxTTL
	}

	// Track adjustment direction
	if newTTL > sd.ttl {
		solanaTTLAdjustments.WithLabelValues("increase", sd.tier).Inc()
	} else if newTTL < sd.ttl {
		solanaTTLAdjustments.WithLabelValues("decrease", sd.tier).Inc()
	}

	sd.ttl = newTTL

	// Adjust type-specific TTLs
	for itemType, typeStats := range sd.typeStats {
		if typeStats.TotalSeen > 10 {
			typeRate := float64(typeStats.Duplicates) / float64(typeStats.TotalSeen)
			typeFactor := 1.0 + (typeRate-targetRate)*adaptiveLearningRate

			// Type-specific adjustments
			switch itemType {
			case "slot":
				typeFactor *= 1.2 // Slots need longer TTL
			case "transaction":
				typeFactor *= 0.8 // Transactions can have shorter TTL
			}

			newTypeTTL := time.Duration(float64(sd.ttl) * typeFactor)
			if newTypeTTL >= sd.minTTL && newTypeTTL <= sd.maxTTL {
				typeStats.AdaptiveTTL = newTypeTTL
			}

			// Update type-specific metrics
			solanaDuplicateRate.WithLabelValues(itemType, sd.tier).Set(typeRate)
		}
	}
}

// adjustTTLBasic performs basic TTL adjustment for non-enterprise tiers
func (sd *SolanaDeduper) adjustTTLBasic(rate float64) {
	switch {
	case rate > 0.50:
		// Lots of duplicates, increase TTL
		sd.ttl = sd.ttl + 10*time.Second
		solanaTTLAdjustments.WithLabelValues("increase", sd.tier).Inc()
	case rate > 0.25:
		sd.ttl = sd.ttl + 5*time.Second
		solanaTTLAdjustments.WithLabelValues("increase", sd.tier).Inc()
	case rate < 0.05:
		// Few duplicates: shrink TTL
		if sd.ttl > 10*time.Second {
			sd.ttl = sd.ttl - 5*time.Second
			solanaTTLAdjustments.WithLabelValues("decrease", sd.tier).Inc()
		}
	default:
		// Small drift
		sd.ttl = sd.ttl + 1*time.Second
	}

	// Bounds checking
	if sd.ttl < sd.minTTL {
		sd.ttl = sd.minTTL
	}
	if sd.ttl > sd.maxTTL {
		sd.ttl = sd.maxTTL
	}
}

// evictOldest implements intelligent eviction based on priority and age
func (sd *SolanaDeduper) evictOldest() {
	if len(sd.order) == 0 {
		return
	}

	if sd.priorityQueuing {
		sd.evictByPriority()
	} else {
		// Simple FIFO eviction
		oldKey := sd.order[0]
		sd.order = sd.order[1:]
		delete(sd.seen, oldKey)
	}
}

// evictByPriority implements priority-based eviction
func (sd *SolanaDeduper) evictByPriority() {
	var lowestPriorityKey string
	lowestPriority := 100
	oldestTime := time.Now()

	// Find the entry with lowest priority and oldest timestamp
	for key, entry := range sd.seen {
		if entry.Priority < lowestPriority ||
			(entry.Priority == lowestPriority && entry.LastSeen.Before(oldestTime)) {
			lowestPriority = entry.Priority
			oldestTime = entry.LastSeen
			lowestPriorityKey = key
		}
	}

	if lowestPriorityKey != "" {
		// Remove from order slice
		for i, key := range sd.order {
			if key == lowestPriorityKey {
				sd.order = append(sd.order[:i], sd.order[i+1:]...)
				break
			}
		}
		delete(sd.seen, lowestPriorityKey)
	}
}

// GetStats returns comprehensive Solana deduplication statistics
func (sd *SolanaDeduper) GetStats() map[string]interface{} {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	globalRate := 0.0
	if sd.totalCount > 0 {
		globalRate = float64(sd.dupCount) / float64(sd.totalCount)
	}

	stats := map[string]interface{}{
		"tier":                  sd.tier,
		"total_seen":            sd.totalCount,
		"duplicates_found":      sd.dupCount,
		"global_duplicate_rate": globalRate,
		"current_ttl_seconds":   sd.ttl.Seconds(),
		"min_ttl_seconds":       sd.minTTL.Seconds(),
		"max_ttl_seconds":       sd.maxTTL.Seconds(),
		"capacity":              sd.capacity,
		"current_size":          len(sd.seen),
		"slot_velocity":         sd.slotVelocity,
		"block_velocity":        sd.blockVelocity,
		"adaptive_learning":     sd.adaptiveLearning,
		"velocity_tracking":     sd.velocityTracking,
		"confidence_scoring":    sd.confidenceScoring,
		"priority_queuing":      sd.priorityQueuing,
		"cross_slot_dedup":      sd.crossSlotDedup,
		"learning_rate":         sd.learningRate,
		"confidence_threshold":  sd.confidenceThreshold,
	}

	// Add type-specific statistics
	typeStatsMap := make(map[string]interface{})
	for itemType, typeStats := range sd.typeStats {
		typeRate := 0.0
		if typeStats.TotalSeen > 0 {
			typeRate = float64(typeStats.Duplicates) / float64(typeStats.TotalSeen)
		}

		typeStatsMap[itemType] = map[string]interface{}{
			"total_seen":               typeStats.TotalSeen,
			"duplicates":               typeStats.Duplicates,
			"duplicate_rate":           typeRate,
			"adaptive_ttl_seconds":     typeStats.AdaptiveTTL.Seconds(),
			"avg_time_between_seconds": typeStats.AvgTimeBetween.Seconds(),
			"velocity":                 typeStats.Velocity,
		}
	}
	stats["type_statistics"] = typeStatsMap

	return stats
}

// Cleanup performs intelligent cleanup of expired entries
func (sd *SolanaDeduper) Cleanup() {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	now := time.Now()
	keysToDelete := []string{}

	for key, entry := range sd.seen {
		// Use type-specific TTL
		typeStats := sd.typeStats[entry.Type]
		currentTTL := sd.ttl
		if typeStats != nil && typeStats.AdaptiveTTL > 0 {
			currentTTL = typeStats.AdaptiveTTL
		}

		// Consider confidence in cleanup decisions
		adjustedTTL := currentTTL
		if sd.confidenceScoring && entry.Confidence < sd.confidenceThreshold {
			adjustedTTL = time.Duration(float64(currentTTL) * entry.Confidence)
		}

		if now.Sub(entry.LastSeen) > adjustedTTL {
			keysToDelete = append(keysToDelete, key)
		}
	}

	// Remove expired entries
	for _, key := range keysToDelete {
		delete(sd.seen, key)

		// Remove from order slice
		for i, orderKey := range sd.order {
			if orderKey == key {
				sd.order = append(sd.order[:i], sd.order[i+1:]...)
				break
			}
		}
	}

	if len(keysToDelete) > 0 && sd.logger != nil {
		sd.logger.Debug("Solana dedup cleanup completed",
			zap.Int("removed", len(keysToDelete)),
			zap.Int("remaining", len(sd.seen)),
			zap.String("tier", sd.tier))
	}
}

// SetTier updates the service tier and reconfigures accordingly
func (sd *SolanaDeduper) SetTier(tier string) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	oldTier := sd.tier
	sd.tier = tier

	// Update tier-dependent features
	sd.txDedup = tier == "ENTERPRISE" || tier == "BUSINESS"
	sd.crossSlotDedup = tier == "ENTERPRISE"
	sd.velocityTracking = tier != "FREE"
	sd.adaptiveLearning = tier == "ENTERPRISE"
	sd.confidenceScoring = tier == "ENTERPRISE" || tier == "BUSINESS"
	sd.priorityQueuing = tier == "ENTERPRISE"

	// Update capacity
	newCapacity := getSolanaCapacityForTier(tier)
	if newCapacity != sd.capacity {
		sd.capacity = newCapacity
		// Trigger cleanup if over capacity
		if len(sd.seen) > sd.capacity {
			sd.enforceCapacity()
		}
	}

	if sd.logger != nil {
		sd.logger.Info("Solana deduper tier updated",
			zap.String("old_tier", oldTier),
			zap.String("new_tier", tier),
			zap.Int("new_capacity", newCapacity))
	}
}

// enforceCapacity reduces cache size to fit within capacity limits
func (sd *SolanaDeduper) enforceCapacity() {
	for len(sd.seen) > sd.capacity && len(sd.order) > 0 {
		sd.evictOldest()
	}
}

// Close gracefully shuts down the Solana deduper
func (sd *SolanaDeduper) Close() error {
	if sd.logger != nil {
		sd.logger.Info("Solana deduper shutdown",
			zap.String("tier", sd.tier),
			zap.Int64("total_processed", sd.totalCount),
			zap.Int64("duplicates_found", sd.dupCount))
	}
	return nil
}

// Legacy support methods for backward compatibility
func newSolanaDeduper() *SolanaDeduper {
	return NewSolanaDeduper("FREE", nil)
}

func (sd *SolanaDeduper) isDup(key string) bool {
	return sd.IsDuplicate(key, "unknown")
}

func (sd *SolanaDeduper) stats() (ttl time.Duration, dupRate float64) {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	ttl = sd.ttl
	dupRate = 0.0
	if sd.totalCount > 0 {
		dupRate = float64(sd.dupCount) / float64(sd.totalCount)
	}
	return ttl, dupRate
}
