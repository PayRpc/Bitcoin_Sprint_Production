// Package securechannel provides a professional Go client for the SecureChannelPool Rust service
// This package offers a comprehensive API for monitoring and managing secure Bitcoin connections
package securechannel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client provides a professional API client for SecureChannelPool monitoring
type Client struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
}

// ClientConfig holds configuration options for the SecureChannelPool client
type ClientConfig struct {
	BaseURL        string        `json:"base_url"`
	Timeout        time.Duration `json:"timeout"`
	RetryAttempts  int           `json:"retry_attempts"`
	RetryDelay     time.Duration `json:"retry_delay"`
	UserAgent      string        `json:"user_agent"`
	EnableMetrics  bool          `json:"enable_metrics"`
	HealthInterval time.Duration `json:"health_interval"`
}

// DefaultConfig returns a production-ready configuration
func DefaultConfig() *ClientConfig {
	return &ClientConfig{
		BaseURL:        "http://127.0.0.1:8335", // Bitcoin Core peer networking port
		Timeout:        10 * time.Second,
		RetryAttempts:  3,
		RetryDelay:     1 * time.Second,
		UserAgent:      "BitcoinSprint-SecureChannel-Client/1.0.0",
		EnableMetrics:  true,
		HealthInterval: 30 * time.Second,
	}
}

// NewClient creates a new professional SecureChannelPool API client
func NewClient(config *ClientConfig) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	return &Client{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		userAgent: config.UserAgent,
	}
}

// PoolStatus represents the complete status of the connection pool
type PoolStatus struct {
	Endpoint          string             `json:"endpoint"`
	ActiveConnections int                `json:"active_connections"`
	TotalReconnects   uint64             `json:"total_reconnects"`
	TotalErrors       uint64             `json:"total_errors"`
	PoolP95LatencyMs  uint64             `json:"pool_p95_latency_ms"`
	Connections       []ConnectionStatus `json:"connections"`
	LastUpdated       time.Time          `json:"last_updated"`
}

// ConnectionStatus represents the status of an individual connection
type ConnectionStatus struct {
	ConnectionID int       `json:"connection_id"`
	LastActivity time.Time `json:"last_activity"`
	Reconnects   uint64    `json:"reconnects"`
	Errors       uint64    `json:"errors"`
	P95LatencyMs uint64    `json:"p95_latency_ms"`
	IsHealthy    bool      `json:"is_healthy"`
	RemoteAddr   string    `json:"remote_addr,omitempty"`
}

// HealthStatus represents the health check response
type HealthStatus struct {
	Status            string    `json:"status"`
	Timestamp         time.Time `json:"timestamp"`
	PoolHealthy       bool      `json:"pool_healthy"`
	ActiveConnections int       `json:"active_connections"`
	ErrorRate         float64   `json:"error_rate,omitempty"`
	AvgLatencyMs      float64   `json:"avg_latency_ms,omitempty"`
}

// MetricsResponse represents Prometheus metrics from the pool
type MetricsResponse struct {
	ContentType string    `json:"content_type"`
	Data        string    `json:"data"`
	Timestamp   time.Time `json:"timestamp"`
}

// APIError represents an error response from the SecureChannelPool API
type APIError struct {
	Code      int       `json:"code"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("SecureChannelPool API Error %d: %s", e.Code, e.Message)
}

// GetPoolStatus retrieves the current status of all connections in the pool
func (c *Client) GetPoolStatus(ctx context.Context) (*PoolStatus, error) {
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
		return nil, c.handleErrorResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var status PoolStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	status.LastUpdated = time.Now()
	return &status, nil
}

// GetHealthStatus retrieves the health status of the connection pool
func (c *Client) GetHealthStatus(ctx context.Context) (*HealthStatus, error) {
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
		return nil, c.handleErrorResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var health HealthStatus
	if err := json.Unmarshal(body, &health); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &health, nil
}

// GetMetrics retrieves Prometheus metrics from the connection pool
func (c *Client) GetMetrics(ctx context.Context) (*MetricsResponse, error) {
	url := fmt.Sprintf("%s/metrics", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &MetricsResponse{
		ContentType: resp.Header.Get("Content-Type"),
		Data:        string(body),
		Timestamp:   time.Now(),
	}, nil
}

// handleErrorResponse parses and returns an APIError from a non-200 response
func (c *Client) handleErrorResponse(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Message != "" {
		return &apiErr
	}
	return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
}
