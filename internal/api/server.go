// Package api provides the main HTTP API server for Bitcoin Sprint
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/cache"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
	"go.uber.org/zap"
)

// ===== SERVER STRUCT AND LIFECYCLE =====

// Server represents the main API server with all dependencies
type Server struct {
	cfg       config.Config
	blockChan chan blocks.BlockEvent
	mem       *mempool.Mempool
	cache     *cache.Cache
	logger    *zap.Logger
	srv       *http.Server // Public API server
	adminSrv  *http.Server // Admin-only server

	// Rate limiting
	rateLimiter *RateLimiter

	// Customer key management
	keyManager *CustomerKeyManager

	// Admin authentication
	adminAuth *AdminAuth

	// WebSocket connection limits
	wsLimiter *WebSocketLimiter

	// Predictive analytics
	predictor *PredictiveAnalytics

	// Tier-aware circuit breaker
	circuitBreaker *CircuitBreaker

	// Blockchain-agnostic backends
	backends *BackendRegistry

	// High-performance Bloom Filter for UTXO lookups
	bloomFilter *BloomFilterManager

	// Injected dependencies for determinism
	clock      Clock
	randReader RandomReader
}

// New creates a new API server instance
func New(cfg config.Config, blockChan chan blocks.BlockEvent, mem *mempool.Mempool, logger *zap.Logger) *Server {
	clock := RealClock{}
	randReader := RealRandomReader{}

	server := &Server{
		cfg:            cfg,
		blockChan:      blockChan,
		mem:            mem,
		logger:         logger,
		rateLimiter:    NewRateLimiter(clock),
		keyManager:     NewCustomerKeyManagerWithConfig(cfg, clock, randReader),
		adminAuth:      NewAdminAuth(),
		wsLimiter:      NewWebSocketLimiter(cfg.WebSocketMaxGlobal, cfg.WebSocketMaxPerIP, cfg.WebSocketMaxPerChain),
		predictor:      NewPredictiveAnalytics(clock),
		circuitBreaker: NewCircuitBreaker(cfg.Tier, clock),
		backends:       NewBackendRegistry(),
		clock:          clock,
		randReader:     randReader,
	}

	// Initialize default Bitcoin backend
	server.backends.Register("btc", &BitcoinBackend{
		blockChan: blockChan,
		mem:       mem,
		cfg:       cfg,
	})

	return server
}

// NewWithCache creates a new API server instance with cache support
func NewWithCache(cfg config.Config, blockChan chan blocks.BlockEvent, mem *mempool.Mempool, cache *cache.Cache, logger *zap.Logger) *Server {
	clock := RealClock{}
	randReader := RealRandomReader{}

	server := &Server{
		cfg:            cfg,
		blockChan:      blockChan,
		mem:            mem,
		cache:          cache,
		logger:         logger,
		rateLimiter:    NewRateLimiter(clock),
		keyManager:     NewCustomerKeyManagerWithConfig(cfg, clock, randReader),
		adminAuth:      NewAdminAuth(),
		wsLimiter:      NewWebSocketLimiter(cfg.WebSocketMaxGlobal, cfg.WebSocketMaxPerIP, cfg.WebSocketMaxPerChain),
		predictor:      NewPredictiveAnalytics(clock),
		circuitBreaker: NewCircuitBreaker(cfg.Tier, clock),
		backends:       NewBackendRegistry(),
		clock:          clock,
		randReader:     randReader,
	}

	// Initialize default Bitcoin backend
	server.backends.Register("btc", &BitcoinBackend{
		blockChan: blockChan,
		mem:       mem,
		cfg:       cfg,
		cache:     cache,
	})

	return server
}

// Stop gracefully shuts down the server
func (s *Server) Stop() {
	if s.srv != nil {
		// Create a timeout context for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := s.srv.Shutdown(ctx); err != nil {
			s.logger.Error("Server shutdown error", zap.Error(err))
		}
	}
}

// Run starts the API server and blocks until shutdown
func (s *Server) Run(ctx context.Context) {
	s.logger.Info("Starting API server",
		zap.String("addr", fmt.Sprintf("%s:%d", s.cfg.APIHost, s.cfg.APIPort)))
	
	// TODO: Implement proper server startup logic
	// This is a placeholder until the API module is fully restructured
	s.logger.Info("API server run method called - implement server startup")
}
