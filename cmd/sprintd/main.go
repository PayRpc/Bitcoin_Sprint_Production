package main

import (
	"context"
	"fmt"
	"log"
	"net"
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
	"github.com/PayRpc/Bitcoin-Sprint/internal/dedup"
	"github.com/PayRpc/Bitcoin-Sprint/internal/license"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
	"github.com/PayRpc/Bitcoin-Sprint/internal/messaging"
	"github.com/PayRpc/Bitcoin-Sprint/internal/p2p"
	"github.com/PayRpc/Bitcoin-Sprint/internal/relay"
	gctuning "github.com/PayRpc/Bitcoin-Sprint/internal/runtime"
	"github.com/PayRpc/Bitcoin-Sprint/internal/throttle"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Global block deduplication index (singleton)
var BlockIdx = dedup.NewBlockIndex(2 * time.Minute)

func main() {
	// Initialize structured logging
	logger, err := initLogger()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("=== MAIN FUNCTION STARTED ===")

	// Initialize GC tuning for optimal performance
	logger.Info("=== INITIALIZING RUNTIME OPTIMIZATIONS ===")
	if err := gctuning.InitializeGCTuning(logger); err != nil {
		logger.Warn("Failed to initialize GC tuning", zap.Error(err))
	}

	// Start GC monitoring
	gctuning.MonitorGCPerformance(logger, 5*time.Minute)

	logger.Info("Starting Bitcoin Sprint",
		zap.String("version", "1.0.0"),
		zap.String("go_version", runtime.Version()),
		zap.Int("cpu_cores", runtime.NumCPU()))

	logger.Info("=== LOADING CONFIGURATION ===")
	// Load configuration
	cfg := config.Load()
	logger.Info("Configuration loaded",
		zap.String("tier", string(cfg.Tier)),
		zap.String("default_chain", cfg.DefaultChain),
		zap.Bool("optimize_system", cfg.OptimizeSystem))

	logger.Info("=== VALIDATING LICENSE ===")
	// Validate license if provided
	if cfg.LicenseKey != "" {
		if !license.Validate(cfg.LicenseKey) {
			logger.Fatal("License validation failed")
		}
		logger.Info("License validated successfully")
	}

	logger.Info("=== INITIALIZING DATABASE ===")
	// Initialize database if configured
	if cfg.DatabaseURL != "" {
		dbCfg := database.Config{
			Type: cfg.DatabaseType,
			URL:  cfg.DatabaseURL,
		}
		db, err := database.New(dbCfg, logger)
		if err != nil {
			logger.Warn("Database connection failed, continuing without database",
				zap.Error(err),
				zap.String("type", cfg.DatabaseType),
				zap.String("url", cfg.DatabaseURL))
			logger.Info("Application will run without database persistence")
		} else {
			defer db.Close()
			logger.Info("Database connection established")
		}
	} else {
		logger.Info("No database configured, running without persistence")
	}

	logger.Info("=== INITIALIZING CORE COMPONENTS ===")
	// Initialize core components
	blockChan := make(chan blocks.BlockEvent, 1000)
	mem := mempool.New()

	// Enhanced RPC service disabled for minimal build

	logger.Info("=== INITIALIZING CACHE ===")
	// Initialize cache
	_ = cache.New(1000, logger) // Cache not currently used

	logger.Info("=== INITIALIZING BROADCASTER ===")
	// Initialize broadcaster for real-time updates with optimizations
	_ = broadcaster.New(logger) // Broadcaster with fan-out batching and pre-encoded frames

	logger.Info("=== INITIALIZING ENDPOINT THROTTLE ===")
	// Initialize endpoint throttling for ETH/SOL connections
	endpointThrottle := throttle.New(logger)
	// Register common endpoints (example - these would come from config)
	endpointThrottle.RegisterEndpoint("https://eth-mainnet.g.alchemy.com/v2/demo")
	endpointThrottle.RegisterEndpoint("https://api.mainnet-beta.solana.com")
	endpointThrottle.RegisterEndpoint("https://rpc.ankr.com/eth")
	endpointThrottle.RegisterEndpoint("https://rpc.ankr.com/solana")
	logger.Info("Endpoint throttling initialized with health monitoring")

	logger.Info("=== INITIALIZING RELAY DISPATCHER ===")
	// Initialize relay dispatcher for multi-chain support
	relayDispatcher := relay.NewRelayDispatcher(cfg, logger)

	logger.Info("=== INITIALIZING P2P CLIENT ===")
	// Initialize P2P client for Bitcoin network
	p2pClient, err := p2p.New(cfg, blockChan, mem, logger)
	if err != nil {
		logger.Fatal("Failed to create P2P client", zap.Error(err))
	}

	logger.Info("=== SETTING UP CONTEXT ===")
	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Info("=== CREATING API SERVER ===")
	// Initialize API server and start it
	logger.Info("Creating API server instance")
	apiServer := api.New(cfg, blockChan, mem, logger)
	if apiServer == nil {
		logger.Fatal("Failed to create API server - returned nil")
	}
	logger.Info("API server instance created successfully")

	logger.Info("=== STARTING API SERVER GOROUTINE ===")
	// Check if port is available before starting
	addr := fmt.Sprintf("%s:%d", cfg.APIHost, cfg.APIPort)
	if !isPortAvailable(addr) {
		logger.Fatal("Port already in use", zap.String("addr", addr), zap.String("solution", "Kill the process using this port or change API_PORT in .env"))
	}
	logger.Info("Port is available", zap.String("addr", addr))

	// Start API server in a goroutine so main can continue
	logger.Info("Starting API server directly")
	go func() {
		logger.Info("API server launch initiated")
		apiServer.Run(ctx)
	}()

	logger.Info("=== INITIALIZING BACKFILL SERVICE ===")
	// Initialize backfill service for historical data processing
	backfillService := messaging.NewBackfillService(cfg, blockChan, mem, logger)
	if err := backfillService.Start(ctx); err != nil {
		logger.Error("Failed to start backfill service", zap.Error(err))
	} else {
		logger.Info("Backfill service started")
	}

	// Enhanced RPC temporarily disabled to prioritize HTTP startup

	logger.Info("=== SETTING UP SIGNAL HANDLING ===")
	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("=== STARTING P2P CLIENT ===")
	// Start P2P client
	go func() {
		p2pClient.Run()
	}()

	logger.Info("=== BITCOIN SPRINT STARTUP COMPLETE ===")
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

	// Stop backfill service
	backfillService.Stop()

	// Enhanced RPC service not started in minimal build

	// Close block deduplication index
	BlockIdx.Close()

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

func isPortAvailable(addr string) bool {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

func initLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return config.Build()
}
