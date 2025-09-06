package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"go.uber.org/zap"
	runtimeopt "../../internal/runtime"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("Failed to create logger: %v", err))
	}
	defer logger.Sync()

	fmt.Println("🚀 Bitcoin Sprint Enterprise Runtime Optimization Demo")
	fmt.Println("=====================================================")

	// Display system information
	displaySystemInfo(logger)

	// Demonstrate different optimization levels
	demonstrateOptimizationLevels(logger)

	// Run performance benchmarks
	runPerformanceBenchmarks(logger)

	// Interactive optimization testing
	runInteractiveDemo(logger)
}

func displaySystemInfo(logger *zap.Logger) {
	fmt.Println("\n📊 System Information:")
	fmt.Println("----------------------")

	sysInfo := runtimeopt.GetSystemInfo()
	
	fmt.Printf("OS/Architecture: %s/%s\n", sysInfo["os"], sysInfo["arch"])
	fmt.Printf("Go Version: %s\n", sysInfo["go_version"])
	fmt.Printf("CPU Cores: %d\n", sysInfo["num_cpu"])
	fmt.Printf("Current GOMAXPROCS: %d\n", sysInfo["gomaxprocs"])
	fmt.Printf("Optimal GOMAXPROCS: %d\n", sysInfo["optimal_threads"])
	fmt.Printf("Real-Time Capable: %t\n", sysInfo["rt_capable"])
	fmt.Printf("Current Heap (MB): %d\n", sysInfo["heap_alloc_mb"])
	fmt.Printf("Pointer Size: %d bytes\n", sysInfo["pointer_size"])
	fmt.Printf("Active Goroutines: %d\n", sysInfo["num_goroutine"])
}

func demonstrateOptimizationLevels(logger *zap.Logger) {
	fmt.Println("\n🔧 Optimization Level Comparison:")
	fmt.Println("----------------------------------")

	levels := []struct {
		name   string
		config *runtimeopt.SystemOptimizationConfig
	}{
		{"Basic (Development)", runtimeopt.DefaultConfig()},
		{"Enterprise (Production)", runtimeopt.EnterpriseConfig()},
		{"Turbo (Ultra-Low Latency)", runtimeopt.TurboConfig()},
	}

	for _, level := range levels {
		fmt.Printf("\n%s:\n", level.name)
		fmt.Printf("  - CPU Pinning: %t\n", level.config.EnableCPUPinning)
		fmt.Printf("  - Memory Locking: %t\n", level.config.EnableMemoryLocking)
		fmt.Printf("  - RT Priority: %t\n", level.config.EnableRTPriority)
		fmt.Printf("  - GC Target: %d%%\n", level.config.GCTargetPercent)
		fmt.Printf("  - Memory Limit: %d%%\n", level.config.MemoryLimitPercent)
		fmt.Printf("  - Thread Stack: %d KB\n", level.config.ThreadStackSize)
		fmt.Printf("  - NUMA Optimization: %t\n", level.config.EnableNUMAOptimization)
		fmt.Printf("  - Latency Tuning: %t\n", level.config.EnableLatencyTuning)
	}
}

func runPerformanceBenchmarks(logger *zap.Logger) {
	fmt.Println("\n⚡ Performance Benchmarks:")
	fmt.Println("--------------------------")

	// Benchmark without optimizations
	fmt.Println("\n1. Baseline (No Optimizations):")
	baselineStats := benchmarkAllocation(1000000, logger)
	displayBenchmarkResults(baselineStats)

	// Benchmark with standard optimizations
	fmt.Println("\n2. Standard Optimizations:")
	optimizer := runtimeopt.NewSystemOptimizer(runtimeopt.DefaultConfig(), logger)
	if err := optimizer.Apply(); err != nil {
		logger.Error("Failed to apply standard optimizations", zap.Error(err))
	} else {
		standardStats := benchmarkAllocation(1000000, logger)
		displayBenchmarkResults(standardStats)
		
		fmt.Printf("Improvement: %.2fx faster\n", 
			float64(baselineStats.Duration)/float64(standardStats.Duration))
	}
	optimizer.Restore()

	// Benchmark with enterprise optimizations  
	fmt.Println("\n3. Enterprise Optimizations:")
	enterpriseOptimizer := runtimeopt.NewSystemOptimizer(runtimeopt.EnterpriseConfig(), logger)
	if err := enterpriseOptimizer.Apply(); err != nil {
		logger.Warn("Failed to apply enterprise optimizations (may need admin privileges)", zap.Error(err))
	} else {
		enterpriseStats := benchmarkAllocation(1000000, logger)
		displayBenchmarkResults(enterpriseStats)
		
		fmt.Printf("Improvement: %.2fx faster than baseline\n", 
			float64(baselineStats.Duration)/float64(enterpriseStats.Duration))
	}
	enterpriseOptimizer.Restore()
}

type BenchmarkStats struct {
	Duration      time.Duration
	Allocations   uint64
	HeapBefore    uint64
	HeapAfter     uint64
	GCCount       uint32
	GCPauseTotalBefore uint64
	GCPauseTotalAfter  uint64
}

func benchmarkAllocation(count int, logger *zap.Logger) BenchmarkStats {
	var m1, m2 runtime.MemStats
	
	// Force GC and read initial stats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	start := time.Now()
	
	// Perform allocations to simulate blockchain processing
	data := make([][]byte, count)
	for i := 0; i < count; i++ {
		// Simulate transaction data (varying sizes)
		size := 64 + (i % 512) // 64-576 bytes
		data[i] = make([]byte, size)
		
		// Simulate some processing
		for j := 0; j < len(data[i]); j++ {
			data[i][j] = byte(j ^ i)
		}
		
		// Occasionally trigger cleanup simulation
		if i%10000 == 0 {
			data[i/2] = nil // Free some memory
		}
	}
	
	duration := time.Since(start)
	
	// Force GC and read final stats
	runtime.GC()
	runtime.ReadMemStats(&m2)
	
	// Keep reference to prevent optimization
	_ = data[count-1]
	
	return BenchmarkStats{
		Duration:      duration,
		Allocations:   m2.Mallocs - m1.Mallocs,
		HeapBefore:    m1.HeapAlloc,
		HeapAfter:     m2.HeapAlloc,
		GCCount:       m2.NumGC - m1.NumGC,
		GCPauseTotalBefore: m1.PauseTotalNs,
		GCPauseTotalAfter:  m2.PauseTotalNs,
	}
}

func displayBenchmarkResults(stats BenchmarkStats) {
	fmt.Printf("  Duration: %v\n", stats.Duration)
	fmt.Printf("  Allocations: %d\n", stats.Allocations)
	fmt.Printf("  Heap Before: %.2f MB\n", float64(stats.HeapBefore)/1024/1024)
	fmt.Printf("  Heap After: %.2f MB\n", float64(stats.HeapAfter)/1024/1024)
	fmt.Printf("  GC Cycles: %d\n", stats.GCCount)
	
	if stats.GCCount > 0 {
		totalPause := stats.GCPauseTotalAfter - stats.GCPauseTotalBefore
		avgPause := time.Duration(totalPause / uint64(stats.GCCount))
		fmt.Printf("  Avg GC Pause: %v\n", avgPause)
	}
	
	fmt.Printf("  Throughput: %.0f ops/sec\n", 
		float64(stats.Allocations)/stats.Duration.Seconds())
}

func runInteractiveDemo(logger *zap.Logger) {
	fmt.Println("\n🎮 Interactive Optimization Demo:")
	fmt.Println("----------------------------------")
	fmt.Println("Running enterprise optimization with live monitoring...")
	fmt.Println("Press Ctrl+C to stop")

	// Apply enterprise optimizations
	optimizer := runtimeopt.NewSystemOptimizer(runtimeopt.EnterpriseConfig(), logger)
	if err := optimizer.Apply(); err != nil {
		logger.Error("Failed to apply optimizations", zap.Error(err))
		return
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\n🛑 Shutdown signal received, cleaning up...")
		cancel()
	}()

	// Start monitoring
	go monitorPerformance(ctx, optimizer, logger)

	// Simulate blockchain workload
	simulateBlockchainWorkload(ctx, logger)

	// Cleanup
	fmt.Println("Restoring system settings...")
	if err := optimizer.Restore(); err != nil {
		logger.Error("Failed to restore settings", zap.Error(err))
	} else {
		fmt.Println("✅ System settings restored successfully")
	}
}

func monitorPerformance(ctx context.Context, optimizer *runtimeopt.SystemOptimizer, logger *zap.Logger) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := optimizer.GetStats()
			
			fmt.Printf("\r📊 Live Stats: Goroutines: %d | Heap: %d MB | GC: %.2f%% | Applied: %t        ",
				stats["num_goroutine"].(int),
				stats["heap_alloc_mb"].(uint64),
				stats["gc_cpu_fraction"].(float64)*100,
				stats["applied"].(bool))
		}
	}
}

func simulateBlockchainWorkload(ctx context.Context, logger *zap.Logger) {
	// Simulate varying blockchain processing workload
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	transactionCount := 0
	
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("\n\n📈 Processed %d simulated transactions\n", transactionCount)
			return
		case <-ticker.C:
			// Simulate transaction processing
			go func() {
				// Simulate transaction validation
				data := make([]byte, 256+transactionCount%512)
				for i := range data {
					data[i] = byte(i ^ transactionCount)
				}
				
				// Simulate some CPU work
				hash := 0
				for _, b := range data {
					hash = hash*31 + int(b)
				}
				
				// Simulate cleanup
				data = nil
				
				transactionCount++
			}()
		}
	}
}
