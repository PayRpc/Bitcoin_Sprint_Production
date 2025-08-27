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
	"github.com/PayRpc/Bitcoin-Sprint/internal/cache"
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
	// Load environment
	_ = godotenv.Load()

	// Config + logger
	cfg := config.Load()
	logger := initLogger(cfg)
	defer logger.Sync()

	// Root context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Performance tuning first
	perf := performance.New(cfg, logger)
	if err := perf.ApplyOptimizations(); err != nil {
		logger.Fatal("Failed to apply performance optimizations", zap.Error(err))
	}

	// Startup log
	logger.Info("Bitcoin Sprint starting...",
		zap.String("version", getVersion()),
		zap.String("go_version", goruntime.Version()),
		zap.Int("num_cpu", goruntime.NumCPU()),
		zap.Int("gomaxprocs", goruntime.GOMAXPROCS(0)),
		zap.String("tier", string(cfg.Tier)),
		zap.Bool("turbo_mode", cfg.Tier == config.TierTurbo || cfg.Tier == config.TierEnterprise),
		zap.Any("perf_stats", perf.GetCurrentStats()),
	)

	// License validation
	if !license.Validate(cfg.LicenseKey) {
		logger.Fatal("Invalid license key")
	}

	// Core modules
	mempoolModule := mempool.New()
	blockChan := make(chan blocks.BlockEvent, 1000)

	// ZMQ client
	zmqClient := zmq.New(cfg, blockChan, mempoolModule, logger)
	go zmqClient.Run(ctx)

	// P2P client
	p2pModule, err := p2p.New(cfg, blockChan, mempoolModule, logger)
	if err != nil {
		logger.Fatal("Failed to create P2P module", zap.Error(err))
	}
	go p2pModule.Run(ctx)

	// Cache layer
	blockCache := cache.New(2000, logger)
	prefetchWorker := cache.NewPrefetchWorker(blockCache, blockChan, logger)
	prefetchWorker.Start(ctx)

	// API server
	apiServer := api.NewWithCache(cfg, blockChan, mempoolModule, blockCache, logger)
	go func() {
		logger.Info("Starting API server",
			zap.String("addr", fmt.Sprintf("%s:%d", cfg.APIHost, cfg.APIPort)))
		apiServer.Run(ctx)
	}()

	// Tier-aware relay loop
	go func() {
		if cfg.Tier == config.TierTurbo || cfg.Tier == config.TierEnterprise {
			runMemoryOptimizedRelay(ctx, blockChan, cfg, logger, p2pModule)
		} else {
			runStandardRelay(ctx, blockChan, cfg, logger, p2pModule)
		}
	}()

	// Shutdown signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	sig := <-sigs
	logger.Info("Shutdown signal received", zap.String("signal", sig.String()))

	// Cancel context for all workers
	cancel()

	// Stop services gracefully
	zmqClient.Stop()
	p2pModule.Stop()
	apiServer.Stop()
	prefetchWorker.Stop()

	close(blockChan)

	logger.Info("Bitcoin Sprint stopped cleanly")
}

// initLogger initializes structured logging
func initLogger(cfg config.Config) *zap.Logger {
	var (
		logger *zap.Logger
		err    error
	)
	if cfg.OptimizeSystem {
		config := zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
		logger, err = config.Build()
	} else {
		config := zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		logger, err = config.Build()
	}
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	return logger
}

func getVersion() string {
	return "2.3.0-production"
}

// runMemoryOptimizedRelay: Enterprise/Turbo relay loop
func runMemoryOptimizedRelay(ctx context.Context, ch <-chan blocks.BlockEvent,
	cfg config.Config, logger *zap.Logger, p2pModule *p2p.Module) {

	logger.Info("Turbo/Enterprise relay loop started",
		zap.Duration("deadline", cfg.WriteDeadline),
		zap.String("tier", string(cfg.Tier)),
	)

	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			start := time.Now()
			err := notifyPeersZeroAlloc(evt.Hash, cfg, logger, p2pModule)
			elapsed := time.Since(start)

			if err != nil {
				logger.Error("Turbo relay failed",
					zap.Error(err),
					zap.Duration("elapsed", elapsed),
					zap.String("blockHash", evt.Hash))
			} else if cfg.Tier != config.TierEnterprise && elapsed > cfg.WriteDeadline {
				logger.Warn("Relay exceeded deadline",
					zap.Duration("elapsed", elapsed),
					zap.Duration("deadline", cfg.WriteDeadline))
			} else {
				logger.Debug("Relay ok", zap.Duration("elapsed", elapsed))
			}
		}
	}
}

// runStandardRelay: Free/Pro/Business tiers
func runStandardRelay(ctx context.Context, ch <-chan blocks.BlockEvent,
	cfg config.Config, logger *zap.Logger, p2pModule *p2p.Module) {

	logger.Info("Standard relay loop started", zap.String("tier", string(cfg.Tier)))

	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			start := time.Now()
			err := p2pModule.BroadcastBlockHash(evt.Hash)
			elapsed := time.Since(start)

			if err != nil {
				logger.Error("Standard relay failed",
					zap.Error(err), zap.String("blockHash", evt.Hash))
			} else {
				logger.Debug("Standard relay ok",
					zap.Duration("elapsed", elapsed),
					zap.String("blockHash", evt.Hash))
			}
		}
	}
}

// notifyPeersZeroAlloc: Turbo/Enterprise relay with deadlines + zero-copy
func notifyPeersZeroAlloc(blockHash string, cfg config.Config,
	logger *zap.Logger, p2pModule *p2p.Module) error {

	if cfg.Tier == config.TierEnterprise {
		// Enterprise ignores strict deadlines
		return p2pModule.BroadcastBlockHash(blockHash)
	}

	// Turbo = strict deadline enforced
	ctx, cancel := context.WithTimeout(context.Background(), cfg.WriteDeadline)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- p2pModule.BroadcastBlockHash(blockHash) }()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("turbo relay timeout %v", cfg.WriteDeadline)
	}
}
