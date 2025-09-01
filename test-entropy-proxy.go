package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

// TestEntropyProxyIntegration tests the Go-to-Rust entropy proxy functionality
func TestEntropyProxyIntegration(t *testing.T) {
	// Test configuration
	baseURL := "http://localhost:8080" // Go server port
	rustURL := "http://localhost:8443" // Rust server port

	// Test data
	testCases := []struct {
		name         string
		endpoint     string
		size         int
		expectStatus int
	}{
		{"Fast Entropy 32 bytes", "/api/v1/enterprise/entropy/fast", 32, 200},
		{"Fast Entropy 64 bytes", "/api/v1/enterprise/entropy/fast", 64, 200},
		{"Hybrid Entropy 32 bytes", "/api/v1/enterprise/entropy/hybrid", 32, 200},
		{"Hybrid Entropy 64 bytes", "/api/v1/enterprise/entropy/hybrid", 64, 200},
	}

	fmt.Println("🧪 Testing Entropy Proxy Integration")
	fmt.Println("====================================")

	// Test each endpoint
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fmt.Printf("\n🔍 Testing: %s\n", tc.name)

			// Prepare request payload
			payload := map[string]int{"size": tc.size}
			jsonData, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// Test Go server proxy
			fmt.Printf("   📡 Testing Go proxy: %s%s\n", baseURL, tc.endpoint)
			resp, err := http.Post(baseURL+tc.endpoint, "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				t.Logf("⚠️  Go server not running (expected for integration test): %v", err)
				return
			}
			defer resp.Body.Close()

			// Check status code
			if resp.StatusCode != tc.expectStatus {
				t.Errorf("Expected status %d, got %d", tc.expectStatus, resp.StatusCode)
			} else {
				fmt.Printf("   ✅ Go proxy status: %d\n", resp.StatusCode)
			}

			// Read response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response: %v", err)
			}

			// Parse JSON response
			var response map[string]interface{}
			if err := json.Unmarshal(body, &response); err != nil {
				t.Fatalf("Failed to parse JSON response: %v", err)
			}

			// Validate response structure
			if entropy, ok := response["entropy"]; ok {
				fmt.Printf("   🔐 Entropy generated: %s... (length: %d)\n", entropy.(string)[:16], len(entropy.(string))/2)
			}

			if size, ok := response["size"]; ok {
				if int(size.(float64)) != tc.size {
					t.Errorf("Expected size %d, got %d", tc.size, int(size.(float64)))
				}
			}

			if source, ok := response["source"]; ok {
				fmt.Printf("   🎯 Source: %s\n", source)
			}

			if timestamp, ok := response["timestamp"]; ok {
				fmt.Printf("   🕒 Timestamp: %s\n", timestamp)
			}
		})
	}

	// Test Rust server directly (if running)
	fmt.Println("\n🔍 Testing Rust server directly:")
	testRustDirect(t, rustURL)
}

// testRustDirect tests the Rust server directly
func testRustDirect(t *testing.T, baseURL string) {
	testCases := []struct {
		name     string
		endpoint string
		size     int
	}{
		{"Direct Fast Entropy", "/entropy/fast", 32},
		{"Direct Hybrid Entropy", "/entropy/hybrid", 32},
	}

	for _, tc := range testCases {
		fmt.Printf("   📡 Testing Rust direct: %s%s\n", baseURL, tc.endpoint)

		payload := map[string]int{"size": tc.size}
		jsonData, _ := json.Marshal(payload)

		resp, err := http.Post(baseURL+tc.endpoint, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("   ⚠️  Rust server not running: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		fmt.Printf("   ✅ Rust direct status: %d\n", resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var response map[string]interface{}
		json.Unmarshal(body, &response)

		if entropy, ok := response["entropy"]; ok {
			fmt.Printf("   🔐 Direct entropy: %s...\n", entropy.(string)[:16])
		}
	}
}

// TestEntropyPerformance tests performance of entropy generation
func TestEntropyPerformance(t *testing.T) {
	baseURL := "http://localhost:8080"
	endpoint := "/api/v1/enterprise/entropy/fast"

	fmt.Println("\n⚡ Testing Entropy Performance")
	fmt.Println("==============================")

	// Test multiple requests
	numRequests := 10
	totalTime := time.Duration(0)

	for i := 0; i < numRequests; i++ {
		payload := map[string]int{"size": 32}
		jsonData, _ := json.Marshal(payload)

		start := time.Now()
		resp, err := http.Post(baseURL+endpoint, "application/json", bytes.NewBuffer(jsonData))
		duration := time.Since(start)
		totalTime += duration

		if err != nil {
			t.Logf("Request %d failed: %v", i+1, err)
			continue
		}
		resp.Body.Close()

		fmt.Printf("   Request %d: %v (status: %d)\n", i+1, duration, resp.StatusCode)
	}

	avgTime := totalTime / time.Duration(numRequests)
	fmt.Printf("   📊 Average response time: %v\n", avgTime)
	fmt.Printf("   🚀 Requests per second: %.1f\n", float64(numRequests)/totalTime.Seconds())
}

// TestEntropyVariability tests that entropy is actually random
func TestEntropyVariability(t *testing.T) {
	baseURL := "http://localhost:8080"
	endpoint := "/api/v1/enterprise/entropy/fast"

	fmt.Println("\n🎲 Testing Entropy Variability")
	fmt.Println("===============================")

	// Generate multiple entropy values
	var entropies []string
	for i := 0; i < 5; i++ {
		payload := map[string]int{"size": 32}
		jsonData, _ := json.Marshal(payload)

		resp, err := http.Post(baseURL+endpoint, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			t.Logf("Request failed: %v", err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var response map[string]interface{}
		json.Unmarshal(body, &response)

		if entropy, ok := response["entropy"]; ok {
			entropyStr := entropy.(string)
			entropies = append(entropies, entropyStr)
			fmt.Printf("   Sample %d: %s...\n", i+1, entropyStr[:16])
		}
	}

	// Check for duplicates (should be very unlikely)
	seen := make(map[string]bool)
	for _, entropy := range entropies {
		if seen[entropy] {
			t.Errorf("Duplicate entropy detected: %s", entropy[:16])
		}
		seen[entropy] = true
	}

	if len(seen) == len(entropies) {
		fmt.Printf("   ✅ All %d entropy samples are unique\n", len(entropies))
	}
}

// BenchmarkEntropyGeneration benchmarks entropy generation performance
func BenchmarkEntropyGeneration(b *testing.B) {
	baseURL := "http://localhost:8080"
	endpoint := "/api/v1/enterprise/entropy/fast"
	payload := map[string]int{"size": 32}
	jsonData, _ := json.Marshal(payload)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := http.Post(baseURL+endpoint, "application/json", bytes.NewBuffer(jsonData))
		if err == nil {
			resp.Body.Close()
		}
	}
}
