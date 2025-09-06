// Package api provides the main HTTP API server for Bitcoin Sprint
package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/cache"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
	"github.com/PayRpc/Bitcoin-Sprint/internal/relay"
	"go.uber.org/zap"
)

// ===== SERVER STRUCT AND LIFECYCLE =====

// Server represents the main API server with all dependencies
type Server struct {
	cfg       config.Config
	blockChan chan blocks.BlockEvent
	mem       *mempool.Mempool
	cache     *cache.Cache
	logger    *zap.Logger
	srv       *http.Server // Public API server
	adminSrv  *http.Server // Admin-only server
	startTime time.Time    // Server start time for uptime tracking

	// Rate limiting
	rateLimiter *RateLimiter

	// HTTP router
	httpMux *http.ServeMux

	// Customer key management
	keyManager        *CustomerKeyManager
	adminAuth         *AdminAuth
	wsLimiter         *WebSocketLimiter
	predictor         *PredictiveAnalytics
	circuitBreaker    *CircuitBreaker
	backends          *BackendRegistry
	ethereumRelay     *relay.EthereumRelay
	solanaRelay       *relay.SolanaRelay
	clock             Clock
	randReader        RandomReader
	enterpriseManager *EnterpriseSecurityManager
}

// New creates a new API server instance
func New(cfg config.Config, blockChan chan blocks.BlockEvent, mem *mempool.Mempool, logger *zap.Logger) *Server {
	clock := RealClock{}
	randReader := RealRandomReader{}

	server := &Server{
		cfg:               cfg,
		blockChan:         blockChan,
		mem:               mem,
		logger:            logger,
		rateLimiter:       NewRateLimiter(clock),
		keyManager:        NewCustomerKeyManagerWithConfig(cfg, clock, randReader),
		adminAuth:         NewAdminAuth(),
		wsLimiter:         NewWebSocketLimiter(cfg.WebSocketMaxGlobal, cfg.WebSocketMaxPerIP, cfg.WebSocketMaxPerChain),
		predictor:         NewPredictiveAnalytics(clock),
		circuitBreaker:    NewCircuitBreaker(cfg.Tier, clock),
		backends:          NewBackendRegistry(),
		httpMux:           http.NewServeMux(), // Initialize HTTP mux
		ethereumRelay:     relay.NewEthereumRelay(cfg, logger),
		solanaRelay:       relay.NewSolanaRelay(cfg, logger),
		clock:             clock,
		randReader:        randReader,
		enterpriseManager: nil, // Will be initialized in Run()
	}

	// Initialize default Bitcoin backend
	btcBackend := &BitcoinBackend{
		blockChan: blockChan,
		mem:       mem,
		cfg:       cfg,
	}
	server.backends.Register("btc", btcBackend)
	server.backends.Register("bitcoin", btcBackend) // alias for handlers

	return server
}

// NewWithCache creates a new API server instance with cache support
func NewWithCache(cfg config.Config, blockChan chan blocks.BlockEvent, mem *mempool.Mempool, cache *cache.Cache, logger *zap.Logger) *Server {
	clock := RealClock{}
	randReader := RealRandomReader{}

	server := &Server{
		cfg:               cfg,
		blockChan:         blockChan,
		mem:               mem,
		cache:             cache,
		logger:            logger,
		rateLimiter:       NewRateLimiter(clock),
		keyManager:        NewCustomerKeyManagerWithConfig(cfg, clock, randReader),
		adminAuth:         NewAdminAuth(),
		wsLimiter:         NewWebSocketLimiter(cfg.WebSocketMaxGlobal, cfg.WebSocketMaxPerIP, cfg.WebSocketMaxPerChain),
		predictor:         NewPredictiveAnalytics(clock),
		circuitBreaker:    NewCircuitBreaker(cfg.Tier, clock),
		backends:          NewBackendRegistry(),
		httpMux:           http.NewServeMux(), // Initialize HTTP mux
		ethereumRelay:     relay.NewEthereumRelay(cfg, logger),
		solanaRelay:       relay.NewSolanaRelay(cfg, logger),
		clock:             clock,
		randReader:        randReader,
		enterpriseManager: nil, // Will be initialized in Run()
	}

	// Initialize default Bitcoin backend
	btcBackend := &BitcoinBackend{
		blockChan: blockChan,
		mem:       mem,
		cfg:       cfg,
		cache:     cache,
	}
	server.backends.Register("btc", btcBackend)
	server.backends.Register("bitcoin", btcBackend)

	return server
}

// Stop gracefully shuts down the server
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

// ===== CUSTOMER KEY MANAGER IMPLEMENTATION =====

// CustomerKeyManager manages customer API keys and their associated tiers
type CustomerKeyManager struct {
	keys       map[string]CustomerKey // SHA256 hash -> key info
	keyHashes  map[string]string      // Original key -> hash mapping
	cfg        config.Config          // Configuration for rate limits
	mu         sync.RWMutex
	clock      Clock
	randReader RandomReader
}

// CustomerKey represents a customer's API key information
type CustomerKey struct {
	Hash               string      `json:"hash"`
	Tier               config.Tier `json:"tier"`
	CreatedAt          time.Time   `json:"created_at"`
	ExpiresAt          time.Time   `json:"expires_at"`
	LastUsed           time.Time   `json:"last_used"`
	RequestCount       int64       `json:"request_count"`
	RateLimitRemaining int         `json:"rate_limit_remaining"`
	ClientIP           string      `json:"client_ip"`
	UserAgent          string      `json:"user_agent"`
}

// NewCustomerKeyManager creates a new customer key manager
func NewCustomerKeyManager(clock Clock, randReader RandomReader) *CustomerKeyManager {
	manager := &CustomerKeyManager{
		keys:       make(map[string]CustomerKey),
		keyHashes:  make(map[string]string),
		cfg:        config.Config{}, // Default config
		clock:      clock,
		randReader: randReader,
	}

	// Initialize with default key for backward compatibility
	defaultKey := "changeme"
	hash := manager.hashKey(defaultKey)
	manager.keys[hash] = CustomerKey{
		Hash:               hash,
		Tier:               config.TierFree,
		CreatedAt:          manager.clock.Now(),
		ExpiresAt:          manager.clock.Now().AddDate(1, 0, 0),
		LastUsed:           manager.clock.Now(),
		RequestCount:       0,
		RateLimitRemaining: 100,
		ClientIP:           "",
		UserAgent:          "",
	}
	manager.keyHashes[defaultKey] = hash

	return manager
}

// NewCustomerKeyManagerWithConfig creates a new customer key manager with config
func NewCustomerKeyManagerWithConfig(cfg config.Config, clock Clock, randReader RandomReader) *CustomerKeyManager {
	manager := &CustomerKeyManager{
		keys:       make(map[string]CustomerKey),
		keyHashes:  make(map[string]string),
		cfg:        cfg,
		clock:      clock,
		randReader: randReader,
	}

	// Initialize with default key for backward compatibility
	defaultKey := "changeme"
	hash := manager.hashKey(defaultKey)
	manager.keys[hash] = CustomerKey{
		Hash:               hash,
		Tier:               cfg.Tier,
		CreatedAt:          manager.clock.Now(),
		ExpiresAt:          manager.clock.Now().AddDate(1, 0, 0),
		LastUsed:           manager.clock.Now(),
		RequestCount:       0,
		RateLimitRemaining: cfg.RateLimits[cfg.Tier].RequestsPerHour,
		ClientIP:           "",
		UserAgent:          "",
	}
	manager.keyHashes[defaultKey] = hash

	// Load API keys from shared data file (created by web frontend)
	manager.loadSharedApiKeys()

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
	if ckm.clock.Now().After(customerKey.ExpiresAt) {
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
		customerKey.LastUsed = ckm.clock.Now()
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
	if _, err := ckm.randReader.Read(keyBytes); err != nil {
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
		CreatedAt:          ckm.clock.Now(),
		ExpiresAt:          ckm.clock.Now().AddDate(1, 0, 0),
		LastUsed:           ckm.clock.Now(),
		RequestCount:       0,
		RateLimitRemaining: ckm.getRateLimitForTier(tier),
		ClientIP:           clientIP,
		UserAgent:          "",
	}

	return newKey, nil
}

// getRateLimitForTier returns the rate limit for a given tier
func (ckm *CustomerKeyManager) getRateLimitForTier(tier config.Tier) int {
	if rateLimit, exists := ckm.cfg.RateLimits[tier]; exists {
		return rateLimit.RequestsPerHour
	}

	// Fallback to default values if config not available
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

// loadSharedApiKeys loads API keys from shared data file created by web frontend
func (ckm *CustomerKeyManager) loadSharedApiKeys() {
	dataPath := filepath.Join("data", "api_keys.json")
	
	// Check if file exists
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		// File doesn't exist, that's ok - no keys to load
		return
	}

	// Read the file
	data, err := os.ReadFile(dataPath)
	if err != nil {
		// Log error but don't fail startup
		return
	}

	// Parse JSON
	var sharedKeys []map[string]interface{}
	if err := json.Unmarshal(data, &sharedKeys); err != nil {
		// Log error but don't fail startup
		return
	}

	// Process each key
	for _, keyData := range sharedKeys {
		key, ok := keyData["key"].(string)
		if !ok {
			continue
		}

		tierStr, ok := keyData["tier"].(string)
		if !ok {
			continue
		}

		// Convert tier string to config.Tier
		var tier config.Tier
		switch tierStr {
		case "FREE":
			tier = config.TierFree
		case "PRO":
			tier = config.TierPro
		case "BUSINESS":
			tier = config.TierBusiness
		case "TURBO":
			tier = config.TierTurbo
		case "ENTERPRISE":
			tier = config.TierEnterprise
		case "ENTERPRISE_PLUS":
			tier = config.TierEnterprise // Map to highest tier
		default:
			tier = config.TierFree
		}

		// Parse dates
		createdAtStr, _ := keyData["created_at"].(string)
		expiresAtStr, _ := keyData["expires_at"].(string)
		
		createdAt := ckm.clock.Now()
		expiresAt := ckm.clock.Now().AddDate(1, 0, 0)
		
		if createdAtStr != "" {
			if parsed, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
				createdAt = parsed
			}
		}
		
		if expiresAtStr != "" {
			if parsed, err := time.Parse(time.RFC3339, expiresAtStr); err == nil {
				expiresAt = parsed
			}
		}

		// Parse usage stats
		requests := int64(0)
		if reqFloat, ok := keyData["requests"].(float64); ok {
			requests = int64(reqFloat)
		}

		// Check if key is revoked
		revoked := false
		if revokedVal, ok := keyData["revoked"].(bool); ok {
			revoked = revokedVal
		}

		// Skip revoked or expired keys
		if revoked || ckm.clock.Now().After(expiresAt) {
			continue
		}

		// Create customer key entry
		hash := ckm.hashKey(key)
		ckm.keyHashes[key] = hash
		ckm.keys[hash] = CustomerKey{
			Hash:               hash,
			Tier:               tier,
			CreatedAt:          createdAt,
			ExpiresAt:          expiresAt,
			LastUsed:           createdAt, // Initialize with creation time
			RequestCount:       requests,
			RateLimitRemaining: ckm.getRateLimitForTier(tier),
			ClientIP:           "",
			UserAgent:          "",
		}
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

// AddAdminKey adds a new admin key
func (aa *AdminAuth) AddAdminKey(key string) {
	hasher := sha256.New()
	hasher.Write([]byte(key))
	hash := hex.EncodeToString(hasher.Sum(nil))

	aa.mu.Lock()
	defer aa.mu.Unlock()
	aa.adminKeys[hash] = true
}

// RemoveAdminKey removes an admin key

// AdminAuth provides admin key-based authentication
type AdminAuth struct {
	adminKeys map[string]bool
	mu        sync.RWMutex
}

func (aa *AdminAuth) RemoveAdminKey(key string) {
	hasher := sha256.New()
	hasher.Write([]byte(key))
	hash := hex.EncodeToString(hasher.Sum(nil))

	aa.mu.Lock()
	defer aa.mu.Unlock()
	delete(aa.adminKeys, hash)
}
