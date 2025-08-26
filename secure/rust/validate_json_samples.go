//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"time"
)

// Type definitions for SecureChannel integration
type PoolStatus struct {
	Endpoint          string             `json:"endpoint"`
	ActiveConnections int                `json:"active_connections"`
	TotalReconnects   uint64             `json:"total_reconnects"`
	TotalErrors       uint64             `json:"total_errors"`
	PoolP95LatencyMs  uint64             `json:"pool_p95_latency_ms"`
	Connections       []ConnectionStatus `json:"connections"`
}

type ConnectionStatus struct {
	ConnectionID int       `json:"connection_id"`
	LastActivity time.Time `json:"last_activity"`
	Reconnects   uint64    `json:"reconnects"`
	Errors       uint64    `json:"errors"`
	P95LatencyMs uint64    `json:"p95_latency_ms"`
}

type HealthStatus struct {
	Status            string    `json:"status"`
	Timestamp         time.Time `json:"timestamp"`
	PoolHealthy       bool      `json:"pool_healthy"`
	ActiveConnections int       `json:"active_connections"`
}

type EnhancedStatusResponse struct {
	Status              string                 `json:"status"`
	Timestamp           string                 `json:"timestamp"`
	Version             string                 `json:"version"`
	MemoryProtection    MemoryProtectionStatus `json:"memory_protection"`
	SecureChannel       *PoolStatus            `json:"secure_channel,omitempty"`
	SecureChannelHealth *HealthStatus          `json:"secure_channel_health,omitempty"`
}

type MemoryProtectionStatus struct {
	Enabled   bool `json:"enabled"`
	SelfCheck bool `json:"self_check"`
}

func main() {
	fmt.Println("Validating sample JSON files...")

	// Test pool status JSON
	validatePoolStatus()

	// Test health status JSON
	validateHealthStatus()

	// Test enhanced status JSON
	validateEnhancedStatus()

	fmt.Println("\n✅ All sample JSON files are valid!")
	fmt.Println("\n📋 Summary:")
	fmt.Println("   • Pool status JSON structure: ✓ Valid")
	fmt.Println("   • Health status JSON structure: ✓ Valid")
	fmt.Println("   • Enhanced status JSON structure: ✓ Valid")
	fmt.Println("   • All timestamp formats: ✓ Compatible")
	fmt.Println("   • All field types: ✓ Correct")
}

func validatePoolStatus() {
	fmt.Println("\n=== Validating sample_pool_status.json ===")

	data, err := ioutil.ReadFile("sample_pool_status.json")
	if err != nil {
		log.Fatalf("Failed to read sample_pool_status.json: %v", err)
	}

	var poolStatus PoolStatus
	if err := json.Unmarshal(data, &poolStatus); err != nil {
		log.Fatalf("Failed to parse pool status JSON: %v", err)
	}

	// Validate structure
	if poolStatus.Endpoint == "" {
		log.Fatalf("Endpoint is empty")
	}

	if poolStatus.ActiveConnections != len(poolStatus.Connections) {
		log.Fatalf("ActiveConnections (%d) doesn't match Connections length (%d)",
			poolStatus.ActiveConnections, len(poolStatus.Connections))
	}

	fmt.Printf("✓ Endpoint: %s\n", poolStatus.Endpoint)
	fmt.Printf("✓ Active connections: %d\n", poolStatus.ActiveConnections)
	fmt.Printf("✓ Total reconnects: %d\n", poolStatus.TotalReconnects)
	fmt.Printf("✓ Total errors: %d\n", poolStatus.TotalErrors)
	fmt.Printf("✓ P95 latency: %d ms\n", poolStatus.PoolP95LatencyMs)

	for i, conn := range poolStatus.Connections {
		fmt.Printf("✓ Connection %d: ID=%d, P95=%d ms\n",
			i+1, conn.ConnectionID, conn.P95LatencyMs)
	}
}

func validateHealthStatus() {
	fmt.Println("\n=== Validating sample_health_status.json ===")

	data, err := ioutil.ReadFile("sample_health_status.json")
	if err != nil {
		log.Fatalf("Failed to read sample_health_status.json: %v", err)
	}

	var healthStatus HealthStatus
	if err := json.Unmarshal(data, &healthStatus); err != nil {
		log.Fatalf("Failed to parse health status JSON: %v", err)
	}

	// Validate structure
	if healthStatus.Status == "" {
		log.Fatalf("Status is empty")
	}

	if healthStatus.Timestamp.IsZero() {
		log.Fatalf("Timestamp is zero")
	}

	fmt.Printf("✓ Status: %s\n", healthStatus.Status)
	fmt.Printf("✓ Timestamp: %s\n", healthStatus.Timestamp.Format(time.RFC3339))
	fmt.Printf("✓ Pool healthy: %t\n", healthStatus.PoolHealthy)
	fmt.Printf("✓ Active connections: %d\n", healthStatus.ActiveConnections)
}

func validateEnhancedStatus() {
	fmt.Println("\n=== Validating sample_enhanced_status.json ===")

	data, err := ioutil.ReadFile("sample_enhanced_status.json")
	if err != nil {
		log.Fatalf("Failed to read sample_enhanced_status.json: %v", err)
	}

	var enhancedStatus EnhancedStatusResponse
	if err := json.Unmarshal(data, &enhancedStatus); err != nil {
		log.Fatalf("Failed to parse enhanced status JSON: %v", err)
	}

	// Validate main structure
	if enhancedStatus.Status == "" {
		log.Fatalf("Status is empty")
	}

	if enhancedStatus.Version == "" {
		log.Fatalf("Version is empty")
	}

	// Validate memory protection
	if !enhancedStatus.MemoryProtection.Enabled {
		log.Printf("⚠️  Memory protection is disabled")
	}

	if !enhancedStatus.MemoryProtection.SelfCheck {
		log.Printf("⚠️  Memory protection self-check is disabled")
	}

	// Validate secure channel section
	if enhancedStatus.SecureChannel == nil {
		log.Fatalf("SecureChannel section is missing")
	}

	if enhancedStatus.SecureChannelHealth == nil {
		log.Fatalf("SecureChannelHealth section is missing")
	}

	fmt.Printf("✓ Status: %s\n", enhancedStatus.Status)
	fmt.Printf("✓ Version: %s\n", enhancedStatus.Version)
	fmt.Printf("✓ Memory protection enabled: %t\n", enhancedStatus.MemoryProtection.Enabled)
	fmt.Printf("✓ Memory protection self-check: %t\n", enhancedStatus.MemoryProtection.SelfCheck)
	fmt.Printf("✓ Secure channel endpoint: %s\n", enhancedStatus.SecureChannel.Endpoint)
	fmt.Printf("✓ Secure channel health: %s\n", enhancedStatus.SecureChannelHealth.Status)

	// Cross-validate connection counts
	scConnections := enhancedStatus.SecureChannel.ActiveConnections
	healthConnections := enhancedStatus.SecureChannelHealth.ActiveConnections

	if scConnections != healthConnections {
		log.Printf("⚠️  Connection count mismatch: SecureChannel=%d, Health=%d",
			scConnections, healthConnections)
	} else {
		fmt.Printf("✓ Connection counts match: %d\n", scConnections)
	}
}
