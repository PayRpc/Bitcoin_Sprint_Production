package circuitbreaker

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

// Test ExponentialBackoff
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
	
	t.Run("Maximum delay", func(t *testing.T) {
		eb := NewExponentialBackoff(baseDelay, maxDelay, multiplier)
		eb.SetJitterStrategy(JitterNone)
		
		// Run until we hit max delay
		var lastDelay time.Duration
		for i := 0; i < 10; i++ {
			lastDelay = eb.NextDelay()
			if lastDelay >= maxDelay {
				break
			}
		}
		
		// Next delay should still be maxDelay
		if d := eb.NextDelay(); d != maxDelay {
			t.Errorf("Delay should be capped at %v, got %v", maxDelay, d)
		}
	})
}

// Test SlidingWindow
func TestSlidingWindow(t *testing.T) {
	windowSize := 10 * time.Second
	bucketSize := 1 * time.Second
	
	t.Run("Basic operation", func(t *testing.T) {
		sw := NewSlidingWindow(windowSize, bucketSize)
		mockClock := &MockClock{currentTime: time.Now()}
		sw.SetClock(mockClock)
		
		// Add some requests
		sw.AddRequest(true, 100*time.Millisecond)
		sw.AddRequest(true, 150*time.Millisecond)
		sw.AddRequest(false, 200*time.Millisecond)
		
		// Check statistics
		reqs, succ, fail, failRate, avgLat := sw.GetStatistics()
		if reqs != 3 || succ != 2 || fail != 1 || failRate != 1.0/3.0 {
			t.Errorf("Invalid statistics: reqs=%d, succ=%d, fail=%d, failRate=%f", 
				reqs, succ, fail, failRate)
		}
		
		expectedAvgLat := 150 * time.Millisecond
		if avgLat != expectedAvgLat {
			t.Errorf("Average latency should be %v, got %v", expectedAvgLat, avgLat)
		}
	})
	
	t.Run("Bucket rotation", func(t *testing.T) {
		sw := NewSlidingWindow(windowSize, bucketSize)
		mockClock := &MockClock{currentTime: time.Now()}
		sw.SetClock(mockClock)
		
		// Add initial requests
		sw.AddRequest(true, 100*time.Millisecond)
		sw.AddRequest(true, 100*time.Millisecond)
		
		// Advance time by 2 buckets
		mockClock.Advance(2 * bucketSize)
		
		// Add more requests
		sw.AddRequest(true, 200*time.Millisecond)
		
		// All requests should still be counted
		reqs, _, _, _, _ := sw.GetStatistics()
		if reqs != 3 {
			t.Errorf("Should have 3 requests in window, got %d", reqs)
		}
		
		// Advance time by entire window
		mockClock.Advance(windowSize)
		
		// Add new request
		sw.AddRequest(true, 300*time.Millisecond)
		
		// Old requests should be gone
		reqs, _, _, _, avgLat := sw.GetStatistics()
		if reqs != 1 {
			t.Errorf("Should have only 1 request in window, got %d", reqs)
		}
		if avgLat != 300*time.Millisecond {
			t.Errorf("Average latency should be 300ms, got %v", avgLat)
		}
	})
}

// Test AdaptiveThreshold
func TestAdaptiveThreshold(t *testing.T) {
	baseThreshold := 100.0
	multiplier := 0.1
	
	t.Run("Adjustment based on performance", func(t *testing.T) {
		at := NewAdaptiveThreshold(baseThreshold, multiplier)
		mockClock := &MockClock{currentTime: time.Now()}
		at.SetClock(mockClock)
		at.SetAdjustmentInterval(time.Second)
		
		// Initial threshold should be base
		if at.currentThreshold != baseThreshold {
			t.Errorf("Initial threshold should be %f, got %f", baseThreshold, at.currentThreshold)
		}
		
		// Improving performance should increase threshold
		mockClock.Advance(2 * time.Second)
		newThreshold := at.AdjustThreshold(1.5) // Positive performance trend
		if newThreshold <= baseThreshold {
			t.Errorf("Threshold should increase with good performance, got %f", newThreshold)
		}
		
		// Degrading performance should decrease threshold
		mockClock.Advance(2 * time.Second)
		newThreshold = at.AdjustThreshold(0.5) // Negative performance trend
		if newThreshold >= at.adjustmentHistory[len(at.adjustmentHistory)-2] {
			t.Errorf("Threshold should decrease with poor performance, got %f", newThreshold)
		}
	})
	
	t.Run("Respect bounds", func(t *testing.T) {
		at := NewAdaptiveThreshold(baseThreshold, multiplier)
		mockClock := &MockClock{currentTime: time.Now()}
		at.SetClock(mockClock)
		at.SetAdjustmentInterval(time.Second)
		
		// Set tight bounds
		minFactor := 0.9
		maxFactor := 1.1
		at.SetThresholdBounds(minFactor, maxFactor)
		
		// Try to go below minimum
		mockClock.Advance(time.Second)
		for i := 0; i < 10; i++ {
			at.AdjustThreshold(0.1) // Very poor performance
			mockClock.Advance(time.Second)
		}
		
		// Should be clamped to min
		minThreshold := baseThreshold * minFactor
		if at.currentThreshold < minThreshold {
			t.Errorf("Threshold should be clamped to min %f, got %f", minThreshold, at.currentThreshold)
		}
		
		// Try to go above maximum
		for i := 0; i < 10; i++ {
			at.AdjustThreshold(2.0) // Very good performance
			mockClock.Advance(time.Second)
		}
		
		// Should be clamped to max
		maxThreshold := baseThreshold * maxFactor
		if at.currentThreshold > maxThreshold {
			t.Errorf("Threshold should be clamped to max %f, got %f", maxThreshold, at.currentThreshold)
		}
	})
}

// Test LatencyDetector
func TestLatencyDetector(t *testing.T) {
	baseLatency := 100 * time.Millisecond
	multiplier := 2.0
	
	t.Run("Detect latency issues", func(t *testing.T) {
		ld := NewLatencyDetector(baseLatency, multiplier)
		mockClock := &MockClock{currentTime: time.Now()}
		ld.SetClock(mockClock)
		
		// Add normal latencies
		for i := 0; i < 9; i++ {
			result := ld.AddLatency(baseLatency)
			if result {
				t.Errorf("Should not detect latency issue with normal values")
			}
			mockClock.Advance(time.Second)
		}
		
		// Add high latencies
		highLatency := time.Duration(float64(baseLatency) * multiplier * 1.5)
		for i := 0; i < 8; i++ {
			ld.AddLatency(highLatency)
			mockClock.Advance(time.Second)
		}
		
		// Should detect latency issue after enough high latencies
		result := ld.AddLatency(highLatency)
		if !result {
			t.Errorf("Should detect latency issue after multiple high values")
		}
	})
	
	t.Run("Prune old data", func(t *testing.T) {
		ld := NewLatencyDetector(baseLatency, multiplier)
		mockClock := &MockClock{currentTime: time.Now()}
		ld.SetClock(mockClock)
		ld.detectionWindow = 30 * time.Second
		
		// Add some latencies
		for i := 0; i < 10; i++ {
			ld.AddLatency(baseLatency)
			mockClock.Advance(time.Second)
		}
		
		// These should be pruned
		mockClock.Advance(31 * time.Second)
		ld.AddLatency(baseLatency)
		
		// Internal check - points array should only have the latest point
		ld.mu.Lock()
		count := len(ld.points)
		ld.mu.Unlock()
		
		if count != 1 {
			t.Errorf("Should have pruned old data, got %d points", count)
		}
	})
}

// Test HealthScorer
func TestHealthScorer(t *testing.T) {
	hs := NewHealthScorer()
	
	t.Run("Perfect metrics", func(t *testing.T) {
		metrics := HealthMetrics{
			SuccessRate:         1.0,
			AverageLatency:      100 * time.Millisecond,
			ErrorRate:           0.0,
			ResourceUtilization: 0.0,
			ThroughputRate:      2000.0,
		}
		
		hs.UpdateMetrics(metrics)
		score := hs.CalculateHealth()
		
		if score < 0.99 {
			t.Errorf("Perfect metrics should give score near 1.0, got %f", score)
		}
	})
	
	t.Run("Poor metrics", func(t *testing.T) {
		metrics := HealthMetrics{
			SuccessRate:         0.5,
			AverageLatency:      1500 * time.Millisecond,
			ErrorRate:           0.2,
			ResourceUtilization: 0.9,
			ThroughputRate:      500.0,
		}
		
		hs.UpdateMetrics(metrics)
		score := hs.CalculateHealth()
		
		if score > 0.5 {
			t.Errorf("Poor metrics should give score below 0.5, got %f", score)
		}
	})
}

// Test Percentile Calculation
func TestCalculatePercentile(t *testing.T) {
	values := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
		60 * time.Millisecond,
		70 * time.Millisecond,
		80 * time.Millisecond,
		90 * time.Millisecond,
		100 * time.Millisecond,
	}
	
	tests := []struct {
		percentile float64
		expected   time.Duration
	}{
		{0, 10 * time.Millisecond},
		{50, 55 * time.Millisecond},
		{90, 91 * time.Millisecond},
		{100, 100 * time.Millisecond},
		{-10, 10 * time.Millisecond},  // Should clamp to 0th
		{110, 100 * time.Millisecond}, // Should clamp to 100th
	}
	
	for _, test := range tests {
		result := calculatePercentile(values, test.percentile)
		if result != test.expected {
			t.Errorf("P%.1f should be %v, got %v", test.percentile, test.expected, result)
		}
	}
	
	// Test empty list
	result := calculatePercentile([]time.Duration{}, 50)
	if result != 0 {
		t.Errorf("Percentile of empty list should be 0, got %v", result)
	}
}
