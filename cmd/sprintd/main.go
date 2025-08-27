package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/api"
	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/license"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
	"github.com/PayRpc/Bitcoin-Sprint/internal/p2p"
	"github.com/PayRpc/Bitcoin-Sprint/internal/runtime"
	"github.com/PayRpc/Bitcoin-Sprint/internal/zmq"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: could not load .env file: %v", err)
	}

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// 1. Load config
	cfg := config.Load()

	logger.Info("Starting Bitcoin Sprint daemon",
		zap.String("version", "2.1.0"),
		zap.String("mode", os.Getenv("APP_MODE")),
		zap.String("tier", string(cfg.Tier)),
	)

	// Log turbo mode activation
	if cfg.Tier == config.TierTurbo || cfg.Tier == config.TierEnterprise {
		logger.Info("⚡ Turbo mode enabled",
			zap.Duration("writeDeadline", cfg.WriteDeadline),
			zap.Bool("sharedMemory", cfg.UseSharedMemory),
			zap.Int("blockBufferSize", cfg.BlockBufferSize),
		)
	}

	// 2. Validate license (temporarily disabled for testing)
	skipLicense := os.Getenv("SKIP_LICENSE_VALIDATION") == "true"
	if !skipLicense && !license.Validate(cfg.LicenseKey) {
		logger.Fatal("invalid license key")
	} else if skipLicense {
		logger.Info("License validation skipped (development mode)")
	}

	// 3. Init subsystems
	mem := mempool.New()
	blockChan := make(chan blocks.BlockEvent, cfg.BlockBufferSize)

	p2pClient, err := p2p.New(cfg, blockChan, mem, logger)
	if err != nil {
		logger.Fatal("failed to create P2P client", zap.Error(err))
	}
	zmqClient := zmq.New(cfg, blockChan, mem, logger)
	apiSrv := api.New(cfg, blockChan, mem, logger)

	// Use shared memory fast-path if Turbo or Enterprise
	if cfg.UseSharedMemory {
		logger.Info("⚡ Starting memory-optimized sprint with tight deadlines")
		go runMemoryOptimizedSprint(blockChan, cfg, logger)
	} else {
		logger.Info("Starting standard sprint processing")
		go runStandardSprint(blockChan, cfg, logger)
	}

	// Optional: Direct P2P bypass + shared memory
	if cfg.UseDirectP2P {
		ctx := context.Background()
		// Connect to first node (in production, you'd iterate through cfg.BitcoinNodes)
		if len(cfg.BitcoinNodes) > 0 {
			err := p2p.NewDirect(ctx, cfg.BitcoinNodes[0], blockChan, logger)
			if err != nil {
				logger.Warn("Failed to start direct P2P", zap.Error(err))
			}
		}
		if cfg.UseMemoryChannel {
			p2p.NewMemoryWatcher(ctx, blockChan, logger)
		}
	}

	if cfg.OptimizeSystem {
		runtime.ApplySystemOptimizations(logger)
	}

	// 4. Start services
	go p2pClient.Run()
	go zmqClient.Run()
	go apiSrv.Run()

	// 5. Wait for shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	// 6. Graceful shutdown
	apiSrv.Stop()
	p2pClient.Stop()
	zmqClient.Stop()
	logger.Info("shutdown complete")
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
