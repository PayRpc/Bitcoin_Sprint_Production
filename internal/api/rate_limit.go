// Package api provides rate limiting functionality
package api

import (
	"math"
	"sync"
	"time"
)

// ===== RATE LIMITER IMPLEMENTATION =====

// RateLimiter manages rate limiting for API requests
type RateLimiter struct {
	buckets map[string]*TokenBucket
	clock   Clock
	mu      sync.RWMutex
}

// TokenBucket implements the token bucket algorithm for rate limiting
type TokenBucket struct {
	tokens         float64
	capacity       float64
	refillRate     float64
	lastRefillTime time.Time
	clock          Clock
	mu             sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(clock Clock) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*TokenBucket),
		clock:   clock,
	}
}

// Allow checks if a request from the given identifier is allowed
func (rl *RateLimiter) Allow(identifier string, capacity float64, refillRate float64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[identifier]
	if !exists {
		bucket = &TokenBucket{
			tokens:         capacity,
			capacity:       capacity,
			refillRate:     refillRate,
			lastRefillTime: rl.clock.Now(),
			clock:          rl.clock,
		}
		rl.buckets[identifier] = bucket
	}

	return bucket.Allow()
}

// Allow checks if the token bucket allows a request
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := tb.clock.Now()
	timePassed := now.Sub(tb.lastRefillTime).Seconds()
	tokensToAdd := timePassed * tb.refillRate

	tb.tokens = math.Min(tb.capacity, tb.tokens+tokensToAdd)
	tb.lastRefillTime = now

	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return true
	}

	return false
}
