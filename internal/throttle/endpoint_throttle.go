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
	URL           string
	SuccessCount  int64
	FailureCount  int64
	LastSuccess   time.Time
	LastFailure   time.Time
	NextRetry     time.Time
	CurrentBackoff time.Duration
	SuccessRate   float64
}

// ThrottleConfig holds throttling configuration
type ThrottleConfig struct {
	MinSuccessRate    float64       // Minimum success rate to prefer endpoint (0.90 = 90%)
	InitialBackoff    time.Duration // Initial backoff duration
	MaxBackoff        time.Duration // Maximum backoff duration
	BackoffMultiplier float64       // Exponential backoff multiplier
	HealthCheckWindow int64         // Number of recent requests to consider for health
}

// EndpointThrottle manages endpoint throttling and health tracking
type EndpointThrottle struct {
	config    *ThrottleConfig
	endpoints map[string]*EndpointStatus
	mu        sync.RWMutex
	logger    *zap.Logger
}

// DefaultThrottleConfig returns optimized throttling settings
func DefaultThrottleConfig() *ThrottleConfig {
	return &ThrottleConfig{
		MinSuccessRate:    0.90,              // Prefer endpoints with ≥90% success rate
		InitialBackoff:    10 * time.Minute,  // Start with 10m backoff
		MaxBackoff:        30 * time.Minute,  // Max 30m backoff
		BackoffMultiplier: 1.5,               // Moderate exponential increase
		HealthCheckWindow: 100,               // Consider last 100 requests
	}
}

// New creates a new endpoint throttle manager
func New(logger *zap.Logger) *EndpointThrottle {
	return &EndpointThrottle{
		config:    DefaultThrottleConfig(),
		endpoints: make(map[string]*EndpointStatus),
		logger:    logger,
	}
}

// RegisterEndpoint adds an endpoint to the throttle manager
func (et *EndpointThrottle) RegisterEndpoint(url string) {
	et.mu.Lock()
	defer et.mu.Unlock()
	
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
	status.NextRetry = time.Time{} // Clear retry delay
	
	et.updateSuccessRate(status)
	
	et.logger.Debug("Recorded success", 
		zap.String("url", url),
		zap.Float64("success_rate", status.SuccessRate),
		zap.Int64("success_count", status.SuccessCount),
	)
}

// RecordFailure records a failed request to an endpoint
func (et *EndpointThrottle) RecordFailure(url string, err error) {
	et.mu.Lock()
	defer et.mu.Unlock()
	
	status, exists := et.endpoints[url]
	if !exists {
		return
	}
	
	status.FailureCount++
	status.LastFailure = time.Now()
	
	// Calculate exponential backoff
	status.CurrentBackoff = time.Duration(float64(status.CurrentBackoff) * et.config.BackoffMultiplier)
	if status.CurrentBackoff > et.config.MaxBackoff {
		status.CurrentBackoff = et.config.MaxBackoff
	}
	
	status.NextRetry = time.Now().Add(status.CurrentBackoff)
	
	et.updateSuccessRate(status)
	
	et.logger.Warn("Recorded failure",
		zap.String("url", url),
		zap.Error(err),
		zap.Float64("success_rate", status.SuccessRate),
		zap.Duration("backoff", status.CurrentBackoff),
		zap.Time("next_retry", status.NextRetry),
	)
}

// updateSuccessRate calculates the current success rate for an endpoint
func (et *EndpointThrottle) updateSuccessRate(status *EndpointStatus) {
	total := status.SuccessCount + status.FailureCount
	if total == 0 {
		status.SuccessRate = 1.0
		return
	}
	
	// Limit to recent requests for health window
	if total > et.config.HealthCheckWindow {
		// For simplicity, use overall ratio. In production, you'd track a sliding window
		status.SuccessRate = float64(status.SuccessCount) / float64(total)
	} else {
		status.SuccessRate = float64(status.SuccessCount) / float64(total)
	}
}

// GetBestEndpoint returns the best available endpoint based on success rate and throttling
func (et *EndpointThrottle) GetBestEndpoint(endpoints []string) (string, error) {
	et.mu.RLock()
	defer et.mu.RUnlock()
	
	now := time.Now()
	var bestEndpoint string
	var bestScore float64 = -1
	
	for _, url := range endpoints {
		status, exists := et.endpoints[url]
		if !exists {
			// Unknown endpoint - register it and consider it viable
			et.mu.RUnlock()
			et.RegisterEndpoint(url)
			et.mu.RLock()
			status = et.endpoints[url]
		}
		
		// Skip endpoints that are in backoff
		if !status.NextRetry.IsZero() && now.Before(status.NextRetry) {
			et.logger.Debug("Endpoint in backoff", 
				zap.String("url", url),
				zap.Duration("remaining", status.NextRetry.Sub(now)),
			)
			continue
		}
		
		// Calculate score based on success rate and recency
		score := et.calculateEndpointScore(status, now)
		
		if score > bestScore {
			bestScore = score
			bestEndpoint = url
		}
	}
	
	if bestEndpoint == "" {
		return "", fmt.Errorf("no available endpoints (all in backoff)")
	}
	
	et.logger.Debug("Selected best endpoint",
		zap.String("url", bestEndpoint),
		zap.Float64("score", bestScore),
	)
	
	return bestEndpoint, nil
}

// calculateEndpointScore computes a score for endpoint selection
func (et *EndpointThrottle) calculateEndpointScore(status *EndpointStatus, now time.Time) float64 {
	// Base score from success rate
	score := status.SuccessRate
	
	// Bonus for high success rate endpoints
	if status.SuccessRate >= et.config.MinSuccessRate {
		score += 0.1 // 10% bonus for meeting minimum success rate
	}
	
	// Bonus for recent successful activity
	timeSinceSuccess := now.Sub(status.LastSuccess)
	if timeSinceSuccess < time.Hour {
		// Recent success gets a small bonus
		recencyBonus := math.Max(0, 0.05*(1.0-timeSinceSuccess.Hours()))
		score += recencyBonus
	}
	
	// Penalty for recent failures
	if !status.LastFailure.IsZero() {
		timeSinceFailure := now.Sub(status.LastFailure)
		if timeSinceFailure < time.Hour {
			failurePenalty := math.Max(0, 0.1*(1.0-timeSinceFailure.Hours()))
			score -= failurePenalty
		}
	}
	
	return math.Max(0, score) // Ensure score is non-negative
}

// GetEndpointStatus returns the current status of an endpoint
func (et *EndpointThrottle) GetEndpointStatus(url string) (*EndpointStatus, bool) {
	et.mu.RLock()
	defer et.mu.RUnlock()
	
	status, exists := et.endpoints[url]
	if !exists {
		return nil, false
	}
	
	// Return a copy to avoid races
	statusCopy := *status
	return &statusCopy, true
}

// GetAllStatuses returns status for all registered endpoints
func (et *EndpointThrottle) GetAllStatuses() map[string]*EndpointStatus {
	et.mu.RLock()
	defer et.mu.RUnlock()
	
	result := make(map[string]*EndpointStatus, len(et.endpoints))
	for url, status := range et.endpoints {
		statusCopy := *status
		result[url] = &statusCopy
	}
	
	return result
}

// IsEndpointHealthy checks if an endpoint meets the minimum health requirements
func (et *EndpointThrottle) IsEndpointHealthy(url string) bool {
	status, exists := et.GetEndpointStatus(url)
	if !exists {
		return false
	}
	
	// Check if endpoint is not in backoff and meets minimum success rate
	now := time.Now()
	notInBackoff := status.NextRetry.IsZero() || now.After(status.NextRetry)
	meetsSuccessRate := status.SuccessRate >= et.config.MinSuccessRate
	
	return notInBackoff && meetsSuccessRate
}

// ResetEndpoint clears the failure history for an endpoint (for manual recovery)
func (et *EndpointThrottle) ResetEndpoint(url string) {
	et.mu.Lock()
	defer et.mu.Unlock()
	
	if status, exists := et.endpoints[url]; exists {
		status.FailureCount = 0
		status.CurrentBackoff = et.config.InitialBackoff
		status.NextRetry = time.Time{}
		status.LastFailure = time.Time{}
		et.updateSuccessRate(status)
		
		et.logger.Info("Reset endpoint", zap.String("url", url))
	}
}
