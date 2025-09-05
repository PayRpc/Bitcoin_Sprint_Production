package p2p

import (
	"sync"
	"time"
)

// Deduper drops repeats within a TTL window. Fixed-capacity to bound memory.
type Deduper struct {
	mu    sync.Mutex
	set   map[string]time.Time
	order []string
	cap   int
	ttl   time.Duration
}

// NewDeduper creates a new deduplication handler with the specified capacity and TTL
func NewDeduper(capacity int, ttl time.Duration) *Deduper {
	if capacity <= 0 {
		capacity = 4096
	}
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &Deduper{
		set:   make(map[string]time.Time, capacity),
		order: make([]string, 0, capacity),
		cap:   capacity,
		ttl:   ttl,
	}
}

// Seen returns true if hash is already seen within TTL; otherwise records it and returns false.
func (d *Deduper) Seen(hash string, now time.Time) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if ts, ok := d.set[hash]; ok {
		if now.Sub(ts) <= d.ttl {
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
func (d *Deduper) Cleanup() {
	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()

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
