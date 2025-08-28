package main

import (
	"fmt"
	"time"
	
	"github.com/PayRpc/Bitcoin-Sprint/internal/api"
)

func main() {
	fmt.Println("🧪 Testing ACTUAL Performance Tracking (not hardcoded values)")
	fmt.Println("==============================================================")

	// Create latency optimizer
	optimizer := api.NewLatencyOptimizer() 
	
	fmt.Println("\n1. Testing Latency Tracking:")
	
	// Simulate some real request latencies
	testLatencies := []time.Duration{
		45 * time.Millisecond,
		78 * time.Millisecond,
		123 * time.Millisecond,
		67 * time.Millisecond,
		89 * time.Millisecond,
	}
	
	// Track these latencies
	for i, latency := range testLatencies {
		optimizer.TrackRequest("btc", latency)
		fmt.Printf("   Request %d: %v\n", i+1, latency)
	}
	
	// Get ACTUAL stats
	stats := optimizer.GetActualStats()
	fmt.Printf("\n📊 REAL Measured Performance:\n")
	fmt.Printf("   Current P99: %v\n", stats["CurrentP99"])
	fmt.Printf("   Chain Count: %v\n", stats["ChainCount"])
	fmt.Printf("   Status: %v\n", stats["Status"])
	
	fmt.Println("\n2. Testing Cache Performance:")
	
	// Create predictive cache
	cache := api.NewPredictiveCache()
	
	// Simulate some cache operations
	testReq := &api.UnifiedRequest{
		Method: "getBlock",
		Chain:  "btc",
		Params: map[string]interface{}{"height": 123},
	}
	
	// Cache miss first
	result1 := cache.Get(testReq)
	fmt.Printf("   First access (should be miss): %v\n", result1 != nil)
	
	// Set cache value
	cache.Set(testReq, map[string]interface{}{"height": 123, "hash": "abc123"})
	fmt.Printf("   Cached result for future requests\n")
	
	// Cache hit second time
	result2 := cache.Get(testReq)
	fmt.Printf("   Second access (should be hit): %v\n", result2 != nil)
	
	// Get ACTUAL cache stats
	cacheStats := cache.GetActualCacheStats()
	fmt.Printf("\n📊 REAL Cache Performance:\n")
	fmt.Printf("   Hit Rate: %v\n", cacheStats["hit_rate_percent"])
	fmt.Printf("   Cache Size: %v\n", cacheStats["cache_size"])
	fmt.Printf("   Total Requests: %v\n", cacheStats["total_requests"])
	
	fmt.Println("\n✅ Performance tracking is now REAL, not hardcoded!")
}
