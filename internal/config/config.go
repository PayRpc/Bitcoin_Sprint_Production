package config

import (
	"os"
	"strconv"
	"time"
)

// TierRateLimit defines rate limits for each tier
type TierRateLimit struct {
	RequestsPerSecond     int           `json:"requests_per_second"`
	RequestsPerHour       int           `json:"requests_per_hour"`
	ConcurrentStreams     int           `json:"concurrent_streams"`
	DataSizeLimitMB       int           `json:"data_size_limit_mb"`
	KeyGenerationPerHour  int           `json:"key_generation_per_hour"`
	WebSocketMessageRate  int           `json:"websocket_message_rate"`
	RefillRate            float64       `json:"refill_rate"` // tokens per second
	BurstCapacity         int           `json:"burst_capacity"`
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
	BitcoinNodes     []string
	ZMQNodes         []string
	PeerListenPort   int
	APIHost          string
	APIPort          int
	AdminPort        int    // Separate port for admin endpoints
	LicenseKey       string
	APIKey           string
	SecureChannelURL string
	UseDirectP2P     bool
	UseMemoryChannel bool
	OptimizeSystem   bool

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
	EnablePrometheus bool   // Enable Prometheus metrics endpoint
	PrometheusPort   int    // Separate port for Prometheus metrics
	EnableTLS        bool   // Enable TLS for admin endpoints
	EnableMTLS       bool   // Enable mTLS for internal metrics
	IdleTimeout      time.Duration // WebSocket idle timeout
	MessageRateLimit int    // WebSocket messages per second per client
	GeneralRateLimit int    // General IP-based rate limit (requests per second)
	WebSocketMaxGlobal int  // Maximum global WebSocket connections
	WebSocketMaxPerIP  int  // Maximum WebSocket connections per IP
	WebSocketMaxPerChain int // Maximum WebSocket connections per chain

	// Persistence settings
	DatabaseType     string // sqlite, postgres, redis
	DatabaseURL      string // Connection string
	EnablePersistence bool  // Enable key persistence
}

// Load reads config from env
func Load() Config {
	tier := Tier(getEnv("TIER", "free"))

	cfg := Config{
		BitcoinNodes:     []string{getEnv("BITCOIN_NODE", "127.0.0.1:8333")},
		ZMQNodes:         []string{getEnv("ZMQ_NODE", "127.0.0.1:28332")},
		PeerListenPort:   getEnvInt("PEER_LISTEN_PORT", 8335),
		APIHost:          getEnv("API_HOST", "0.0.0.0"),
		APIPort:          getEnvInt("API_PORT", 8080),
		AdminPort:        getEnvInt("ADMIN_PORT", 8081), // Separate admin port
		LicenseKey:       getEnv("LICENSE_KEY", ""),
		APIKey:           getEnv("API_KEY", "changeme"),
		SecureChannelURL: getEnv("SECURE_CHANNEL_URL", "tcp://127.0.0.1:9000"),
		UseDirectP2P:     getEnvBool("USE_DIRECT_P2P", false),
		UseMemoryChannel: getEnvBool("USE_MEMORY_CHANNEL", false),
		OptimizeSystem:   getEnvBool("OPTIMIZE_SYSTEM", true), // Default to optimized

		// Performance optimizations (permanent defaults for 99.9% SLA)
		GCPercent:       getEnvInt("GC_PERCENT", 25),          // Aggressive GC by default
		MaxCPUCores:     getEnvInt("MAX_CPU_CORES", 0),        // Auto-detect all cores
		HighPriority:    getEnvBool("HIGH_PRIORITY", true),    // High priority by default
		LockOSThread:    getEnvBool("LOCK_OS_THREAD", true),   // Pin threads by default
		PreallocBuffers: getEnvBool("PREALLOC_BUFFERS", true), // Pre-allocate by default

		// Enterprise-ready settings
		EnablePrometheus: getEnvBool("ENABLE_PROMETHEUS", true),
		PrometheusPort:   getEnvInt("PROMETHEUS_PORT", 9090),
		EnableTLS:        getEnvBool("ENABLE_TLS", false),
		EnableMTLS:       getEnvBool("ENABLE_MTLS", false),
		IdleTimeout:      time.Duration(getEnvInt("IDLE_TIMEOUT_SEC", 300)) * time.Second,
		MessageRateLimit: getEnvInt("MESSAGE_RATE_LIMIT", 100),
		GeneralRateLimit: getEnvInt("GENERAL_RATE_LIMIT", 100),
		WebSocketMaxGlobal: getEnvInt("WEBSOCKET_MAX_GLOBAL", 1000),
		WebSocketMaxPerIP:  getEnvInt("WEBSOCKET_MAX_PER_IP", 10),
		WebSocketMaxPerChain: getEnvInt("WEBSOCKET_MAX_PER_CHAIN", 100),

		// Persistence settings
		DatabaseType:      getEnv("DATABASE_TYPE", "sqlite"),
		DatabaseURL:       getEnv("DATABASE_URL", "./sprint.db"),
		EnablePersistence: getEnvBool("ENABLE_PERSISTENCE", true),

		// Blockchain-agnostic settings
		SupportedChains: []string{"btc", "eth", "sol", "polygon", "arbitrum"},
		DefaultChain:    getEnv("DEFAULT_CHAIN", "btc"),

		// Turbo mode defaults
		Tier:               tier,
		WriteDeadline:      2 * time.Second,
		UseSharedMemory:    false,
		BlockBufferSize:    1024,
		EnableKernelBypass: getEnvBool("ENABLE_KERNEL_BYPASS", false),
		MockFastBlocks:     getEnvBool("MOCK_FAST_BLOCKS", false),
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
		cfg.PipelineWorkers = 4 // Increased workers for turbo mode
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
