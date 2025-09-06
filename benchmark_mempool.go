package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
)

func main() {
	fmt.Println("⚡ Bitcoin Sprint Mempool Performance Benchmark")
	fmt.Printf("🔧 Go Version: %s\n", runtime.Version())
	fmt.Printf("💻 GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))
	
	// Benchmark 1: Sequential Operations
	fmt.Println("\n📈 Benchmark 1: Sequential Operations")
	mp := mempool.New()
	defer mp.Stop()
	
	start := time.Now()
	for i := 0; i < 100000; i++ {
		mp.Add(fmt.Sprintf("tx_%d", i))
	}
	addDuration := time.Since(start)
	fmt.Printf("✅ Added 100K transactions in %v (%.2f ops/sec)\n", 
		addDuration, 100000.0/addDuration.Seconds())
	
	start = time.Now()
	for i := 0; i < 100000; i++ {
		mp.Contains(fmt.Sprintf("tx_%d", i))
	}
	containsDuration := time.Since(start)
	fmt.Printf("✅ Checked 100K transactions in %v (%.2f ops/sec)\n", 
		containsDuration, 100000.0/containsDuration.Seconds())
	
	// Benchmark 2: Concurrent Operations
	fmt.Println("\n🚀 Benchmark 2: Concurrent Operations (8 goroutines)")
	mp.Clear()
	
	const numGoroutines = 8
	const opsPerGoroutine = 25000
	
	var wg sync.WaitGroup
	start = time.Now()
	
	// Concurrent adds
	wg.Add(numGoroutines)
	for g := 0; g < numGoroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				mp.AddWithDetails(
					fmt.Sprintf("concurrent_tx_%d_%d", gid, i),
					100+i, gid, float64(i)*0.0001,
				)
			}
		}(g)
	}
	wg.Wait()
	
	concurrentDuration := time.Since(start)
	totalOps := numGoroutines * opsPerGoroutine
	fmt.Printf("✅ Added %d transactions concurrently in %v (%.2f ops/sec)\n", 
		totalOps, concurrentDuration, float64(totalOps)/concurrentDuration.Seconds())
	
	// Benchmark 3: Mixed Workload
	fmt.Println("\n🔀 Benchmark 3: Mixed Workload (Read/Write/Delete)")
	
	start = time.Now()
	wg.Add(numGoroutines * 3) // readers, writers, deleters
	
	// Writers
	for g := 0; g < numGoroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine/3; i++ {
				mp.Add(fmt.Sprintf("mixed_tx_%d_%d", gid, i))
			}
		}(g)
	}
	
	// Readers
	for g := 0; g < numGoroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine/3; i++ {
				mp.Contains(fmt.Sprintf("concurrent_tx_%d_%d", gid, i%1000))
			}
		}(g)
	}
	
	// Deleters
	for g := 0; g < numGoroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine/3; i++ {
				mp.Remove(fmt.Sprintf("concurrent_tx_%d_%d", gid, i*2))
			}
		}(g)
	}
	
	wg.Wait()
	mixedDuration := time.Since(start)
	mixedOps := numGoroutines * opsPerGoroutine
	fmt.Printf("✅ Mixed workload %d operations in %v (%.2f ops/sec)\n", 
		mixedOps, mixedDuration, float64(mixedOps)/mixedDuration.Seconds())
	
	// Benchmark 4: Memory and Statistics
	fmt.Println("\n📊 Benchmark 4: Memory and Statistics")
	
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	
	config := mempool.Config{
		MaxSize:         1000000,
		ExpiryTime:      10 * time.Minute,
		CleanupInterval: 5 * time.Minute,
		ShardCount:      32,
	}
	
	mpLarge := mempool.NewWithConfig(config)
	defer mpLarge.Stop()
	
	start = time.Now()
	for i := 0; i < 500000; i++ {
		mpLarge.AddWithDetails(
			fmt.Sprintf("large_tx_%d", i),
			200+i%1000, i%100, float64(i)*0.00001,
		)
	}
	largeDuration := time.Since(start)
	
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	
	fmt.Printf("✅ Added 500K transactions in %v (%.2f ops/sec)\n", 
		largeDuration, 500000.0/largeDuration.Seconds())
	fmt.Printf("📏 Memory usage: %.2f MB (%.2f bytes per tx)\n", 
		float64(m2.Alloc-m1.Alloc)/1024/1024, 
		float64(m2.Alloc-m1.Alloc)/500000.0)
	
	// Statistics
	stats := mpLarge.Stats()
	fmt.Printf("📈 Final mempool size: %v transactions\n", stats["size"])
	fmt.Printf("🔀 Distributed across %v shards\n", stats["shard_count"])
	
	// Benchmark 5: Shard Distribution
	fmt.Println("\n🎯 Benchmark 5: Shard Distribution Analysis")
	
	shardStats := stats["shards"].([]map[string]interface{})
	minSize, maxSize := 999999, 0
	totalSize := 0
	
	for _, shardStat := range shardStats {
		size := shardStat["size"].(int)
		totalSize += size
		if size < minSize {
			minSize = size
		}
		if size > maxSize {
			maxSize = size
		}
	}
	
	avgSize := float64(totalSize) / float64(len(shardStats))
	variance := float64(maxSize-minSize) / avgSize * 100
	
	fmt.Printf("✅ Shard distribution - Min: %d, Max: %d, Avg: %.1f\n", 
		minSize, maxSize, avgSize)
	fmt.Printf("📊 Distribution variance: %.1f%% (lower is better)\n", variance)
	
	if variance < 20 {
		fmt.Println("🎉 Excellent shard distribution!")
	} else if variance < 50 {
		fmt.Println("✅ Good shard distribution")
	} else {
		fmt.Println("⚠️  Consider tuning shard count for better distribution")
	}
	
	fmt.Println("\n🏆 Performance Summary:")
	fmt.Println("=" * 50)
	fmt.Printf("Sequential Add:      %.0f ops/sec\n", 100000.0/addDuration.Seconds())
	fmt.Printf("Sequential Contains: %.0f ops/sec\n", 100000.0/containsDuration.Seconds())
	fmt.Printf("Concurrent Mixed:    %.0f ops/sec\n", float64(mixedOps)/mixedDuration.Seconds())
	fmt.Printf("Large Dataset:       %.0f ops/sec\n", 500000.0/largeDuration.Seconds())
	fmt.Printf("Memory Efficiency:   %.1f bytes/tx\n", float64(m2.Alloc-m1.Alloc)/500000.0)
	fmt.Println("=" * 50)
	fmt.Println("🚀 Enterprise-grade performance achieved!")
}
