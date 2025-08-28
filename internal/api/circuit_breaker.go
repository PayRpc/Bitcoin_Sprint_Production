// Package api provides circuit breaker functionality
package api

import (
	"fmt"
	"sync"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
)

// ===== CIRCUIT BREAKER IMPLEMENTATION =====

// CircuitBreaker implements a tier-aware circuit breaker pattern
type CircuitBreaker struct {
	tier              config.Tier
	state             string
	failureCount      int64
	failureThreshold  int64
	lastFailureTime   time.Time
	resetTimeout      time.Duration
	halfOpenMaxCalls  int64
	halfOpenCallCount int64
	clock             Clock
	mu                sync.RWMutex
}

// NewCircuitBreaker creates a tier-aware circuit breaker
func NewCircuitBreaker(tier config.Tier, clock Clock) *CircuitBreaker {
	var failureThreshold int64 = 5
	var resetTimeout = 60 * time.Second
	var halfOpenMaxCalls int64 = 3

	// Tier-specific configuration
	switch tier {
	case config.TierFree:
		failureThreshold = 3 // Fail fast for free tier
		resetTimeout = 120 * time.Second
		halfOpenMaxCalls = 1
	case config.TierPro, config.TierBusiness:
		failureThreshold = 10
		resetTimeout = 30 * time.Second
		halfOpenMaxCalls = 5
	case config.TierTurbo, config.TierEnterprise:
		failureThreshold = 20 // Higher tolerance for premium tiers
		resetTimeout = 15 * time.Second
		halfOpenMaxCalls = 10
	}

	return &CircuitBreaker{
		tier:             tier,
		state:            "closed",
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		halfOpenMaxCalls: halfOpenMaxCalls,
		clock:            clock,
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := cb.clock.Now()

	// Check if circuit should transition from open to half-open
	if cb.state == "open" && now.Sub(cb.lastFailureTime) > cb.resetTimeout {
		cb.state = "half-open"
		cb.halfOpenCallCount = 0
	}

	// Reject calls if circuit is open
	if cb.state == "open" {
		return fmt.Errorf("circuit breaker is open")
	}

	// Limit calls in half-open state
	if cb.state == "half-open" {
		if cb.halfOpenCallCount >= cb.halfOpenMaxCalls {
			return fmt.Errorf("circuit breaker half-open call limit reached")
		}
		cb.halfOpenCallCount++
	}

	// Execute the function
	err := fn()

	if err != nil {
		cb.recordFailure(now)
		return err
	}

	// Success - reset circuit if it was half-open
	if cb.state == "half-open" {
		cb.state = "closed"
		cb.failureCount = 0
	}

	return nil
}

// recordFailure records a failure and potentially opens the circuit
func (cb *CircuitBreaker) recordFailure(now time.Time) {
	cb.failureCount++
	cb.lastFailureTime = now

	if cb.failureCount >= cb.failureThreshold {
		cb.state = "open"
	}
}

// ShouldQueue determines if requests should be queued based on tier
func (cb *CircuitBreaker) ShouldQueue() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.tier {
	case config.TierFree:
		return false // Drop excess requests immediately
	case config.TierPro, config.TierBusiness:
		return cb.state != "open" // Queue if circuit is not open
	case config.TierTurbo, config.TierEnterprise:
		return true // Always queue for premium tiers (hedging + retries)
	default:
		return false
	}
}
