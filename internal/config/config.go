package config

import (
	"os"
	"strconv"
	"time"
)

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
	LicenseKey       string
	APIKey           string
	SecureChannelURL string
	UseDirectP2P     bool
	UseMemoryChannel bool
	OptimizeSystem   bool
	
	// Turbo mode enhancements
	Tier             Tier
	WriteDeadline    time.Duration
	UseSharedMemory  bool
	BlockBufferSize  int
	EnableKernelBypass bool
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
		LicenseKey:       getEnv("LICENSE_KEY", ""),
		APIKey:           getEnv("API_KEY", "changeme"),
		SecureChannelURL: getEnv("SECURE_CHANNEL_URL", "tcp://127.0.0.1:9000"),
		UseDirectP2P:     getEnvBool("USE_DIRECT_P2P", false),
		UseMemoryChannel: getEnvBool("USE_MEMORY_CHANNEL", false),
		OptimizeSystem:   getEnvBool("OPTIMIZE_SYSTEM", false),
		
		// Turbo mode defaults
		Tier:             tier,
		WriteDeadline:    2 * time.Second,
		UseSharedMemory:  false,
		BlockBufferSize:  1024,
		EnableKernelBypass: getEnvBool("ENABLE_KERNEL_BYPASS", false),
	}

	// Apply tier-based optimizations
	switch tier {
	case TierTurbo:
		cfg.WriteDeadline = 500 * time.Microsecond
		cfg.UseSharedMemory = true
		cfg.BlockBufferSize = 2048
		cfg.UseMemoryChannel = true
		cfg.UseDirectP2P = true
	case TierEnterprise:
		cfg.WriteDeadline = 200 * time.Microsecond
		cfg.UseSharedMemory = true
		cfg.BlockBufferSize = 4096
		cfg.UseMemoryChannel = true
		cfg.UseDirectP2P = true
		cfg.OptimizeSystem = true
	case TierBusiness:
		cfg.WriteDeadline = 1 * time.Second
		cfg.BlockBufferSize = 1536
	case TierPro:
		cfg.WriteDeadline = 1500 * time.Millisecond
		cfg.BlockBufferSize = 1280
	}

	return cfg
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
