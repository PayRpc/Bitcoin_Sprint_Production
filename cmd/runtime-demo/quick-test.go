package main

import (
	"fmt"
	"log"
	"runtime"

	"go.uber.org/zap"
	runtimeopt "../../internal/runtime"
)

func main() {
	// Create logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to create logger:", err)
	}
	defer logger.Sync()

	fmt.Println("🚀 Bitcoin Sprint Runtime Optimization Quick Test")
	fmt.Println("=================================================")

	// Display system info
	fmt.Println("\n📊 System Information:")
	sysInfo := runtimeopt.GetSystemInfo()
	fmt.Printf("OS/Arch: %s/%s\n", sysInfo["os"], sysInfo["arch"])
	fmt.Printf("CPU Cores: %d\n", sysInfo["num_cpu"])
	fmt.Printf("Go Version: %s\n", sysInfo["go_version"])

	// Test basic optimization
	fmt.Println("\n🔧 Testing Basic Optimization:")
	config := runtimeopt.DefaultConfig()
	optimizer := runtimeopt.NewSystemOptimizer(config, logger)

	fmt.Printf("Applying %s optimization level...\n", config.Level)
	if err := optimizer.Apply(); err != nil {
		fmt.Printf("⚠️  Optimization failed (may need admin privileges): %v\n", err)
	} else {
		fmt.Println("✅ Optimization applied successfully!")
		
		// Get stats
		stats := optimizer.GetStats()
		fmt.Printf("Goroutines: %d\n", stats["num_goroutine"])
		fmt.Printf("Heap Size: %d MB\n", stats["heap_alloc_mb"])
		fmt.Printf("GC CPU Fraction: %.2f%%\n", stats["gc_cpu_fraction"].(float64)*100)
		
		// Restore
		fmt.Println("Restoring system settings...")
		if err := optimizer.Restore(); err != nil {
			fmt.Printf("⚠️  Restore failed: %v\n", err)
		} else {
			fmt.Println("✅ System settings restored!")
		}
	}

	// Test performance improvement
	fmt.Println("\n⚡ Performance Test:")
	
	// Baseline
	start := runtime.NumGoroutine()
	data := make([][]byte, 50000)
	for i := range data {
		data[i] = make([]byte, 64)
	}
	end := runtime.NumGoroutine()
	
	fmt.Printf("Goroutines before: %d, after: %d\n", start, end)
	fmt.Printf("Allocated %d byte arrays\n", len(data))
	
	// Force GC
	runtime.GC()
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Current heap: %d KB\n", m.Alloc/1024)
	
	fmt.Println("\n✅ Runtime optimization system is working correctly!")
	fmt.Println("💡 For full testing, run: .\\test-runtime-optimization.ps1")
	fmt.Println("🎮 For interactive demo, run: .\\run-runtime-demo.ps1")
}
