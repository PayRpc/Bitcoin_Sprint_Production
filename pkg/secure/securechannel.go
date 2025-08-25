// Professional SecureChannel Integration for Bitcoin Sprint
// This file extends the existing secure package with SecureChannelPool integration
package secure

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// SecureChannelPoolClient provides professional integration with the Rust SecureChannelPool
type SecureChannelPoolClient struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
	mu         sync.RWMutex
	lastStatus *PoolStatus
	lastHealth *HealthStatus
}

// PoolStatus represents the status of the SecureChannelPool
type PoolStatus struct {
	Endpoint          string             `json:"endpoint"`
	ActiveConnections int                `json:"active_connections"`
	TotalReconnects   uint64             `json:"total_reconnects"`
	TotalErrors       uint64             `json:"total_errors"`
	PoolP95LatencyMs  uint64             `json:"pool_p95_latency_ms"`
	Connections       []ConnectionStatus `json:"connections"`
}

// ConnectionStatus represents individual connection status
type ConnectionStatus struct {
	ConnectionID int       `json:"connection_id"`
	LastActivity time.Time `json:"last_activity"`
	Reconnects   uint64    `json:"reconnects"`
	Errors       uint64    `json:"errors"`
	P95LatencyMs uint64    `json:"p95_latency_ms"`
}

// HealthStatus represents pool health information
type HealthStatus struct {
	Status            string    `json:"status"`
	Timestamp         time.Time `json:"timestamp"`
	PoolHealthy       bool      `json:"pool_healthy"`
	ActiveConnections int       `json:"active_connections"`
}

// NewSecureChannelPoolClient creates a new client for the SecureChannelPool
func NewSecureChannelPoolClient(baseURL string) *SecureChannelPoolClient {
	if baseURL == "" {
		baseURL = "http://localhost:9090"
	}

	return &SecureChannelPoolClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		userAgent: "BitcoinSprint-SecureChannel/1.0.0",
	}
}

// GetPoolStatus retrieves the current pool status
func (c *SecureChannelPoolClient) GetPoolStatus(ctx context.Context) (*PoolStatus, error) {
	url := fmt.Sprintf("%s/status/connections", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var status PoolStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Cache the status
	c.mu.Lock()
	c.lastStatus = &status
	c.mu.Unlock()

	return &status, nil
}

// GetHealthStatus retrieves the current health status
func (c *SecureChannelPoolClient) GetHealthStatus(ctx context.Context) (*HealthStatus, error) {
	url := fmt.Sprintf("%s/healthz", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	var health HealthStatus
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode health response: %w", err)
	}

	// Cache the health status
	c.mu.Lock()
	c.lastHealth = &health
	c.mu.Unlock()

	return &health, nil
}

// IsHealthy performs a quick health check
func (c *SecureChannelPoolClient) IsHealthy(ctx context.Context) bool {
	health, err := c.GetHealthStatus(ctx)
	if err != nil {
		return false
	}
	return health.Status == "healthy" && health.PoolHealthy
}

// GetCachedStatus returns the last cached status (for performance)
func (c *SecureChannelPoolClient) GetCachedStatus() *PoolStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastStatus
}

// GetCachedHealth returns the last cached health status (for performance)
func (c *SecureChannelPoolClient) GetCachedHealth() *HealthStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastHealth
}

// GetConnectionStats calculates aggregated connection statistics
func (c *SecureChannelPoolClient) GetConnectionStats(ctx context.Context) (*ConnectionStats, error) {
	poolStatus, err := c.GetPoolStatus(ctx)
	if err != nil {
		return nil, err
	}

	stats := &ConnectionStats{
		TotalConnections:  len(poolStatus.Connections),
		ActiveConnections: poolStatus.ActiveConnections,
		TotalReconnects:   poolStatus.TotalReconnects,
		TotalErrors:       poolStatus.TotalErrors,
		PoolP95LatencyMs:  poolStatus.PoolP95LatencyMs,
	}

	// Calculate additional statistics
	if len(poolStatus.Connections) > 0 {
		var totalLatency uint64
		var healthyConnections int

		for _, conn := range poolStatus.Connections {
			totalLatency += conn.P95LatencyMs
			// Consider a connection healthy if it has recent activity and low error rate
			if time.Since(conn.LastActivity) < 5*time.Minute && conn.Errors < conn.Reconnects {
				healthyConnections++
			}
		}

		stats.AvgLatencyMs = float64(totalLatency) / float64(len(poolStatus.Connections))
		stats.HealthyPercentage = float64(healthyConnections) / float64(len(poolStatus.Connections)) * 100
	}

	if poolStatus.TotalReconnects > 0 {
		stats.ErrorRate = float64(poolStatus.TotalErrors) / float64(poolStatus.TotalReconnects) * 100
	}

	return stats, nil
}

// ConnectionStats represents aggregated connection statistics
type ConnectionStats struct {
	TotalConnections  int     `json:"total_connections"`
	ActiveConnections int     `json:"active_connections"`
	TotalReconnects   uint64  `json:"total_reconnects"`
	TotalErrors       uint64  `json:"total_errors"`
	PoolP95LatencyMs  uint64  `json:"pool_p95_latency_ms"`
	AvgLatencyMs      float64 `json:"avg_latency_ms"`
	HealthyPercentage float64 `json:"healthy_percentage"`
	ErrorRate         float64 `json:"error_rate"`
}
