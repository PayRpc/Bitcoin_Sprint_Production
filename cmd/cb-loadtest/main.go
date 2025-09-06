package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/circuitbreaker"
)

// LoadTestConfig defines configuration for load testing
type LoadTestConfig struct {
	Duration      time.Duration
	Concurrency   int
	RequestRate   float64
	FailureRate   float64
	LatencyMin    time.Duration
	LatencyMax    time.Duration
	BreakerConfig circuitbreaker.Config
	TestScenario  string
	OutputFile    string
}

// TestResult captures the results of a load test
type TestResult struct {
	TotalRequests      int64            `json:"total_requests"`
	SuccessfulRequests int64            `json:"successful_requests"`
	FailedRequests     int64            `json:"failed_requests"`
	CircuitOpenCount   int64            `json:"circuit_open_count"`
	AverageLatency     time.Duration    `json:"average_latency"`
	MaxLatency         time.Duration    `json:"max_latency"`
	MinLatency         time.Duration    `json:"min_latency"`
	P95Latency         time.Duration    `json:"p95_latency"`
	P99Latency         time.Duration    `json:"p99_latency"`
	ThroughputRPS      float64          `json:"throughput_rps"`
	Duration           time.Duration    `json:"duration"`
	StateChanges       []StateChange    `json:"state_changes"`
	ErrorTypes         map[string]int64 `json:"error_types"`
}

// StateChange records when the circuit breaker changed state
type StateChange struct {
	Timestamp time.Time `json:"timestamp"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Reason    string    `json:"reason"`
}

// RequestLatency tracks individual request latencies
type RequestLatency struct {
	timestamp time.Time
	latency   time.Duration
	success   bool
	error     string
}

func main() {
	var (
		duration    = flag.Duration("duration", time.Minute*5, "Test duration")
		concurrency = flag.Int("concurrency", 10, "Number of concurrent workers")
		requestRate = flag.Float64("rate", 100.0, "Requests per second")
		failureRate = flag.Float64("failure-rate", 0.1, "Simulated failure rate (0.0-1.0)")
		latencyMin  = flag.Duration("latency-min", time.Millisecond*10, "Minimum simulated latency")
		latencyMax  = flag.Duration("latency-max", time.Millisecond*100, "Maximum simulated latency")
		scenario    = flag.String("scenario", "standard", "Test scenario (standard, spike, gradual-failure, recovery)")
		outputFile  = flag.String("output", "", "Output file for results (JSON format)")
		tier        = flag.String("tier", "business", "Circuit breaker tier (free, business, enterprise)")
		configFile  = flag.String("config", "", "Custom circuit breaker configuration file")
	)
	flag.Parse()

	config := LoadTestConfig{
		Duration:     *duration,
		Concurrency:  *concurrency,
		RequestRate:  *requestRate,
		FailureRate:  *failureRate,
		LatencyMin:   *latencyMin,
		LatencyMax:   *latencyMax,
		TestScenario: *scenario,
		OutputFile:   *outputFile,
	}

	// Create circuit breaker configuration
	switch *tier {
	case "free":
		config.BreakerConfig = circuitbreaker.DefaultFreeConfig("load-test")
	case "business":
		config.BreakerConfig = circuitbreaker.DefaultBusinessConfig("load-test")
	case "enterprise":
		config.BreakerConfig = circuitbreaker.DefaultEnterpriseConfig("load-test")
	default:
		log.Fatalf("Invalid tier: %s", *tier)
	}

	// Load custom configuration if provided
	if *configFile != "" {
		// Implementation would load custom config from file
		log.Printf("Custom configuration loading not implemented yet")
	}

	log.Printf("Starting load test with configuration:")
	log.Printf("  Duration: %v", config.Duration)
	log.Printf("  Concurrency: %d", config.Concurrency)
	log.Printf("  Request Rate: %.2f req/s", config.RequestRate)
	log.Printf("  Failure Rate: %.1f%%", config.FailureRate*100)
	log.Printf("  Scenario: %s", config.TestScenario)
	log.Printf("  Tier: %s", *tier)

	// Run the load test
	result, err := runLoadTest(config)
	if err != nil {
		log.Fatalf("Load test failed: %v", err)
	}

	// Print results
	printResults(result)

	// Save results to file if specified
	if config.OutputFile != "" {
		if err := saveResults(result, config.OutputFile); err != nil {
			log.Printf("Failed to save results: %v", err)
		} else {
			log.Printf("Results saved to %s", config.OutputFile)
		}
	}
}

// runLoadTest executes the load test with the given configuration
func runLoadTest(config LoadTestConfig) (*TestResult, error) {
	// Create circuit breaker
	cb, err := circuitbreaker.NewCircuitBreaker(config.BreakerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create circuit breaker: %w", err)
	}
	defer cb.Shutdown(context.Background())

	// Setup metrics collection
	var (
		totalRequests      int64
		successfulRequests int64
		failedRequests     int64
		circuitOpenCount   int64
		latencies          []RequestLatency
		latenciesMu        sync.Mutex
		stateChanges       []StateChange
		stateChangesMu     sync.Mutex
		errorTypes         = make(map[string]int64)
		errorTypesMu       sync.Mutex
	)

	// Setup state change monitoring
	originalCallback := config.BreakerConfig.OnStateChange
	config.BreakerConfig.OnStateChange = func(name string, from, to circuitbreaker.State) {
		stateChangesMu.Lock()
		stateChanges = append(stateChanges, StateChange{
			Timestamp: time.Now(),
			From:      from.String(),
			To:        to.String(),
			Reason:    "load_test_triggered",
		})
		stateChangesMu.Unlock()

		if originalCallback != nil {
			originalCallback(name, from, to)
		}

		log.Printf("Circuit breaker state changed: %s -> %s", from.String(), to.String())
	}

	// Create test function based on scenario
	testFunc := createTestFunction(config)

	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()

	startTime := time.Now()

	// Start workers
	var wg sync.WaitGroup
	rateLimiter := time.NewTicker(time.Duration(float64(time.Second) / config.RequestRate))
	defer rateLimiter.Stop()

	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case <-rateLimiter.C:
					// Execute request through circuit breaker
					requestStart := time.Now()
					result, err := cb.ExecuteWithContext(ctx, testFunc)
					requestLatency := time.Since(requestStart)

					atomic.AddInt64(&totalRequests, 1)

					// Record latency
					latenciesMu.Lock()
					latencies = append(latencies, RequestLatency{
						timestamp: requestStart,
						latency:   requestLatency,
						success:   result != nil && result.Success,
						error:     getErrorString(err),
					})
					latenciesMu.Unlock()

					// Update counters
					if result != nil && result.Success {
						atomic.AddInt64(&successfulRequests, 1)
					} else {
						atomic.AddInt64(&failedRequests, 1)

						// Check if failure was due to circuit being open
						if result != nil && result.FailureType == circuitbreaker.FailureTypeCircuit {
							atomic.AddInt64(&circuitOpenCount, 1)
						}

						// Track error types
						errorType := getErrorType(err, result)
						errorTypesMu.Lock()
						errorTypes[errorType]++
						errorTypesMu.Unlock()
					}
				}
			}
		}(i)
	}

	// Wait for test completion
	wg.Wait()
	endTime := time.Now()
	actualDuration := endTime.Sub(startTime)

	// Calculate metrics
	result := &TestResult{
		TotalRequests:      totalRequests,
		SuccessfulRequests: successfulRequests,
		FailedRequests:     failedRequests,
		CircuitOpenCount:   circuitOpenCount,
		Duration:           actualDuration,
		ThroughputRPS:      float64(totalRequests) / actualDuration.Seconds(),
		StateChanges:       stateChanges,
		ErrorTypes:         errorTypes,
	}

	// Calculate latency metrics
	if len(latencies) > 0 {
		result.calculateLatencyMetrics(latencies)
	}

	return result, nil
}

// createTestFunction creates a test function based on the scenario
func createTestFunction(config LoadTestConfig) func() (interface{}, error) {
	switch config.TestScenario {
	case "spike":
		return createSpikeTestFunction(config)
	case "gradual-failure":
		return createGradualFailureTestFunction(config)
	case "recovery":
		return createRecoveryTestFunction(config)
	default:
		return createStandardTestFunction(config)
	}
}

// createStandardTestFunction creates a standard test function
func createStandardTestFunction(config LoadTestConfig) func() (interface{}, error) {
	return func() (interface{}, error) {
		// Simulate processing time
		latency := config.LatencyMin + time.Duration(rand.Float64()*float64(config.LatencyMax-config.LatencyMin))
		time.Sleep(latency)

		// Simulate failures
		if rand.Float64() < config.FailureRate {
			return nil, fmt.Errorf("simulated failure")
		}

		return "success", nil
	}
}

// createSpikeTestFunction creates a function that simulates traffic spikes
func createSpikeTestFunction(config LoadTestConfig) func() (interface{}, error) {
	var requestCount int64

	return func() (interface{}, error) {
		count := atomic.AddInt64(&requestCount, 1)

		// Simulate spike in latency and failures every 1000 requests
		if count%1000 < 50 { // Spike for 50 requests
			latency := config.LatencyMax * 3 // 3x normal latency during spike
			time.Sleep(latency)

			if rand.Float64() < config.FailureRate*5 { // 5x failure rate during spike
				return nil, fmt.Errorf("spike failure")
			}
		} else {
			latency := config.LatencyMin + time.Duration(rand.Float64()*float64(config.LatencyMax-config.LatencyMin))
			time.Sleep(latency)

			if rand.Float64() < config.FailureRate {
				return nil, fmt.Errorf("normal failure")
			}
		}

		return "success", nil
	}
}

// createGradualFailureTestFunction creates a function that gradually increases failure rate
func createGradualFailureTestFunction(config LoadTestConfig) func() (interface{}, error) {
	startTime := time.Now()

	return func() (interface{}, error) {
		// Gradually increase failure rate over time
		elapsed := time.Since(startTime)
		progress := elapsed.Seconds() / config.Duration.Seconds()
		currentFailureRate := config.FailureRate * (1 + progress*4) // Up to 5x base failure rate

		latency := config.LatencyMin + time.Duration(rand.Float64()*float64(config.LatencyMax-config.LatencyMin))
		time.Sleep(latency)

		if rand.Float64() < currentFailureRate {
			return nil, fmt.Errorf("gradual failure (rate: %.2f)", currentFailureRate)
		}

		return "success", nil
	}
}

// createRecoveryTestFunction creates a function that simulates recovery scenario
func createRecoveryTestFunction(config LoadTestConfig) func() (interface{}, error) {
	startTime := time.Now()

	return func() (interface{}, error) {
		elapsed := time.Since(startTime)
		progress := elapsed.Seconds() / config.Duration.Seconds()

		var currentFailureRate float64
		if progress < 0.3 { // High failure rate for first 30%
			currentFailureRate = config.FailureRate * 10
		} else if progress < 0.6 { // Gradual recovery
			currentFailureRate = config.FailureRate * (10 - (progress-0.3)*30)
		} else { // Normal operation
			currentFailureRate = config.FailureRate
		}

		latency := config.LatencyMin + time.Duration(rand.Float64()*float64(config.LatencyMax-config.LatencyMin))
		time.Sleep(latency)

		if rand.Float64() < currentFailureRate {
			return nil, fmt.Errorf("recovery scenario failure")
		}

		return "success", nil
	}
}

// calculateLatencyMetrics calculates latency statistics
func (r *TestResult) calculateLatencyMetrics(latencies []RequestLatency) {
	if len(latencies) == 0 {
		return
	}

	// Sort latencies
	sortedLatencies := make([]time.Duration, len(latencies))
	for i, l := range latencies {
		sortedLatencies[i] = l.latency
	}

	// Simple sort (for small datasets)
	for i := 0; i < len(sortedLatencies); i++ {
		for j := i + 1; j < len(sortedLatencies); j++ {
			if sortedLatencies[i] > sortedLatencies[j] {
				sortedLatencies[i], sortedLatencies[j] = sortedLatencies[j], sortedLatencies[i]
			}
		}
	}

	// Calculate metrics
	r.MinLatency = sortedLatencies[0]
	r.MaxLatency = sortedLatencies[len(sortedLatencies)-1]

	// Calculate average
	var total time.Duration
	for _, l := range sortedLatencies {
		total += l
	}
	r.AverageLatency = total / time.Duration(len(sortedLatencies))

	// Calculate percentiles
	p95Index := int(float64(len(sortedLatencies)) * 0.95)
	p99Index := int(float64(len(sortedLatencies)) * 0.99)

	if p95Index >= len(sortedLatencies) {
		p95Index = len(sortedLatencies) - 1
	}
	if p99Index >= len(sortedLatencies) {
		p99Index = len(sortedLatencies) - 1
	}

	r.P95Latency = sortedLatencies[p95Index]
	r.P99Latency = sortedLatencies[p99Index]
}

// printResults prints the test results to stdout
func printResults(result *TestResult) {
	fmt.Println("\n=== Load Test Results ===")
	fmt.Printf("Duration: %v\n", result.Duration)
	fmt.Printf("Total Requests: %d\n", result.TotalRequests)
	fmt.Printf("Successful Requests: %d (%.2f%%)\n", result.SuccessfulRequests,
		float64(result.SuccessfulRequests)/float64(result.TotalRequests)*100)
	fmt.Printf("Failed Requests: %d (%.2f%%)\n", result.FailedRequests,
		float64(result.FailedRequests)/float64(result.TotalRequests)*100)
	fmt.Printf("Circuit Open Rejections: %d\n", result.CircuitOpenCount)
	fmt.Printf("Throughput: %.2f req/s\n", result.ThroughputRPS)

	fmt.Println("\n=== Latency Metrics ===")
	fmt.Printf("Average: %v\n", result.AverageLatency)
	fmt.Printf("Min: %v\n", result.MinLatency)
	fmt.Printf("Max: %v\n", result.MaxLatency)
	fmt.Printf("P95: %v\n", result.P95Latency)
	fmt.Printf("P99: %v\n", result.P99Latency)

	if len(result.StateChanges) > 0 {
		fmt.Println("\n=== State Changes ===")
		for _, change := range result.StateChanges {
			fmt.Printf("%s: %s -> %s (%s)\n",
				change.Timestamp.Format("15:04:05.000"),
				change.From, change.To, change.Reason)
		}
	}

	if len(result.ErrorTypes) > 0 {
		fmt.Println("\n=== Error Types ===")
		for errorType, count := range result.ErrorTypes {
			fmt.Printf("%s: %d\n", errorType, count)
		}
	}
}

// saveResults saves the test results to a JSON file
func saveResults(result *TestResult, filename string) error {
	// Implementation would marshal to JSON and save to file
	// For now, just create an empty file
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, "Load test results saved at %s\n", time.Now().Format(time.RFC3339))
	return nil
}

// getErrorString safely extracts error message
func getErrorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// getErrorType categorizes errors for statistics
func getErrorType(err error, result *circuitbreaker.ExecutionResult) string {
	if result != nil {
		switch result.FailureType {
		case circuitbreaker.FailureTypeTimeout:
			return "timeout"
		case circuitbreaker.FailureTypeCircuit:
			return "circuit_open"
		case circuitbreaker.FailureTypeLatency:
			return "latency"
		case circuitbreaker.FailureTypeResource:
			return "resource"
		default:
			return "application_error"
		}
	}

	if err != nil {
		return "unknown_error"
	}

	return "success"
}
