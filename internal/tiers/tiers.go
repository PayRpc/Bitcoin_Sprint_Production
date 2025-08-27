package tiers

import (
	"os"
	"time"
)

// TierConfig defines speed/security settings per tier
type TierConfig struct {
	Name          string        `json:"name"`
	BlockDeadline time.Duration `json:"-"`
	RelayFanout   int           `json:"relay_fanout"`
	MaxPeers      int           `json:"max_peers"`
	SecurityLevel string        `json:"security_level"`
	LatencyTarget string        `json:"latency_target"`
	Features      []string      `json:"features"`
}

// TierInfo provides customer-facing tier information
type TierInfo struct {
	Service      string          `json:"service"`
	Version      string          `json:"version"`
	CurrentTier  string          `json:"current_tier"`
	TierConfig   TierConfig      `json:"tier_config"`
	Security     SecurityInfo    `json:"security"`
	Performance  PerformanceInfo `json:"performance"`
	APIEndpoints []string        `json:"api_endpoints"`
}

type SecurityInfo struct {
	Secrets      string `json:"secrets"`
	Handshake    string `json:"handshake"`
	Relay        string `json:"relay"`
	AuditLogging bool   `json:"audit_logging"`
}

type PerformanceInfo struct {
	CoreDetectionSpeed  string `json:"core_detection_speed"`
	TurboDetectionSpeed string `json:"turbo_detection_speed"`
	Throughput          string `json:"throughput"`
	Resilience          string `json:"resilience"`
}

// GetTierConfig loads settings based on SPRINT_TIER env
func GetTierConfig() TierConfig {
	switch os.Getenv("SPRINT_TIER") {
	case "turbo":
		return TierConfig{
			Name:          "turbo",
			BlockDeadline: 5 * time.Millisecond, // auto-throttled vs 500µs
			RelayFanout:   100,
			MaxPeers:      64,
			SecurityLevel: "SecureBuffer + HMAC + AES-GCM",
			LatencyTarget: "≤5ms",
			Features:      []string{"Shared Memory", "Direct P2P", "Memory Channel", "Rust SecureBuffer", "HMAC Auth", "AES-GCM Encryption"},
		}
	case "enterprise":
		return TierConfig{
			Name:          "enterprise",
			BlockDeadline: 20 * time.Millisecond,
			RelayFanout:   250,
			MaxPeers:      256,
			SecurityLevel: "AES-GCM + Replay Protection + Audit Logs",
			LatencyTarget: "≤20ms",
			Features:      []string{"Encrypted Relays", "Replay Protection", "Audit Logging", "Circuit Breakers", "Rate Limiting"},
		}
	case "lite":
		return TierConfig{
			Name:          "lite",
			BlockDeadline: 1 * time.Second,
			RelayFanout:   2,
			MaxPeers:      4,
			SecurityLevel: "SecureBuffer only",
			LatencyTarget: "≤1s",
			Features:      []string{"Basic SecureBuffer", "Simple P2P"},
		}
	default: // Standard
		return TierConfig{
			Name:          "standard",
			BlockDeadline: 300 * time.Millisecond,
			RelayFanout:   10,
			MaxPeers:      16,
			SecurityLevel: "TLS + HMAC + SecureBuffer",
			LatencyTarget: "≤300ms",
			Features:      []string{"TLS Security", "HMAC Auth", "SecureBuffer", "Basic P2P"},
		}
	}
}

// GetTierInfo returns complete tier information for API responses
func GetTierInfo(version string) TierInfo {
	config := GetTierConfig()

	return TierInfo{
		Service:     "Bitcoin Sprint",
		Version:     version,
		CurrentTier: config.Name,
		TierConfig:  config,
		Security: SecurityInfo{
			Secrets:      "Rust SecureBuffer (mlock, zeroized)",
			Handshake:    "HMAC-SHA256 with replay protection",
			Relay:        "AES-256-GCM encrypted notifications",
			AuditLogging: config.Name == "enterprise",
		},
		Performance: PerformanceInfo{
			CoreDetectionSpeed:  "200ms vs 10–30s (Core)",
			TurboDetectionSpeed: "≤5ms (Linux prod) / ≤5ms (Windows dev fallback)",
			Throughput:          "200k+ req/sec benchmarked",
			Resilience:          "Circuit breakers + per-peer rate caps",
		},
		APIEndpoints: []string{
			"/status",
			"/latest",
			"/metrics",
			"/stream",
			"/tier-info",
			"/health",
		},
	}
}
