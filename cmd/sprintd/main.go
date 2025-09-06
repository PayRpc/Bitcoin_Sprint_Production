package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/PayRpc/Bitcoin-Sprint/internal/api"
	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks/bitcoin"
	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks/ethereum"
	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks/solana"
	"github.com/PayRpc/Bitcoin-Sprint/internal/broadcaster"
	"github.com/PayRpc/Bitcoin-Sprint/internal/cache"
	"github.com/PayRpc/Bitcoin-Sprint/internal/circuitbreaker"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/database"
	"github.com/PayRpc/Bitcoin-Sprint/internal/dedup"
	"github.com/PayRpc/Bitcoin-Sprint/internal/license"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
	"github.com/PayRpc/Bitcoin-Sprint/internal/messaging"
	"github.com/PayRpc/Bitcoin-Sprint/internal/metrics"
	"github.com/PayRpc/Bitcoin-Sprint/internal/middleware"
	"github.com/PayRpc/Bitcoin-Sprint/internal/migrations"
	"github.com/PayRpc/Bitcoin-Sprint/internal/p2p"
	"github.com/PayRpc/Bitcoin-Sprint/internal/ratelimit"
	"github.com/PayRpc/Bitcoin-Sprint/internal/relay"
	gctuning "github.com/PayRpc/Bitcoin-Sprint/internal/runtime"
	runtimeopt "github.com/PayRpc/Bitcoin-Sprint/internal/runtime"
	"github.com/PayRpc/Bitcoin-Sprint/internal/throttle"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"
)

// The content below implements a ServiceManager-based application entrypoint.
// It is intentionally verbose and defensive to improve runtime reliability,
// observability and graceful shutdown handling.

const (
	AppName        = "Bitcoin Sprint"
	AppVersion     = "2.0.0"
	ShutdownGrace  = 30 * time.Second
	StartupTimeout = 2 * time.Minute
	// Security constants
	MaxAPIRateRequests = 100
	MaxAPIRatePeriod   = 1 * time.Minute
	MaxWSConns         = 1000
	MaxHeaderBytes     = 1 << 20 // 1MB
)

// ServiceManager orchestrates application lifecycle with proper dependency injection
type ServiceManager struct {
	cfg    *config.Config
	logger *zap.Logger

	// Core infrastructure
	db              *database.DB
	metricsRegistry *prometheus.Registry
	blockIdx        *dedup.BlockIndex
	circuitBreakers map[string]*circuitbreaker.Manager

	// Communication channels
	blockChan    chan blocks.BlockEvent
	shutdownChan chan struct{}

	// Core services
	mempool         *mempool.Mempool
	cache           *cache.Cache
	blockProcessor  *blocks.BlockProcessor
	broadcaster     *broadcaster.Broadcaster
	throttleManager *throttle.EndpointThrottle
	relayDispatcher *relay.RelayDispatcher
	p2pClient       *p2p.Client
	apiServer       *api.Server
	backfillService *messaging.BackfillService
	rateLimiter     *ratelimit.RateLimiter

	// Runtime optimization
	runtimeOptimizer *runtimeopt.SystemOptimizer

	// Lifecycle management
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	startupErrors chan error
	healthServer  *http.Server
	startTime     time.Time
	// Security
	licenseKey  string
	licenseInfo *license.LicenseInfo
}

func main() {
	logger := mustInitLogger()
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
		}
	}()

	// Set up panic recovery
	defer func() {
		if r := recover(); r != nil {
			logger.Fatal("Application panic",
				zap.Any("panic", r),
				zap.String("stack", string(getStacktrace())))
		}
	}()

	logger.Info("Starting Bitcoin Sprint",
		zap.String("version", AppVersion),
		zap.String("go_version", runtime.Version()),
		zap.Int("cpu_cores", runtime.NumCPU()),
		zap.Int("gomaxprocs", runtime.GOMAXPROCS(0)))

	// Create service manager with proper error handling
	sm, err := NewServiceManager(logger)
	if err != nil {
		logger.Fatal("Failed to create service manager", zap.Error(err))
	}

	// Set up signal handling before starting services
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Start all services with proper error propagation
	if err := sm.Start(); err != nil {
		logger.Fatal("Failed to start services", zap.Error(err))
	}

	logger.Info("Bitcoin Sprint startup complete",
		zap.String("api_address", fmt.Sprintf("%s:%d", sm.cfg.APIHost, sm.cfg.APIPort)),
		zap.String("health_address", fmt.Sprintf("%s:%d", sm.cfg.APIHost, sm.cfg.APIPort+1)),
		zap.String("tier", string(sm.cfg.Tier)))

	// Wait for shutdown signal with startup error monitoring
	select {
	case sig := <-sigChan:
		logger.Info("Shutdown signal received", zap.String("signal", sig.String()))
	case err := <-sm.startupErrors:
		logger.Fatal("Critical startup error", zap.Error(err))
	}

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), ShutdownGrace)
	defer shutdownCancel()

	if err := sm.Shutdown(shutdownCtx); err != nil {
		logger.Error("Shutdown completed with errors", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Bitcoin Sprint shutdown complete")
}

// NewServiceManager creates a new service manager with dependency injection
func NewServiceManager(logger *zap.Logger) (*ServiceManager, error) {
	cfg := config.Load()
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Generate node ID if not provided
	if cfg.NodeID == "" {
		hostname, _ := os.Hostname()
		hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", hostname, time.Now().UnixNano())))
		cfg.NodeID = hex.EncodeToString(hash[:])[:16]
	}

	ctx, cancel := context.WithCancel(context.Background())

	sm := &ServiceManager{
		cfg:             &cfg,
		logger:          logger,
		ctx:             ctx,
		cancel:          cancel,
		blockChan:       make(chan blocks.BlockEvent, cfg.BlockChannelBuffer),
		shutdownChan:    make(chan struct{}),
		startupErrors:   make(chan error, 20), // Increased buffer for startup errors
		blockIdx:        dedup.NewBlockIndex(cfg.BlockDeduplicationWindow),
		startTime:       time.Now(),
		circuitBreakers: make(map[string]*circuitbreaker.Manager),
		licenseInfo:     &license.LicenseInfo{},
	}

	return sm, nil
}

// Start initializes and starts all services with proper error handling and health checks
func (sm *ServiceManager) Start() error {
	startCtx, startCancel := context.WithTimeout(sm.ctx, StartupTimeout)
	defer startCancel()

	sm.logger.Info("Initializing runtime optimizations")
	if err := sm.initializeRuntime(); err != nil {
		return fmt.Errorf("runtime initialization failed: %w", err)
	}

	sm.logger.Info("Validating license")
	if err := sm.validateLicense(); err != nil {
		return fmt.Errorf("license validation failed: %w", err)
	}

	sm.logger.Info("Initializing metrics registry")
	if err := sm.initializeMetrics(); err != nil {
		return fmt.Errorf("failed to initialize metrics: %w", err)
	}

	sm.logger.Info("Initializing circuit breakers")
	sm.initializeCircuitBreakers()

	sm.logger.Info("Connecting to database")
	if err := sm.initializeDatabase(startCtx); err != nil {
		if sm.cfg.RequireDatabase {
			return fmt.Errorf("database initialization failed: %w", err)
		}
		sm.logger.Warn("Database initialization failed, continuing without persistence", zap.Error(err))
	}

	sm.logger.Info("Initializing core services")
	if err := sm.initializeCoreServices(); err != nil {
		return fmt.Errorf("failed to initialize core services: %w", err)
	}

	sm.logger.Info("Starting network services")
	if err := sm.startNetworkServices(startCtx); err != nil {
		return fmt.Errorf("failed to start network services: %w", err)
	}

	sm.logger.Info("Starting background services")
	if err := sm.startBackgroundServices(startCtx); err != nil {
		return fmt.Errorf("failed to start background services: %w", err)
	}

	sm.logger.Info("Starting health monitoring")
	if err := sm.startHealthServer(); err != nil {
		return fmt.Errorf("failed to start health server: %w", err)
	}

	// Start continuous health monitoring
	sm.wg.Add(1)
	go sm.monitorHealth()

	// Start circuit breaker monitoring
	sm.wg.Add(1)
	go sm.monitorCircuitBreakers()

	return nil
}

// Shutdown gracefully stops all services in reverse dependency order
func (sm *ServiceManager) Shutdown(ctx context.Context) error {
	sm.logger.Info("Starting graceful shutdown")
	close(sm.shutdownChan)

	var shutdownErrors []error

	// Stop accepting new requests
	if sm.healthServer != nil {
		if err := sm.healthServer.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("health server shutdown: %w", err))
		}
	}

	if sm.apiServer != nil {
		sm.apiServer.Stop()
	}

	// Stop background services
	if sm.backfillService != nil {
		sm.backfillService.Stop()
	}

	// Stop P2P client
	if sm.p2pClient != nil {
		sm.p2pClient.Stop()
	}

	// Stop relay dispatcher
	if sm.relayDispatcher != nil {
		if err := sm.relayDispatcher.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("relay dispatcher shutdown: %w", err))
		}
	}

	// Stop block processor
	if sm.blockProcessor != nil {
		if err := sm.blockProcessor.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("block processor shutdown: %w", err))
		}
	}

	// Cancel context and wait for goroutines
	sm.cancel()

	// Wait for background goroutines with timeout
	done := make(chan struct{})
	go func() {
		sm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		sm.logger.Info("All goroutines stopped successfully")
	case <-ctx.Done():
		sm.logger.Warn("Shutdown timeout exceeded, some goroutines may still be running")
		shutdownErrors = append(shutdownErrors, fmt.Errorf("shutdown timeout"))
	}

	// Close infrastructure components
	if sm.blockIdx != nil {
		sm.blockIdx.Close()
	}

	if sm.db != nil {
		sm.db.Close()
	}

	// Restore runtime optimization settings
	if sm.runtimeOptimizer != nil {
		sm.logger.Info("Restoring runtime optimization settings")
		if err := sm.runtimeOptimizer.Restore(); err != nil {
			sm.logger.Warn("Failed to restore runtime settings", zap.Error(err))
			shutdownErrors = append(shutdownErrors, fmt.Errorf("runtime optimizer restore: %w", err))
		} else {
			sm.logger.Info("Runtime settings restored successfully")
		}
	}

	if len(shutdownErrors) > 0 {
		return fmt.Errorf("shutdown completed with %d errors: %v", len(shutdownErrors), shutdownErrors)
	}

	return nil
}

// initializeRuntime sets up GC tuning and performance monitoring
func (sm *ServiceManager) initializeRuntime() error {
	// Initialize legacy GC tuning for backward compatibility
	if err := gctuning.InitializeGCTuning(sm.logger); err != nil {
		return fmt.Errorf("GC tuning initialization failed: %w", err)
	}

	// Initialize new runtime optimization system
	sm.logger.Info("Initializing advanced runtime optimization system")
	
	// Determine optimization level based on configuration
	var optConfig *runtimeopt.SystemOptimizationConfig
	if sm.cfg.Tier == config.TierEnterprise {
		optConfig = runtimeopt.EnterpriseConfig()
		sm.logger.Info("Using Enterprise optimization level")
	} else if sm.cfg.Tier == config.TierTurbo {
		optConfig = runtimeopt.TurboConfig()
		sm.logger.Info("Using Turbo optimization level")
	} else if sm.cfg.Tier == config.TierBusiness {
		optConfig = runtimeopt.EnterpriseConfig() // Use Enterprise for Business tier
		sm.logger.Info("Using Enterprise optimization level for Business tier")
	} else if sm.cfg.OptimizeSystem {
		optConfig = runtimeopt.DefaultConfig()
		sm.logger.Info("Using Default optimization level")
	} else {
		optConfig = runtimeopt.DefaultConfig() // Use Default for Basic tier
		sm.logger.Info("Using Default optimization level for Basic tier")
	}
	
	// Apply runtime optimizations
	sm.runtimeOptimizer = runtimeopt.NewSystemOptimizer(optConfig, sm.logger)
	if err := sm.runtimeOptimizer.Apply(); err != nil {
		sm.logger.Warn("Failed to apply runtime optimizations", zap.Error(err))
		// Continue startup even if optimizations fail
	} else {
		sm.logger.Info("Runtime optimizations applied successfully",
			zap.Int("level", int(optConfig.Level)),
			zap.Bool("cpu_pinning", optConfig.EnableCPUPinning),
			zap.Bool("memory_locking", optConfig.EnableMemoryLocking),
			zap.Bool("rt_priority", optConfig.EnableRTPriority))
	}

	// Start GC monitoring in background
	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		gctuning.MonitorGCPerformance(sm.logger, 5*time.Minute)
	}()

	// Start runtime optimization monitoring
	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		sm.monitorRuntimeOptimization()
	}()

	return nil
}

// validateLicense checks license validity with proper error handling
func (sm *ServiceManager) validateLicense() error {
	if sm.cfg.LicenseKey == "" {
		if requireLicense := getEnvBool("REQUIRE_LICENSE", false); requireLicense {
			return fmt.Errorf("license key is required but not provided")
		}
		sm.logger.Info("No license key provided, running in open source mode")
		return nil
	}

	// Validate license key
	if !license.Validate(sm.cfg.LicenseKey) {
		return fmt.Errorf("invalid license key")
	}

	// Get license info
	licenseInfo := license.GetInfo(sm.cfg.LicenseKey)
	if !licenseInfo.Valid {
		return fmt.Errorf("invalid license")
	}

	sm.logger.Info("License validated successfully",
		zap.String("tier", licenseInfo.Tier),
		zap.Int("requests_per_hour", licenseInfo.RequestsPerHour),
		zap.Int("concurrent_connections", licenseInfo.ConcurrentConnections))

	// Check if license is about to expire
	expirationTime := time.Unix(licenseInfo.ExpiresAt, 0)
	if time.Until(expirationTime) < 7*24*time.Hour {
		sm.logger.Warn("License will expire soon",
			zap.Time("expires_at", expirationTime),
			zap.String("remaining", time.Until(expirationTime).String()))
	}

	return nil
}

// initializeMetrics sets up metrics collection and export
func (sm *ServiceManager) initializeMetrics() error {
	// Create new registry
	sm.metricsRegistry = prometheus.NewRegistry()

	// Register metrics in the service manager for convenient access
	if sm.cfg.EnablePrometheus {
		// Set up HTTP server for metrics export if configured
		go func() {
			metricsAddr := fmt.Sprintf(":%d", sm.cfg.PrometheusPort)
			sm.logger.Info("Starting Prometheus metrics server", zap.String("address", metricsAddr))

			http.Handle("/metrics", promhttp.HandlerFor(
				sm.metricsRegistry,
				promhttp.HandlerOpts{},
			))

			if err := http.ListenAndServe(metricsAddr, nil); err != nil {
				sm.logger.Error("Prometheus server error", zap.Error(err))
			}
		}()
	}

	return nil
}

// initializeCircuitBreakers sets up circuit breakers for external services
func (sm *ServiceManager) initializeCircuitBreakers() {
	// Circuit breaker for external APIs
	sm.circuitBreakers["external_apis"] = circuitbreaker.NewManager(
		circuitbreaker.ManagerConfig{
			Name:             "external_apis",
			MaxFailures:      5,
			ResetTimeout:     30 * time.Second,
			FailureThreshold: 0.5,
			SuccessThreshold: 3,
			Timeout:          10 * time.Second,
			OnStateChange:    sm.onCircuitBreakerStateChange,
			Logger:           sm.logger,
		},
	)

	// Circuit breaker for database operations
	sm.circuitBreakers["database"] = circuitbreaker.NewManager(
		circuitbreaker.ManagerConfig{
			Name:             "database",
			MaxFailures:      3,
			ResetTimeout:     60 * time.Second,
			FailureThreshold: 0.5,
			SuccessThreshold: 2,
			Timeout:          15 * time.Second,
			OnStateChange:    sm.onCircuitBreakerStateChange,
			Logger:           sm.logger,
		},
	)

	// Circuit breaker for block processing
	sm.circuitBreakers["block_processing"] = circuitbreaker.NewManager(
		circuitbreaker.ManagerConfig{
			Name:             "block_processing",
			MaxFailures:      5,
			ResetTimeout:     45 * time.Second,
			FailureThreshold: 0.6,
			SuccessThreshold: 3,
			Timeout:          30 * time.Second,
			OnStateChange:    sm.onCircuitBreakerStateChange,
			Logger:           sm.logger,
		},
	)
}

// initializeDatabase connects to database with retry logic and runs migrations
func (sm *ServiceManager) initializeDatabase(ctx context.Context) error {
	if sm.cfg.DatabaseURL == "" {
		if sm.cfg.RequireDatabase {
			return fmt.Errorf("database is required but no URL provided")
		}
		sm.logger.Info("No database configured, running without persistence")
		return nil
	}

	dbCfg := database.Config{
		Type:     sm.cfg.DatabaseType,
		URL:      sm.cfg.DatabaseURL,
		MaxConns: 10, // Default value
		MinConns: 2,  // Default value
	}

	var err error
	// Try to connect with retry
	for i := 0; i < 5; i++ {
		sm.db, err = database.New(dbCfg, sm.logger)
		if err == nil {
			break
		}
		sm.logger.Warn("Database connection failed, retrying...", 
			zap.Error(err), 
			zap.Int("attempt", i+1))
		time.Sleep(10 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("database connection failed after retries: %w", err)
	}

	// Test connection
	if err := sm.db.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Run migrations (always enabled for simplicity)
	sm.logger.Info("Running database migrations")
	migrationRunner := migrations.NewRunner(sm.logger)
	if err := migrationRunner.Up(ctx); err != nil {
		return fmt.Errorf("database migrations failed: %w", err)
	}
	sm.logger.Info("Database migrations completed successfully")

	sm.logger.Info("Database connection established",
		zap.String("type", sm.cfg.DatabaseType),
		zap.String("url", obfuscateDBURL(sm.cfg.DatabaseURL)))

	return nil
}

// initializeCoreServices sets up in-memory services
func (sm *ServiceManager) initializeCoreServices() error {
	var err error

	// Initialize rate limiter
	sm.rateLimiter = ratelimit.NewRateLimiter(sm.cfg.GeneralRateLimit, time.Second)

	// Initialize mempool with enhanced configuration
	mempoolMetrics := mempool.NewMempoolMetrics(sm.metricsRegistry)
	sm.mempool = mempool.NewWithMetricsAndConfig(mempool.Config{
		MaxSize:         sm.cfg.MempoolMaxSize,
		ExpiryTime:      5 * time.Minute,     // Default value
		CleanupInterval: 30 * time.Second,    // Default value
	}, mempoolMetrics)

	// Initialize cache with enhanced configuration
	sm.cache = cache.NewWithMetrics(sm.cfg.CacheSize, sm.logger)

	// Initialize broadcaster with enhanced configuration
	sm.broadcaster = broadcaster.New(sm.logger)

	// Initialize throttle manager
	sm.throttleManager = throttle.NewEndpointThrottle(nil, sm.logger)

	// Register endpoints from configuration (without circuit breaker for now)
	for _, endpoint := range sm.cfg.ExternalEndpoints {
		protectedEndpoint := throttle.ProtectedEndpoint{
			URL:      endpoint.URL,
			Priority: endpoint.Priority,
			Timeout:  endpoint.Timeout,
			// CircuitBreaker: nil, // Skip circuit breaker integration for now
		}
		// Note: EndpointThrottle will register endpoints automatically when they're used
		_ = protectedEndpoint // Keep for future use
	}

	// Initialize relay dispatcher
	sm.relayDispatcher = relay.NewRelayDispatcher(*sm.cfg, sm.logger)
	if err != nil {
		return fmt.Errorf("failed to create relay dispatcher: %w", err)
	}

	// Initialize BlockProcessor with configuration
	processorConfig := blocks.ProcessorConfig{
		MaxConcurrentBlocks: 64,
		ProcessingTimeout:   30 * time.Second,
		ValidationTimeout:   10 * time.Second,
		RetryAttempts:       3,
		RetryDelay:          100 * time.Millisecond,
		CacheSize:           1000,
		CacheTTL:            5 * time.Minute,
		EnableMetrics:       true,
		EnableDedup:         true,
	}

	sm.blockProcessor, err = blocks.NewBlockProcessor(processorConfig, sm.logger)
	if err != nil {
		return fmt.Errorf("failed to create block processor: %w", err)
	}

	// Register blockchain validators
	bitcoinValidator := bitcoin.NewValidator(sm.logger)
	ethereumValidator := ethereum.NewValidator(sm.logger)
	solanaValidator := solana.NewValidator(sm.logger)

	sm.blockProcessor.RegisterValidator("bitcoin", bitcoinValidator)
	sm.blockProcessor.RegisterValidator("ethereum", ethereumValidator)
	sm.blockProcessor.RegisterValidator("solana", solanaValidator)

	// Register blockchain processors
	bitcoinProcessor := bitcoin.NewProcessor(sm.logger)
	ethereumProcessor := ethereum.NewProcessor(sm.logger)
	solanaProcessor := solana.NewProcessor(sm.logger)

	sm.blockProcessor.RegisterProcessor("bitcoin", bitcoinProcessor)
	sm.blockProcessor.RegisterProcessor("ethereum", ethereumProcessor)
	sm.blockProcessor.RegisterProcessor("solana", solanaProcessor)

	return nil
}

// startNetworkServices initializes P2P and API servers
func (sm *ServiceManager) startNetworkServices(ctx context.Context) error {
	// Check port availability
	apiAddr := fmt.Sprintf("%s:%d", sm.cfg.APIHost, sm.cfg.APIPort)
	if !isPortAvailable(apiAddr) {
		return fmt.Errorf("API port %s is not available", apiAddr)
	}

		// Initialize P2P client with simple configuration
	var err error
	sm.p2pClient, err = p2p.New(*sm.cfg, sm.blockChan, sm.mempool, sm.logger)
	if err != nil {
		return fmt.Errorf("failed to create P2P client: %w", err)
	}

	// Start P2P client
	go func() {
		sm.p2pClient.Run()
	}()

	// Initialize API server with simple constructor
	if sm.cache != nil {
		sm.apiServer = api.NewWithCache(*sm.cfg, sm.blockChan, sm.mempool, sm.cache, sm.logger)
	} else {
		sm.apiServer = api.New(*sm.cfg, sm.blockChan, sm.mempool, sm.logger)
	}

	// Start API server
	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		if err := sm.apiServer.Run(ctx); err != nil && err != http.ErrServerClosed {
			select {
			case sm.startupErrors <- fmt.Errorf("API server error: %w", err):
			default:
			}
		}
	}()

	return nil
}

// startBackgroundServices initializes backfill and other background tasks
func (sm *ServiceManager) startBackgroundServices(ctx context.Context) error {
	var err error
	// Initialize backfill service with circuit breaker protection
	backfillConfig := messaging.BackfillConfig{
		BatchSize:      sm.cfg.BackfillBatchSize,
		Parallelism:    sm.cfg.BackfillParallelism,
		Timeout:        sm.cfg.BackfillTimeout,
		RetryAttempts:  sm.cfg.BackfillRetryAttempts,
		CircuitBreaker: sm.circuitBreakers["database"],
		MaxBlockRange:  sm.cfg.BackfillMaxBlockRange,
	}
	sm.backfillService, err = messaging.NewBackfillServiceWithMetricsAndConfig(
		backfillConfig, sm.cfg, sm.blockChan, sm.mempool, sm.logger, sm.metrics)
	if err != nil {
		return fmt.Errorf("failed to create backfill service: %w", err)
	}

	if err := sm.backfillService.Start(ctx); err != nil {
		return fmt.Errorf("failed to start backfill service: %w", err)
	}

	// Start block processing
	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		sm.processBlocks()
	}()

	// Start cache pruning
	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		sm.pruneCache()
	}()

	return nil
}

// startHealthServer starts HTTP health check endpoint
func (sm *ServiceManager) startHealthServer() error {
	healthAddr := fmt.Sprintf("%s:%d", sm.cfg.APIHost, sm.cfg.APIPort+1)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", sm.healthCheckHandler)
	mux.HandleFunc("/metrics", sm.metrics.Handler())
	mux.HandleFunc("/ready", sm.readinessHandler)
	mux.HandleFunc("/debug/pprof/", middleware.Profiling(sm.cfg.EnableProfiling))

	sm.healthServer = &http.Server{
		Addr:           healthAddr,
		Handler:        middleware.SecurityHeadersHandler(mux, sm.cfg.EnableSecurityHeaders),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: MaxHeaderBytes,
	}

	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		sm.logger.Info("Health server starting", zap.String("address", healthAddr))
		if sm.cfg.HTTPSEnabled {
			if err := sm.healthServer.ListenAndServeTLS(
				sm.cfg.HTTPSCertFile, sm.cfg.HTTPSKeyFile); err != nil && err != http.ErrServerClosed {
				select {
				case sm.startupErrors <- fmt.Errorf("health server TLS error: %w", err):
				default:
				}
			}
		} else {
			if err := sm.healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				select {
				case sm.startupErrors <- fmt.Errorf("health server error: %w", err):
				default:
				}
			}
		}
	}()

	return nil
}

// processBlocks handles block events from the P2P network
func (sm *ServiceManager) processBlocks() {
	// Use a ticker for batch processing if configured
	var batchTimer *time.Ticker
	var batch []blocks.BlockEvent
	if sm.cfg.BlockBatchSize > 1 {
		batchTimer = time.NewTicker(sm.cfg.BlockBatchTimeout)
		defer batchTimer.Stop()
		batch = make([]blocks.BlockEvent, 0, sm.cfg.BlockBatchSize)
	}

	for {
		select {
		case blockEvent := <-sm.blockChan:
			if sm.cfg.BlockBatchSize > 1 {
				// Batch processing mode
				batch = append(batch, blockEvent)
				if len(batch) >= sm.cfg.BlockBatchSize {
					if err := sm.handleBlockBatch(batch); err != nil {
						sm.logger.Error("Failed to process block batch", zap.Error(err))
					}
					batch = batch[:0] // Reset batch
				}
			} else {
				// Immediate processing mode
				if err := sm.handleBlockEvent(blockEvent); err != nil {
					sm.logger.Error("Failed to process block event", zap.Error(err))
				}
			}
		case <-batchTimer.C:
			// Process any remaining blocks in the batch
			if len(batch) > 0 {
				if err := sm.handleBlockBatch(batch); err != nil {
					sm.logger.Error("Failed to process block batch", zap.Error(err))
				}
				batch = batch[:0] // Reset batch
			}
		case <-sm.shutdownChan:
			// Process any remaining blocks before shutdown
			if len(batch) > 0 {
				if err := sm.handleBlockBatch(batch); err != nil {
					sm.logger.Error("Failed to process final block batch during shutdown", zap.Error(err))
				}
			}
			return
		}
	}
}

// handleBlockEvent processes a single block event
func (sm *ServiceManager) handleBlockEvent(event blocks.BlockEvent) error {
	startTime := time.Now()
	// Deduplicate block
	if sm.blockIdx.IsDuplicate(event.Block.Hash()) {
		sm.metrics.IncrementCounter("blocks_duplicate")
		return nil
	}

	sm.blockIdx.Add(event.Block.Hash())
	sm.metrics.IncrementCounter("blocks_processed")

	// Update cache
	if sm.cache != nil {
		if err := sm.cache.SetBlock(event.Block); err != nil {
			sm.logger.Warn("Failed to cache block",
				zap.String("hash", event.Block.Hash()),
				zap.Error(err))
		}
	}

	// Broadcast to websocket clients
	if sm.broadcaster != nil {
		sm.broadcaster.BroadcastBlock(event.Block)
	}

	// Store in database if available (with circuit breaker protection)
	if sm.db != nil {
		// Use circuit breaker for database operations
		_, err := sm.circuitBreakers["database"].Execute(func() (interface{}, error) {
			return nil, sm.db.StoreBlock(sm.ctx, event.Block)
		})
		if err != nil {
			sm.logger.Error("Failed to store block in database",
				zap.String("hash", event.Block.Hash()),
				zap.Error(err))
			// Don't return error - continue processing
		}
	}

	// Measure processing time
	processingTime := time.Since(startTime)
	sm.metrics.ObserveHistogram("block_processing_time_ms", float64(processingTime.Milliseconds()))
	sm.logger.Debug("Block processed successfully",
		zap.String("hash", event.Block.Hash()),
		zap.Int64("height", event.Block.Height()),
		zap.Duration("processing_time", processingTime))

	return nil
}

// handleBlockBatch processes a batch of block events
func (sm *ServiceManager) handleBlockBatch(batch []blocks.BlockEvent) error {
	if len(batch) == 0 {
		return nil
	}

	startTime := time.Now()
	sm.logger.Debug("Processing block batch", zap.Int("batch_size", len(batch)))

	// Deduplicate blocks in batch
	uniqueBlocks := make([]blocks.Block, 0, len(batch))
	for _, event := range batch {
		if !sm.blockIdx.IsDuplicate(event.Block.Hash()) {
			sm.blockIdx.Add(event.Block.Hash())
			uniqueBlocks = append(uniqueBlocks, event.Block)
		} else {
			sm.metrics.IncrementCounter("blocks_duplicate")
		}
	}

	if len(uniqueBlocks) == 0 {
		return nil // All blocks were duplicates
	}

	// Update cache in batch if supported
	if sm.cache != nil && sm.cache.SupportsBatching() {
		if err := sm.cache.SetBlocks(uniqueBlocks); err != nil {
			sm.logger.Warn("Failed to cache blocks in batch", zap.Error(err))
		}
	}

	// Broadcast to websocket clients
	if sm.broadcaster != nil {
		for _, block := range uniqueBlocks {
			sm.broadcaster.BroadcastBlock(block)
		}
	}

	// Store in database if available (with circuit breaker protection)
	if sm.db != nil {
		_, err := sm.circuitBreakers["database"].Execute(func() (interface{}, error) {
			return nil, sm.db.StoreBlocks(sm.ctx, uniqueBlocks)
		})
		if err != nil {
			sm.logger.Error("Failed to store blocks in database",
				zap.Int("batch_size", len(uniqueBlocks)),
				zap.Error(err))
		}
	}

	// Update metrics
	sm.metrics.IncrementCounterBy("blocks_processed", int64(len(uniqueBlocks)))
	processingTime := time.Since(startTime)
	sm.metrics.ObserveHistogram("block_batch_processing_time_ms", float64(processingTime.Milliseconds()))
	sm.metrics.ObserveHistogram("block_batch_size", float64(len(uniqueBlocks)))

	sm.logger.Debug("Block batch processed successfully",
		zap.Int("original_size", len(batch)),
		zap.Int("processed_size", len(uniqueBlocks)),
		zap.Duration("processing_time", processingTime))

	return nil
}

// pruneCache periodically prunes the cache
func (sm *ServiceManager) pruneCache() {
	if sm.cache == nil {
		return
	}

	ticker := time.NewTicker(sm.cfg.CachePruneInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			startTime := time.Now()
			pruned := sm.cache.Prune()
			sm.metrics.IncrementCounterBy("cache_pruned_items", int64(pruned))
			sm.logger.Debug("Cache pruned",
				zap.Int("items_pruned", pruned),
				zap.Duration("prune_time", time.Since(startTime)))
		case <-sm.shutdownChan:
			return
		}
	}
}

// monitorHealth continuously monitors service health
func (sm *ServiceManager) monitorHealth() {
	defer sm.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.checkServiceHealth()
		case <-sm.shutdownChan:
			return
		}
	}
}

// monitorCircuitBreakers continuously monitors circuit breaker states
func (sm *ServiceManager) monitorCircuitBreakers() {
	defer sm.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for name, cb := range sm.circuitBreakers {
				state := cb.State()
				sm.metrics.SetGauge("circuit_breaker_state", float64(state), map[string]string{"breaker": name})
			}
		case <-sm.shutdownChan:
			return
		}
	}
}

// checkServiceHealth performs health checks on all services
func (sm *ServiceManager) checkServiceHealth() {
	// Check P2P connection
	if sm.p2pClient != nil {
		peers := sm.p2pClient.PeerCount()
		sm.metrics.SetGauge("p2p_peers", float64(peers))
		if peers == 0 {
			sm.logger.Warn("No P2P peers connected")
			sm.metrics.SetGauge("p2p_health", 0)
		} else {
			sm.metrics.SetGauge("p2p_health", 1)
		}
	}

	// Check database connection
	if sm.db != nil {
		ctx, cancel := context.WithTimeout(sm.ctx, 5*time.Second)
		defer cancel()
		if err := sm.db.Ping(ctx); err != nil {
			sm.logger.Warn("Database health check failed", zap.Error(err))
			sm.metrics.SetGauge("database_health", 0)
		} else {
			sm.metrics.SetGauge("database_health", 1)
		}
	}

	// Check cache health
	if sm.cache != nil {
		if sm.cache.HealthCheck() {
			sm.metrics.SetGauge("cache_health", 1)
		} else {
			sm.metrics.SetGauge("cache_health", 0)
		}
	}

	// Check relay dispatcher health
	if sm.relayDispatcher != nil {
		healthStatus := sm.relayDispatcher.GetHealthStatus()
		allHealthy := true
		for network, status := range healthStatus {
			healthValue := 0.0
			if status.IsHealthy {
				healthValue = 1.0
			} else {
				allHealthy = false
			}
			sm.metrics.SetGauge("relay_health", healthValue, map[string]string{"network": network})
		}
		if allHealthy {
			sm.metrics.SetGauge("relay_overall_health", 1)
		} else {
			sm.metrics.SetGauge("relay_overall_health", 0)
		}
	}

	// Check memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	sm.metrics.SetGauge("memory_alloc_bytes", float64(m.Alloc))
	sm.metrics.SetGauge("memory_sys_bytes", float64(m.Sys))
	sm.metrics.SetGauge("memory_heap_objects", float64(m.HeapObjects))
	sm.metrics.SetGauge("goroutines", float64(runtime.NumGoroutine()))
}

// healthCheckHandler returns service health status
func (sm *ServiceManager) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":      "healthy",
		"timestamp":   time.Now().UTC(),
		"version":     AppVersion,
		"node_id":     sm.cfg.NodeID,
		"uptime":      time.Since(sm.startTime).String(),
		"environment": sm.cfg.Environment,
	}

	// Add service-specific health details
	services := make(map[string]interface{})
	// P2P health
	if sm.p2pClient != nil {
		services["p2p"] = map[string]interface{}{
			"healthy": sm.p2pClient.PeerCount() > 0,
			"peers":   sm.p2pClient.PeerCount(),
		}
	}
	// Database health
	if sm.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		dbHealthy := sm.db.Ping(ctx) == nil
		services["database"] = map[string]interface{}{
			"healthy": dbHealthy,
		}
		if !dbHealthy {
			health["status"] = "degraded"
		}
	}
	// Cache health
	if sm.cache != nil {
		services["cache"] = map[string]interface{}{
			"healthy": sm.cache.HealthCheck(),
			"size":    sm.cache.Size(),
		}
	}
	health["services"] = services

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Node-ID", sm.cfg.NodeID)
	if health["status"] != "healthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	if err := json.NewEncoder(w).Encode(health); err != nil {
		sm.logger.Error("Failed to encode health response", zap.Error(err))
	}
}

// readinessHandler returns readiness status
func (sm *ServiceManager) readinessHandler(w http.ResponseWriter, r *http.Request) {
	ready := map[string]bool{
		"api":   sm.apiServer != nil,
		"p2p":   sm.p2pClient != nil && sm.p2pClient.PeerCount() > 0,
		"cache": sm.cache != nil && sm.cache.HealthCheck(),
		"relay": sm.relayDispatcher != nil,
		"db":    sm.db == nil || sm.db.Ping(r.Context()) == nil,
	}

	allReady := true
	for service, isReady := range ready {
		if !isReady {
			allReady = false
			sm.logger.Warn("Service not ready", zap.String("service", service))
		}
	}

	status := http.StatusOK
	if !allReady {
		status = http.StatusServiceUnavailable
	}

	response := map[string]interface{}{
		"ready":    allReady,
		"services": ready,
		"node_id":  sm.cfg.NodeID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Node-ID", sm.cfg.NodeID)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		sm.logger.Error("Failed to encode readiness response", zap.Error(err))
	}
}

// onCircuitBreakerStateChange handles circuit breaker state changes
func (sm *ServiceManager) onCircuitBreakerStateChange(name string, from, to circuitbreaker.State) {
	sm.logger.Info("Circuit breaker state changed",
		zap.String("name", name),
		zap.String("from", from.String()),
		zap.String("to", to.String()))
	sm.metrics.IncrementCounter("circuit_breaker_state_changes",
		map[string]string{"breaker": name, "from": from.String(), "to": to.String()})
}

// validateConfig validates the loaded configuration
func validateConfig(cfg config.Config) error {
	// Validate API configuration
	if cfg.APIPort <= 0 || cfg.APIPort > 65535 {
		return fmt.Errorf("invalid API port: %d", cfg.APIPort)
	}
	if !isValidHost(cfg.APIHost) {
		return fmt.Errorf("invalid API host: %s", cfg.APIHost)
	}

	// Validate network configuration
	if cfg.P2PListenAddress != "" {
		if _, _, err := net.SplitHostPort(cfg.P2PListenAddress); err != nil {
			return fmt.Errorf("invalid P2P listen address: %w", err)
		}
	}

	// Validate resource limits
	if cfg.BlockChannelBuffer <= 0 {
		return fmt.Errorf("block channel buffer must be positive")
	}
	if cfg.CacheSize <= 0 {
		return fmt.Errorf("cache size must be positive")
	}
	if cfg.MempoolMaxSize <= 0 {
		return fmt.Errorf("mempool max size must be positive")
	}

	// Validate timeouts
	if cfg.APIReadTimeout <= 0 {
		return fmt.Errorf("API read timeout must be positive")
	}
	if cfg.APIWriteTimeout <= 0 {
		return fmt.Errorf("API write timeout must be positive")
	}
	if cfg.P2PPeerTimeout <= 0 {
		return fmt.Errorf("P2P peer timeout must be positive")
	}

	// Validate database configuration
	if cfg.DatabaseType != "" && cfg.DatabaseURL == "" {
		return fmt.Errorf("database type specified but no URL provided")
	}
	if cfg.DatabaseType != "" {
		if !isValidDatabaseType(cfg.DatabaseType) {
			return fmt.Errorf("unsupported database type: %s", cfg.DatabaseType)
		}
	}

	// Validate external endpoints
	for i, endpoint := range cfg.ExternalEndpoints {
		if _, err := url.Parse(endpoint.URL); err != nil {
			return fmt.Errorf("invalid external endpoint URL %s: %w", endpoint.URL, err)
		}
		if endpoint.Timeout <= 0 {
			return fmt.Errorf("timeout for endpoint %s must be positive", endpoint.URL)
		}
		if endpoint.Priority < 0 || endpoint.Priority > 100 {
			return fmt.Errorf("priority for endpoint %s must be between 0 and 100", endpoint.URL)
		}
		_ = i
	}

	// Validate rate limiting configuration
	if cfg.GlobalAPIRateLimit <= 0 {
		return fmt.Errorf("global API rate limit must be positive")
	}
	if cfg.PerIPRateLimit <= 0 {
		return fmt.Errorf("per-IP rate limit must be positive")
	}

	// Validate HTTPS configuration if enabled
	if cfg.HTTPSEnabled {
		if cfg.HTTPSCertFile == "" {
			return fmt.Errorf("HTTPS certificate file must be specified when HTTPS is enabled")
		}
		if cfg.HTTPSKeyFile == "" {
			return fmt.Errorf("HTTPS key file must be specified when HTTPS is enabled")
		}
		// Check if certificate files exist
		if _, err := os.Stat(cfg.HTTPSCertFile); os.IsNotExist(err) {
			return fmt.Errorf("HTTPS certificate file does not exist: %s", cfg.HTTPSCertFile)
		}
		if _, err := os.Stat(cfg.HTTPSKeyFile); os.IsNotExist(err) {
			return fmt.Errorf("HTTPS key file does not exist: %s", cfg.HTTPSKeyFile)
		}
	}

	return nil
}

// Helper functions

// isValidHost validates a hostname or IP address
func isValidHost(host string) bool {
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "0.0.0.0" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return true
	}
	// Validate hostname
	if len(host) > 253 {
		return false
	}
	// Check each label in the hostname
	labels := strings.Split(host, ".")
	for _, label := range labels {
		if len(label) < 1 || len(label) > 63 {
			return false
		}
		if !regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?$`).MatchString(label) {
			return false
		}
	}
	return true
}

// isValidDatabaseType validates database type
func isValidDatabaseType(dbType string) bool {
	validTypes := []string{"postgres", "mysql", "sqlite", "cassandra"}
	for _, validType := range validTypes {
		if dbType == validType {
			return true
		}
	}
	return false
}

// isPortAvailable checks if a port is available for binding
func isPortAvailable(addr string) bool {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// obfuscateDBURL obfuscates sensitive information in database URL for logging
func obfuscateDBURL(dbURL string) string {
	u, err := url.Parse(dbURL)
	if err != nil {
		return "invalid-url"
	}
	// Clear password and sensitive query parameters
	if u.User != nil {
		if _, hasPassword := u.User.Password(); hasPassword {
			u.User = url.UserPassword(u.User.Username(), "xxxxx")
		}
	}
	// Remove sensitive query parameters
	query := u.Query()
	for _, param := range []string{"password", "pass", "key", "secret", "token"} {
		if query.Has(param) {
			query.Set(param, "xxxxx")
		}
	}
	u.RawQuery = query.Encode()
	return u.String()
}

// getStacktrace gets the current stacktrace
func getStacktrace() []byte {
	buf := make([]byte, 1<<16)
	n := runtime.Stack(buf, true)
	return buf[:n]
}

// mustInitLogger initializes the logger or panics
func mustInitLogger() *zap.Logger {
	// Try to load config from environment first
	config := zap.NewProductionConfig()
	// Set log level from environment or default to info
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(logLevel)); err != nil {
		level = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(level)
	// Configure encoder
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.StacktraceKey = "stacktrace"
	config.EncoderConfig.MessageKey = "message"
	// Set output paths
	if os.Getenv("LOG_FORMAT") == "console" {
		config.Encoding = "console"
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config.Encoding = "json"
	}
	// Add service name to all logs
	hostname, _ := os.Hostname()
	config.InitialFields = map[string]interface{}{
		"service":    AppName,
		"version":    AppVersion,
		"hostname":   hostname,
		"go_version": runtime.Version(),
	}

	logger, err := config.Build()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	return logger
}

// monitorRuntimeOptimization monitors runtime optimization performance
func (sm *ServiceManager) monitorRuntimeOptimization() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			if sm.runtimeOptimizer != nil {
				stats := sm.runtimeOptimizer.GetStats()
				
				// Update metrics if available
				if sm.metrics != nil {
					if val, ok := stats["heap_alloc_mb"].(uint64); ok {
						sm.metrics.SetGauge("runtime_heap_mb", float64(val))
					}
					if val, ok := stats["num_goroutine"].(int); ok {
						sm.metrics.SetGauge("runtime_goroutines", float64(val))
					}
					if val, ok := stats["gc_cpu_fraction"].(float64); ok {
						sm.metrics.SetGauge("runtime_gc_cpu_fraction", val)
					}
					if val, ok := stats["applied"].(bool); ok {
						if val {
							sm.metrics.SetGauge("runtime_optimization_active", 1)
						} else {
							sm.metrics.SetGauge("runtime_optimization_active", 0)
						}
					}
				}
				
				// Log periodic runtime stats
				if applied, ok := stats["applied"].(bool); ok && applied {
					sm.logger.Debug("Runtime optimization stats",
						zap.Any("stats", stats))
				}
			}
		}
	}
}
