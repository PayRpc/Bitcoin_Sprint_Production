// Package api provides the main entry point for the Bitcoin Sprint API server
package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/cache"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
	"go.uber.org/zap"
)

// ===== MAIN SERVER ENTRY POINT =====

// StartServer starts the API server with all components initialized
func StartServer(cfg config.Config, blockChan chan blocks.BlockEvent, mem *mempool.Mempool, cache *cache.Cache, logger *zap.Logger) error {
	// Create server instance
	server := NewWithCache(cfg, blockChan, mem, cache, logger)

	// Initialize Bloom Filter if enabled
	if cfg.BloomFilterEnabled {
		server.bloomFilter = NewBloomFilterManager(cfg)
	}

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Public endpoints (with auth)
	mux.HandleFunc("/v1/latest", server.recoveryMiddleware(server.auth(server.latestHandler)))
	mux.HandleFunc("/v1/stream", server.recoveryMiddleware(server.auth(server.streamHandler)))
	mux.HandleFunc("/v1/status", server.recoveryMiddleware(server.auth(server.statusHandler)))
	mux.HandleFunc("/v1/mempool", server.recoveryMiddleware(server.auth(server.mempoolHandler)))
	mux.HandleFunc("/v1/generate-key", server.recoveryMiddleware(server.generateKeyHandler))
	mux.HandleFunc("/v1/analytics/summary", server.recoveryMiddleware(server.auth(server.analyticsSummaryHandler)))
	mux.HandleFunc("/v1/license/info", server.recoveryMiddleware(server.auth(server.licenseInfoHandler)))

	// Multi-chain endpoints
	mux.HandleFunc("/chains", server.recoveryMiddleware(server.chainsHandler))
	mux.HandleFunc("/v1/", server.recoveryMiddleware(server.chainAwareHandler))

	// Public endpoints (no auth required)
	mux.HandleFunc("/health", server.recoveryMiddleware(server.healthHandler))
	mux.HandleFunc("/version", server.recoveryMiddleware(server.versionHandler))

	// Apply security middleware
	secureHandler := server.securityMiddleware(mux)

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	// Try to start server with port auto-retry
	basePort := cfg.APIPort
	maxRetries := 10
	var finalAddr string

	// Find an available port
	for retry := 0; retry < maxRetries; retry++ {
		port := basePort + retry
		addr := cfg.APIHost + ":" + strconv.Itoa(port)

		// Create server with timeouts and security settings
		server.srv = &http.Server{
			Addr:         addr,
			Handler:      secureHandler,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 60 * time.Second,
			IdleTimeout:  120 * time.Second,
		}

		logger.Info("API attempting to start", zap.String("addr", addr), zap.Int("attempt", retry+1))

		// Try to bind to this port
		listener, bindErr := net.Listen("tcp", addr)
		if bindErr != nil {
			logger.Warn("Port busy, trying next", zap.String("addr", addr), zap.Error(bindErr))
			continue
		}

		// Port is available, start server in a goroutine
		finalAddr = addr
		logger.Info("API binding successful", zap.String("addr", finalAddr))

		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Info("API server started", zap.String("addr", finalAddr))

			// Start serving (this blocks until Shutdown is called or an error occurs)
			if err := server.srv.Serve(listener); err != nil && err != http.ErrServerClosed {
				logger.Error("API server error", zap.Error(err))
			}
		}()

		break
	}

	// If we exhausted all port retries
	if finalAddr == "" {
		logger.Error("Failed to bind to any port",
			zap.Int("basePort", basePort),
			zap.Int("maxRetries", maxRetries))
		return fmt.Errorf("failed to bind to any port")
	}

	// Start predictive prefetch
	server.startPredictivePrefetch(ctx)

	// Wait for interrupt signal
	go func() {
		<-sigChan
		logger.Info("Shutdown signal received, gracefully stopping server...")

		// Create a timeout context for shutdown
		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 15*time.Second)
		defer shutdownCancel()

		// Attempt graceful shutdown
		if err := server.srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("Server shutdown error", zap.Error(err))
		}

		cancel() // Cancel the main context to signal all operations to stop
	}()

	// Block until context is cancelled and all goroutines finish
	<-ctx.Done()
	wg.Wait()
	logger.Info("Server gracefully stopped")

	return nil
}

// startPredictivePrefetch starts a background worker that prefetches N+1/N+2 headers for cache warming
func (s *Server) startPredictivePrefetch(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Prefetch for each registered backend
				for _, chain := range s.backends.List() {
					if backend, exists := s.backends.Get(chain); exists {
						// Trigger predictive warm-up by calling GetLatestBlock
						// This ensures the cache is hot for subsequent requests
						go func(b ChainBackend, chainName string) {
							_, err := b.GetLatestBlock()
							if err != nil {
								s.logger.Debug("Prefetch failed for chain",
									zap.String("chain", chainName),
									zap.Error(err))
							} else {
								s.logger.Debug("Prefetch completed for chain", zap.String("chain", chainName))
							}
						}(backend, chain)
					}
				}
			}
		}
	}()
}
