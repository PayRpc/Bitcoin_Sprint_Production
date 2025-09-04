package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/Bitcoin_Sprint/internal/api"
	"github.com/Bitcoin_Sprint/internal/config"
)

func main() {
	fmt.Println("🧪 TIER ENFORCEMENT UNIT TEST")
	fmt.Println("=" + strings.Repeat("=", 40))

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create middleware instance
	middleware := api.NewMiddleware(cfg)

	// Test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Test different tiers
	tiers := []struct {
		name string
		key  string
		rate int
	}{
		{"free", "free-test-key", cfg.Tiers.Free.RateLimit},
		{"pro", "pro-test-key", cfg.Tiers.Pro.RateLimit},
		{"enterprise", "enterprise-test-key", cfg.Tiers.Enterprise.RateLimit},
	}

	for _, tier := range tiers {
		fmt.Printf("\n🔍 Testing %s tier (limit: %d req/sec)\n", tier.name, tier.rate)
		fmt.Println("-" + strings.Repeat("-", 30))

		// Create a request with the API key
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Key", tier.key)

		// Test multiple requests to trigger rate limiting
		testRequests := tier.rate * 2 // Test beyond the limit
		rateLimited := 0
		successful := 0

		start := time.Now()
		for i := 0; i < testRequests; i++ {
			w := httptest.NewRecorder()

			// Apply middleware
			middleware.AuthMiddleware(testHandler).ServeHTTP(w, req)

			if w.Code == http.StatusTooManyRequests {
				rateLimited++
			} else if w.Code == http.StatusOK {
				successful++
			}
		}
		duration := time.Since(start)

		fmt.Printf("Requests: %d, Successful: %d, Rate Limited: %d\n", testRequests, successful, rateLimited)
		fmt.Printf("Duration: %.2fs, Rate: %.1f req/sec\n", duration.Seconds(), float64(testRequests)/duration.Seconds())

		if rateLimited > 0 {
			fmt.Printf("✅ %s tier rate limiting working!\n", tier.name)
		} else {
			fmt.Printf("⚠️ No rate limiting detected for %s tier\n", tier.name)
		}
	}

	fmt.Println("\n🎯 UNIT TEST COMPLETE")
	fmt.Println("\n💡 This test validates the middleware logic directly.")
	fmt.Println("If rate limiting works here but not in production,")
	fmt.Println("the issue is likely in server startup or configuration.")
}
