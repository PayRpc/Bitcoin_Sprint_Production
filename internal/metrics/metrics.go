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
)
