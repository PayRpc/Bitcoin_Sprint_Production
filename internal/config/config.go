package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// TierRateLimit defines rate limits for each tier
type TierRateLimit struct {
	RequestsPerSecond    int     `json:"requests_per_second"`
	RequestsPerHour      int     `json:"requests_per_hour"`
	ConcurrentStreams    int     `json:"concurrent_streams"`
	DataSizeLimitMB      int     `json:"data_size_limit_mb"`
	KeyGenerationPerHour int     `json:"key_generation_per_hour"`
	WebSocketMessageRate int     `json:"websocket_message_rate"`
	RefillRate           float64 `json:"refill_rate"` // tokens per second
	BurstCapacity        int     `json:"burst_capacity"`
}

// Tier represents the performance tier for the application
type Tier string

const (
	TierFree       Tier = "free"
	TierPro        Tier = "pro"
	TierBusiness   Tier = "business"
	TierTurbo      Tier = "turbo"
	TierEnterprise Tier = "enterprise"
)

// Config holds runtime configuration
type Config struct {
	BitcoinNodes             []string
	ZMQNodes                 []string
	PeerListenPort           int
	APIHost                  string
	APIPort                  int
	AdminPort                int // Separate port for admin endpoints
	LicenseKey               string
	APIKey                   string
	SecureChannelURL         string
	UseDirectP2P             bool
	UseMemoryChannel         bool
	OptimizeSystem           bool
	NodeID                   string        // Unique identifier for this node
	RequireDatabase          bool          // Whether database is required
	BlockChannelBuffer       int           // Size of block channel buffer
	BlockDeduplicationWindow time.Duration // Time window for deduplication
	CacheSize                int           // Size of cache in entries
	MempoolMaxSize           int           // Maximum size of mempool in entries

	// Performance optimizations (permanent for 99.9% SLA compliance)
	GCPercent       int  // Aggressive GC tuning (default: 25)
	MaxCPUCores     int  // GOMAXPROCS setting (0 = auto-detect)
	HighPriority    bool // Use high process priority
	LockOSThread    bool // Pin main thread to OS thread
	PreallocBuffers bool // Pre-allocate memory buffers

	// Tier-aware P2P settings
	Tier               Tier
	WriteDeadline      time.Duration
	UseSharedMemory    bool
	BlockBufferSize    int
	EnableKernelBypass bool
	MockFastBlocks     bool // Enable fast block simulation for testing/demo

	// Tier-aware limits
	MaxOutstandingHeadersPerPeer int // Maximum headers per peer
	PipelineWorkers              int // Number of pipeline workers

	// Enterprise-ready rate limiting (per-tier tunable)
	RateLimits map[Tier]TierRateLimit

	// Blockchain-agnostic settings
	SupportedChains []string // List of supported blockchains
	DefaultChain    string   // Default blockchain (btc, eth, sol, etc.)

	// Security settings
	EnablePrometheus     bool          // Enable Prometheus metrics endpoint
	PrometheusPort       int           // Separate port for Prometheus metrics
	EnableTLS            bool          // Enable TLS for admin endpoints
	EnableMTLS           bool          // Enable mTLS for internal metrics
	IdleTimeout          time.Duration // WebSocket idle timeout
	MessageRateLimit     int           // WebSocket messages per second per client
	GeneralRateLimit     int           // General IP-based rate limit (requests per second)
	WebSocketMaxGlobal   int           // Maximum global WebSocket connections
	WebSocketMaxPerIP    int           // Maximum WebSocket connections per IP
	WebSocketMaxPerChain int           // Maximum WebSocket connections per chain

	// Persistence settings
	DatabaseType      string // sqlite, postgres, redis
	DatabaseURL       string // Connection string
	EnablePersistence bool   // Enable key persistence

	// Sprint relay peer settings
	SprintRelayPeers []string // List of Sprint relay peers requiring authentication

	// RPC Configuration (for backfill and batch operations)
	RPCEnabled       bool          `json:"rpc_enabled"`
	RPCURL           string        `json:"rpc_url"`
	RPCUsername      string        `json:"rpc_username"`
	RPCPassword      string        `json:"rpc_password"`
	RPCTimeout       time.Duration `json:"rpc_timeout"`
	RPCBatchSize     int           `json:"rpc_batch_size"`
	RPCRetryAttempts int           `json:"rpc_retry_attempts"`
	RPCRetryMaxWait  time.Duration `json:"rpc_retry_max_wait"`
	RPCSkipMempool   bool          `json:"rpc_skip_mempool"`
	RPCFailedTxFile  string        `json:"rpc_failed_tx_file"`
	RPCLastIDFile    string        `json:"rpc_last_id_file"`
	RPCWorkers       int           `json:"rpc_workers"`
	RPCMessageTopic  string        `json:"rpc_message_topic"`

	// API timeouts
	APIReadTimeout  time.Duration `json:"api_read_timeout"`
	APIWriteTimeout time.Duration `json:"api_write_timeout"`
	APIIdleTimeout  time.Duration `json:"api_idle_timeout"`

	// P2P configuration
	P2PListenAddress   string        `json:"p2p_listen_address"`
	P2PBootstrapPeers  []string      `json:"p2p_bootstrap_peers"`
	P2PMaxPeers        int           `json:"p2p_max_peers"`
	P2PPeerTimeout     time.Duration `json:"p2p_peer_timeout"`
	P2PDialTimeout     time.Duration `json:"p2p_dial_timeout"`
	P2PProtocolVersion string        `json:"p2p_protocol_version"`

	// WebSocket configuration
	WSWriteTimeout   time.Duration `json:"ws_write_timeout"`
	WSPingInterval   time.Duration `json:"ws_ping_interval"`
	WSMaxMessageSize int           `json:"ws_max_message_size"`

	// CORS configuration
	EnableCORS     bool     `json:"enable_cors"`
	CORSOrigins    []string `json:"cors_origins"`
	TrustedProxies []string `json:"trusted_proxies"`

	// Compression and security
	EnableCompression     bool `json:"enable_compression"`
	EnableSecurityHeaders bool `json:"enable_security_headers"`

	// Relay configuration
	RelayMaxConcurrent int           `json:"relay_max_concurrent"`
	RelayTimeout       time.Duration `json:"relay_timeout"`
	RelayRetryAttempts int           `json:"relay_retry_attempts"`
	RelayRetryDelay    time.Duration `json:"relay_retry_delay"`

	// External endpoint configuration
	ExternalEndpoints []ExternalEndpoint `json:"external_endpoints"`
}

// Load reads config from env
func Load() Config {
	// Load environment variables from .env files
	loadEnvironmentConfig()

	tier := Tier(getEnv("TIER", "free"))

	cfg := Config{
		BitcoinNodes:             []string{getEnv("BITCOIN_NODE", "127.0.0.1:8333")},
		ZMQNodes:                 []string{getEnv("ZMQ_NODE", "127.0.0.1:28332")},
		PeerListenPort:           getEnvInt("PEER_LISTEN_PORT", 8335),
		AdminPort:                getEnvInt("ADMIN_PORT", 8081),
		LicenseKey:               getEnv("LICENSE_KEY", ""),
		APIKey:                   getEnv("API_KEY", "changeme"),
		SecureChannelURL:         getEnv("SECURE_CHANNEL_URL", "tcp://127.0.0.1:9000"),
		UseDirectP2P:             getEnvBool("USE_DIRECT_P2P", false),
		UseMemoryChannel:         getEnvBool("USE_MEMORY_CHANNEL", false),
		OptimizeSystem:           getEnvBool("OPTIMIZE_SYSTEM", true),
		RequireDatabase:          getEnvBool("REQUIRE_DATABASE", false),
		BlockChannelBuffer:       getEnvInt("BLOCK_CHANNEL_BUFFER", 1000),
		BlockDeduplicationWindow: time.Duration(getEnvInt("BLOCK_DEDUPLICATION_WINDOW", 60)) * time.Second,
		CacheSize:                getEnvInt("CACHE_SIZE", 10000),
		MempoolMaxSize:           getEnvInt("MEMPOOL_MAX_SIZE", 50000),

		// Performance optimizations (permanent defaults for 99.9% SLA)
		GCPercent:       getEnvInt("GC_PERCENT", 25),
		MaxCPUCores:     getEnvInt("MAX_CPU_CORES", 0),
		HighPriority:    getEnvBool("HIGH_PRIORITY", true),
		LockOSThread:    getEnvBool("LOCK_OS_THREAD", true),
		PreallocBuffers: getEnvBool("PREALLOC_BUFFERS", true),

		// Enterprise-ready settings
		EnablePrometheus:     getEnvBool("ENABLE_PROMETHEUS", true),
		PrometheusPort:       getEnvInt("PROMETHEUS_PORT", 9090),
		EnableTLS:            getEnvBool("ENABLE_TLS", false),
		EnableMTLS:           getEnvBool("ENABLE_MTLS", false),
		IdleTimeout:          time.Duration(getEnvInt("IDLE_TIMEOUT_SEC", 300)) * time.Second,
		MessageRateLimit:     getEnvInt("MESSAGE_RATE_LIMIT", 100),
		GeneralRateLimit:     getEnvInt("GENERAL_RATE_LIMIT", 100),
		WebSocketMaxGlobal:   getEnvInt("WEBSOCKET_MAX_GLOBAL", 1000),
		WebSocketMaxPerIP:    getEnvInt("WEBSOCKET_MAX_PER_IP", 10),
		WebSocketMaxPerChain: getEnvInt("WEBSOCKET_MAX_PER_CHAIN", 100),

		// Persistence settings
		DatabaseType:      getEnv("DATABASE_TYPE", "sqlite"),
		DatabaseURL:       getEnv("DATABASE_URL", "./sprint.db"),
		EnablePersistence: getEnvBool("ENABLE_PERSISTENCE", true),

		// Blockchain-agnostic settings
		SupportedChains: []string{"btc", "eth", "sol", "polygon", "arbitrum"},
		DefaultChain:    getEnv("DEFAULT_CHAIN", "btc"),

		// Sprint relay peer settings
		SprintRelayPeers: getEnvSlice("SPRINT_RELAY_PEERS", []string{}),

		// RPC Configuration for backfill operations
		RPCEnabled:       getEnvBool("RPC_ENABLED", false),
		RPCURL:           getEnv("RPC_URL", "http://127.0.0.1:8332"),
		RPCUsername:      getEnv("RPC_USERNAME", "sprint"),
		RPCPassword:      getEnv("RPC_PASSWORD", "sprint_password_2025"),
		RPCTimeout:       time.Duration(getEnvInt("RPC_TIMEOUT_SEC", 30)) * time.Second,
		RPCBatchSize:     getEnvInt("RPC_BATCH_SIZE", 50),
		RPCRetryAttempts: getEnvInt("RPC_RETRY_ATTEMPTS", 3),
		RPCRetryMaxWait:  time.Duration(getEnvInt("RPC_RETRY_MAX_WAIT_MIN", 5)) * time.Minute,
		RPCSkipMempool:   getEnvBool("RPC_SKIP_MEMPOOL", false),

		// API timeouts
		APIReadTimeout:  time.Duration(getEnvInt("API_READ_TIMEOUT_SEC", 30)) * time.Second,
		APIWriteTimeout: time.Duration(getEnvInt("API_WRITE_TIMEOUT_SEC", 30)) * time.Second,
		P2PPeerTimeout:  time.Duration(getEnvInt("P2P_PEER_TIMEOUT_SEC", 30)) * time.Second,
		RPCFailedTxFile:  getEnv("RPC_FAILED_TX_FILE", "./failed_txs.txt"),
		RPCLastIDFile:    getEnv("RPC_LAST_ID_FILE", "./last_id.txt"),
		RPCWorkers:       getEnvInt("RPC_WORKERS", 10),
		RPCMessageTopic:  getEnv("RPC_MESSAGE_TOPIC", "bitcoin.transactions"),
	}

	// Parse API_ADDR if provided (format: host:port)
	apiAddr := getEnv("API_ADDR", "")
	if apiAddr != "" {
		if host, port, err := parseAddr(apiAddr); err == nil {
			cfg.APIHost = host
			cfg.APIPort = port
		}
	} else {
		// Use individual settings
		cfg.APIHost = getEnv("API_HOST", "0.0.0.0")
		cfg.APIPort = getEnvInt("API_PORT", 8080)
	}

	// Parse METRICS_ADDR if provided (format: host:port)
	metricsAddr := getEnv("METRICS_ADDR", "")
	if metricsAddr != "" {
		if _, port, err := parseAddr(metricsAddr); err == nil {
			cfg.PrometheusPort = port
			// Note: We don't override the host for metrics as it's typically localhost
		}
	} else {
		cfg.PrometheusPort = getEnvInt("PROMETHEUS_PORT", 9090)
	}

	// Initialize tier-based rate limits
	cfg.RateLimits = getDefaultRateLimits()

	// Apply tier-based optimizations
	switch tier {
	case TierTurbo:
		cfg.WriteDeadline = 300 * time.Microsecond // Reduced from 500µs for 1-3ms target
		cfg.UseSharedMemory = true
		cfg.BlockBufferSize = 4096 // Increased buffer size
		cfg.UseMemoryChannel = true
		cfg.UseDirectP2P = true
		cfg.MaxOutstandingHeadersPerPeer = 10000 // Increased for higher throughput
		cfg.PipelineWorkers = 4                  // Increased workers for turbo mode
	case TierEnterprise:
		cfg.WriteDeadline = 150 * time.Microsecond // Reduced for sub-1ms target
		cfg.UseSharedMemory = true
		cfg.BlockBufferSize = 8192 // Larger buffer for enterprise
		cfg.UseMemoryChannel = true
		cfg.UseDirectP2P = true
		cfg.OptimizeSystem = true
		cfg.MaxOutstandingHeadersPerPeer = 5000
		cfg.PipelineWorkers = 2
	case TierBusiness:
		cfg.WriteDeadline = 800 * time.Millisecond // Reduced from 1s
		cfg.BlockBufferSize = 2048
		cfg.MaxOutstandingHeadersPerPeer = 2000
		cfg.PipelineWorkers = 2
	case TierPro:
		cfg.WriteDeadline = 1200 * time.Millisecond // Reduced from 1.5s
		cfg.BlockBufferSize = 1536
		cfg.MaxOutstandingHeadersPerPeer = 1000
		cfg.PipelineWorkers = 2
	case TierFree:
		cfg.MaxOutstandingHeadersPerPeer = 500 // Increased from 200
		cfg.PipelineWorkers = 1
	}

	return cfg
}

// getDefaultRateLimits returns default rate limits for each tier
func getDefaultRateLimits() map[Tier]TierRateLimit {
	return map[Tier]TierRateLimit{
		TierFree: {
			RequestsPerSecond:    1,
			RequestsPerHour:      1000,
			ConcurrentStreams:    1,
			DataSizeLimitMB:      10,
			KeyGenerationPerHour: 5,
			WebSocketMessageRate: 10,
			RefillRate:           1.0 / 3600.0, // 1 request per hour
			BurstCapacity:        5,
		},
		TierPro: {
			RequestsPerSecond:    10,
			RequestsPerHour:      10000,
			ConcurrentStreams:    5,
			DataSizeLimitMB:      100,
			KeyGenerationPerHour: 20,
			WebSocketMessageRate: 50,
			RefillRate:           10.0 / 3600.0, // 10 requests per hour
			BurstCapacity:        50,
		},
		TierBusiness: {
			RequestsPerSecond:    50,
			RequestsPerHour:      50000,
			ConcurrentStreams:    20,
			DataSizeLimitMB:      500,
			KeyGenerationPerHour: 100,
			WebSocketMessageRate: 200,
			RefillRate:           50.0 / 3600.0, // 50 requests per hour
			BurstCapacity:        250,
		},
		TierTurbo: {
			RequestsPerSecond:    100,
			RequestsPerHour:      100000,
			ConcurrentStreams:    50,
			DataSizeLimitMB:      1000,
			KeyGenerationPerHour: 500,
			WebSocketMessageRate: 500,
			RefillRate:           100.0 / 3600.0, // 100 requests per hour
			BurstCapacity:        500,
		},
		TierEnterprise: {
			RequestsPerSecond:    500,
			RequestsPerHour:      500000,
			ConcurrentStreams:    100,
			DataSizeLimitMB:      5000,
			KeyGenerationPerHour: 1000,
			WebSocketMessageRate: 1000,
			RefillRate:           500.0 / 3600.0, // 500 requests per hour
			BurstCapacity:        2500,
		},
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		i, err := strconv.Atoi(v)
		if err == nil {
			return i
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		return v == "1" || v == "true"
	}
	return def
}

func getEnvSlice(key string, def []string) []string {
	if v := os.Getenv(key); v != "" {
		// Simple comma-separated parsing (can be enhanced for more complex formats)
		if v == "" {
			return def
		}
		// Split by comma and trim spaces
		parts := strings.Split(v, ",")
		result := make([]string, len(parts))
		for i, part := range parts {
			result[i] = strings.TrimSpace(part)
		}
		return result
	}
	return def
}

// loadEnvironmentConfig loads .env files with tier-specific support
func loadEnvironmentConfig() {
	// First, try to load default .env file
	if err := godotenv.Load(); err == nil {
		log.Printf("Config: Loaded default .env file")
	} else {
		log.Printf("Config: No default .env file found, using system environment variables")
	}

	// Check for tier-specific .env file
	tier := getEnv("TIER", "")
	if tier != "" {
		tierEnvFile := fmt.Sprintf(".env.%s", tier)
		if err := godotenv.Load(tierEnvFile); err == nil {
			log.Printf("Config: Loaded tier-specific .env file: %s", tierEnvFile)
		} else {
			log.Printf("Config: No tier-specific .env file found: %s", tierEnvFile)
		}
	}

	// Also check for RELAY_TIER (legacy support)
	relayTier := getEnv("RELAY_TIER", "")
	if relayTier != "" && relayTier != tier {
		relayTierEnvFile := fmt.Sprintf(".env.%s", strings.ToLower(relayTier))
		if err := godotenv.Load(relayTierEnvFile); err == nil {
			log.Printf("Config: Loaded relay tier .env file: %s", relayTierEnvFile)
		}
	}
}

// Get retrieves a configuration value by key with a default fallback
func (c *Config) Get(key, def string) string {
	// For now, just return environment variables directly
	// In a production system, this could be enhanced to support
	// configuration files, databases, etc.
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// GetStringSlice retrieves a configuration value as a string slice
func (c *Config) GetStringSlice(key string) []string {
	if v := os.Getenv(key); v != "" {
		if v == "" {
			return []string{}
		}
		// Split by comma and trim spaces
		parts := strings.Split(v, ",")
		result := make([]string, len(parts))
		for i, part := range parts {
			result[i] = strings.TrimSpace(part)
		}
		return result
	}
	return []string{}
}

// GetInt retrieves a configuration value as an integer with a default fallback
func (c *Config) GetInt(key string) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return 0
}

// GetDuration retrieves a configuration value as a duration
func (c *Config) GetDuration(key string) time.Duration {
	if v := os.Getenv(key); v != "" {
		// Try parsing as seconds first
		if i, err := strconv.Atoi(v); err == nil {
			return time.Duration(i) * time.Second
		}
		// Try parsing as duration string
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return 0
}

// parseAddr parses a host:port string and returns host, port, and error
func parseAddr(addr string) (string, int, error) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid address format: %s", addr)
	}
	host := parts[0]
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("invalid port: %s", parts[1])
	}
	return host, port, nil
}
