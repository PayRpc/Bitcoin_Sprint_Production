// Simple Multi-Chain Test Server
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func main() {
	fmt.Println("🚀 Multi-Chain Sprint Test Server")
	fmt.Println("Testing updated infrastructure...")
	fmt.Println("")

	// Test endpoints
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/api/v1/sprint/value", handleSprintValue)
	http.HandleFunc("/api/v1/universal/bitcoin/latest", handleBitcoinLatest)

	fmt.Println("🌐 Test endpoints:")
	fmt.Println("   Health:         http://localhost:8080/health")
	fmt.Println("   Sprint Value:   http://localhost:8080/api/v1/sprint/value")
	fmt.Println("   Bitcoin Latest: http://localhost:8080/api/v1/universal/bitcoin/latest")
	fmt.Println("")
	fmt.Println("Server starting on port 8080...")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"platform":  "Multi-Chain Sprint",
		"version":   "2.1.0",
		"chains":    []string{"bitcoin", "ethereum", "solana", "cosmos", "polkadot"},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func handleSprintValue(w http.ResponseWriter, r *http.Request) {
	value := map[string]interface{}{
		"platform":    "Multi-Chain Sprint",
		"description": "Enterprise blockchain infrastructure",
		"chains":      []string{"bitcoin", "ethereum", "solana", "cosmos", "polkadot"},
		"advantages": map[string]string{
			"latency":     "Flat P99 <89ms vs Infura 250-2000ms",
			"cost":        "50% cheaper than Alchemy",
			"unified_api": "Single API for all chains",
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(value)
}

func handleBitcoinLatest(w http.ResponseWriter, r *http.Request) {
	latest := map[string]interface{}{
		"chain":     "bitcoin",
		"endpoint":  "/api/v1/universal/bitcoin/latest",
		"mock_mode": true,
		"block": map[string]interface{}{
			"height":    860001,
			"hash":      "000000000000000000123456789abcdef",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
		"status": "active",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(latest)
}
