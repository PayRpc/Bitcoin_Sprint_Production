// SPDX-License-Identifier: MIT
// Bitcoin Sprint - RPC Edition (Enterprise-Ready)
// Copyright (c) 2025 BitcoinCab.inc

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Versioning - populated at build time via -ldflags
var (
	Version = "dev"
	Commit  = "unknown"
)

// ───────────────────────── Types ─────────────────────────

type Config struct {
	LicenseKey   string   `json:"license_key"`
	Tier         string   `json:"tier"`
	MetricsURL   string   `json:"metrics_url"`
	RPCNodes     []string `json:"rpc_nodes"`
	APIBase      string   `json:"api_base"`
	RPCUser      string   `json:"rpc_user"`
	RPCPass      string   `json:"rpc_pass"`
	PollInterval int      `json:"poll_interval"` // seconds, default 5
	TurboMode    bool     `json:"turbo_mode"`    // enable ultra-aggressive fan-out
}

type License struct {
	Valid      bool     `json:"valid"`
	Tier       string   `json:"tier"`
	BlockLimit int      `json:"block_limit"`
	Peers      []string `json:"peers"`
}

type Metrics struct {
	BlockHash  string  `json:"block_hash"`
	Latency    float64 `json:"latency_ms"`
	PeerCount  int     `json:"peer_count"`
	Timestamp  int64   `json:"timestamp"`
	LicenseKey string  `json:"license_key"`
	Height     int     `json:"height"`
	RPCNode    string  `json:"rpc_node"`
}

type Sprint struct {
	config      Config
	license     License
	peers       map[string]net.Conn
	metrics     chan Metrics
	blocksSent  int
	client      *http.Client
	nodeBackoff map[string]time.Time

	// HOT PATH OPTIMIZATIONS: Pre-marshaled requests
	getBlockchainInfoReq []byte
	preEncodedPayload    []byte

	// PERFORMANCE: Reduce lock contention
	blocksSentMu sync.RWMutex

	mu  sync.RWMutex
	ctx context.Context
}

// ───────────────────────── Main ─────────────────────────

func main() {
	// Quick version switch: print and exit if requested to avoid spinning the build script
	if len(os.Args) > 1 {
		if os.Args[1] == "--version" || os.Args[1] == "-v" {
			fmt.Printf("Bitcoin Sprint v%s (commit %s)\n", Version, Commit)
			return
		}
	}
	// Modern Go 1.20+ uses automatic random seeding
	// No need for explicit rand.Seed() call

	// Print application banner
	log.Printf("Bitcoin Sprint v%s - Enterprise Bitcoin Block Detection", Version)
	log.Printf("Copyright © 2025 BitcoinCab.inc")

	s := &Sprint{
		peers:   make(map[string]net.Conn),
		metrics: make(chan Metrics, 500),
		client: &http.Client{
			Timeout: 2 * time.Second, // OPTIMIZED: Faster timeout for hot path
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion:         tls.VersionTLS12,
					InsecureSkipVerify: false, // Enforce certificate validation
				},
				DialContext:           (&net.Dialer{Timeout: 1 * time.Second, KeepAlive: 60 * time.Second}).DialContext,
				ForceAttemptHTTP2:     false,           // OPTIMIZED: HTTP/1.1 for lower latency
				ResponseHeaderTimeout: 1 * time.Second, // OPTIMIZED: Ultra-fast response
				MaxIdleConns:          100,             // OPTIMIZED: Large connection pool
				MaxIdleConnsPerHost:   50,              // OPTIMIZED: High per-host limit
				IdleConnTimeout:       60 * time.Second,
			},
		},
		nodeBackoff: make(map[string]time.Time),
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx

	// HOT PATH OPTIMIZATION: Pre-marshal common requests at startup
	s.getBlockchainInfoReq = []byte(`{"jsonrpc":"1.0","id":"sprint","method":"getblockchaininfo","params":[]}`)

	// Pre-encode sprint payload template with placeholders
	payload := struct {
		Type     string `json:"type"`
		Hash     string `json:"hash"`
		Ts       string `json:"ts"`
		Version  string `json:"version"`
		Protocol int    `json:"protocol"`
	}{
		Type:     "block",
		Hash:     "HASH_PLACEHOLDER",
		Ts:       "TS_PLACEHOLDER",
		Version:  Version,
		Protocol: 1,
	}
	payloadBytes, _ := json.Marshal(payload)
	s.preEncodedPayload = append(payloadBytes, '\n') // Add newline for framing

	if err := s.LoadConfig(); err != nil {
		log.Fatal("Config error:", err)
	}
	if err := s.ValidateLicense(); err != nil {
		log.Fatal("License error:", err)
	}

	go s.StartWebDashboard()
	go s.StartBlockPoller()
	go s.ConnectToPeers()
	go s.StartMetricsReporter()

	// graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down Bitcoin Sprint RPC edition...")
		cancel()
		os.Exit(0)
	}()

	log.Println("Bitcoin Sprint (RPC edition) running...")
	// Log dashboard URL with actual port (SPRINT_DASH_PORT or PORT, default 8080)
	dashPort := os.Getenv("SPRINT_DASH_PORT")
	if dashPort == "" {
		dashPort = os.Getenv("PORT")
	}
	if dashPort == "" {
		dashPort = "8080"
	}
	log.Printf("Dashboard: http://localhost:%s", strings.TrimPrefix(dashPort, ":"))
	log.Printf("Tier: %s", s.license.Tier)

	mode := "SAFE"
	if s.config.TurboMode {
		mode = "TURBO"
	}
	log.Printf("⚡ Bitcoin Sprint running in %s mode", mode)

	<-s.ctx.Done()
}

// ───────────────────────── Block Poller ─────────────────────────

func (s *Sprint) StartBlockPoller() {
	var lastHash string
	var consecutiveErrors int

	// Default to 1s interval if not specified for optimal performance
	interval := time.Duration(s.config.PollInterval) * time.Second
	if interval == 0 {
		interval = 1 * time.Second // ultra-fast default for better performance
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Track last block time to optimize polling
	lastBlockTime := time.Now()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			var (
				hash   string
				height int
				node   string
				err    error
			)

			if s.config.TurboMode {
				hash, height, node, err = s.getBestBlockTurbo()
			} else {
				hash, height, node, err = s.getBestBlock()
			}
			if err != nil {
				log.Printf("RPC poll error: %v", err)
				consecutiveErrors++

				// Smart exponential backoff with jitter
				if consecutiveErrors > 3 {
					backoff := time.Duration(math.Min(float64(consecutiveErrors*2), 30)) * time.Second
					jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
					interval = backoff + jitter
					ticker.Reset(interval)
					log.Printf("Backing off for %v due to errors", interval)
				}
				continue
			}

			// Reset error counter on success
			consecutiveErrors = 0

			if hash != "" && hash != lastHash {
				lastHash = hash
				s.OnNewBlock(hash, height, node)
				lastBlockTime = time.Now()

				// ULTRA-AGGRESSIVE: 250ms polling after new block
				interval = 250 * time.Millisecond
				ticker.Reset(interval)
				log.Printf("New block detected - ULTRA-AGGRESSIVE polling: %v", interval)

				// BURST PROBE: Immediate rapid detection for next block
				go func(currentHash string) {
					if nextHash, nextHeight, nextNode, err := s.burstProbe(currentHash); err == nil {
						s.OnNewBlock(nextHash, nextHeight, nextNode)
					}
				}(hash)

				// Prefetch block details in background
				go s.prefetchBlock(hash, node)

				// Start parallel multi-node monitoring
				go s.startParallelMonitoring(hash, height)

				// Start predictive block monitoring
				if s.config.TurboMode {
					// Turbo: faster mempool predictor cadence
					go s.startPredictiveMonitoringTurbo()
				} else {
					go s.startPredictiveMonitoring()
				}
			} else {
				// AGGRESSIVE adaptive polling based on time since last block
				elapsed := time.Since(lastBlockTime)

				switch {
				case elapsed < 45*time.Second:
					// HYPER-TIGHT polling for 45s after block (250ms)
					if interval != 250*time.Millisecond {
						interval = 250 * time.Millisecond
						ticker.Reset(interval)
					}
				case elapsed < 2*time.Minute:
					// AGGRESSIVE polling for normal periods (500ms)
					if interval != 500*time.Millisecond {
						interval = 500 * time.Millisecond
						ticker.Reset(interval)
					}
				case elapsed < 5*time.Minute:
					// TIGHT polling during medium periods (1s)
					if interval != 1*time.Second {
						interval = 1 * time.Second
						ticker.Reset(interval)
					}
				case elapsed < 10*time.Minute:
					// STANDARD polling during quiet periods (2s)
					if interval != 2*time.Second {
						interval = 2 * time.Second
						ticker.Reset(interval)
					}
				default:
					// MINIMUM relaxation during long quiet periods (5s)
					if interval != 5*time.Second {
						interval = 5 * time.Second
						ticker.Reset(interval)
					}
				}
			}
		}
	}
}

func (s *Sprint) getBestBlock() (string, int, string, error) {
	// Ensure we have at least one RPC node
	if len(s.config.RPCNodes) == 0 {
		// Try localhost as fallback
		s.config.RPCNodes = []string{"http://127.0.0.1:8332"}
		log.Printf("No RPC nodes configured, using fallback: %s", s.config.RPCNodes[0])
	}

	// OPTIMIZATION: Parallel fan-out for first-response-wins
	if len(s.config.RPCNodes) > 1 {
		return s.getBestBlockParallel()
	}

	// Single node fallback
	return s.getBestBlockSingle()
}

// getBestBlockParallel implements parallel fan-out for multiple nodes
func (s *Sprint) getBestBlockParallel() (string, int, string, error) {
	type result struct {
		hash   string
		height int
		node   string
		err    error
	}

	results := make(chan result, len(s.config.RPCNodes))
	ctx, cancel := context.WithTimeout(s.ctx, 2*time.Second)
	defer cancel()

	// Fire all requests simultaneously
	activeNodes := 0
	for _, node := range s.config.RPCNodes {
		// Skip nodes in backoff
		if t, ok := s.nodeBackoff[node]; ok && time.Now().Before(t) {
			continue
		}

		activeNodes++
		go func(nodeURL string) {
			hash, height, err := s.tryRPCOptimized(nodeURL, s.getBlockchainInfoReq)
			select {
			case results <- result{hash, height, nodeURL, err}:
			case <-ctx.Done():
			}
		}(node)
	}

	if activeNodes == 0 {
		return "", 0, "", fmt.Errorf("no RPC nodes available (all in backoff)")
	}

	// Return first successful response
	for i := 0; i < activeNodes; i++ {
		select {
		case res := <-results:
			if res.err == nil {
				// Clear backoff on success
				delete(s.nodeBackoff, res.node)
				return res.hash, res.height, res.node, nil
			}
			// Set backoff on failure
			s.setNodeBackoff(res.node, res.err)
		case <-ctx.Done():
			return "", 0, "", fmt.Errorf("parallel RPC timeout")
		}
	}

	return "", 0, "", fmt.Errorf("all parallel RPC calls failed")
}

// getBestBlockSingle handles single node requests
func (s *Sprint) getBestBlockSingle() (string, int, string, error) {
	node := s.config.RPCNodes[0]

	// Skip if in backoff
	if t, ok := s.nodeBackoff[node]; ok && time.Now().Before(t) {
		return "", 0, "", fmt.Errorf("node %s in backoff until %v", node, t)
	}

	hash, height, err := s.tryRPCOptimized(node, s.getBlockchainInfoReq)
	if err != nil {
		s.setNodeBackoff(node, err)
		return "", 0, "", err
	}

	delete(s.nodeBackoff, node)
	return hash, height, node, nil
}

// setNodeBackoff implements smart backoff with jitter
func (s *Sprint) setNodeBackoff(node string, err error) {
	baseDelay := 3 * time.Second // OPTIMIZED: Faster recovery
	if prevTime, ok := s.nodeBackoff[node]; ok {
		sinceErr := time.Since(prevTime)
		if sinceErr < 20*time.Second { // OPTIMIZED: Faster max backoff
			baseDelay = time.Duration(math.Min(20, float64(baseDelay.Seconds()*1.5))) * time.Second
		}
	}
	jitter := time.Duration(rand.Intn(2000)) * time.Millisecond // OPTIMIZED: Less jitter
	s.nodeBackoff[node] = time.Now().Add(baseDelay + jitter)
}

// tryRPCOptimized is the hot path optimized version of tryRPC
func (s *Sprint) tryRPCOptimized(node string, reqBody []byte) (string, int, error) {
	// OPTIMIZATION: Ultra-short timeout for hot path
	ctx, cancel := context.WithTimeout(s.ctx, 1*time.Second)
	defer cancel()

	// Create request with pre-allocated body
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, node, bytes.NewReader(reqBody))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication and headers (optimized)
	if s.config.RPCUser != "" || s.config.RPCPass != "" {
		req.SetBasicAuth(s.config.RPCUser, s.config.RPCPass)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "keep-alive")

	// Execute request with timing
	start := time.Now()
	resp, err := s.client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return "", 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status (fast path)
	if resp.StatusCode != 200 {
		return "", 0, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	// OPTIMIZATION: Direct decode to target struct
	var envelope struct {
		Result struct {
			BestHash string `json:"bestblockhash"`
			Height   int    `json:"blocks"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return "", 0, fmt.Errorf("decode failed: %w", err)
	}

	// Check for RPC error
	if envelope.Error != nil {
		return "", 0, fmt.Errorf("rpc error: %d %s", envelope.Error.Code, envelope.Error.Message)
	}

	if envelope.Result.BestHash != "" {
		log.Printf("RPC call successful to %s in %dms", node, latency)
		return envelope.Result.BestHash, envelope.Result.Height, nil
	}

	return "", 0, fmt.Errorf("no valid block hash in response")
}

// burstProbe implements burst probing for immediate block detection
func (s *Sprint) burstProbe(currentHash string) (string, int, string, error) {
	log.Printf("🚀 BURST PROBE: 5 rapid calls in 50ms intervals")

	for i := 0; i < 5; i++ {
		hash, height, node, err := s.getBestBlock()
		if err == nil && hash != currentHash {
			log.Printf("🎯 BURST SUCCESS: New block %s detected on probe %d", hash[:8], i+1)
			return hash, height, node, nil
		}

		if i < 4 { // Don't sleep after last probe
			time.Sleep(50 * time.Millisecond)
		}
	}

	return "", 0, "", fmt.Errorf("burst probe failed to detect new block")
}

func (s *Sprint) prefetchBlock(hash, node string) {
	reqBody := []byte(fmt.Sprintf(`{"jsonrpc":"1.0","id":"sprint","method":"getblockheader","params":["%s"]}`, hash))
	if t, ok := s.nodeBackoff[node]; ok && time.Now().Before(t) {
		return
	}
	ctx, cancel := context.WithTimeout(s.ctx, 3*time.Second) // faster prefetch
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, node, bytes.NewReader(reqBody))
	req.SetBasicAuth(s.config.RPCUser, s.config.RPCPass)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	cancel()
	if err == nil && resp.StatusCode == 200 {
		resp.Body.Close()
		return
	}
	if err != nil {
		log.Printf("prefetch error %s: %v", node, err)
	} else if resp != nil {
		log.Printf("prefetch bad status %s: %d", node, resp.StatusCode)
	}
}

// ───────────────────────── On New Block ─────────────────────────

func (s *Sprint) OnNewBlock(hash string, height int, node string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.license.Tier == "free" && s.blocksSent >= s.license.BlockLimit {
		log.Printf("Free tier block limit reached (%d/day)", s.license.BlockLimit)
		return
	}

	start := time.Now()
	var sent int
	if s.config.TurboMode {
		sent = s.SprintBlockTurbo(hash)
	} else {
		sent = s.SprintBlock(hash)
	}
	lat := float64(time.Since(start).Milliseconds())

	s.blocksSent++
	m := Metrics{
		BlockHash:  hash,
		Height:     height,
		Latency:    lat,
		PeerCount:  sent,
		Timestamp:  time.Now().Unix(),
		LicenseKey: s.config.LicenseKey,
		RPCNode:    node,
	}

	select {
	case s.metrics <- m:
	default:
		log.Printf("metrics buffer full, dropping block %s", hash[:8])
	}

	log.Printf("Block %s (h=%d) sprinted in %.1fms to %d peers via %s",
		hash[:8], height, lat, sent, node)
}

// getBestBlockTurbo queries all nodes in parallel and returns the fastest valid response.
func (s *Sprint) getBestBlockTurbo() (string, int, string, error) {
	if len(s.config.RPCNodes) == 0 {
		return "", 0, "", fmt.Errorf("no RPC nodes configured")
	}

	ctx, cancel := context.WithTimeout(s.ctx, 2*time.Second) // shorter timeout
	defer cancel()

	type result struct {
		hash   string
		height int
		node   string
		err    error
	}
	results := make(chan result, len(s.config.RPCNodes))

	reqBody := []byte(`{"jsonrpc":"1.0","id":"sprint","method":"getblockchaininfo","params":[]}`)

	for _, node := range s.config.RPCNodes {
		go func(n string) {
			h, height, err := s.tryRPCOptimized(n, reqBody)
			select {
			case results <- result{hash: h, height: height, node: n, err: err}:
			case <-ctx.Done():
			}
		}(node)
	}

	// take first valid response, or timeout
	for range s.config.RPCNodes {
		select {
		case r := <-results:
			if r.err == nil && r.hash != "" {
				return r.hash, r.height, r.node, nil
			}
		case <-ctx.Done():
			return "", 0, "", fmt.Errorf("turbo poll timeout")
		}
	}
	return "", 0, "", fmt.Errorf("all turbo RPC calls failed")
}

// SprintBlockTurbo sends a pre-encoded notification to peers with tight deadlines and async logging.
func (s *Sprint) SprintBlockTurbo(hash string) int {
	s.mu.RLock()
	peers := make([]net.Conn, 0, len(s.peers))
	peerAddrs := make([]string, 0, len(s.peers))
	for addr, c := range s.peers {
		peers = append(peers, c)
		peerAddrs = append(peerAddrs, addr)
	}
	s.mu.RUnlock()
	if len(peers) == 0 {
		return 0
	}

	// Pre-encode payload once per sprint
	payload := struct {
		Type     string `json:"type"`
		Hash     string `json:"hash"`
		Ts       int64  `json:"ts"`
		Version  string `json:"version"`
		Protocol int    `json:"protocol"`
	}{
		Type: "block", Hash: hash, Ts: time.Now().UnixMilli(),
		Version: Version, Protocol: 1,
	}
	b, _ := json.Marshal(payload)
	message := append(b, '\n')

	var wg sync.WaitGroup
	var success int32

	for i, conn := range peers {
		addr := peerAddrs[i]
		wg.Add(1)
		go func(c net.Conn, a string) {
			defer wg.Done()
			_ = c.SetWriteDeadline(time.Now().Add(200 * time.Millisecond))
			if tcp, ok := c.(*net.TCPConn); ok {
				_ = tcp.SetNoDelay(true)
			}
			if _, err := c.Write(message); err == nil {
				atomic.AddInt32(&success, 1)
			} else {
				// drop slow/broken peer
				s.mu.Lock()
				if pc, exists := s.peers[a]; exists && pc == c {
					_ = pc.Close()
					delete(s.peers, a)
				}
				s.mu.Unlock()
			}
		}(conn, addr)
	}

	wg.Wait()
	succ := int(atomic.LoadInt32(&success))
	// Async non-blocking log
	go log.Printf("⚡ TURBO block %s sprinted to %d peers", hash[:8], succ)
	return succ
}

// ───────────────────────── Stubs / Helpers ─────────────────────────

// LoadConfig loads configuration from environment or a local file.
func (s *Sprint) LoadConfig() error {
	// defaults with sensible initial values
	cfg := Config{
		PollInterval: 5,
		Tier:         "free",
		RPCNodes:     []string{"http://127.0.0.1:8332"}, // Default to local node
	}

	// try read config.json from working dir
	configPaths := []string{"config.json", "/etc/bitcoin-sprint/config.json"}
	for _, path := range configPaths {
		if data, err := os.ReadFile(path); err == nil {
			var fileCfg Config
			if err := json.Unmarshal(data, &fileCfg); err == nil {
				// adopt values from file
				cfg = fileCfg

				// Ensure we don't have nil slices even when loading from file
				if cfg.RPCNodes == nil {
					cfg.RPCNodes = []string{"http://127.0.0.1:8332"}
				}

				log.Printf("Loaded configuration from %s", path)
				break
			} else {
				// if file exists but is invalid, return error
				return fmt.Errorf("invalid config file %s: %w", path, err)
			}
		}
	}

	// environment overrides (if present)
	if v := os.Getenv("SPRINT_LICENSE"); v != "" {
		cfg.LicenseKey = v
	}
	if v := os.Getenv("SPRINT_TIER"); v != "" {
		cfg.Tier = v
	}
	if v := os.Getenv("SPRINT_METRICS_URL"); v != "" {
		cfg.MetricsURL = v
	}
	if v := os.Getenv("SPRINT_RPC_NODE"); v != "" {
		// allow comma-separated list
		cfg.RPCNodes = nil
		for _, n := range strings.Split(v, ",") {
			n = strings.TrimSpace(n)
			if n != "" {
				// Ensure URL has a scheme
				if !strings.Contains(n, "://") {
					n = "http://" + n
				}
				cfg.RPCNodes = append(cfg.RPCNodes, n)
			}
		}
	}
	if v := os.Getenv("SPRINT_API_BASE"); v != "" {
		cfg.APIBase = v
	}
	if v := os.Getenv("SPRINT_RPC_USER"); v != "" {
		cfg.RPCUser = v
	}
	if v := os.Getenv("SPRINT_RPC_PASS"); v != "" {
		cfg.RPCPass = v
	}
	if v := os.Getenv("SPRINT_POLL_INTERVAL"); v != "" {
		if iv, err := strconv.Atoi(v); err == nil && iv > 0 {
			cfg.PollInterval = iv
		}
	}

	// Turbo mode env override
	if v := os.Getenv("SPRINT_TURBO"); v != "" {
		lv := strings.ToLower(strings.TrimSpace(v))
		if lv == "1" || lv == "true" || lv == "yes" || lv == "on" {
			cfg.TurboMode = true
		}
	}

	// Fallback to local node if no nodes were specified
	if len(cfg.RPCNodes) == 0 {
		cfg.RPCNodes = []string{"http://127.0.0.1:8332"}
		log.Printf("No RPC nodes specified, defaulting to %s", cfg.RPCNodes[0])
	}

	// Log configuration summary (excluding sensitive information)
	log.Printf("Configuration loaded: Tier=%s, Nodes=%d, PollInterval=%ds, Turbo=%v",
		cfg.Tier, len(cfg.RPCNodes), cfg.PollInterval, cfg.TurboMode)

	s.config = cfg
	return nil
}

// ValidateLicense obtains or validates license details from API or local config.
// Behaviour: if no license key and tier is free, create a free license locally.
// Otherwise, attempt to call APIBase to retrieve license details; on network
// failure, fall back to a conservative local license if possible.
func (s *Sprint) ValidateLicense() error {
	// Free tier with no key = automatic license
	if s.config.Tier == "free" && s.config.LicenseKey == "" {
		// Updated to 25 blocks/day for free tier
		s.license = License{Valid: true, Tier: "free", BlockLimit: 25, Peers: []string{}}
		log.Println("Using free tier license with 25 blocks/day limit")
		return nil
	}

	// If no API base provided or no license key, use local defaults
	if s.config.APIBase == "" || s.config.LicenseKey == "" {
		// Ensure tier is valid
		tier := s.config.Tier
		if tier == "" {
			tier = "standard"
		}

		// Set block limits based on tier
		blockLimit := 100 // default
		switch strings.ToLower(tier) {
		case "standard":
			blockLimit = 100
		case "premium":
			blockLimit = 1000
		case "enterprise":
			blockLimit = 999999 // effectively unlimited
		}

		s.license = License{Valid: true, Tier: tier, BlockLimit: blockLimit, Peers: []string{}}
		log.Printf("Using local license configuration: tier=%s, limit=%d blocks/day", tier, blockLimit)
		return nil
	}

	return s.validateRemoteLicense()
}

// validateRemoteLicense handles the remote license validation logic
func (s *Sprint) validateRemoteLicense() error {
	// Try calling remote license validation endpoint with security measures
	log.Printf("Validating license key with remote service: %s", s.config.APIBase)
	url := strings.TrimRight(s.config.APIBase, "/") + "/v1/license/validate"
	body := map[string]string{
		"license_key": s.config.LicenseKey,
		"version":     Version, // Include version for compatibility checking
		"client_id":   getSystemIdentifier(),
	}

	// Parse response with size limit for security
	var out struct {
		Valid bool     `json:"valid"`
		Tier  string   `json:"tier"`
		Limit int      `json:"block_limit"`
		Peers []string `json:"peers"`
	}

	// Prepare request
	buf, err := json.Marshal(body)
	if err != nil {
		log.Printf("Failed to marshal license request: %v", err)
		return s.fallbackLicense()
	}

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	// Create request with proper error handling
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		log.Printf("Failed to create license validation request: %v", err)
		return s.fallbackLicense()
	}

	// Set secure headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("BitcoinSprint/%s", Version))
	req.Header.Set("X-Client-Version", Version)

	// Execute request with timeout
	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("License validation network error: %v", err)
		return s.fallbackLicense()
	}
	defer resp.Body.Close()

	// Check for valid response
	if resp.StatusCode != 200 {
		log.Printf("License validation failed with status: %d", resp.StatusCode)
		return s.fallbackLicense()
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 10240)) // 10KB max
	if err != nil {
		log.Printf("Failed to read license response: %v", err)
		return s.fallbackLicense()
	}

	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		log.Printf("Failed to parse license response: %v", err)
		return s.fallbackLicense()
	}

	// Validate response contents
	if !out.Valid || out.Tier == "" || out.Limit <= 0 {
		log.Printf("Invalid license response content: valid=%v, tier=%s, limit=%d",
			out.Valid, out.Tier, out.Limit)
		return s.fallbackLicense()
	}

	// Success case
	s.license = License{
		Valid:      out.Valid,
		Tier:       out.Tier,
		BlockLimit: out.Limit,
		Peers:      out.Peers,
	}
	log.Printf("License validated successfully: tier=%s, limit=%d blocks/day",
		out.Tier, out.Limit)
	return nil
}

// fallbackLicense creates a conservative fallback license when validation fails
func (s *Sprint) fallbackLicense() error {
	// Conservative fallback to free tier
	s.license = License{Valid: false, Tier: "free", BlockLimit: 5, Peers: []string{}}
	return fmt.Errorf("license validation failed, using restricted free tier")
}

// getSystemIdentifier returns a unique but anonymized identifier for this system
func getSystemIdentifier() string {
	// Use hostname as a base but hash it for privacy
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}

	// Create simple hash
	sum := 0
	for _, c := range hostname {
		sum = (sum*31 + int(c)) % 1000000
	}

	// Format as anonymized ID
	return fmt.Sprintf("node-%06d", sum)
}

// StartWebDashboard launches a small HTTP dashboard showing recent metrics.
// It runs in its own goroutine and returns immediately.
func (s *Sprint) StartWebDashboard() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><head><title>Bitcoin Sprint</title></head><body><h1>Bitcoin Sprint</h1><p>Metrics endpoint: <a href="/metrics">/metrics</a></p></body></html>`))
	})

	// metrics endpoint returns last N metrics from the channel without blocking
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// drain available metrics non-blocking (up to 100)
		var batch []Metrics
		for i := 0; i < 100; i++ {
			select {
			case m := <-s.metrics:
				batch = append(batch, m)
			default:
				i = 100
			}
		}
		_ = json.NewEncoder(w).Encode(batch)
	})

	// choose port from env (SPRINT_DASH_PORT or PORT) or default to 8080
	port := os.Getenv("SPRINT_DASH_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	srv := &http.Server{Addr: port, Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("dashboard server: %v", err)
		}
	}()
}

// ConnectToPeers maintains TCP connections to peers listed in the license. It
// dials each peer in the background and keeps the connection in s.peers.
func (s *Sprint) ConnectToPeers() {
	go func() {
		backoff := map[string]time.Time{}
		for {
			select {
			case <-s.ctx.Done():
				// close existing connections
				for k, c := range s.peers {
					_ = c.Close()
					delete(s.peers, k)
				}
				return
			default:
			}

			peers := s.license.Peers
			if len(peers) == 0 {
				// nothing to do — sleep and check later
				time.Sleep(2 * time.Second)
				continue
			}

			for _, addr := range peers {
				if _, ok := s.peers[addr]; ok {
					continue // already connected
				}
				if t, ok := backoff[addr]; ok && time.Now().Before(t) {
					continue
				}
				// try dial
				conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
				if err != nil {
					// set retry backoff
					backoff[addr] = time.Now().Add(5 * time.Second)
					log.Printf("peer dial failed %s: %v", addr, err)
					continue
				}
				// store connection
				s.mu.Lock()
				s.peers[addr] = conn
				s.mu.Unlock()
				log.Printf("connected to peer %s", addr)
			}

			time.Sleep(1 * time.Second)
		}
	}()
}

// StartMetricsReporter batches metrics and posts them to the configured MetricsURL.
func (s *Sprint) StartMetricsReporter() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		var batch []Metrics
		for {
			select {
			case <-s.ctx.Done():
				return
			case m := <-s.metrics:
				batch = append(batch, m)
				if len(batch) >= 25 {
					s.postMetrics(batch)
					batch = nil
				}
			case <-ticker.C:
				if len(batch) > 0 {
					s.postMetrics(batch)
					batch = nil
				}
			}
		}
	}()
}

func (s *Sprint) postMetrics(batch []Metrics) {
	if s.config.MetricsURL == "" {
		return
	}
	buf, _ := json.Marshal(batch)
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.config.MetricsURL, bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("metrics post failed: %v", err)
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

// SprintBlock writes a small notification to each connected peer and returns
// the number of peers that accepted the message. Writes are done with a short
// deadline so this function won't block for long.
func (s *Sprint) SprintBlock(hash string) int {
	s.mu.RLock()
	peers := make([]net.Conn, 0, len(s.peers))
	peerAddrs := make([]string, 0, len(s.peers))
	for addr, c := range s.peers {
		peers = append(peers, c)
		peerAddrs = append(peerAddrs, addr)
	}
	s.mu.RUnlock()

	if len(peers) == 0 {
		return 0
	}

	var wg sync.WaitGroup
	type peerResult struct {
		addr    string
		success bool
		latency time.Duration
		err     error
	}

	results := make(chan peerResult, len(peers))
	start := time.Now()

	// OPTIMIZATION: Use pre-encoded template with hot path substitution
	message := s.preEncodedPayload
	// Replace placeholder hash in pre-encoded JSON
	hashBytes := []byte(hash)
	message = bytes.Replace(message, []byte("HASH_PLACEHOLDER"), hashBytes, 1)

	// Add timestamp for real-time update (minimal overhead)
	ts := time.Now().UnixNano() / int64(time.Millisecond)
	tsBytes := []byte(fmt.Sprintf("%d", ts))
	message = bytes.Replace(message, []byte("TS_PLACEHOLDER"), tsBytes, 1)

	// Send to all peers concurrently with ultra-fast timeouts
	for i, conn := range peers {
		addr := peerAddrs[i]
		wg.Add(1)
		go func(conn net.Conn, addr string) {
			defer wg.Done()

			// HOT PATH OPTIMIZATION: 200ms deadline for maximum speed
			peerStart := time.Now()
			conn.SetWriteDeadline(time.Now().Add(200 * time.Millisecond))

			// TCP_NODELAY for immediate transmission
			if tcpConn, ok := conn.(*net.TCPConn); ok {
				_ = tcpConn.SetNoDelay(true)
			}

			// Single write operation (optimized)
			n, err := conn.Write(message)
			latency := time.Since(peerStart)

			if err != nil || n != len(message) {
				results <- peerResult{
					addr:    addr,
					success: false,
					latency: latency,
					err:     err,
				}

				// Fast connection cleanup
				s.mu.Lock()
				if peerConn, exists := s.peers[addr]; exists && peerConn == conn {
					_ = peerConn.Close()
					delete(s.peers, addr)
				}
				s.mu.Unlock()
				return
			}

			results <- peerResult{
				addr:    addr,
				success: true,
				latency: latency,
				err:     nil,
			}
		}(conn, addr)
	}

	// HOT PATH: Faster timeout for maximum responsiveness
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All completed
	case <-time.After(300 * time.Millisecond): // Reduced from 750ms
		// Continue processing partial results
	}

	// Fast results collection
	close(results)
	success := 0
	totalLatency := int64(0)
	for result := range results {
		if result.success {
			success++
			totalLatency += result.latency.Milliseconds()
		}
	}

	// Optimized logging (reduce contention)
	if success > 0 {
		s.blocksSentMu.Lock()
		s.blocksSent++
		s.blocksSentMu.Unlock()

		if success > 0 {
			avgLatency := float64(totalLatency) / float64(success)
			log.Printf("⚡ SPRINT: Block %s → %d peers in %.1fms avg (%.1fms total)",
				hash[:8], success, avgLatency, float64(time.Since(start).Milliseconds()))
		}
	}

	return success
}

// startParallelMonitoring starts aggressive parallel monitoring across all RPC nodes
// to maximize block detection speed and achieve claimed performance advantages
func (s *Sprint) startParallelMonitoring(currentHash string, currentHeight int) {
	if len(s.config.RPCNodes) <= 1 {
		return // Need multiple nodes for parallel monitoring
	}

	log.Printf("Starting PARALLEL monitoring across %d nodes for ultra-fast detection", len(s.config.RPCNodes))

	// Monitor all nodes simultaneously for 30 seconds after block detection
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	results := make(chan struct {
		node    string
		hash    string
		height  int
		latency time.Duration
	}, len(s.config.RPCNodes))

	// ULTRA-AGGRESSIVE: Poll all nodes every 100ms simultaneously
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for _, node := range s.config.RPCNodes {
		wg.Add(1)
		go func(nodeURL string) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					start := time.Now()
					reqBody := []byte(`{"jsonrpc":"1.0","id":"parallel","method":"getblockchaininfo","params":[]}`)

					hash, height, err := s.tryRPCOptimized(nodeURL, reqBody)
					latency := time.Since(start)

					if err == nil && hash != currentHash && height > currentHeight {
						// NEW BLOCK DETECTED BY PARALLEL MONITORING!
						results <- struct {
							node    string
							hash    string
							height  int
							latency time.Duration
						}{nodeURL, hash, height, latency}

						log.Printf(" PARALLEL DETECTION: New block %s at height %d via %s in %v",
							hash[:8], height, nodeURL, latency)

						// Trigger immediate main detection
						s.OnNewBlock(hash, height, nodeURL)
						return
					}
				}
			}
		}(node)
	}

	// Wait for completion or timeout
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process any results
	for result := range results {
		log.Printf("Parallel node %s detected block %s in %v",
			result.node, result.hash[:8], result.latency)
	}
}

// startPredictiveMonitoring uses mempool analysis to predict when blocks are likely
// and increases polling frequency accordingly for maximum speed advantage
func (s *Sprint) startPredictiveMonitoring() {
	if len(s.config.RPCNodes) == 0 {
		return
	}

	log.Printf("Starting PREDICTIVE monitoring for mempool-based block prediction")

	// Monitor mempool every 2 seconds to predict blocks
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var lastMempoolSize int
	highActivityThreshold := 50 // transactions

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Get mempool info from primary node
			node := s.config.RPCNodes[0]
			reqBody := []byte(`{"jsonrpc":"1.0","id":"predictive","method":"getmempoolinfo","params":[]}`)

			ctx, cancel := context.WithTimeout(s.ctx, 2*time.Second)
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, node, bytes.NewReader(reqBody))
			if err != nil {
				cancel()
				continue
			}

			req.SetBasicAuth(s.config.RPCUser, s.config.RPCPass)
			req.Header.Set("Content-Type", "application/json")

			resp, err := s.client.Do(req)
			if err != nil {
				cancel()
				continue
			}

			var result struct {
				Result struct {
					Size int `json:"size"`
				} `json:"result"`
			}

			if json.NewDecoder(resp.Body).Decode(&result) == nil {
				mempoolSize := result.Result.Size
				resp.Body.Close()
				cancel()

				// Detect high mempool activity indicating likely block soon
				if mempoolSize > highActivityThreshold && mempoolSize > lastMempoolSize*2 {
					log.Printf(" PREDICTIVE: High mempool activity detected (%d txs), expecting block soon - BOOSTING polling", mempoolSize)

					// Start HYPER-AGGRESSIVE polling for 60 seconds
					go s.hyperAggressivePolling(60 * time.Second)
				}

				lastMempoolSize = mempoolSize
			} else {
				resp.Body.Close()
				cancel()
			}
		}
	}
}

// startPredictiveMonitoringTurbo is a faster mempool-based predictor used in Turbo mode.
func (s *Sprint) startPredictiveMonitoringTurbo() {
	if len(s.config.RPCNodes) == 0 {
		return
	}

	log.Printf("Starting PREDICTIVE monitoring (Turbo) with 500ms cadence")

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastSize int
	spike := 40 // lower threshold in turbo

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			node := s.config.RPCNodes[0]
			reqBody := []byte(`{"jsonrpc":"1.0","id":"predictive","method":"getmempoolinfo","params":[]}`)
			cctx, ccancel := context.WithTimeout(s.ctx, 1*time.Second)
			req, err := http.NewRequestWithContext(cctx, http.MethodPost, node, bytes.NewReader(reqBody))
			if err != nil {
				ccancel()
				continue
			}
			req.SetBasicAuth(s.config.RPCUser, s.config.RPCPass)
			req.Header.Set("Content-Type", "application/json")
			resp, err := s.client.Do(req)
			if err != nil {
				ccancel()
				continue
			}
			var out struct {
				Result struct {
					Size int `json:"size"`
				} `json:"result"`
			}
			if json.NewDecoder(resp.Body).Decode(&out) == nil {
				resp.Body.Close()
				ccancel()
				size := out.Result.Size
				if size > spike && (lastSize == 0 || size > lastSize*2) {
					log.Printf(" PREDICTIVE (Turbo): mempool spike %d → BOOST 100ms", size)
					go s.hyperAggressivePolling(60 * time.Second)
				}
				lastSize = size
			} else {
				resp.Body.Close()
				ccancel()
			}
		}
	}
}

// hyperAggressivePolling starts extremely fast polling (100ms) for a specified duration
// to catch blocks immediately when high activity is detected
func (s *Sprint) hyperAggressivePolling(duration time.Duration) {
	log.Printf(" HYPER-AGGRESSIVE POLLING: 100ms intervals for %v", duration)

	ctx, cancel := context.WithTimeout(s.ctx, duration)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var lastHash string

	for {
		select {
		case <-ctx.Done():
			log.Printf("HYPER-AGGRESSIVE polling completed")
			return
		case <-ticker.C:
			if len(s.config.RPCNodes) == 0 {
				continue
			}

			reqBody := []byte(`{"jsonrpc":"1.0","id":"hyper","method":"getblockchaininfo","params":[]}`)
			hash, height, err := s.tryRPCOptimized(s.config.RPCNodes[0], reqBody)

			if err == nil && hash != "" && hash != lastHash {
				log.Printf(" HYPER-AGGRESSIVE DETECTION: Block %s at height %d", hash[:8], height)
				s.OnNewBlock(hash, height, s.config.RPCNodes[0])
				lastHash = hash
				return // Block found, mission accomplished
			}
		}
	}
}
