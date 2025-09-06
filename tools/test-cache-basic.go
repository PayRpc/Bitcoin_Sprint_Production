package main

import (
	"fmt"
	"log"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/cache"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	fmt.Println("🚀 Testing Enterprise Cache Basic Functionality")

	// Create cache with default configuration
	cache := cache.New(1000, logger)
	if cache == nil {
		fmt.Println("❌ Failed to create cache")
		return
	}

	fmt.Println("✅ Cache created successfully")

	// Test basic operations
	testBlock := blocks.BlockEvent{
		Height:      1,
		Hash:        "test_hash",
		Chain:       blocks.ChainBitcoin,
		Timestamp:   time.Now(),
		DetectedAt:  time.Now(),
		Source:      "test",
		Tier:        "ENTERPRISE",
		RelayTimeMs: 50,
		Status:      blocks.StatusProcessed,
	}

	// Set latest block
	err = cache.SetLatestBlock(testBlock)
	if err != nil {
		fmt.Printf("❌ Failed to set latest block: %v\n", err)
		return
	}

	fmt.Println("✅ Latest block set successfully")

	// Get latest block
	retrievedBlock, found := cache.GetLatestBlock()
	if !found {
		fmt.Println("❌ Failed to retrieve latest block")
		return
	}

	if retrievedBlock.Height != testBlock.Height {
		fmt.Printf("❌ Block height mismatch: expected %d, got %d\n",
			testBlock.Height, retrievedBlock.Height)
		return
	}

	fmt.Println("✅ Latest block retrieved successfully")

	// Test generic cache operations
	err = cache.Set("test_key", "test_value", time.Minute)
	if err != nil {
		fmt.Printf("❌ Failed to set cache entry: %v\n", err)
		return
	}

	fmt.Println("✅ Cache entry set successfully")

	value, found := cache.Get("test_key")
	if !found {
		fmt.Println("❌ Failed to retrieve cache entry")
		return
	}

	if value != "test_value" {
		fmt.Printf("❌ Cache value mismatch: expected 'test_value', got %v\n", value)
		return
	}

	fmt.Println("✅ Cache entry retrieved successfully")

	// Get cache metrics
	metrics := cache.GetMetrics()
	if metrics != nil {
		fmt.Printf("📊 Cache Metrics:\n")
		fmt.Printf("   Total Requests: %d\n", metrics.TotalRequests)
		fmt.Printf("   Cache Hits: %d\n", metrics.CacheHits)
		fmt.Printf("   Cache Misses: %d\n", metrics.CacheMisses)
		fmt.Printf("   Hit Rate: %.2f%%\n", metrics.HitRate*100)
		fmt.Printf("   Memory Usage: %d bytes\n", metrics.MemoryUsage)
		fmt.Printf("   Health Score: %.2f\n", metrics.HealthScore)
	}

	fmt.Println("\n🎉 Enterprise Cache basic test completed successfully!")
}
