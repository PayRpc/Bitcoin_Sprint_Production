package mempool

import (
	"sync"
	"time"
)

// Mempool with TTL eviction
type Mempool struct {
	mu    sync.RWMutex
	items map[string]int64 // txid → expiryUnix
	ttl   int64
}

func New() *Mempool {
	m := &Mempool{
		items: make(map[string]int64),
		ttl:   300, // 5 min TTL
	}
	go m.gcLoop()
	return m
}

func (m *Mempool) Add(txid string) {
	m.mu.Lock()
	defer m.mu.Unlock()
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
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items)
}

func (m *Mempool) gcLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now().Unix()
		m.mu.Lock()
		for k, exp := range m.items {
			if exp < now {
				delete(m.items, k)
			}
		}
		m.mu.Unlock()
	}
}
