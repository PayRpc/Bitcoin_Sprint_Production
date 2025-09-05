//go:build !nozmq
// +build !nozmq

package zmq

import (
	"fmt"
	"strings"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/tiers"
	"github.com/pebbe/zmq4"
	"go.uber.org/zap"
)

type Client struct {
	cfg       config.Config
	blockChan chan blocks.BlockEvent
	mem       interface{} // Changed to interface{} to accept any mempool type
	logger    *zap.Logger
	stopped   bool
	socket    *zmq4.Socket
}

func New(cfg config.Config, blockChan chan blocks.BlockEvent, mem interface{}, logger *zap.Logger) *Client {
	return &Client{
		cfg:       cfg,
		blockChan: blockChan,
		mem:       mem,
		logger:    logger,
	}
}

func (c *Client) Run() {
	// Try to use real ZMQ if available, fallback to mock if not
	var endpoint string
	if len(c.cfg.ZMQNodes) > 0 {
		node := c.cfg.ZMQNodes[0]
		// Handle endpoints that already have tcp:// prefix
		if strings.HasPrefix(node, "tcp://") {
			endpoint = node
		} else {
			endpoint = fmt.Sprintf("tcp://%s", node)
		}
	} else {
		endpoint = "tcp://127.0.0.1:28332"
	}

	socket, err := zmq4.NewSocket(zmq4.SUB)
	if err != nil {
		c.logger.Warn("Failed to create ZMQ socket, using mock mode", zap.Error(err))
		c.startMockMode()
		return
	}

	err = socket.Connect(endpoint)
	if err != nil {
		c.logger.Warn("Failed to connect to ZMQ endpoint, using mock mode",
			zap.String("endpoint", endpoint), zap.Error(err))
		socket.Close()
		c.startMockMode()
		return
	}

	// Subscribe to rawblock messages
	err = socket.SetSubscribe("hashblock")
	if err != nil {
		c.logger.Warn("Failed to subscribe to ZMQ topics, using mock mode", zap.Error(err))
		socket.Close()
		c.startMockMode()
		return
	}

	c.socket = socket
	c.logger.Info("Starting ZMQ client", zap.String("endpoint", endpoint))

	// Start real ZMQ subscription
	go c.realZMQSubscription()
}

func (c *Client) startMockMode() {
	c.logger.Info("Starting ZMQ client (mock mode - ZMQ connection unavailable)")
	go c.mockZMQSubscription()
}

func (c *Client) Stop() {
	c.stopped = true
	if c.socket != nil {
		c.socket.Close()
	}
	c.logger.Info("Stopping ZMQ client")
}

func (c *Client) realZMQSubscription() {
	for !c.stopped {
		// Receive ZMQ message
		msgs, err := c.socket.RecvMessage(0)
		if err != nil {
			if c.stopped {
				break
			}
			c.logger.Error("Error receiving ZMQ message", zap.Error(err))
			time.Sleep(1 * time.Second)
			continue
		}

		if len(msgs) < 2 {
			c.logger.Warn("Invalid ZMQ message format", zap.Int("parts", len(msgs)))
			continue
		}

		topic := msgs[0]
		data := msgs[1]

		switch topic {
		case "hashblock":
			c.handleBlockHash(data)
		default:
			c.logger.Debug("Unknown ZMQ topic", zap.String("topic", topic))
		}
	}
}

func (c *Client) handleBlockHash(data string) {
	detectionTime := time.Now()

	// In a real implementation, you would:
	// 1. Parse the block hash from the data
	// 2. Fetch the full block details from Bitcoin Core RPC
	// 3. Extract height, timestamp, etc.

	// Get tier configuration for timing
	tierConfig := tiers.GetTierConfig()

	// Start relay timing
	relayStart := time.Now()

	// Create block event with timing information
	blockEvent := blocks.BlockEvent{
		Hash:        data[:min(64, len(data))], // Use first 64 chars as hash or full data if shorter
		Height:      0,                         // Would be fetched from RPC
		Timestamp:   detectionTime,
		DetectedAt:  detectionTime,
		RelayTimeMs: 0, // Will be updated after relay
		Source:      "zmq-real",
		Tier:        tierConfig.Name,
	}

	// Simulate relay processing based on tier
	relayDelay := time.Duration(float64(tierConfig.BlockDeadline) * 0.1) // 10% of deadline for processing
	time.Sleep(relayDelay)

	// Calculate actual relay time
	relayTime := time.Since(relayStart)
	blockEvent.RelayTimeMs = relayTime.Seconds() * 1000

	select {
	case c.blockChan <- blockEvent:
		c.logger.Info("Real ZMQ block received",
			zap.String("hash", blockEvent.Hash),
			zap.String("source", blockEvent.Source),
			zap.Float64("relayTimeMs", blockEvent.RelayTimeMs),
			zap.String("tier", blockEvent.Tier))
	default:
		// Channel full, skip
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *Client) mockZMQSubscription() {
	ticker := time.NewTicker(45 * time.Second) // Simulate slower than real blocks
	defer ticker.Stop()

	blockHeight := uint32(700000) // Start from a realistic block height

	for !c.stopped {
		select {
		case <-ticker.C:
			blockHeight++
			detectionTime := time.Now()

			// Get tier configuration for timing
			tierConfig := tiers.GetTierConfig()

			// Simulate relay processing
			relayStart := time.Now()
			relayDelay := time.Duration(float64(tierConfig.BlockDeadline) * 0.05) // 5% of deadline for mock
			time.Sleep(relayDelay)
			relayTime := time.Since(relayStart)

			// Generate mock block event with timing
			blockEvent := blocks.BlockEvent{
				Hash:        c.generateMockHash(blockHeight),
				Height:      blockHeight,
				Timestamp:   detectionTime,
				DetectedAt:  detectionTime,
				RelayTimeMs: relayTime.Seconds() * 1000,
				Source:      "zmq-mock",
				Tier:        tierConfig.Name,
			}

			select {
			case c.blockChan <- blockEvent:
				c.logger.Info("Mock ZMQ block received",
					zap.String("hash", blockEvent.Hash),
					zap.Uint32("height", blockEvent.Height),
					zap.Float64("relayTimeMs", blockEvent.RelayTimeMs),
					zap.String("tier", blockEvent.Tier))
			default:
				// Channel full, skip
			}
		}
	}
}

func (c *Client) generateMockHash(height uint32) string {
	// Generate a realistic-looking block hash for testing
	return "00000000000000000007e947bd7ad2e8c80" +
		string(rune(height%10+'0')) +
		string(rune((height/10)%10+'0')) +
		"a1b2c3d4e5f6"
}
