package circuitbreaker

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Manager-specific constants that alias the main State type
const (
	// Closed means the circuit breaker is closed (allowing requests)
	Closed = StateClosed
	// Open means the circuit breaker is open (blocking requests)
	Open = StateOpen
	// HalfOpen means the circuit breaker is allowing a test request
	HalfOpen = StateHalfOpen
)

// ManagerConfig holds configuration for a circuit breaker manager
type ManagerConfig struct {
	Name             string
	MaxFailures      int
	ResetTimeout     time.Duration
	FailureThreshold float64
	SuccessThreshold int
	Timeout          time.Duration
	OnStateChange    func(name string, from State, to State)
	Logger           *zap.Logger
}

// Manager manages a circuit breaker
type Manager struct {
	name             string
	maxFailures      int
	resetTimeout     time.Duration
	failureThreshold float64
	successThreshold int
	timeout          time.Duration
	onStateChange    func(name string, from State, to State)
	logger           *zap.Logger

	mu              sync.RWMutex
	failures        int
	successes       int
	state           State
	lastStateChange time.Time
	generation      int
}

// Name returns the circuit breaker name
func (m *Manager) Name() string {
	return m.name
}

// NewManager creates a new circuit breaker manager
func NewManager(config ManagerConfig) *Manager {
	if config.MaxFailures <= 0 {
		config.MaxFailures = 5
	}
	if config.ResetTimeout <= 0 {
		config.ResetTimeout = 30 * time.Second
	}
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 0.5
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = 2
	}
	if config.Timeout <= 0 {
		config.Timeout = 5 * time.Second
	}

	cb := &Manager{
		name:             config.Name,
		maxFailures:      config.MaxFailures,
		resetTimeout:     config.ResetTimeout,
		failureThreshold: config.FailureThreshold,
		successThreshold: config.SuccessThreshold,
		timeout:          config.Timeout,
		onStateChange:    config.OnStateChange,
		logger:           config.Logger,
		state:            Closed,
		lastStateChange:  time.Now(),
	}

	return cb
}

// Execute executes the given function with circuit breaker protection
func (cb *Manager) Execute(f func() error) error {
	if !cb.AllowRequest() {
		return fmt.Errorf("circuit breaker %s is open", cb.name)
	}

	generation := cb.generation
	err := f()

	if err != nil {
		cb.RecordFailure()
		return err
	}

	cb.RecordSuccess()
	return nil
}

// AllowRequest checks if a request should be allowed
func (cb *Manager) AllowRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case Closed:
		return true
	case Open:
		if time.Since(cb.lastStateChange) > cb.resetTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			defer cb.mu.Unlock()
			cb.toHalfOpen()
			return true
		}
		return false
	case HalfOpen:
		return true
	default:
		return true
	}
}

// RecordSuccess records a successful request
func (cb *Manager) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case Closed:
		cb.failures = 0
	case HalfOpen:
		cb.successes++
		if cb.successes >= cb.successThreshold {
			cb.toClosed()
		}
	}
}

// RecordFailure records a failed request
func (cb *Manager) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case Closed:
		cb.failures++
		if cb.failures >= cb.maxFailures {
			cb.toOpen()
		}
	case HalfOpen:
		cb.toOpen()
	}
}

// State returns the current state of the circuit breaker
func (cb *Manager) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// toOpen transitions the circuit breaker to the open state
func (cb *Manager) toOpen() {
	if cb.state != Open {
		prev := cb.state
		cb.state = Open
		cb.lastStateChange = time.Now()
		cb.generation++
		if cb.onStateChange != nil {
			cb.onStateChange(cb.name, prev, Open)
		}
		if cb.logger != nil {
			cb.logger.Info("Circuit breaker opened",
				zap.String("name", cb.name),
				zap.Int("failures", cb.failures),
			)
		}
	}
}

// toHalfOpen transitions the circuit breaker to the half-open state
func (cb *Manager) toHalfOpen() {
	if cb.state != HalfOpen {
		prev := cb.state
		cb.state = HalfOpen
		cb.lastStateChange = time.Now()
		cb.successes = 0
		if cb.onStateChange != nil {
			cb.onStateChange(cb.name, prev, HalfOpen)
		}
		if cb.logger != nil {
			cb.logger.Info("Circuit breaker half-opened",
				zap.String("name", cb.name),
			)
		}
	}
}

// toClosed transitions the circuit breaker to the closed state
func (cb *Manager) toClosed() {
	if cb.state != Closed {
		prev := cb.state
		cb.state = Closed
		cb.lastStateChange = time.Now()
		cb.failures = 0
		cb.successes = 0
		if cb.onStateChange != nil {
			cb.onStateChange(cb.name, prev, Closed)
		}
		if cb.logger != nil {
			cb.logger.Info("Circuit breaker closed",
				zap.String("name", cb.name),
			)
		}
	}
}
