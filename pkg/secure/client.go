package secure

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

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
		BaseURL:        "http://127.0.0.1:9191",
		Timeout:        10 * time.Second,
		RetryAttempts:  3,
		RetryDelay:     1 * time.Second,
		UserAgent:      "BitcoinSprint-SecureChannel-Client/1.0.0",
		EnableMetrics:  true,
		HealthInterval: 30 * time.Second,
	}
}

// Client is a thin wrapper over SecureChannelPoolClient to preserve existing API
type Client struct {
	inner      *SecureChannelPoolClient
	httpClient *http.Client
	userAgent  string
}

// NewClient creates a new SecureChannel client compatible with existing code
func NewClient(config *ClientConfig) *Client {
	if config == nil {
		config = DefaultConfig()
	}
	inner := NewSecureChannelPoolClient(config.BaseURL)
	// Override timeout/userAgent for HTTP client if needed
	inner.httpClient = &http.Client{Timeout: config.Timeout}
	inner.userAgent = config.UserAgent

	return &Client{
		inner:      inner,
		httpClient: inner.httpClient,
		userAgent:  inner.userAgent,
	}
}

// GetPoolStatus retrieves the current status of all connections in the pool
func (c *Client) GetPoolStatus(ctx context.Context) (*PoolStatus, error) {
	return c.inner.GetPoolStatus(ctx)
}

// GetHealthStatus retrieves the health status of the connection pool
func (c *Client) GetHealthStatus(ctx context.Context) (*HealthStatus, error) {
	return c.inner.GetHealthStatus(ctx)
}

// GetMetrics retrieves Prometheus metrics from the connection pool
func (c *Client) GetMetrics(ctx context.Context) (*MetricsResponse, error) {
	return c.inner.GetMetrics(ctx)
}

// IsHealthy performs a quick health check on the connection pool
func (c *Client) IsHealthy(ctx context.Context) (bool, error) {
	health, err := c.GetHealthStatus(ctx)
	if err != nil {
		return false, err
	}
	return health.Status == "healthy" && health.PoolHealthy, nil
}

// WaitForHealthy waits for the connection pool to become healthy
func (c *Client) WaitForHealthy(ctx context.Context, maxWait time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, maxWait)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for pool to become healthy: %w", ctx.Err())
		case <-ticker.C:
			healthy, err := c.IsHealthy(ctx)
			if err == nil && healthy {
				return nil
			}
		}
	}
}

// GetConnectionByID retrieves status for a specific connection
func (c *Client) GetConnectionByID(ctx context.Context, connectionID int) (*ConnectionStatus, error) {
	poolStatus, err := c.GetPoolStatus(ctx)
	if err != nil {
		return nil, err
	}
	for _, conn := range poolStatus.Connections {
		if conn.ConnectionID == connectionID {
			return &conn, nil
		}
	}
	return nil, fmt.Errorf("connection %d not found", connectionID)
}

// GetConnectionStats returns aggregated statistics for all connections
func (c *Client) GetConnectionStats(ctx context.Context) (*ConnectionStats, error) {
	return c.inner.GetConnectionStats(ctx)
}

// Monitor provides continuous monitoring capabilities
type Monitor struct {
	client   *Client
	interval time.Duration
	stopCh   chan struct{}
}

// NewMonitor creates a new connection pool monitor
func NewMonitor(client *Client, interval time.Duration) *Monitor {
	return &Monitor{
		client:   client,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// MonitorCallback is called with monitoring data
type MonitorCallback func(*PoolStatus, *HealthStatus, error)

// Start begins continuous monitoring of the connection pool
func (m *Monitor) Start(ctx context.Context, callback MonitorCallback) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			poolStatus, err1 := m.client.GetPoolStatus(ctx)
			healthStatus, err2 := m.client.GetHealthStatus(ctx)

			// Report any errors, but prefer pool status errors
			var err error
			if err1 != nil {
				err = err1
			} else if err2 != nil {
				err = err2
			}

			callback(poolStatus, healthStatus, err)
		}
	}
}

// Stop halts the monitoring process
func (m *Monitor) Stop() {
	close(m.stopCh)
}
