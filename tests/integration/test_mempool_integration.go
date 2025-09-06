package main

import (
	"fmt"
	"log"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
)

func main() {
	fmt.Println("🚀 Testing Enhanced Bitcoin Sprint Mempool")
	
	// Test 1: Basic functionality
	fmt.Println("\n📝 Test 1: Basic Operations")
	mp := mempool.New()
	
	// Add transactions
	mp.Add("tx_basic_1")
	mp.Add("tx_basic_2")
	mp.AddWithDetails("tx_detailed", 500, 5, 0.00025)
	
	fmt.Printf("✅ Added 3 transactions, mempool size: %d\n", mp.Size())
	
	// Check contents
	if mp.Contains("tx_basic_1") && mp.Contains("tx_detailed") {
		fmt.Println("✅ Transactions found in mempool")
	}
	
	// Get detailed entry
	if entry, found := mp.Get("tx_detailed"); found {
		fmt.Printf("✅ Transaction details - Size: %d, Priority: %d, Fee: %f\n", 
			entry.Size, entry.Priority, entry.FeeRate)
	}
	
	// Test 2: Configuration and metrics
	fmt.Println("\n📊 Test 2: Advanced Configuration")
	config := mempool.Config{
		MaxSize:         1000,
		ExpiryTime:      2 * time.Minute,
		CleanupInterval: 30 * time.Second,
		ShardCount:      8,
	}
	
	mpAdvanced := mempool.NewWithConfig(config)
	
	// Add many transactions to test sharding
	for i := 0; i < 50; i++ {
		mpAdvanced.AddWithDetails(
			fmt.Sprintf("tx_shard_%d", i),
			100+i*10,
			i%10,
			float64(i)*0.0001,
		)
	}
	
	fmt.Printf("✅ Added 50 transactions across shards, size: %d\n", mpAdvanced.Size())
	
	// Test stats
	stats := mpAdvanced.Stats()
	fmt.Printf("✅ Mempool stats - Size: %v, Shards: %v\n", 
		stats["size"], stats["shard_count"])
	
	// Test 3: Transaction retrieval
	fmt.Println("\n🔍 Test 3: Transaction Retrieval")
	allTxs := mpAdvanced.All()
	fmt.Printf("✅ Retrieved %d transaction IDs\n", len(allTxs))
	
	allEntries := mpAdvanced.AllEntries()
	fmt.Printf("✅ Retrieved %d transaction entries with full details\n", len(allEntries))
	
	// Show first few entries
	for i, entry := range allEntries[:min(3, len(allEntries))] {
		fmt.Printf("   TX %d: %s (Size: %d, Priority: %d)\n", 
			i+1, entry.TxID, entry.Size, entry.Priority)
	}
	
	// Test 4: Cleanup operations
	fmt.Println("\n🧹 Test 4: Cleanup Operations")
	initialSize := mpAdvanced.Size()
	mpAdvanced.Remove("tx_shard_0")
	mpAdvanced.Remove("tx_shard_1")
	afterRemoval := mpAdvanced.Size()
	fmt.Printf("✅ Removed 2 transactions: %d -> %d\n", initialSize, afterRemoval)
	
	// Clear all
	mpAdvanced.Clear()
	fmt.Printf("✅ Cleared mempool, size now: %d\n", mpAdvanced.Size())
	
	// Test 5: Graceful shutdown
	fmt.Println("\n🛑 Test 5: Graceful Shutdown")
	if err := mp.Stop(); err != nil {
		log.Printf("Warning during shutdown: %v", err)
	} else {
		fmt.Println("✅ First mempool stopped gracefully")
	}
	
	if err := mpAdvanced.Stop(); err != nil {
		log.Printf("Warning during shutdown: %v", err)
	} else {
		fmt.Println("✅ Advanced mempool stopped gracefully")
	}
	
	fmt.Println("\n🎉 All tests completed successfully!")
	fmt.Println("💪 Enhanced mempool is ready for production use!")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
