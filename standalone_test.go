package main_test

import (
	"testing"
	"time"
)

// MockClock for testing time-dependent behavior
type MockClock struct {
	currentTime time.Time
}

func (m *MockClock) Now() time.Time {
	return m.currentTime
}

func (m *MockClock) Advance(d time.Duration) {
	m.currentTime = m.currentTime.Add(d)
}

// Clock interface
type Clock interface{ Now() time.Time }

// RNG interface
type RNG interface{ Float64() float64 }

// MockRNG for testing randomization behavior
type MockRNG struct {
	values []float64
	index  int
}

func NewMockRNG(values []float64) *MockRNG {
	return &MockRNG{values: values, index: 0}
}

func (m *MockRNG) Float64() float64 {
	if len(m.values) == 0 {
		return 0.5 // Default value
	}
	v := m.values[m.index]
	m.index = (m.index + 1) % len(m.values)
	return v
}

// JitterStrategy defines how randomized backoff delays are produced.
type JitterStrategy int

const (
	JitterNone JitterStrategy = iota // exact delay
	JitterFull                       // uniform in [0, d)
	JitterEqual                      // uniform in [0.5d, 1.5d]
)

// ExponentialBackoff implements exponential backoff with configurable jitter
type ExponentialBackoff struct {
	baseDelay   time.Duration
	maxDelay    time.Duration
	multiplier  float64
	jitterMode  JitterStrategy
	attempt     int
	delayBase   time.Duration // un-jittered state
	lastBackoff time.Time
	clock       Clock
	rng         RNG
}

// NewExponentialBackoff creates a new exponential backoff
func NewExponentialBackoff(baseDelay, maxDelay time.Duration, multiplier float64) *ExponentialBackoff {
	return &ExponentialBackoff{
		baseDelay:  baseDelay,
		maxDelay:   maxDelay,
		multiplier: multiplier,
		jitterMode: JitterFull,
		delayBase:  baseDelay,
		clock:      &MockClock{time.Now()},
		rng:        &MockRNG{[]float64{0.5}, 0},
	}
}

// NextDelay returns the next delay duration
func (eb *ExponentialBackoff) NextDelay() time.Duration {
	// progress base (un-jittered) delay
	if eb.attempt > 0 {
		next := time.Duration(float64(eb.delayBase) * eb.multiplier)
		if next > eb.maxDelay {
			next = eb.maxDelay
		}
		eb.delayBase = next
	}
	eb.attempt++
	eb.lastBackoff = eb.clock.Now()

	// Apply jitter on the returned value only
	d := eb.delayBase
	switch eb.jitterMode {
	case JitterNone:
		// no-op
	case JitterFull:
		d = time.Duration(eb.rng.Float64() * float64(d)) // [0, d)
	case JitterEqual:
		f := 0.5 + eb.rng.Float64() // [0.5, 1.5)
		d = time.Duration(f * float64(d))
	}
	return d
}

// Reset resets the backoff state
func (eb *ExponentialBackoff) Reset() {
	eb.attempt = 0
	eb.delayBase = eb.baseDelay
	eb.lastBackoff = time.Time{}
}

// SetJitterStrategy sets the jitter strategy
func (eb *ExponentialBackoff) SetJitterStrategy(strategy JitterStrategy) {
	eb.jitterMode = strategy
}

// SetClock sets the clock implementation
func (eb *ExponentialBackoff) SetClock(clock Clock) {
	eb.clock = clock
}

// SetRNG sets the random number generator
func (eb *ExponentialBackoff) SetRNG(rng RNG) {
	eb.rng = rng
}

func TestExponentialBackoff(t *testing.T) {
	baseDelay := 100 * time.Millisecond
	maxDelay := 10 * time.Second
	multiplier := 2.0
	
	t.Run("Base behavior", func(t *testing.T) {
		eb := NewExponentialBackoff(baseDelay, maxDelay, multiplier)
		eb.SetJitterStrategy(JitterNone) // Disable jitter for predictable testing
		
		// First delay should be baseDelay
		if d := eb.NextDelay(); d != baseDelay {
			t.Errorf("First delay should be %v, got %v", baseDelay, d)
		}
		
		// Second delay should be baseDelay * multiplier
		expected := time.Duration(float64(baseDelay) * multiplier)
		if d := eb.NextDelay(); d != expected {
			t.Errorf("Second delay should be %v, got %v", expected, d)
		}
		
		// Third delay should be baseDelay * multiplier^2
		expected = time.Duration(float64(baseDelay) * multiplier * multiplier)
		if d := eb.NextDelay(); d != expected {
			t.Errorf("Third delay should be %v, got %v", expected, d)
		}
		
		// Reset should return to baseDelay
		eb.Reset()
		if d := eb.NextDelay(); d != baseDelay {
			t.Errorf("After reset, delay should be %v, got %v", baseDelay, d)
		}
	})
	
	t.Run("Jitter strategies", func(t *testing.T) {
		eb := NewExponentialBackoff(baseDelay, maxDelay, multiplier)
		mockRng := NewMockRNG([]float64{0.5}) // Always return 0.5
		eb.SetRNG(mockRng)
		
		// Test JitterFull (0-100% of delay)
		eb.SetJitterStrategy(JitterFull)
		eb.Reset()
		expected := time.Duration(float64(baseDelay) * 0.5) // 50% of base delay
		if d := eb.NextDelay(); d != expected {
			t.Errorf("JitterFull should give %v, got %v", expected, d)
		}
		
		// Test JitterEqual (50-150% of delay)
		eb.SetJitterStrategy(JitterEqual)
		eb.Reset()
		expected = time.Duration(float64(baseDelay) * 1.0) // 50% + 50% = 100% of base delay
		if d := eb.NextDelay(); d != expected {
			t.Errorf("JitterEqual should give %v, got %v", expected, d)
		}
	})
}

func main() {
	testing.Main(func(pat, str string) (bool, error) { return true, nil },
		[]testing.InternalTest{{Name: "TestExponentialBackoff", F: TestExponentialBackoff}},
		nil, nil)
}
