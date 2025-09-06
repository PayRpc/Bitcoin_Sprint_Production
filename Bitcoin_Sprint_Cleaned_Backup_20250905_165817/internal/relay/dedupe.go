package relay

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Deduplication metrics
var (
	duplicateBlocksSuppressed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "relay_duplicate_blocks_suppressed_total",
		Help: "Number of duplicate block announcements dropped by the deduper",
	}, []string{"network"})
)

// BlockDeduper drops repeats within a TTL window with fixed-capacity to bound memory usage
type BlockDeduper struct {
	mu    sync.Mutex
	set   map[string]time.Time
	order []string
	cap   int
	ttl   time.Duration
}

// NewBlockDeduper creates a new deduplication handler with the specified capacity and TTL
func NewBlockDeduper(capacity int, ttl time.Duration) *BlockDeduper {
	if capacity <= 0 {
		capacity = 4096
	}
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &BlockDeduper{
		set:   make(map[string]time.Time, capacity),
		order: make([]string, 0, capacity),
		cap:   capacity,
		ttl:   ttl,
	}
}

// Seen returns true if hash is already seen within TTL; otherwise records it and returns false.
// It takes a network parameter for metrics tracking.
func (d *BlockDeduper) Seen(hash string, now time.Time, network string) bool {
	// Safety checks
	if d == nil {
		return false // If no deduper, never consider it a duplicate
	}

	if hash == "" {
		return false // Empty hashes are never considered duplicates
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if ts, ok := d.set[hash]; ok {
		if now.Sub(ts) <= d.ttl {
			duplicateBlocksSuppressed.WithLabelValues(network).Inc()
			return true
		}
		// expired → treat as new (below will refresh ts)
	}

	// record new
	d.set[hash] = now
	d.order = append(d.order, hash)

	// evict oldest if over cap
	if len(d.order) > d.cap {
		old := d.order[0]
		d.order = d.order[1:]
		delete(d.set, old)
	}
	return false
}

// Cleanup removes expired entries opportunistically; call from a ticker.
func (d *BlockDeduper) Cleanup() {
	// Safety check for nil deduper
	if d == nil {
		return
	}

	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()

	// Nothing to clean if empty
	if len(d.order) == 0 {
		return
	}

	// compact order by skipping expired ones
	w := 0
	for _, h := range d.order {
		if ts, ok := d.set[h]; ok && now.Sub(ts) <= d.ttl {
			d.order[w] = h
			w++
			continue
		}
		delete(d.set, h)
	}
	d.order = d.order[:w]
}
