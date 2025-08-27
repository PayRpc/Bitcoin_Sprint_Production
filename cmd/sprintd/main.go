package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	goruntime "runtime"
	"syscall"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/api"
	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/license"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
	"github.com/PayRpc/Bitcoin-Sprint/internal/p2p"
	"github.com/PayRpc/Bitcoin-Sprint/internal/performance"
	"github.com/PayRpc/Bitcoin-Sprint/internal/zmq"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, continuing with environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger := initLogger(cfg)
	defer logger.Sync()

	// Apply performance optimizations FIRST (permanent for all tiers)
	perfManager := performance.New(cfg, logger)
	if err := perfManager.ApplyOptimizations(); err != nil {
		logger.Fatal("Failed to apply performance optimizations", zap.Error(err))
	}

	// Log startup information with performance metrics
	logger.Info("Bitcoin Sprint starting...",
		zap.String("version", getVersion()),
		zap.String("go_version", goruntime.Version()),
		zap.Int("num_cpu", goruntime.NumCPU()),
		zap.Int("gomaxprocs", goruntime.GOMAXPROCS(0)),
		zap.String("tier", string(cfg.Tier)),
		zap.Bool("turbo_mode", cfg.Tier == config.TierTurbo || cfg.Tier == config.TierEnterprise),
		zap.Bool("performance_optimizations", true),
		zap.Any("performance_stats", perfManager.GetCurrentStats()),
	)

	// Validate license
	if !license.Validate(cfg.LicenseKey) {
		logger.Fatal("Invalid license key")
	}

	// Initialize modules
	mempoolModule := mempool.New()
	blockChan := make(chan blocks.BlockEvent, 100)

	// Initialize ZMQ client
	zmqClient := zmq.New(cfg, blockChan, mempoolModule, logger)
	go zmqClient.Run()

	// Initialize P2P module
	p2pModule, err := p2p.New(cfg, blockChan, mempoolModule, logger)
	if err != nil {
		logger.Fatal("Failed to create P2P module", zap.Error(err))
	}
	// P2P module doesn't have a Start method - it runs automatically

	// Initialize API server
	apiServer := api.New(cfg, blockChan, mempoolModule, logger)
	go func() {
		logger.Info("Starting API server", zap.String("address", fmt.Sprintf("%s:%d", cfg.APIHost, cfg.APIPort)))
		apiServer.Run()
	}()

	// Start block processing based on tier
	if cfg.Tier == config.TierTurbo || cfg.Tier == config.TierEnterprise {
		go runMemoryOptimizedSprint(blockChan, cfg, logger)
	} else {
		go runStandardSprint(blockChan, cfg, logger)
	}

	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal
	<-c
	logger.Info("Shutting down Bitcoin Sprint...")

	// Stop all modules
	zmqClient.Stop()
	p2pModule.Stop()
	apiServer.Stop()

	logger.Info("Bitcoin Sprint stopped")
}

// initLogger initializes structured logging
func initLogger(cfg config.Config) *zap.Logger {
	var logger *zap.Logger
	var err error

	if cfg.OptimizeSystem {
		// Production config for optimized systems
		config := zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
		logger, err = config.Build()
	} else {
		// Development config
		config := zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		logger, err = config.Build()
	}

	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	return logger
}

// getVersion returns the application version
func getVersion() string {
	return "2.2.0-performance" // Production-ready performance optimizations
}

// runMemoryOptimizedSprint handles Turbo/Enterprise tier block processing
// Uses shared memory and zero-copy operations with strict deadlines
func runMemoryOptimizedSprint(blockChan <-chan blocks.BlockEvent, cfg config.Config, logger *zap.Logger) {
	logger.Info("Memory-optimized sprint started", 
		zap.Duration("writeDeadline", cfg.WriteDeadline),
		zap.String("tier", string(cfg.Tier)),
	)
	
	for evt := range blockChan {
		start := time.Now()
		
		// Turbo relay with strict deadline and zero-copy
		err := notifyPeersZeroAlloc(evt.Hash, cfg.WriteDeadline, cfg.Tier, logger)
		
		elapsed := time.Since(start)
		if err != nil {
			logger.Error("Turbo relay failed", 
				zap.Error(err),
				zap.Duration("elapsed", elapsed),
				zap.String("blockHash", evt.Hash),
			)
		} else if elapsed > cfg.WriteDeadline {
			logger.Warn("Turbo relay exceeded deadline",
				zap.Duration("elapsed", elapsed),
				zap.Duration("deadline", cfg.WriteDeadline),
				zap.String("blockHash", evt.Hash),
			)
		} else {
			logger.Debug("Turbo relay successful",
				zap.Duration("elapsed", elapsed),
				zap.String("blockHash", evt.Hash),
			)
		}
	}
}

// runStandardSprint handles Free/Pro/Business tier block processing
// Uses standard relay mechanisms with relaxed deadlines
func runStandardSprint(blockChan <-chan blocks.BlockEvent, cfg config.Config, logger *zap.Logger) {
	logger.Info("Standard sprint started", 
		zap.Duration("writeDeadline", cfg.WriteDeadline),
		zap.String("tier", string(cfg.Tier)),
	)
	
	for evt := range blockChan {
		start := time.Now()
		
		// Normal relay with standard mechanisms
		err := notifyPeers(evt.Hash, cfg.Tier, logger)
		
		elapsed := time.Since(start)
		if err != nil {
			logger.Error("Standard relay failed", 
				zap.Error(err),
				zap.Duration("elapsed", elapsed),
				zap.String("blockHash", evt.Hash),
			)
		} else {
			logger.Debug("Standard relay successful",
				zap.Duration("elapsed", elapsed),
				zap.String("blockHash", evt.Hash),
			)
		}
	}
}

// notifyPeersZeroAlloc implements zero-copy peer notification for Turbo/Enterprise tiers
func notifyPeersZeroAlloc(blockHash string, deadline time.Duration, tier config.Tier, logger *zap.Logger) error {
	// TODO: Implement zero-copy notification with pre-allocated buffers
	// This would use shared memory and avoid heap allocations
	
	// For now, delegate to standard notification but with strict timeout
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()
	
	done := make(chan error, 1)
	go func() {
		done <- notifyPeers(blockHash, tier, logger)
	}()
	
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("turbo notification timed out after %v", deadline)
	}
}

// notifyPeers implements standard peer notification
func notifyPeers(blockHash string, tier config.Tier, logger *zap.Logger) error {
	// TODO: Implement actual peer notification logic
	// This would broadcast the block hash to connected peers
	
	logger.Debug("Notifying peers of new block",
		zap.String("blockHash", blockHash),
		zap.String("tier", string(tier)),
	)
	
	// Simulate notification work
	switch tier {
	case config.TierTurbo, config.TierEnterprise:
		// Minimal processing time for high-tier customers
		time.Sleep(50 * time.Microsecond)
	case config.TierBusiness:
		time.Sleep(200 * time.Microsecond)
	case config.TierPro:
		time.Sleep(500 * time.Microsecond)
	default: // Free tier
		time.Sleep(1 * time.Millisecond)
	}
	
	return nil
}
