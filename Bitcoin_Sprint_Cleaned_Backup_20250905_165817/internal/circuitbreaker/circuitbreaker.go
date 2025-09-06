package circuitbreaker

import "time"

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Config is a simple configuration used by the stub
type Config struct {
	Name             string
	MaxFailures      int
	ResetTimeout     time.Duration
	HalfOpenMaxCalls int
	Timeout          time.Duration
	OnStateChange    func(name string, from, to State)
}

// CircuitBreaker is a lightweight stub implementation
type CircuitBreaker struct{}

// NewCircuitBreaker returns a stub circuit breaker instance
func NewCircuitBreaker(cfg Config) *CircuitBreaker {
	return &CircuitBreaker{}
}

// Execute runs the provided function and returns its result
func (cb *CircuitBreaker) Execute(fn func() (interface{}, error)) (interface{}, error) {
	return fn()
}

// State returns a simple state (closed)
func (cb *CircuitBreaker) State() State {
	return StateClosed
}
