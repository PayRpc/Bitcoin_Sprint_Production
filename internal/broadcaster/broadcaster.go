package broadcaster

import (
	"bytes"
	"encoding/json"
	"sync"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"go.uber.org/zap"
)

// PreEncodedFrame holds pre-serialized message data for reuse
type PreEncodedFrame struct {
	data     []byte
	size     int
	created  time.Time
}

// BlockMessage represents the message structure sent to clients
type BlockMessage struct {
	Type      string            `json:"type"`
	Block     blocks.BlockEvent `json:"block"`
	Timestamp time.Time         `json:"timestamp"`
}

// BatchedBroadcast represents a batched broadcast event with pre-encoded frame
type BatchedBroadcast struct {
	event   blocks.BlockEvent
	frame   *PreEncodedFrame
	clients []chan blocks.BlockEvent
	tiers   []config.Tier
}

// Broadcaster manages tier-aware block event publishing to subscribers with fan-out batching
type Broadcaster struct {
	subs        map[chan blocks.BlockEvent]config.Tier
	mu          sync.RWMutex
	logger      *zap.Logger
	batchChan   chan BatchedBroadcast
	stopChan    chan struct{}
	wg          sync.WaitGroup
	framePool   sync.Pool // Pool for reusing byte buffers
}

// New creates a new tier-aware broadcaster with fan-out batching and pre-encoded frames
func New(logger *zap.Logger) *Broadcaster {
	b := &Broadcaster{
		subs:      make(map[chan blocks.BlockEvent]config.Tier),
		logger:    logger,
		batchChan: make(chan BatchedBroadcast, 1000),
		stopChan:  make(chan struct{}),
		framePool: sync.Pool{
			New: func() interface{} {
				return &bytes.Buffer{}
			},
		},
	}
	
	// Start the batching worker
	b.wg.Add(1)
	go b.fanOutBatcher()
	
	return b
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

// Publish publishes a block event to all subscribers with pre-encoded frames and fan-out batching
func (b *Broadcaster) Publish(event blocks.BlockEvent) {
	b.mu.RLock()
	if len(b.subs) == 0 {
		b.mu.RUnlock()
		return
	}
	
	// Pre-encode the message once for all subscribers
	frame, err := b.createPreEncodedFrame(event)
	if err != nil {
		b.logger.Error("Failed to create pre-encoded frame", zap.Error(err))
		b.mu.RUnlock()
		return
	}
	
	// Collect all subscribers for batching
	clients := make([]chan blocks.BlockEvent, 0, len(b.subs))
	tiers := make([]config.Tier, 0, len(b.subs))
	
	for ch, tier := range b.subs {
		clients = append(clients, ch)
		tiers = append(tiers, tier)
	}
	b.mu.RUnlock()
	
	// Send to batch channel for aggregated writes
	select {
	case b.batchChan <- BatchedBroadcast{
		event:   event,
		frame:   frame,
		clients: clients,
		tiers:   tiers,
	}:
	default:
		// Channel full, skip this broadcast
		b.logger.Warn("Batch channel full, dropping broadcast")
	}
}

// createPreEncodedFrame serializes the block event once for reuse across all connections
func (b *Broadcaster) createPreEncodedFrame(event blocks.BlockEvent) (*PreEncodedFrame, error) {
	// Get buffer from pool
	buf := b.framePool.Get().(*bytes.Buffer)
	buf.Reset()
	defer b.framePool.Put(buf)
	
	// Create the message structure
	message := BlockMessage{
		Type:      "block_event",
		Block:     event,
		Timestamp: time.Now(),
	}
	
	// Encode to JSON
	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(message); err != nil {
		return nil, err
	}
	
	// Create frame with copy of the data
	frame := &PreEncodedFrame{
		data:    make([]byte, buf.Len()),
		size:    buf.Len(),
		created: time.Now(),
	}
	copy(frame.data, buf.Bytes())
	
	return frame, nil
}

// fanOutBatcher implements the 5ms tick aggregation with up to 64 clients per batch
func (b *Broadcaster) fanOutBatcher() {
	defer b.wg.Done()
	
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	
	var pendingBroadcasts []BatchedBroadcast
	const maxBatchSize = 64
	
	for {
		select {
		case <-b.stopChan:
			// Flush remaining broadcasts before stopping
			b.flushBroadcasts(pendingBroadcasts)
			return
			
		case broadcast := <-b.batchChan:
			pendingBroadcasts = append(pendingBroadcasts, broadcast)
			
			// If we hit the max batch size, flush immediately
			if len(pendingBroadcasts) >= maxBatchSize {
				b.flushBroadcasts(pendingBroadcasts)
				pendingBroadcasts = pendingBroadcasts[:0] // Clear slice while keeping capacity
			}
			
		case <-ticker.C:
			// 5ms tick - flush all pending broadcasts
			if len(pendingBroadcasts) > 0 {
				b.flushBroadcasts(pendingBroadcasts)
				pendingBroadcasts = pendingBroadcasts[:0] // Clear slice while keeping capacity
			}
		}
	}
}

// flushBroadcasts writes all pending broadcasts to their respective channels
func (b *Broadcaster) flushBroadcasts(broadcasts []BatchedBroadcast) {
	for _, broadcast := range broadcasts {
		for i, ch := range broadcast.clients {
			tier := broadcast.tiers[i]
			
			select {
			case ch <- broadcast.event:
				// Successfully sent
			default:
				// Channel full - handle based on tier
				if tier == config.TierFree {
					// Free tier: drop the event
					b.logger.Debug("Dropping event for free tier subscriber (channel full)")
				} else {
					// Paid tiers: try to overwrite
					select {
					case <-ch: // Remove old event
						select {
						case ch <- broadcast.event: // Try to send new event
						default:
							b.logger.Warn("Failed to overwrite for paid tier subscriber")
						}
					default:
						b.logger.Warn("Failed to overwrite for paid tier subscriber (channel empty)")
					}
				}
			}
		}
	}
}

// Close stops the broadcaster and its batching worker
func (b *Broadcaster) Close() {
	close(b.stopChan)
	b.wg.Wait()
	close(b.batchChan)
}// getBufferSize returns the appropriate buffer size for a tier
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
