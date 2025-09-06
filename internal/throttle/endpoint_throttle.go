package throttle

import (
	"fmt"
	"math"
	"sync"
	"time"

	"go.uber.org/zap"
)

// EndpointStatus tracks the health and performance of an endpoint
type EndpointStatus struct {
	URL            string
	SuccessCount   int64
	FailureCount   int64
	LastSuccess    time.Time
	LastFailure    time.Time
	NextRetry      time.Time
	CurrentBackoff time.Duration
	SuccessRate    float64
}

// ThrottleConfig holds throttling configuration
type ThrottleConfig struct {
	MaxRetries        int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffMultiplier float64
	SuccessThreshold  float64
}

// EndpointThrottle manages endpoint throttling with adaptive backoff
type EndpointThrottle struct {
	mu        sync.RWMutex
	endpoints map[string]*EndpointStatus
	config    *ThrottleConfig
	logger    *zap.Logger
}

// NewEndpointThrottle creates a new endpoint throttle manager
func NewEndpointThrottle(config *ThrottleConfig, logger *zap.Logger) *EndpointThrottle {
	if config == nil {
		config = &ThrottleConfig{
			MaxRetries:        3,
			InitialBackoff:    time.Second,
			MaxBackoff:        time.Minute * 5,
			BackoffMultiplier: 2.0,
			SuccessThreshold:  0.8,
		}
	}

	return &EndpointThrottle{
		endpoints: make(map[string]*EndpointStatus),
		config:    config,
		logger:    logger,
	}
}

// MetricsProvider defines the interface for metrics collection
type MetricsProvider interface {
	IncrementCounter(name string, tags map[string]string)
	RecordGauge(name string, value float64, tags map[string]string)
	RecordHistogram(name string, value float64, tags map[string]string)
}

// EndpointThrottleWithMetrics extends EndpointThrottle with metrics
type EndpointThrottleWithMetrics struct {
	*EndpointThrottle
	metrics MetricsProvider
}

// NewWithMetrics creates a new endpoint throttle with metrics support
func NewWithMetrics(logger *zap.Logger, metrics MetricsProvider) *EndpointThrottleWithMetrics {
	baseThrottle := NewEndpointThrottle(nil, logger)
	return &EndpointThrottleWithMetrics{
		EndpointThrottle: baseThrottle,
		metrics:          metrics,
	}
}

// CircuitBreaker defines the interface for circuit breakers
type CircuitBreaker interface {
	Execute(func() (interface{}, error)) (interface{}, error)
	AllowRequest() bool
	Name() string
}

// ProtectedEndpoint represents an endpoint with circuit breaker protection
type ProtectedEndpoint struct {
	URL            string
	Priority       int
	Timeout        time.Duration
	CircuitBreaker CircuitBreaker
}

// RegisterEndpoint registers a protected endpoint
func (et *EndpointThrottleWithMetrics) RegisterEndpoint(endpoint ProtectedEndpoint) {
	et.mu.Lock()
	defer et.mu.Unlock()

	et.registerEndpoint(endpoint.URL)

	et.metrics.IncrementCounter("endpoint_registered", map[string]string{
		"url":      endpoint.URL,
		"priority": fmt.Sprintf("%d", endpoint.Priority),
	})

	et.logger.Info("Registered protected endpoint",
		zap.String("url", endpoint.URL),
		zap.Int("priority", endpoint.Priority),
		zap.Duration("timeout", endpoint.Timeout),
		zap.String("circuit_breaker", endpoint.CircuitBreaker.Name()))
}

// ThrottleCfg holds all scoring and throttle parameters
type ThrottleCfg struct {
	MinSuccessRate     float64       // 0.90
	BonusIfAbove       float64       // 0.10
	RecentSuccessMax   float64       // 0.05
	RecentFailureMax   float64       // 0.10
	SuccessHalfLife    time.Duration // 10 * time.Minute
	FailureHalfLife    time.Duration // 60 * time.Minute
	Cap                float64       // 1.15
	Floor              float64       // 0.20
	InitialBackoff     time.Duration // 10 * time.Minute
	MaxBackoff         time.Duration // 30 * time.Minute
	BackoffMultiplier  float64       // 1.5
	HealthCheckWindow  int64         // 100
	EnableLatencyBlend bool
	LatencyWeight      float64 // 0.20
	LatencyRefMs       float64 // p95 fleet median in ms
}

func DefaultThrottleCfg() *ThrottleCfg {
	return &ThrottleCfg{
		MinSuccessRate:     0.90,
		BonusIfAbove:       0.10,
		RecentSuccessMax:   0.05,
		RecentFailureMax:   0.10,
		SuccessHalfLife:    10 * time.Minute,
		FailureHalfLife:    60 * time.Minute,
		Cap:                1.15,
		Floor:              0.20,
		InitialBackoff:     10 * time.Minute,
		MaxBackoff:         30 * time.Minute,
		BackoffMultiplier:  1.5,
		HealthCheckWindow:  100,
		EnableLatencyBlend: true,
		LatencyWeight:      0.20,
		LatencyRefMs:       200.0, // Example default
	}
}

func (et *EndpointThrottle) registerEndpoint(url string) {
	if _, exists := et.endpoints[url]; !exists {
		et.endpoints[url] = &EndpointStatus{
			URL:            url,
			LastSuccess:    time.Now(),
			CurrentBackoff: et.config.InitialBackoff,
			SuccessRate:    1.0, // Start optimistic
		}
		et.logger.Info("Registered endpoint", zap.String("url", url))
	}
}

// RecordSuccess records a successful request to an endpoint
func (et *EndpointThrottle) RecordSuccess(url string) {
	et.mu.Lock()
	defer et.mu.Unlock()

	status, exists := et.endpoints[url]
	if !exists {
		return
	}

	status.SuccessCount++
	status.LastSuccess = time.Now()
	status.CurrentBackoff = et.config.InitialBackoff // Reset backoff on success
}

// RecordFailure records a failed request to an endpoint
func (et *EndpointThrottle) RecordFailure(url string) {
	et.mu.Lock()
	defer et.mu.Unlock()

	status, exists := et.endpoints[url]
	if !exists {
		et.registerEndpoint(url)
		status = et.endpoints[url]
	}

	status.FailureCount++
	status.LastFailure = time.Now()

	// Increase backoff
	status.CurrentBackoff = time.Duration(float64(status.CurrentBackoff) * et.config.BackoffMultiplier)
	if status.CurrentBackoff > et.config.MaxBackoff {
		status.CurrentBackoff = et.config.MaxBackoff
	}

	status.NextRetry = time.Now().Add(status.CurrentBackoff)
}

// ShouldThrottle checks if an endpoint should be throttled
func (et *EndpointThrottle) ShouldThrottle(url string) bool {
	et.mu.RLock()
	defer et.mu.RUnlock()

	status, exists := et.endpoints[url]
	if !exists {
		return false
	}

	// Check if we're in backoff period
	if time.Now().Before(status.NextRetry) {
		return true
	}

	// Check success rate
	total := status.SuccessCount + status.FailureCount
	if total > 0 {
		successRate := float64(status.SuccessCount) / float64(total)
		status.SuccessRate = successRate
		return successRate < et.config.SuccessThreshold
	}

	return false
}

// GetStatus returns the current status of an endpoint
func (et *EndpointThrottle) GetStatus(url string) (*EndpointStatus, error) {
	et.mu.RLock()
	defer et.mu.RUnlock()

	status, exists := et.endpoints[url]
	if !exists {
		return nil, fmt.Errorf("endpoint not found: %s", url)
	}

	// Return a copy to avoid race conditions
	return &EndpointStatus{
		URL:            status.URL,
		SuccessCount:   status.SuccessCount,
		FailureCount:   status.FailureCount,
		LastSuccess:    status.LastSuccess,
		LastFailure:    status.LastFailure,
		NextRetry:      status.NextRetry,
		CurrentBackoff: status.CurrentBackoff,
		SuccessRate:    status.SuccessRate,
	}, nil
}

// Exponential decay helper
func expDecay(dt time.Duration, halfLife time.Duration) float64 {
	if halfLife <= 0 {
		return 0
	}
	return math.Exp(-math.Ln2 * float64(dt) / float64(halfLife))
}

// Clamp helper
func clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

// calculateEndpointScore implements exponential decay, cap/floor, and latency blend
func calculateEndpointScore(
	successRate float64,
	lastSuccessAgo time.Duration,
	lastFailureAgo *time.Duration,
	p95LatencyMs float64,
	cfg *ThrottleCfg,
) float64 {
	score := clamp(successRate, 0, 1)
	if successRate >= cfg.MinSuccessRate {
		score += cfg.BonusIfAbove
	}
	rs := cfg.RecentSuccessMax * expDecay(lastSuccessAgo, cfg.SuccessHalfLife)
	score += rs
	if lastFailureAgo != nil {
		rf := cfg.RecentFailureMax * expDecay(*lastFailureAgo, cfg.FailureHalfLife)
		score -= rf
	}
	if cfg.EnableLatencyBlend && cfg.LatencyRefMs > 0 {
		latFactor := cfg.LatencyRefMs / math.Max(p95LatencyMs, 1.0)
		latFactor = clamp(latFactor, 0.5, 1.5)
		w := clamp(cfg.LatencyWeight, 0, 0.5)
		blend := (1.0-w)*1.0 + w*latFactor
		score *= clamp(blend, 0.8, 1.2)
	}
	score = clamp(score, cfg.Floor, cfg.Cap)
	return score
}

// Reset clears all endpoint statistics
func (et *EndpointThrottle) Reset() {
	et.mu.Lock()
	defer et.mu.Unlock()

	et.endpoints = make(map[string]*EndpointStatus)
	et.logger.Info("Endpoint throttle statistics reset")
}

// GetAllStatuses returns the status of all tracked endpoints
func (et *EndpointThrottle) GetAllStatuses() map[string]*EndpointStatus {
	et.mu.RLock()
	defer et.mu.RUnlock()

	result := make(map[string]*EndpointStatus)
	for url, status := range et.endpoints {
		result[url] = &EndpointStatus{
			URL:            status.URL,
			SuccessCount:   status.SuccessCount,
			FailureCount:   status.FailureCount,
			LastSuccess:    status.LastSuccess,
			LastFailure:    status.LastFailure,
			NextRetry:      status.NextRetry,
			CurrentBackoff: status.CurrentBackoff,
			SuccessRate:    status.SuccessRate,
		}
	}

	return result
}
