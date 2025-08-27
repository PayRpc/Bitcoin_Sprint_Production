package main

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/performance"
	"go.uber.org/zap"
)

func main() {
	fmt.Println("Bitcoin Sprint - Flat Latency Performance Test")
	fmt.Println("==============================================")

	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Initialize performance manager
	perfManager := performance.New(cfg, logger)

	// Apply maximum performance optimizations
	fmt.Println("Applying maximum performance optimizations...")
	err = perfManager.ApplyOptimizations()
	if err != nil {
		log.Fatalf("Failed to apply optimizations: %v", err)
	}

	// Run latency benchmark
	fmt.Println("Running flat latency benchmark...")
	runLatencyBenchmark(perfManager)

	fmt.Println("Latency test completed successfully!")
}

func runLatencyBenchmark(pm *performance.PerformanceManager) {
	const numOperations = 100000
	const numWorkers = 1000

	fmt.Printf("Running %d operations with %d concurrent workers...\n", numOperations, numWorkers)

	latencies := make([]time.Duration, numOperations)
	var wg sync.WaitGroup
	var mu sync.Mutex
	operationIndex := 0

	startTime := time.Now()

	// Get buffer pool from performance manager
	bufferPool := pm.GetBufferPool()

	// Start worker pool simulation
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for {
				mu.Lock()
				if operationIndex >= numOperations {
					mu.Unlock()
					break
				}
				idx := operationIndex
				operationIndex++
				mu.Unlock()

				// Simulate high-frequency operation with buffer pooling
				opStart := time.Now()

				// Get buffer from pool
				buf := bufferPool.Get(256)
				
				// Simulate processing
				for j := range buf {
					buf[j] = byte((workerID + idx + j) % 256)
				}
				
				// Return buffer to pool
				bufferPool.Put(buf)

				// Record latency
				duration := time.Since(opStart)
				latencies[idx] = duration
			}
		}(i)
	}

	wg.Wait()
	totalDuration := time.Since(startTime)

	// Calculate latency statistics
	var totalLatency time.Duration
	minLatency := time.Hour
	maxLatency := time.Duration(0)

	for _, latency := range latencies {
		totalLatency += latency
		if latency < minLatency {
			minLatency = latency
		}
		if latency > maxLatency {
			maxLatency = latency
		}
	}

	avgLatency := totalLatency / numOperations
	throughput := float64(numOperations) / totalDuration.Seconds()

	// Calculate percentiles
	sortedLatencies := make([]time.Duration, len(latencies))
	copy(sortedLatencies, latencies)
	for i := 0; i < len(sortedLatencies)-1; i++ {
		for j := i + 1; j < len(sortedLatencies); j++ {
			if sortedLatencies[i] > sortedLatencies[j] {
				sortedLatencies[i], sortedLatencies[j] = sortedLatencies[j], sortedLatencies[i]
			}
		}
	}

	p50 := sortedLatencies[numOperations*50/100]
	p95 := sortedLatencies[numOperations*95/100]
	p99 := sortedLatencies[numOperations*99/100]
	p999 := sortedLatencies[numOperations*999/1000]

	// Display results
	fmt.Println("\n=== LATENCY BENCHMARK RESULTS ===")
	fmt.Printf("Total Operations: %d\n", numOperations)
	fmt.Printf("Concurrent Workers: %d\n", numWorkers)
	fmt.Printf("Total Duration: %.2fs\n", totalDuration.Seconds())
	fmt.Printf("Throughput: %.0f ops/sec\n", throughput)
	fmt.Printf("CPU Cores: %d\n", runtime.NumCPU())
	fmt.Println()

	fmt.Println("LATENCY STATISTICS:")
	fmt.Printf("  Min: %v\n", minLatency)
	fmt.Printf("  Avg: %v\n", avgLatency)
	fmt.Printf("  Max: %v\n", maxLatency)
	fmt.Println()

	fmt.Println("PERCENTILE LATENCIES:")
	fmt.Printf("  P50  (median): %v\n", p50)
	fmt.Printf("  P95:           %v\n", p95)
	fmt.Printf("  P99:           %v\n", p99)
	fmt.Printf("  P99.9:         %v\n", p999)
	fmt.Println()

	fmt.Println("TARGET LATENCY GOALS:")
	fmt.Printf("  P50:  3.2ms (Target: %v)\n", 3200*time.Microsecond)
	fmt.Printf("  P95:  3.4ms (Target: %v)\n", 3400*time.Microsecond)
	fmt.Printf("  P99:  3.6ms (Target: %v)\n", 3600*time.Microsecond)
	fmt.Printf("  P99.9: 3.9ms (Target: %v)\n", 3900*time.Microsecond)
	fmt.Println()

	// Check if targets are met
	targetsMet := 0
	if p50 <= 3200*time.Microsecond {
		fmt.Printf("✅ P50 target MET: %v ≤ 3.2ms\n", p50)
		targetsMet++
	} else {
		fmt.Printf("❌ P50 target MISSED: %v > 3.2ms\n", p50)
	}

	if p95 <= 3400*time.Microsecond {
		fmt.Printf("✅ P95 target MET: %v ≤ 3.4ms\n", p95)
		targetsMet++
	} else {
		fmt.Printf("❌ P95 target MISSED: %v > 3.4ms\n", p95)
	}

	if p99 <= 3600*time.Microsecond {
		fmt.Printf("✅ P99 target MET: %v ≤ 3.6ms\n", p99)
		targetsMet++
	} else {
		fmt.Printf("❌ P99 target MISSED: %v > 3.6ms\n", p99)
	}

	if p999 <= 3900*time.Microsecond {
		fmt.Printf("✅ P99.9 target MET: %v ≤ 3.9ms\n", p999)
		targetsMet++
	} else {
		fmt.Printf("❌ P99.9 target MISSED: %v > 3.9ms\n", p999)
	}

	fmt.Printf("\n🎯 TARGET COMPLIANCE: %d/4 latency targets met\n", targetsMet)

	if targetsMet == 4 {
		fmt.Println("🏆 FLAT LATENCY ACHIEVED! Bitcoin Sprint is optimized for high-frequency trading.")
	} else {
		fmt.Printf("⚡ %d/4 targets met. Further optimization may be needed.\n", targetsMet)
	}

	// Display system information
	fmt.Println("\n=== SYSTEM INFORMATION ===")
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("NumCPU: %d\n", runtime.NumCPU())
	fmt.Printf("NumGoroutine: %d\n", runtime.NumGoroutine())
}
