package p2p

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"go.uber.org/zap"
)

// NewDirect creates a direct P2P connection to a Bitcoin node
func NewDirect(ctx context.Context, peerAddr string, blockChan chan blocks.BlockEvent, logger *zap.Logger) error {
	logger.Info("Starting direct P2P connection", zap.String("peer", peerAddr))

	// Connect to the peer
	conn, err := net.DialTimeout("tcp", peerAddr, 10*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Handle the connection in a goroutine
	go func() {
		defer conn.Close()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Simulate block detection for now
				// In a real implementation, this would parse Bitcoin protocol messages
				time.Sleep(30 * time.Second)

				blockEvent := blocks.BlockEvent{
					Hash:      "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f",
					Height:    1,
					Timestamp: time.Now(),
					Source:    "direct-p2p",
				}

				select {
				case blockChan <- blockEvent:
					logger.Info("Direct P2P block detected", zap.String("hash", blockEvent.Hash))
				default:
					// Channel full, skip
				}
			}
		}
	}()

	return nil
}

// NewMemoryWatcher creates an in-memory block watcher for testing
func NewMemoryWatcher(ctx context.Context, blockChan chan blocks.BlockEvent, logger *zap.Logger) {
	logger.Info("Starting memory block watcher")

	var blockHeight int64 = 1
	var mu sync.Mutex

	go func() {
		ticker := time.NewTicker(15 * time.Second) // Faster than real blocks for testing
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mu.Lock()
				blockHeight++

				blockEvent := blocks.BlockEvent{
					Hash:      generateTestHash(blockHeight),
					Height:    uint32(blockHeight),
					Timestamp: time.Now(),
					Source:    "memory-watcher",
				}
				mu.Unlock()

				select {
				case blockChan <- blockEvent:
					logger.Info("Memory watcher generated block",
						zap.String("hash", blockEvent.Hash),
						zap.Uint32("height", blockEvent.Height))
				default:
					// Channel full, skip
				}
			}
		}
	}()
}

// generateTestHash creates a test block hash based on height
func generateTestHash(height int64) string {
	// Simple hash generation for testing
	// In reality, this would be the actual block hash
	return "000000000000000000000000000000000000000000000000000000000000" +
		string(rune(height%10+'0')) + string(rune((height/10)%10+'0'))
}
