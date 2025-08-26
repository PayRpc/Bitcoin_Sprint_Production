package zmq

import (
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
	"go.uber.org/zap"
)

type Client struct {
	cfg       config.Config
	blockChan chan blocks.BlockEvent
	mem       *mempool.Mempool
	logger    *zap.Logger
	stopped   bool
}

func New(cfg config.Config, blockChan chan blocks.BlockEvent, mem *mempool.Mempool, logger *zap.Logger) *Client {
	return &Client{
		cfg:       cfg,
		blockChan: blockChan,
		mem:       mem,
		logger:    logger,
	}
}

func (c *Client) Run() {
	c.logger.Info("Starting ZMQ client (mock mode - ZMQ C library not available)")
	
	// Mock ZMQ functionality for demonstration
	// In production, this would connect to Bitcoin Core's ZMQ interface
	go c.mockZMQSubscription()
}

func (c *Client) Stop() {
	c.stopped = true
	c.logger.Info("Stopping ZMQ client")
}

func (c *Client) mockZMQSubscription() {
	ticker := time.NewTicker(45 * time.Second) // Simulate slower than real blocks
	defer ticker.Stop()
	
	blockHeight := uint32(700000) // Start from a realistic block height
	
	for !c.stopped {
		select {
		case <-ticker.C:
			blockHeight++
			
			// Generate mock block event
			blockEvent := blocks.BlockEvent{
				Hash:      c.generateMockHash(blockHeight),
				Height:    blockHeight,
				Timestamp: time.Now(),
				Source:    "zmq-mock",
			}
			
			select {
			case c.blockChan <- blockEvent:
				c.logger.Info("Mock ZMQ block received", 
					zap.String("hash", blockEvent.Hash),
					zap.Uint32("height", blockEvent.Height))
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
