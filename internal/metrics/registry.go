package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusRegistry is a wrapper around Prometheus registry
type PrometheusRegistry struct {
	registry *prometheus.Registry
}

// NewRegistry creates a new Prometheus registry
func NewRegistry() *PrometheusRegistry {
	return &PrometheusRegistry{
		registry: prometheus.NewRegistry(),
	}
}

// Register registers a collector with the registry
func (r *PrometheusRegistry) Register(collector prometheus.Collector) error {
	return r.registry.Register(collector)
}

// MustRegister registers a collector with the registry and panics on error
func (r *PrometheusRegistry) MustRegister(collectors ...prometheus.Collector) {
	r.registry.MustRegister(collectors...)
}

// Unregister unregisters a collector from the registry
func (r *PrometheusRegistry) Unregister(collector prometheus.Collector) bool {
	return r.registry.Unregister(collector)
}

// GetRegistry returns the underlying Prometheus registry
func (r *PrometheusRegistry) GetRegistry() *prometheus.Registry {
	return r.registry
}

// Various metrics exposed by the application
var (
	DeduplicationProcessingTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "deduplication_processing_seconds",
			Help: "Time taken to process deduplication requests",
		},
		[]string{"operation", "network"},
	)

	DeduplicationCacheSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "deduplication_cache_entries",
			Help: "Number of entries in the deduplication cache",
		},
	)
)

func init() {
	// Register default metrics
	prometheus.MustRegister(DeduplicationProcessingTime)
	prometheus.MustRegister(DeduplicationCacheSize)
}
