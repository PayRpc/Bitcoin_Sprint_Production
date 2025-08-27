//go:build nozmq
// +build nozmq

package zmq

import (
	"context"
	"fmt"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
	"github.com/PayRpc/Bitcoin-Sprint/internal/tiers"
	"go.uber.org/zap"
)

// Client is a ZMQ client that tries real ZMQ first, falls back to mock
type Client struct {
	cfg       config.Config
	blockChan chan blocks.BlockEvent
	mem       *mempool.Mempool
	logger    *zap.Logger
	stopped   bool
	cancel    context.CancelFunc
	socket    interface{} // Use interface{} to avoid import issues
}

// New returns a ZMQ client that tries real ZMQ first, falls back to mock
func New(cfg config.Config, blockChan chan blocks.BlockEvent, mem *mempool.Mempool, logger *zap.Logger) *Client {
	logger.Info("ZMQ client: attempting real ZMQ connection (with fallback to mock)")
	return &Client{
		cfg:       cfg,
		blockChan: blockChan,
		mem:       mem,
		logger:    logger,
	}
}

// Run starts the ZMQ client with real ZMQ attempt, fallback to mock
func (c *Client) Run() {
	// Try to use real ZMQ if available
	var endpoint string
	if len(c.cfg.ZMQNodes) > 0 {
		endpoint = fmt.Sprintf("tcp://%s", c.cfg.ZMQNodes[0])
	} else {
		endpoint = "tcp://127.0.0.1:28332"
	}

	c.logger.Info("Attempting real ZMQ connection", zap.String("endpoint", endpoint))

	// Try to connect to real ZMQ (this will fail in nozmq build, triggering mock mode)
	if c.tryRealZMQConnection(endpoint) {
		c.logger.Info("Real ZMQ connection successful")
		go c.realZMQSubscription()
	} else {
		c.logger.Info("Real ZMQ connection failed, using enhanced mock mode")
		ctx, cancel := context.WithCancel(context.Background())
		c.cancel = cancel
		go c.mockZMQSubscription(ctx)
	}
}

// tryRealZMQConnection attempts to connect to real ZMQ
func (c *Client) tryRealZMQConnection(endpoint string) bool {
	// In nozmq build, this will always return false
	// In real build, this would try actual ZMQ connection
	defer func() {
		if r := recover(); r != nil {
			c.logger.Debug("ZMQ connection attempt failed (expected in nozmq build)", zap.Any("error", r))
		}
	}()

	// This code would work in real ZMQ build:
	// socket, err := zmq4.NewSocket(zmq4.SUB)
	// if err != nil {
	//     return false
	// }
	// err = socket.Connect(endpoint)
	// if err != nil {
	//     socket.Close()
	//     return false
	// }
	// err = socket.SetSubscribe("hashblock")
	// if err != nil {
	//     socket.Close()
	//     return false
	// }
	// c.socket = socket
	// return true

	return false // Always fallback to mock in nozmq build
}

// realZMQSubscription would handle real ZMQ messages (placeholder)
func (c *Client) realZMQSubscription() {
	// This would contain real ZMQ subscription logic
	// For now, just call mock since we're in nozmq build
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.mockZMQSubscription(ctx)
}

// Stop stops the client
func (c *Client) Stop() {
	c.logger.Info("Stopping ZMQ client")
	c.stopped = true
	if c.cancel != nil {
		c.cancel()
	}
	if c.socket != nil {
		// In real build: c.socket.Close()
		c.logger.Debug("Socket closed")
	}
	c.logger.Info("ZMQ client stopped")
}

// mockZMQSubscription simulates realistic Bitcoin block detection
func (c *Client) mockZMQSubscription(ctx context.Context) {
	// Simulate realistic Bitcoin block timing (avg 10 minutes, but variable)
	baseInterval := 10 * time.Minute
	if c.cfg.MockFastBlocks {
		baseInterval = 30 * time.Second // For testing/demo purposes
	}

	ticker := time.NewTicker(baseInterval)
	defer ticker.Stop()

	blockHeight := uint32(860000) // Start from current realistic block height (Aug 2025)
	c.logger.Info("Mock ZMQ starting block simulation",
		zap.Uint32("startingHeight", blockHeight),
		zap.Duration("baseInterval", baseInterval))

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Mock ZMQ subscription stopped")
			return
		case <-ticker.C:
			if c.stopped {
				return
			}

			blockHeight++
			detectionTime := time.Now()

			// Get tier configuration for realistic timing simulation
			tierConfig := tiers.GetTierConfig()

			// Simulate realistic relay processing time
			relayStart := time.Now()
			// Mock relay delay: 2-8ms for enterprise, 50-200ms for free tier
			var relayDelay time.Duration
			switch tierConfig.Name {
			case "ENTERPRISE", "PRO":
				relayDelay = time.Duration(2+blockHeight%6) * time.Millisecond
			case "FREE":
				relayDelay = time.Duration(50+(blockHeight%150)) * time.Millisecond
			default:
				relayDelay = time.Duration(10+(blockHeight%40)) * time.Millisecond
			}

			time.Sleep(relayDelay)
			relayTime := time.Since(relayStart)

			// Generate realistic block event
			blockEvent := blocks.BlockEvent{
				Hash:        c.generateMockHash(blockHeight),
				Height:      blockHeight,
				Timestamp:   detectionTime,
				DetectedAt:  detectionTime,
				RelayTimeMs: relayTime.Seconds() * 1000,
				Source:      "zmq-mock-enhanced",
				Tier:        tierConfig.Name,
			}

			select {
			case c.blockChan <- blockEvent:
				c.logger.Info("Mock ZMQ block detected",
					zap.String("hash", blockEvent.Hash),
					zap.Uint32("height", blockEvent.Height),
					zap.Float64("relayTimeMs", blockEvent.RelayTimeMs),
					zap.String("tier", blockEvent.Tier),
					zap.String("source", "mock-simulation"))
			default:
				c.logger.Warn("Block channel full, skipping mock block",
					zap.Uint32("height", blockEvent.Height))
			}

			// Vary the next block timing (8-15 minutes typically)
			nextInterval := time.Duration(8+blockHeight%7) * time.Minute
			if c.cfg.MockFastBlocks {
				nextInterval = time.Duration(20+(blockHeight%40)) * time.Second
			}
			ticker.Reset(nextInterval)
		}
	}
}

// generateMockHash creates realistic-looking Bitcoin block hashes
func (c *Client) generateMockHash(height uint32) string {
	// Bitcoin block hashes start with zeros and contain hex characters
	// This generates a realistic-looking hash with proper leading zeros
	baseHash := "000000000000000000"
	heightStr := ""

	// Encode height into the hash in a realistic way
	h := height
	for i := 0; i < 8; i++ {
		char := "0123456789abcdef"[h%16]
		heightStr = string(char) + heightStr
		h /= 16
	}

	// Add some randomness based on current time and height
	now := time.Now().UnixNano()
	randomPart := ""
	for i := 0; i < 40; i++ {
		char := "0123456789abcdef"[(now+int64(height)*int64(i))%16]
		randomPart += string(char)
	}

	return baseHash + heightStr + randomPart[:32]
}
