// Package api provides HTTP handlers for the API server
package api

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// ===== HTTP HANDLERS =====

// latestHandler handles requests for the latest block
func (s *Server) latestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "Method not allowed",
		})
		return
	}

	// Get the latest block from the backend
	backend, exists := s.backends.Get("bitcoin")
	if !exists {
		s.jsonResponse(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Bitcoin backend not available",
		})
		return
	}

	block, err := backend.GetLatestBlock()
	if err != nil {
		s.logger.Error("Failed to get latest block", zap.Error(err))
		s.jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to get latest block",
		})
		return
	}

	s.turboJsonResponse(w, http.StatusOK, block)
}

// streamHandler handles WebSocket streaming of blocks
func (s *Server) streamHandler(w http.ResponseWriter, r *http.Request) {
	// Acquire WebSocket connection slot
	clientIP := getClientIP(r)
	if !s.wsLimiter.Acquire(clientIP) {
		http.Error(w, "WebSocket connection limit reached", http.StatusTooManyRequests)
		return
	}
	defer s.wsLimiter.Release(clientIP)

	// Get the backend for streaming
	backend, exists := s.backends.Get("bitcoin")
	if !exists {
		http.Error(w, "Bitcoin backend not available", http.StatusServiceUnavailable)
		return
	}

	// WebSocket upgrade
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
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
	conn.SetReadDeadline(s.clock.Now().Add(60 * time.Second))

	// Handle ping/pong to keep connection alive
	conn.SetPingHandler(func(string) error {
		// Reset the read deadline on ping
		conn.SetReadDeadline(s.clock.Now().Add(60 * time.Second))
		return conn.WriteControl(
			websocket.PongMessage,
			[]byte{},
			s.clock.Now().Add(10*time.Second),
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
			conn.SetReadDeadline(s.clock.Now().Add(60 * time.Second))
		}
	}()

	// Create a channel for streaming blocks from the backend
	blockChan := make(chan blocks.BlockEvent, 10)

	// Start streaming from the backend
	go backend.StreamBlocks(ctx, blockChan)

	// Stream blocks to client
	for {
		select {
		case blk, ok := <-blockChan:
			if !ok {
				// Channel closed
				return
			}

			// Set a write deadline
			conn.SetWriteDeadline(s.clock.Now().Add(10 * time.Second))

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

// healthHandler handles health check requests
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"status":    "healthy",
		"timestamp": s.clock.Now().UTC().Format(time.RFC3339),
		"version":   "2.1.0",
		"service":   "bitcoin-sprint-api",
	}
	s.turboJsonResponse(w, http.StatusOK, resp)
}

// versionHandler handles version information requests
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
		"timestamp":  s.clock.Now().UTC().Format(time.RFC3339),
	}
	s.turboJsonResponse(w, http.StatusOK, resp)
}

// generateKeyHandler handles API key generation requests
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

// statusHandler handles status information requests
func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "Method not allowed",
		})
		return
	}

	// Get status from all backends
	status := s.backends.GetStatus()

	// Add server-specific status information
	status["server"] = map[string]interface{}{
		"uptime":      "unknown", // Would need to track this
		"connections": "unknown", // Would need to track this
		"version":     "2.2.0-performance",
		"tier":        string(s.cfg.Tier),
		"turbo_mode":  s.cfg.Tier == "turbo" || s.cfg.Tier == "enterprise",
	}

	s.jsonResponse(w, http.StatusOK, status)
}

// mempoolHandler handles mempool information requests
func (s *Server) mempoolHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "Method not allowed",
		})
		return
	}

	// Get mempool size from backend
	backend, exists := s.backends.Get("bitcoin")
	if !exists {
		s.jsonResponse(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Bitcoin backend not available",
		})
		return
	}

	mempoolSize := backend.GetMempoolSize()

	resp := map[string]interface{}{
		"size":      mempoolSize,
		"timestamp": s.clock.Now().UTC().Format(time.RFC3339),
	}

	s.jsonResponse(w, http.StatusOK, resp)
}

// analyticsSummaryHandler handles analytics summary requests
func (s *Server) analyticsSummaryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "Method not allowed",
		})
		return
	}

	// Get analytics data
	summary := s.predictor.GetAnalyticsSummary()

	resp := map[string]interface{}{
		"analytics": summary,
		"timestamp": s.clock.Now().UTC().Format(time.RFC3339),
	}

	s.jsonResponse(w, http.StatusOK, resp)
}

// licenseInfoHandler handles license information requests
func (s *Server) licenseInfoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "Method not allowed",
		})
		return
	}

	// This would integrate with the license system
	resp := map[string]interface{}{
		"tier":      string(s.cfg.Tier),
		"features":  []string{"basic", "standard"}, // Would be dynamic
		"valid":     true,
		"timestamp": s.clock.Now().UTC().Format(time.RFC3339),
	}

	s.jsonResponse(w, http.StatusOK, resp)
}

// chainsHandler returns information about all registered blockchain backends
func (s *Server) chainsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	chains := s.backends.List()
	status := s.backends.GetStatus()

	response := map[string]interface{}{
		"chains":       chains,
		"status":       status,
		"total_chains": len(chains),
		"timestamp":    s.clock.Now().UTC().Format(time.RFC3339),
	}

	s.jsonResponse(w, http.StatusOK, response)
}

// chainAwareHandler routes requests to the appropriate chain backend based on URL path
func (s *Server) chainAwareHandler(w http.ResponseWriter, r *http.Request) {
	// Parse path: /v1/{chain}/{endpoint}
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid path format. Use /v1/{chain}/{endpoint}", http.StatusBadRequest)
		return
	}

	chain := pathParts[1]
	endpoint := pathParts[2]

	// Get the backend for this chain
	backend, exists := s.backends.Get(chain)
	if !exists {
		http.Error(w, fmt.Sprintf("Chain '%s' not supported", chain), http.StatusNotFound)
		return
	}

	// Route to appropriate handler based on endpoint
	switch endpoint {
	case "latest":
		s.chainLatestHandler(backend, w, r)
	case "status":
		s.chainStatusHandler(backend, w, r)
	case "stream":
		s.chainStreamHandler(backend, w, r)
	case "metrics":
		s.chainMetricsHandler(backend, w, r)
	default:
		http.Error(w, fmt.Sprintf("Unknown endpoint '%s'", endpoint), http.StatusNotFound)
	}
}

// chainLatestHandler handles /v1/{chain}/latest requests
func (s *Server) chainLatestHandler(backend ChainBackend, w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	block, err := backend.GetLatestBlock()
	if err != nil {
		s.logger.Error("Failed to get latest block",
			zap.String("chain", "unknown"),
			zap.Error(err))
		http.Error(w, "Failed to get latest block", http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusOK, block)
}

// chainStatusHandler handles /v1/{chain}/status requests
func (s *Server) chainStatusHandler(backend ChainBackend, w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := backend.GetStatus()
	s.jsonResponse(w, http.StatusOK, status)
}

// chainMetricsHandler handles /v1/{chain}/metrics requests
func (s *Server) chainMetricsHandler(backend ChainBackend, w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := map[string]interface{}{
		"mempool_size":   backend.GetMempoolSize(),
		"predictive_eta": backend.GetPredictiveETA(),
		"timestamp":      s.clock.Now().UTC().Format(time.RFC3339),
	}

	s.jsonResponse(w, http.StatusOK, metrics)
}

// chainStreamHandler handles /v1/{chain}/stream requests
func (s *Server) chainStreamHandler(backend ChainBackend, w http.ResponseWriter, r *http.Request) {
	// Extract chain from URL path for quota management
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	chain := pathParts[1] // Already validated in chainAwareHandler

	// Acquire WebSocket connection for specific chain
	clientIP := getClientIP(r)
	if !s.wsLimiter.AcquireForChain(clientIP, chain) {
		http.Error(w, fmt.Sprintf("WebSocket connection limit reached for %s chain", chain), http.StatusTooManyRequests)
		return
	}
	defer s.wsLimiter.ReleaseForChain(clientIP, chain)

	// WebSocket upgrade logic (similar to existing streamHandler)
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			allowedOrigins := []string{
				"https://api.bitcoin-sprint.com",
				"https://dashboard.bitcoin-sprint.com",
				"http://localhost:3000",
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

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade to WebSocket", zap.Error(err))
		return
	}
	defer conn.Close()

	conn.SetReadDeadline(s.clock.Now().Add(60 * time.Second))

	conn.SetPingHandler(func(string) error {
		conn.SetReadDeadline(s.clock.Now().Add(60 * time.Second))
		return conn.WriteControl(websocket.PongMessage, []byte{}, s.clock.Now().Add(10*time.Second))
	})

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Start reader goroutine
	go func() {
		defer cancel()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
			conn.SetReadDeadline(s.clock.Now().Add(60 * time.Second))
		}
	}()

	// Stream blocks from the specific chain backend
	blockChan := make(chan blocks.BlockEvent, 100)
	go backend.StreamBlocks(ctx, blockChan)

	for {
		select {
		case <-ctx.Done():
			return
		case blk := <-blockChan:
			conn.SetWriteDeadline(s.clock.Now().Add(10 * time.Second))
			if err := conn.WriteJSON(blk); err != nil {
				s.logger.Debug("Error writing to WebSocket", zap.Error(err))
				return
			}
		}
	}
}
