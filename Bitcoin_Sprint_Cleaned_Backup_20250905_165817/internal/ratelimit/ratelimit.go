package ratelimit

import (
	"time"
)

// RateLimiter is a minimal stub for rate limiting
type RateLimiter struct {
	maxRequests int
	period      time.Duration
}

// NewRateLimiter constructs a stub rate limiter
func NewRateLimiter(maxRequests int, period time.Duration) *RateLimiter {
	return &RateLimiter{maxRequests: maxRequests, period: period}
}

// Allow always permits in the stub implementation
func (r *RateLimiter) Allow(key string) bool {
	return true
}
