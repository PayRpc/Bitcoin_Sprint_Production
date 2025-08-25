// SPDX-License-Identifier: MIT
// Bitcoin Sprint - RPC Edition (Enterprise-Ready)
// Copyright (c) 2025 BitcoinCab.inc

package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/PayRpc/Bitcoin-Sprint/pkg/secure"
)

// Versioning - populated at build time via -ldflags
var (
	Version = "dev"
	Commit  = "unknown"
)

// ───────────────────────── Types ─────────────────────────

type Config struct {
	LicenseKey     *secure.SecureBuffer `json:"-"` // Secure: license key
	Tier           string               `json:"tier"`
	MetricsURL     string               `json:"metrics_url"`
	RPCNodes       []string             `json:"rpc_nodes"`
	APIBase        string               `json:"api_base"`
	RPCUser        string               `json:"rpc_user"`
	RPCPass        *secure.SecureBuffer `json:"-"`                // Secure: RPC password
	PollInterval   int                  `json:"poll_interval"`    // seconds, default 5
	TurboMode      bool                 `json:"turbo_mode"`       // enable ultra-aggressive fan-out
	MaxPeers       int                  `json:"max_peers"`        // maximum peer connections
	LogLevel       string               `json:"log_level"`        // debug, info, warn, error
	RateLimits     map[string]int       `json:"rate_limits"`      // e.g., {"/latest": 5}
	PeerSecret     *secure.SecureBuffer `json:"-"`                // Secure: shared secret for peer auth
	PeerListenPort int                  `json:"peer_listen_port"` // port for peer mesh networking, default 8335

	// Plain text versions for JSON marshaling/unmarshaling
	LicenseKeyPlain string `json:"license_key"`
	RPCPassPlain    string `json:"rpc_pass"`
	PeerSecretPlain string `json:"peer_secret"`
}

// SecureConfig provides secure memory handling for sensitive configuration data
func (c *Config) InitializeSecureFields() error {
	// Initialize SecureBuffers for sensitive data
	if c.LicenseKeyPlain != "" {
		c.LicenseKey = secure.NewSecureBuffer(len(c.LicenseKeyPlain))
		if c.LicenseKey == nil {
			return fmt.Errorf("failed to create secure buffer for license key")
		}
		if !c.LicenseKey.Copy([]byte(c.LicenseKeyPlain)) {
			return fmt.Errorf("failed to copy license key to secure buffer")
		}
		// Clear plain text version
		c.LicenseKeyPlain = ""
	}

	if c.RPCPassPlain != "" {
		c.RPCPass = secure.NewSecureBuffer(len(c.RPCPassPlain))
		if c.RPCPass == nil {
			return fmt.Errorf("failed to create secure buffer for RPC password")
		}
		if !c.RPCPass.Copy([]byte(c.RPCPassPlain)) {
			return fmt.Errorf("failed to copy RPC password to secure buffer")
		}
		// Clear plain text version
		c.RPCPassPlain = ""
	}

	if c.PeerSecretPlain != "" {
		c.PeerSecret = secure.NewSecureBuffer(len(c.PeerSecretPlain))
		if c.PeerSecret == nil {
			return fmt.Errorf("failed to create secure buffer for peer secret")
		}
		if !c.PeerSecret.Copy([]byte(c.PeerSecretPlain)) {
			return fmt.Errorf("failed to copy peer secret to secure buffer")
		}
		// Clear plain text version
		c.PeerSecretPlain = ""
	}

	return nil
}

// GetLicenseKey returns the license key from secure memory
func (c *Config) GetLicenseKey() string {
	if c.LicenseKey == nil {
		return ""
	}
	data := c.LicenseKey.Data()
	if data == nil {
		return ""
	}
	return string(data)
}

// GetRPCPass returns the RPC password from secure memory
func (c *Config) GetRPCPass() string {
	if c.RPCPass == nil {
		return ""
	}
	data := c.RPCPass.Data()
	if data == nil {
		return ""
	}
	return string(data)
}

// GetPeerSecret returns the peer secret from secure memory
func (c *Config) GetPeerSecret() string {
	if c.PeerSecret == nil {
		return ""
	}
	data := c.PeerSecret.Data()
	if data == nil {
		return ""
	}
	return string(data)
}

// Cleanup securely frees all sensitive data
func (c *Config) Cleanup() {
	if c.LicenseKey != nil {
		c.LicenseKey.Free()
		c.LicenseKey = nil
	}
	if c.RPCPass != nil {
		c.RPCPass.Free()
		c.RPCPass = nil
	}
	if c.PeerSecret != nil {
		c.PeerSecret.Free()
		c.PeerSecret = nil
	}
}

// MarshalJSON implements custom JSON marshaling for Config
func (c *Config) MarshalJSON() ([]byte, error) {
	// Prepare plain text versions for JSON serialization using WithBytes
	if c.LicenseKey != nil {
		_ = c.LicenseKey.WithBytes(func(b []byte) error {
			c.LicenseKeyPlain = string(b)
			return nil
		})
	}
	if c.RPCPass != nil {
		_ = c.RPCPass.WithBytes(func(b []byte) error {
			c.RPCPassPlain = string(b)
			return nil
		})
	}
	if c.PeerSecret != nil {
		_ = c.PeerSecret.WithBytes(func(b []byte) error {
			c.PeerSecretPlain = string(b)
			return nil
		})
	}

	// Create a copy with plain text fields for marshaling
	type ConfigAlias Config
	alias := (*ConfigAlias)(c)

	data, err := json.Marshal(alias)

	// Immediately clear plain text versions after marshaling
	c.LicenseKeyPlain = ""
	c.RPCPassPlain = ""
	c.PeerSecretPlain = ""

	return data, err
}

// UnmarshalJSON implements custom JSON unmarshaling for Config
func (c *Config) UnmarshalJSON(data []byte) error {
	// Use alias to avoid recursion
	type ConfigAlias Config
	alias := (*ConfigAlias)(c)

	if err := json.Unmarshal(data, alias); err != nil {
		return err
	}

	// Initialize secure fields from plain text versions
	return c.InitializeSecureFields()
}

type License struct {
	Valid      bool     `json:"valid"`
	LicenseKey string   `json:"license_key"`
	Tier       string   `json:"tier"`
	BlockLimit int      `json:"block_limit"`
	Peers      []string `json:"peers"`
	ExpiresAt  int64    `json:"expires_at"`
	DailyReset int64    `json:"daily_reset"` // Unix timestamp of last reset
}

type Metrics struct {
	BlockHash  string  `json:"block_hash"`
	Latency    float64 `json:"latency_ms"`
	PeerCount  int     `json:"peer_count"`
	Timestamp  int64   `json:"timestamp"`
	LicenseKey string  `json:"license_key"`
	Height     int     `json:"height"`
	RPCNode    string  `json:"rpc_node"`
	Success    bool    `json:"success"`
}

// API Response Types for Enhanced Endpoints
type StatusResponse struct {
	Tier             string    `json:"tier"`
	LicenseKey       string    `json:"license_key"`
	Valid            bool      `json:"valid"`
	BlocksSentToday  int64     `json:"blocks_sent_today"`
	BlockLimit       int       `json:"block_limit"`
	PeersConnected   int       `json:"peers_connected"`
	UptimeSeconds    int64     `json:"uptime_seconds"`
	Version          string    `json:"version"`
	TurboModeEnabled bool      `json:"turbo_mode_enabled"`
	LastBlockTime    time.Time `json:"last_block_time"`
	RPCNodesActive   int       `json:"rpc_nodes_active"`
}

type PredictiveResponse struct {
	MempoolSize             int     `json:"mempool_size"`
	Trend                   string  `json:"trend"`
	ProbabilityNextBlock60s float64 `json:"probability_next_block_60s"`
	LastUpdate              int64   `json:"last_update"`
	AverageBlockTime        float64 `json:"average_block_time_minutes"`
}

// Peer handshake message
type PeerHandshake struct {
	LicenseKey string `json:"license_key"`
	Timestamp  int64  `json:"timestamp"`
	Signature  string `json:"signature"` // HMAC-SHA256 of LicenseKey+Timestamp
}

// Block message for gossip relay
type BlockMessage struct {
	Type      string `json:"type"`
	Hash      string `json:"hash"`
	Ts        string `json:"ts"`
	Version   string `json:"version"`
	Protocol  int    `json:"protocol"`
	MessageID string `json:"message_id"` // Unique ID to prevent relay loops
}

// Enhanced rate limiter with token bucket
type RateLimiter struct {
	mu             sync.RWMutex
	limiters       map[string]*rate.Limiter
	standardLimits map[string]int
	turboLimits    map[string]int
	cleanupTicker  *time.Ticker
}

// Performance monitoring
type PerformanceMetrics struct {
	mu                sync.RWMutex
	avgBlockDetection time.Duration
	avgSprintTime     time.Duration
	totalBlocks       int64
	failedConnections int64
	startTime         time.Time
	nodeLatencies     map[string]time.Duration
}

type Sprint struct {
	config         Config
	license        License
	peers          map[string]*PeerConnection
	metrics        chan Metrics
	blocksSent     int64
	client         *http.Client
	nodeBackoff    map[string]*BackoffState
	rateLimiter    *RateLimiter
	startTime      time.Time
	lastMempool    int
	latestMetric   *Metrics
	perfMetrics    *PerformanceMetrics
	circuitBreaker *CircuitBreaker
	seenMessages   map[string]time.Time // Tracks relayed message IDs
	seenMessagesMu sync.RWMutex

	// Hot path optimizations
	getBlockchainInfoReq []byte
	getMempoolInfoReq    []byte
	preEncodedPayload    []byte

	// Current state
	currentBlockHash   string
	currentBlockHeight int
	lastBlockTime      time.Time

	// Synchronization
	mu       sync.RWMutex
	peersMu  sync.RWMutex
	ctx      context.Context
	cancelFn context.CancelFunc
}

type PeerConnection struct {
	conn      net.Conn
	addr      string
	connected time.Time
	lastSent  time.Time
	failures  int64
	successes int64
	authed    bool
}

type BackoffState struct {
	until     time.Time
	attempts  int
	lastError error
}

type CircuitBreaker struct {
	mu          sync.RWMutex
	failures    int
	maxFailures int
	breakUntil  time.Time
}

// ───────────────────────── Rate Limiter Implementation ─────────────────────────

func NewRateLimiter(config Config) *RateLimiter {
	rl := &RateLimiter{
		limiters:       make(map[string]*rate.Limiter),
		standardLimits: config.RateLimits,
		turboLimits:    make(map[string]int),
	}

	// Apply turbo mode multiplier (5x standard limits)
	for endpoint, limit := range rl.standardLimits {
		rl.turboLimits[endpoint] = limit * 5
	}

	// Fallback defaults if not configured
	defaultStandards := map[string]int{
		"/latest":     4,
		"/metrics":    2,
		"/status":     1,
		"/predictive": 2,
		"/stream":     1,
	}
	for endpoint, limit := range defaultStandards {
		if _, exists := rl.standardLimits[endpoint]; !exists {
			rl.standardLimits[endpoint] = limit
			rl.turboLimits[endpoint] = limit * 5
		}
	}

	rl.cleanupTicker = time.NewTicker(30 * time.Second)
	go rl.cleanupWorker()

	return rl
}

func (s *Sprint) cleanupSeenMessages() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.seenMessagesMu.Lock()
			for id, ts := range s.seenMessages {
				if time.Since(ts) > 1*time.Hour {
					delete(s.seenMessages, id)
				}
			}
			s.seenMessagesMu.Unlock()
		}
	}
}

func (rl *RateLimiter) cleanupWorker() {
	for range rl.cleanupTicker.C {
		rl.mu.Lock()
		for key, limiter := range rl.limiters {
			if limiter.Tokens() == float64(limiter.Limit()) {
				delete(rl.limiters, key)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Allow(clientIP, endpoint string, isTurbo bool) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	key := clientIP + ":" + endpoint
	limiter, exists := rl.limiters[key]
	if !exists {
		limitVal := 1.0
		if isTurbo {
			if l, ok := rl.turboLimits[endpoint]; ok {
				limitVal = float64(l)
			} else {
				limitVal = 5.0
			}
		} else {
			if l, ok := rl.standardLimits[endpoint]; ok {
				limitVal = float64(l)
			}
		}
		// Burst up to 2x limit
		limiter = rate.NewLimiter(rate.Limit(limitVal), int(limitVal*2))
		rl.limiters[key] = limiter
	}

	return limiter.Allow()
}

func (rl *RateLimiter) Stop() {
	if rl.cleanupTicker != nil {
		rl.cleanupTicker.Stop()
	}
}

// ───────────────────────── Main ─────────────────────────

func main() {
	// Version check
	if len(os.Args) > 1 {
		if os.Args[1] == "--version" || os.Args[1] == "-v" {
			fmt.Printf("Bitcoin Sprint v%s (commit %s)\n", Version, Commit)
			return
		}
	}

	// Print application banner
	log.Printf("Bitcoin Sprint v%s - Enterprise Bitcoin Block Detection", Version)
	log.Printf("Copyright © 2025 BitcoinCab.inc")

	// Create main sprint instance
	s, err := NewSprint()
	if err != nil {
		log.Fatal("Failed to create Sprint instance:", err)
	}

	// Load configuration and validate license
	if err := s.LoadConfig(); err != nil {
		log.Fatal("Configuration error:", err)
	}

	if err := s.ValidateLicense(); err != nil {
		log.Fatal("License error:", err)
	}

	// Initialize logger
	logger, err := initLogger(s.config.LogLevel)
	if err != nil {
		log.Fatal("Logger initialization error:", err)
	}
	zap.ReplaceGlobals(logger)

	// Start all services
	if err := s.Start(); err != nil {
		log.Fatal("Failed to start Sprint:", err)
	}

	// Setup graceful shutdown
	s.WaitForShutdown()
}

func initLogger(logLevel string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	switch strings.ToLower(logLevel) {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		return nil, fmt.Errorf("invalid log_level: %s", logLevel)
	}
	return cfg.Build()
}

func NewSprint() (*Sprint, error) {
	ctx, cancel := context.WithCancel(context.Background())

	s := &Sprint{
		peers:        make(map[string]*PeerConnection),
		metrics:      make(chan Metrics, 5000), // Large buffer
		nodeBackoff:  make(map[string]*BackoffState),
		seenMessages: make(map[string]time.Time), // Initialize seen messages
		startTime:    time.Now(),
		perfMetrics: &PerformanceMetrics{
			startTime:     time.Now(),
			nodeLatencies: make(map[string]time.Duration),
		},
		circuitBreaker: &CircuitBreaker{maxFailures: 10},
		ctx:            ctx,
		cancelFn:       cancel,
	}

	// Create optimized HTTP client
	s.client = &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: false,
			},
			DialContext: (&net.Dialer{
				Timeout:   2 * time.Second,
				KeepAlive: 60 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     false,
			ResponseHeaderTimeout: 2 * time.Second,
			MaxIdleConns:          200,
			MaxIdleConnsPerHost:   50,
			MaxConnsPerHost:       100,
			IdleConnTimeout:       90 * time.Second,
		},
	}

	// Pre-marshal common requests
	s.getBlockchainInfoReq = []byte(`{"jsonrpc":"1.0","id":"sprint","method":"getblockchaininfo","params":[]}`)
	s.getMempoolInfoReq = []byte(`{"jsonrpc":"1.0","id":"sprint","method":"getmempoolinfo","params":[]}`)

	// Pre-encode sprint payload template
	payload := BlockMessage{
		Type:      "block",
		Hash:      "HASH_PLACEHOLDER",
		Ts:        "TS_PLACEHOLDER",
		Version:   Version,
		Protocol:  2, // Updated protocol version
		MessageID: "MESSAGE_ID_PLACEHOLDER",
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload template: %w", err)
	}
	s.preEncodedPayload = append(payloadBytes, '\n')

	return s, nil
}

func (s *Sprint) Start() error {
	zap.L().Info("Starting Bitcoin Sprint services...")

	// Initialize rate limiter
	s.rateLimiter = NewRateLimiter(s.config)

	// Start core services
	go s.StartWebDashboard()
	go s.StartBlockPoller()
	go s.ConnectToPeers()
	go s.StartMetricsReporter()
	go s.StartPerformanceMonitor()
	go s.StartLicenseMonitor()
	go s.StartPeerListener(":" + strconv.Itoa(s.config.PeerListenPort)) // Start inbound listener for peer mesh networking
	go s.cleanupSeenMessages()                                          // Start cleanup for seen messages

	// Start advanced monitoring if turbo mode enabled
	if s.config.TurboMode {
		go s.StartPredictiveMonitoring()
		zap.L().Info("Bitcoin Sprint running in TURBO mode")
	} else {
		zap.L().Info("Bitcoin Sprint running in STANDARD mode")
	}

	// Log configuration
	dashPort := s.getDashboardPort()
	zap.L().Info("Dashboard started", zap.String("url", "http://localhost"+dashPort))
	zap.L().Info("Configuration",
		zap.String("tier", s.config.Tier),
		zap.Int("nodes", len(s.config.RPCNodes)),
		zap.Int("poll_interval", s.config.PollInterval))

	return nil
}

func (s *Sprint) WaitForShutdown() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigCh
	zap.L().Info("Shutting down Bitcoin Sprint...")

	// Graceful shutdown
	s.Shutdown()
	zap.L().Info("Bitcoin Sprint shutdown complete")
}

func (s *Sprint) Shutdown() {
	// Cancel context
	s.cancelFn()

	// Close peer connections
	s.peersMu.Lock()
	for _, peer := range s.peers {
		if peer.conn != nil {
			peer.conn.Close()
		}
	}
	s.peersMu.Unlock()

	// Stop rate limiter
	s.rateLimiter.Stop()

	// Close metrics channel
	close(s.metrics)

	// Cleanup sensitive configuration data
	s.config.Cleanup()
}

// ───────────────────────── Config and License ─────────────────────────

func (s *Sprint) LoadConfig() error {
	// Default configuration
	s.config = Config{
		PollInterval:   5,
		Tier:           "free",
		LogLevel:       "info",
		MaxPeers:       100,
		PeerListenPort: 8335,
		MetricsURL:     "https://api.bitcoincab.inc/metrics",
		APIBase:        "http://localhost:8080",
		RateLimits:     make(map[string]int),
	}

	// Override with env vars for sensitive fields - use plain text versions temporarily
	if user := os.Getenv("RPC_USER"); user != "" {
		s.config.RPCUser = user
	}
	if pass := os.Getenv("RPC_PASS"); pass != "" {
		s.config.RPCPassPlain = pass
	}
	if secret := os.Getenv("PEER_SECRET"); secret != "" {
		s.config.PeerSecretPlain = secret
	}

	// Read config file
	data, err := os.ReadFile("config.json")
	if err != nil {
		if os.IsNotExist(err) {
			zap.L().Warn("No config.json found, using defaults")
			// Still need to initialize secure fields from env vars
			if err := s.config.InitializeSecureFields(); err != nil {
				return fmt.Errorf("failed to initialize secure fields: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to read config.json: %w", err)
	}

	// Unmarshal JSON into config (this will call InitializeSecureFields via UnmarshalJSON)
	if err := json.Unmarshal(data, &s.config); err != nil {
		return fmt.Errorf("failed to parse config.json: %w", err)
	}

	// Override JSON config with env vars if present
	if pass := os.Getenv("RPC_PASS"); pass != "" {
		if s.config.RPCPass != nil {
			s.config.RPCPass.Free()
		}
		s.config.RPCPassPlain = pass
		if err := s.config.InitializeSecureFields(); err != nil {
			return fmt.Errorf("failed to reinitialize RPC password from env: %w", err)
		}
	}
	if secret := os.Getenv("PEER_SECRET"); secret != "" {
		if s.config.PeerSecret != nil {
			s.config.PeerSecret.Free()
		}
		s.config.PeerSecretPlain = secret
		if err := s.config.InitializeSecureFields(); err != nil {
			return fmt.Errorf("failed to reinitialize peer secret from env: %w", err)
		}
	}

	// Set default peer listen port if not specified
	if s.config.PeerListenPort == 0 {
		s.config.PeerListenPort = 8335
	}

	// Validate required fields using secure buffers without creating strings
	if s.config.LicenseKey == nil || s.config.LicenseKey.Data() == nil || len(s.config.LicenseKey.Data()) == 0 {
		return fmt.Errorf("license_key is required in config")
	}
	if len(s.config.RPCNodes) == 0 {
		return fmt.Errorf("at least one RPC node must be specified")
	}
	for _, node := range s.config.RPCNodes {
		if !strings.HasPrefix(node, "https://") && !strings.HasPrefix(node, "http://localhost") {
			return fmt.Errorf("RPC node %s must use HTTPS or be localhost", node)
		}
	}
	if s.config.PeerSecret == nil || s.config.PeerSecret.Data() == nil || len(s.config.PeerSecret.Data()) == 0 {
		return fmt.Errorf("peer_secret is required for secure peer connections")
	}

	zap.L().Info("Loaded configuration",
		zap.String("tier", s.config.Tier),
		zap.Int("rpc_nodes", len(s.config.RPCNodes)),
		zap.Bool("turbo_mode", s.config.TurboMode))

	return nil
}

func (s *Sprint) ValidateLicense() error {
	// Try local license file first
	data, err := os.ReadFile("license.json")
	if err == nil {
		if err := json.Unmarshal(data, &s.license); err != nil {
			return fmt.Errorf("failed to parse license.json: %w", err)
		}
		if s.license.Valid && s.config.LicenseKey == s.license.LicenseKey && s.license.ExpiresAt > time.Now().Unix() {
			if s.isLicenseResetNeeded() {
				if err := s.resetLicenseBlocks(); err != nil {
					return fmt.Errorf("failed to reset license: %w", err)
				}
			}
			zap.L().Info("Local license validated",
				zap.String("tier", s.license.Tier),
				zap.String("expires", time.Unix(s.license.ExpiresAt, 0).String()))
			return nil
		}
	}

	// Fallback to remote license check
	return s.validateLicenseRemote()
}

func (s *Sprint) validateLicenseRemote() error {
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	var req *http.Request
	var err error
	if s.config.LicenseKey != nil {
		err = s.config.LicenseKey.WithBytes(func(b []byte) error {
			// Build JSON payload transiently
			payload := map[string]string{"license_key": string(b)}
			rb, merr := json.Marshal(payload)
			if merr != nil {
				return merr
			}
			req, err = http.NewRequestWithContext(ctx, http.MethodPost, s.config.MetricsURL+"/license/validate", bytes.NewReader(rb))
			// zero temporary rb
			for i := range rb {
				rb[i] = 0
			}
			return err
		})
		if err != nil {
			return fmt.Errorf("failed to build license request: %w", err)
		}
	} else {
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, s.config.MetricsURL+"/license/validate", bytes.NewReader([]byte("{}")))
	}
	if err != nil {
		return fmt.Errorf("failed to create license validation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("license validation failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("license validation HTTP %d: %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(&s.license); err != nil {
		return fmt.Errorf("failed to decode license response: %w", err)
	}

	if !s.license.Valid || s.license.ExpiresAt <= time.Now().Unix() {
		return fmt.Errorf("invalid or expired license: valid=%v, expires=%d", s.license.Valid, s.license.ExpiresAt)
	}

	// Initialize daily reset
	s.license.DailyReset = time.Now().Unix()
	if err := s.saveLicense(); err != nil {
		zap.L().Warn("Failed to save license", zap.Error(err))
	}

	zap.L().Info("Remote license validated",
		zap.String("tier", s.license.Tier),
		zap.Int("block_limit", s.license.BlockLimit),
		zap.String("expires", time.Unix(s.license.ExpiresAt, 0).String()))
	return nil
}

func (s *Sprint) StartLicenseMonitor() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if s.isLicenseResetNeeded() {
				if err := s.resetLicenseBlocks(); err != nil {
					zap.L().Error("Failed to reset license blocks", zap.Error(err))
				}
			}
			if time.Now().Unix() > s.license.ExpiresAt-24*3600 {
				if err := s.validateLicenseRemote(); err != nil {
					zap.L().Error("Failed to revalidate license", zap.Error(err))
				}
			}
		}
	}
}

func (s *Sprint) isLicenseResetNeeded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Now().Unix() > s.license.DailyReset+24*3600
}

func (s *Sprint) resetLicenseBlocks() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	var req *http.Request
	var err error
	if s.config.LicenseKey != nil {
		if err := s.config.LicenseKey.WithBytes(func(b []byte) error {
			payload := map[string]string{"license_key": string(b), "tier": s.license.Tier}
			rb, merr := json.Marshal(payload)
			if merr != nil {
				return merr
			}
			req, err = http.NewRequestWithContext(ctx, http.MethodPost, s.config.MetricsURL+"/license/reset", bytes.NewReader(rb))
			for i := range rb {
				rb[i] = 0
			}
			return err
		}); err != nil {
			return fmt.Errorf("failed to build license reset request: %w", err)
		}
	} else {
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, s.config.MetricsURL+"/license/reset", bytes.NewReader([]byte("{}")))
	}
	if err != nil {
		return fmt.Errorf("failed to create license reset request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("license reset failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("license reset HTTP %d: %s", resp.StatusCode, string(body))
	}

	s.license.DailyReset = time.Now().Unix()
	atomic.StoreInt64(&s.blocksSent, 0)
	if err := s.saveLicense(); err != nil {
		zap.L().Warn("Failed to save license after reset", zap.Error(err))
	}

	zap.L().Info("License block limit reset", zap.String("tier", s.license.Tier))
	return nil
}

func (s *Sprint) saveLicense() error {
	data, err := json.Marshal(s.license)
	if err != nil {
		return fmt.Errorf("failed to marshal license: %w", err)
	}
	return os.WriteFile("license.json", data, 0600)
}

// ───────────────────────── Block Polling ─────────────────────────

func (s *Sprint) StartBlockPoller() {
	baseInterval := time.Duration(s.config.PollInterval) * time.Second
	ticker := time.NewTicker(baseInterval)
	defer ticker.Stop()

	zap.L().Info("Block poller started", zap.Duration("interval", baseInterval))

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.pollForNewBlocks()

			// Adaptive interval in turbo mode
			if s.config.TurboMode {
				interval := baseInterval
				if s.lastMempool > 1000 {
					interval = baseInterval / 2
				} else if s.lastMempool < 100 {
					interval = baseInterval * 2
				}
				ticker.Reset(interval)
			}
		}
	}
}

func (s *Sprint) pollForNewBlocks() {
	hash, height, node, err := s.getBestBlock()
	if err != nil {
		if s.shouldLog("debug") {
			zap.L().Debug("Block poll failed", zap.Error(err))
		}
		return
	}

	// Check if this is a new block
	s.mu.Lock()
	isNewBlock := hash != s.currentBlockHash
	if isNewBlock {
		s.currentBlockHash = hash
		s.currentBlockHeight = height
		s.lastBlockTime = time.Now()
	}
	s.mu.Unlock()

	if isNewBlock {
		zap.L().Info("New block detected",
			zap.String("hash_prefix", hash[:8]),
			zap.Int("height", height),
			zap.String("node", node))
		s.OnNewBlock(hash, height, node, "") // Empty messageID for locally detected blocks
	}
}

func (s *Sprint) getBestBlock() (string, int, string, error) {
	s.circuitBreaker.mu.RLock()
	if time.Now().Before(s.circuitBreaker.breakUntil) {
		s.circuitBreaker.mu.RUnlock()
		return "", 0, "", fmt.Errorf("circuit breaker open until %v", s.circuitBreaker.breakUntil)
	}
	s.circuitBreaker.mu.RUnlock()

	var hash string
	var height int
	var node string
	var err error

	if len(s.config.RPCNodes) > 1 && s.config.TurboMode {
		hash, height, node, err = s.getBestBlockParallel()
	} else {
		hash, height, node, err = s.getBestBlockSingle()
	}

	if err != nil {
		s.circuitBreaker.mu.Lock()
		s.circuitBreaker.failures++
		if s.circuitBreaker.failures >= s.circuitBreaker.maxFailures {
			s.circuitBreaker.breakUntil = time.Now().Add(30 * time.Second)
			zap.L().Warn("Circuit breaker opened",
				zap.Int("failures", s.circuitBreaker.failures),
				zap.Duration("duration", 30*time.Second))
		}
		s.circuitBreaker.mu.Unlock()
	} else {
		s.circuitBreaker.mu.Lock()
		s.circuitBreaker.failures = 0
		s.circuitBreaker.mu.Unlock()
	}

	return hash, height, node, err
}

func (s *Sprint) getBestBlockParallel() (string, int, string, error) {
	type nodeInfo struct {
		url     string
		latency time.Duration
	}
	nodes := make([]nodeInfo, 0, len(s.config.RPCNodes))
	s.perfMetrics.mu.RLock()
	for _, url := range s.config.RPCNodes {
		if s.isNodeInBackoff(url) {
			continue
		}
		latency := s.perfMetrics.nodeLatencies[url]
		nodes = append(nodes, nodeInfo{url, latency})
	}
	s.perfMetrics.mu.RUnlock()

	if len(nodes) == 0 {
		return "", 0, "", fmt.Errorf("no RPC nodes available (all in backoff)")
	}

	// Sort by historical latency (lowest first)
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].latency < nodes[j].latency
	})

	type result struct {
		hash   string
		height int
		node   string
		err    error
		took   time.Duration
	}

	results := make(chan result, len(nodes))
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	for _, ni := range nodes {
		go func(nodeURL string) {
			start := time.Now()
			hash, height, err := s.queryNodeWithRetry(nodeURL, s.getBlockchainInfoReq, 3)
			took := time.Since(start)

			select {
			case results <- result{hash, height, nodeURL, err, took}:
			case <-ctx.Done():
			}
		}(ni.url)
	}

	// Return first successful response
	for i := 0; i < len(nodes); i++ {
		select {
		case res := <-results:
			if res.err == nil {
				s.clearNodeBackoff(res.node)
				s.updateNodeLatency(res.node, res.took)
				if s.shouldLog("debug") {
					zap.L().Debug("RPC success", zap.String("node", res.node), zap.Duration("took", res.took))
				}
				return res.hash, res.height, res.node, nil
			}
			s.setNodeBackoff(res.node, res.err)
		case <-ctx.Done():
			return "", 0, "", fmt.Errorf("parallel RPC timeout")
		}
	}

	return "", 0, "", fmt.Errorf("all parallel RPC calls failed")
}

func (s *Sprint) getBestBlockSingle() (string, int, string, error) {
	node := s.config.RPCNodes[0]

	if s.isNodeInBackoff(node) {
		backoff := s.nodeBackoff[node]
		return "", 0, "", fmt.Errorf("node %s in backoff until %v (last error: %v)",
			node, backoff.until, backoff.lastError)
	}

	start := time.Now()
	hash, height, err := s.queryNodeWithRetry(node, s.getBlockchainInfoReq, 3)
	took := time.Since(start)
	if err != nil {
		s.setNodeBackoff(node, err)
		return "", 0, "", err
	}

	s.clearNodeBackoff(node)
	s.updateNodeLatency(node, took)
	return hash, height, node, nil
}

func (s *Sprint) queryNodeWithRetry(node string, reqBody []byte, retries int) (string, int, error) {
	var lastErr error
	for i := 0; i < retries; i++ {
		hash, height, err := s.queryNode(node, reqBody)
		if err == nil {
			return hash, height, nil
		}
		lastErr = err
		if s.shouldLog("debug") {
			zap.L().Debug("RPC retry", zap.Int("attempt", i+1), zap.Int("max", retries), zap.String("node", node), zap.Error(err))
		}
		time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
	}
	return "", 0, fmt.Errorf("failed after %d retries: %w", retries, lastErr)
}

func (s *Sprint) queryNode(node string, reqBody []byte) (string, int, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, node, bytes.NewReader(reqBody))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication and headers
	if s.config.RPCUser != "" || s.config.RPCPass != nil {
		secure.SetBasicAuthHeader(req, s.config.RPCUser, s.config.RPCPass)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "keep-alive")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", 0, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

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

	if envelope.Error != nil {
		return "", 0, fmt.Errorf("RPC error %d: %s", envelope.Error.Code, envelope.Error.Message)
	}

	if envelope.Result.BestHash == "" {
		return "", 0, fmt.Errorf("empty block hash in response")
	}

	return envelope.Result.BestHash, envelope.Result.Height, nil
}

func (s *Sprint) updateNodeLatency(node string, latency time.Duration) {
	s.perfMetrics.mu.Lock()
	defer s.perfMetrics.mu.Unlock()

	current := s.perfMetrics.nodeLatencies[node]
	if current == 0 {
		s.perfMetrics.nodeLatencies[node] = latency
	} else {
		// Exponential moving average
		alpha := 0.2
		s.perfMetrics.nodeLatencies[node] = time.Duration(float64(current)*(1-alpha) + float64(latency)*alpha)
	}
}

// ───────────────────────── Node Backoff Management ─────────────────────────

func (s *Sprint) isNodeInBackoff(node string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if backoff, exists := s.nodeBackoff[node]; exists {
		return time.Now().Before(backoff.until)
	}
	return false
}

func (s *Sprint) setNodeBackoff(node string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	backoff := s.nodeBackoff[node]
	if backoff == nil {
		backoff = &BackoffState{attempts: 0}
		s.nodeBackoff[node] = backoff
	}

	backoff.attempts++
	backoff.lastError = err

	// Exponential backoff with jitter
	baseDelay := time.Duration(math.Pow(2, float64(backoff.attempts-1))) * time.Second
	if baseDelay > 60*time.Second {
		baseDelay = 60 * time.Second
	}

	jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
	backoff.until = time.Now().Add(baseDelay + jitter)

	zap.L().Warn("Node entering backoff",
		zap.String("node", node),
		zap.Duration("duration", baseDelay+jitter),
		zap.Int("attempt", backoff.attempts),
		zap.Error(err))
}

func (s *Sprint) clearNodeBackoff(node string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if backoff, exists := s.nodeBackoff[node]; exists && backoff.attempts > 0 {
		zap.L().Info("Node recovered", zap.String("node", node), zap.Int("failed_attempts", backoff.attempts))
	}

	delete(s.nodeBackoff, node)
}

// ───────────────────────── Block Broadcasting ─────────────────────────

func (s *Sprint) OnNewBlock(hash string, height int, node string, messageID string) {
	start := time.Now()

	// Check license limits
	if s.isBlockLimitReached() {
		zap.L().Warn("Block limit reached",
			zap.String("tier", s.license.Tier),
			zap.Int("limit", s.license.BlockLimit))
		return
	}

	// Generate a unique message ID if not provided (for locally detected blocks)
	if messageID == "" {
		messageID = hex.EncodeToString([]byte(hash + strconv.FormatInt(time.Now().UnixNano(), 16)))
	}

	// Check if message was already processed
	s.seenMessagesMu.Lock()
	if _, exists := s.seenMessages[messageID]; exists {
		s.seenMessagesMu.Unlock()
		zap.L().Debug("Skipping already processed block message",
			zap.String("hash_prefix", hash[:8]),
			zap.String("message_id", messageID))
		return
	}
	s.seenMessages[messageID] = time.Now()
	s.seenMessagesMu.Unlock()

	var sent int
	if s.config.TurboMode {
		sent = s.SprintBlockTurbo(hash, messageID)
	} else {
		sent = s.SprintBlock(hash, messageID)
	}

	latency := float64(time.Since(start).Milliseconds())
	atomic.AddInt64(&s.blocksSent, 1)

	// Create metrics
	m := Metrics{
		BlockHash:  hash,
		Height:     height,
		Latency:    latency,
		PeerCount:  sent,
		Timestamp:  time.Now().Unix(),
		LicenseKey: maskLicenseKeyFromSecure(s.config.LicenseKey),
		RPCNode:    node,
		Success:    sent > 0,
	}

	// Store latest metric
	s.mu.Lock()
	s.latestMetric = &m
	s.mu.Unlock()

	// Send to metrics channel (with overflow protection)
	select {
	case s.metrics <- m:
	default:
		zap.L().Warn("Metrics buffer full, dropping metric", zap.String("block_hash_prefix", hash[:8]))
		// Drain oldest to make room
		select {
		case <-s.metrics:
		default:
		}
		s.metrics <- m
	}

	// Update performance metrics
	s.perfMetrics.mu.Lock()
	s.perfMetrics.totalBlocks++
	if s.perfMetrics.totalBlocks == 1 {
		s.perfMetrics.avgSprintTime = time.Duration(latency) * time.Millisecond
		s.perfMetrics.avgBlockDetection = time.Since(s.lastBlockTime)
	} else {
		alpha := 0.1
		s.perfMetrics.avgSprintTime = time.Duration(float64(s.perfMetrics.avgSprintTime)*(1-alpha) + latency*float64(time.Millisecond)*alpha)
		s.perfMetrics.avgBlockDetection = time.Duration(float64(s.perfMetrics.avgBlockDetection)*(1-alpha) + float64(time.Since(s.lastBlockTime))*alpha)
	}
	s.perfMetrics.mu.Unlock()

	zap.L().Info("Block sprinted",
		zap.String("hash_prefix", hash[:8]),
		zap.Int("height", height),
		zap.Float64("latency_ms", latency),
		zap.Int("peers", sent),
		zap.String("node", node),
		zap.String("message_id", messageID))
}

func (s *Sprint) isBlockLimitReached() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	switch s.license.Tier {
	case "free":
		return atomic.LoadInt64(&s.blocksSent) >= int64(s.license.BlockLimit)
	case "pro":
		return atomic.LoadInt64(&s.blocksSent) >= int64(s.license.BlockLimit*2)
	case "enterprise":
		return false // Unlimited for enterprise
	default:
		return true
	}
}

func (s *Sprint) SprintBlock(hash string, messageID string) int {
	return s.sprintBlockInternal(hash, messageID, false)
}

func (s *Sprint) SprintBlockTurbo(hash string, messageID string) int {
	return s.sprintBlockInternal(hash, messageID, true)
}

func (s *Sprint) sprintBlockInternal(hash string, messageID string, turbo bool) int {
	s.peersMu.RLock()
	peers := make([]*PeerConnection, 0, len(s.peers))
	for _, peer := range s.peers {
		if peer.authed {
			peers = append(peers, peer)
		}
	}
	s.peersMu.RUnlock()

	if len(peers) == 0 {
		return 0
	}

	// Prepare message
	message := s.prepareMessage(hash, messageID)

	var wg sync.WaitGroup
	var successCount atomic.Int32

	// Fan-out concurrency limit
	concurrency := len(peers)
	if turbo {
		concurrency = min(concurrency, 100)
	} else {
		concurrency = min(concurrency, 50)
	}

	sem := make(chan struct{}, concurrency)
	for _, peer := range peers {
		sem <- struct{}{}
		wg.Add(1)
		go func(p *PeerConnection) {
			defer wg.Done()
			defer func() { <-sem }()

			if s.sendToPeer(p, message) {
				successCount.Add(1)
				atomic.AddInt64(&p.successes, 1)
				p.lastSent = time.Now()
			} else {
				atomic.AddInt64(&p.failures, 1)
			}
		}(peer)
	}

	wg.Wait()
	return int(successCount.Load())
}

func (s *Sprint) prepareMessage(hash string, messageID string) []byte {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	msg := bytes.ReplaceAll(s.preEncodedPayload, []byte("HASH_PLACEHOLDER"), []byte(hash))
	msg = bytes.ReplaceAll(msg, []byte("TS_PLACEHOLDER"), []byte(ts))
	msg = bytes.ReplaceAll(msg, []byte("MESSAGE_ID_PLACEHOLDER"), []byte(messageID))
	return msg
}

func (s *Sprint) sendToPeer(peer *PeerConnection, message []byte) bool {
	if peer.conn == nil || !peer.authed {
		return false
	}

	if err := peer.conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		zap.L().Error("Failed to set write deadline", zap.String("peer", peer.addr), zap.Error(err))
		return false
	}

	_, err := peer.conn.Write(message)
	if err != nil {
		zap.L().Error("Failed to send to peer", zap.String("peer", peer.addr), zap.Error(err))
		peer.conn.Close()
		peer.conn = nil
		peer.authed = false
		return false
	}

	return true
}

// ───────────────────────── Peers ─────────────────────────

func (s *Sprint) ConnectToPeers() {
	zap.L().Info("Starting peer connection manager...")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.connectToPeersOnce()
		}
	}
}

func (s *Sprint) connectToPeersOnce() {
	s.peersMu.Lock()
	defer s.peersMu.Unlock()

	// Disconnect stale peers
	for addr, peer := range s.peers {
		if time.Since(peer.connected) > 2*time.Hour || peer.failures > 5 || !peer.authed {
			if peer.conn != nil {
				peer.conn.Close()
			}
			delete(s.peers, addr)
			zap.L().Info("Disconnected stale peer",
				zap.String("addr", addr),
				zap.Int64("failures", peer.failures),
				zap.Bool("authed", peer.authed))
		}
	}

	// Connect to new peers up to MaxPeers
	currentPeers := len(s.peers)
	for _, addr := range s.license.Peers {
		if currentPeers >= s.config.MaxPeers {
			break
		}
		if _, exists := s.peers[addr]; exists {
			continue
		}

		conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
		if err != nil {
			zap.L().Warn("Failed to connect to peer", zap.String("addr", addr), zap.Error(err))
			s.perfMetrics.mu.Lock()
			s.perfMetrics.failedConnections++
			s.perfMetrics.mu.Unlock()
			continue
		}

		// Perform handshake
		peer := &PeerConnection{
			conn:      conn,
			addr:      addr,
			connected: time.Now(),
		}
		if err := s.performPeerHandshake(peer); err != nil {
			zap.L().Warn("Peer handshake failed", zap.String("addr", addr), zap.Error(err))
			conn.Close()
			s.perfMetrics.mu.Lock()
			s.perfMetrics.failedConnections++
			s.perfMetrics.mu.Unlock()
			continue
		}

		s.peers[addr] = peer
		currentPeers++
		zap.L().Info("Connected and authenticated peer", zap.String("addr", addr))
	}

	zap.L().Info("Active peers", zap.Int("count", len(s.peers)), zap.Int("max", s.config.MaxPeers))
}

func (s *Sprint) performPeerHandshake(peer *PeerConnection) error {
	// Create and send handshake message without copying the license into a
	// long-lived Go string: marshal and write bytes inside WithBytes.
	if s.config.LicenseKey != nil {
		if err := s.config.LicenseKey.WithBytes(func(b []byte) error {
			handshake := PeerHandshake{
				LicenseKey: string(b),
				Timestamp:  time.Now().Unix(),
			}
			sigData := []byte(string(b) + strconv.FormatInt(handshake.Timestamp, 10))
			if s.config.PeerSecret != nil {
				handshake.Signature = s.config.PeerSecret.HMACHex(sigData)
			} else {
				mac := hmac.New(sha256.New, []byte(""))
				mac.Write(sigData)
				handshake.Signature = hex.EncodeToString(mac.Sum(nil))
			}
			data, merr := json.Marshal(handshake)
			if merr != nil {
				return merr
			}
			data = append(data, '\n')

			if err := peer.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
				return fmt.Errorf("failed to set write deadline: %w", err)
			}
			if _, werr := peer.conn.Write(data); werr != nil {
				return fmt.Errorf("failed to send handshake: %w", werr)
			}
			// zero marshal buffer
			for i := range data {
				data[i] = 0
			}
			return nil
		}); err != nil {
			return err
		}
	} else {
		handshake := PeerHandshake{LicenseKey: "", Timestamp: time.Now().Unix()}
		data, err := json.Marshal(handshake)
		if err != nil {
			return fmt.Errorf("failed to marshal handshake: %w", err)
		}
		data = append(data, '\n')
		if err := peer.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
			return fmt.Errorf("failed to set write deadline: %w", err)
		}
		if _, err := peer.conn.Write(data); err != nil {
			return fmt.Errorf("failed to send handshake: %w", err)
		}
	}

	// Read response
	if err := peer.conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return fmt.Errorf("failed to set read deadline: %w", err)
	}
	respData, err := io.ReadAll(io.LimitReader(peer.conn, 1024))
	if err != nil {
		return fmt.Errorf("failed to read handshake response: %w", err)
	}

	var resp PeerHandshake
	if err := json.Unmarshal(respData, &resp); err != nil {
		return fmt.Errorf("failed to parse handshake response: %w", err)
	}

	// Verify response signature
	respSigData := []byte(resp.LicenseKey + strconv.FormatInt(resp.Timestamp, 10))
	var expectedSig string
	if s.config.PeerSecret != nil {
		expectedSig = s.config.PeerSecret.HMACHex(respSigData)
	} else {
		mac := hmac.New(sha256.New, []byte(""))
		mac.Write(respSigData)
		expectedSig = hex.EncodeToString(mac.Sum(nil))
	}
	if resp.Signature != expectedSig || time.Now().Unix()-resp.Timestamp > 30 {
		return fmt.Errorf("invalid handshake signature or timestamp")
	}

	peer.authed = true
	return nil
}

// StartPeerListener starts a TCP server that accepts inbound peer connections.
func (s *Sprint) StartPeerListener(listenAddr string) {
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		zap.L().Fatal("Failed to start peer listener", zap.String("addr", listenAddr), zap.Error(err))
	}

	zap.L().Info("Peer listener started", zap.String("addr", listenAddr))

	go func() {
		defer ln.Close()
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-s.ctx.Done():
					return
				default:
					zap.L().Warn("Failed to accept peer connection", zap.Error(err))
					continue
				}
			}

			go s.handleInboundPeer(conn)
		}
	}()
}

func (s *Sprint) handleInboundPeer(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			zap.L().Error("Recovered from panic in inbound peer handler", zap.Any("err", r))
		}
		conn.Close()
	}()

	peerAddr := conn.RemoteAddr().String()

	// Read handshake
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		zap.L().Error("Failed to set read deadline", zap.Error(err))
		return
	}

	data, err := io.ReadAll(io.LimitReader(conn, 1024))
	if err != nil {
		zap.L().Warn("Failed to read handshake", zap.String("peer", peerAddr), zap.Error(err))
		return
	}

	var hs PeerHandshake
	if err := json.Unmarshal(data, &hs); err != nil {
		zap.L().Warn("Invalid handshake JSON", zap.String("peer", peerAddr), zap.Error(err))
		return
	}

	// Verify signature
	sigData := []byte(hs.LicenseKey + strconv.FormatInt(hs.Timestamp, 10))
	mac := hmac.New(sha256.New, []byte(s.config.PeerSecret))
	mac.Write(sigData)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if hs.Signature != expectedSig || time.Now().Unix()-hs.Timestamp > 30 {
		zap.L().Warn("Invalid handshake signature", zap.String("peer", peerAddr))
		return
	}

	// Send handshake response without exposing license as a long-lived string.
	if s.config.LicenseKey != nil {
		if err := s.config.LicenseKey.WithBytes(func(b []byte) error {
			resp := PeerHandshake{LicenseKey: string(b), Timestamp: time.Now().Unix()}
			respSigData := []byte(string(b) + strconv.FormatInt(resp.Timestamp, 10))
			if s.config.PeerSecret != nil {
				resp.Signature = s.config.PeerSecret.HMACHex(respSigData)
			} else {
				mac = hmac.New(sha256.New, []byte(""))
				mac.Write(respSigData)
				resp.Signature = hex.EncodeToString(mac.Sum(nil))
			}
			respData, _ := json.Marshal(resp)
			respData = append(respData, '\n')

			if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
				zap.L().Error("Failed to set write deadline", zap.Error(err))
				return err
			}
			if _, werr := conn.Write(respData); werr != nil {
				zap.L().Warn("Failed to send handshake response", zap.String("peer", peerAddr), zap.Error(werr))
				return werr
			}
			for i := range respData {
				respData[i] = 0
			}
			return nil
		}); err != nil {
			zap.L().Warn("Failed to marshal/send handshake response", zap.Error(err))
			return
		}
	} else {
		resp := PeerHandshake{LicenseKey: "", Timestamp: time.Now().Unix()}
		respData, _ := json.Marshal(resp)
		respData = append(respData, '\n')
		if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
			zap.L().Error("Failed to set write deadline", zap.Error(err))
			return
		}
		if _, err := conn.Write(respData); err != nil {
			zap.L().Warn("Failed to send handshake response", zap.String("peer", peerAddr), zap.Error(err))
			return
		}
	}

	// Add peer
	peer := &PeerConnection{
		conn:      conn,
		addr:      peerAddr,
		connected: time.Now(),
		authed:    true,
	}

	s.peersMu.Lock()
	if len(s.peers) < s.config.MaxPeers {
		s.peers[peer.addr] = peer
		zap.L().Info("Inbound peer authenticated", zap.String("addr", peer.addr))
	} else {
		zap.L().Warn("Max peers reached, closing connection", zap.String("addr", peer.addr))
		s.peersMu.Unlock()
		return
	}
	s.peersMu.Unlock()

	// Start reading block messages for gossip relay
	for {
		if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
			zap.L().Error("Failed to set read deadline for block message", zap.String("peer", peerAddr), zap.Error(err))
			s.removePeer(peerAddr)
			return
		}

		data, err := io.ReadAll(io.LimitReader(conn, 1024))
		if err != nil {
			if err == io.EOF || strings.Contains(err.Error(), "closed") {
				zap.L().Info("Peer disconnected", zap.String("peer", peerAddr))
			} else {
				zap.L().Warn("Failed to read block message", zap.String("peer", peerAddr), zap.Error(err))
			}
			s.removePeer(peerAddr)
			return
		}

		var msg BlockMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			zap.L().Warn("Invalid block message JSON", zap.String("peer", peerAddr), zap.Error(err))
			continue
		}

		if msg.Type != "block" || msg.Protocol != 2 {
			zap.L().Warn("Invalid block message", zap.String("peer", peerAddr), zap.String("type", msg.Type), zap.Int("protocol", msg.Protocol))
			continue
		}

		// Check if message was already processed
		s.seenMessagesMu.Lock()
		if _, exists := s.seenMessages[msg.MessageID]; exists {
			s.seenMessagesMu.Unlock()
			zap.L().Debug("Skipping already processed block message",
				zap.String("hash_prefix", msg.Hash[:8]),
				zap.String("message_id", msg.MessageID),
				zap.String("peer", peerAddr))
			continue
		}
		s.seenMessages[msg.MessageID] = time.Now()
		s.seenMessagesMu.Unlock()

		// Validate timestamp
		ts, err := strconv.ParseInt(msg.Ts, 10, 64)
		if err != nil || time.Now().Unix()-ts > 60 {
			zap.L().Warn("Invalid or stale block message timestamp",
				zap.String("peer", peerAddr),
				zap.String("ts", msg.Ts))
			continue
		}

		// Log received block
		zap.L().Info("Received block from peer",
			zap.String("hash_prefix", msg.Hash[:8]),
			zap.String("peer", peerAddr),
			zap.String("message_id", msg.MessageID))

		// Trigger OnNewBlock for relay (height and node are unknown for relayed blocks)
		s.OnNewBlock(msg.Hash, 0, peerAddr, msg.MessageID)
	}
}

func (s *Sprint) removePeer(addr string) {
	s.peersMu.Lock()
	defer s.peersMu.Unlock()
	if peer, exists := s.peers[addr]; exists {
		if peer.conn != nil {
			peer.conn.Close()
		}
		delete(s.peers, addr)
		zap.L().Info("Removed peer", zap.String("addr", addr))
	}
}

// ───────────────────────── Dashboard ─────────────────────────

func (s *Sprint) StartWebDashboard() {
	mux := http.NewServeMux()

	// Register endpoints
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/latest", s.handleLatest)
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.HandleFunc("/predictive", s.handlePredictive)
	mux.HandleFunc("/stream", s.handleStream)

	// Secure middleware
	securedMux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		mux.ServeHTTP(w, r)
	})

	port := s.getDashboardPort()
	server := &http.Server{
		Addr:         port,
		Handler:      securedMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	zap.L().Info("Starting web dashboard", zap.String("port", port))
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Error("Web dashboard error", zap.Error(err))
		}
	}()

	// Handle graceful shutdown
	<-s.ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

func (s *Sprint) handleStatus(w http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr
	if !s.rateLimiter.Allow(clientIP, "/status", s.config.TurboMode) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	s.mu.RLock()
	resp := StatusResponse{
		Tier:             s.license.Tier,
		LicenseKey:       maskLicenseKey(s.config.LicenseKey),
		Valid:            s.license.Valid,
		BlocksSentToday:  atomic.LoadInt64(&s.blocksSent),
		BlockLimit:       s.license.BlockLimit,
		PeersConnected:   len(s.peers),
		UptimeSeconds:    int64(time.Since(s.startTime).Seconds()),
		Version:          Version,
		TurboModeEnabled: s.config.TurboMode,
		LastBlockTime:    s.lastBlockTime,
		RPCNodesActive:   len(s.config.RPCNodes) - len(s.nodeBackoff),
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Sprint) handleLatest(w http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr
	if !s.rateLimiter.Allow(clientIP, "/latest", s.config.TurboMode) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	s.mu.RLock()
	if s.latestMetric == nil {
		s.mu.RUnlock()
		http.Error(w, "No blocks detected yet", http.StatusServiceUnavailable)
		return
	}
	metric := *s.latestMetric
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metric)
}

func (s *Sprint) handleMetrics(w http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr
	if !s.rateLimiter.Allow(clientIP, "/metrics", s.config.TurboMode) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Collect recent metrics (last 100 or buffer size)
	metrics := make([]Metrics, 0, 100)
	for i := 0; i < 100; i++ {
		select {
		case m := <-s.metrics:
			metrics = append(metrics, m)
		default:
			return // Exit the loop when no more metrics available
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (s *Sprint) handlePredictive(w http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr
	if !s.rateLimiter.Allow(clientIP, "/predictive", s.config.TurboMode) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	s.mu.RLock()
	resp := PredictiveResponse{
		MempoolSize:             s.lastMempool,
		Trend:                   s.calculateMempoolTrend(),
		ProbabilityNextBlock60s: s.calculateBlockProbability(),
		LastUpdate:              time.Now().Unix(),
		AverageBlockTime:        s.perfMetrics.avgBlockDetection.Minutes(),
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Sprint) handleStream(w http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr
	if !s.rateLimiter.Allow(clientIP, "/stream", s.config.TurboMode) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	for {
		select {
		case m := <-s.metrics:
			data, _ := json.Marshal(m)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Sprint) getDashboardPort() string {
	u, err := url.Parse(s.config.APIBase)
	if err != nil {
		return ":8080"
	}
	return u.Host
}

func maskLicenseKey(key string) string {
	if len(key) < 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// maskLicenseKeyFromSecure masks a license key stored in a SecureBuffer without
// copying it into a long-lived Go string.
func maskLicenseKeyFromSecure(key *secure.SecureBuffer) string {
	if key == nil {
		return "****"
	}
	var out string
	_ = key.WithBytes(func(b []byte) error {
		if len(b) < 8 {
			out = "****"
			return nil
		}
		out = string(b[:4]) + "****" + string(b[len(b)-4:])
		return nil
	})
	return out
}

// ───────────────────────── Predictive Monitoring ─────────────────────────

func (s *Sprint) StartPredictiveMonitoring() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			mempoolSize, err := s.getMempoolSize()
			if err != nil {
				if s.shouldLog("debug") {
					zap.L().Debug("Mempool query failed", zap.Error(err))
				}
				continue
			}
			s.mu.Lock()
			s.lastMempool = mempoolSize
			s.mu.Unlock()
		}
	}
}

func (s *Sprint) getMempoolSize() (int, error) {
	_, _, node, err := s.getBestBlockSingle()
	if err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(s.ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, node, bytes.NewReader(s.getMempoolInfoReq))
	if err != nil {
		return 0, fmt.Errorf("failed to create mempool request: %w", err)
	}
	secure.SetBasicAuthHeader(req, s.config.RPCUser, s.config.RPCPass)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("mempool request failed: %w", err)
	}
	defer resp.Body.Close()

	var envelope struct {
		Result struct {
			Size int `json:"size"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return 0, fmt.Errorf("decode mempool failed: %w", err)
	}
	if envelope.Error != nil {
		return 0, fmt.Errorf("mempool RPC error %d: %s", envelope.Error.Code, envelope.Error.Message)
	}

	return envelope.Result.Size, nil
}

func (s *Sprint) calculateMempoolTrend() string {
	if s.lastMempool > 1000 {
		return "high"
	} else if s.lastMempool > 100 {
		return "medium"
	}
	return "low"
}

func (s *Sprint) calculateBlockProbability() float64 {
	mempool := float64(s.lastMempool)
	if mempool < 100 {
		return 0.1
	} else if mempool < 1000 {
		return 0.5
	}
	return 0.9
}

// ───────────────────────── Metrics Reporter ─────────────────────────

func (s *Sprint) StartMetricsReporter() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.reportMetrics()
		}
	}
}

func (s *Sprint) reportMetrics() {
	var metrics []Metrics

	// Collect up to 100 metrics or until no more available
	for len(metrics) < 100 {
		select {
		case m := <-s.metrics:
			metrics = append(metrics, m)
		default:
			// No more metrics available
			if len(metrics) == 0 {
				return
			}
			// Process what we have
			break
		}
	}

	// Async POST to metrics endpoint
	go func(metrics []Metrics) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		data, err := json.Marshal(metrics)
		if err != nil {
			zap.L().Error("Failed to marshal metrics", zap.Error(err))
			return
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.MetricsURL, bytes.NewReader(data))
		if err != nil {
			zap.L().Error("Failed to create metrics request", zap.Error(err))
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.client.Do(req)
		if err != nil {
			zap.L().Error("Failed to send metrics", zap.Error(err))
			// Re-queue metrics for next attempt
			for _, m := range metrics {
				select {
				case s.metrics <- m:
				default:
					zap.L().Warn("Metrics buffer full, dropping metric", zap.String("block_hash_prefix", m.BlockHash[:8]))
				}
			}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
			zap.L().Error("Metrics endpoint error", zap.Int("status", resp.StatusCode), zap.String("body", string(body)))
			return
		}

		zap.L().Info("Reported metrics batch", zap.Int("count", len(metrics)))
	}(metrics)
}

// ───────────────────────── Performance Monitor ─────────────────────────

func (s *Sprint) StartPerformanceMonitor() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.perfMetrics.mu.RLock()
			zap.L().Info("Performance metrics",
				zap.Int64("total_blocks", s.perfMetrics.totalBlocks),
				zap.Float64("avg_sprint_time_ms", s.perfMetrics.avgSprintTime.Seconds()*1000),
				zap.Int64("failed_connections", s.perfMetrics.failedConnections),
				zap.Float64("uptime_hours", time.Since(s.perfMetrics.startTime).Hours()))
			s.perfMetrics.mu.RUnlock()
		}
	}
}

// ───────────────────────── Helpers ─────────────────────────

func (s *Sprint) shouldLog(level string) bool {
	lvl := strings.ToLower(s.config.LogLevel)
	return lvl == "debug" ||
		(lvl == "info" && level != "debug") ||
		(lvl == "warn" && (level == "warn" || level == "error")) ||
		(lvl == "error" && level == "error")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
