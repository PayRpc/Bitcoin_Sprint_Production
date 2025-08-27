package broadcaster

import (
	"sync"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"go.uber.org/zap"
)

// Broadcaster manages tier-aware block event publishing to subscribers
type Broadcaster struct {
	subs   map[chan blocks.BlockEvent]config.Tier
	mu     sync.RWMutex
	logger *zap.Logger
}

// New creates a new tier-aware broadcaster
func New(logger *zap.Logger) *Broadcaster {
	return &Broadcaster{
		subs:   make(map[chan blocks.BlockEvent]config.Tier),
		logger: logger,
	}
}

// Subscribe adds a new subscriber with the specified tier
func (b *Broadcaster) Subscribe(tier config.Tier) <-chan blocks.BlockEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Buffer size based on tier
	bufferSize := b.getBufferSize(tier)
	ch := make(chan blocks.BlockEvent, bufferSize)
	b.subs[ch] = tier

	b.logger.Debug("New subscriber added",
		zap.String("tier", string(tier)),
		zap.Int("bufferSize", bufferSize),
		zap.Int("totalSubscribers", len(b.subs)),
	)

	return ch
}

// Unsubscribe removes a subscriber
func (b *Broadcaster) Unsubscribe(ch <-chan blocks.BlockEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Find and remove the channel from our map
	for subCh, tier := range b.subs {
		if subCh == ch {
			delete(b.subs, subCh)
			close(subCh)

			b.logger.Debug("Subscriber removed",
				zap.String("tier", string(tier)),
				zap.Int("remainingSubscribers", len(b.subs)),
			)
			break
		}
	}
}

// Publish broadcasts a block event to all subscribers with tier-aware delivery
func (b *Broadcaster) Publish(evt blocks.BlockEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	start := time.Now()
	delivered := 0
	turboOverwrites := 0
	skipped := 0

	for ch, tier := range b.subs {
		select {
		case ch <- evt:
			delivered++
		default:
			// Channel is full - apply tier-based strategy
			if tier == config.TierTurbo || tier == config.TierEnterprise {
				// Turbo/Enterprise: overwrite old event to ensure latest data
				select {
				case <-ch: // Remove old event
					turboOverwrites++
				default:
					// Channel somehow became available
				}

				select {
				case ch <- evt: // Send new event
					delivered++
				default:
					// Still full, this shouldn't happen with proper buffer sizing
					skipped++
					b.logger.Warn("Turbo channel still full after overwrite",
						zap.String("tier", string(tier)),
						zap.String("blockHash", evt.Hash),
					)
				}
			} else {
				// Lower tiers: just skip if full
				skipped++
			}
		}
	}

	elapsed := time.Since(start)

	// Log performance metrics
	if elapsed > 1*time.Millisecond {
		b.logger.Warn("Slow broadcast detected",
			zap.Duration("elapsed", elapsed),
			zap.Int("delivered", delivered),
			zap.Int("turboOverwrites", turboOverwrites),
			zap.Int("skipped", skipped),
			zap.String("blockHash", evt.Hash),
		)
	} else {
		b.logger.Debug("Block broadcast completed",
			zap.Duration("elapsed", elapsed),
			zap.Int("delivered", delivered),
			zap.Int("turboOverwrites", turboOverwrites),
			zap.Int("skipped", skipped),
			zap.String("blockHash", evt.Hash),
		)
	}
}

// getBufferSize returns the appropriate buffer size for a tier
func (b *Broadcaster) getBufferSize(tier config.Tier) int {
	switch tier {
	case config.TierEnterprise:
		return 4096
	case config.TierTurbo:
		return 2048
	case config.TierBusiness:
		return 1536
	case config.TierPro:
		return 1280
	default: // Free
		return 512
	}
}

// Stats returns current broadcaster statistics
type Stats struct {
	TotalSubscribers  int                     `json:"totalSubscribers"`
	SubscribersByTier map[config.Tier]int     `json:"subscribersByTier"`
	BufferUtilization map[config.Tier]float64 `json:"bufferUtilization"`
}

// GetStats returns current broadcaster statistics
func (b *Broadcaster) GetStats() Stats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	stats := Stats{
		TotalSubscribers:  len(b.subs),
		SubscribersByTier: make(map[config.Tier]int),
		BufferUtilization: make(map[config.Tier]float64),
	}

	// Count subscribers by tier and calculate buffer utilization
	bufferCounts := make(map[config.Tier]int)
	bufferTotals := make(map[config.Tier]int)

	for ch, tier := range b.subs {
		stats.SubscribersByTier[tier]++

		// Calculate buffer utilization
		bufferLen := len(ch)
		bufferCap := cap(ch)

		bufferCounts[tier] += bufferLen
		bufferTotals[tier] += bufferCap
	}

	// Calculate average utilization per tier
	for tier, total := range bufferTotals {
		if total > 0 {
			stats.BufferUtilization[tier] = float64(bufferCounts[tier]) / float64(total) * 100
		}
	}

	return stats
}
