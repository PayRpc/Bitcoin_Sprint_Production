//go:build example
// +build example

// Professional SecureChannel API Integration Test
// This demonstrates the enterprise-grade SecureChannel integration with Bitcoin Sprint

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Test structures matching the API responses
type StatusResponse struct {
	Status        string               `json:"status"`
	Version       string               `json:"version"`
	Uptime        string               `json:"uptime"`
	TotalRequests int64                `json:"total_requests"`
	SecureChannel *SecureChannelStatus `json:"secure_channel,omitempty"`
}

type SecureChannelStatus struct {
	Enabled           bool                   `json:"enabled"`
	Endpoint          string                 `json:"endpoint"`
	ActiveConnections int                    `json:"active_connections"`
	TotalReconnects   uint64                 `json:"total_reconnects"`
	TotalErrors       uint64                 `json:"total_errors"`
	PoolP95LatencyMs  uint64                 `json:"pool_p95_latency_ms"`
	ServiceUptime     string                 `json:"service_uptime"`
	HealthStatus      string                 `json:"health_status"`
	LastCheck         string                 `json:"last_check"`
	ConnectionSummary map[string]interface{} `json:"connection_summary"`
}

type HealthResponse struct {
	Status            string    `json:"status"`
	Timestamp         time.Time `json:"timestamp"`
	PoolHealthy       bool      `json:"pool_healthy"`
	ActiveConnections int       `json:"active_connections"`
}

type ConnectionsResponse struct {
	Connections   []map[string]interface{} `json:"connections"`
	Summary       map[string]interface{}   `json:"summary"`
	Endpoint      string                   `json:"endpoint"`
	ServiceUptime string                   `json:"service_uptime"`
}

func main() {
	baseURL := "http://localhost:8080"

	fmt.Println("🚀 Bitcoin Sprint SecureChannel Professional API Test")
	fmt.Println("============================================================")

	// Test 1: Enhanced Status Endpoint
	fmt.Println("\n📊 Testing Enhanced Status Endpoint...")
	testEnhancedStatus(baseURL)

	// Test 2: SecureChannel Status
	fmt.Println("\n🔐 Testing SecureChannel Status...")
	testSecureChannelStatus(baseURL)

	// Test 3: Health Check
	fmt.Println("\n💚 Testing Health Check...")
	testHealthCheck(baseURL)

	// Test 4: Connection Monitoring
	fmt.Println("\n🌐 Testing Connection Monitoring...")
	testConnectionMonitoring(baseURL)

	fmt.Println("\n✅ Professional API Integration Test Complete!")
}

func testEnhancedStatus(baseURL string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/status", nil)
	if err != nil {
		log.Printf("❌ Failed to create request: %v", err)
		return
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Bitcoin-Sprint-Test/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Failed to read response: %v", err)
		return
	}

	var status StatusResponse
	if err := json.Unmarshal(body, &status); err != nil {
		log.Printf("❌ Failed to parse JSON: %v", err)
		return
	}

	fmt.Printf("   Status: %s\n", status.Status)
	fmt.Printf("   Version: %s\n", status.Version)
	fmt.Printf("   Uptime: %s\n", status.Uptime)

	if status.SecureChannel != nil {
		sc := status.SecureChannel
		fmt.Printf("   SecureChannel Enabled: %v\n", sc.Enabled)
		fmt.Printf("   Active Connections: %d\n", sc.ActiveConnections)
		fmt.Printf("   Health: %s\n", sc.HealthStatus)
		fmt.Printf("   Service Uptime: %s\n", sc.ServiceUptime)
	} else {
		fmt.Printf("   SecureChannel: Not available\n")
	}
}

func testSecureChannelStatus(baseURL string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/v1/secure-channel/status", nil)
	if err != nil {
		log.Printf("❌ Failed to create request: %v", err)
		return
	}

	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   Status Code: %d\n", resp.StatusCode)
	fmt.Printf("   SecureChannel Header: %s\n", resp.Header.Get("X-Bitcoin-Sprint-SecureChannel"))

	if resp.StatusCode == http.StatusServiceUnavailable {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("   Response: %s\n", string(body))
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Failed to read response: %v", err)
		return
	}

	var poolStatus map[string]interface{}
	if err := json.Unmarshal(body, &poolStatus); err != nil {
		log.Printf("❌ Failed to parse JSON: %v", err)
		return
	}

	fmt.Printf("   Endpoint: %v\n", poolStatus["endpoint"])
	fmt.Printf("   Active Connections: %v\n", poolStatus["active_connections"])
	fmt.Printf("   Total Reconnects: %v\n", poolStatus["total_reconnects"])
}

func testHealthCheck(baseURL string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/v1/secure-channel/health", nil)
	if err != nil {
		log.Printf("❌ Failed to create request: %v", err)
		return
	}

	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   Status Code: %d\n", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Failed to read response: %v", err)
		return
	}

	if resp.StatusCode == http.StatusServiceUnavailable {
		fmt.Printf("   Response: %s\n", string(body))
		return
	}

	var health HealthResponse
	if err := json.Unmarshal(body, &health); err != nil {
		log.Printf("❌ Failed to parse JSON: %v", err)
		return
	}

	fmt.Printf("   Health Status: %s\n", health.Status)
	fmt.Printf("   Pool Healthy: %v\n", health.PoolHealthy)
	fmt.Printf("   Active Connections: %d\n", health.ActiveConnections)
	fmt.Printf("   Timestamp: %s\n", health.Timestamp.Format(time.RFC3339))
}

func testConnectionMonitoring(baseURL string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/v1/secure-channel/connections", nil)
	if err != nil {
		log.Printf("❌ Failed to create request: %v", err)
		return
	}

	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   Status Code: %d\n", resp.StatusCode)
	fmt.Printf("   Total Connections: %s\n", resp.Header.Get("X-Total-Connections"))
	fmt.Printf("   Active Connections: %s\n", resp.Header.Get("X-Active-Connections"))

	if resp.StatusCode == http.StatusServiceUnavailable {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("   Response: %s\n", string(body))
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Failed to read response: %v", err)
		return
	}

	var connections ConnectionsResponse
	if err := json.Unmarshal(body, &connections); err != nil {
		log.Printf("❌ Failed to parse JSON: %v", err)
		return
	}

	fmt.Printf("   Endpoint: %s\n", connections.Endpoint)
	fmt.Printf("   Service Uptime: %s\n", connections.ServiceUptime)
	fmt.Printf("   Connection Count: %d\n", len(connections.Connections))

	if summary := connections.Summary; summary != nil {
		fmt.Printf("   Error Rate: %v\n", summary["error_rate"])
		fmt.Printf("   Avg Latency: %v ms\n", summary["avg_latency_ms"])
		fmt.Printf("   Healthy %: %v\n", summary["healthy_percentage"])
	}
}
