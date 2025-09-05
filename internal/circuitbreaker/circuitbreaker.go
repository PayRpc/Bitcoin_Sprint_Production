package circuitbreaker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// State represents the circuit breaker state with enterprise features
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
	StateForceOpen
	StateForceClose
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	case StateForceOpen:
		return "force-open"
	case StateForceClose:
		return "force-close"
	default:
		return "unknown"
	}
}

// FailureType categorizes different types of failures
type FailureType int

const (
	FailureTypeTimeout FailureType = iota
	FailureTypeError
	FailureTypeLatency
	FailureTypeResource
	FailureTypeCircuit
)

// Policy defines circuit breaker behavior policies
type Policy int

const (
	PolicyStandard Policy = iota
	PolicyConservative
	PolicyAggressive
	PolicyAdaptive
	PolicyTierBased
)

// Config provides comprehensive configuration for enterprise circuit breaker
type Config struct {
	// Basic settings
	Name                  string            `json:"name"`
	MaxFailures          int               `json:"max_failures"`
	ResetTimeout         time.Duration     `json:"reset_timeout"`
	HalfOpenMaxCalls     int               `json:"half_open_max_calls"`
	Timeout              time.Duration     `json:"timeout"`
	
	// Advanced algorithm settings
	Policy               Policy            `json:"policy"`
	FailureThreshold     float64           `json:"failure_threshold"`
	LatencyThreshold     time.Duration     `json:"latency_threshold"`
	WindowSize           time.Duration     `json:"window_size"`
	MinRequestsThreshold int               `json:"min_requests_threshold"`
	
	// Adaptive features
	EnableAdaptive       bool              `json:"enable_adaptive"`
	AdaptiveMultiplier   float64           `json:"adaptive_multiplier"`
	MaxAdaptiveTimeout   time.Duration     `json:"max_adaptive_timeout"`
	
	// Health scoring
	EnableHealthScoring  bool              `json:"enable_health_scoring"`
	HealthThreshold      float64           `json:"health_threshold"`
	
	// Monitoring and callbacks
	EnableMetrics        bool              `json:"enable_metrics"`
	OnStateChange        func(name string, from, to State) `json:"-"`
	OnFailure           func(name string, failureType FailureType) `json:"-"`
	OnRecovery          func(name string, recoveryTime time.Duration) `json:"-"`
	
	// Tier-based settings
	TierSettings         map[string]TierConfig `json:"tier_settings"`
}

// TierConfig defines tier-specific circuit breaker behavior
type TierConfig struct {
	FailureThreshold     int               `json:"failure_threshold"`
	ResetTimeout         time.Duration     `json:"reset_timeout"`
	HalfOpenMaxCalls     int               `json:"half_open_max_calls"`
	Priority             int               `json:"priority"`
	QueueEnabled         bool              `json:"queue_enabled"`
	RetryEnabled         bool              `json:"retry_enabled"`
}

// ExecutionResult contains detailed execution information
type ExecutionResult struct {
	Success      bool              `json:"success"`
	Duration     time.Duration     `json:"duration"`
	Error        error             `json:"error,omitempty"`
	FailureType  FailureType       `json:"failure_type,omitempty"`
	State        State             `json:"state"`
	Attempt      int               `json:"attempt"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// SlidingWindow tracks statistics over a time window
type SlidingWindow struct {
	mu           sync.RWMutex
	buckets      []WindowBucket
	bucketSize   time.Duration
	windowSize   time.Duration
	currentIndex int
	lastUpdate   time.Time
}

// WindowBucket holds statistics for a time bucket
type WindowBucket struct {
	timestamp     time.Time
	requests      int64
	failures      int64
	latencySum    int64
	latencyCount  int64
	maxLatency    time.Duration
	minLatency    time.Duration
}

// AdaptiveThreshold manages dynamic threshold adjustment
type AdaptiveThreshold struct {
	mu                  sync.RWMutex
	currentThreshold    float64
	baseThreshold       float64
	multiplier          float64
	lastAdjustment      time.Time
	adjustmentHistory   []float64
	performanceHistory  []float64
}

// HealthScorer calculates overall system health
type HealthScorer struct {
	mu                sync.RWMutex
	metrics           HealthMetrics
	weights           HealthWeights
	lastCalculation   time.Time
	calculationInterval time.Duration
}

// HealthMetrics tracks various health indicators
type HealthMetrics struct {
	SuccessRate       float64 `json:"success_rate"`
	AverageLatency    time.Duration `json:"average_latency"`
	ErrorRate         float64 `json:"error_rate"`
	ResourceUtilization float64 `json:"resource_utilization"`
	ThroughputRate    float64 `json:"throughput_rate"`
}

// HealthWeights defines importance of different health factors
type HealthWeights struct {
	SuccessRate       float64 `json:"success_rate"`
	Latency           float64 `json:"latency"`
	ErrorRate         float64 `json:"error_rate"`
	ResourceUsage     float64 `json:"resource_usage"`
	Throughput        float64 `json:"throughput"`
}

// CircuitBreakerMetrics tracks comprehensive performance metrics
type CircuitBreakerMetrics struct {
	mu                    sync.RWMutex
	
	// Basic counters (atomic for performance)
	TotalRequests         int64     `json:"total_requests"`
	SuccessfulRequests    int64     `json:"successful_requests"`
	FailedRequests        int64     `json:"failed_requests"`
	TimeoutRequests       int64     `json:"timeout_requests"`
	CircuitOpenRequests   int64     `json:"circuit_open_requests"`
	
	// State tracking
	StateChanges          int64     `json:"state_changes"`
	LastStateChange       time.Time `json:"last_state_change"`
	TimeInState           map[State]time.Duration `json:"time_in_state"`
	
	// Performance metrics
	AverageLatency        time.Duration `json:"average_latency"`
	P50Latency            time.Duration `json:"p50_latency"`
	P95Latency            time.Duration `json:"p95_latency"`
	P99Latency            time.Duration `json:"p99_latency"`
	MaxLatency            time.Duration `json:"max_latency"`
	MinLatency            time.Duration `json:"min_latency"`
	
	// Health scoring
	HealthScore           float64   `json:"health_score"`
	FailureRate           float64   `json:"failure_rate"`
	RecoveryTime          time.Duration `json:"recovery_time"`
	
	// Advanced metrics
	ConsecutiveFailures   int64     `json:"consecutive_failures"`
	ConsecutiveSuccesses  int64     `json:"consecutive_successes"`
	LastFailureTime       time.Time `json:"last_failure_time"`
	LastSuccessTime       time.Time `json:"last_success_time"`
}

// EnterpriseCircuitBreaker implements comprehensive circuit breaker functionality
type EnterpriseCircuitBreaker struct {
	// Core configuration
	config               *Config
	logger               *zap.Logger
	
	// State management
	mu                   sync.RWMutex
	state                State
	stateChangedAt       time.Time
	
	// Failure tracking
	consecutiveFailures  int64
	consecutiveSuccesses int64
	halfOpenCalls        int64
	lastFailureTime      time.Time
	lastSuccessTime      time.Time
	
	// Advanced algorithms
	slidingWindow        *SlidingWindow
	adaptiveThreshold    *AdaptiveThreshold
	healthScorer         *HealthScorer
	
	// Performance tracking
	metrics              *CircuitBreakerMetrics
	latencyHistory       []time.Duration
	
	// Tier management
	currentTier          string
	tierConfigs          map[string]TierConfig
	
	// Control mechanisms
	forceState           *State
	maintenanceMode      bool
	
	// Background management
	ctx                  context.Context
	cancel               context.CancelFunc
	workerGroup          sync.WaitGroup
	shutdownChan         chan struct{}
}

// NewCircuitBreaker creates an enterprise-grade circuit breaker
func NewCircuitBreaker(cfg Config) (*EnterpriseCircuitBreaker, error) {
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	cb := &EnterpriseCircuitBreaker{
		config:           &cfg,
		logger:           zap.NewNop(), // Default logger
		state:            StateClosed,
		stateChangedAt:   time.Now(),
		tierConfigs:      cfg.TierSettings,
		ctx:              ctx,
		cancel:           cancel,
		shutdownChan:     make(chan struct{}),
		metrics:          newCircuitBreakerMetrics(),
		latencyHistory:   make([]time.Duration, 0, 1000),
	}
	
	// Initialize advanced components
	cb.slidingWindow = newSlidingWindow(cfg.WindowSize, time.Minute)
	cb.adaptiveThreshold = newAdaptiveThreshold(cfg.FailureThreshold, cfg.AdaptiveMultiplier)
	cb.healthScorer = newHealthScorer()
	
	// Start background workers
	cb.startBackgroundWorkers()
	
	return cb, nil
}

// Execute runs a function with comprehensive circuit breaker protection
func (cb *EnterpriseCircuitBreaker) Execute(fn func() (interface{}, error)) (*ExecutionResult, error) {
	return cb.ExecuteWithContext(context.Background(), fn)
}

// ExecuteWithContext runs a function with context and full protection
func (cb *EnterpriseCircuitBreaker) ExecuteWithContext(ctx context.Context, fn func() (interface{}, error)) (*ExecutionResult, error) {
	startTime := time.Now()
	
	// Check if execution is allowed
	if !cb.allowRequest() {
		atomic.AddInt64(&cb.metrics.CircuitOpenRequests, 1)
		return &ExecutionResult{
			Success:     false,
			Duration:    time.Since(startTime),
			Error:       fmt.Errorf("circuit breaker is %s", cb.state.String()),
			State:       cb.state,
			FailureType: FailureTypeCircuit,
		}, nil
	}
	
	// Execute with timeout and monitoring
	result := cb.executeWithMonitoring(ctx, fn, startTime)
	
	// Record result and update state
	cb.recordResult(result)
	
	return result, nil
}

// Allow checks if a request should be allowed through
func (cb *EnterpriseCircuitBreaker) Allow() bool {
	return cb.allowRequest()
}

// State returns the current circuit breaker state
func (cb *EnterpriseCircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// ForceOpen forces the circuit breaker to open state
func (cb *EnterpriseCircuitBreaker) ForceOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	oldState := cb.state
	cb.state = StateForceOpen
	cb.stateChangedAt = time.Now()
	cb.forceState = &cb.state
	
	cb.notifyStateChange(oldState, cb.state)
}

// ForceClose forces the circuit breaker to closed state
func (cb *EnterpriseCircuitBreaker) ForceClose() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	oldState := cb.state
	cb.state = StateForceClose
	cb.stateChangedAt = time.Now()
	cb.forceState = &cb.state
	
	cb.notifyStateChange(oldState, cb.state)
}

// Reset clears the forced state and returns to normal operation
func (cb *EnterpriseCircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	oldState := cb.state
	cb.forceState = nil
	cb.state = StateClosed
	cb.consecutiveFailures = 0
	cb.consecutiveSuccesses = 0
	cb.halfOpenCalls = 0
	cb.stateChangedAt = time.Now()
	
	cb.notifyStateChange(oldState, cb.state)
}

// GetMetrics returns comprehensive circuit breaker metrics
func (cb *EnterpriseCircuitBreaker) GetMetrics() *CircuitBreakerMetrics {
	cb.metrics.mu.RLock()
	defer cb.metrics.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	metrics := *cb.metrics
	
	// Calculate dynamic metrics
	totalRequests := atomic.LoadInt64(&cb.metrics.TotalRequests)
	successfulRequests := atomic.LoadInt64(&cb.metrics.SuccessfulRequests)
	
	if totalRequests > 0 {
		metrics.FailureRate = float64(totalRequests-successfulRequests) / float64(totalRequests)
	}
	
	// Update health score
	if cb.config.EnableHealthScoring {
		metrics.HealthScore = cb.healthScorer.CalculateHealth()
	}
	
	return &metrics
}

// SetTier updates the tier configuration for the circuit breaker
func (cb *EnterpriseCircuitBreaker) SetTier(tier string) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	if tierConfig, exists := cb.tierConfigs[tier]; exists {
		cb.currentTier = tier
		cb.adaptTierSettings(tierConfig)
		return nil
	}
	
	return fmt.Errorf("tier %s not found", tier)
}

// Shutdown gracefully shuts down the circuit breaker
func (cb *EnterpriseCircuitBreaker) Shutdown(ctx context.Context) error {
	close(cb.shutdownChan)
	
	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		cb.workerGroup.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		cb.cancel()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
