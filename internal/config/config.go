package config

import (
	"os"
	"strconv"
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
}

// Load reads config from env
func Load() Config {
	return Config{
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
