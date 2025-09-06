package relay

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type solanaProm struct {
	endpointScore   *prometheus.GaugeVec
	endpointLatency *prometheus.GaugeVec
	endpointState   *prometheus.GaugeVec // 0=closed,1=half-open,2=open

	wsReconnects prometheus.Counter
	dupDropped   prometheus.Counter
	ttlSeconds   prometheus.Gauge
}

func newSolanaProm(namespace string) *solanaProm {
	lbls := []string{"endpoint"}
	return &solanaProm{
		endpointScore: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "solana",
			Name:      "endpoint_score",
			Help:      "Weighted score per endpoint",
		}, lbls),

		endpointLatency: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "solana",
			Name:      "endpoint_latency_ms",
			Help:      "EWMA latency per endpoint (milliseconds)",
		}, lbls),

		endpointState: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "solana",
			Name:      "endpoint_breaker_state",
			Help:      "Circuit breaker state: 0=closed,1=half-open,2=open",
		}, lbls),

		wsReconnects: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "solana",
			Name:      "ws_reconnects_total",
			Help:      "Total websocket reconnects",
		}),

		dupDropped: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "solana",
			Name:      "duplicates_dropped_total",
			Help:      "Total duplicate blocks dropped by deduper",
		}),

		ttlSeconds: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "solana",
			Name:      "dedup_ttl_seconds",
			Help:      "Current adaptive dedup TTL (seconds)",
		}),
	}
}
