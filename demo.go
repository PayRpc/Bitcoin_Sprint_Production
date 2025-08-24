//go:build demo

package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Demo-specific configuration fields
var fastDemoMode bool

func init() {
	// Check for fast demo flag
	for _, arg := range os.Args[1:] {
		if arg == "--fast-demo" {
			fastDemoMode = true
			break
		}
	}
}

// StartBlockPoller (Demo Version) - Generates fake blocks for testing without Bitcoin Core
func (s *Sprint) StartBlockPoller() {
	var interval time.Duration
	var intervalStr string

	// Check for fast demo mode
	if fastDemoMode {
		interval = 5 * time.Second
		intervalStr = "5 seconds"
	} else {
		interval = 30 * time.Second
		intervalStr = "30 seconds"
	}

	log.Printf("🎯 DEMO MODE: Generating fake blocks every %s", intervalStr)

	currentHeight := 850000
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Generate initial block immediately
	s.generateDemoBlock(currentHeight)
	currentHeight++

	for {
		select {
		case <-s.ctx.Done():
			log.Printf("Demo mode stopped")
			return
		case <-ticker.C:
			s.generateDemoBlock(currentHeight)
			currentHeight++
		}
	}
}

// generateDemoBlock creates a fake block for demo purposes
func (s *Sprint) generateDemoBlock(height int) {
	// Generate a realistic-looking block hash
	hash := fmt.Sprintf("00000000000000000%07x%016x", height, time.Now().UnixNano())

	log.Printf("🎯 DEMO BLOCK DETECTED: %s at height %d", hash[:16], height)

	// Generate demo metrics
	metrics := Metrics{
		BlockHash:  hash,
		Height:     height,
		Latency:    float64(5 + (height % 10)), // Simulate 5-15ms latency
		PeerCount:  3,                          // Simulate 3 demo peers
		Timestamp:  time.Now().Unix(),
		LicenseKey: s.config.LicenseKey,
		RPCNode:    "demo://localhost:8332",
	}

	// Store latest metric for /latest endpoint
	s.mu.Lock()
	s.latestMetric = &metrics
	s.mu.Unlock()

	// Send to metrics channel for API
	select {
	case s.metrics <- metrics:
	default:
		// Channel full, ignore (same as real implementation)
	}

	// Simulate sprint to peers
	if s.config.TurboMode {
		log.Printf("⚡ TURBO DEMO: Block %s sprinted to %d peers in %.1fms",
			hash[:8], metrics.PeerCount, metrics.Latency)
	} else {
		log.Printf("📦 DEMO: Block %s sprinted to %d peers in %.1fms",
			hash[:8], metrics.PeerCount, metrics.Latency)
	}
}
