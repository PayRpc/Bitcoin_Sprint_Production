package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// AttemptRecord holds one connection attempt result
type AttemptRecord struct {
	Address          string        `json:"address"`
	Timestamp        time.Time     `json:"timestamp"`
	TcpSuccess       bool          `json:"tcp_success"`
	TcpError         string        `json:"tcp_error,omitempty"`
	HandshakeSuccess bool          `json:"handshake_success"`
	HandshakeError   string        `json:"handshake_error,omitempty"`
	ConnectLatency   time.Duration `json:"connect_latency,omitempty"`
	ResponseTime     time.Duration `json:"response_time,omitempty"`
}

// RacingConfig holds configuration for competitive RPC racing
type RacingConfig struct {
	MaxConcurrentRaces  int           `json:"max_concurrent_races"`
	RaceTimeout         time.Duration `json:"race_timeout"`
	RetryAttempts       int           `json:"retry_attempts"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthCooldown      time.Duration `json:"health_cooldown"`
	MaxResponseBytes    int64         `json:"max_response_bytes"`
}

// EndpointHealth tracks health and performance per endpoint
type EndpointHealth struct {
	URL              string        `json:"url"`
	AvgLatency       time.Duration `json:"avg_latency_ms"`
	SuccessRate      float64       `json:"success_rate"`
	LastSuccess      time.Time     `json:"last_success"`
	ConsecutiveFails int           `json:"consecutive_fails"`
	IsHealthy        bool          `json:"is_healthy"`
	Region           string        `json:"region,omitempty"`
	lastUnhealthyAt  time.Time     `json:"-"` // Internal cooldown tracking
	mu               sync.RWMutex
}

// RaceResult contains the result of a single RPC race
type RaceResult struct {
	Response     []byte        `json:"-"`
	Latency      time.Duration `json:"latency_ms"`
	Endpoint     string        `json:"endpoint"`
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
	StatusCode   int           `json:"status_code"`
	ResponseSize int           `json:"response_size"`
}

// RPCRacer manages competitive RPC request racing
type RPCRacer struct {
	endpoints       []*EndpointHealth
	config          RacingConfig
	httpClient      *http.Client
	activeRaces     atomic.Int64
	totalRaces      atomic.Int64
	winsPerEndpoint map[string]int64
	mu              sync.RWMutex
	// chain context for diagnostics and probe selection
	chain       string
	probeMethod string
}

// Rolling buffer per protocol
type diagBuffer struct {
	Attempts    []AttemptRecord
	LastError   string
	DialedPeers []string
	mu          sync.Mutex
}

var (
	diagState = map[string]*diagBuffer{
		"bitcoin":  {Attempts: make([]AttemptRecord, 0, 50), DialedPeers: make([]string, 0, 20)},
		"ethereum": {Attempts: make([]AttemptRecord, 0, 50), DialedPeers: make([]string, 0, 20)},
		"solana":   {Attempts: make([]AttemptRecord, 0, 50), DialedPeers: make([]string, 0, 20)},
	}
)

// RecordAttempt safely appends to buffer for given chain
func RecordAttempt(protocol string, rec AttemptRecord) {
	buf, ok := diagState[protocol]
	if !ok {
		return
	}
	buf.mu.Lock()
	defer buf.mu.Unlock()

	if len(buf.Attempts) >= 50 {
		buf.Attempts = buf.Attempts[1:] // drop oldest
	}
	buf.Attempts = append(buf.Attempts, rec)

	if rec.HandshakeError != "" || rec.TcpError != "" {
		buf.LastError = rec.HandshakeError
		if buf.LastError == "" {
			buf.LastError = rec.TcpError
		} else if rec.TcpError != "" {
			buf.LastError += "; " + rec.TcpError
		}
	}
	if rec.Address != "" {
		buf.DialedPeers = append(buf.DialedPeers, rec.Address)
		if len(buf.DialedPeers) > 20 {
			buf.DialedPeers = buf.DialedPeers[1:]
		}
	}
}

// SetLastError sets the last error message for a protocol in diagnostics
func SetLastError(protocol string, msg string) {
	buf, ok := diagState[protocol]
	if !ok {
		return
	}
	buf.mu.Lock()
	buf.LastError = msg
	buf.mu.Unlock()
}

// snapshotBuffer returns a copy-safe view of the diag buffer for JSON
func snapshotBuffer(protocol string) map[string]interface{} {
	buf, ok := diagState[protocol]
	if !ok || buf == nil {
		return map[string]interface{}{
			"connection_attempts": []AttemptRecord{},
			"last_error":          "",
			"dialed_peers":        []string{},
		}
	}
	buf.mu.Lock()
	attempts := make([]AttemptRecord, len(buf.Attempts))
	copy(attempts, buf.Attempts)
	peers := make([]string, len(buf.DialedPeers))
	copy(peers, buf.DialedPeers)
	lastErr := buf.LastError
	buf.mu.Unlock()
	return map[string]interface{}{
		"connection_attempts": attempts,
		"last_error":          lastErr,
		"dialed_peers":        peers,
	}
}

// p2pDiagHandler returns peer diagnostics
// Commented out due to undefined types (Server, ProtocolType, etc.)
/*
func (s *Server) p2pDiagHandler(w http.ResponseWriter, r *http.Request) {
	// Build response from existing p2pClients map
	clients := map[string]interface{}{}

	// Known protocols
	protocols := []ProtocolType{ProtocolBitcoin, ProtocolEthereum, ProtocolSolana}
	for _, p := range protocols {
		var count int
		ids := []string{}
		if c := s.p2pClients[p]; c != nil {
			count = c.GetPeerCount()
			ids = c.GetPeerIDs()
		}
		snap := snapshotBuffer(string(p))
		backendStatus := "online"
		if count == 0 {
			// Graceful degradation signal for UI
			backendStatus = "fallback_rpc"
		}
		clients[string(p)] = map[string]interface{}{
			"peer_count":     count,
			"peer_ids":       ids,
			"backend_status": backendStatus,
			// merge snapshot fields
			"connection_attempts": snap["connection_attempts"],
			"last_error":          snap["last_error"],
			"dialed_peers":        snap["dialed_peers"],
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"p2p_clients": clients,
		"ts":          time.Now().UTC().Format(time.RFC3339),
	})
}
*/

// NewRPCRacer creates a new competitive RPC racing engine
// chain: e.g. "ethereum" or "solana" (used for diagnostics)
// probeMethod: JSON-RPC method used for periodic health checks (e.g. "eth_blockNumber", "getSlot")
func NewRPCRacer(endpoints []string, config RacingConfig, chain string, probeMethod string) *RPCRacer {
	// Set sensible defaults for new config options
	if config.HealthCooldown == 0 {
		config.HealthCooldown = 20 * time.Second
	}
	if config.MaxResponseBytes == 0 {
		config.MaxResponseBytes = 2 << 20 // 2 MiB
	}

	healthyEndpoints := make([]*EndpointHealth, len(endpoints))
	for i, ep := range endpoints {
		healthyEndpoints[i] = &EndpointHealth{
			URL:         ep,
			IsHealthy:   true,
			LastSuccess: time.Now(),
		}
	}

	// Optimized HTTP client for maximum speed
	client := &http.Client{
		Timeout: config.RaceTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  false, // Enable compression for efficiency
			ForceAttemptHTTP2:   true,  // HTTP/2 for better performance
		},
	}

	racer := &RPCRacer{
		endpoints:       healthyEndpoints,
		config:          config,
		httpClient:      client,
		winsPerEndpoint: make(map[string]int64),
		chain:           chain,
		probeMethod:     probeMethod,
	}

	// Start background health monitoring
	go racer.healthMonitor()

	return racer
}

// RaceRequest races multiple endpoints and returns the fastest successful response
func (r *RPCRacer) RaceRequest(ctx context.Context, method string, params interface{}) (*RaceResult, error) {
	currentRace := r.totalRaces.Add(1)
	r.activeRaces.Add(1)
	defer r.activeRaces.Add(-1)

	startTime := time.Now()

	// Prepare JSON-RPC request
	reqBody, err := r.prepareRPCRequest(method, params)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare request: %v", err)
	}

	// Get healthy endpoints, sorted by performance
	healthyEndpoints := r.getHealthyEndpointsSorted()
	if len(healthyEndpoints) == 0 {
		return nil, fmt.Errorf("no healthy endpoints available")
	}

	// Limit concurrent races for resource management
	maxRacers := minInt(len(healthyEndpoints), r.config.MaxConcurrentRaces)

	// Race context with timeout
	raceCtx, cancel := context.WithTimeout(ctx, r.config.RaceTimeout)
	defer cancel()

	// Channel for race results
	results := make(chan *RaceResult, maxRacers)

	// Start racing goroutines
	for i := 0; i < maxRacers; i++ {
		go r.raceEndpointWithRetries(raceCtx, healthyEndpoints[i], reqBody, results, currentRace)
	}

	// Wait for first successful result or all failures
	var bestResult *RaceResult
	var lastError error

	for i := 0; i < maxRacers; i++ {
		select {
		case result := <-results:
			if result.Success && bestResult == nil {
				// First successful result wins the race
				bestResult = result
				r.recordWin(result.Endpoint)
				// cancel remaining goroutines immediately
				cancel()
				// Update diagnostics
				RecordAttempt(r.chain, AttemptRecord{
					Address:          result.Endpoint,
					Timestamp:        time.Now(),
					TcpSuccess:       true,
					HandshakeSuccess: true,
					ConnectLatency:   result.Latency,
					ResponseTime:     result.Latency,
				})
			}

			if !result.Success {
				lastError = fmt.Errorf("endpoint %s failed: %s", result.Endpoint, result.Error)
				// Record failure for diagnostics
				RecordAttempt(r.chain, AttemptRecord{
					Address:          result.Endpoint,
					Timestamp:        time.Now(),
					TcpSuccess:       false,
					HandshakeSuccess: false,
					TcpError:         result.Error,
				})
			}

		case <-raceCtx.Done():
			if bestResult == nil {
				return nil, fmt.Errorf("race timeout after %v", r.config.RaceTimeout)
			}
		}

		if bestResult != nil {
			break
		}
	}

	if bestResult != nil {
		totalLatency := time.Since(startTime)
		fmt.Printf("🏁 Race #%d won by %s in %v (total race time: %v)\n",
			currentRace, bestResult.Endpoint, bestResult.Latency, totalLatency)
		return bestResult, nil
	}

	return nil, fmt.Errorf("all endpoints failed, last error: %v", lastError)
}

// raceEndpointWithRetries performs per-endpoint retries with jittered backoff
func (r *RPCRacer) raceEndpointWithRetries(ctx context.Context, endpoint *EndpointHealth, reqBody []byte, results chan<- *RaceResult, raceID int64) {
	attempts := maxInt(1, r.config.RetryAttempts)
	var last *RaceResult
	for i := 0; i < attempts; i++ {
		if ctx.Err() != nil {
			break
		}
		res := r.singleRPC(ctx, endpoint, reqBody, raceID)
		last = res
		if res.Success {
			results <- res
			return
		}
		// jittered exponential backoff
		backoff := jitterBackoff(i)
		select {
		case <-ctx.Done():
			break
		case <-time.After(backoff):
		}
	}
	if last == nil {
		last = &RaceResult{Endpoint: endpoint.URL, Success: false, Error: "no attempt executed"}
	}
	results <- last
}

// singleRPC issues one HTTP JSON-RPC request and updates health
func (r *RPCRacer) singleRPC(ctx context.Context, endpoint *EndpointHealth, reqBody []byte, raceID int64) *RaceResult {
	startTime := time.Now()

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint.URL, bytes.NewBuffer(reqBody))
	if err != nil {
		return &RaceResult{Endpoint: endpoint.URL, Success: false, Error: "request creation failed: " + err.Error(), Latency: time.Since(startTime)}
	}

	// Set competitive headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "bitcoin-sprint/1.0")
	// Do NOT set Accept-Encoding manually; http.Transport manages gzip.

	resp, err := r.httpClient.Do(req)
	if err != nil {
		latency := time.Since(startTime)
		r.updateEndpointHealth(endpoint, false, latency)
		return &RaceResult{Endpoint: endpoint.URL, Success: false, Error: "request failed: " + err.Error(), Latency: latency}
	}
	defer resp.Body.Close()

	latency := time.Since(startTime)

	// Read response with size limit + overflow detection (N+1)
	max := r.config.MaxResponseBytes
	if max <= 0 {
		max = 2 << 20
	}
	limited := &io.LimitedReader{R: resp.Body, N: max + 1}
	body, err := io.ReadAll(limited)
	if err != nil {
		r.updateEndpointHealth(endpoint, false, latency)
		return &RaceResult{Endpoint: endpoint.URL, Success: false, Error: "response read failed: " + err.Error(), StatusCode: resp.StatusCode, Latency: latency}
	}
	if int64(len(body)) > max {
		r.updateEndpointHealth(endpoint, false, latency)
		return &RaceResult{Endpoint: endpoint.URL, Success: false, Error: fmt.Sprintf("response too large (>%d bytes)", max), StatusCode: resp.StatusCode, Latency: latency}
	}

	ok := resp.StatusCode == http.StatusOK && !jsonRPCError(body)
	r.updateEndpointHealth(endpoint, ok, latency)
	if ok {
		RecordAttempt(r.chain, AttemptRecord{
			Address: endpoint.URL, Timestamp: time.Now(),
			TcpSuccess: true, HandshakeSuccess: true,
			ConnectLatency: latency, ResponseTime: latency,
		})
		return &RaceResult{Response: body, Endpoint: endpoint.URL, Success: true, StatusCode: resp.StatusCode, ResponseSize: len(body), Latency: latency}
	}
	errMsg := "http " + strconv.Itoa(resp.StatusCode)
	if e := extractJSONRPCError(body); e != "" {
		errMsg = e
	}
	RecordAttempt(r.chain, AttemptRecord{
		Address: endpoint.URL, Timestamp: time.Now(),
		TcpSuccess: false, HandshakeSuccess: false, TcpError: errMsg,
	})
	return &RaceResult{Endpoint: endpoint.URL, Success: false, Error: errMsg, StatusCode: resp.StatusCode, ResponseSize: len(body), Latency: latency}
}

// prepareRPCRequest creates a JSON-RPC request body
func (r *RPCRacer) prepareRPCRequest(method string, params interface{}) ([]byte, error) {
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	}
	return json.Marshal(request)
}

// getHealthyEndpointsSorted returns healthy endpoints sorted by performance
func (r *RPCRacer) getHealthyEndpointsSorted() []*EndpointHealth {
	r.mu.RLock()
	defer r.mu.RUnlock()

	healthy := make([]*EndpointHealth, 0, len(r.endpoints))
	now := time.Now()

	for _, ep := range r.endpoints {
		ep.mu.RLock()
		isHealthy := ep.IsHealthy
		lastUnhealthy := ep.lastUnhealthyAt
		ep.mu.RUnlock()

		// Apply health cooldown - don't use recently failed endpoints
		if !isHealthy && !lastUnhealthy.IsZero() && now.Sub(lastUnhealthy) < r.config.HealthCooldown {
			continue // Skip endpoints in cooldown period
		}

		// Re-enable endpoints after cooldown
		if !isHealthy && !lastUnhealthy.IsZero() && now.Sub(lastUnhealthy) >= r.config.HealthCooldown {
			ep.mu.Lock()
			ep.IsHealthy = true     // Give it another chance
			ep.ConsecutiveFails = 0 // Reset failure count
			ep.mu.Unlock()
		}

		if isHealthy || now.Sub(lastUnhealthy) >= r.config.HealthCooldown {
			healthy = append(healthy, ep)
		}
	}

	// Sort by performance (lowest latency first, then by success rate)
	// This gives faster endpoints priority in racing
	for i := 0; i < len(healthy)-1; i++ {
		for j := i + 1; j < len(healthy); j++ {
			healthy[i].mu.RLock()
			healthy[j].mu.RLock()

			iLatency := healthy[i].AvgLatency
			jLatency := healthy[j].AvgLatency
			iSuccess := healthy[i].SuccessRate
			jSuccess := healthy[j].SuccessRate

			healthy[i].mu.RUnlock()
			healthy[j].mu.RUnlock()

			// Prefer lower latency, then higher success rate
			if iLatency > jLatency || (iLatency == jLatency && iSuccess < jSuccess) {
				healthy[i], healthy[j] = healthy[j], healthy[i]
			}
		}
	}

	return healthy
}

// updateEndpointHealth updates health metrics for an endpoint with cooldown logic
func (r *RPCRacer) updateEndpointHealth(endpoint *EndpointHealth, success bool, latency time.Duration) {
	endpoint.mu.Lock()
	defer endpoint.mu.Unlock()

	if success {
		// Exponential moving average for latency
		if endpoint.AvgLatency == 0 {
			endpoint.AvgLatency = latency
		} else {
			// 20% weight for new measurement
			endpoint.AvgLatency = time.Duration(0.8*float64(endpoint.AvgLatency) + 0.2*float64(latency))
		}

		endpoint.LastSuccess = time.Now()
		endpoint.ConsecutiveFails = 0
		endpoint.IsHealthy = true

		// Update success rate (simplified)
		endpoint.SuccessRate = math.Min(endpoint.SuccessRate*0.9+0.1, 1.0)
	} else {
		endpoint.ConsecutiveFails++
		endpoint.SuccessRate = math.Max(endpoint.SuccessRate*0.9, 0.0)

		// Mark unhealthy after 3 consecutive failures
		if endpoint.ConsecutiveFails >= 3 {
			endpoint.IsHealthy = false
			endpoint.lastUnhealthyAt = time.Now() // Start cooldown period
		}
	}
}

// recordWin tracks which endpoint wins races (for optimization)
func (r *RPCRacer) recordWin(endpoint string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.winsPerEndpoint[endpoint]++
}

// healthMonitor runs periodic health checks on endpoints
func (r *RPCRacer) healthMonitor() {
	ticker := time.NewTicker(r.config.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		r.performHealthChecks()
	}
}

// performHealthChecks pings all endpoints to maintain health status
func (r *RPCRacer) performHealthChecks() {
	for _, endpoint := range r.endpoints {
		go func(ep *EndpointHealth) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			// Minimal JSON-RPC probe; single request (chain-specific)
			method := r.probeMethod
			if method == "" {
				method = "getSlot"
			}
			probe, _ := json.Marshal(map[string]any{
				"jsonrpc": "2.0",
				"method":  method,
				"params":  []any{},
				"id":      1,
			})
			res := r.singleRPC(ctx, ep, probe, 0)
			r.updateEndpointHealth(ep, res.Success, res.Latency)
		}(endpoint)
	}
}

// GetRacingStats returns statistics about racing performance with enhanced metrics
func (r *RPCRacer) GetRacingStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	inCooldown := 0

	stats := map[string]interface{}{
		"total_races":       r.totalRaces.Load(),
		"active_races":      r.activeRaces.Load(),
		"wins_per_endpoint": r.winsPerEndpoint,
		"endpoint_health":   make([]map[string]interface{}, len(r.endpoints)),
		"config": map[string]interface{}{
			"health_cooldown_sec": r.config.HealthCooldown.Seconds(),
			"max_response_mb":     float64(r.config.MaxResponseBytes) / (1024 * 1024),
			"race_timeout_ms":     r.config.RaceTimeout.Milliseconds(),
		},
	}

	for i, ep := range r.endpoints {
		ep.mu.RLock()
		cooldownRemaining := float64(0)
		if !ep.IsHealthy && !ep.lastUnhealthyAt.IsZero() {
			remaining := r.config.HealthCooldown - now.Sub(ep.lastUnhealthyAt)
			if remaining > 0 {
				cooldownRemaining = remaining.Seconds()
				inCooldown++
			}
		}

		stats["endpoint_health"].([]map[string]interface{})[i] = map[string]interface{}{
			"url":                    ep.URL,
			"is_healthy":             ep.IsHealthy,
			"avg_latency_ms":         ep.AvgLatency.Milliseconds(),
			"success_rate":           ep.SuccessRate,
			"consecutive_fails":      ep.ConsecutiveFails,
			"last_success":           ep.LastSuccess.Format(time.RFC3339),
			"cooldown_remaining_sec": cooldownRemaining,
		}
		ep.mu.RUnlock()
	}

	stats["endpoints_in_cooldown"] = inCooldown

	return stats
}

// Utility helpers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ±20% jittered exponential backoff, capped
func jitterBackoff(retry int) time.Duration {
	base := 50 * time.Millisecond
	max := 750 * time.Millisecond
	d := time.Duration(float64(base.Nanoseconds()) * math.Pow(2, float64(retry)))
	if d > max {
		d = max
	}
	// use a cheap LCG on top of time for jitter without pulling math/rand global
	n := time.Now().UnixNano()
	// -0.2 .. +0.2
	jitter := float64((n%400_000)-200_000) / 1_000_000.0
	return time.Duration(float64(d.Nanoseconds()) * (1.0 + jitter))
}

func jsonRPCError(b []byte) bool {
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		return false
	}
	_, has := m["error"]
	return has
}
func extractJSONRPCError(b []byte) string {
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		return ""
	}
	if e, ok := m["error"]; ok {
		bs, _ := json.Marshal(e)
		return string(bs)
	}
	return ""
}
