package relay

import (
	"math"
	"sync"
	"time"
)

type breakerState int

const (
	breakerClosed breakerState = iota
	breakerOpen
	breakerHalfOpen
)

type endpointStats struct {
	url       string
	ewmaRTT   float64 // exponential moving average latency (ms)
	failures  int64
	successes int64
	lastErr   string
	lastSeen  time.Time

	// circuit breaker
	state      breakerState
	trippedAt  time.Time
	breakUntil time.Time
}

func (e *endpointStats) recordSuccess(latency time.Duration) {
	e.successes++
	e.lastSeen = time.Now()
	// EWMA for latency (alpha 0.2)
	const alpha = 0.2
	lat := float64(latency.Milliseconds())
	if e.ewmaRTT == 0 {
		e.ewmaRTT = lat
	} else {
		e.ewmaRTT = (1.0-alpha)*e.ewmaRTT + alpha*lat
	}
	// successful call helps close breaker if half-open
	if e.state == breakerHalfOpen {
		// If we gather enough consecutive successes, close it
		if e.successes%3 == 0 {
			e.state = breakerClosed
		}
	}
}

func (e *endpointStats) recordFailure(errStr string) {
	e.failures++
	e.lastErr = errStr
	e.lastSeen = time.Now()

	// open circuit after threshold
	if e.failures >= 5 && e.state != breakerOpen {
		e.state = breakerOpen
		e.trippedAt = time.Now()
		e.breakUntil = time.Now().Add(30 * time.Second)
	} else if e.state == breakerOpen && time.Now().After(e.breakUntil) {
		// move to half-open after wait
		e.state = breakerHalfOpen
	}
}

func (e *endpointStats) available() bool {
	switch e.state {
	case breakerClosed:
		return true
	case breakerHalfOpen:
		return true // allow probing
	case breakerOpen:
		return time.Now().After(e.breakUntil)
	default:
		return true
	}
}

func (e *endpointStats) score() float64 {
	// Smaller RTT => higher score; more failures => lower score
	// Add small constant to avoid div by zero
	rtt := e.ewmaRTT
	if rtt <= 0 {
		rtt = 50
	}
	failPenalty := 1.0 / (1.0 + math.Log1p(float64(e.failures)))
	statePenalty := 1.0
	if e.state == breakerOpen {
		statePenalty = 0.1
	} else if e.state == breakerHalfOpen {
		statePenalty = 0.5
	}
	return (1.0 / rtt) * failPenalty * statePenalty
}

// ----- Manager -----

type endpointHealth struct {
	mu    sync.RWMutex
	stats map[string]*endpointStats
}

func newEndpointHealth(endpoints []string) *endpointHealth {
	m := &endpointHealth{
		stats: make(map[string]*endpointStats, len(endpoints)),
	}
	for _, e := range endpoints {
		m.stats[e] = &endpointStats{url: e}
	}
	return m
}

func (m *endpointHealth) recordSuccess(endpoint string, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if st, ok := m.stats[endpoint]; ok {
		st.recordSuccess(latency)
	}
}

func (m *endpointHealth) recordFailure(endpoint string, err string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if st, ok := m.stats[endpoint]; ok {
		st.recordFailure(err)
	}
}

func (m *endpointHealth) pickWeighted() (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := 0.0
	weights := make(map[string]float64, len(m.stats))
	for _, st := range m.stats {
		if !st.available() {
			continue
		}
		w := st.score()
		if w <= 0 {
			w = 0.0001
		}
		weights[st.url] = w
		total += w
	}
	if total == 0 {
		// fallback: anything available?
		for _, st := range m.stats {
			if st.available() {
				return st.url, true
			}
		}
		return "", false
	}
	// weighted random: deterministic alternative: choose max score
	// here we pick max score for stability
	var bestURL string
	bestScore := -1.0
	for u, w := range weights {
		if w > bestScore {
			bestScore = w
			bestURL = u
		}
	}
	return bestURL, bestURL != ""
}

func (m *endpointHealth) snapshot() map[string]endpointStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]endpointStats, len(m.stats))
	for k, v := range m.stats {
		out[k] = *v
	}
	return out
}
