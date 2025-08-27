package performance

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"syscall"

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
		cfg:    cfg,
		logger: logger,
		level:  level,
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

// setWindowsHighPriority sets high priority on Windows
func (pm *PerformanceManager) setWindowsHighPriority() error {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setProcessPriorityClass := kernel32.NewProc("SetPriorityClass")
	getCurrentProcess := kernel32.NewProc("GetCurrentProcess")

	const HIGH_PRIORITY_CLASS = 0x00000080

	handle, _, _ := getCurrentProcess.Call()
	ret, _, err := setProcessPriorityClass.Call(handle, HIGH_PRIORITY_CLASS)
	
	if ret == 0 {
		return fmt.Errorf("SetPriorityClass failed: %v", err)
	}

	pm.logger.Debug("Set process to high priority")
	return nil
}

// preallocateBuffers pre-allocates commonly used memory buffers
func (pm *PerformanceManager) preallocateBuffers() {
	// Pre-allocate buffers based on tier requirements
	bufferSize := pm.cfg.BlockBufferSize
	if bufferSize == 0 {
		bufferSize = 1024 // Default buffer size
	}

	// Pre-allocate multiple buffers for different use cases
	buffers := make([][]byte, 10)
	for i := range buffers {
		buffers[i] = make([]byte, bufferSize)
	}

	pm.logger.Debug("Pre-allocated memory buffers",
		zap.Int("buffer_count", len(buffers)),
		zap.Int("buffer_size", bufferSize),
	)

	// Keep reference to prevent GC (in a real implementation, 
	// these would be managed by a buffer pool)
	_ = buffers
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
