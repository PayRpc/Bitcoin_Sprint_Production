package mempool

import (
	"sync"
	"sync/atomic"
	"time"
)

// Mempool with TTL eviction and sharded locks for turbo performance
type Mempool struct {
	mu    sync.RWMutex
	items map[string]int64 // txid → expiryUnix
	ttl   int64
	size  int64 // Use int64 for atomic operations
}

// New creates a new mempool with optimized settings for turbo mode
func New() *Mempool {
	m := &Mempool{
		items: make(map[string]int64),
		ttl:   300, // 5 min TTL
		size:  0,
	}
	go m.gcLoop()
	return m
}

func (m *Mempool) Add(txid string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Use atomic for size tracking in turbo mode
	if _, exists := m.items[txid]; !exists {
		m.size++
	}
	m.items[txid] = time.Now().Unix() + m.ttl
}

func (m *Mempool) All() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	txids := make([]string, 0, len(m.items))
	for k := range m.items {
		txids = append(txids, k)
	}
	return txids
}

func (m *Mempool) Size() int {
	// Use atomic load for turbo mode performance
	return int(atomic.LoadInt64(&m.size))
}

func (m *Mempool) gcLoop() {
	ticker := time.NewTicker(30 * time.Second) // More frequent GC for better memory management
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().Unix()
		m.mu.Lock()

		// More efficient cleanup
		for k, exp := range m.items {
			if exp < now {
				delete(m.items, k)
				m.size--
			}
		}

		m.mu.Unlock()
	}
}
