package circuitbreaker

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Additional states for enterprise features
const (
	StateForceOpen State = iota + 3 // Starting from 3 since types.go defines 0, 1, 2
	StateForceClose
)

// String overrides the base implementation to include enterprise states
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

// EnterpriseConfig extends the base Config with additional enterprise features
type EnterpriseConfig struct {
	// Embed the base Config
	Config

	// Enterprise-specific settings
	MaxFailures      int           `json:"max_failures"`
	ResetTimeout     time.Duration `json:"reset_timeout"`
	HalfOpenMaxCalls int           `json:"half_open_max_calls"`

	// Advanced algorithm settings
	Policy               Policy        `json:"policy"`
	FailureThreshold     float64       `json:"failure_threshold"`
	LatencyThreshold     time.Duration `json:"latency_threshold"`
	WindowSize           time.Duration `json:"window_size"`
	MinRequestsThreshold int           `json:"min_requests_threshold"`

	// Adaptive features
	EnableAdaptive     bool          `json:"enable_adaptive"`
	AdaptiveMultiplier float64       `json:"adaptive_multiplier"`
	MaxAdaptiveTimeout time.Duration `json:"max_adaptive_timeout"`

	// Health scoring
	EnableHealthScoring bool    `json:"enable_health_scoring"`
	HealthThreshold     float64 `json:"health_threshold"`

	// Monitoring and callbacks
	EnableMetrics bool                                          `json:"enable_metrics"`
	OnStateChange func(name string, from, to State)             `json:"-"`
	OnFailure     func(name string, failureType FailureType)    `json:"-"`
	OnRecovery    func(name string, recoveryTime time.Duration) `json:"-"`

	// Tier-based settings
	TierSettings map[string]TierConfig `json:"tier_settings"`
}

// TierConfig defines tier-specific circuit breaker behavior
type TierConfig struct {
	FailureThreshold int           `json:"failure_threshold"`
	ResetTimeout     time.Duration `json:"reset_timeout"`
	HalfOpenMaxCalls int           `json:"half_open_max_calls"`
	Priority         int           `json:"priority"`
	QueueEnabled     bool          `json:"queue_enabled"`
	RetryEnabled     bool          `json:"retry_enabled"`
}

// ExecutionResult contains detailed execution information
type ExecutionResult struct {
	Success     bool                   `json:"success"`
	Duration    time.Duration          `json:"duration"`
	Error       error                  `json:"error,omitempty"`
	FailureType FailureType            `json:"failure_type,omitempty"`
	State       State                  `json:"state"`
	Attempt     int                    `json:"attempt"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// All algorithm-related types have been moved to algorithms.go

// CircuitBreakerMetrics tracks comprehensive performance metrics
type CircuitBreakerMetrics struct {
	mu sync.RWMutex

	// Basic counters (atomic for performance)
	TotalRequests       int64 `json:"total_requests"`
	SuccessfulRequests  int64 `json:"successful_requests"`
	FailedRequests      int64 `json:"failed_requests"`
	TimeoutRequests     int64 `json:"timeout_requests"`
	CircuitOpenRequests int64 `json:"circuit_open_requests"`

	// State tracking
	StateChanges    int64                   `json:"state_changes"`
	LastStateChange time.Time               `json:"last_state_change"`
	TimeInState     map[State]time.Duration `json:"time_in_state"`

	// Performance metrics
	AverageLatency time.Duration `json:"average_latency"`
	P50Latency     time.Duration `json:"p50_latency"`
	P95Latency     time.Duration `json:"p95_latency"`
	P99Latency     time.Duration `json:"p99_latency"`
	MaxLatency     time.Duration `json:"max_latency"`
	MinLatency     time.Duration `json:"min_latency"`

	// Health scoring
	HealthScore  float64       `json:"health_score"`
	FailureRate  float64       `json:"failure_rate"`
	RecoveryTime time.Duration `json:"recovery_time"`

	// Advanced metrics
	ConsecutiveFailures  int64     `json:"consecutive_failures"`
	ConsecutiveSuccesses int64     `json:"consecutive_successes"`
	LastFailureTime      time.Time `json:"last_failure_time"`
	LastSuccessTime      time.Time `json:"last_success_time"`
}

// EnterpriseCircuitBreaker implements comprehensive circuit breaker functionality
type EnterpriseCircuitBreaker struct {
	// Core configuration
	config *EnterpriseConfig
	logger *zap.Logger

	// State management
	mu             sync.RWMutex
	state          State
	stateChangedAt time.Time

	// Failure tracking
	consecutiveFailures  int64
	consecutiveSuccesses int64
	halfOpenCalls        int64
	lastFailureTime      time.Time
	lastSuccessTime      time.Time

	// Advanced algorithms
	slidingWindow     *SlidingWindow
	adaptiveThreshold *AdaptiveThreshold
	healthScorer      *HealthScorer

	// Performance tracking
	metrics        *CircuitBreakerMetrics
	latencyHistory []time.Duration

	// Tier management
	currentTier string
	tierConfigs map[string]TierConfig

	// Control mechanisms
	forceState      *State
	maintenanceMode bool

	// Background management
	ctx          context.Context
	cancel       context.CancelFunc
	workerGroup  sync.WaitGroup
	shutdownChan chan struct{}
}

// NewEnterpriseCircuitBreaker creates a new enterprise circuit breaker instance  
func NewEnterpriseCircuitBreaker(cfg Config) (*EnterpriseCircuitBreaker, error) {
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Type assert TierSettings to proper type
	var tierConfigs map[string]TierConfig
	if cfg.TierSettings != nil {
		if tc, ok := cfg.TierSettings.(map[string]TierConfig); ok {
			tierConfigs = tc
		} else {
			// Initialize with empty map if type assertion fails
			tierConfigs = make(map[string]TierConfig)
		}
	} else {
		tierConfigs = make(map[string]TierConfig)
	}

	// Create an EnterpriseConfig with defaults from base Config
	enterpriseConfig := &EnterpriseConfig{
		Name:             cfg.Name,
		FailureThreshold: cfg.FailureThreshold,
		MaxFailures:      int(cfg.FailureThreshold * 10), // Convert to count
		ResetTimeout:     cfg.Timeout,
		HalfOpenMaxCalls: cfg.HalfOpenMaxConcurrency,
		Timeout:          cfg.Timeout,
		MinSamples:       cfg.MinSamples,
		TierSettings:     tierConfigs,
	}

	cb := &EnterpriseCircuitBreaker{
		config:         enterpriseConfig,
		logger:         zap.NewNop(), // Default logger
		state:          StateClosed,
		stateChangedAt: time.Now(),
		tierConfigs:    tierConfigs,
		ctx:            ctx,
		cancel:         cancel,
		shutdownChan:   make(chan struct{}),
		metrics:        newCircuitBreakerMetrics(),
		latencyHistory: make([]time.Duration, 0, 1000),
	}

	// Initialize advanced components
	cb.slidingWindow = NewSlidingWindow(10*time.Second, time.Second)
	cb.adaptiveThreshold = NewAdaptiveThreshold(cfg.FailureThreshold, 0.1)
	cb.healthScorer = NewHealthScorer()

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

// Core implementation methods

// allowRequest determines if a request should be allowed
func (cb *EnterpriseCircuitBreaker) allowRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	// Handle forced states
	if cb.forceState != nil {
		switch *cb.forceState {
		case StateForceOpen:
			return false
		case StateForceClose:
			return true
		}
	}

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if it's time to try half-open
		if time.Since(cb.stateChangedAt) >= cb.config.ResetTimeout {
			cb.changeState(StateHalfOpen)
			return true
		}
		return false
	case StateHalfOpen:
		// Allow limited requests in half-open state
		halfOpenCalls := atomic.LoadInt64(&cb.halfOpenCalls)
		return halfOpenCalls < int64(cb.config.HalfOpenMaxCalls)
	default:
		return false
	}
}

// executeWithMonitoring executes function with comprehensive monitoring
func (cb *EnterpriseCircuitBreaker) executeWithMonitoring(ctx context.Context, fn func() (interface{}, error), startTime time.Time) *ExecutionResult {
	result := &ExecutionResult{
		State:   cb.state,
		Attempt: 1,
	}

	// Create execution context with timeout
	execCtx := ctx
	if cb.config.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, cb.config.Timeout)
		defer cancel()
	}

	// Execute in goroutine for timeout handling
	resultChan := make(chan struct {
		value interface{}
		err   error
	}, 1)

	go func() {
		value, err := fn()
		resultChan <- struct {
			value interface{}
			err   error
		}{value, err}
	}()

	// Wait for completion or timeout
	select {
	case res := <-resultChan:
		result.Duration = time.Since(startTime)
		result.Success = res.err == nil
		result.Error = res.err
		
		if res.err != nil {
			result.FailureType = cb.classifyFailure(res.err, result.Duration)
		}

	case <-execCtx.Done():
		result.Duration = time.Since(startTime)
		result.Success = false
		result.Error = fmt.Errorf("execution timeout after %v", result.Duration)
		result.FailureType = FailureTypeTimeout
	}

	return result
}

// recordResult records execution result and updates circuit breaker state
func (cb *EnterpriseCircuitBreaker) recordResult(result *ExecutionResult) {
	atomic.AddInt64(&cb.metrics.TotalRequests, 1)

	if result.Success {
		atomic.AddInt64(&cb.metrics.SuccessfulRequests, 1)
		cb.onSuccess(result)
	} else {
		atomic.AddInt64(&cb.metrics.FailedRequests, 1)
		cb.onFailure(result)
	}

	// Update latency tracking
	cb.updateLatencyMetrics(result.Duration)

	// Update sliding window
	if cb.slidingWindow != nil {
		cb.slidingWindow.AddRequest(result.Success, result.Duration)
	}

	// Update health scorer
	if cb.healthScorer != nil {
		// Create metrics from result and update health scorer
		metrics := HealthMetrics{
			SuccessRate:    1.0,
			ErrorRate:      0.0,
			AvgLatency:     result.Duration,
			P50Latency:     result.Duration,
			P95Latency:     result.Duration,
			P99Latency:     result.Duration,
			ThroughputRate: 1.0,
		}
		if !result.Success {
			metrics.SuccessRate = 0.0
			metrics.ErrorRate = 1.0
		}
		cb.healthScorer.UpdateMetrics(metrics)
	}
}

// onSuccess handles successful execution
func (cb *EnterpriseCircuitBreaker) onSuccess(result *ExecutionResult) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	atomic.AddInt64(&cb.consecutiveSuccesses, 1)
	atomic.StoreInt64(&cb.consecutiveFailures, 0)
	cb.lastSuccessTime = time.Now()

	switch cb.state {
	case StateHalfOpen:
		successCount := atomic.LoadInt64(&cb.consecutiveSuccesses)
		if successCount >= int64(cb.config.HalfOpenMaxCalls) {
			cb.changeState(StateClosed)
		}
	}

	// Notify callback
	if cb.config.OnRecovery != nil && cb.consecutiveFailures == 0 {
		recoveryTime := time.Since(cb.lastFailureTime)
		go cb.config.OnRecovery(cb.config.Name, recoveryTime)
	}
}

// onFailure handles failed execution
func (cb *EnterpriseCircuitBreaker) onFailure(result *ExecutionResult) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	atomic.AddInt64(&cb.consecutiveFailures, 1)
	atomic.StoreInt64(&cb.consecutiveSuccesses, 0)
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		if atomic.LoadInt64(&cb.consecutiveFailures) >= int64(cb.config.MaxFailures) {
			cb.changeState(StateOpen)
		}
	case StateHalfOpen:
		cb.changeState(StateOpen)
	}

	// Notify callback
	if cb.config.OnFailure != nil {
		go cb.config.OnFailure(cb.config.Name, result.FailureType)
	}
}

// changeState changes the circuit breaker state
func (cb *EnterpriseCircuitBreaker) changeState(newState State) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.stateChangedAt = time.Now()

	// Reset counters for specific state transitions
	switch newState {
	case StateHalfOpen:
		atomic.StoreInt64(&cb.halfOpenCalls, 0)
	case StateClosed:
		atomic.StoreInt64(&cb.consecutiveFailures, 0)
	case StateOpen:
		atomic.StoreInt64(&cb.halfOpenCalls, 0)
	}

	atomic.AddInt64(&cb.metrics.StateChanges, 1)
	cb.metrics.LastStateChange = time.Now()

	cb.notifyStateChange(oldState, newState)
}

// notifyStateChange notifies about state changes
func (cb *EnterpriseCircuitBreaker) notifyStateChange(from, to State) {
	if cb.config.OnStateChange != nil {
		go cb.config.OnStateChange(cb.config.Name, from, to)
	}

	if cb.logger != nil {
		cb.logger.Info("Circuit breaker state changed",
			zap.String("name", cb.config.Name),
			zap.String("from", from.String()),
			zap.String("to", to.String()))
	}
}

// classifyFailure determines the type of failure
func (cb *EnterpriseCircuitBreaker) classifyFailure(err error, duration time.Duration) FailureType {
	if duration >= cb.config.Timeout {
		return FailureTypeTimeout
	}

	if duration >= cb.config.LatencyThreshold {
		return FailureTypeLatency
	}

	// Check error type
	errStr := err.Error()
	if contains(errStr, "timeout", "deadline") {
		return FailureTypeTimeout
	}
	if contains(errStr, "resource", "limit", "quota") {
		return FailureTypeResource
	}

	return FailureTypeError
}

// updateLatencyMetrics updates latency tracking
func (cb *EnterpriseCircuitBreaker) updateLatencyMetrics(duration time.Duration) {
	cb.metrics.mu.Lock()
	defer cb.metrics.mu.Unlock()

	// Update min/max
	if cb.metrics.MinLatency == 0 || duration < cb.metrics.MinLatency {
		cb.metrics.MinLatency = duration
	}
	if duration > cb.metrics.MaxLatency {
		cb.metrics.MaxLatency = duration
	}

	// Add to history for percentile calculation
	cb.latencyHistory = append(cb.latencyHistory, duration)
	if len(cb.latencyHistory) > 1000 {
		cb.latencyHistory = cb.latencyHistory[1:]
	}

	// Calculate average (simple moving average)
	if len(cb.latencyHistory) > 0 {
		var total time.Duration
		for _, d := range cb.latencyHistory {
			total += d
		}
		cb.metrics.AverageLatency = total / time.Duration(len(cb.latencyHistory))
	}
}

// adaptTierSettings adapts settings based on tier configuration
func (cb *EnterpriseCircuitBreaker) adaptTierSettings(tierConfig TierConfig) {
	cb.config.MaxFailures = tierConfig.FailureThreshold
	cb.config.ResetTimeout = tierConfig.ResetTimeout
	cb.config.HalfOpenMaxCalls = tierConfig.HalfOpenMaxCalls
}

// startBackgroundWorkers starts background maintenance workers
func (cb *EnterpriseCircuitBreaker) startBackgroundWorkers() {
	// Metrics aggregation worker
	cb.workerGroup.Add(1)
	go cb.metricsWorker()

	// Health monitoring worker
	if cb.config.EnableHealthScoring {
		cb.workerGroup.Add(1)
		go cb.healthWorker()
	}

	// Adaptive threshold worker
	if cb.config.EnableAdaptive {
		cb.workerGroup.Add(1)
		go cb.adaptiveWorker()
	}
}

// metricsWorker handles periodic metrics aggregation
func (cb *EnterpriseCircuitBreaker) metricsWorker() {
	defer cb.workerGroup.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cb.aggregateMetrics()
		case <-cb.shutdownChan:
			return
		}
	}
}

// healthWorker monitors circuit breaker health
func (cb *EnterpriseCircuitBreaker) healthWorker() {
	defer cb.workerGroup.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cb.checkHealth()
		case <-cb.shutdownChan:
			return
		}
	}
}

// adaptiveWorker handles adaptive threshold adjustments
func (cb *EnterpriseCircuitBreaker) adaptiveWorker() {
	defer cb.workerGroup.Done()

	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cb.adjustAdaptiveThreshold()
		case <-cb.shutdownChan:
			return
		}
	}
}

// aggregateMetrics performs periodic metrics aggregation
func (cb *EnterpriseCircuitBreaker) aggregateMetrics() {
	// Calculate percentiles from latency history
	if len(cb.latencyHistory) > 0 {
		sorted := make([]time.Duration, len(cb.latencyHistory))
		copy(sorted, cb.latencyHistory)
		
		// Simple sort for percentiles (could use more efficient algorithm)
		for i := 0; i < len(sorted); i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[i] > sorted[j] {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}

		cb.metrics.mu.Lock()
		if len(sorted) > 0 {
			cb.metrics.P50Latency = sorted[len(sorted)*50/100]
			cb.metrics.P95Latency = sorted[len(sorted)*95/100]
			cb.metrics.P99Latency = sorted[len(sorted)*99/100]
		}
		cb.metrics.mu.Unlock()
	}
}

// checkHealth performs health assessment
func (cb *EnterpriseCircuitBreaker) checkHealth() {
	if cb.healthScorer != nil {
		health := cb.healthScorer.CalculateHealth()
		
		cb.metrics.mu.Lock()
		cb.metrics.HealthScore = health
		cb.metrics.mu.Unlock()

		// Take action based on health score
		if health < cb.config.HealthThreshold {
			cb.logger.Warn("Circuit breaker health score low",
				zap.String("name", cb.config.Name),
				zap.Float64("health_score", health),
				zap.Float64("threshold", cb.config.HealthThreshold))
		}
	}
}

// adjustAdaptiveThreshold adjusts thresholds based on recent performance
func (cb *EnterpriseCircuitBreaker) adjustAdaptiveThreshold() {
	if cb.adaptiveThreshold != nil {
		newThreshold := cb.adaptiveThreshold.Adjust()
		
		cb.mu.Lock()
		cb.config.FailureThreshold = newThreshold
		cb.mu.Unlock()

		cb.logger.Debug("Adaptive threshold adjusted",
			zap.String("name", cb.config.Name),
			zap.Float64("new_threshold", newThreshold))
	}
}

// Utility functions

// validateConfig validates circuit breaker configuration
func validateConfig(cfg *Config) error {
	if cfg.Name == "" {
		return fmt.Errorf("name is required")
	}
	if cfg.MaxFailures <= 0 {
		return fmt.Errorf("max failures must be positive")
	}
	if cfg.ResetTimeout <= 0 {
		return fmt.Errorf("reset timeout must be positive")
	}
	if cfg.HalfOpenMaxCalls <= 0 {
		return fmt.Errorf("half open max calls must be positive")
	}
	if cfg.FailureThreshold < 0 || cfg.FailureThreshold > 1 {
		return fmt.Errorf("failure threshold must be between 0 and 1")
	}
	return nil
}

// newCircuitBreakerMetrics creates new metrics instance
func newCircuitBreakerMetrics() *CircuitBreakerMetrics {
	return &CircuitBreakerMetrics{
		TimeInState: make(map[State]time.Duration),
	}
}

// contains checks if any of the target strings are contained in the source
func contains(source string, targets ...string) bool {
	source = strings.ToLower(source)
	for _, target := range targets {
		if strings.Contains(source, strings.ToLower(target)) {
			return true
		}
	}
	return false
}
