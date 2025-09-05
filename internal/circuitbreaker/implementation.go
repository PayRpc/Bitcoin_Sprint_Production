package circuitbreaker

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Core implementation methods for EnterpriseCircuitBreaker

// allowRequest determines if a request should be allowed through the circuit breaker
func (cb *EnterpriseCircuitBreaker) allowRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	// Check for forced states
	if cb.forceState != nil {
		switch *cb.forceState {
		case StateForceOpen:
			return false
		case StateForceClose:
			return true
		}
	}
	
	// Check maintenance mode
	if cb.maintenanceMode {
		return false
	}
	
	switch cb.state {
	case StateClosed:
		return true
		
	case StateOpen:
		// Check if enough time has passed to attempt reset
		if time.Since(cb.stateChangedAt) >= cb.config.ResetTimeout {
			// Check recovery probability
			recoveryProb := cb.calculateRecoveryProbability()
			if recoveryProb > 0.5 { // 50% threshold for recovery attempt
				cb.setState(StateHalfOpen)
				return true
			}
		}
		return false
		
	case StateHalfOpen:
		// Allow limited requests in half-open state
		return atomic.LoadInt64(&cb.halfOpenCalls) < int64(cb.config.HalfOpenMaxCalls)
		
	default:
		return false
	}
}

// executeWithMonitoring executes the function with comprehensive monitoring
func (cb *EnterpriseCircuitBreaker) executeWithMonitoring(ctx context.Context, fn func() (interface{}, error), startTime time.Time) *ExecutionResult {
	// Increment half-open calls if in half-open state
	if cb.state == StateHalfOpen {
		atomic.AddInt64(&cb.halfOpenCalls, 1)
	}
	
	// Create timeout context if configured
	if cb.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cb.config.Timeout)
		defer cancel()
	}
	
	// Execute with timeout handling
	resultChan := make(chan executionContext, 1)
	
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- executionContext{
					result: nil,
					err:    fmt.Errorf("panic recovered: %v", r),
					panic:  true,
				}
			}
		}()
		
		result, err := fn()
		resultChan <- executionContext{
			result: result,
			err:    err,
			panic:  false,
		}
	}()
	
	// Wait for result or timeout
	select {
	case execResult := <-resultChan:
		duration := time.Since(startTime)
		
		// Determine success and failure type
		success := execResult.err == nil && !execResult.panic
		failureType := cb.determineFailureType(execResult.err, duration, execResult.panic)
		
		// Check latency-based failure detection
		if success && cb.config.LatencyThreshold > 0 && duration > cb.config.LatencyThreshold {
			success = false
			failureType = FailureTypeLatency
		}
		
		return &ExecutionResult{
			Success:     success,
			Duration:    duration,
			Error:       execResult.err,
			State:       cb.state,
			FailureType: failureType,
			Attempt:     1,
			Metadata:    map[string]interface{}{
				"panic":         execResult.panic,
				"execution_id":  generateExecutionID(),
				"tier":          cb.currentTier,
			},
		}
		
	case <-ctx.Done():
		duration := time.Since(startTime)
		atomic.AddInt64(&cb.metrics.TimeoutRequests, 1)
		
		return &ExecutionResult{
			Success:     false,
			Duration:    duration,
			Error:       ctx.Err(),
			State:       cb.state,
			FailureType: FailureTypeTimeout,
			Attempt:     1,
			Metadata:    map[string]interface{}{
				"timeout":       true,
				"execution_id":  generateExecutionID(),
				"tier":          cb.currentTier,
			},
		}
	}
}

// recordResult records the execution result and updates circuit breaker state
func (cb *EnterpriseCircuitBreaker) recordResult(result *ExecutionResult) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	// Update basic metrics
	atomic.AddInt64(&cb.metrics.TotalRequests, 1)
	
	if result.Success {
		atomic.AddInt64(&cb.metrics.SuccessfulRequests, 1)
		cb.consecutiveSuccesses++
		cb.consecutiveFailures = 0
		cb.lastSuccessTime = time.Now()
		
		// Update sliding window
		cb.slidingWindow.AddRequest(true, result.Duration)
		
		// Handle state transitions for successful requests
		cb.handleSuccessfulRequest()
		
	} else {
		atomic.AddInt64(&cb.metrics.FailedRequests, 1)
		cb.consecutiveFailures++
		cb.consecutiveSuccesses = 0
		cb.lastFailureTime = time.Now()
		
		// Update sliding window
		cb.slidingWindow.AddRequest(false, result.Duration)
		
		// Handle state transitions for failed requests
		cb.handleFailedRequest(result.FailureType)
		
		// Notify failure callback
		if cb.config.OnFailure != nil {
			go cb.config.OnFailure(cb.config.Name, result.FailureType)
		}
	}
	
	// Update latency metrics
	cb.updateLatencyMetrics(result.Duration)
	
	// Update health scoring
	if cb.config.EnableHealthScoring {
		cb.updateHealthMetrics()
	}
	
	// Update adaptive threshold
	if cb.config.EnableAdaptive {
		cb.updateAdaptiveThreshold()
	}
}

// handleSuccessfulRequest manages state transitions for successful requests
func (cb *EnterpriseCircuitBreaker) handleSuccessfulRequest() {
	switch cb.state {
	case StateHalfOpen:
		// Check if we should transition back to closed
		if cb.consecutiveSuccesses >= int64(cb.config.HalfOpenMaxCalls) {
			cb.setState(StateClosed)
			atomic.StoreInt64(&cb.halfOpenCalls, 0)
			
			// Record recovery
			recoveryTime := time.Since(cb.stateChangedAt)
			cb.metrics.RecoveryTime = recoveryTime
			
			if cb.config.OnRecovery != nil {
				go cb.config.OnRecovery(cb.config.Name, recoveryTime)
			}
		}
	}
}

// handleFailedRequest manages state transitions for failed requests
func (cb *EnterpriseCircuitBreaker) handleFailedRequest(failureType FailureType) {
	switch cb.state {
	case StateClosed:
		// Check if we should transition to open
		if cb.shouldTripCircuit() {
			cb.setState(StateOpen)
		}
		
	case StateHalfOpen:
		// Any failure in half-open state trips the circuit
		cb.setState(StateOpen)
		atomic.StoreInt64(&cb.halfOpenCalls, 0)
	}
}

// shouldTripCircuit determines if the circuit should trip based on current conditions
func (cb *EnterpriseCircuitBreaker) shouldTripCircuit() bool {
	// Get current failure statistics
	requests, _, failureRate, _ := cb.slidingWindow.GetStatistics()
	
	// Check minimum request threshold
	if requests < int64(cb.config.MinRequestsThreshold) {
		return false
	}
	
	// Get current threshold (adaptive or fixed)
	threshold := cb.config.FailureThreshold
	if cb.config.EnableAdaptive {
		threshold = cb.adaptiveThreshold.AdjustThreshold(1.0 - failureRate)
	}
	
	// Check failure rate against threshold
	if failureRate >= threshold {
		return true
	}
	
	// Check consecutive failures for tier-based logic
	tierConfig := cb.getTierConfig()
	if cb.consecutiveFailures >= int64(tierConfig.FailureThreshold) {
		return true
	}
	
	// Check health score if enabled
	if cb.config.EnableHealthScoring {
		healthScore := cb.healthScorer.CalculateHealth()
		if healthScore < cb.config.HealthThreshold {
			return true
		}
	}
	
	return false
}

// setState safely changes the circuit breaker state
func (cb *EnterpriseCircuitBreaker) setState(newState State) {
	oldState := cb.state
	cb.state = newState
	cb.stateChangedAt = time.Now()
	
	// Update metrics
	atomic.AddInt64(&cb.metrics.StateChanges, 1)
	cb.metrics.LastStateChange = cb.stateChangedAt
	
	// Update time in state tracking
	if cb.metrics.TimeInState == nil {
		cb.metrics.TimeInState = make(map[State]time.Duration)
	}
	
	if oldState != newState {
		timeDiff := time.Since(cb.stateChangedAt)
		cb.metrics.TimeInState[oldState] += timeDiff
		
		// Notify state change callback
		cb.notifyStateChange(oldState, newState)
	}
}

// notifyStateChange calls the state change callback if configured
func (cb *EnterpriseCircuitBreaker) notifyStateChange(from, to State) {
	if cb.config.OnStateChange != nil {
		go cb.config.OnStateChange(cb.config.Name, from, to)
	}
	
	if cb.logger != nil {
		cb.logger.Info("Circuit breaker state changed",
			zap.String("name", cb.config.Name),
			zap.String("from", from.String()),
			zap.String("to", to.String()),
			zap.Time("timestamp", time.Now()),
		)
	}
}

// updateLatencyMetrics updates latency-related metrics
func (cb *EnterpriseCircuitBreaker) updateLatencyMetrics(duration time.Duration) {
	// Add to latency history
	cb.latencyHistory = append(cb.latencyHistory, duration)
	if len(cb.latencyHistory) > 1000 {
		cb.latencyHistory = cb.latencyHistory[1:]
	}
	
	// Update min/max latency
	if duration > cb.metrics.MaxLatency {
		cb.metrics.MaxLatency = duration
	}
	
	if cb.metrics.MinLatency == 0 || duration < cb.metrics.MinLatency {
		cb.metrics.MinLatency = duration
	}
	
	// Calculate percentiles if we have enough data
	if len(cb.latencyHistory) >= 10 {
		cb.metrics.P50Latency = calculatePercentile(cb.latencyHistory, 50)
		cb.metrics.P95Latency = calculatePercentile(cb.latencyHistory, 95)
		cb.metrics.P99Latency = calculatePercentile(cb.latencyHistory, 99)
		
		// Calculate average latency
		total := time.Duration(0)
		for _, lat := range cb.latencyHistory {
			total += lat
		}
		cb.metrics.AverageLatency = total / time.Duration(len(cb.latencyHistory))
	}
}

// updateHealthMetrics updates health scoring metrics
func (cb *EnterpriseCircuitBreaker) updateHealthMetrics() {
	totalRequests := atomic.LoadInt64(&cb.metrics.TotalRequests)
	successfulRequests := atomic.LoadInt64(&cb.metrics.SuccessfulRequests)
	
	if totalRequests > 0 {
		successRate := float64(successfulRequests) / float64(totalRequests)
		errorRate := 1.0 - successRate
		
		// Get current resource utilization
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		resourceUtilization := float64(m.Alloc) / float64(m.Sys)
		
		// Calculate throughput rate (requests per second)
		windowStats := cb.slidingWindow
		requests, _, _, _ := windowStats.GetStatistics()
		throughputRate := float64(requests) / cb.config.WindowSize.Seconds()
		
		healthMetrics := HealthMetrics{
			SuccessRate:         successRate,
			AverageLatency:      cb.metrics.AverageLatency,
			ErrorRate:           errorRate,
			ResourceUtilization: resourceUtilization,
			ThroughputRate:      throughputRate,
		}
		
		cb.healthScorer.UpdateMetrics(healthMetrics)
	}
}

// updateAdaptiveThreshold updates the adaptive threshold based on current performance
func (cb *EnterpriseCircuitBreaker) updateAdaptiveThreshold() {
	_, _, failureRate, _ := cb.slidingWindow.GetStatistics()
	currentPerformance := 1.0 - failureRate
	cb.adaptiveThreshold.AdjustThreshold(currentPerformance)
}

// calculateRecoveryProbability calculates the probability of successful recovery
func (cb *EnterpriseCircuitBreaker) calculateRecoveryProbability() float64 {
	// Base calculation on time since last failure and consecutive failures
	timeSinceFailure := time.Since(cb.lastFailureTime)
	
	// Get tier-specific recovery parameters
	tierConfig := cb.getTierConfig()
	baseTimeout := tierConfig.ResetTimeout
	
	// Time factor: longer time since failure increases probability
	timeFactor := math.Min(1.0, float64(timeSinceFailure)/float64(baseTimeout))
	
	// Failure factor: more consecutive failures decrease probability
	failureFactor := math.Pow(0.8, float64(cb.consecutiveFailures))
	
	// Health factor: better health score increases probability
	healthFactor := 1.0
	if cb.config.EnableHealthScoring {
		healthFactor = cb.healthScorer.CalculateHealth()
	}
	
	// Combined probability
	probability := timeFactor * failureFactor * healthFactor
	
	return math.Max(0.1, math.Min(0.9, probability))
}

// getTierConfig returns the appropriate tier configuration
func (cb *EnterpriseCircuitBreaker) getTierConfig() TierConfig {
	if cb.currentTier != "" {
		if tierConfig, exists := cb.tierConfigs[cb.currentTier]; exists {
			return tierConfig
		}
	}
	
	// Return default configuration
	return TierConfig{
		FailureThreshold: cb.config.MaxFailures,
		ResetTimeout:     cb.config.ResetTimeout,
		HalfOpenMaxCalls: cb.config.HalfOpenMaxCalls,
		Priority:         1,
		QueueEnabled:     false,
		RetryEnabled:     true,
	}
}

// adaptTierSettings adapts circuit breaker settings based on tier configuration
func (cb *EnterpriseCircuitBreaker) adaptTierSettings(tierConfig TierConfig) {
	cb.config.MaxFailures = tierConfig.FailureThreshold
	cb.config.ResetTimeout = tierConfig.ResetTimeout
	cb.config.HalfOpenMaxCalls = tierConfig.HalfOpenMaxCalls
}

// determineFailureType categorizes the type of failure that occurred
func (cb *EnterpriseCircuitBreaker) determineFailureType(err error, duration time.Duration, panic bool) FailureType {
	if panic {
		return FailureTypeError
	}
	
	if err != nil {
		// Check if it's a timeout error
		if err == context.DeadlineExceeded || err == context.Canceled {
			return FailureTypeTimeout
		}
		
		// Check error message for specific patterns
		errMsg := err.Error()
		switch {
		case contains(errMsg, "timeout"):
			return FailureTypeTimeout
		case contains(errMsg, "resource"):
			return FailureTypeResource
		case contains(errMsg, "circuit"):
			return FailureTypeCircuit
		default:
			return FailureTypeError
		}
	}
	
	// Check for latency-based failure
	if cb.config.LatencyThreshold > 0 && duration > cb.config.LatencyThreshold {
		return FailureTypeLatency
	}
	
	return FailureTypeError
}

// Helper types and functions
type executionContext struct {
	result interface{}
	err    error
	panic  bool
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr
}

func generateExecutionID() string {
	return fmt.Sprintf("exec_%d", time.Now().UnixNano())
}

// newCircuitBreakerMetrics creates a new metrics instance
func newCircuitBreakerMetrics() *CircuitBreakerMetrics {
	return &CircuitBreakerMetrics{
		TimeInState: make(map[State]time.Duration),
		MinLatency:  time.Hour, // High initial value
	}
}
