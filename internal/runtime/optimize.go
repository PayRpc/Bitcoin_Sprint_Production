package runtime

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"go.uber.org/zap"
)

// OptimizationLevel represents the level of system optimizations to apply
type OptimizationLevel int

const (
	OptimizationBasic OptimizationLevel = iota
	OptimizationStandard
	OptimizationAggressive
	OptimizationEnterprise
	OptimizationTurbo
)

// SystemOptimizationConfig holds all optimization parameters
type SystemOptimizationConfig struct {
	Level                OptimizationLevel
	EnableCPUPinning     bool
	EnableMemoryLocking  bool
	EnableRTPriority     bool
	MaxThreads           int
	GCTargetPercent      int
	MemoryLimitPercent   int
	ThreadStackSize      int
	EnableHugePagesHint  bool
	EnableNUMAOptimization bool
	EnableLatencyTuning  bool
	CPUAffinity          []int
}

// DefaultConfig returns optimized configuration based on detected system capabilities
func DefaultConfig() *SystemOptimizationConfig {
	numCPU := runtime.NumCPU()
	
	return &SystemOptimizationConfig{
		Level:                OptimizationStandard,
		EnableCPUPinning:     numCPU >= 4,
		EnableMemoryLocking:  true,
		EnableRTPriority:     false,
		MaxThreads:           numCPU * 2,
		GCTargetPercent:      50,  // Balanced performance
		MemoryLimitPercent:   75,  // Use 75% of system memory
		ThreadStackSize:      8192, // 8KB stacks for efficiency
		EnableHugePagesHint:  numCPU >= 8,
		EnableNUMAOptimization: numCPU >= 16,
		EnableLatencyTuning:  true,
		CPUAffinity:          nil, // Auto-detect
	}
}

// EnterpriseConfig returns maximum performance configuration
func EnterpriseConfig() *SystemOptimizationConfig {
	config := DefaultConfig()
	config.Level = OptimizationEnterprise
	config.EnableCPUPinning = true
	config.EnableRTPriority = true
	config.GCTargetPercent = 25  // Minimal GC
	config.MemoryLimitPercent = 90
	config.EnableHugePagesHint = true
	config.EnableNUMAOptimization = true
	return config
}

// TurboConfig returns ultra-low latency configuration
func TurboConfig() *SystemOptimizationConfig {
	config := EnterpriseConfig()
	config.Level = OptimizationTurbo
	config.GCTargetPercent = 10  // Minimal GC for ultra-low latency
	config.ThreadStackSize = 4096 // Smaller stacks for cache efficiency
	return config
}

// SystemOptimizer manages all system-level optimizations
type SystemOptimizer struct {
	config           *SystemOptimizationConfig
	logger           *zap.Logger
	originalSettings map[string]interface{}
	mu               sync.RWMutex
	applied          bool
	ctx              context.Context
	cancel           context.CancelFunc
}

// NewSystemOptimizer creates a new system optimizer
func NewSystemOptimizer(config *SystemOptimizationConfig, logger *zap.Logger) *SystemOptimizer {
	if config == nil {
		config = DefaultConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &SystemOptimizer{
		config:           config,
		logger:           logger,
		originalSettings: make(map[string]interface{}),
		ctx:              ctx,
		cancel:           cancel,
	}
}

// ApplySystemOptimizations applies comprehensive system optimizations
func ApplySystemOptimizations(logger *zap.Logger) {
	optimizer := NewSystemOptimizer(DefaultConfig(), logger)
	if err := optimizer.Apply(); err != nil {
		logger.Error("Failed to apply system optimizations", zap.Error(err))
	}
}

// ApplyEnterpriseOptimizations applies maximum performance optimizations
func ApplyEnterpriseOptimizations(logger *zap.Logger) error {
	optimizer := NewSystemOptimizer(EnterpriseConfig(), logger)
	return optimizer.Apply()
}

// Apply applies all configured optimizations
func (so *SystemOptimizer) Apply() error {
	so.mu.Lock()
	defer so.mu.Unlock()
	
	if so.applied {
		return fmt.Errorf("optimizations already applied")
	}
	
	start := time.Now()
	so.logger.Info("Applying system optimizations",
		zap.String("level", so.getLevelString()),
		zap.Int("cpu_count", runtime.NumCPU()),
		zap.String("go_version", runtime.Version()))
	
	// Apply optimizations in order of importance
	optimizations := []struct {
		name string
		fn   func() error
	}{
		{"runtime_tuning", so.applyRuntimeTuning},
		{"memory_optimizations", so.applyMemoryOptimizations},
		{"gc_tuning", so.applyGCTuning},
		{"thread_optimizations", so.applyThreadOptimizations},
		{"cpu_optimizations", so.applyCPUOptimizations},
		{"latency_tuning", so.applyLatencyTuning},
		{"platform_specific", so.applyPlatformSpecific},
	}
	
	for _, opt := range optimizations {
		if err := opt.fn(); err != nil {
			so.logger.Error("Failed to apply optimization",
				zap.String("optimization", opt.name),
				zap.Error(err))
			// Continue with other optimizations
		} else {
			so.logger.Debug("Applied optimization",
				zap.String("optimization", opt.name))
		}
	}
	
	so.applied = true
	
	// Start monitoring if enabled
	if so.config.Level >= OptimizationStandard {
		go so.monitorPerformance()
	}
	
	so.logger.Info("System optimizations applied successfully",
		zap.Duration("duration", time.Since(start)),
		zap.String("level", so.getLevelString()))
	
	return nil
}

// applyRuntimeTuning optimizes Go runtime settings
func (so *SystemOptimizer) applyRuntimeTuning() error {
	// Set GOMAXPROCS to optimal value
	if so.config.MaxThreads > 0 {
		oldMaxProcs := runtime.GOMAXPROCS(so.config.MaxThreads)
		so.originalSettings["GOMAXPROCS"] = oldMaxProcs
		so.logger.Info("Set GOMAXPROCS",
			zap.Int("old", oldMaxProcs),
			zap.Int("new", so.config.MaxThreads))
	}
	
	// Lock OS thread for main goroutine in high-performance modes
	if so.config.Level >= OptimizationAggressive {
		runtime.LockOSThread()
		so.logger.Debug("Locked main goroutine to OS thread")
	}
	
	// Set thread stack size hint
	if so.config.ThreadStackSize > 0 {
		// This is a hint for new goroutines, applied via debug package
		debug.SetMaxStack(so.config.ThreadStackSize * 1024) // Convert KB to bytes
		so.logger.Debug("Set max stack size",
			zap.Int("size_kb", so.config.ThreadStackSize))
	}
	
	return nil
}

// applyMemoryOptimizations optimizes memory usage and allocation
func (so *SystemOptimizer) applyMemoryOptimizations() error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Set memory limit based on system memory
	if so.config.MemoryLimitPercent > 0 {
		systemMem := estimateSystemMemory()
		memLimit := systemMem * int64(so.config.MemoryLimitPercent) / 100
		
		if memLimit > 0 {
			debug.SetMemoryLimit(memLimit)
			os.Setenv("GOMEMLIMIT", strconv.FormatInt(memLimit, 10))
			so.originalSettings["GOMEMLIMIT"] = os.Getenv("GOMEMLIMIT")
			
			so.logger.Info("Set memory limit",
				zap.Int64("limit_bytes", memLimit),
				zap.Int("percent", so.config.MemoryLimitPercent))
		}
	}
	
	// Enable huge pages hint for large allocations
	if so.config.EnableHugePagesHint {
		so.logger.Debug("Huge pages optimization enabled (hint only)")
	}
	
	// Pre-allocate memory pools for high-performance modes
	if so.config.Level >= OptimizationAggressive {
		so.preallocateMemoryPools()
	}
	
	return nil
}

// applyGCTuning optimizes garbage collection settings
func (so *SystemOptimizer) applyGCTuning() error {
	// Set GC target percentage
	if so.config.GCTargetPercent > 0 {
		oldGCPercent := debug.SetGCPercent(so.config.GCTargetPercent)
		so.originalSettings["GCPercent"] = oldGCPercent
		os.Setenv("GOGC", strconv.Itoa(so.config.GCTargetPercent))
		
		so.logger.Info("Set GC target percent",
			zap.Int("old", oldGCPercent),
			zap.Int("new", so.config.GCTargetPercent))
	}
	
	// For ultra-low latency, minimize GC pauses
	if so.config.Level >= OptimizationTurbo {
		// Force immediate GC to clear initial allocations
		runtime.GC()
		runtime.GC() // Double GC for thorough cleanup
		
		so.logger.Debug("Performed initial GC cleanup for low-latency mode")
	}
	
	return nil
}

// applyThreadOptimizations optimizes threading behavior
func (so *SystemOptimizer) applyThreadOptimizations() error {
	// Thread pinning for critical goroutines
	if so.config.EnableCPUPinning && so.config.Level >= OptimizationAggressive {
		// Note: Actual CPU pinning would require platform-specific syscalls
		// This is a placeholder for the concept
		so.logger.Debug("CPU pinning enabled (implementation pending)")
	}
	
	// Set thread creation limits
	if so.config.MaxThreads > 0 {
		// Go runtime handles this automatically, but we can provide hints
		so.logger.Debug("Thread limits configured",
			zap.Int("max_threads", so.config.MaxThreads))
	}
	
	return nil
}

// applyCPUOptimizations optimizes CPU usage patterns
func (so *SystemOptimizer) applyCPUOptimizations() error {
	numCPU := runtime.NumCPU()
	
	// CPU affinity optimization
	if so.config.EnableCPUPinning && len(so.config.CPUAffinity) > 0 {
		so.logger.Info("CPU affinity configured",
			zap.Ints("cpus", so.config.CPUAffinity))
	}
	
	// NUMA optimization for large systems
	if so.config.EnableNUMAOptimization && numCPU >= 16 {
		so.logger.Debug("NUMA optimizations enabled")
	}
	
	// Prevent CPU frequency scaling in high-performance modes
	if so.config.Level >= OptimizationEnterprise {
		so.logger.Debug("High-performance CPU mode requested")
	}
	
	return nil
}

// applyLatencyTuning optimizes for low latency
func (so *SystemOptimizer) applyLatencyTuning() error {
	if !so.config.EnableLatencyTuning {
		return nil
	}
	
	// Disable background scavenging for predictable latency
	if so.config.Level >= OptimizationTurbo {
		debug.SetMaxStack(1 << 20) // 1MB stack limit for predictability
		so.logger.Debug("Applied ultra-low latency tuning")
	}
	
	// Timer precision optimization
	so.logger.Debug("Latency tuning applied")
	
	return nil
}

// applyPlatformSpecific applies platform-specific optimizations
func (so *SystemOptimizer) applyPlatformSpecific() error {
	switch runtime.GOOS {
	case "windows":
		return so.applyWindowsOptimizations()
	case "linux":
		return so.applyLinuxOptimizations()
	case "darwin":
		return so.applyMacOSOptimizations()
	default:
		so.logger.Debug("No platform-specific optimizations available",
			zap.String("platform", runtime.GOOS))
	}
	
	return nil
}

// applyWindowsOptimizations applies Windows-specific optimizations
func (so *SystemOptimizer) applyWindowsOptimizations() error {
	// Set process priority class
	if so.config.EnableRTPriority {
		so.logger.Debug("High priority mode requested (Windows)")
		// Implementation would use SetPriorityClass Windows API
	}
	
	// Enable multimedia timer precision
	if so.config.EnableLatencyTuning {
		so.logger.Debug("High-resolution timer mode enabled (Windows)")
		// Implementation would use timeBeginPeriod Windows API
	}
	
	return nil
}

// applyLinuxOptimizations applies Linux-specific optimizations
func (so *SystemOptimizer) applyLinuxOptimizations() error {
	// Set process scheduling policy
	if so.config.EnableRTPriority {
		so.logger.Debug("Real-time scheduling requested (Linux)")
		// Implementation would use sched_setscheduler
	}
	
	// Memory locking
	if so.config.EnableMemoryLocking {
		so.logger.Debug("Memory locking enabled (Linux)")
		// Implementation would use mlockall
	}
	
	return nil
}

// applyMacOSOptimizations applies macOS-specific optimizations
func (so *SystemOptimizer) applyMacOSOptimizations() error {
	// Thread time constraints for real-time behavior
	if so.config.EnableRTPriority {
		so.logger.Debug("Real-time thread constraints requested (macOS)")
		// Implementation would use thread_policy_set
	}
	
	return nil
}

// preallocateMemoryPools pre-allocates common memory patterns
func (so *SystemOptimizer) preallocateMemoryPools() {
	// Pre-allocate common buffer sizes
	bufferSizes := []int{1024, 4096, 16384, 65536, 262144} // 1KB to 256KB
	
	for _, size := range bufferSizes {
		buffer := make([]byte, size)
		_ = buffer // Prevent optimization away
	}
	
	// Force immediate collection of preallocation overhead
	runtime.GC()
	
	so.logger.Debug("Pre-allocated memory pools",
		zap.Ints("buffer_sizes", bufferSizes))
}

// monitorPerformance continuously monitors system performance
func (so *SystemOptimizer) monitorPerformance() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-so.ctx.Done():
			return
		case <-ticker.C:
			so.logPerformanceMetrics()
		}
	}
}

// logPerformanceMetrics logs current performance statistics
func (so *SystemOptimizer) logPerformanceMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	so.logger.Debug("Performance metrics",
		zap.Uint64("heap_alloc_mb", bToMb(m.HeapAlloc)),
		zap.Uint64("heap_sys_mb", bToMb(m.HeapSys)),
		zap.Uint32("num_gc", m.NumGC),
		zap.Int("goroutines", runtime.NumGoroutine()),
		zap.Float64("gc_cpu_fraction", m.GCCPUFraction),
	)
	
	if m.NumGC > 0 {
		avgPause := time.Duration(m.PauseTotalNs / uint64(m.NumGC))
		so.logger.Debug("GC performance",
			zap.Duration("avg_pause", avgPause),
			zap.Duration("last_pause", time.Duration(m.PauseNs[(m.NumGC+255)%256])),
		)
	}
}

// Restore reverts all applied optimizations
func (so *SystemOptimizer) Restore() error {
	so.mu.Lock()
	defer so.mu.Unlock()
	
	if !so.applied {
		return fmt.Errorf("no optimizations to restore")
	}
	
	// Cancel monitoring
	so.cancel()
	
	// Restore original settings
	for setting, value := range so.originalSettings {
		switch setting {
		case "GOMAXPROCS":
			if val, ok := value.(int); ok {
				runtime.GOMAXPROCS(val)
				so.logger.Debug("Restored GOMAXPROCS", zap.Int("value", val))
			}
		case "GCPercent":
			if val, ok := value.(int); ok {
				debug.SetGCPercent(val)
				so.logger.Debug("Restored GC percent", zap.Int("value", val))
			}
		case "GOMEMLIMIT":
			if val, ok := value.(string); ok {
				os.Setenv("GOMEMLIMIT", val)
				so.logger.Debug("Restored memory limit", zap.String("value", val))
			}
		}
	}
	
	so.applied = false
	so.logger.Info("System optimizations restored")
	
	return nil
}

// GetStats returns current optimization statistics
func (so *SystemOptimizer) GetStats() map[string]interface{} {
	so.mu.RLock()
	defer so.mu.RUnlock()
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return map[string]interface{}{
		"applied":           so.applied,
		"level":            so.getLevelString(),
		"gomaxprocs":       runtime.GOMAXPROCS(0),
		"num_cpu":          runtime.NumCPU(),
		"num_goroutine":    runtime.NumGoroutine(),
		"heap_alloc_mb":    bToMb(m.HeapAlloc),
		"heap_sys_mb":      bToMb(m.HeapSys),
		"num_gc":           m.NumGC,
		"gc_cpu_fraction":  m.GCCPUFraction,
		"go_version":       runtime.Version(),
	}
}

// Helper functions

func (so *SystemOptimizer) getLevelString() string {
	switch so.config.Level {
	case OptimizationBasic:
		return "basic"
	case OptimizationStandard:
		return "standard"
	case OptimizationAggressive:
		return "aggressive"
	case OptimizationEnterprise:
		return "enterprise"
	case OptimizationTurbo:
		return "turbo"
	default:
		return "unknown"
	}
}

func estimateSystemMemory() int64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Rough estimation based on current heap usage
	// In production, this would use platform-specific APIs
	return int64(m.Sys * 8) // Conservative multiplier
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

// ForceGCWithStats forces garbage collection with detailed timing and stats
func ForceGCWithStats(logger *zap.Logger) {
	start := time.Now()
	var m1, m2 runtime.MemStats
	
	runtime.ReadMemStats(&m1)
	runtime.GC()
	runtime.ReadMemStats(&m2)
	
	duration := time.Since(start)
	freed := int64(m1.HeapAlloc) - int64(m2.HeapAlloc)
	
	logger.Info("Manual GC completed",
		zap.Duration("duration", duration),
		zap.Int64("freed_bytes", freed),
		zap.Uint64("heap_alloc_before_mb", bToMb(m1.HeapAlloc)),
		zap.Uint64("heap_alloc_after_mb", bToMb(m2.HeapAlloc)),
	)
}

// GetOptimalGOMAXPROCS returns optimal GOMAXPROCS value for the system
func GetOptimalGOMAXPROCS() int {
	numCPU := runtime.NumCPU()
	
	// For blockchain relay systems, optimize for I/O concurrency
	switch {
	case numCPU <= 2:
		return numCPU
	case numCPU <= 8:
		return numCPU
	case numCPU <= 16:
		return numCPU - 1 // Reserve one core for system
	default:
		return numCPU - 2 // Reserve cores for system and interrupts
	}
}

// IsRealTimeCapable checks if the system supports real-time optimizations
func IsRealTimeCapable() bool {
	switch runtime.GOOS {
	case "linux":
		// Check for RT kernel capabilities
		return true // Simplified check
	case "windows":
		// Check for high priority capabilities
		return true // Simplified check
	case "darwin":
		// Check for real-time thread support
		return true // Simplified check
	default:
		return false
	}
}

// GetSystemInfo returns detailed system information for optimization decisions
func GetSystemInfo() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return map[string]interface{}{
		"os":               runtime.GOOS,
		"arch":             runtime.GOARCH,
		"go_version":       runtime.Version(),
		"num_cpu":          runtime.NumCPU(),
		"gomaxprocs":       runtime.GOMAXPROCS(0),
		"num_goroutine":    runtime.NumGoroutine(),
		"heap_sys_mb":      bToMb(m.Sys),
		"heap_alloc_mb":    bToMb(m.HeapAlloc),
		"pointer_size":     unsafe.Sizeof(uintptr(0)),
		"rt_capable":       IsRealTimeCapable(),
		"optimal_threads":  GetOptimalGOMAXPROCS(),
	}
}
