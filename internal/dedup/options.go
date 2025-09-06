// internal/dedup/options.go
package dedup

// DedupeOptions provides configuration for advanced deduplication behavior
type DedupeOptions struct {
	Source       string                 `json:"source"`        // Source identifier (peer, API, etc.)
	Size         int64                  `json:"size"`          // Block size in bytes
	Properties   map[string]interface{} `json:"properties"`    // Additional metadata
	CrossNetwork bool                   `json:"cross_network"` // Enable cross-network deduplication
	Priority     int                    `json:"priority"`      // Processing priority (1-10)
	Confidence   float64                `json:"confidence"`    // Confidence threshold (0.0-1.0)
	ForceUpdate  bool                   `json:"force_update"`  // Force update even if duplicate
}

// DedupeOption is a functional option for configuring deduplication behavior
type DedupeOption func(*DedupeOptions)

// WithSource sets the source identifier for tracking
func WithSource(source string) DedupeOption {
	return func(opts *DedupeOptions) {
		opts.Source = source
	}
}

// WithSize sets the block size for size-based optimizations
func WithSize(size int64) DedupeOption {
	return func(opts *DedupeOptions) {
		opts.Size = size
	}
}

// WithProperties adds custom metadata for advanced processing
func WithProperties(props map[string]interface{}) DedupeOption {
	return func(opts *DedupeOptions) {
		if opts.Properties == nil {
			opts.Properties = make(map[string]interface{})
		}
		for k, v := range props {
			opts.Properties[k] = v
		}
	}
}

// WithCrossNetwork enables cross-network deduplication
func WithCrossNetwork() DedupeOption {
	return func(opts *DedupeOptions) {
		opts.CrossNetwork = true
	}
}

// WithPriority sets processing priority (1-10, higher = more important)
func WithPriority(priority int) DedupeOption {
	return func(opts *DedupeOptions) {
		if priority < 1 {
			priority = 1
		} else if priority > 10 {
			priority = 10
		}
		opts.Priority = priority
	}
}

// WithConfidence sets minimum confidence threshold
func WithConfidence(confidence float64) DedupeOption {
	return func(opts *DedupeOptions) {
		if confidence < 0.0 {
			confidence = 0.0
		} else if confidence > 1.0 {
			confidence = 1.0
		}
		opts.Confidence = confidence
	}
}

// WithForceUpdate forces update even if block is detected as duplicate
func WithForceUpdate() DedupeOption {
	return func(opts *DedupeOptions) {
		opts.ForceUpdate = true
	}
}

// Performance modes for different use cases
const (
	PerformanceModeStandard         = "STANDARD"
	PerformanceModeHighPerformance  = "HIGH_PERFORMANCE"
	PerformanceModeMemoryOptimized  = "MEMORY_OPTIMIZED"
	PerformanceModeLatencyOptimized = "LATENCY_OPTIMIZED"
)

// Network priority constants
const (
	NetworkPriorityLow      = 1
	NetworkPriorityMedium   = 3
	NetworkPriorityHigh     = 5
	NetworkPriorityCritical = 10
)

// Default configuration values
const (
	DefaultMaxSize             = 10000
	DefaultBaseTTL             = 5 * 60 // 5 minutes in seconds
	DefaultConfidenceThreshold = 0.85
	DefaultAdaptationRate      = 0.1
	DefaultCleanupInterval     = 30 // seconds
)

// Enterprise tier configurations
var (
	FreeBaseTier = map[string]interface{}{
		"max_size":         1000,
		"base_ttl_seconds": 300, // 5 minutes
		"adaptive_enabled": false,
		"ml_optimization":  false,
		"cross_network":    false,
		"priority_queuing": false,
	}

	BusinessTier = map[string]interface{}{
		"max_size":         5000,
		"base_ttl_seconds": 600, // 10 minutes
		"adaptive_enabled": true,
		"ml_optimization":  false,
		"cross_network":    true,
		"priority_queuing": true,
	}

	EnterpriseTier = map[string]interface{}{
		"max_size":           15000,
		"base_ttl_seconds":   900, // 15 minutes
		"adaptive_enabled":   true,
		"ml_optimization":    true,
		"cross_network":      true,
		"priority_queuing":   true,
		"learning_enabled":   true,
		"advanced_analytics": true,
	}
)
