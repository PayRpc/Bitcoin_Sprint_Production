package circuitbreaker

import (
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// Exponential Backoff Algorithm Implementation
type ExponentialBackoff struct {
	mu               sync.RWMutex
	baseDelay        time.Duration
	maxDelay         time.Duration
	multiplier       float64
	jitter           bool
	attempt          int
	currentDelay     time.Duration
	lastBackoffTime  time.Time
}

func newExponentialBackoff(baseDelay, maxDelay time.Duration, multiplier float64) *ExponentialBackoff {
	return &ExponentialBackoff{
		baseDelay:   baseDelay,
		maxDelay:    maxDelay,
		multiplier:  multiplier,
		jitter:      true,
		attempt:     0,
	}
}

func (eb *ExponentialBackoff) NextDelay() time.Duration {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	
	if eb.attempt == 0 {
		eb.currentDelay = eb.baseDelay
	} else {
		eb.currentDelay = time.Duration(float64(eb.currentDelay) * eb.multiplier)
		if eb.currentDelay > eb.maxDelay {
			eb.currentDelay = eb.maxDelay
		}
	}
	
	eb.attempt++
	eb.lastBackoffTime = time.Now()
	
	// Add jitter to prevent thundering herd
	if eb.jitter {
		jitterAmount := time.Duration(rand.Float64() * float64(eb.currentDelay) * 0.1)
		eb.currentDelay += jitterAmount
	}
	
	return eb.currentDelay
}

func (eb *ExponentialBackoff) Reset() {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	
	eb.attempt = 0
	eb.currentDelay = eb.baseDelay
}

// Sliding Window Statistics Implementation
func newSlidingWindow(windowSize, bucketSize time.Duration) *SlidingWindow {
	bucketCount := int(windowSize / bucketSize)
	if bucketCount < 1 {
		bucketCount = 1
	}
	
	buckets := make([]WindowBucket, bucketCount)
	now := time.Now()
	
	for i := range buckets {
		buckets[i] = WindowBucket{
			timestamp:  now.Add(-time.Duration(i) * bucketSize),
			minLatency: time.Hour, // High initial value
		}
	}
	
	return &SlidingWindow{
		buckets:      buckets,
		bucketSize:   bucketSize,
		windowSize:   windowSize,
		currentIndex: 0,
		lastUpdate:   now,
	}
}

func (sw *SlidingWindow) AddRequest(success bool, latency time.Duration) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	
	now := time.Now()
	sw.rotateIfNeeded(now)
	
	currentBucket := &sw.buckets[sw.currentIndex]
	currentBucket.requests++
	
	if !success {
		currentBucket.failures++
	}
	
	if latency > 0 {
		currentBucket.latencySum += int64(latency)
		currentBucket.latencyCount++
		
		if latency > currentBucket.maxLatency {
			currentBucket.maxLatency = latency
		}
		
		if latency < currentBucket.minLatency {
			currentBucket.minLatency = latency
		}
	}
}

func (sw *SlidingWindow) GetStatistics() (requests, failures int64, failureRate float64, avgLatency time.Duration) {
	sw.mu.RLock()
	defer sw.mu.RUnlock()
	
	now := time.Now()
	cutoff := now.Add(-sw.windowSize)
	
	var totalLatency int64
	var latencyCount int64
	
	for _, bucket := range sw.buckets {
		if bucket.timestamp.After(cutoff) {
			requests += bucket.requests
			failures += bucket.failures
			totalLatency += bucket.latencySum
			latencyCount += bucket.latencyCount
		}
	}
	
	if requests > 0 {
		failureRate = float64(failures) / float64(requests)
	}
	
	if latencyCount > 0 {
		avgLatency = time.Duration(totalLatency / latencyCount)
	}
	
	return
}

func (sw *SlidingWindow) rotateIfNeeded(now time.Time) {
	if now.Sub(sw.lastUpdate) >= sw.bucketSize {
		sw.currentIndex = (sw.currentIndex + 1) % len(sw.buckets)
		
		// Reset the new bucket
		sw.buckets[sw.currentIndex] = WindowBucket{
			timestamp:  now,
			minLatency: time.Hour,
		}
		
		sw.lastUpdate = now
	}
}

// Adaptive Threshold Algorithm Implementation
func newAdaptiveThreshold(baseThreshold, multiplier float64) *AdaptiveThreshold {
	return &AdaptiveThreshold{
		currentThreshold:   baseThreshold,
		baseThreshold:      baseThreshold,
		multiplier:         multiplier,
		adjustmentHistory:  make([]float64, 0, 100),
		performanceHistory: make([]float64, 0, 100),
	}
}

func (at *AdaptiveThreshold) AdjustThreshold(currentPerformance float64) float64 {
	at.mu.Lock()
	defer at.mu.Unlock()
	
	now := time.Now()
	
	// Don't adjust too frequently
	if now.Sub(at.lastAdjustment) < time.Minute {
		return at.currentThreshold
	}
	
	// Record performance history
	at.performanceHistory = append(at.performanceHistory, currentPerformance)
	if len(at.performanceHistory) > 100 {
		at.performanceHistory = at.performanceHistory[1:]
	}
	
	// Calculate trend
	trend := at.calculateTrend()
	
	// Adjust threshold based on trend
	if trend > 0.1 { // Performance improving
		at.currentThreshold = math.Min(at.currentThreshold*1.1, at.baseThreshold*2.0)
	} else if trend < -0.1 { // Performance degrading
		at.currentThreshold = math.Max(at.currentThreshold*0.9, at.baseThreshold*0.5)
	}
	
	at.adjustmentHistory = append(at.adjustmentHistory, at.currentThreshold)
	if len(at.adjustmentHistory) > 100 {
		at.adjustmentHistory = at.adjustmentHistory[1:]
	}
	
	at.lastAdjustment = now
	return at.currentThreshold
}

func (at *AdaptiveThreshold) calculateTrend() float64 {
	if len(at.performanceHistory) < 5 {
		return 0
	}
	
	n := len(at.performanceHistory)
	recent := at.performanceHistory[n-5:]
	older := at.performanceHistory[max(0, n-10):n-5]
	
	if len(older) == 0 {
		return 0
	}
	
	recentAvg := average(recent)
	olderAvg := average(older)
	
	if olderAvg == 0 {
		return 0
	}
	
	return (recentAvg - olderAvg) / olderAvg
}

// Health Scorer Implementation
func newHealthScorer() *HealthScorer {
	return &HealthScorer{
		weights: HealthWeights{
			SuccessRate:   0.3,
			Latency:       0.25,
			ErrorRate:     0.2,
			ResourceUsage: 0.15,
			Throughput:    0.1,
		},
		calculationInterval: time.Minute,
	}
}

func (hs *HealthScorer) UpdateMetrics(metrics HealthMetrics) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	
	hs.metrics = metrics
	hs.lastCalculation = time.Now()
}

func (hs *HealthScorer) CalculateHealth() float64 {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	
	// Calculate weighted health score
	score := 0.0
	
	// Success rate contribution (0-1)
	score += hs.metrics.SuccessRate * hs.weights.SuccessRate
	
	// Latency contribution (inverse relationship)
	latencyScore := math.Max(0, 1.0-float64(hs.metrics.AverageLatency)/float64(time.Second))
	score += latencyScore * hs.weights.Latency
	
	// Error rate contribution (inverse relationship)
	errorScore := math.Max(0, 1.0-hs.metrics.ErrorRate)
	score += errorScore * hs.weights.ErrorRate
	
	// Resource utilization contribution (inverse relationship)
	resourceScore := math.Max(0, 1.0-hs.metrics.ResourceUtilization)
	score += resourceScore * hs.weights.ResourceUsage
	
	// Throughput contribution (normalized)
	throughputScore := math.Min(1.0, hs.metrics.ThroughputRate/1000.0)
	score += throughputScore * hs.weights.Throughput
	
	return math.Max(0, math.Min(1.0, score))
}

// Latency-Based Detection Algorithm
type LatencyDetector struct {
	mu                   sync.RWMutex
	baselineLatency      time.Duration
	thresholdMultiplier  float64
	latencyHistory       []time.Duration
	detectionWindow      time.Duration
	lastUpdate          time.Time
}

func newLatencyDetector(baselineLatency time.Duration, multiplier float64) *LatencyDetector {
	return &LatencyDetector{
		baselineLatency:     baselineLatency,
		thresholdMultiplier: multiplier,
		latencyHistory:      make([]time.Duration, 0, 100),
		detectionWindow:     time.Minute * 5,
	}
}

func (ld *LatencyDetector) AddLatency(latency time.Duration) bool {
	ld.mu.Lock()
	defer ld.mu.Unlock()
	
	ld.latencyHistory = append(ld.latencyHistory, latency)
	if len(ld.latencyHistory) > 100 {
		ld.latencyHistory = ld.latencyHistory[1:]
	}
	
	// Check if current latency indicates degradation
	threshold := time.Duration(float64(ld.baselineLatency) * ld.thresholdMultiplier)
	
	if latency > threshold {
		// Check recent trend
		return ld.checkLatencyTrend()
	}
	
	return false
}

func (ld *LatencyDetector) checkLatencyTrend() bool {
	if len(ld.latencyHistory) < 10 {
		return false
	}
	
	// Check if recent latencies are consistently high
	recentCount := min(10, len(ld.latencyHistory))
	recent := ld.latencyHistory[len(ld.latencyHistory)-recentCount:]
	
	threshold := time.Duration(float64(ld.baselineLatency) * ld.thresholdMultiplier)
	highLatencyCount := 0
	
	for _, lat := range recent {
		if lat > threshold {
			highLatencyCount++
		}
	}
	
	// If more than 70% of recent requests are slow, trigger detection
	return float64(highLatencyCount)/float64(len(recent)) > 0.7
}

// Recovery Probability Calculator
type RecoveryCalculator struct {
	mu                    sync.RWMutex
	lastFailureTime       time.Time
	consecutiveFailures   int
	historicalRecoveryTime time.Duration
	baseRecoveryProbability float64
}

func newRecoveryCalculator() *RecoveryCalculator {
	return &RecoveryCalculator{
		baseRecoveryProbability: 0.1,
		historicalRecoveryTime:  time.Minute * 30,
	}
}

func (rc *RecoveryCalculator) CalculateRecoveryProbability() float64 {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	
	if rc.lastFailureTime.IsZero() {
		return 1.0
	}
	
	timeSinceFailure := time.Since(rc.lastFailureTime)
	
	// Base probability increases with time
	timeFactor := math.Min(1.0, float64(timeSinceFailure)/float64(rc.historicalRecoveryTime))
	
	// Consecutive failures reduce probability
	failureFactor := math.Pow(0.8, float64(rc.consecutiveFailures))
	
	probability := rc.baseRecoveryProbability + (1.0-rc.baseRecoveryProbability)*timeFactor*failureFactor
	
	return math.Max(0.0, math.Min(1.0, probability))
}

func (rc *RecoveryCalculator) RecordFailure() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	
	rc.lastFailureTime = time.Now()
	rc.consecutiveFailures++
}

func (rc *RecoveryCalculator) RecordSuccess() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	
	rc.consecutiveFailures = 0
}

// Utility functions
func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	
	return sum / float64(len(values))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Percentile calculation for latency metrics
func calculatePercentile(values []time.Duration, percentile float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	
	sorted := make([]time.Duration, len(values))
	copy(sorted, values)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	
	index := int(float64(len(sorted)) * percentile / 100.0)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	
	return sorted[index]
}
