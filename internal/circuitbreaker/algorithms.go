package circuitbreaker

import (
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// ---- Infrastructure for testability (deterministic time & randomness) ----
type Clock interface{ Now() time.Time }
type realClock struct{}
func (realClock) Now() time.Time { return time.Now() }

type RNG interface{ Float64() float64 }
type defaultRNG struct{}
func (defaultRNG) Float64() float64 { return rand.Float64() }

// JitterStrategy defines how randomized backoff delays are produced.
type JitterStrategy int
const (
	JitterNone JitterStrategy = iota // exact delay
	JitterFull                       // uniform in [0, d)
	JitterEqual                      // uniform in [0.5d, 1.5d]
)

// ExponentialBackoff implements exponential backoff with configurable jitter
type ExponentialBackoff struct {
	mu          sync.Mutex
	clock       Clock
	rng         RNG
	baseDelay   time.Duration
	maxDelay    time.Duration
	multiplier  float64
	jitterMode  JitterStrategy
	attempt     int
	delayBase   time.Duration // un-jittered state
	lastBackoff time.Time
}

// NewExponentialBackoff creates a new exponential backoff with default settings
func NewExponentialBackoff(baseDelay, maxDelay time.Duration, multiplier float64) *ExponentialBackoff {
	return &ExponentialBackoff{
		clock:      realClock{},
		rng:        defaultRNG{},
		baseDelay:  baseDelay,
		maxDelay:   maxDelay,
		multiplier: multiplier,
		jitterMode: JitterFull,
		delayBase:  baseDelay,
	}
}

// NextDelay returns the next delay duration based on the current attempt
func (eb *ExponentialBackoff) NextDelay() time.Duration {
	eb.mu.Lock()
	defer eb.mu.Unlock()

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

// Reset resets the backoff to its initial state
func (eb *ExponentialBackoff) Reset() {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.attempt = 0
	eb.delayBase = eb.baseDelay
	eb.lastBackoff = time.Time{}
}

// SetJitterStrategy sets the jitter strategy
func (eb *ExponentialBackoff) SetJitterStrategy(strategy JitterStrategy) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.jitterMode = strategy
}

// SetClock sets the clock implementation (for testing)
func (eb *ExponentialBackoff) SetClock(clock Clock) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.clock = clock
}

// SetRNG sets the random number generator (for testing)
func (eb *ExponentialBackoff) SetRNG(rng RNG) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.rng = rng
}

// WindowBucket holds statistics for a time slice in the sliding window
type WindowBucket struct {
	timestamp    time.Time
	requests     int64
	successes    int64
	failures     int64
	latencySum   int64
	latencyCount int64
	minLatency   time.Duration
	maxLatency   time.Duration
}

// SlidingWindow implements a time-based sliding window for statistics
type SlidingWindow struct {
	mu           sync.RWMutex
	buckets      []WindowBucket
	bucketSize   time.Duration
	windowSize   time.Duration
	currentIndex int
	lastUpdate   time.Time
	clock        Clock
}

// NewSlidingWindow creates a new sliding window with the specified window and bucket sizes
func NewSlidingWindow(windowSize, bucketSize time.Duration) *SlidingWindow {
	bucketCount := int(windowSize / bucketSize)
	if bucketCount < 1 {
		bucketCount = 1
	}
	
	clock := realClock{}
	now := clock.Now()
	buckets := make([]WindowBucket, bucketCount)
	
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
		clock:        clock,
	}
}

// AddRequest adds a request to the current bucket
func (sw *SlidingWindow) AddRequest(success bool, latency time.Duration) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	
	now := sw.clock.Now()
	sw.rotateIfNeeded(now)
	
	currentBucket := &sw.buckets[sw.currentIndex]
	currentBucket.requests++
	
	if success {
		currentBucket.successes++
	} else {
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

// GetStatistics returns the aggregated statistics over the sliding window
func (sw *SlidingWindow) GetStatistics() (requests, successes, failures int64, failureRate float64, avgLatency time.Duration) {
	sw.mu.RLock()
	defer sw.mu.RUnlock()
	
	now := sw.clock.Now()
	cutoff := now.Add(-sw.windowSize)
	
	var totalLatency int64
	var latencyCount int64
	
	for _, bucket := range sw.buckets {
		if bucket.timestamp.After(cutoff) {
			requests += bucket.requests
			successes += bucket.successes
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

// SetClock sets the clock implementation (for testing)
func (sw *SlidingWindow) SetClock(clock Clock) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.clock = clock
}

// rotateIfNeeded advances the window if enough time has passed
func (sw *SlidingWindow) rotateIfNeeded(now time.Time) {
	delta := now.Sub(sw.lastUpdate)
	if delta < sw.bucketSize {
		return
	}
	steps := int(delta / sw.bucketSize)
	if steps > len(sw.buckets) {
		// entire window expired; reset all buckets
		steps = len(sw.buckets)
	}
	for i := 0; i < steps; i++ {
		sw.currentIndex = (sw.currentIndex + 1) % len(sw.buckets)
		sw.buckets[sw.currentIndex] = WindowBucket{
			timestamp:  sw.lastUpdate.Add(time.Duration(i+1) * sw.bucketSize),
			minLatency: time.Hour,
		}
	}
	sw.lastUpdate = sw.lastUpdate.Add(time.Duration(steps) * sw.bucketSize)
}

// AdaptiveThreshold implements a self-adjusting threshold based on performance trends
type AdaptiveThreshold struct {
	mu                 sync.Mutex
	currentThreshold   float64
	baseThreshold      float64
	multiplier         float64
	adjustEvery        time.Duration
	lastAdjustment     time.Time
	adjustmentHistory  []float64
	performanceHistory []float64
	minFactor          float64 // Minimum threshold as factor of base (e.g., 0.5x)
	maxFactor          float64 // Maximum threshold as factor of base (e.g., 2.0x)
	clock              Clock
}

// NewAdaptiveThreshold creates a new adaptive threshold with the specified base and multiplier
func NewAdaptiveThreshold(baseThreshold, multiplier float64) *AdaptiveThreshold {
	return &AdaptiveThreshold{
		currentThreshold:    baseThreshold,
		baseThreshold:       baseThreshold,
		multiplier:          multiplier,
		adjustEvery:         time.Minute,
		minFactor:           0.5, // clamp to base × [0.5, 2.0] by default
		maxFactor:           2.0,
		adjustmentHistory:   make([]float64, 0, 100),
		performanceHistory:  make([]float64, 0, 100),
		clock:               realClock{},
	}
}

// AdjustThreshold updates the threshold based on current performance metrics
func (at *AdaptiveThreshold) AdjustThreshold(currentPerformance float64) float64 {
	at.mu.Lock()
	defer at.mu.Unlock()
	
	now := at.clock.Now()
	
	// Don't adjust too frequently
	if now.Sub(at.lastAdjustment) < at.adjustEvery {
		return at.currentThreshold
	}
	
	// Record performance history
	at.performanceHistory = append(at.performanceHistory, currentPerformance)
	if len(at.performanceHistory) > 100 {
		at.performanceHistory = at.performanceHistory[1:]
	}
	
	// Calculate trend
	trend := at.calculateTrend()
	
	// Adjust threshold based on trend and multiplier
	switch {
	case trend > 0.1: // Performance improving
		at.currentThreshold *= 1.0 + 0.1*at.multiplier
	case trend < -0.1: // Performance degrading
		at.currentThreshold *= 1.0 - 0.1*at.multiplier
	}
	
	// Clamp to safety bounds
	lo := at.baseThreshold * at.minFactor
	hi := at.baseThreshold * at.maxFactor
	if at.currentThreshold < lo {
		at.currentThreshold = lo
	} else if at.currentThreshold > hi {
		at.currentThreshold = hi
	}
	
	at.adjustmentHistory = append(at.adjustmentHistory, at.currentThreshold)
	if len(at.adjustmentHistory) > 100 {
		at.adjustmentHistory = at.adjustmentHistory[1:]
	}
	
	at.lastAdjustment = now
	return at.currentThreshold
}

// SetAdjustmentInterval sets how frequently the threshold can be adjusted
func (at *AdaptiveThreshold) SetAdjustmentInterval(interval time.Duration) {
	at.mu.Lock()
	defer at.mu.Unlock()
	at.adjustEvery = interval
}

// SetThresholdBounds sets the minimum and maximum factors relative to base threshold
func (at *AdaptiveThreshold) SetThresholdBounds(minFactor, maxFactor float64) {
	at.mu.Lock()
	defer at.mu.Unlock()
	at.minFactor = minFactor
	at.maxFactor = maxFactor
}

// SetClock sets the clock implementation (for testing)
func (at *AdaptiveThreshold) SetClock(clock Clock) {
	at.mu.Lock()
	defer at.mu.Unlock()
	at.clock = clock
}

// calculateTrend computes the performance trend as a relative change
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

// HealthWeights define how different metrics contribute to overall health
type HealthWeights struct {
	SuccessRate   float64
	Latency       float64
	ErrorRate     float64
	ResourceUsage float64
	Throughput    float64
}

// HealthTargets defines the target values for various health metrics
type HealthTargets struct {
	P50Latency time.Duration // good latency target
	MaxLatency time.Duration // worst acceptable latency
	TargetTPS  float64       // target throughput for score=1.0
}

// HealthMetrics contains the current values for various health metrics
type HealthMetrics struct {
	SuccessRate         float64
	AverageLatency      time.Duration
	ErrorRate           float64
	ResourceUtilization float64
	ThroughputRate      float64
}

// HealthScorer calculates a health score based on multiple metrics
type HealthScorer struct {
	mu                  sync.RWMutex
	weights             HealthWeights
	targets             HealthTargets
	metrics             HealthMetrics
	lastCalculation     time.Time
	calculationInterval time.Duration
}

// Health Scorer Implementation
func NewHealthScorer() *HealthScorer {
	return &HealthScorer{
		weights: HealthWeights{
			SuccessRate:   0.3,
			Latency:       0.25,
			ErrorRate:     0.2,
			ResourceUsage: 0.15,
			Throughput:    0.1,
		},
		targets: HealthTargets{
			P50Latency: 200 * time.Millisecond,
			MaxLatency: 2 * time.Second,
			TargetTPS:  2000,
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
	m := hs.metrics
	w := hs.weights
	t := hs.targets
	hs.mu.RUnlock()

	// Success rate (already 0..1)
	okScore := clamp01(m.SuccessRate)

	// Latency: piecewise linear from P50 (score=1) to Max (score=0)
	lat := float64(m.AverageLatency)
	var latScore float64
	if lat <= float64(t.P50Latency) {
		latScore = 1.0
	} else {
		den := float64(t.MaxLatency - t.P50Latency)
		if den <= 0 {
			latScore = 0
		} else {
			latScore = 1.0 - clamp01((lat-float64(t.P50Latency))/den)
		}
	}

	// Error rate & resource utilization are inverse contributions
	errScore := clamp01(1.0 - m.ErrorRate)
	resScore := clamp01(1.0 - m.ResourceUtilization)

	// Throughput normalized against target
	tputScore := clamp01(m.ThroughputRate / maxFloat(1.0, t.TargetTPS))

	score := okScore*w.SuccessRate + latScore*w.Latency + errScore*w.ErrorRate + resScore*w.ResourceUsage + tputScore*w.Throughput
	return clamp01(score)
}

// Latency-Based Detection Algorithm
// LatencyDetector tracks latency samples and detects performance degradation
type LatencyDetector struct {
	mu                  sync.Mutex
	baselineLatency     time.Duration
	thresholdMultiplier float64
	points              []latencyPoint
	detectionWindow     time.Duration
	clock               Clock
}

// latencyPoint represents a single latency measurement with timestamp
type latencyPoint struct {
	t time.Time
	d time.Duration
}

// NewLatencyDetector creates a new detector with specified baseline and multiplier
func NewLatencyDetector(baselineLatency time.Duration, multiplier float64) *LatencyDetector {
	return &LatencyDetector{
		baselineLatency:     baselineLatency,
		thresholdMultiplier: multiplier,
		points:              make([]latencyPoint, 0, 128),
		detectionWindow:     5 * time.Minute,
		clock:               realClock{},
	}
}

// AddLatency adds a latency sample and checks if performance has degraded
func (ld *LatencyDetector) AddLatency(latency time.Duration) bool {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	now := ld.clock.Now()
	ld.points = append(ld.points, latencyPoint{t: now, d: latency})
	// cap memory
	if len(ld.points) > 1000 {
		ld.points = ld.points[len(ld.points)-1000:]
	}
	// prune outside detection window
	cut := now.Add(-ld.detectionWindow)
	i := 0
	for ; i < len(ld.points) && ld.points[i].t.Before(cut); i++ {
	}
	if i > 0 {
		ld.points = ld.points[i:]
	}

	threshold := time.Duration(float64(ld.baselineLatency) * ld.thresholdMultiplier)
	if latency <= threshold || len(ld.points) < 10 {
		return false
	}
	return ld.checkLatencyTrend(threshold)
}

// SetClock sets the clock implementation (for testing)
func (ld *LatencyDetector) SetClock(clock Clock) {
	ld.mu.Lock()
	defer ld.mu.Unlock()
	ld.clock = clock
}

// checkLatencyTrend determines if there's a persistent latency issue
func (ld *LatencyDetector) checkLatencyTrend(threshold time.Duration) bool {
	if len(ld.points) < 10 {
		return false
	}
	// evaluate the last N samples (bounded by available)
	N := 10
	if len(ld.points) < N {
		N = len(ld.points)
	}
	recent := ld.points[len(ld.points)-N:]
	slow := 0
	for _, p := range recent {
		if p.d > threshold {
			slow++
		}
	}
	// trigger if >70% of recent requests are slow
	return float64(slow)/float64(N) > 0.7
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
	n := len(values)
	if n == 0 {
		return 0
	}
	// clamp percentile to [0,100]
	p := percentile
	if p < 0 {
		p = 0
	} else if p > 100 {
		p = 100
	}
	sorted := make([]time.Duration, n)
	copy(sorted, values)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	if n == 1 || p == 0 {
		return sorted[0]
	}
	if p == 100 {
		return sorted[n-1]
	}
	// linear interpolation on rank positions [0..n-1]
	pos := (p / 100.0) * float64(n-1)
	i := int(math.Floor(pos))
	f := pos - float64(i)
	if i+1 >= n {
		return sorted[i]
	}
	return time.Duration((1.0-f)*float64(sorted[i]) + f*float64(sorted[i+1]))
}

// ---- Small helpers ----
func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
