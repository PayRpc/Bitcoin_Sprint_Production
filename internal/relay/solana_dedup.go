package relay

import (
	"sync"
	"time"
)

type solanaDeduper struct {
	mu          sync.Mutex
	seen        map[string]time.Time
	ttl         time.Duration
	minTTL      time.Duration
	maxTTL      time.Duration
	dupCount    int64
	totalCount  int64
	lastAdjust  time.Time
	adjustEvery time.Duration
}

func newSolanaDeduper() *solanaDeduper {
	return &solanaDeduper{
		seen:        make(map[string]time.Time, 4096),
		ttl:         25 * time.Second, // sensible default for Solana slot churn
		minTTL:      5 * time.Second,
		maxTTL:      2 * time.Minute,
		adjustEvery: 30 * time.Second,
		lastAdjust:  time.Now(),
	}
}

func (d *solanaDeduper) isDup(key string) bool {
	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()

	d.totalCount++
	if t, ok := d.seen[key]; ok {
		if now.Sub(t) <= d.ttl {
			d.dupCount++
			return true
		}
	}
	d.seen[key] = now
	// cleanup opportunistically
	for k, ts := range d.seen {
		if now.Sub(ts) > d.ttl {
			delete(d.seen, k)
		}
	}
	// periodic TTL adjustment
	if now.Sub(d.lastAdjust) >= d.adjustEvery {
		d.adjustTTLLocked()
		d.lastAdjust = now
	}
	return false
}

func (d *solanaDeduper) adjustTTLLocked() {
	if d.totalCount < 20 {
		return
	}
	rate := float64(d.dupCount) / float64(d.totalCount)
	switch {
	case rate > 0.50:
		// lots of duplicates, increase TTL aggressively
		d.ttl = d.ttl + 10*time.Second
	case rate > 0.25:
		d.ttl = d.ttl + 5*time.Second
	case rate < 0.05:
		// few duplicates: shrink TTL
		if d.ttl > 10*time.Second {
			d.ttl = d.ttl - 5*time.Second
		}
	default:
		// small drift
		d.ttl = d.ttl + 1*time.Second
	}
	if d.ttl < d.minTTL {
		d.ttl = d.minTTL
	}
	if d.ttl > d.maxTTL {
		d.ttl = d.maxTTL
	}
	// mild decay on counters
	d.totalCount = d.totalCount / 2
	d.dupCount = d.dupCount / 2
}

func (d *solanaDeduper) stats() (ttl time.Duration, dupRate float64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	ttl = d.ttl
	if d.totalCount == 0 {
		return ttl, 0
	}
	return ttl, float64(d.dupCount) / float64(d.totalCount)
}
