package relay

import (
	"sync"
	"time"
	
	"go.uber.org/zap"
)

// DedupeRecord stores information about a block for deduplication
type DedupeRecord struct {
	BlockHash  string
	Network    string
	FirstSeen  time.Time
	LastSeen   time.Time
	SeenCount  int
	Properties map[string]interface{}
}

// AdaptiveBlockDeduper provides deduplication with adaptive TTL based on block frequency
type AdaptiveBlockDeduper struct {
	mu            sync.RWMutex
	blocks        map[string]*DedupeRecord
	maxSize       int
	baseTTL       time.Duration
	extendedTTL   time.Duration
	cleanupTicker *time.Ticker
	blockStats    map[string]*networkStats // Stats per network
	logger        *zap.Logger
}

// networkStats tracks metrics for optimizing deduplication per network
type networkStats struct {
	mu               sync.RWMutex
	avgTimeBetween   time.Duration // Average time between blocks
	blockCount       int
	lastBlockTime    time.Time
	duplicateRate    float64 // Rate of duplicates seen
	duplicatesTotal  int
	blocksTotal      int
	adaptiveTTL      time.Duration
	lastRecalculated time.Time
}

// NewAdaptiveBlockDeduper creates a new adaptive block deduper
func NewAdaptiveBlockDeduper(maxSize int, baseTTL time.Duration, logger *zap.Logger) *AdaptiveBlockDeduper {
	deduper := &AdaptiveBlockDeduper{
		blocks:      make(map[string]*DedupeRecord, maxSize),
		maxSize:     maxSize,
		baseTTL:     baseTTL,
		extendedTTL: baseTTL * 3, // Default to 3x for high-frequency blocks
		blockStats:  make(map[string]*networkStats),
		logger:      logger,
	}
	
	// Start cleanup routine
	deduper.cleanupTicker = time.NewTicker(baseTTL / 2)
	go deduper.cleanupLoop()
	
	return deduper
}

// Seen checks if a block has been seen before and records it
func (abd *AdaptiveBlockDeduper) Seen(blockHash string, timestamp time.Time, network string) bool {
	abd.mu.Lock()
	defer abd.mu.Unlock()
	
	// Create network stats if it doesn't exist
	if _, exists := abd.blockStats[network]; !exists {
		abd.blockStats[network] = &networkStats{
			avgTimeBetween:   abd.baseTTL / 10, // Initial guess
			lastRecalculated: timestamp,
			adaptiveTTL:      abd.baseTTL,
		}
	}
	
	// Check if block exists
	key := network + ":" + blockHash
	record, exists := abd.blocks[key]
	
	// Update network statistics
	abd.updateNetworkStats(network, timestamp, exists)
	
	if !exists {
		// If we're at capacity, remove oldest item
		if len(abd.blocks) >= abd.maxSize {
			abd.removeOldest()
		}
		
		// Add new record
		abd.blocks[key] = &DedupeRecord{
			BlockHash:  blockHash,
			Network:    network,
			FirstSeen:  timestamp,
			LastSeen:   timestamp,
			SeenCount:  1,
			Properties: make(map[string]interface{}),
		}
		return false
	}
	
	// Update existing record
	record.LastSeen = timestamp
	record.SeenCount++
	
	return true
}

// updateNetworkStats updates statistics for block timing and duplication rate
func (abd *AdaptiveBlockDeduper) updateNetworkStats(network string, timestamp time.Time, isDuplicate bool) {
	stats := abd.blockStats[network]
	stats.mu.Lock()
	defer stats.mu.Unlock()
	
	// Update block count and time between blocks
	if !stats.lastBlockTime.IsZero() {
		timeSinceLast := timestamp.Sub(stats.lastBlockTime)
		
		// Only update timing stats if it's not a duplicate or if more than 100ms has passed
		// This helps filter out batched duplicate notifications
		if !isDuplicate || timeSinceLast > 100*time.Millisecond {
			// Exponential moving average for time between blocks
			alpha := 0.2 // Smoothing factor
			if stats.avgTimeBetween == 0 {
				stats.avgTimeBetween = timeSinceLast
			} else {
				stats.avgTimeBetween = time.Duration(float64(stats.avgTimeBetween)*(1-alpha) + float64(timeSinceLast)*alpha)
			}
			stats.blockCount++
		}
	}
	
	// Update last block time
	stats.lastBlockTime = timestamp
	
	// Update duplicate statistics
	stats.blocksTotal++
	if isDuplicate {
		stats.duplicatesTotal++
	}
	
	// Recalculate duplicate rate every 100 blocks or at least once per minute
	if stats.blocksTotal%100 == 0 || timestamp.Sub(stats.lastRecalculated) > time.Minute {
		if stats.blocksTotal > 0 {
			stats.duplicateRate = float64(stats.duplicatesTotal) / float64(stats.blocksTotal)
		}
		
		// Adjust adaptive TTL based on block frequency and duplicate rate
		abd.adjustAdaptiveTTL(stats, network)
		stats.lastRecalculated = timestamp
	}
}

// adjustAdaptiveTTL dynamically adjusts the TTL based on network characteristics
func (abd *AdaptiveBlockDeduper) adjustAdaptiveTTL(stats *networkStats, network string) {
	// Baseline: use at least 5x the average time between blocks
	minTTL := stats.avgTimeBetween * 5
	
	// If we're seeing high duplicate rates, increase TTL
	if stats.duplicateRate > 0.3 { // More than 30% duplicates
		factor := 2.0 + stats.duplicateRate*3 // Scale based on duplicate rate
		newTTL := time.Duration(float64(minTTL) * factor)
		
		// Cap at extended TTL
		if newTTL > abd.extendedTTL {
			newTTL = abd.extendedTTL
		}
		
		stats.adaptiveTTL = newTTL
	} else {
		// Lower duplicate rates: use baseline or slightly higher
		factor := 1.0 + stats.duplicateRate*2
		newTTL := time.Duration(float64(minTTL) * factor)
		
		// Always use at least the base TTL
		if newTTL < abd.baseTTL {
			newTTL = abd.baseTTL
		}
		
		stats.adaptiveTTL = newTTL
	}
	
	// Log TTL adjustment if significant change (more than 20%)
	if abd.logger != nil && (float64(stats.adaptiveTTL)/float64(abd.baseTTL) > 1.2 || float64(stats.adaptiveTTL)/float64(abd.baseTTL) < 0.8) {
		abd.logger.Debug("Adjusted block deduplication TTL",
			zap.String("network", network),
			zap.Duration("new_ttl", stats.adaptiveTTL),
			zap.Duration("base_ttl", abd.baseTTL),
			zap.Float64("duplicate_rate", stats.duplicateRate),
			zap.Duration("avg_time_between_blocks", stats.avgTimeBetween))
	}
}

// getTTL returns the appropriate TTL for a network
func (abd *AdaptiveBlockDeduper) getTTL(network string) time.Duration {
	abd.mu.RLock()
	defer abd.mu.RUnlock()
	
	if stats, exists := abd.blockStats[network]; exists {
		stats.mu.RLock()
		defer stats.mu.RUnlock()
		return stats.adaptiveTTL
	}
	
	return abd.baseTTL
}

// removeOldest removes the oldest record
func (abd *AdaptiveBlockDeduper) removeOldest() {
	var oldestKey string
	var oldestTime time.Time
	
	// Find the oldest record
	for key, record := range abd.blocks {
		if oldestTime.IsZero() || record.LastSeen.Before(oldestTime) {
			oldestKey = key
			oldestTime = record.LastSeen
		}
	}
	
	// Remove the oldest record
	if oldestKey != "" {
		delete(abd.blocks, oldestKey)
	}
}

// cleanupLoop periodically cleans up expired records
func (abd *AdaptiveBlockDeduper) cleanupLoop() {
	for range abd.cleanupTicker.C {
		abd.cleanup()
	}
}

// cleanup removes expired records
func (abd *AdaptiveBlockDeduper) cleanup() {
	abd.mu.Lock()
	defer abd.mu.Unlock()
	
	now := time.Now()
	keysToDelete := []string{}
	
	for key, record := range abd.blocks {
		ttl := abd.getTTLNoLock(record.Network)
		if now.Sub(record.LastSeen) > ttl {
			keysToDelete = append(keysToDelete, key)
		}
	}
	
	for _, key := range keysToDelete {
		delete(abd.blocks, key)
	}
	
	if len(keysToDelete) > 0 && abd.logger != nil {
		abd.logger.Debug("Cleaned up expired block records", 
			zap.Int("removed", len(keysToDelete)), 
			zap.Int("remaining", len(abd.blocks)))
	}
}

// getTTLNoLock returns the TTL for a network without locking (caller must hold lock)
func (abd *AdaptiveBlockDeduper) getTTLNoLock(network string) time.Duration {
	if stats, exists := abd.blockStats[network]; exists {
		stats.mu.RLock()
		defer stats.mu.RUnlock()
		return stats.adaptiveTTL
	}
	
	return abd.baseTTL
}

// GetStats returns statistics about the deduplication system
func (abd *AdaptiveBlockDeduper) GetStats() map[string]interface{} {
	abd.mu.RLock()
	defer abd.mu.RUnlock()
	
	stats := make(map[string]interface{})
	stats["total_records"] = len(abd.blocks)
	stats["max_size"] = abd.maxSize
	stats["base_ttl_seconds"] = abd.baseTTL.Seconds()
	
	networkStats := make(map[string]interface{})
	for network, ns := range abd.blockStats {
		ns.mu.RLock()
		netStats := map[string]interface{}{
			"duplicate_rate":           ns.duplicateRate,
			"total_blocks_seen":        ns.blocksTotal,
			"total_duplicates":         ns.duplicatesTotal,
			"adaptive_ttl_seconds":     ns.adaptiveTTL.Seconds(),
			"avg_time_between_seconds": ns.avgTimeBetween.Seconds(),
		}
		ns.mu.RUnlock()
		networkStats[network] = netStats
	}
	stats["networks"] = networkStats
	
	return stats
}

// Stop stops the cleanup ticker
func (abd *AdaptiveBlockDeduper) Stop() {
	if abd.cleanupTicker != nil {
		abd.cleanupTicker.Stop()
	}
}
