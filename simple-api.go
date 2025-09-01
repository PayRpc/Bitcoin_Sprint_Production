package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"
)

var startTime = time.Now()
var requestCount = 0

type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
	Uptime    int64  `json:"uptime"`
}

type MetricsResponse struct {
	Connections  int    `json:"bitcoin_sprint_active_connections"`
	Uptime       int64  `json:"bitcoin_sprint_uptime_seconds"`
	Memory       uint64 `json:"go_memstats_heap_alloc_bytes"`
	Goroutines   int    `json:"go_goroutines"`
	RequestCount int    `json:"bitcoin_sprint_request_count"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	requestCount++

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	response := HealthResponse{
		Status:    "healthy",
		Version:   "2.5.0-enterprise",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Uptime:    int64(time.Since(startTime).Seconds()),
	}

	json.NewEncoder(w).Encode(response)
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	requestCount++

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	uptime := int64(time.Since(startTime).Seconds())
	connections := 7 // Simulating 7 Bitcoin connections

	metrics := fmt.Sprintf(`# HELP bitcoin_sprint_active_connections Number of active P2P connections
# TYPE bitcoin_sprint_active_connections gauge
bitcoin_sprint_active_connections %d

# HELP bitcoin_sprint_uptime_seconds Server uptime in seconds
# TYPE bitcoin_sprint_uptime_seconds counter
bitcoin_sprint_uptime_seconds %d

# HELP go_memstats_heap_alloc_bytes Number of heap bytes allocated
# TYPE go_memstats_heap_alloc_bytes gauge
go_memstats_heap_alloc_bytes %d

# HELP go_goroutines Number of goroutines that currently exist
# TYPE go_goroutines gauge
go_goroutines %d

# HELP bitcoin_sprint_request_count Total number of HTTP requests
# TYPE bitcoin_sprint_request_count counter
bitcoin_sprint_request_count %d

# HELP bitcoin_sprint_bitcoin_peers Number of Bitcoin peers connected
# TYPE bitcoin_sprint_bitcoin_peers gauge
bitcoin_sprint_bitcoin_peers %d

# HELP bitcoin_sprint_ethereum_peers Number of Ethereum peers connected
# TYPE bitcoin_sprint_ethereum_peers gauge
bitcoin_sprint_ethereum_peers 2
`, connections, uptime, memStats.HeapAlloc, runtime.NumGoroutine(), requestCount, connections)

	w.Write([]byte(metrics))
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	requestCount++

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	response := map[string]interface{}{
		"api_status":           "online",
		"bitcoin_connections":  7,
		"ethereum_connections": 2,
		"solana_connections":   0,
		"uptime_seconds":       int64(time.Since(startTime).Seconds()),
		"version":              "2.5.0-enterprise",
		"tier":                 "enterprise",
		"rust_server_status":   "online",
	}

	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/metrics", metricsHandler)
	http.HandleFunc("/status", statusHandler)

	fmt.Println("🚀 Bitcoin Sprint TURBO API Server Starting...")
	fmt.Println("📡 Server listening on http://localhost:8080")
	fmt.Println("🔗 Endpoints:")
	fmt.Println("   - GET /health   - Health check")
	fmt.Println("   - GET /metrics  - Prometheus metrics")
	fmt.Println("   - GET /status   - System status")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
