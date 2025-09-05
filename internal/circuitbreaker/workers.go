package circuitbreaker

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"go.uber.org/zap"
)

// Background worker implementations

// startBackgroundWorkers initializes all background workers
func (cb *EnterpriseCircuitBreaker) startBackgroundWorkers() {
	// Start metrics collection worker
	cb.workerGroup.Add(1)
	go cb.metricsCollectionWorker()
	
	// Start health monitoring worker
	if cb.config.EnableHealthScoring {
		cb.workerGroup.Add(1)
		go cb.healthMonitoringWorker()
	}
	
	// Start adaptive threshold worker
	if cb.config.EnableAdaptive {
		cb.workerGroup.Add(1)
		go cb.adaptiveThresholdWorker()
	}
	
	// Start state management worker
	cb.workerGroup.Add(1)
	go cb.stateManagementWorker()
	
	// Start cleanup worker
	cb.workerGroup.Add(1)
	go cb.cleanupWorker()
}

// metricsCollectionWorker periodically collects and updates metrics
func (cb *EnterpriseCircuitBreaker) metricsCollectionWorker() {
	defer cb.workerGroup.Done()
	
	ticker := time.NewTicker(time.Second * 30) // Collect metrics every 30 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cb.collectMetrics()
			
		case <-cb.shutdownChan:
			return
			
		case <-cb.ctx.Done():
			return
		}
	}
}

// healthMonitoringWorker monitors system health and updates health scores
func (cb *EnterpriseCircuitBreaker) healthMonitoringWorker() {
	defer cb.workerGroup.Done()
	
	ticker := time.NewTicker(time.Minute) // Check health every minute
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cb.updateSystemHealth()
			
		case <-cb.shutdownChan:
			return
			
		case <-cb.ctx.Done():
			return
		}
	}
}

// adaptiveThresholdWorker manages adaptive threshold adjustments
func (cb *EnterpriseCircuitBreaker) adaptiveThresholdWorker() {
	defer cb.workerGroup.Done()
	
	ticker := time.NewTicker(time.Minute * 2) // Adjust thresholds every 2 minutes
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cb.performAdaptiveAdjustment()
			
		case <-cb.shutdownChan:
			return
			
		case <-cb.ctx.Done():
			return
		}
	}
}

// stateManagementWorker manages circuit breaker state transitions
func (cb *EnterpriseCircuitBreaker) stateManagementWorker() {
	defer cb.workerGroup.Done()
	
	ticker := time.NewTicker(time.Second * 10) // Check state every 10 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cb.evaluateStateTransitions()
			
		case <-cb.shutdownChan:
			return
			
		case <-cb.ctx.Done():
			return
		}
	}
}

// cleanupWorker performs periodic cleanup of old data
func (cb *EnterpriseCircuitBreaker) cleanupWorker() {
	defer cb.workerGroup.Done()
	
	ticker := time.NewTicker(time.Hour) // Cleanup every hour
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cb.performCleanup()
			
		case <-cb.shutdownChan:
			return
			
		case <-cb.ctx.Done():
			return
		}
	}
}

// collectMetrics gathers comprehensive system metrics
func (cb *EnterpriseCircuitBreaker) collectMetrics() {
	cb.metrics.mu.Lock()
	defer cb.metrics.mu.Unlock()
	
	// Update consecutive failure/success counters
	cb.metrics.ConsecutiveFailures = cb.consecutiveFailures
	cb.metrics.ConsecutiveSuccesses = cb.consecutiveSuccesses
	cb.metrics.LastFailureTime = cb.lastFailureTime
	cb.metrics.LastSuccessTime = cb.lastSuccessTime
	
	// Calculate current failure rate
	totalRequests := atomic.LoadInt64(&cb.metrics.TotalRequests)
	successfulRequests := atomic.LoadInt64(&cb.metrics.SuccessfulRequests)
	
	if totalRequests > 0 {
		cb.metrics.FailureRate = float64(totalRequests-successfulRequests) / float64(totalRequests)
	}
	
	// Update health score if enabled
	if cb.config.EnableHealthScoring {
		cb.metrics.HealthScore = cb.healthScorer.CalculateHealth()
	}
	
	// Log metrics if logger is configured
	if cb.logger != nil && cb.config.EnableMetrics {
		cb.logger.Debug("Circuit breaker metrics",
			zap.String("name", cb.config.Name),
			zap.String("state", cb.state.String()),
			zap.Int64("total_requests", totalRequests),
			zap.Int64("successful_requests", successfulRequests),
			zap.Float64("failure_rate", cb.metrics.FailureRate),
			zap.Float64("health_score", cb.metrics.HealthScore),
			zap.Duration("avg_latency", cb.metrics.AverageLatency),
		)
	}
}

// updateSystemHealth updates overall system health metrics
func (cb *EnterpriseCircuitBreaker) updateSystemHealth() {
	// Get current system resource utilization
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Calculate resource utilization
	resourceUtilization := float64(m.Alloc) / float64(m.Sys)
	if resourceUtilization > 1.0 {
		resourceUtilization = 1.0
	}
	
	// Get current statistics
	requests, failures, failureRate, avgLatency := cb.slidingWindow.GetStatistics()
	
	// Calculate throughput rate
	windowDuration := cb.config.WindowSize.Seconds()
	throughputRate := float64(requests) / windowDuration
	
	// Update health metrics
	healthMetrics := HealthMetrics{
		SuccessRate:         1.0 - failureRate,
		AverageLatency:      avgLatency,
		ErrorRate:           failureRate,
		ResourceUtilization: resourceUtilization,
		ThroughputRate:      throughputRate,
	}
	
	cb.healthScorer.UpdateMetrics(healthMetrics)
	
	// Check if health score indicates need for action
	healthScore := cb.healthScorer.CalculateHealth()
	
	if healthScore < cb.config.HealthThreshold && cb.state == StateClosed {
		if cb.logger != nil {
			cb.logger.Warn("Low health score detected",
				zap.String("name", cb.config.Name),
				zap.Float64("health_score", healthScore),
				zap.Float64("threshold", cb.config.HealthThreshold),
			)
		}
		
		// Consider preemptive circuit opening
		if healthScore < cb.config.HealthThreshold*0.5 {
			cb.mu.Lock()
			cb.setState(StateOpen)
			cb.mu.Unlock()
			
			if cb.logger != nil {
				cb.logger.Info("Circuit breaker opened due to low health score",
					zap.String("name", cb.config.Name),
					zap.Float64("health_score", healthScore),
				)
			}
		}
	}
}

// performAdaptiveAdjustment adjusts thresholds based on current performance
func (cb *EnterpriseCircuitBreaker) performAdaptiveAdjustment() {
	if !cb.config.EnableAdaptive {
		return
	}
	
	// Get current performance metrics
	_, _, failureRate, _ := cb.slidingWindow.GetStatistics()
	currentPerformance := 1.0 - failureRate
	
	// Adjust adaptive threshold
	newThreshold := cb.adaptiveThreshold.AdjustThreshold(currentPerformance)
	
	if cb.logger != nil {
		cb.logger.Debug("Adaptive threshold adjustment",
			zap.String("name", cb.config.Name),
			zap.Float64("current_performance", currentPerformance),
			zap.Float64("new_threshold", newThreshold),
			zap.Float64("failure_rate", failureRate),
		)
	}
}

// evaluateStateTransitions checks if state transitions are needed
func (cb *EnterpriseCircuitBreaker) evaluateStateTransitions() {
	cb.mu.RLock()
	currentState := cb.state
	stateTime := time.Since(cb.stateChangedAt)
	cb.mu.RUnlock()
	
	switch currentState {
	case StateOpen:
		// Check if we should attempt recovery
		if stateTime >= cb.config.ResetTimeout {
			recoveryProb := cb.calculateRecoveryProbability()
			
			if recoveryProb > 0.6 { // 60% threshold for automatic recovery attempt
				cb.mu.Lock()
				cb.setState(StateHalfOpen)
				cb.mu.Unlock()
				
				if cb.logger != nil {
					cb.logger.Info("Circuit breaker transitioning to half-open",
						zap.String("name", cb.config.Name),
						zap.Float64("recovery_probability", recoveryProb),
						zap.Duration("time_in_open", stateTime),
					)
				}
			}
		}
		
	case StateHalfOpen:
		// Check if half-open state has been active too long
		maxHalfOpenTime := cb.config.ResetTimeout * 2
		if stateTime >= maxHalfOpenTime {
			cb.mu.Lock()
			cb.setState(StateOpen)
			atomic.StoreInt64(&cb.halfOpenCalls, 0)
			cb.mu.Unlock()
			
			if cb.logger != nil {
				cb.logger.Info("Circuit breaker returned to open due to timeout",
					zap.String("name", cb.config.Name),
					zap.Duration("time_in_half_open", stateTime),
				)
			}
		}
	}
}

// performCleanup removes old data and optimizes memory usage
func (cb *EnterpriseCircuitBreaker) performCleanup() {
	// Cleanup latency history if it gets too large
	if len(cb.latencyHistory) > 2000 {
		// Keep only the most recent 1000 entries
		cb.latencyHistory = cb.latencyHistory[len(cb.latencyHistory)-1000:]
	}
	
	// Cleanup adaptive threshold history
	cb.adaptiveThreshold.mu.Lock()
	if len(cb.adaptiveThreshold.adjustmentHistory) > 200 {
		cb.adaptiveThreshold.adjustmentHistory = cb.adaptiveThreshold.adjustmentHistory[len(cb.adaptiveThreshold.adjustmentHistory)-100:]
	}
	if len(cb.adaptiveThreshold.performanceHistory) > 200 {
		cb.adaptiveThreshold.performanceHistory = cb.adaptiveThreshold.performanceHistory[len(cb.adaptiveThreshold.performanceHistory)-100:]
	}
	cb.adaptiveThreshold.mu.Unlock()
	
	// Force garbage collection periodically
	runtime.GC()
	
	if cb.logger != nil {
		cb.logger.Debug("Performed periodic cleanup",
			zap.String("name", cb.config.Name),
			zap.Int("latency_history_size", len(cb.latencyHistory)),
		)
	}
}

// Configuration validation

// validateConfig validates the circuit breaker configuration
func validateConfig(cfg *Config) error {
	if cfg.Name == "" {
		return fmt.Errorf("name is required")
	}
	
	if cfg.MaxFailures <= 0 {
		return fmt.Errorf("max_failures must be positive")
	}
	
	if cfg.ResetTimeout <= 0 {
		return fmt.Errorf("reset_timeout must be positive")
	}
	
	if cfg.HalfOpenMaxCalls <= 0 {
		return fmt.Errorf("half_open_max_calls must be positive")
	}
	
	if cfg.FailureThreshold < 0 || cfg.FailureThreshold > 1 {
		return fmt.Errorf("failure_threshold must be between 0 and 1")
	}
	
	if cfg.WindowSize <= 0 {
		cfg.WindowSize = time.Minute * 5 // Default window size
	}
	
	if cfg.MinRequestsThreshold <= 0 {
		cfg.MinRequestsThreshold = 10 // Default minimum requests
	}
	
	if cfg.EnableAdaptive {
		if cfg.AdaptiveMultiplier <= 0 {
			cfg.AdaptiveMultiplier = 1.5 // Default multiplier
		}
		
		if cfg.MaxAdaptiveTimeout <= 0 {
			cfg.MaxAdaptiveTimeout = cfg.ResetTimeout * 10 // Default max timeout
		}
	}
	
	if cfg.EnableHealthScoring {
		if cfg.HealthThreshold <= 0 || cfg.HealthThreshold > 1 {
			cfg.HealthThreshold = 0.7 // Default health threshold
		}
	}
	
	// Validate tier settings
	for tierName, tierConfig := range cfg.TierSettings {
		if tierConfig.FailureThreshold <= 0 {
			return fmt.Errorf("tier %s: failure_threshold must be positive", tierName)
		}
		
		if tierConfig.ResetTimeout <= 0 {
			return fmt.Errorf("tier %s: reset_timeout must be positive", tierName)
		}
		
		if tierConfig.HalfOpenMaxCalls <= 0 {
			return fmt.Errorf("tier %s: half_open_max_calls must be positive", tierName)
		}
	}
	
	return nil
}

// Default configurations for different tiers

// DefaultFreeConfig returns a default configuration for free tier
func DefaultFreeConfig(name string) Config {
	return Config{
		Name:                  name,
		MaxFailures:          3,
		ResetTimeout:         time.Minute * 2,
		HalfOpenMaxCalls:     2,
		Timeout:              time.Second * 30,
		Policy:               PolicyConservative,
		FailureThreshold:     0.5,
		LatencyThreshold:     time.Second * 10,
		WindowSize:           time.Minute * 5,
		MinRequestsThreshold: 5,
		EnableAdaptive:       false,
		EnableHealthScoring:  false,
		EnableMetrics:        true,
		TierSettings: map[string]TierConfig{
			"FREE": {
				FailureThreshold: 3,
				ResetTimeout:     time.Minute * 2,
				HalfOpenMaxCalls: 2,
				Priority:         1,
				QueueEnabled:     false,
				RetryEnabled:     true,
			},
		},
	}
}

// DefaultBusinessConfig returns a default configuration for business tier
func DefaultBusinessConfig(name string) Config {
	return Config{
		Name:                  name,
		MaxFailures:          10,
		ResetTimeout:         time.Second * 30,
		HalfOpenMaxCalls:     5,
		Timeout:              time.Second * 60,
		Policy:               PolicyStandard,
		FailureThreshold:     0.6,
		LatencyThreshold:     time.Second * 5,
		WindowSize:           time.Minute * 10,
		MinRequestsThreshold: 10,
		EnableAdaptive:       true,
		AdaptiveMultiplier:   1.5,
		MaxAdaptiveTimeout:   time.Minute * 15,
		EnableHealthScoring:  true,
		HealthThreshold:      0.7,
		EnableMetrics:        true,
		TierSettings: map[string]TierConfig{
			"BUSINESS": {
				FailureThreshold: 10,
				ResetTimeout:     time.Second * 30,
				HalfOpenMaxCalls: 5,
				Priority:         2,
				QueueEnabled:     true,
				RetryEnabled:     true,
			},
		},
	}
}

// DefaultEnterpriseConfig returns a default configuration for enterprise tier
func DefaultEnterpriseConfig(name string) Config {
	return Config{
		Name:                  name,
		MaxFailures:          20,
		ResetTimeout:         time.Second * 15,
		HalfOpenMaxCalls:     10,
		Timeout:              time.Second * 120,
		Policy:               PolicyAdaptive,
		FailureThreshold:     0.7,
		LatencyThreshold:     time.Second * 2,
		WindowSize:           time.Minute * 15,
		MinRequestsThreshold: 20,
		EnableAdaptive:       true,
		AdaptiveMultiplier:   2.0,
		MaxAdaptiveTimeout:   time.Minute * 30,
		EnableHealthScoring:  true,
		HealthThreshold:      0.8,
		EnableMetrics:        true,
		TierSettings: map[string]TierConfig{
			"ENTERPRISE": {
				FailureThreshold: 20,
				ResetTimeout:     time.Second * 15,
				HalfOpenMaxCalls: 10,
				Priority:         3,
				QueueEnabled:     true,
				RetryEnabled:     true,
			},
		},
	}
}
