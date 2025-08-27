package performance

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"

	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"go.uber.org/zap"
)

// OptimizationLevel represents different performance optimization levels
type OptimizationLevel int

const (
	LevelStandard OptimizationLevel = iota // Standard performance
	LevelHigh                              // High performance (default)
	LevelMaximum                           // Maximum performance (99.9% SLA)
)

// PerformanceManager handles system-level performance optimizations
type PerformanceManager struct {
	cfg    config.Config
	logger *zap.Logger
	level  OptimizationLevel

	// Enhanced buffer pool for memory management
	bufferPool *BufferPool
}

// BufferPool manages reusable memory buffers
type BufferPool struct {
	pools    map[int]*sync.Pool
	mu       sync.RWMutex
	maxSize  int
}

// New creates a new performance manager with automatic optimization level detection
func New(cfg config.Config, logger *zap.Logger) *PerformanceManager {
	level := LevelHigh // Default to high performance
	
	// Determine optimization level based on tier
	switch cfg.Tier {
	case config.TierTurbo, config.TierEnterprise:
		level = LevelMaximum // 99.9% SLA compliance
	case config.TierPro, config.TierBusiness:
		level = LevelHigh // High performance
	default:
		level = LevelStandard // Standard performance
	}

	return &PerformanceManager{
		cfg:        cfg,
		logger:     logger,
		level:      level,
		bufferPool: NewBufferPool(),
	}
}

// NewBufferPool creates a new buffer pool for memory management
func NewBufferPool() *BufferPool {
	return &BufferPool{
		pools:   make(map[int]*sync.Pool),
		maxSize: 1024 * 1024, // 1MB max buffer size
	}
}

// Get retrieves a buffer from the pool
func (bp *BufferPool) Get(size int) []byte {
	if size > bp.maxSize {
		return make([]byte, size)
	}
	
	bp.mu.RLock()
	pool, exists := bp.pools[size]
	bp.mu.RUnlock()
	
	if !exists {
		bp.mu.Lock()
		if bp.pools[size] == nil {
			bp.pools[size] = &sync.Pool{
				New: func() interface{} {
					return make([]byte, size)
				},
			}
		}
		pool = bp.pools[size]
		bp.mu.Unlock()
	}
	
	return pool.Get().([]byte)
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(buf []byte) {
	size := cap(buf)
	if size > bp.maxSize {
		return
	}
	
	bp.mu.RLock()
	pool, exists := bp.pools[size]
	bp.mu.RUnlock()
	
	if exists {
		// Clear buffer before returning to pool
		for i := range buf {
			buf[i] = 0
		}
		pool.Put(buf[:0]) // Reset length but keep capacity
	}
}

// ApplyOptimizations applies all performance optimizations based on configuration
func (pm *PerformanceManager) ApplyOptimizations() error {
	pm.logger.Info("Applying performance optimizations",
		zap.String("level", pm.getOptimizationLevelName()),
		zap.String("tier", string(pm.cfg.Tier)),
	)

	// 1. Runtime optimizations (always applied)
	pm.applyRuntimeOptimizations()

	// 2. Memory optimizations 
	pm.applyMemoryOptimizations()

	// 3. CPU optimizations
	pm.applyCPUOptimizations()

	// 4. System-level optimizations (if enabled)
	if pm.cfg.OptimizeSystem {
		if err := pm.applySystemOptimizations(); err != nil {
			pm.logger.Warn("System optimizations failed", zap.Error(err))
			// Don't fail startup, just log warning
		}
	}

	pm.logger.Info("Performance optimizations applied successfully",
		zap.Int("gomaxprocs", runtime.GOMAXPROCS(0)),
		zap.Int("gc_percent", pm.cfg.GCPercent),
	)

	return nil
}

// GetBufferPool returns the buffer pool for external use
func (pm *PerformanceManager) GetBufferPool() *BufferPool {
	return pm.bufferPool
}

// applyRuntimeOptimizations applies Go runtime optimizations
func (pm *PerformanceManager) applyRuntimeOptimizations() {
	// Lock main thread to OS thread for consistent latency
	if pm.cfg.LockOSThread {
		runtime.LockOSThread()
		pm.logger.Debug("Locked main thread to OS thread")
	}

	// Configure garbage collector
	if pm.cfg.GCPercent > 0 {
		oldPercent := debug.SetGCPercent(pm.cfg.GCPercent)
		pm.logger.Debug("Configured garbage collector",
			zap.Int("old_percent", oldPercent),
			zap.Int("new_percent", pm.cfg.GCPercent),
		)
	}

	// Set CPU core usage
	if pm.cfg.MaxCPUCores > 0 {
		oldProcs := runtime.GOMAXPROCS(pm.cfg.MaxCPUCores)
		pm.logger.Debug("Configured CPU cores",
			zap.Int("old_procs", oldProcs),
			zap.Int("new_procs", pm.cfg.MaxCPUCores),
		)
	} else if pm.cfg.MaxCPUCores == 0 {
		// Auto-detect and use all available cores
		cores := runtime.NumCPU()
		runtime.GOMAXPROCS(cores)
		pm.logger.Debug("Auto-configured CPU cores", zap.Int("cores", cores))
	}
}

// applyMemoryOptimizations applies memory-related optimizations
func (pm *PerformanceManager) applyMemoryOptimizations() {
	if pm.level >= LevelHigh {
		// Pre-allocate commonly used buffers
		if pm.cfg.PreallocBuffers {
			pm.preallocateBuffers()
		}
	}

	if pm.level >= LevelMaximum {
		// Maximum performance: disable GC for ultra-low latency
		debug.SetGCPercent(-1)
		pm.logger.Debug("Disabled garbage collector for maximum performance")
	}
}

// applyCPUOptimizations applies CPU-related optimizations
func (pm *PerformanceManager) applyCPUOptimizations() {
	// CPU affinity and priority optimizations will be applied
	// in applySystemOptimizations() as they require OS-specific calls
}

// applySystemOptimizations applies OS-level optimizations
func (pm *PerformanceManager) applySystemOptimizations() error {
	if pm.cfg.HighPriority {
		if err := pm.setHighPriority(); err != nil {
			return fmt.Errorf("failed to set high priority: %w", err)
		}
	}

	return nil
}

// setHighPriority sets the process to high priority (Windows-specific)
func (pm *PerformanceManager) setHighPriority() error {
	if runtime.GOOS == "windows" {
	return pm.setWindowsHighPriority()
	}
	
	// For Unix-like systems, we could implement nice() calls here
	pm.logger.Debug("High priority optimization not implemented for this OS")
	return nil
}

// preallocateBuffers pre-allocates commonly used memory buffers
func (pm *PerformanceManager) preallocateBuffers() {
	// Pre-allocate buffers based on tier requirements
	bufferSize := pm.cfg.BlockBufferSize
	if bufferSize == 0 {
		bufferSize = 1024 // Default buffer size
	}

	// Pre-populate the buffer pool with commonly used buffer sizes
	bufferSizes := []int{bufferSize, bufferSize*2, bufferSize*4, 4096, 8192}
	
	totalBuffers := 0
	for _, size := range bufferSizes {
		// Pre-allocate 5 buffers of each size
		for i := 0; i < 5; i++ {
			buf := pm.bufferPool.Get(size)
			pm.bufferPool.Put(buf) // Return to pool immediately
			totalBuffers++
		}
	}

	pm.logger.Debug("Pre-populated buffer pool",
		zap.Int("total_buffers", totalBuffers),
		zap.Int("buffer_sizes", len(bufferSizes)),
		zap.Int("pool_max_size", pm.bufferPool.maxSize),
	)
}

// getOptimizationLevelName returns the string representation of optimization level
func (pm *PerformanceManager) getOptimizationLevelName() string {
	switch pm.level {
	case LevelMaximum:
		return "maximum"
	case LevelHigh:
		return "high"
	default:
		return "standard"
	}
}

// GetCurrentStats returns current performance statistics
func (pm *PerformanceManager) GetCurrentStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"optimization_level": pm.getOptimizationLevelName(),
		"tier":              string(pm.cfg.Tier),
		"runtime": map[string]interface{}{
			"gomaxprocs":     runtime.GOMAXPROCS(0),
			"num_cpu":        runtime.NumCPU(),
			"num_goroutines": runtime.NumGoroutine(),
			"gc_percent":     pm.cfg.GCPercent,
		},
		"memory": map[string]interface{}{
			"alloc_mb":       m.Alloc / 1024 / 1024,
			"total_alloc_mb": m.TotalAlloc / 1024 / 1024,
			"sys_mb":         m.Sys / 1024 / 1024,
			"num_gc":         m.NumGC,
		},
		"config": map[string]interface{}{
			"lock_os_thread":    pm.cfg.LockOSThread,
			"high_priority":     pm.cfg.HighPriority,
			"prealloc_buffers":  pm.cfg.PreallocBuffers,
			"optimize_system":   pm.cfg.OptimizeSystem,
		},
	}
}
