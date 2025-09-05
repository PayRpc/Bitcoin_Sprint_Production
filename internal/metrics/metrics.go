// internal/metrics/metrics.go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// BlockDuplicatesIgnored tracks blocks dropped by dedup layer
	BlockDuplicatesIgnored = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "blocks_duplicates_ignored_total",
			Help: "Blocks dropped by dedup layer",
		},
		[]string{"source"},
	)

	// BlocksProcessed tracks blocks fully processed
	BlocksProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "blocks_processed_total",
			Help: "Blocks fully processed",
		},
		[]string{"source"},
	)

	// BlockProcessingDuration tracks processing time per block
	BlockProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "block_processing_duration_seconds",
			Help:    "Time spent processing blocks",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"source"},
	)

	// DeduplicationCacheSize tracks the current size of deduplication cache
	DeduplicationCacheSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "deduplication_cache_size",
			Help: "Current number of entries in deduplication cache",
		},
	)

	// DeduplicationHitRate tracks cache hit rate
	DeduplicationHitRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "deduplication_hit_rate",
			Help: "Percentage of blocks that were duplicates",
		},
		[]string{"source"},
	)

	// DeduplicationProcessingTime tracks processing time for deduplication operations
	DeduplicationProcessingTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "deduplication_processing_duration_seconds",
			Help:    "Time spent on deduplication processing",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "network"},
	)

	// DeduplicationDuplicatesDetected tracks number of duplicates detected
	DeduplicationDuplicatesDetected = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "deduplication_duplicates_detected_total",
			Help: "Number of duplicate entries detected",
		},
		[]string{"network", "type"},
	)

	// DeduplicationMemoryUsage tracks memory usage of deduplication system
	DeduplicationMemoryUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "deduplication_memory_usage_bytes",
			Help: "Memory usage of deduplication system",
		},
		[]string{"component"},
	)

	// DeduplicationEfficiency tracks efficiency metrics
	DeduplicationEfficiency = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "deduplication_efficiency_ratio",
			Help: "Efficiency ratio of deduplication system",
		},
		[]string{"network", "metric"},
	)

	// DeduplicationNetworkDuplicateRate tracks network-specific duplicate rates
	DeduplicationNetworkDuplicateRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "deduplication_network_duplicate_rate",
			Help: "Duplicate rate per network",
		},
		[]string{"network"},
	)

	// DeduplicationNetworkTTL tracks network-specific TTL values
	DeduplicationNetworkTTL = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "deduplication_network_ttl_seconds",
			Help: "TTL values per network",
		},
		[]string{"network"},
	)
)
