// Package api provides WebSocket connection management
package api

import (
	"sync"
)

// ===== WEBSOCKET LIMITER IMPLEMENTATION =====

// WebSocketLimiter manages WebSocket connection limits
type WebSocketLimiter struct {
	globalSem   chan struct{}
	perIPSem    map[string]chan struct{}
	perChainSem map[string]chan struct{}
	maxPerIP    int
	maxPerChain int
	mu          sync.RWMutex
}

// NewWebSocketLimiter creates a new WebSocket connection limiter
func NewWebSocketLimiter(maxGlobal, maxPerIP, maxPerChain int) *WebSocketLimiter {
	return &WebSocketLimiter{
		globalSem:   make(chan struct{}, maxGlobal),
		perIPSem:    make(map[string]chan struct{}),
		perChainSem: make(map[string]chan struct{}),
		maxPerIP:    maxPerIP,
		maxPerChain: maxPerChain,
	}
}

// Acquire acquires a WebSocket connection slot
func (wsl *WebSocketLimiter) Acquire(clientIP string) bool {
	// Try to acquire global slot
	select {
	case wsl.globalSem <- struct{}{}:
		// Acquired global slot, now try per-IP slot
		wsl.mu.Lock()
		if wsl.perIPSem[clientIP] == nil {
			wsl.perIPSem[clientIP] = make(chan struct{}, wsl.maxPerIP)
		}
		perIPSem := wsl.perIPSem[clientIP]
		wsl.mu.Unlock()

		select {
		case perIPSem <- struct{}{}:
			// Successfully acquired both slots
			return true
		default:
			// Failed to acquire per-IP slot, release global slot
			<-wsl.globalSem
			return false
		}
	default:
		// Failed to acquire global slot
		return false
	}
}

// Release releases a WebSocket connection slot
func (wsl *WebSocketLimiter) Release(clientIP string) {
	wsl.mu.RLock()
	perIPSem := wsl.perIPSem[clientIP]
	wsl.mu.RUnlock()

	if perIPSem != nil {
		select {
		case <-perIPSem:
			// Released per-IP slot
		default:
			// No slot to release
		}
	}

	select {
	case <-wsl.globalSem:
		// Released global slot
	default:
		// No slot to release
	}
}

// AcquireForChain acquires a WebSocket connection slot for a specific chain
func (wsl *WebSocketLimiter) AcquireForChain(clientIP, chain string) bool {
	// First try to acquire global slot
	select {
	case wsl.globalSem <- struct{}{}:
		// Acquired global slot, now try per-IP and per-chain slots
		wsl.mu.Lock()
		if wsl.perIPSem[clientIP] == nil {
			wsl.perIPSem[clientIP] = make(chan struct{}, wsl.maxPerIP)
		}
		if wsl.perChainSem[chain] == nil {
			wsl.perChainSem[chain] = make(chan struct{}, wsl.maxPerChain)
		}
		perIPSem := wsl.perIPSem[clientIP]
		perChainSem := wsl.perChainSem[chain]
		wsl.mu.Unlock()

		select {
		case perIPSem <- struct{}{}:
			// Acquired per-IP slot, now try per-chain slot
			select {
			case perChainSem <- struct{}{}:
				// Successfully acquired all slots
				return true
			default:
				// Failed to acquire per-chain slot, release per-IP and global slots
				<-perIPSem
				<-wsl.globalSem
				return false
			}
		default:
			// Failed to acquire per-IP slot, release global slot
			<-wsl.globalSem
			return false
		}
	default:
		// Failed to acquire global slot
		return false
	}
}

// ReleaseForChain releases a WebSocket connection slot for a specific chain
func (wsl *WebSocketLimiter) ReleaseForChain(clientIP, chain string) {
	wsl.mu.RLock()
	perIPSem := wsl.perIPSem[clientIP]
	perChainSem := wsl.perChainSem[chain]
	wsl.mu.RUnlock()

	if perChainSem != nil {
		select {
		case <-perChainSem:
			// Released per-chain slot
		default:
			// No slot to release
		}
	}

	if perIPSem != nil {
		select {
		case <-perIPSem:
			// Released per-IP slot
		default:
			// No slot to release
		}
	}

	select {
	case <-wsl.globalSem:
		// Released global slot
	default:
		// No slot to release
	}
}
