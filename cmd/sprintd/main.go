package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/PayRpc/Bitcoin-Sprint/internal/api"
	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/license"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
	"github.com/PayRpc/Bitcoin-Sprint/internal/p2p"
	"github.com/PayRpc/Bitcoin-Sprint/internal/runtime"
	"github.com/PayRpc/Bitcoin-Sprint/internal/zmq"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// 1. Load config
	cfg := config.Load()

	// 2. Validate license
	if !license.Validate(cfg.LicenseKey) {
		logger.Fatal("invalid license key")
	}

	// 3. Init subsystems
	mem := mempool.New()
	blockChan := make(chan blocks.BlockEvent, 1024)

	p2pClient, err := p2p.New(cfg, blockChan, mem, logger)
	if err != nil {
		logger.Fatal("failed to create P2P client", zap.Error(err))
	}
	zmqClient := zmq.New(cfg, blockChan, mem, logger)
	apiSrv := api.New(cfg, blockChan, mem, logger)

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
