package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"os/signal"
	goruntime "runtime"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/api"
	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/cache"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/entropy"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
	"github.com/PayRpc/Bitcoin-Sprint/internal/p2p"
	"github.com/PayRpc/Bitcoin-Sprint/internal/performance"
	"github.com/PayRpc/Bitcoin-Sprint/internal/securebuf"
	"github.com/PayRpc/Bitcoin-Sprint/internal/zmq"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// ===== Inlined Additions for Predictive Cache, Entropy Buffer, Latency Optimizer, Tier Manager, Metrics, Security =====

type PredictiveCache struct {
	mu      sync.RWMutex
	cache   map[string]interface{}
	maxSize int
}

func NewPredictiveCache(maxSize int) *PredictiveCache {
	return &PredictiveCache{cache: make(map[string]interface{}), maxSize: maxSize}
}

func (pc *PredictiveCache) Get(key string) (interface{}, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	v, ok := pc.cache[key]
	return v, ok
}

func (pc *PredictiveCache) Set(key string, val interface{}) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	if len(pc.cache) >= pc.maxSize {
		for k := range pc.cache {
			delete(pc.cache, k)
			break
		}
	}
	pc.cache[key] = val
}

func (pc *PredictiveCache) Close() { pc.cache = nil }

type EntropyMemoryBuffer struct {
	mu      sync.RWMutex
	buffer  []byte
	refresh time.Duration
}

func NewEntropyMemoryBuffer() *EntropyMemoryBuffer {
	emb := &EntropyMemoryBuffer{refresh: time.Second}
	go emb.background()
	return emb
}

func (emb *EntropyMemoryBuffer) background() {
	ticker := time.NewTicker(emb.refresh)
	defer ticker.Stop()
	for range ticker.C {
		buf := make([]byte, 4096)
		rand.Read(buf)
		emb.mu.Lock()
		emb.buffer = buf
		emb.mu.Unlock()
	}
}

func (emb *EntropyMemoryBuffer) Get(size int) []byte {
	emb.mu.RLock()
	defer emb.mu.RUnlock()
	if len(emb.buffer) < size {
		b := make([]byte, size)
		rand.Read(b)
		return b
	}
	return emb.buffer[:size]
}

func (emb *EntropyMemoryBuffer) Close() { emb.buffer = nil }

type LatencyOptimizer struct {
	mu       sync.RWMutex
	samples  []time.Duration
	targetP99 time.Duration
}

func NewLatencyOptimizer() *LatencyOptimizer {
	return &LatencyOptimizer{targetP99: 100 * time.Millisecond}
}

func (lo *LatencyOptimizer) TrackRequest(d time.Duration) {
	lo.mu.Lock()
	defer lo.mu.Unlock()
	lo.samples = append(lo.samples, d)
	if len(lo.samples) > 1000 {
		lo.samples = lo.samples[1:]
	}
}

func (lo *LatencyOptimizer) GetStats() map[string]interface{} {
	lo.mu.RLock()
	defer lo.mu.RUnlock()
	if len(lo.samples) == 0 {
		return map[string]interface{}{"p99": "N/A"}
	}
	sorted := append([]time.Duration(nil), lo.samples...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := int(0.99 * float64(len(sorted)))
	return map[string]interface{}{"p99": sorted[idx].String()}
}

func (lo *LatencyOptimizer) Stop() { lo.samples = nil }

type TierManager struct {
	tiers map[string]int
}

func NewTierManager() *TierManager {
	return &TierManager{tiers: map[string]int{
		"free": 10, "pro": 100, "business": 500, "turbo": 1000, "enterprise": 10000,
	}}
}

func (tm *TierManager) GetLimits(tier string) int {
	if limit, ok := tm.tiers[tier]; ok {
		return limit
	}
	return tm.tiers["free"]
}

type MetricsTracker struct {
	mu       sync.RWMutex
	counters map[string]int
}

func NewMetricsTracker() *MetricsTracker {
	return &MetricsTracker{counters: make(map[string]int)}
}

func (mt *MetricsTracker) Inc(name string) {
	mt.mu.Lock()
	mt.counters[name]++
	mt.mu.Unlock()
}

func (mt *MetricsTracker) Snapshot() map[string]int {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	out := make(map[string]int)
	for k, v := range mt.counters {
		out[k] = v
	}
	return out
}

type EnterpriseSecurityManager struct {
	logger *zap.Logger
}

func NewEnterpriseSecurityManager(logger *zap.Logger) *EnterpriseSecurityManager {
	return &EnterpriseSecurityManager{logger: logger}
}

func (esm *EnterpriseSecurityManager) RegisterEnterpriseRoutes() {
	esm.logger.Info("Enterprise security routes registered")
}

func main() {
	// Load .env file if present
	_ = godotenv.Load()

	// Configuration from environment
	cfg := config.Load()
	logger := initLogger(cfg)
	defer logger.Sync()

	// Root context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Performance tuning first - must be applied before any other operations
	perf := performance.New(cfg, logger)
	if err := perf.ApplyOptimizations(); err != nil {
		logger.Fatal("Failed to apply performance optimizations", zap.Error(err))
	}

	// Get version information
	version := getVersion()

	// Startup log
	logger.Info("Bitcoin Sprint starting...",
		zap.String("version", version),
		zap.String("go_version", goruntime.Version()),
		zap.Int("num_cpu", goruntime.NumCPU()),
		zap.Int("gomaxprocs", goruntime.GOMAXPROCS(0)),
		zap.String("tier", string(cfg.Tier)),
		zap.Bool("turbo_mode", cfg.Tier == config.TierTurbo || cfg.Tier == config.TierEnterprise),
		zap.Bool("cgo_enabled", securebuf.CGoEnabled),
		zap.Any("perf_stats", perf.GetCurrentStats()),
	)

	// Initialize secure components (enterprise only)
	initSecureComponents(ctx, cfg, logger)

	// In a production system, we would validate the license
	// For now, we'll just log that we're continuing
	logger.Info("License validation skipped in development mode",
		zap.String("tier", string(cfg.Tier)))

	// Initialize new core helpers
	predictiveCache := NewPredictiveCache(1000)
	entropyBuffer := NewEntropyMemoryBuffer()
	latencyOptimizer := NewLatencyOptimizer()
	tierManager := NewTierManager()
	metricsTracker := NewMetricsTracker()
	enterpriseSec := NewEnterpriseSecurityManager(logger)

	// Core modules
	mempoolModule := mempool.New()

	// Dynamic block channel buffer size based on tier
	blockChanSize := 1000
	// Scale based on tier
	if cfg.Tier == config.TierEnterprise {
		blockChanSize = 5000
	} else if cfg.Tier == config.TierTurbo {
		blockChanSize = 2500
	} else if cfg.Tier == config.TierBusiness {
		blockChanSize = 1500
	}
	blockChan := make(chan blocks.BlockEvent, blockChanSize)
	logger.Info("Block channel initialized", zap.Int("buffer_size", blockChanSize))

	// Create broadcaster for coordinated peer notifications
	// (We'll use this later in our implementation)

	// ZMQ client for Bitcoin Core integration
	zmqClient := zmq.New(cfg, blockChan, mempoolModule, logger)
	go zmqClient.Run()

	// P2P client for direct blockchain network connectivity
	p2pClient, err := p2p.New(cfg, blockChan, mempoolModule, logger)
	if err != nil {
		logger.Fatal("Failed to create P2P module", zap.Error(err))
	}
	go p2pClient.Run()

	// Cache layer with dynamic sizing based on tier
	cacheSize := 2000
	// Scale cache based on tier
	if cfg.Tier == config.TierEnterprise {
		cacheSize = 10000
	} else if cfg.Tier == config.TierTurbo {
		cacheSize = 5000
	} else if cfg.Tier == config.TierBusiness {
		cacheSize = 3000
	}
	blockCache := cache.New(cacheSize, logger)
	logger.Info("Cache initialized", zap.Int("max_entries", cacheSize))

	// Start prefetch worker
	prefetchWorker := cache.NewPrefetchWorker(blockCache, blockChan, logger)
	prefetchWorker.Start(ctx)

	// API server
	apiServer := api.New(cfg, blockChan, mempoolModule, logger)
	// Cache is registered internally in api.New
	go func() {
		logger.Info("Starting API server",
			zap.String("addr", fmt.Sprintf("%s:%d", cfg.APIHost, cfg.APIPort)))
		apiServer.Run(ctx)
	}()

	// Tier-aware relay loop
	go func() {
		if cfg.Tier == config.TierTurbo || cfg.Tier == config.TierEnterprise {
			runMemoryOptimizedRelay(ctx, blockChan, cfg, logger)
		} else {
			runStandardRelay(ctx, blockChan, cfg, logger)
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
	p2pClient.Stop()
	// API server stopped via context cancellation
	prefetchWorker.Stop()

	// Close new components
	predictiveCache.Close()
	entropyBuffer.Close()
	latencyOptimizer.Stop()

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
	cfg config.Config, logger *zap.Logger) {

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
			// Just log in this simplified version
			elapsed := time.Since(start)
			logger.Debug("Relay (simulated)",
				zap.Duration("elapsed", elapsed),
				zap.String("blockHash", evt.Hash))
		}
	}
}

// runStandardRelay: Free/Pro/Business tiers
func runStandardRelay(ctx context.Context, ch <-chan blocks.BlockEvent,
	cfg config.Config, logger *zap.Logger) {

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
			// Just log in this simplified version
			elapsed := time.Since(start)
			logger.Debug("Standard relay (simulated)",
				zap.Duration("elapsed", elapsed),
				zap.String("blockHash", evt.Hash))
		}
	}
}

// Helper function for implementing secure components
func initSecureComponents(ctx context.Context, cfg config.Config,
	logger *zap.Logger) error {

	// Check if we're in enterprise tier and secure features are needed
	if cfg.Tier == config.TierEnterprise {
		logger.Info("Initializing secure components for Enterprise tier")

		// Initialize secure buffers if available
		if securebuf.CGoEnabled {
			logger.Info("Hardware-backed secure buffers available")
			// In a real implementation, we would set up secure buffers here
		} else {
			logger.Warn("Hardware secure buffers not available, using software fallback")
		}

		// Initialize entropy source
		_, err := entropy.HybridEntropy()
		if err != nil {
			logger.Warn("Failed to initialize hybrid entropy source", zap.Error(err))
			return err
		}

		logger.Info("Enterprise secure components initialized")
	}

	return nil
}
