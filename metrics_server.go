package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
	"encoding/json"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	sprintChainBlockHeight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sprint_chain_block_height",
			Help: "Current block height of the blockchain",
		},
		[]string{"chain"},
	)

	sprintChainPeerCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sprint_chain_peer_count",
			Help: "Number of connected peers",
		},
		[]string{"chain"},
	)

	sprintChainHealthScore = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sprint_chain_health_score",
			Help: "Health score of the blockchain connection",
		},
		[]string{"chain"},
	)

	sprintAPIRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sprint_api_requests_total",
			Help: "Total number of API requests",
		},
		[]string{"chain", "method"},
	)

	sprintAPIRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "sprint_api_request_duration_seconds",
			Help:    "API request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"chain", "method"},
	)
)

// Entropy Bridge Metrics
var (
	entropyBridgeAvailable = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "bitcoin_sprint_entropy_bridge_available",
			Help: "Entropy bridge availability status (1 = available, 0 = unavailable)",
		},
	)

	entropyBridgeRustAvailable = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "bitcoin_sprint_entropy_bridge_rust_available",
			Help: "Rust entropy bridge availability (1 = available, 0 = unavailable)",
		},
	)

	entropyBridgeFallbackMode = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "bitcoin_sprint_entropy_bridge_fallback_mode",
			Help: "Entropy bridge fallback mode status (1 = fallback active, 0 = primary active)",
		},
	)

	entropyBridgeStatusFetchTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "bitcoin_sprint_entropy_bridge_status_fetch_total",
			Help: "Total number of entropy bridge status fetches",
		},
	)

	entropyBridgeStatusFetchErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "bitcoin_sprint_entropy_bridge_status_fetch_errors_total",
			Help: "Total number of entropy bridge status fetch errors",
		},
	)
)

func init() {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())
}

// Entropy Bridge Status Structure
type EntropyBridgeStatus struct {
	Status struct {
		Available    bool `json:"available"`
		RustAvailable bool `json:"rustAvailable"`
		FallbackMode bool `json:"fallbackMode"`
		Timestamp    int64 `json:"timestamp"`
	} `json:"status"`
	Uptime            float64 `json:"uptime"`
	LastSecretGenerated *int64 `json:"lastSecretGenerated,omitempty"`
}

// Fetch entropy bridge status from Next.js application
func fetchEntropyBridgeStatus() (*EntropyBridgeStatus, error) {
	entropyBridgeStatusFetchTotal.Inc()

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("http://localhost:3002/api/admin/entropy-status")
	if err != nil {
		entropyBridgeStatusFetchErrors.Inc()
		return nil, fmt.Errorf("failed to fetch entropy bridge status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		entropyBridgeStatusFetchErrors.Inc()
		return nil, fmt.Errorf("entropy bridge status endpoint returned status: %d", resp.StatusCode)
	}

	var status EntropyBridgeStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		entropyBridgeStatusFetchErrors.Inc()
		return nil, fmt.Errorf("failed to decode entropy bridge status: %v", err)
	}

	return &status, nil
}

// Update entropy bridge metrics
func updateEntropyBridgeMetrics() {
	status, err := fetchEntropyBridgeStatus()
	if err != nil {
		log.Printf("Failed to fetch entropy bridge status: %v", err)
		// Set error state
		entropyBridgeAvailable.Set(0)
		entropyBridgeRustAvailable.Set(0)
		entropyBridgeFallbackMode.Set(1)
		return
	}

	// Update metrics based on status
	if status.Status.Available {
		entropyBridgeAvailable.Set(1)
	} else {
		entropyBridgeAvailable.Set(0)
	}

	if status.Status.RustAvailable {
		entropyBridgeRustAvailable.Set(1)
	} else {
		entropyBridgeRustAvailable.Set(0)
	}

	if status.Status.FallbackMode {
		entropyBridgeFallbackMode.Set(1)
	} else {
		entropyBridgeFallbackMode.Set(0)
	}

	log.Printf("Updated entropy bridge metrics: available=%t, rust=%t, fallback=%t",
		status.Status.Available, status.Status.RustAvailable, status.Status.FallbackMode)
}

func updateMetrics() {
	for {
		// Update Ethereum metrics with realistic values
		blockHeight := 18500000 + rand.Intn(1000)
		peerCount := 40 + rand.Intn(20)
		healthScore := 8.0 + rand.Float64()*2

		sprintChainBlockHeight.WithLabelValues("ethereum").Set(float64(blockHeight))
		sprintChainPeerCount.WithLabelValues("ethereum").Set(float64(peerCount))
		sprintChainHealthScore.WithLabelValues("ethereum").Set(healthScore)

		// Simulate some API requests
		sprintAPIRequestsTotal.WithLabelValues("ethereum", "eth_blockNumber").Inc()
		sprintAPIRequestDuration.WithLabelValues("ethereum", "eth_blockNumber").Observe(0.05 + rand.Float64()*0.1)

		// Update entropy bridge metrics
		updateEntropyBridgeMetrics()

		fmt.Printf("Updated metrics: block=%d, peers=%d, health=%.1f\n", blockHeight, peerCount, healthScore)

		time.Sleep(30 * time.Second)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// Register metrics with Prometheus
	prometheus.MustRegister(sprintChainBlockHeight)
	prometheus.MustRegister(sprintChainPeerCount)
	prometheus.MustRegister(sprintChainHealthScore)
	prometheus.MustRegister(sprintAPIRequestsTotal)
	prometheus.MustRegister(sprintAPIRequestDuration)

	// Register entropy bridge metrics
	prometheus.MustRegister(entropyBridgeAvailable)
	prometheus.MustRegister(entropyBridgeRustAvailable)
	prometheus.MustRegister(entropyBridgeFallbackMode)
	prometheus.MustRegister(entropyBridgeStatusFetchTotal)
	prometheus.MustRegister(entropyBridgeStatusFetchErrors)

	log.Println("Prometheus metrics registered successfully")

	// Start metrics update goroutine
	go updateMetrics()

	// Expose metrics endpoint using Prometheus handler
	http.Handle("/metrics", promhttp.Handler())

	// Also add a simple health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Health endpoint accessed")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Println("HTTP handlers registered")
	log.Println("Starting HTTP server on :8081")
	server := &http.Server{
		Addr:    ":8081",
		Handler: nil,
	}
	log.Fatal(server.ListenAndServe())
}
