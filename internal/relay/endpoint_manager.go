package relay

import (
	"math/rand"
	"sync"
	"time"
)

// EndpointHealth tracks the health and performance metrics of an endpoint
type EndpointHealth struct {
	URL              string
	LastSeen         time.Time
	LastError        time.Time
	ErrorCount       int
	SuccessCount     int
	ResponseTimes    []time.Duration
	MaxResponseTimes int
	CircuitOpen      bool
	CircuitOpenUntil time.Time
	Weight           float64 // Weight for weighted selection
	mu               sync.RWMutex
}

// NewEndpointHealth creates a new endpoint health tracker
func NewEndpointHealth(url string) *EndpointHealth {
	return &EndpointHealth{
		URL:              url,
		LastSeen:         time.Now(),
		ResponseTimes:    make([]time.Duration, 0, 10),
		MaxResponseTimes: 10,
		Weight:           1.0, // Start with equal weighting
	}
}

// RecordSuccess records a successful operation against this endpoint
func (eh *EndpointHealth) RecordSuccess(responseTime time.Duration) {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	eh.LastSeen = time.Now()
	eh.SuccessCount++

	// Record response time with moving window
	eh.ResponseTimes = append(eh.ResponseTimes, responseTime)
	if len(eh.ResponseTimes) > eh.MaxResponseTimes {
		eh.ResponseTimes = eh.ResponseTimes[1:]
	}

	// Recalculate weight based on performance (lower response time = higher weight)
	if len(eh.ResponseTimes) > 0 {
		avgTime := eh.getAverageResponseTime()
		// Scale weight inversely with response time: faster endpoints get higher weight
		// Add small constant to avoid division by zero and normalize weight range
		eh.Weight = 1.0 / (float64(avgTime.Milliseconds()) + 50.0) * 1000.0

		// Ensure minimum weight
		if eh.Weight < 0.1 {
			eh.Weight = 0.1
		}
	}

	// Reset circuit breaker if it was open
	if eh.CircuitOpen && time.Now().After(eh.CircuitOpenUntil) {
		eh.CircuitOpen = false
	}
}

// RecordError records an error from this endpoint
func (eh *EndpointHealth) RecordError() {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	now := time.Now()
	eh.LastError = now
	eh.ErrorCount++

	// Implement circuit breaker pattern
	// Open circuit after consecutive errors with exponential timeout
	if eh.SuccessCount == 0 || float64(eh.ErrorCount)/float64(eh.SuccessCount+1) > 0.5 {
		backoffTime := time.Duration(int(1<<uint(min(eh.ErrorCount, 8)))) * time.Second
		eh.CircuitOpen = true
		eh.CircuitOpenUntil = now.Add(backoffTime)
		eh.Weight = eh.Weight * 0.5 // Reduce weight on errors
		if eh.Weight < 0.1 {
			eh.Weight = 0.1 // Minimum weight to ensure eventual retry
		}
	}
}

// IsHealthy returns whether the endpoint is considered healthy
func (eh *EndpointHealth) IsHealthy() bool {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	if eh.CircuitOpen && time.Now().Before(eh.CircuitOpenUntil) {
		return false
	}

	// Consider healthy if:
	// 1. No errors or more successes than errors
	// 2. Last seen recently (within last 60 seconds)
	return (eh.ErrorCount == 0 || eh.SuccessCount > eh.ErrorCount) &&
		time.Since(eh.LastSeen) < 60*time.Second
}

// GetWeight returns the current weight of this endpoint for selection
func (eh *EndpointHealth) GetWeight() float64 {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	// Return zero weight if circuit is open
	if eh.CircuitOpen && time.Now().Before(eh.CircuitOpenUntil) {
		return 0.0
	}

	return eh.Weight
}

// getAverageResponseTime calculates the average response time
func (eh *EndpointHealth) getAverageResponseTime() time.Duration {
	if len(eh.ResponseTimes) == 0 {
		return 0
	}

	var total time.Duration
	for _, t := range eh.ResponseTimes {
		total += t
	}
	return total / time.Duration(len(eh.ResponseTimes))
}

// EndpointSelector implements intelligent endpoint selection strategies
type EndpointSelector struct {
	endpoints map[string]*EndpointHealth
	mu        sync.RWMutex
}

// NewEndpointSelector creates a new endpoint selector
func NewEndpointSelector(initialEndpoints []string) *EndpointSelector {
	selector := &EndpointSelector{
		endpoints: make(map[string]*EndpointHealth),
	}

	// Initialize with default endpoints
	for _, url := range initialEndpoints {
		selector.AddEndpoint(url)
	}

	return selector
}

// AddEndpoint adds a new endpoint to the selector
func (es *EndpointSelector) AddEndpoint(url string) {
	es.mu.Lock()
	defer es.mu.Unlock()

	if _, exists := es.endpoints[url]; !exists {
		es.endpoints[url] = NewEndpointHealth(url)
	}
}

// RemoveEndpoint removes an endpoint from the selector
func (es *EndpointSelector) RemoveEndpoint(url string) {
	es.mu.Lock()
	defer es.mu.Unlock()

	delete(es.endpoints, url)
}

// RecordSuccess records a successful operation against the endpoint
func (es *EndpointSelector) RecordSuccess(url string, responseTime time.Duration) {
	es.mu.RLock()
	eh, exists := es.endpoints[url]
	es.mu.RUnlock()

	if exists {
		eh.RecordSuccess(responseTime)
	}
}

// RecordError records an error from the endpoint
func (es *EndpointSelector) RecordError(url string) {
	es.mu.RLock()
	eh, exists := es.endpoints[url]
	es.mu.RUnlock()

	if exists {
		eh.RecordError()
	}
}

// GetBestEndpoint selects the best endpoint based on weighted probability
func (es *EndpointSelector) GetBestEndpoint() string {
	es.mu.RLock()
	defer es.mu.RUnlock()

	// Handle empty case
	if len(es.endpoints) == 0 {
		return ""
	}

	// Calculate total weight
	var totalWeight float64
	weights := make(map[string]float64)

	for url, eh := range es.endpoints {
		w := eh.GetWeight()
		weights[url] = w
		totalWeight += w
	}

	// If no viable endpoints (all weights are 0), reset all weights to minimum
	if totalWeight <= 0 {
		for url := range es.endpoints {
			weights[url] = 0.1
			totalWeight += 0.1
		}
	}

	// Weighted random selection
	r := rand.Float64() * totalWeight
	var cumulativeWeight float64

	for url, weight := range weights {
		cumulativeWeight += weight
		if r <= cumulativeWeight {
			return url
		}
	}

	// Fallback - use first endpoint
	for url := range es.endpoints {
		return url
	}

	return ""
}

// GetAllEndpoints returns all endpoints with their health status
func (es *EndpointSelector) GetAllEndpoints() map[string]bool {
	es.mu.RLock()
	defer es.mu.RUnlock()

	result := make(map[string]bool)
	for url, eh := range es.endpoints {
		result[url] = eh.IsHealthy()
	}

	return result
}

// GetHealthyEndpoints returns all healthy endpoints
func (es *EndpointSelector) GetHealthyEndpoints() []string {
	es.mu.RLock()
	defer es.mu.RUnlock()

	var healthy []string
	for url, eh := range es.endpoints {
		if eh.IsHealthy() {
			healthy = append(healthy, url)
		}
	}

	return healthy
}

// Helper function since Go < 1.21 doesn't have min for integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
