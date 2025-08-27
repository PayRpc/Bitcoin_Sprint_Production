// Package api provides predictive analytics functionality
package api

import (
	"sync"
	"time"
)

// ===== PREDICTIVE ANALYTICS IMPLEMENTATION =====

// PredictiveAnalytics provides predictive analytics for block timing and fee estimation
type PredictiveAnalytics struct {
	blockHistory []BlockTiming
	clock        Clock
	mu           sync.RWMutex
}

// BlockTiming represents timing information for a block
type BlockTiming struct {
	Height    int64
	Timestamp time.Time
	Size      int
}

// NewPredictiveAnalytics creates a new predictive analytics handler
func NewPredictiveAnalytics(clock Clock) *PredictiveAnalytics {
	return &PredictiveAnalytics{
		blockHistory: make([]BlockTiming, 0, 100), // Keep last 100 blocks
		clock:        clock,
	}
}

// RecordBlock records a new block for predictive analytics
func (pa *PredictiveAnalytics) RecordBlock(height int64, size int) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	block := BlockTiming{
		Height:    height,
		Timestamp: pa.clock.Now(),
		Size:      size,
	}

	pa.blockHistory = append(pa.blockHistory, block)

	// Keep only the last 100 blocks
	if len(pa.blockHistory) > 100 {
		pa.blockHistory = pa.blockHistory[1:]
	}
}

// PredictNextBlockETA predicts the ETA for the next block
func (pa *PredictiveAnalytics) PredictNextBlockETA() float64 {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	if len(pa.blockHistory) < 2 {
		return 420.0 // Default 7 minutes
	}

	// Calculate average interval between recent blocks
	var totalInterval float64
	count := 0

	for i := 1; i < len(pa.blockHistory); i++ {
		interval := pa.blockHistory[i].Timestamp.Sub(pa.blockHistory[i-1].Timestamp).Seconds()
		if interval > 0 && interval < 3600 { // Reasonable bounds (max 1 hour)
			totalInterval += interval
			count++
		}
	}

	if count == 0 {
		return 420.0
	}

	return totalInterval / float64(count)
}

// GetAnalyticsSummary returns a summary of predictive analytics data
func (pa *PredictiveAnalytics) GetAnalyticsSummary() map[string]interface{} {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	summary := map[string]interface{}{
		"total_blocks_recorded":  len(pa.blockHistory),
		"next_block_eta_seconds": pa.PredictNextBlockETA(),
		"timestamp":              pa.clock.Now().UTC().Format(time.RFC3339),
	}

	if len(pa.blockHistory) > 0 {
		latest := pa.blockHistory[len(pa.blockHistory)-1]
		summary["latest_block_height"] = latest.Height
		summary["latest_block_timestamp"] = latest.Timestamp.Format(time.RFC3339)
		summary["latest_block_size"] = latest.Size
	}

	return summary
}
