package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/api"
	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/broadcaster"
	"github.com/PayRpc/Bitcoin-Sprint/internal/cache"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/database"
	"github.com/PayRpc/Bitcoin-Sprint/internal/entropy"
	"github.com/PayRpc/Bitcoin-Sprint/internal/license"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
	"github.com/PayRpc/Bitcoin-Sprint/internal/p2p"
	"github.com/PayRpc/Bitcoin-Sprint/internal/performance"
	"github.com/PayRpc/Bitcoin-Sprint/internal/relay"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Initialize structured logging
	logger, err := initLogger()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting Bitcoin Sprint",
		zap.String("version", "1.0.0"),
		zap.String("go_version", runtime.Version()),
		zap.Int("cpu_cores", runtime.NumCPU()))

	// Load configuration
	cfg := config.Load()
	logger.Info("Configuration loaded",
		zap.String("tier", string(cfg.Tier)),
		zap.String("default_chain", cfg.DefaultChain),
		zap.Bool("optimize_system", cfg.OptimizeSystem))

	// Validate license if provided
	if cfg.LicenseKey != "" {
		if !license.Validate(cfg.LicenseKey) {
			logger.Fatal("Invalid license key")
		}
		licenseInfo := license.GetInfo(cfg.LicenseKey)
		logger.Info("License validated",
			zap.String("tier", licenseInfo.Tier),
			zap.Bool("valid", licenseInfo.Valid))
	}

	// Initialize performance optimizations
	if cfg.OptimizeSystem {
		perfManager := performance.New(cfg, logger)
		if err := perfManager.ApplyOptimizations(); err != nil {
			logger.Error("Failed to apply performance optimizations", zap.Error(err))
		} else {
			logger.Info("Performance optimizations applied")
		}
	}

	// Initialize entropy for cryptographic operations (no explicit initialization needed)
	logger.Info("Entropy system ready")

	// Initialize database connection
	var db *database.DB
	if cfg.EnablePersistence {
		dbConfig := database.Config{
			Type:     cfg.DatabaseType,
			URL:      cfg.DatabaseURL,
			MaxConns: 10,
			MinConns: 2,
		}
		db, err = database.New(dbConfig, logger)
		if err != nil {
			logger.Fatal("Failed to connect to database", zap.Error(err))
		}
		defer db.Close()
		logger.Info("Database connection established")
	}

	// Initialize core components
	blockChan := make(chan blocks.BlockEvent, 1000)
	mem := mempool.New()
	cache := cache.New(1000, logger)

	// Initialize broadcaster for real-time updates
	broadcaster := broadcaster.New(logger)

	// Initialize relay dispatcher for multi-chain support
	relayDispatcher := relay.NewRelayDispatcher(cfg, logger)

	// Initialize P2P client for Bitcoin network
	p2pClient, err := p2p.New(cfg, blockChan, mem, logger)
	if err != nil {
		logger.Fatal("Failed to create P2P client", zap.Error(err))
	}

	// Initialize runtime optimizer
	runtime.ApplySystemOptimizations(logger)

	// Initialize API server
	server := api.New(cfg, blockChan, mem, logger)

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start all services
	serviceErrors := make(chan error, 10)

	// Start P2P client
	go func() {
		logger.Info("Starting P2P client")
		p2pClient.Run()
	}()

	// Start relay dispatcher
	go func() {
		logger.Info("Starting relay dispatcher")
		if err := relayDispatcher.StreamAllBlocks(ctx, blockChan); err != nil {
			serviceErrors <- fmt.Errorf("relay dispatcher error: %w", err)
		}
	}()

	// Start broadcaster (no explicit start needed, it's passive)
	logger.Info("Broadcaster initialized")

	// Start runtime optimizer (apply optimizations)
	logger.Info("Runtime optimizations applied (placeholder)")

	// Start API server in a goroutine
	go func() {
		logger.Info("Starting API server")
		server.Run(ctx)
	}()

	// Start admin server if configured
	if cfg.AdminPort > 0 {
		logger.Info("Admin server not implemented yet", zap.Int("port", cfg.AdminPort))
	}

	// Start Prometheus metrics server if enabled
	if cfg.EnablePrometheus {
		go func() {
			logger.Info("Starting Prometheus metrics server", zap.Int("port", cfg.PrometheusPort))
			if err := startMetricsServer(ctx, cfg.PrometheusPort); err != nil {
				serviceErrors <- fmt.Errorf("metrics server error: %w", err)
			}
		}()
	}

	// Health check routine
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := performHealthCheck(server, p2pClient, relayDispatcher); err != nil {
					logger.Warn("Health check failed", zap.Error(err))
				}
			}
		}
	}()

	// Main event loop
	logger.Info("Bitcoin Sprint is running and ready to serve requests")

	select {
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	case err := <-serviceErrors:
		logger.Error("Service error, initiating shutdown", zap.Error(err))
	}

	// Graceful shutdown
	logger.Info("Initiating graceful shutdown")
	cancel()

	// Wait for services to shut down
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop services in reverse order
	p2pClient.Stop()

	if err := relayDispatcher.Shutdown(ctx); err != nil {
		logger.Error("Error stopping relay dispatcher", zap.Error(err))
	}

	// Broadcaster doesn't need explicit stopping
	logger.Info("Broadcaster stopped")

	// Runtime optimizations are persistent
	logger.Info("Runtime optimizations remain active")

	// Close channels
	close(blockChan)

	logger.Info("Bitcoin Sprint shutdown complete")
}

// initLogger initializes structured logging with appropriate level
func initLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()

	// Set log level based on environment
	logLevel := os.Getenv("LOG_LEVEL")
	switch logLevel {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info":
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	// Enable caller info for development
	if logLevel == "debug" {
		config.Development = true
		config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	}

	return config.Build()
}

// startMetricsServer starts a Prometheus metrics server
func startMetricsServer(ctx context.Context, port int) error {
	// This would integrate with a metrics collection system
	// For now, just log that it would start
	log.Printf("Metrics server would start on port %d", port)
	return nil
}

// performHealthCheck performs a comprehensive health check of all services
func performHealthCheck(server *api.Server, p2pClient *p2p.Client, relayDispatcher *relay.RelayDispatcher) error {
	// Check API server health
	if server == nil {
		return fmt.Errorf("API server is nil")
	}

	// Check P2P client health (simplified check)
	if p2pClient == nil {
		return fmt.Errorf("P2P client is nil")
	}

	// Check relay dispatcher health
	if relayDispatcher == nil {
		return fmt.Errorf("relay dispatcher is nil")
	}

	healthStatus := relayDispatcher.GetHealthStatus()
	for network, status := range healthStatus {
		if !status.IsHealthy {
			return fmt.Errorf("relay client %s is unhealthy: %s", network, status.Status)
		}
	}

	return nil
}
