package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

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

func init() {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())
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
