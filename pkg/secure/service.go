//go:build ignore
// +build ignore

// Package secure provides Bitcoin Sprint's secure channel integration
// This service integrates the Rust SecureChannelPool with the Go Bitcoin Sprint API
package secure

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Service provides professional SecureChannel integration for Bitcoin Sprint
type Service struct {
	client         *Client
	config         *ServiceConfig
	healthCache    *HealthCache
	metricsCache   *MetricsCache
	lastPoolStatus *PoolStatus
	lastHealthy    time.Time
	mu             sync.RWMutex

	// Prometheus metrics
	apiRequestDuration *prometheus.HistogramVec
	apiRequestTotal    *prometheus.CounterVec
	poolHealthGauge    prometheus.Gauge
	connectionGauge    prometheus.Gauge
}

// ServiceConfig holds configuration for the SecureChannel service
type ServiceConfig struct {
	RustPoolURL      string        `json:"rust_pool_url"`
	CacheTimeout     time.Duration `json:"cache_timeout"`
	HealthTimeout    time.Duration `json:"health_timeout"`
	MonitorInterval  time.Duration `json:"monitor_interval"`
	EnableMetrics    bool          `json:"enable_metrics"`
	MetricsNamespace string        `json:"metrics_namespace"`
	LogLevel         string        `json:"log_level"`
}

// DefaultServiceConfig returns production-ready service configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
	RustPoolURL:      "http://127.0.0.1:9191",
		CacheTimeout:     30 * time.Second,
		HealthTimeout:    5 * time.Second,
		MonitorInterval:  15 * time.Second,
		EnableMetrics:    true,
		MetricsNamespace: "bitcoin_sprint_secure_channel",
		LogLevel:         "info",
	}
}

// HealthCache provides cached health status with TTL
type HealthCache struct {
	status    *HealthStatus
	timestamp time.Time
	ttl       time.Duration
	mu        sync.RWMutex
}

// MetricsCache provides cached metrics with TTL
type MetricsCache struct {
	data      *MetricsResponse
	timestamp time.Time
	ttl       time.Duration
	mu        sync.RWMutex
}

// NewService creates a new professional SecureChannel service
func NewService(config *ServiceConfig) (*Service, error) {
	if config == nil {
		config = DefaultServiceConfig()
	}

	clientConfig := &ClientConfig{
		BaseURL:        config.RustPoolURL,
		Timeout:        config.HealthTimeout,
		RetryAttempts:  3,
		RetryDelay:     1 * time.Second,
		UserAgent:      "BitcoinSprint-SecureChannel-Service/1.0.0",
		EnableMetrics:  config.EnableMetrics,
		HealthInterval: config.MonitorInterval,
	}

	client := NewClient(clientConfig)

	// Initialize Prometheus metrics
	apiRequestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.MetricsNamespace,
			Name:      "api_request_duration_seconds",
			Help:      "Duration of API requests to SecureChannelPool",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"endpoint", "method"},
	)

	apiRequestTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.MetricsNamespace,
			Name:      "api_requests_total",
			Help:      "Total number of API requests to SecureChannelPool",
		},
		[]string{"endpoint", "method", "status"},
	)

	poolHealthGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: config.MetricsNamespace,
			Name:      "pool_healthy",
			Help:      "Whether the SecureChannelPool is healthy (1) or not (0)",
		},
	)

	connectionGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: config.MetricsNamespace,
			Name:      "active_connections",
			Help:      "Number of active connections in the SecureChannelPool",
		},
	)

	if config.EnableMetrics {
		prometheus.MustRegister(apiRequestDuration, apiRequestTotal, poolHealthGauge, connectionGauge)
	}

	service := &Service{
		client: client,
		config: config,
		healthCache: &HealthCache{
			ttl: config.CacheTimeout,
		},
		metricsCache: &MetricsCache{
			ttl: config.CacheTimeout,
		},
		apiRequestDuration: apiRequestDuration,
		apiRequestTotal:    apiRequestTotal,
		poolHealthGauge:    poolHealthGauge,
		connectionGauge:    connectionGauge,
	}

	return service, nil
}

// Start begins the SecureChannel service with monitoring
func (s *Service) Start(ctx context.Context) error {
	log.Printf("Starting SecureChannel service with Rust pool at %s", s.config.RustPoolURL)

	// Verify connection to Rust pool
	ctx, cancel := context.WithTimeout(ctx, s.config.HealthTimeout)
	defer cancel()

	healthy, err := s.client.IsHealthy(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to SecureChannelPool: %w", err)
	}

	if !healthy {
		log.Printf("Warning: SecureChannelPool is not healthy at startup")
	} else {
		log.Printf("Successfully connected to SecureChannelPool")
		s.lastHealthy = time.Now()
	}

	// Start background monitoring
	go s.startMonitoring(ctx)

	return nil
}

// GetPoolStatus returns the current pool status with caching
func (s *Service) GetPoolStatus(ctx context.Context) (*PoolStatus, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		s.apiRequestDuration.WithLabelValues("pool_status", "GET").Observe(duration)
	}()

	status, err := s.client.GetPoolStatus(ctx)
	if err != nil {
		s.apiRequestTotal.WithLabelValues("pool_status", "GET", "error").Inc()
		return nil, err
	}

	s.apiRequestTotal.WithLabelValues("pool_status", "GET", "success").Inc()

	// Update cached status
	s.mu.Lock()
	s.lastPoolStatus = status
	s.mu.Unlock()

	// Update Prometheus metrics
	if s.config.EnableMetrics {
		s.connectionGauge.Set(float64(status.ActiveConnections))
	}

	return status, nil
}

// GetHealthStatus returns cached health status or fetches new data
func (s *Service) GetHealthStatus(ctx context.Context) (*HealthStatus, error) {
	// Check cache first
	s.healthCache.mu.RLock()
	if s.healthCache.status != nil && time.Since(s.healthCache.timestamp) < s.healthCache.ttl {
		status := s.healthCache.status
		s.healthCache.mu.RUnlock()
		return status, nil
	}
	s.healthCache.mu.RUnlock()

	// Fetch fresh data
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		s.apiRequestDuration.WithLabelValues("health", "GET").Observe(duration)
	}()

	status, err := s.client.GetHealthStatus(ctx)
	if err != nil {
		s.apiRequestTotal.WithLabelValues("health", "GET", "error").Inc()
		return nil, err
	}

	s.apiRequestTotal.WithLabelValues("health", "GET", "success").Inc()

	// Update cache
	s.healthCache.mu.Lock()
	s.healthCache.status = status
	s.healthCache.timestamp = time.Now()
	s.healthCache.mu.Unlock()

	// Update Prometheus metrics
	if s.config.EnableMetrics {
		if status.Status == "healthy" && status.PoolHealthy {
			s.poolHealthGauge.Set(1)
			s.lastHealthy = time.Now()
		} else {
			s.poolHealthGauge.Set(0)
		}
	}

	return status, nil
}

// GetConnectionStats returns aggregated connection statistics
func (s *Service) GetConnectionStats(ctx context.Context) (*ConnectionStats, error) {
	return s.client.GetConnectionStats(ctx)
}

// IsHealthy performs a quick health check
func (s *Service) IsHealthy(ctx context.Context) (bool, error) {
	health, err := s.GetHealthStatus(ctx)
	if err != nil {
		return false, err
	}
	return health.Status == "healthy" && health.PoolHealthy, nil
}

// RegisterRoutes adds SecureChannel endpoints to a Gin router
func (s *Service) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/secure-channel")
	{
		api.GET("/status", s.handlePoolStatus)
		api.GET("/health", s.handleHealth)
		api.GET("/connections", s.handleConnections)
		api.GET("/connections/:id", s.handleConnectionByID)
		api.GET("/stats", s.handleStats)
		api.GET("/metrics", s.handleMetrics)
	}

	// Register Prometheus metrics endpoint
	if s.config.EnableMetrics {
		router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}
}

// Enhanced status response for Bitcoin Sprint integration
type EnhancedStatusResponse struct {
	Status              string                 `json:"status"`
	Timestamp           string                 `json:"timestamp"`
	Version             string                 `json:"version"`
	MemoryProtection    MemoryProtectionStatus `json:"memory_protection"`
	SecureChannel       *PoolStatus            `json:"secure_channel,omitempty"`
	SecureChannelHealth *HealthStatus          `json:"secure_channel_health,omitempty"`
	LastHealthyTime     *string                `json:"last_healthy_time,omitempty"`
	ServiceUptime       string                 `json:"service_uptime"`
}

type MemoryProtectionStatus struct {
	Enabled   bool `json:"enabled"`
	SelfCheck bool `json:"self_check"`
}

// GetEnhancedStatus returns a comprehensive status for Bitcoin Sprint integration
func (s *Service) GetEnhancedStatus(ctx context.Context) (*EnhancedStatusResponse, error) {
	response := &EnhancedStatusResponse{
		Status:    "ok",
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   "1.0.0",
		MemoryProtection: MemoryProtectionStatus{
			Enabled:   true,
			SelfCheck: true,
		},
		ServiceUptime: time.Since(s.lastHealthy).String(),
	}

	// Get pool status (non-blocking)
	poolStatus, err1 := s.GetPoolStatus(ctx)
	if err1 == nil {
		response.SecureChannel = poolStatus
	}

	// Get health status (non-blocking)
	healthStatus, err2 := s.GetHealthStatus(ctx)
	if err2 == nil {
		response.SecureChannelHealth = healthStatus
	}

	// Add last healthy time if available
	s.mu.RLock()
	if !s.lastHealthy.IsZero() {
		lastHealthyStr := s.lastHealthy.Format(time.RFC3339)
		response.LastHealthyTime = &lastHealthyStr
	}
	s.mu.RUnlock()

	// Set overall status based on health
	if err1 != nil || err2 != nil || (healthStatus != nil && (!healthStatus.PoolHealthy || healthStatus.Status != "healthy")) {
		response.Status = "degraded"
	}

	return response, nil
}

// HTTP Handlers

func (s *Service) handlePoolStatus(c *gin.Context) {
	ctx := c.Request.Context()
	status, err := s.GetPoolStatus(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Failed to get pool status",
			"details": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, status)
}

func (s *Service) handleHealth(c *gin.Context) {
	ctx := c.Request.Context()
	health, err := s.GetHealthStatus(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Health check failed",
			"details": err.Error(),
		})
		return
	}

	if health.Status == "healthy" && health.PoolHealthy {
		c.JSON(http.StatusOK, health)
	} else {
		c.JSON(http.StatusServiceUnavailable, health)
	}
}

func (s *Service) handleConnections(c *gin.Context) {
	ctx := c.Request.Context()
	status, err := s.GetPoolStatus(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Failed to get connections",
			"details": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"connections": status.Connections,
		"total":       len(status.Connections),
		"active":      status.ActiveConnections,
	})
}

func (s *Service) handleConnectionByID(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var connectionID int
	if _, err := fmt.Sscanf(id, "%d", &connectionID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid connection ID",
		})
		return
	}

	conn, err := s.client.GetConnectionByID(ctx, connectionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Connection not found",
			"details": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, conn)
}

func (s *Service) handleStats(c *gin.Context) {
	ctx := c.Request.Context()
	stats, err := s.GetConnectionStats(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Failed to get statistics",
			"details": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (s *Service) handleMetrics(c *gin.Context) {
	ctx := c.Request.Context()

	// Check cache first
	s.metricsCache.mu.RLock()
	if s.metricsCache.data != nil && time.Since(s.metricsCache.timestamp) < s.metricsCache.ttl {
		data := s.metricsCache.data
		s.metricsCache.mu.RUnlock()
		c.Header("Content-Type", data.ContentType)
		c.String(http.StatusOK, data.Data)
		return
	}
	s.metricsCache.mu.RUnlock()

	// Fetch fresh metrics
	metrics, err := s.client.GetMetrics(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Failed to get metrics",
			"details": err.Error(),
		})
		return
	}

	// Update cache
	s.metricsCache.mu.Lock()
	s.metricsCache.data = metrics
	s.metricsCache.timestamp = time.Now()
	s.metricsCache.mu.Unlock()

	c.Header("Content-Type", metrics.ContentType)
	c.String(http.StatusOK, metrics.Data)
}

// startMonitoring runs background monitoring of the SecureChannelPool
func (s *Service) startMonitoring(ctx context.Context) {
	monitor := NewMonitor(s.client, s.config.MonitorInterval)

	monitor.Start(ctx, func(poolStatus *PoolStatus, healthStatus *HealthStatus, err error) {
		if err != nil {
			log.Printf("Monitoring error: %v", err)
			return
		}

		if poolStatus != nil {
			s.mu.Lock()
			s.lastPoolStatus = poolStatus
			s.mu.Unlock()

			if s.config.EnableMetrics {
				s.connectionGauge.Set(float64(poolStatus.ActiveConnections))
			}
		}

		if healthStatus != nil {
			if s.config.EnableMetrics {
				if healthStatus.Status == "healthy" && healthStatus.PoolHealthy {
					s.poolHealthGauge.Set(1)
					s.lastHealthy = time.Now()
				} else {
					s.poolHealthGauge.Set(0)
				}
			}
		}
	})
}
