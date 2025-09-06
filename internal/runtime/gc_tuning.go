package runtime

import (
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// GCConfig holds garbage collection tuning parameters
type GCConfig struct {
	MemoryLimitPercent int           // Target memory limit as percentage of system memory
	TargetPause        time.Duration // Target GC pause time
	GCPercent          int           // GOGC value for GC frequency
}

// DefaultGCConfig returns optimized GC settings for Bitcoin Sprint
func DefaultGCConfig() *GCConfig {
	return &GCConfig{
		MemoryLimitPercent: 70,                     // Use 70% of available memory
		TargetPause:        125 * time.Microsecond, // Target 100-150µs pauses
		GCPercent:          50,                     // More frequent GC for lower latency
	}
}

// InitializeGCTuning applies optimized garbage collection settings
func InitializeGCTuning(logger *zap.Logger) error {
	config := DefaultGCConfig()

	// Set GOMEMLIMIT if not already set
	if os.Getenv("GOMEMLIMIT") == "" {
		memLimit := getMemoryLimit(config.MemoryLimitPercent)
		if memLimit > 0 {
			os.Setenv("GOMEMLIMIT", strconv.FormatInt(memLimit, 10))
			logger.Info("Set GOMEMLIMIT", zap.Int64("bytes", memLimit), zap.Int("percent", config.MemoryLimitPercent))
		}
	}

	// Set GOGC for frequency tuning
	if os.Getenv("GOGC") == "" {
		os.Setenv("GOGC", strconv.Itoa(config.GCPercent))
		logger.Info("Set GOGC", zap.Int("percent", config.GCPercent))
	}

	// Configure soft memory limit
	if memLimit := getMemoryLimit(config.MemoryLimitPercent); memLimit > 0 {
		debug.SetMemoryLimit(memLimit)
		logger.Info("Set soft memory limit", zap.Int64("bytes", memLimit))
	}

	// Set GC percent
	oldGCPercent := debug.SetGCPercent(config.GCPercent)
	logger.Info("Configured GC percent", zap.Int("old", oldGCPercent), zap.Int("new", config.GCPercent))

	// Log current runtime settings
	logGCStats(logger)

	return nil
}

// getMemoryLimit calculates memory limit based on system memory and percentage
func getMemoryLimit(percent int) int64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Estimate total system memory from current heap
	// This is a rough approximation - in production you'd use system calls
	estimatedTotal := int64(m.Sys * 4) // Rough multiplier

	return estimatedTotal * int64(percent) / 100
}

// logGCStats logs current GC statistics for monitoring
func logGCStats(logger *zap.Logger) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	logger.Info("GC Statistics",
		zap.Uint64("heap_alloc_kb", bToKb(m.HeapAlloc)),
		zap.Uint64("heap_sys_kb", bToKb(m.HeapSys)),
		zap.Uint64("heap_idle_kb", bToKb(m.HeapIdle)),
		zap.Uint64("heap_inuse_kb", bToKb(m.HeapInuse)),
		zap.Uint32("num_gc", m.NumGC),
		zap.Uint64("total_pause_ns", m.PauseTotalNs),
		zap.Float64("gc_cpu_fraction", m.GCCPUFraction),
	)

	if m.NumGC > 0 {
		avgPause := time.Duration(m.PauseTotalNs / uint64(m.NumGC))
		logger.Info("GC Timing",
			zap.Duration("avg_pause", avgPause),
			zap.Duration("last_pause", time.Duration(m.PauseNs[(m.NumGC+255)%256])),
		)
	}
}

// MonitorGCPerformance starts a goroutine to periodically log GC stats
func MonitorGCPerformance(logger *zap.Logger, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			logGCStats(logger)
		}
	}()
}

// bToKb converts bytes to kilobytes
func bToKb(b uint64) uint64 {
	return b / 1024
}

// TriggerGC forces a garbage collection cycle (for testing/debugging)
func TriggerGC(logger *zap.Logger) {
	start := time.Now()
	runtime.GC()
	duration := time.Since(start)

	logger.Info("Manual GC triggered", zap.Duration("duration", duration))
	logGCStats(logger)
}
