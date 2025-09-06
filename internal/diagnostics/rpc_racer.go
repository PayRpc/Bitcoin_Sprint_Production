package diagnostics

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/relay"
	"go.uber.org/zap"
)

// RacingConfig holds configuration for competitive RPC racing
type RacingConfig struct {
	MaxConcurrentRaces  int           `json:"max_concurrent_races"`
	RaceTimeout         time.Duration `json:"race_timeout"`
	RetryAttempts       int           `json:"retry_attempts"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthCooldown      time.Duration `json:"health_cooldown"`
	MaxResponseBytes    int64         `json:"max_response_bytes"`
}

// RaceResult contains the result of a single RPC race
type RaceResult struct {
	Response     []byte        `json:"-"`
	Latency      time.Duration `json:"latency_ms"`
	Endpoint     string        `json:"endpoint"`
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
	StatusCode   int           `json:"status_code,omitempty"`
	ResponseSize int64         `json:"response_size"`
	RaceID       int64         `json:"race_id"`
}

// RPCRacer manages competitive endpoint racing for optimal performance
type RPCRacer struct {
	endpoints   []*relay.EndpointHealth // Use existing EndpointHealth from relay package
	config      RacingConfig
	chain       string
	probeMethod string
	raceCounter int64
	monitor     *P2PMonitor
	logger      *zap.Logger
	client      *http.Client
	mu          sync.RWMutex
}

// NewRPCRacer creates a production-ready RPC racing engine
func NewRPCRacer(endpoints []string, config RacingConfig, chain string, probeMethod string, monitor *P2PMonitor, logger *zap.Logger) *RPCRacer {
	if config.MaxConcurrentRaces <= 0 {
		config.MaxConcurrentRaces = 5
	}
	if config.RaceTimeout == 0 {
		config.RaceTimeout = 10 * time.Second
	}
	if config.MaxResponseBytes == 0 {
		config.MaxResponseBytes = 10 * 1024 * 1024 // 10MB
	}

	healthyEndpoints := make([]*relay.EndpointHealth, len(endpoints))
	for i, url := range endpoints {
		healthyEndpoints[i] = relay.NewEndpointHealth(url)
	}

	return &RPCRacer{
		endpoints:   healthyEndpoints,
		config:      config,
		chain:       chain,
		probeMethod: probeMethod,
		monitor:     monitor,
		logger:      logger,
		client: &http.Client{
			Timeout: config.RaceTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     30 * time.Second,
			},
		},
	}
}

// Race executes competitive RPC calls and returns the fastest successful response
func (r *RPCRacer) Race(ctx context.Context, reqBody []byte) (*RaceResult, error) {
	raceID := atomic.AddInt64(&r.raceCounter, 1)

	r.mu.RLock()
	activeEndpoints := make([]*relay.EndpointHealth, 0, len(r.endpoints))
	for _, ep := range r.endpoints {
		if ep.IsHealthy() {
			activeEndpoints = append(activeEndpoints, ep)
		}
	}
	r.mu.RUnlock()

	if len(activeEndpoints) == 0 {
		return nil, fmt.Errorf("no healthy endpoints available for %s", r.chain)
	}

	// Limit concurrent races
	maxRaces := min(len(activeEndpoints), r.config.MaxConcurrentRaces)
	results := make(chan *RaceResult, maxRaces)
	raceCtx, cancel := context.WithTimeout(ctx, r.config.RaceTimeout)
	defer cancel()

	// Launch races
	for i := 0; i < maxRaces; i++ {
		go r.raceEndpointWithRetries(raceCtx, activeEndpoints[i], reqBody, results, raceID)
	}

	// Wait for first successful result
	for i := 0; i < maxRaces; i++ {
		select {
		case result := <-results:
			if result.Success {
				r.recordSuccess(result.Endpoint, result.Latency)
				return result, nil
			}
			r.recordFailure(result.Endpoint, result.Error)
		case <-raceCtx.Done():
			return nil, fmt.Errorf("race timeout for %s", r.chain)
		}
	}

	return nil, fmt.Errorf("all endpoints failed for %s", r.chain)
}

// raceEndpointWithRetries executes RPC call with retries
func (r *RPCRacer) raceEndpointWithRetries(ctx context.Context, endpoint *relay.EndpointHealth, reqBody []byte, results chan<- *RaceResult, raceID int64) {
	for attempt := 0; attempt <= r.config.RetryAttempts; attempt++ {
		start := time.Now()

		result := r.executeRPCCall(ctx, endpoint.URL, reqBody, raceID)
		result.Latency = time.Since(start)

		if result.Success {
			results <- result
			return
		}

		// Record diagnostic info
		if r.monitor != nil {
			r.monitor.RecordAttempt(r.chain, AttemptRecord{
				Address:          endpoint.URL,
				Timestamp:        start,
				TcpSuccess:       result.StatusCode > 0,
				TcpError:         result.Error,
				HandshakeSuccess: result.Success,
				HandshakeError:   result.Error,
				ConnectLatency:   result.Latency,
				ResponseTime:     result.Latency,
			})
		}

		// On final attempt, send result anyway
		if attempt == r.config.RetryAttempts {
			results <- result
			return
		}

		// Brief retry delay
		select {
		case <-ctx.Done():
			results <- &RaceResult{Success: false, Error: "context cancelled", Endpoint: endpoint.URL, RaceID: raceID}
			return
		case <-time.After(time.Duration(attempt*100) * time.Millisecond):
			// Continue to next attempt
		}
	}
}

// executeRPCCall performs the actual HTTP request
func (r *RPCRacer) executeRPCCall(ctx context.Context, endpoint string, reqBody []byte, raceID int64) *RaceResult {
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, nil)
	if err != nil {
		return &RaceResult{
			Success:  false,
			Error:    fmt.Sprintf("request creation failed: %v", err),
			Endpoint: endpoint,
			RaceID:   raceID,
		}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(strings.NewReader(string(reqBody)))

	resp, err := r.client.Do(req)
	if err != nil {
		return &RaceResult{
			Success:  false,
			Error:    fmt.Sprintf("request failed: %v", err),
			Endpoint: endpoint,
			RaceID:   raceID,
		}
	}
	defer resp.Body.Close()

	// Limit response size
	limitedReader := io.LimitReader(resp.Body, r.config.MaxResponseBytes)
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return &RaceResult{
			Success:    false,
			Error:      fmt.Sprintf("response read failed: %v", err),
			Endpoint:   endpoint,
			StatusCode: resp.StatusCode,
			RaceID:     raceID,
		}
	}

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	errorMsg := ""
	if !success {
		errorMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return &RaceResult{
		Response:     respBody,
		Success:      success,
		Error:        errorMsg,
		Endpoint:     endpoint,
		StatusCode:   resp.StatusCode,
		ResponseSize: int64(len(respBody)),
		RaceID:       raceID,
	}
}

// recordSuccess updates endpoint health on successful operation
func (r *RPCRacer) recordSuccess(endpoint string, latency time.Duration) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, ep := range r.endpoints {
		if ep.URL == endpoint {
			ep.RecordSuccess(latency)
			break
		}
	}
}

// recordFailure updates endpoint health on failed operation
func (r *RPCRacer) recordFailure(endpoint string, errorMsg string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, ep := range r.endpoints {
		if ep.URL == endpoint {
			ep.RecordError()
			break
		}
	}
}

// GetHealthyEndpoints returns currently healthy endpoints
func (r *RPCRacer) GetHealthyEndpoints() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var healthy []string
	for _, ep := range r.endpoints {
		if ep.IsHealthy() {
			healthy = append(healthy, ep.URL)
		}
	}
	return healthy
}

// Helper function for Go < 1.21 compatibility
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
