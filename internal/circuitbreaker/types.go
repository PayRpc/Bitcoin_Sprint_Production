package circuitbreaker

import "time"

// State represents the current state of a circuit breaker
type State int

const (
	StateClosed   State = iota // Normal operation
	StateOpen                  // Circuit is open (rejecting requests)
	StateHalfOpen              // Testing if circuit can be closed
)

// Note: String() method is implemented in circuitbreaker.go

// Config holds the configuration for a circuit breaker
type Config struct {
	Name                   string
	FailureThreshold       float64
	SuccessThreshold       int
	Timeout                time.Duration
	HalfOpenMaxConcurrency int
	MinSamples             int
	TripStrategy           string
	CooldownStrategy       string
	Logger                 interface{}
	Metrics                interface{}
	// Enterprise features
	TierSettings        interface{}
	EnableHealthScoring bool
}
