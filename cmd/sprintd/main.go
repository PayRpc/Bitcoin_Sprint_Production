package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/PayRpc/Bitcoin-Sprint/internal/api"
	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/broadcaster"
	"github.com/PayRpc/Bitcoin-Sprint/internal/cache"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/database"
	"github.com/PayRpc/Bitcoin-Sprint/internal/license"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
	"github.com/PayRpc/Bitcoin-Sprint/internal/p2p"
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
			logger.Fatal("License validation failed")
		}
		logger.Info("License validated successfully")
	}

	// Initialize database if configured
	if cfg.DatabaseURL != "" {
		dbCfg := database.Config{
			Type: cfg.DatabaseType,
			URL:  cfg.DatabaseURL,
		}
		db, err := database.New(dbCfg, logger)
		if err != nil {
			logger.Error("Failed to connect to database", zap.Error(err))
		} else {
			defer db.Close()
			logger.Info("Database connection established")
		}
	}

	// Initialize core components
	blockChan := make(chan blocks.BlockEvent, 1000)
	mem := mempool.New()

	// Initialize cache
	_ = cache.New(1000, logger) // Cache not currently used

	// Initialize broadcaster for real-time updates
	_ = broadcaster.New(logger) // Broadcaster not currently used

	// Initialize relay dispatcher for multi-chain support
	relayDispatcher := relay.NewRelayDispatcher(cfg, logger)

	// Initialize P2P client for Bitcoin network
	p2pClient, err := p2p.New(cfg, blockChan, mem, logger)
	if err != nil {
		logger.Fatal("Failed to create P2P client", zap.Error(err))
	}

	// Initialize API server (not started in this version)
	_ = api.New(cfg, blockChan, mem, logger)

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start P2P client
	go func() {
		p2pClient.Run()
	}()

	logger.Info("Bitcoin Sprint started successfully",
		zap.String("api_host", cfg.APIHost),
		zap.Int("api_port", cfg.APIPort),
		zap.Int("p2p_port", cfg.PeerListenPort))

	// Wait for shutdown signal
	<-sigChan
	logger.Info("Shutdown signal received")

	// Graceful shutdown
	cancel()

	// Stop services in reverse order
	p2pClient.Stop()

	if err := relayDispatcher.Shutdown(ctx); err != nil {
		logger.Error("Error stopping relay dispatcher", zap.Error(err))
	}

	// Check health status before shutdown
	healthStatus := relayDispatcher.GetHealthStatus()
	for network, status := range healthStatus {
		if !status.IsHealthy {
			logger.Warn("Relay client unhealthy during shutdown",
				zap.String("network", network),
				zap.String("error", status.ErrorMessage))
		}
	}

	logger.Info("Bitcoin Sprint shutdown complete")
}

func initLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return config.Build()
}
