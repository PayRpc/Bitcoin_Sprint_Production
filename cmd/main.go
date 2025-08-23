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
	"syscall"
	"time"
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

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// ───────────────────────── Main ─────────────────────────

func main() {
	// Modern Go 1.20+ uses automatic random seeding
	// No need for explicit rand.Seed() call

	// Print application banner
	log.Printf("Bitcoin Sprint v1.0.3 - Enterprise Bitcoin Block Detection")
	log.Printf("Copyright © 2025 BitcoinCab.inc")

	s := &Sprint{
		peers:   make(map[string]net.Conn),
		metrics: make(chan Metrics, 500),
		client: &http.Client{
			Timeout: 5 * time.Second, // faster overall timeout
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion:         tls.VersionTLS12,
					InsecureSkipVerify: false, // Enforce certificate validation
				},
				DialContext:           (&net.Dialer{Timeout: 2 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
				ForceAttemptHTTP2:     true,            // Enable HTTP/2 for better performance
				ResponseHeaderTimeout: 2 * time.Second, // faster response
				MaxIdleConns:          10,
				MaxIdleConnsPerHost:   5,
				IdleConnTimeout:       30 * time.Second,
			},
		},
		nodeBackoff: make(map[string]time.Time),
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())

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
		s.cancel()
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
			hash, height, node, err := s.getBestBlock()
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

				// Ultra-tight polling after new block (1s)
				interval = 1 * time.Second
				ticker.Reset(interval)
				log.Printf("New block detected - tightening poll interval to %v", interval)

				// Prefetch block details in background
				go s.prefetchBlock(hash, node)
			} else {
				// Adaptive polling based on time since last block
				elapsed := time.Since(lastBlockTime)

				switch {
				case elapsed < 30*time.Second:
					// Stay tight for 30s after block
					if interval != 1*time.Second {
						interval = 1 * time.Second
						ticker.Reset(interval)
					}
				case elapsed < 2*time.Minute:
					// Medium polling for normal periods
					if interval != 2*time.Second {
						interval = 2 * time.Second
						ticker.Reset(interval)
					}
				case elapsed < 10*time.Minute:
					// Wider polling during likely quiet periods
					if interval != 5*time.Second {
						interval = 5 * time.Second
						ticker.Reset(interval)
					}
				default:
					// Maximum relaxation during long quiet periods
					if interval != 10*time.Second {
						interval = 10 * time.Second
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

	reqBody := []byte(`{"jsonrpc":"1.0","id":"sprint","method":"getblockchaininfo","params":[]}`)

	// Track errors for better reporting
	errors := make([]error, 0, len(s.config.RPCNodes))

	for _, node := range s.config.RPCNodes {
		// Skip nodes in backoff period
		if t, ok := s.nodeBackoff[node]; ok && time.Now().Before(t) {
			continue
		}

		// Try RPC call
		hash, height, err := s.tryRPC(node, reqBody)
		if err == nil {
			// Clear backoff on success
			delete(s.nodeBackoff, node)
			return hash, height, node, nil
		}

		// Track error for detailed reporting
		errors = append(errors, fmt.Errorf("%s: %w", node, err))
		log.Printf("RPC node failed %s: %v", node, err)

		// Smart backoff with jitter
		baseDelay := 5 * time.Second
		if prevTime, ok := s.nodeBackoff[node]; ok {
			// Exponential backoff up to 30 seconds
			sinceErr := time.Since(prevTime)
			if sinceErr < 30*time.Second {
				baseDelay = time.Duration(math.Min(30, float64(baseDelay.Seconds()*2))) * time.Second
			}
		}
		jitter := time.Duration(rand.Intn(3000)) * time.Millisecond
		s.nodeBackoff[node] = time.Now().Add(baseDelay + jitter)
	}

	// Detailed error when all nodes fail
	if len(errors) > 0 {
		errorMsg := "All RPC nodes failed:\n"
		for i, err := range errors {
			errorMsg += fmt.Sprintf("  [%d] %v\n", i+1, err)
		}
		return "", 0, "", fmt.Errorf(errorMsg)
	}

	return "", 0, "", fmt.Errorf("no RPC nodes available")
}

func (s *Sprint) tryRPC(node string, body []byte) (string, int, error) {
	// Use a shorter timeout for faster detection
	ctx, cancel := context.WithTimeout(s.ctx, 3*time.Second)
	defer cancel()

	// Create request with proper error handling
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, node, bytes.NewReader(body))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication and headers
	if s.config.RPCUser != "" || s.config.RPCPass != "" {
		req.SetBasicAuth(s.config.RPCUser, s.config.RPCPass)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "keep-alive") // Optimize connection reuse

	// Execute request with timing
	start := time.Now()
	resp, err := s.client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return "", 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Read error body for better diagnostics
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", 0, fmt.Errorf("bad status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return "", 0, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for RPC error
	if envelope.Error != nil && envelope.Error.Message != "" {
		return "", 0, fmt.Errorf("rpc error: %d %s", envelope.Error.Code, envelope.Error.Message)
	}

	// Try to parse in primary format
	var result struct {
		BestHash string `json:"bestblockhash"`
		Height   int    `json:"blocks"`
	}

	if err := json.Unmarshal(envelope.Result, &result); err == nil {
		// Track successful RPC call latency for metrics
		if result.BestHash != "" {
			log.Printf("RPC call successful to %s in %dms", node, latency)
			return result.BestHash, result.Height, nil
		}
	}

	// Try alternate response format (some nodes wrap differently)
	var nested struct {
		Result struct {
			BestHash string `json:"bestblockhash"`
			Height   int    `json:"blocks"`
		} `json:"result"`
	}

	if err := json.Unmarshal(envelope.Result, &nested); err != nil {
		return "", 0, fmt.Errorf("failed to parse response in any known format")
	}

	if nested.Result.BestHash != "" {
		log.Printf("RPC call successful (alternate format) to %s in %dms", node, latency)
		return nested.Result.BestHash, nested.Result.Height, nil
	}

	return "", 0, fmt.Errorf("response contained no valid block hash")
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
	sent := s.SprintBlock(hash)
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

	// Fallback to local node if no nodes were specified
	if len(cfg.RPCNodes) == 0 {
		cfg.RPCNodes = []string{"http://127.0.0.1:8332"}
		log.Printf("No RPC nodes specified, defaulting to %s", cfg.RPCNodes[0])
	}

	// Log configuration summary (excluding sensitive information)
	log.Printf("Configuration loaded: Tier=%s, Nodes=%d, PollInterval=%ds",
		cfg.Tier, len(cfg.RPCNodes), cfg.PollInterval)

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
		"version":     "1.0.3", // Include version for compatibility checking
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
	req.Header.Set("User-Agent", "BitcoinSprint/1.0.3")
	req.Header.Set("X-Client-Version", "1.0.3")

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
		// No connected peers
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

	// Prepare payload with metadata to optimize validation
	payload := struct {
		Type     string `json:"type"`
		Hash     string `json:"hash"`
		Ts       int64  `json:"ts"`
		Version  string `json:"version"`
		Protocol int    `json:"protocol"`
	}{
		Type:     "block",
		Hash:     hash,
		Ts:       time.Now().UnixNano() / int64(time.Millisecond), // millisecond precision
		Version:  "1.0.3",
		Protocol: 1, // Protocol version for future compatibility
	}

	b, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal block notification: %v", err)
		return 0
	}

	// Add newline for message framing
	message := append(b, '\n')

	// Send to all peers concurrently with individual timeouts
	for i, conn := range peers {
		addr := peerAddrs[i]
		wg.Add(1)
		go func(conn net.Conn, addr string) {
			defer wg.Done()

			// Ultra-fast deadline for optimal performance
			peerStart := time.Now()
			conn.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))

			// TCP_NODELAY for immediate transmission without buffering
			if tcpConn, ok := conn.(*net.TCPConn); ok {
				_ = tcpConn.SetNoDelay(true)
			}

			// Write the message
			n, err := conn.Write(message)
			latency := time.Since(peerStart)

			if err != nil || n != len(message) {
				// Record failure
				results <- peerResult{
					addr:    addr,
					success: false,
					latency: latency,
					err:     err,
				}

				// Drop connection on write error
				s.mu.Lock()
				if peerConn, exists := s.peers[addr]; exists && peerConn == conn {
					log.Printf("Dropping peer connection to %s: %v", addr, err)
					_ = peerConn.Close()
					delete(s.peers, addr)
				}
				s.mu.Unlock()
				return
			}

			// Record success
			results <- peerResult{
				addr:    addr,
				success: true,
				latency: latency,
				err:     nil,
			}
		}(conn, addr)
	}

	// Wait for all goroutines with a timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Use timeout to ensure we don't wait too long
	select {
	case <-done:
		// All done normally
	case <-time.After(750 * time.Millisecond):
		log.Printf("Some peer notifications timed out")
	}

	// Collect results
	close(results)
	success := 0
	totalLatency := int64(0)
	for result := range results {
		if result.success {
			success++
			totalLatency += result.latency.Milliseconds()
		}
	}

	// Calculate average latency if any successful
	if success > 0 {
		avgLatency := float64(totalLatency) / float64(success)
		log.Printf("Block %s sprinted to %d peers in avg %.1fms (total %.1fms)",
			hash[:8], success, avgLatency, float64(time.Since(start).Milliseconds()))
	}

	return success
}
