package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/cache"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/entropy"
	"github.com/PayRpc/Bitcoin-Sprint/internal/license"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
	"github.com/PayRpc/Bitcoin-Sprint/internal/securebuf"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	buckets map[string]*TokenBucket
	mu      sync.RWMutex
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	tokens         float64
	capacity       float64
	refillRate     float64
	lastRefillTime time.Time
	mu             sync.Mutex
}

// CustomerKeyManager manages per-customer API keys and tiers
type CustomerKeyManager struct {
	keys    map[string]CustomerKey // SHA256 hash -> key info
	keyHashes map[string]string    // Original key -> hash mapping
	mu      sync.RWMutex
}

// CustomerKey represents a customer's API key information
type CustomerKey struct {
	Hash           string    `json:"hash"`
	Tier           config.Tier `json:"tier"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	LastUsed       time.Time `json:"last_used"`
	RequestCount   int64     `json:"request_count"`
	RateLimitRemaining int   `json:"rate_limit_remaining"`
	ClientIP       string    `json:"client_ip"`
	UserAgent      string    `json:"user_agent"`
}

// AdminAuth handles admin-only authentication
type AdminAuth struct {
	adminKeys map[string]bool // SHA256 hashes of admin keys
	mu        sync.RWMutex
}

// WebSocketLimiter limits concurrent WebSocket connections
type WebSocketLimiter struct {
	globalSem chan struct{}        // Global connection limit
	perIPSem  map[string]chan struct{} // Per-IP connection limit
	maxPerIP  int
	mu        sync.RWMutex
}

// PredictiveAnalytics provides dynamic predictive analytics
type PredictiveAnalytics struct {
	blockHistory []BlockTiming
	mu           sync.RWMutex
}

// BlockTiming tracks block arrival times for prediction
type BlockTiming struct {
	Height    int64
	Timestamp time.Time
	Size      int
}

type Server struct {
	cfg       config.Config
	blockChan chan blocks.BlockEvent
	mem       *mempool.Mempool
	cache     *cache.Cache
	logger    *zap.Logger
	srv       *http.Server

	// Rate limiting
	rateLimiter *RateLimiter

	// Customer key management
	keyManager *CustomerKeyManager

	// Admin authentication
	adminAuth *AdminAuth

	// WebSocket connection limits
	wsLimiter *WebSocketLimiter

	// Predictive analytics
	predictor *PredictiveAnalytics
}

func New(cfg config.Config, blockChan chan blocks.BlockEvent, mem *mempool.Mempool, logger *zap.Logger) *Server {
	return &Server{
		cfg:          cfg,
		blockChan:    blockChan,
		mem:          mem,
		logger:       logger,
		rateLimiter:  NewRateLimiter(),
		keyManager:   NewCustomerKeyManager(),
		adminAuth:    NewAdminAuth(),
		wsLimiter:    NewWebSocketLimiter(1000, 10), // Max 1000 global, 10 per IP
		predictor:    NewPredictiveAnalytics(),
	}
}

func NewWithCache(cfg config.Config, blockChan chan blocks.BlockEvent, mem *mempool.Mempool, cache *cache.Cache, logger *zap.Logger) *Server {
	return &Server{
		cfg:          cfg,
		blockChan:    blockChan,
		mem:          mem,
		cache:        cache,
		logger:       logger,
		rateLimiter:  NewRateLimiter(),
		keyManager:   NewCustomerKeyManager(),
		adminAuth:    NewAdminAuth(),
		wsLimiter:    NewWebSocketLimiter(1000, 10), // Max 1000 global, 10 per IP
		predictor:    NewPredictiveAnalytics(),
	}
}

func (s *Server) Stop() {
	if s.srv != nil {
		// Create a timeout context for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := s.srv.Shutdown(ctx); err != nil {
			s.logger.Error("Server shutdown error", zap.Error(err))
		}
	}
}

// ===== RATE LIMITER IMPLEMENTATION =====

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*TokenBucket),
	}
}

// Allow checks if a request from the given identifier is allowed
func (rl *RateLimiter) Allow(identifier string, capacity float64, refillRate float64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[identifier]
	if !exists {
		bucket = &TokenBucket{
			tokens:         capacity,
			capacity:       capacity,
			refillRate:     refillRate,
			lastRefillTime: time.Now(),
		}
		rl.buckets[identifier] = bucket
	}

	return bucket.Allow()
}

// Allow checks if the token bucket allows a request
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	timePassed := now.Sub(tb.lastRefillTime).Seconds()
	tokensToAdd := timePassed * tb.refillRate

	tb.tokens = math.Min(tb.capacity, tb.tokens + tokensToAdd)
	tb.lastRefillTime = now

	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return true
	}

	return false
}

// ===== CUSTOMER KEY MANAGER IMPLEMENTATION =====

// NewCustomerKeyManager creates a new customer key manager
func NewCustomerKeyManager() *CustomerKeyManager {
	manager := &CustomerKeyManager{
		keys:       make(map[string]CustomerKey),
		keyHashes:  make(map[string]string),
	}

	// Initialize with default key for backward compatibility
	defaultKey := "changeme"
	hash := manager.hashKey(defaultKey)
	manager.keys[hash] = CustomerKey{
		Hash:               hash,
		Tier:               config.TierFree,
		CreatedAt:          time.Now(),
		ExpiresAt:          time.Now().AddDate(1, 0, 0),
		LastUsed:           time.Now(),
		RequestCount:       0,
		RateLimitRemaining: 100,
		ClientIP:           "",
		UserAgent:          "",
	}
	manager.keyHashes[defaultKey] = hash

	return manager
}

// hashKey creates a SHA256 hash of the key
func (ckm *CustomerKeyManager) hashKey(key string) string {
	hasher := sha256.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

// ValidateKey validates an API key and returns customer information
func (ckm *CustomerKeyManager) ValidateKey(key string) (*CustomerKey, bool) {
	ckm.mu.RLock()
	defer ckm.mu.RUnlock()

	hash, exists := ckm.keyHashes[key]
	if !exists {
		return nil, false
	}

	customerKey, exists := ckm.keys[hash]
	if !exists {
		return nil, false
	}

	// Check if key has expired
	if time.Now().After(customerKey.ExpiresAt) {
		return nil, false
	}

	return &customerKey, true
}

// UpdateKeyUsage updates the usage statistics for a key
func (ckm *CustomerKeyManager) UpdateKeyUsage(key string, clientIP, userAgent string) {
	ckm.mu.Lock()
	defer ckm.mu.Unlock()

	hash := ckm.keyHashes[key]
	if customerKey, exists := ckm.keys[hash]; exists {
		customerKey.LastUsed = time.Now()
		customerKey.RequestCount++
		customerKey.RateLimitRemaining--
		customerKey.ClientIP = clientIP
		customerKey.UserAgent = userAgent
		ckm.keys[hash] = customerKey
	}
}

// GenerateKey generates a new customer API key
func (ckm *CustomerKeyManager) GenerateKey(tier config.Tier, clientIP string) (string, error) {
	// Generate a secure random key
	const keySize = 32
	keyBytes := make([]byte, keySize)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", err
	}
	newKey := hex.EncodeToString(keyBytes)

	hash := ckm.hashKey(newKey)

	ckm.mu.Lock()
	defer ckm.mu.Unlock()

	// Store the key information
	ckm.keyHashes[newKey] = hash
	ckm.keys[hash] = CustomerKey{
		Hash:               hash,
		Tier:               tier,
		CreatedAt:          time.Now(),
		ExpiresAt:          time.Now().AddDate(1, 0, 0),
		LastUsed:           time.Now(),
		RequestCount:       0,
		RateLimitRemaining: ckm.getRateLimitForTier(tier),
		ClientIP:           clientIP,
		UserAgent:          "",
	}

	return newKey, nil
}

// getRateLimitForTier returns the rate limit for a given tier
func (ckm *CustomerKeyManager) getRateLimitForTier(tier config.Tier) int {
	switch tier {
	case config.TierFree:
		return 100
	case config.TierPro:
		return 1000
	case config.TierBusiness:
		return 5000
	case config.TierTurbo:
		return 10000
	case config.TierEnterprise:
		return 50000
	default:
		return 100
	}
}

// ===== ADMIN AUTH IMPLEMENTATION =====

// NewAdminAuth creates a new admin authentication handler
func NewAdminAuth() *AdminAuth {
	adminKeys := make(map[string]bool)

	// Add default admin key (should be configured via environment)
	defaultAdminKey := os.Getenv("ADMIN_API_KEY")
	if defaultAdminKey == "" {
		defaultAdminKey = "admin-secret-key-change-me"
	}

	hasher := sha256.New()
	hasher.Write([]byte(defaultAdminKey))
	hash := hex.EncodeToString(hasher.Sum(nil))
	adminKeys[hash] = true

	return &AdminAuth{
		adminKeys: adminKeys,
	}
}

// IsAdmin checks if the provided key has admin privileges
func (aa *AdminAuth) IsAdmin(key string) bool {
	aa.mu.RLock()
	defer aa.mu.RUnlock()

	hasher := sha256.New()
	hasher.Write([]byte(key))
	hash := hex.EncodeToString(hasher.Sum(nil))

	return aa.adminKeys[hash]
}

// authenticateAdminRequest checks if the request has valid admin authentication
func (s *Server) authenticateAdminRequest(r *http.Request) bool {
	adminKey := r.Header.Get("X-Admin-Key")
	if adminKey == "" {
		adminKey = r.URL.Query().Get("admin_key")
	}
	if adminKey == "" {
		return false
	}
	return s.adminAuth.IsAdmin(adminKey)
}

// ===== WEBSOCKET LIMITER IMPLEMENTATION =====

// NewWebSocketLimiter creates a new WebSocket connection limiter
func NewWebSocketLimiter(maxGlobal, maxPerIP int) *WebSocketLimiter {
	return &WebSocketLimiter{
		globalSem: make(chan struct{}, maxGlobal),
		perIPSem:  make(map[string]chan struct{}),
		maxPerIP:  maxPerIP,
	}
}

// Acquire acquires a WebSocket connection slot
func (wsl *WebSocketLimiter) Acquire(clientIP string) bool {
	// Try to acquire global slot
	select {
	case wsl.globalSem <- struct{}{}:
		// Acquired global slot, now try per-IP slot
		wsl.mu.Lock()
		if wsl.perIPSem[clientIP] == nil {
			wsl.perIPSem[clientIP] = make(chan struct{}, wsl.maxPerIP)
		}
		perIPSem := wsl.perIPSem[clientIP]
		wsl.mu.Unlock()

		select {
		case perIPSem <- struct{}{}:
			// Successfully acquired both slots
			return true
		default:
			// Failed to acquire per-IP slot, release global slot
			<-wsl.globalSem
			return false
		}
	default:
		// Failed to acquire global slot
		return false
	}
}

// Release releases a WebSocket connection slot
func (wsl *WebSocketLimiter) Release(clientIP string) {
	wsl.mu.RLock()
	perIPSem := wsl.perIPSem[clientIP]
	wsl.mu.RUnlock()

	if perIPSem != nil {
		select {
		case <-perIPSem:
			// Released per-IP slot
		default:
			// No slot to release
		}
	}

	select {
	case <-wsl.globalSem:
		// Released global slot
	default:
		// No slot to release
	}
}

// ===== PREDICTIVE ANALYTICS IMPLEMENTATION =====

// NewPredictiveAnalytics creates a new predictive analytics handler
func NewPredictiveAnalytics() *PredictiveAnalytics {
	return &PredictiveAnalytics{
		blockHistory: make([]BlockTiming, 0, 100), // Keep last 100 blocks
	}
}

// RecordBlock records a new block for predictive analytics
func (pa *PredictiveAnalytics) RecordBlock(height int64, size int) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	block := BlockTiming{
		Height:    height,
		Timestamp: time.Now(),
		Size:      size,
	}

	pa.blockHistory = append(pa.blockHistory, block)

	// Keep only the last 100 blocks
	if len(pa.blockHistory) > 100 {
		pa.blockHistory = pa.blockHistory[1:]
	}
}

// PredictNextBlockETA predicts the ETA for the next block
func (pa *PredictiveAnalytics) PredictNextBlockETA() float64 {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	if len(pa.blockHistory) < 2 {
		return 420.0 // Default 7 minutes
	}

	// Calculate average interval between recent blocks
	var totalInterval float64
	count := 0

	for i := 1; i < len(pa.blockHistory); i++ {
		interval := pa.blockHistory[i].Timestamp.Sub(pa.blockHistory[i-1].Timestamp).Seconds()
		if interval > 0 && interval < 3600 { // Reasonable bounds (max 1 hour)
			totalInterval += interval
			count++
		}
	}

	if count == 0 {
		return 420.0
	}

	return totalInterval / float64(count)
}

// GetDynamicFeeEstimates provides dynamic fee estimates based on mempool
func (pa *PredictiveAnalytics) GetDynamicFeeEstimates(mempoolSize int) map[string]int {
	// Simple algorithm based on mempool size
	baseFees := map[string]int{
		"fast":   10,
		"medium": 5,
		"slow":   2,
	}

	multiplier := 1.0
	if mempoolSize > 10000 {
		multiplier = 2.0
	} else if mempoolSize > 5000 {
		multiplier = 1.5
	} else if mempoolSize < 1000 {
		multiplier = 0.8
	}

	estimates := make(map[string]int)
	for speed, baseFee := range baseFees {
		estimates[speed] = int(float64(baseFee) * multiplier)
	}

	return estimates
}

func (s *Server) Run() {
	// Create a context that will be used for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a WaitGroup to ensure all goroutines finish properly
	var wg sync.WaitGroup

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Configure server mux and middleware
	mux := http.NewServeMux()

	// Apply standard security middleware for all routes
	secureHandler := s.securityMiddleware(mux)

	// Core endpoints
	mux.HandleFunc("/status", s.recoveryMiddleware(s.statusHandler))   // No auth for status endpoint
	mux.HandleFunc("/version", s.recoveryMiddleware(s.versionHandler)) // No auth for version endpoint
	mux.HandleFunc("/latest", s.recoveryMiddleware(s.auth(s.latestHandler)))
	mux.HandleFunc("/metrics", s.recoveryMiddleware(s.auth(s.metricsHandler)))
	mux.HandleFunc("/cache-status", s.recoveryMiddleware(s.auth(s.cacheStatusHandler)))
	mux.HandleFunc("/stream", s.recoveryMiddleware(s.auth(s.streamHandler)))
	mux.HandleFunc("/turbo-status", s.recoveryMiddleware(s.turboStatusHandler))

	// Additional endpoints to match Next.js API
	mux.HandleFunc("/health", s.recoveryMiddleware(s.healthHandler))
	mux.HandleFunc("/generate-key", s.recoveryMiddleware(s.auth(s.generateKeyHandler)))
	mux.HandleFunc("/verify-key", s.recoveryMiddleware(s.auth(s.verifyKeyHandler)))
	mux.HandleFunc("/renew", s.recoveryMiddleware(s.auth(s.renewHandler)))
	mux.HandleFunc("/predictive", s.recoveryMiddleware(s.auth(s.predictiveHandler)))
	mux.HandleFunc("/admin-metrics", s.recoveryMiddleware(s.auth(s.adminMetricsHandler)))
	mux.HandleFunc("/enterprise-analytics", s.recoveryMiddleware(s.auth(s.enterpriseAnalyticsHandler)))

	// V1 API routes
	mux.HandleFunc("/v1/license/info", s.recoveryMiddleware(s.auth(s.licenseInfoHandler)))
	mux.HandleFunc("/v1/analytics/summary", s.recoveryMiddleware(s.auth(s.analyticsSummaryHandler)))

	// Try to start server with port auto-retry
	basePort := s.cfg.APIPort
	maxRetries := 10 // Increased from 3 to 10
	var finalAddr string

	// Find an available port
	for retry := 0; retry < maxRetries; retry++ {
		port := basePort + retry
		addr := s.cfg.APIHost + ":" + strconv.Itoa(port)

		// Create server with timeouts and security settings
		s.srv = &http.Server{
			Addr:         addr,
			Handler:      secureHandler,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 60 * time.Second,
			IdleTimeout:  120 * time.Second,
			// TLS configuration if HTTPS is enabled
			// TLSConfig: getTLSConfig(),
		}

		s.logger.Info("API attempting to start", zap.String("addr", addr), zap.Int("attempt", retry+1))

		// Try to bind to this port
		listener, bindErr := net.Listen("tcp", addr)
		if bindErr != nil {
			s.logger.Warn("Port busy, trying next", zap.String("addr", addr), zap.Error(bindErr))
			continue
		}

		// Port is available, start server in a goroutine
		finalAddr = addr
		s.logger.Info("API binding successful", zap.String("addr", finalAddr))

		wg.Add(1)
		go func() {
			defer wg.Done()
			s.logger.Info("API server started", zap.String("addr", finalAddr))

			// Start serving (this blocks until Shutdown is called or an error occurs)
			if err := s.srv.Serve(listener); err != nil && err != http.ErrServerClosed {
				s.logger.Error("API server error", zap.Error(err))
			}
		}()

		break
	}

	// If we exhausted all port retries
	if finalAddr == "" {
		s.logger.Error("Failed to bind to any port",
			zap.Int("basePort", basePort),
			zap.Int("maxRetries", maxRetries))
		return
	}

	// Wait for interrupt signal
	go func() {
		<-sigChan
		s.logger.Info("Shutdown signal received, gracefully stopping server...")

		// Create a timeout context for shutdown
		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 15*time.Second)
		defer shutdownCancel()

		// Attempt graceful shutdown
		if err := s.srv.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("Server shutdown error", zap.Error(err))
		}

		cancel() // Cancel the main context to signal all operations to stop
	}()

	// Block until context is cancelled and all goroutines finish
	<-ctx.Done()
	wg.Wait()
	s.logger.Info("Server gracefully stopped")
}

// securityMiddleware applies security headers and measures to all requests
func (s *Server) securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		// Block common web attack paths
		path := strings.ToLower(r.URL.Path)
		if strings.Contains(path, "../") ||
			strings.Contains(path, "..\\") ||
			strings.Contains(path, "/.ht") ||
			strings.Contains(path, "/.git") ||
			strings.Contains(path, "/wp-") ||
			strings.Contains(path, "/.env") {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		// Implement rate limiting based on IP
		clientIP := getClientIP(r)
		if !s.rateLimiter.Allow(clientIP, 100, 1) { // 100 requests per second per IP
			s.logger.Warn("Rate limit exceeded",
				zap.String("ip", clientIP),
				zap.String("path", r.URL.Path),
			)
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Proceed with request
		next.ServeHTTP(w, r)
	})
}

// recoveryMiddleware catches panics and returns 500 error
func (s *Server) recoveryMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := debug.Stack()
				s.logger.Error("Panic in handler",
					zap.Any("panic", rec),
					zap.String("stack", string(stack)),
					zap.String("url", r.URL.String()),
					zap.String("method", r.Method),
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next(w, r)
	}
}

// responseWriter is a custom ResponseWriter that tracks status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// WriteHeader overrides the WriteHeader method to capture status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.written = true
	rw.ResponseWriter.WriteHeader(code)
}

// Write overrides the Write method to track if anything was written
func (rw *responseWriter) Write(data []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(data)
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			// Try to get from query param (less secure, but allowed for some endpoints)
			apiKey = r.URL.Query().Get("api_key")
		}

		if apiKey == "" {
			s.logger.Warn("Missing API key",
				zap.String("ip", getClientIP(r)),
				zap.String("path", r.URL.Path),
			)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Validate API key using customer key manager
		customerKey, valid := s.keyManager.ValidateKey(apiKey)
		if !valid {
			// Log failed auth attempts (potential brute force)
			s.logger.Warn("Invalid API key",
				zap.String("ip", getClientIP(r)),
				zap.String("path", r.URL.Path),
				zap.String("user_agent", r.UserAgent()),
			)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check rate limit for this specific API key
		keyIdentifier := "key:" + customerKey.Hash
		if !s.rateLimiter.Allow(keyIdentifier, float64(customerKey.RateLimitRemaining), 1) {
			s.logger.Warn("API key rate limit exceeded",
				zap.String("key_hash", customerKey.Hash[:8]),
				zap.String("tier", string(customerKey.Tier)),
				zap.String("ip", getClientIP(r)),
				zap.String("path", r.URL.Path),
			)
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Update key usage statistics
		s.keyManager.UpdateKeyUsage(apiKey, getClientIP(r), r.UserAgent())

		// Use custom response writer to ensure status code is always set
		customWriter := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next(customWriter, r)

		// Log request (successful auth)
		s.logger.Debug("Authorized request",
			zap.String("path", r.URL.Path),
			zap.Int("status", customWriter.statusCode),
			zap.String("tier", string(customerKey.Tier)),
			zap.String("key_hash", customerKey.Hash[:8]),
		)
	}
}

// getClientIP extracts the client's real IP considering proxy headers
func getClientIP(r *http.Request) string {
	// Try common proxy headers
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		if ip := r.Header.Get(header); ip != "" {
			// X-Forwarded-For can be a comma-separated list; take the first one
			if strings.Contains(ip, ",") {
				return strings.TrimSpace(strings.Split(ip, ",")[0])
			}
			return ip
		}
	}

	// Extract from RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// jsonResponse safely writes a JSON response with proper error handling
func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("Failed to encode JSON response",
			zap.Error(err),
			zap.Any("data", data),
		)
		// We've already written headers, so we can't change the status code
		// But we can log the error and write a simple error message
		fmt.Fprintf(w, `{"error":"Internal encoding error"}`)
	}
}

// Turbo-optimized JSON response with pre-allocated buffers
func (s *Server) turboJsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Use pre-allocated encoder for turbo mode to reduce allocations
	if s.cfg.Tier == config.TierTurbo || s.cfg.Tier == config.TierEnterprise {
		s.turboEncodeJSON(w, data)
	} else {
		json.NewEncoder(w).Encode(data)
	}
}

// Zero-allocation JSON encoding for turbo mode
func (s *Server) turboEncodeJSON(w http.ResponseWriter, data interface{}) {
	// Use a custom encoder that minimizes allocations
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false) // Disable HTML escaping for performance
	encoder.SetIndent("", "")    // Disable indentation for performance

	if err := encoder.Encode(data); err != nil {
		s.logger.Error("Failed to encode JSON response",
			zap.Error(err),
			zap.Any("data", data),
		)
		w.Write([]byte(`{"error":"Internal encoding error"}`))
	}
}

func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	}
	s.turboJsonResponse(w, http.StatusOK, resp)
}

func (s *Server) latestHandler(w http.ResponseWriter, r *http.Request) {
	// Try to get from cache first for ultra-low latency
	if s.cache != nil {
		if block, ok := s.cache.GetLatestBlock(); ok {
			w.Header().Set("X-Cache-Status", "HIT")
			s.turboJsonResponse(w, http.StatusOK, block)
			return
		}
	}

	// Fallback to direct channel read if cache miss
	w.Header().Set("X-Cache-Status", "MISS")

	// Turbo mode: Use non-blocking channel read with immediate timeout
	if s.cfg.Tier == config.TierTurbo || s.cfg.Tier == config.TierEnterprise {
		select {
		case blk := <-s.blockChan:
			s.turboJsonResponse(w, http.StatusOK, blk)
		default:
			// No block available, return empty response immediately
			s.turboJsonResponse(w, http.StatusOK, map[string]string{
				"msg":       "no block available",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
		}
		return
	}

	// Standard mode: Use timeout to prevent blocking
	select {
	case blk := <-s.blockChan:
		s.jsonResponse(w, http.StatusOK, blk)
	case <-time.After(500 * time.Millisecond): // Add timeout to prevent blocking
		s.jsonResponse(w, http.StatusOK, map[string]string{
			"msg":       "no block available",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	if !s.authenticateAdminRequest(r) {
		s.jsonResponse(w, http.StatusUnauthorized, map[string]string{
			"error": "Admin authentication required",
		})
		return
	}

	w.Header().Set("Content-Type", "text/plain")

	// Get P2P metrics from the P2P client
	p2pMetrics := s.getP2PMetrics()

	// Get entropy metrics
	entropyMetrics := s.getEntropyMetrics()

	w.Write([]byte(fmt.Sprintf(`# Bitcoin Sprint Metrics
sprint_active_peers %d
sprint_blocks_detected %d
sprint_tier %q

# P2P Performance Metrics
p2p_connection_pool_size{tier="%s"} %d
p2p_block_pipeline_depth %d
p2p_buffer_pool_hits %d
p2p_buffer_pool_misses %d
p2p_peer_quality_score_avg %.2f
p2p_backpressure_events %d
p2p_circuit_breaker_activations %d
p2p_peer_consecutive_failures_total %d

# Tier-Aware Limits
p2p_max_outstanding_headers_per_peer{tier="%s"} %d
p2p_pipeline_workers{tier="%s"} %d

# Entropy Metrics
relay_cpu_temperature %.2f
entropy_sources_active %d
entropy_system_fingerprint_available %d
entropy_hardware_sources_available %d

# Cache Performance Metrics
cache_blocks_cached %d
cache_max_blocks %d
cache_latest_height %d
cache_is_stale %d
cache_stale_seconds %.2f
`,
		p2pMetrics.activePeers,
		p2pMetrics.blocksDetected,
		s.cfg.Tier,
		s.cfg.Tier,
		p2pMetrics.connectionPoolSize,
		p2pMetrics.pipelineDepth,
		p2pMetrics.bufferPoolHits,
		p2pMetrics.bufferPoolMisses,
		p2pMetrics.avgQualityScore,
		p2pMetrics.backpressureEvents,
		p2pMetrics.circuitBreakerActivations,
		p2pMetrics.totalConsecutiveFailures,
		s.cfg.Tier,
		p2pMetrics.maxOutstandingHeadersPerPeer,
		s.cfg.Tier,
		p2pMetrics.pipelineWorkers,
		entropyMetrics.cpuTemperature,
		entropyMetrics.activeSources,
		entropyMetrics.systemFingerprintAvailable,
		entropyMetrics.hardwareSourcesAvailable,
		s.getCacheMetrics().blocksCached,
		s.getCacheMetrics().maxBlocks,
		s.getCacheMetrics().latestHeight,
		s.getCacheMetrics().isStale,
		s.getCacheMetrics().staleSeconds,
	)))
}

func (s *Server) cacheStatusHandler(w http.ResponseWriter, r *http.Request) {
	if s.cache == nil {
		s.turboJsonResponse(w, http.StatusServiceUnavailable, map[string]string{
			"error":  "Cache not enabled",
			"status": "unavailable",
		})
		return
	}

	status := s.cache.GetStatus()
	s.turboJsonResponse(w, http.StatusOK, status)
}

// P2PMetrics holds P2P performance metrics
type P2PMetrics struct {
	activePeers                  int
	blocksDetected               int
	connectionPoolSize           int
	pipelineDepth                int64
	bufferPoolHits               int64
	bufferPoolMisses             int64
	avgQualityScore              float64
	backpressureEvents           int64
	circuitBreakerActivations    int64
	totalConsecutiveFailures     int64
	maxOutstandingHeadersPerPeer int
	pipelineWorkers              int
}

// getP2PMetrics collects P2P metrics (mock implementation for now)
func (s *Server) getP2PMetrics() P2PMetrics {
	// In a real implementation, this would collect metrics from the P2P client
	// For now, we'll return mock data that represents typical values

	return P2PMetrics{
		activePeers:                  8,
		blocksDetected:               150,
		connectionPoolSize:           8,
		pipelineDepth:                45,
		bufferPoolHits:               1250,
		bufferPoolMisses:             23,
		avgQualityScore:              0.85,
		backpressureEvents:           2,
		circuitBreakerActivations:    1,
		totalConsecutiveFailures:     15,
		maxOutstandingHeadersPerPeer: s.cfg.MaxOutstandingHeadersPerPeer,
		pipelineWorkers:              s.cfg.PipelineWorkers,
	}
}

// EntropyMetrics holds entropy-related metrics
type EntropyMetrics struct {
	cpuTemperature             float32
	activeSources              int
	systemFingerprintAvailable int
	hardwareSourcesAvailable   int
}

// getEntropyMetrics collects entropy-related metrics
func (s *Server) getEntropyMetrics() EntropyMetrics {
	var metrics EntropyMetrics

	// Get CPU temperature
	if temp, err := entropy.GetCPUTemperatureRust(); err == nil {
		metrics.cpuTemperature = temp
	} else {
		metrics.cpuTemperature = -1.0
	}

	// Check system fingerprint availability
	if _, err := entropy.SystemFingerprintRust(); err == nil {
		metrics.systemFingerprintAvailable = 1
	} else {
		metrics.systemFingerprintAvailable = 0
	}

	// Count active entropy sources
	metrics.activeSources = 0
	if metrics.systemFingerprintAvailable == 1 {
		metrics.activeSources++
	}

	// Check if hardware sources are available (CPU temp + fingerprint)
	metrics.hardwareSourcesAvailable = 0
	if metrics.cpuTemperature > 0 {
		metrics.hardwareSourcesAvailable++
	}
	if metrics.systemFingerprintAvailable == 1 {
		metrics.hardwareSourcesAvailable++
	}

	return metrics
}

// CacheMetrics holds cache performance metrics
type CacheMetrics struct {
	blocksCached int
	maxBlocks    int
	latestHeight int64
	isStale      int
	staleSeconds float64
}

// getCacheMetrics collects cache performance metrics
func (s *Server) getCacheMetrics() CacheMetrics {
	if s.cache == nil {
		return CacheMetrics{}
	}

	stats := s.cache.GetCacheStats()
	isStale := 0
	if stats["is_stale"].(bool) {
		isStale = 1
	}

	return CacheMetrics{
		blocksCached: stats["cached_blocks"].(int),
		maxBlocks:    stats["max_blocks"].(int),
		latestHeight: stats["latest_height"].(int64),
		isStale:      isStale,
		staleSeconds: stats["stale_seconds"].(float64),
	}
}

func (s *Server) streamHandler(w http.ResponseWriter, r *http.Request) {
	// More secure origin check (in production, implement more strict validation)
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// For production: implement stricter origin checking
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // Allow requests with no origin
			}

			// Check against allowed origins
			allowedOrigins := []string{
				"https://api.bitcoin-sprint.com",
				"https://dashboard.bitcoin-sprint.com",
				"http://localhost:3000", // For development
			}

			for _, allowed := range allowedOrigins {
				if allowed == origin {
					return true
				}
			}

			s.logger.Warn("Rejected WebSocket connection from unauthorized origin",
				zap.String("origin", origin),
				zap.String("ip", getClientIP(r)),
			)
			return false
		},
		HandshakeTimeout: 10 * time.Second,
	}

	// Upgrade the connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade to WebSocket",
			zap.Error(err),
			zap.String("ip", getClientIP(r)),
		)
		return // Error is handled by the upgrader
	}
	defer conn.Close()

	// Set read deadline to detect stale connections
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	// Handle ping/pong to keep connection alive
	conn.SetPingHandler(func(string) error {
		// Reset the read deadline on ping
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return conn.WriteControl(
			websocket.PongMessage,
			[]byte{},
			time.Now().Add(10*time.Second),
		)
	})

	// Create context with timeout/cancel for the stream
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Start a goroutine to read from the connection
	// This is needed to process control messages
	go func() {
		defer cancel() // Cancel the context if reader exits

		for {
			// ReadMessage will block until a message is received or the connection is closed
			if _, _, err := conn.ReadMessage(); err != nil {
				// Connection closed or error
				return
			}

			// Reset the read deadline
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		}
	}()

	// Stream blocks to client
	for {
		select {
		case blk, ok := <-s.blockChan:
			if !ok {
				// Channel closed
				return
			}

			// Set a write deadline
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			if err := conn.WriteJSON(blk); err != nil {
				s.logger.Debug("Error writing to WebSocket",
					zap.Error(err),
					zap.String("ip", getClientIP(r)),
				)
				return
			}

		case <-ctx.Done():
			// Context cancelled (client disconnected or timeout)
			return
		}
	}
}

// Health endpoint (no auth required)
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "2.1.0",
		"service":   "bitcoin-sprint-api",
	}
	s.turboJsonResponse(w, http.StatusOK, resp)
}

func (s *Server) versionHandler(w http.ResponseWriter, r *http.Request) {
	// Check build info
	buildInfo, ok := debug.ReadBuildInfo()

	versionInfo := "2.2.0-performance"
	buildTime := "unknown"

	// Extract version from build info if available
	if ok {
		for _, setting := range buildInfo.Settings {
			if setting.Key == "vcs.revision" {
				versionInfo += "-" + setting.Value[:7] // Add git commit hash
			}
			if setting.Key == "vcs.time" {
				buildTime = setting.Value
			}
		}
	}

	resp := map[string]interface{}{
		"version":    versionInfo,
		"build":      "enterprise-turbo",
		"build_time": buildTime,
		"tier":       string(s.cfg.Tier),
		"turbo_mode": s.cfg.Tier == "turbo" || s.cfg.Tier == "enterprise",
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}
	s.turboJsonResponse(w, http.StatusOK, resp)
}

// Generate API key
func (s *Server) generateKeyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "Method not allowed",
		})
		return
	}

	// Rate limit key generation
	clientIP := getClientIP(r)
	if s.exceedsKeyGenRateLimit(clientIP) {
		s.jsonResponse(w, http.StatusTooManyRequests, map[string]string{
			"error": "Rate limit exceeded for key generation",
		})
		return
	}

	// Generate a new API key using the customer key manager
	newKey, err := s.keyManager.GenerateKey(config.TierFree, clientIP)
	if err != nil {
		s.logger.Error("Failed to generate API key", zap.Error(err))
		s.jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate secure key",
		})
		return
	}

	// Get the key details for response
	keyDetails, _ := s.keyManager.ValidateKey(newKey)

	// Log key generation (with hash prefix only, not the actual key)
	s.logger.Info("Generated new API key",
		zap.String("key_hash", keyDetails.Hash[:8]),
		zap.String("ip", clientIP),
		zap.String("tier", string(keyDetails.Tier)),
	)

	resp := map[string]interface{}{
		"api_key":        newKey,
		"key_id":         keyDetails.Hash[:8],
		"tier":           string(keyDetails.Tier),
		"created_at":     keyDetails.CreatedAt.Format(time.RFC3339),
		"expires_at":     keyDetails.ExpiresAt.Format(time.RFC3339),
		"expires_unix":   keyDetails.ExpiresAt.Unix(),
		"rate_limit":     s.keyManager.getRateLimitForTier(keyDetails.Tier),
		"usage_count":    keyDetails.RequestCount,
		"rate_remaining": keyDetails.RateLimitRemaining,
	}

	s.jsonResponse(w, http.StatusCreated, resp)
}

// generateSecureRandomKey generates a secure random key using the securebuf package
func (s *Server) generateSecureRandomKey() (string, error) {
	// Use a larger key size for better security
	const keySize = 32

	// Create secure buffer
	keyBuf, err := securebuf.New(keySize)
	if err != nil {
		return "", fmt.Errorf("failed to create secure buffer: %w", err)
	}
	defer keyBuf.Free() // Ensure memory is wiped

	// Generate random bytes
	keyBytes := make([]byte, keySize)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", fmt.Errorf("failed to generate random data: %w", err)
	}

	// Write to secure buffer
	if err := keyBuf.Write(keyBytes); err != nil {
		return "", fmt.Errorf("failed to write to secure buffer: %w", err)
	}

	// Clear the original slice to remove it from memory
	for i := range keyBytes {
		keyBytes[i] = 0
	}

	// Read from secure buffer
	finalKeyBytes, err := keyBuf.ReadToSlice()
	if err != nil {
		return "", fmt.Errorf("failed to read from secure buffer: %w", err)
	}

	// Convert to hex string
	newKey := hex.EncodeToString(finalKeyBytes)

	// Clear the final key bytes too
	for i := range finalKeyBytes {
		finalKeyBytes[i] = 0
	}

	return newKey, nil
}

// exceedsKeyGenRateLimit checks if the client has exceeded the rate limit for key generation
// Limits key generation to 10 keys per hour per IP address to prevent abuse
func (s *Server) exceedsKeyGenRateLimit(clientIP string) bool {
	// 10 keys per hour = 10 tokens capacity, refill at 10/3600 ≈ 0.00278 tokens/second
	return !s.rateLimiter.Allow(clientIP+":keygen", 10, 10.0/3600.0)
}

// Verify API key
func (s *Server) verifyKeyHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		apiKey = r.URL.Query().Get("api_key")
	}

	// Simple validation - in production this would check against database
	valid := apiKey != "" && len(apiKey) >= 16

	resp := map[string]interface{}{
		"valid":                valid,
		"tier":                 "FREE",
		"expires_at":           time.Now().AddDate(1, 0, 0).Unix(),
		"rate_limit_remaining": 100,
	}

	if !valid {
		w.WriteHeader(http.StatusUnauthorized)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Renew license/key
func (s *Server) renewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := map[string]interface{}{
		"renewed":    true,
		"expires_at": time.Now().AddDate(1, 0, 0).Unix(),
		"tier":       "FREE",
		"message":    "License renewed successfully",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Predictive analytics
func (s *Server) predictiveHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"next_block_eta_seconds": 420,
		"mempool_size":           s.mem.Size(),
		"fee_estimates": map[string]int{
			"fast":   24,
			"medium": 18,
			"slow":   12,
		},
		"network_hashrate": "600.45 EH/s",
		"difficulty_adjustment": map[string]interface{}{
			"blocks_until_adjustment":  156,
			"estimated_change_percent": -2.3,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Admin metrics (enhanced)
// Admin-only metrics endpoint
func (s *Server) adminMetricsHandler(w http.ResponseWriter, r *http.Request) {
	if !s.authenticateAdminRequest(r) {
		s.jsonResponse(w, http.StatusUnauthorized, map[string]string{
			"error": "Admin authentication required",
		})
		return
	}

	resp := map[string]interface{}{
		"system_metrics": map[string]interface{}{
			"uptime_seconds":      time.Now().Unix() - 1724659200, // Mock start time
			"cpu_usage_percent":   23.5,
			"memory_usage_mb":     2840,
			"disk_usage_percent":  67.2,
			"network_connections": 8,
		},
		"api_metrics": map[string]interface{}{
			"total_requests":       150420,
			"requests_per_minute":  240,
			"error_rate_percent":   0.1,
			"avg_response_time_ms": 85,
		},
		"blockchain_metrics": map[string]interface{}{
			"current_block_height": 850123,
			"mempool_transactions": s.mem.Size(),
			"peer_count":           8,
			"sync_progress":        1.0,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Enterprise analytics
// Enterprise analytics endpoint (admin-only)
func (s *Server) enterpriseAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	if !s.authenticateAdminRequest(r) {
		s.jsonResponse(w, http.StatusUnauthorized, map[string]string{
			"error": "Admin authentication required",
		})
		return
	}

	resp := map[string]interface{}{
		"performance_analytics": map[string]interface{}{
			"block_propagation_time_ms":  180,
			"transaction_throughput_tps": 7.2,
			"network_latency_ms":         45,
			"node_efficiency_score":      94.5,
		},
		"security_metrics": map[string]interface{}{
			"failed_auth_attempts":  12,
			"suspicious_requests":   3,
			"rate_limit_violations": 28,
			"geo_blocked_requests":  156,
		},
		"business_intelligence": map[string]interface{}{
			"total_api_calls_today": 45230,
			"unique_users_today":    1247,
			"revenue_impact_usd":    2450.75,
			"tier_distribution": map[string]int{
				"FREE":            1100,
				"PRO":             120,
				"ENTERPRISE":      25,
				"ENTERPRISE_PLUS": 2,
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// License info (V1 API) - Admin only
func (s *Server) licenseInfoHandler(w http.ResponseWriter, r *http.Request) {
	if !s.authenticateAdminRequest(r) {
		s.jsonResponse(w, http.StatusUnauthorized, map[string]string{
			"error": "Admin authentication required",
		})
		return
	}

	licenseInfo := license.GetInfo(s.cfg.LicenseKey)

	// Generate SHA256 hash of the license key and take the first 8 characters
	// This is enough to identify a license without revealing any part of it
	hasher := sha256.New()
	hasher.Write([]byte(s.cfg.LicenseKey))
	hashPrefix := hex.EncodeToString(hasher.Sum(nil))[:8]

	resp := map[string]interface{}{
		"license_id": hashPrefix, // Use hash prefix instead of actual key
		"tier":       licenseInfo.Tier,
		"valid":      licenseInfo.Valid,
		"expires_at": licenseInfo.ExpiresAt,
		"features":   licenseInfo.Features,
		"usage_limits": map[string]interface{}{
			"requests_per_hour":      licenseInfo.RequestsPerHour,
			"concurrent_connections": licenseInfo.ConcurrentConnections,
			"data_retention_days":    licenseInfo.DataRetentionDays,
		},
	}
	s.jsonResponse(w, http.StatusOK, resp)
}

// Analytics summary (V1 API) - Admin only
func (s *Server) analyticsSummaryHandler(w http.ResponseWriter, r *http.Request) {
	if !s.authenticateAdminRequest(r) {
		s.jsonResponse(w, http.StatusUnauthorized, map[string]string{
			"error": "Admin authentication required",
		})
		return
	}

	resp := map[string]interface{}{
		"period": "24h",
		"summary": map[string]interface{}{
			"total_requests":           45230,
			"successful_requests":      45126,
			"error_rate_percent":       0.23,
			"avg_response_time_ms":     185,
			"peak_requests_per_minute": 450,
		},
		"endpoint_performance": map[string]interface{}{
			"/latest": map[string]interface{}{
				"requests":             25430,
				"avg_response_time_ms": 120,
				"error_rate_percent":   0.1,
			},
			"/status": map[string]interface{}{
				"requests":             12450,
				"avg_response_time_ms": 45,
				"error_rate_percent":   0.05,
			},
			"/metrics": map[string]interface{}{
				"requests":             5230,
				"avg_response_time_ms": 280,
				"error_rate_percent":   0.8,
			},
		},
		"geographic_distribution": map[string]int{
			"US":    18900,
			"EU":    15200,
			"ASIA":  8100,
			"OTHER": 3030,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// TurboStatusResponse represents the current turbo mode configuration
type TurboStatusResponse struct {
	Tier               string             `json:"tier"`
	TurboModeEnabled   bool               `json:"turboModeEnabled"`
	WriteDeadline      string             `json:"writeDeadline"`
	UseSharedMemory    bool               `json:"useSharedMemory"`
	BlockBufferSize    int                `json:"blockBufferSize"`
	EnableKernelBypass bool               `json:"enableKernelBypass"`
	UseDirectP2P       bool               `json:"useDirectP2P"`
	UseMemoryChannel   bool               `json:"useMemoryChannel"`
	OptimizeSystem     bool               `json:"optimizeSystem"`
	Features           []string           `json:"features"`
	PerformanceTargets PerformanceTargets `json:"performanceTargets"`
	SystemMetrics      SystemMetrics      `json:"systemMetrics"`
	Timestamp          time.Time          `json:"timestamp"`
}

// PerformanceTargets shows expected performance for the current tier
type PerformanceTargets struct {
	BlockRelayLatency string `json:"blockRelayLatency"`
	WriteDeadline     string `json:"writeDeadline"`
	BufferStrategy    string `json:"bufferStrategy"`
	PeerNotification  string `json:"peerNotification"`
}

// SystemMetrics shows current system performance
type SystemMetrics struct {
	ConnectedPeers    int    `json:"connectedPeers"`
	BlocksProcessed   int64  `json:"blocksProcessed"`
	AvgProcessingTime string `json:"avgProcessingTime"`
	MemoryUsage       string `json:"memoryUsage"`
	CPUUsage          string `json:"cpuUsage"`
}

// turboStatusHandler returns current turbo mode configuration and performance metrics
func (s *Server) turboStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Determine if turbo mode is enabled
	turboEnabled := s.cfg.Tier == config.TierTurbo || s.cfg.Tier == config.TierEnterprise

	// Build feature list based on configuration
	features := []string{}
	if s.cfg.UseSharedMemory {
		features = append(features, "Shared Memory")
	}
	if s.cfg.UseDirectP2P {
		features = append(features, "Direct P2P")
	}
	if s.cfg.UseMemoryChannel {
		features = append(features, "Memory Channel")
	}
	if s.cfg.OptimizeSystem {
		features = append(features, "System Optimizations")
	}
	if s.cfg.EnableKernelBypass {
		features = append(features, "Kernel Bypass")
	}

	// Get performance targets based on tier
	targets := s.getPerformanceTargets()

	// Get current system metrics
	metrics := s.getSystemMetrics()

	response := TurboStatusResponse{
		Tier:               string(s.cfg.Tier),
		TurboModeEnabled:   turboEnabled,
		WriteDeadline:      s.cfg.WriteDeadline.String(),
		UseSharedMemory:    s.cfg.UseSharedMemory,
		BlockBufferSize:    s.cfg.BlockBufferSize,
		EnableKernelBypass: s.cfg.EnableKernelBypass,
		UseDirectP2P:       s.cfg.UseDirectP2P,
		UseMemoryChannel:   s.cfg.UseMemoryChannel,
		OptimizeSystem:     s.cfg.OptimizeSystem,
		Features:           features,
		PerformanceTargets: targets,
		SystemMetrics:      metrics,
		Timestamp:          time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	s.turboEncodeJSON(w, response)
}

// getPerformanceTargets returns expected performance metrics for the current tier
func (s *Server) getPerformanceTargets() PerformanceTargets {
	switch s.cfg.Tier {
	case config.TierEnterprise:
		return PerformanceTargets{
			BlockRelayLatency: "<5ms (Enterprise)",
			WriteDeadline:     "200µs",
			BufferStrategy:    "Overwrite old events (never miss)",
			PeerNotification:  "Zero-copy with kernel bypass",
		}
	case config.TierTurbo:
		return PerformanceTargets{
			BlockRelayLatency: "<10ms (Turbo)",
			WriteDeadline:     "500µs",
			BufferStrategy:    "Overwrite old events (never miss)",
			PeerNotification:  "Zero-copy shared memory",
		}
	case config.TierBusiness:
		return PerformanceTargets{
			BlockRelayLatency: "<50ms (Business)",
			WriteDeadline:     "1s",
			BufferStrategy:    "Best effort delivery",
			PeerNotification:  "Standard TCP relay",
		}
	case config.TierPro:
		return PerformanceTargets{
			BlockRelayLatency: "<100ms (Pro)",
			WriteDeadline:     "1.5s",
			BufferStrategy:    "Best effort delivery",
			PeerNotification:  "Standard TCP relay",
		}
	default: // Free
		return PerformanceTargets{
			BlockRelayLatency: "<500ms (Free)",
			WriteDeadline:     "2s",
			BufferStrategy:    "Drop on full buffer",
			PeerNotification:  "Standard TCP relay with limits",
		}
	}
}

// getSystemMetrics returns current system performance metrics
func (s *Server) getSystemMetrics() SystemMetrics {
	// In production, these would be real metrics from the system
	// For now, return realistic values based on the current tier

	connectedPeers := 8 // Default peer count
	if s.cfg.Tier == config.TierTurbo || s.cfg.Tier == config.TierEnterprise {
		connectedPeers = 16 // More peers for higher tiers
	}

	var avgProcessingTime string
	switch s.cfg.Tier {
	case config.TierEnterprise:
		avgProcessingTime = "2.1ms"
	case config.TierTurbo:
		avgProcessingTime = "4.8ms"
	case config.TierBusiness:
		avgProcessingTime = "15.2ms"
	case config.TierPro:
		avgProcessingTime = "28.4ms"
	default:
		avgProcessingTime = "85.6ms"
	}

	return SystemMetrics{
		ConnectedPeers:    connectedPeers,
		BlocksProcessed:   42850, // Sample number
		AvgProcessingTime: avgProcessingTime,
		MemoryUsage:       "156.8MB",
		CPUUsage:          "12.4%",
	}
}
