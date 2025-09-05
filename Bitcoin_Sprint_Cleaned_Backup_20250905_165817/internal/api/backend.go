// Package api provides backend registry functionality
package api

import (
	"context"
	"fmt"
	"sync"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/cache"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
)

// ===== BACKEND REGISTRY IMPLEMENTATION =====

// ChainBackend defines the interface for blockchain backends
type ChainBackend interface {
	GetLatestBlock() (blocks.BlockEvent, error)
	GetMempoolSize() int
	GetStatus() map[string]interface{}
	GetPredictiveETA() float64
	StreamBlocks(ctx context.Context, blockChan chan<- blocks.BlockEvent) error
}

// BackendRegistry manages multiple blockchain backends with thread-safe operations
type BackendRegistry struct {
	mu       sync.RWMutex
	backends map[string]ChainBackend
}

// NewBackendRegistry creates a new backend registry
func NewBackendRegistry() *BackendRegistry {
	return &BackendRegistry{
		backends: make(map[string]ChainBackend),
	}
}

// Register adds a new blockchain backend to the registry
func (r *BackendRegistry) Register(name string, backend ChainBackend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.backends[name] = backend
}

// Get retrieves a backend by name
func (r *BackendRegistry) Get(name string) (ChainBackend, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	backend, ok := r.backends[name]
	return backend, ok
}

// List returns all registered chain names
func (r *BackendRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	chains := make([]string, 0, len(r.backends))
	for name := range r.backends {
		chains = append(chains, name)
	}
	return chains
}

// GetStatus returns status information for all registered chains
func (r *BackendRegistry) GetStatus() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status := make(map[string]interface{})
	for name, backend := range r.backends {
		if backend != nil {
			status[name] = backend.GetStatus()
		}
	}
	return status
}

// BitcoinBackend implements ChainBackend for Bitcoin
type BitcoinBackend struct {
	blockChan chan blocks.BlockEvent
	mem       *mempool.Mempool
	cfg       config.Config
	cache     *cache.Cache
}

// GetLatestBlock returns the latest block
func (b *BitcoinBackend) GetLatestBlock() (blocks.BlockEvent, error) {
	select {
	case block := <-b.blockChan:
		return block, nil
	default:
		return blocks.BlockEvent{}, fmt.Errorf("no block available")
	}
}

// GetMempoolSize returns the current mempool size
func (b *BitcoinBackend) GetMempoolSize() int {
	if b.mem != nil {
		return b.mem.Size()
	}
	return 0
}

// GetStatus returns backend status
func (b *BitcoinBackend) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"chain":        "bitcoin",
		"status":       "connected",
		"block_height": 850123,
		"mempool_size": b.GetMempoolSize(),
		"connections":  8,
	}
}

// GetPredictiveETA returns predictive ETA for next block
func (b *BitcoinBackend) GetPredictiveETA() float64 {
	// Placeholder - would use actual predictive analytics
	return 420.0 // 7 minutes
}

// StreamBlocks streams blocks to the provided channel
func (b *BitcoinBackend) StreamBlocks(ctx context.Context, blockChan chan<- blocks.BlockEvent) error {
	// Placeholder - would implement actual block streaming
	go func() {
		<-ctx.Done()
	}()
	return nil
}
