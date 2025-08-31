package diag

import (
    "sync"
    "time"
)

// AttemptRecord captures a single connection attempt
type AttemptRecord struct {
    Timestamp        time.Time `json:"timestamp"`
    Address          string    `json:"address"`
    TcpSuccess       bool      `json:"tcp_success"`
    TcpError         string    `json:"tcp_error,omitempty"`
    HandshakeSuccess bool      `json:"handshake_success"`
    HandshakeError   string    `json:"handshake_error,omitempty"`
}

type attemptBuffer struct {
    mu    sync.RWMutex
    items []AttemptRecord
    cap   int
}

var (
    buffersMu sync.RWMutex
    buffers   = make(map[string]*attemptBuffer)
    defaultCap = 100
)

func getBuffer(network string) *attemptBuffer {
    buffersMu.RLock()
    b := buffers[network]
    buffersMu.RUnlock()
    if b != nil {
        return b
    }

    buffersMu.Lock()
    defer buffersMu.Unlock()
    // double-check
    if buffers[network] == nil {
        buffers[network] = &attemptBuffer{items: make([]AttemptRecord, 0, defaultCap), cap: defaultCap}
    }
    return buffers[network]
}

// RecordAttempt appends a record to the rolling buffer for a network
func RecordAttempt(network string, rec AttemptRecord) {
    b := getBuffer(network)
    b.mu.Lock()
    defer b.mu.Unlock()
    if len(b.items) >= b.cap {
        // drop oldest
        copy(b.items[0:], b.items[1:])
        b.items[b.cap-1] = rec
    } else {
        b.items = append(b.items, rec)
    }
}

// GetAttempts returns a copy of recent attempts for a network
func GetAttempts(network string) []AttemptRecord {
    b := getBuffer(network)
    b.mu.RLock()
    defer b.mu.RUnlock()
    out := make([]AttemptRecord, len(b.items))
    copy(out, b.items)
    return out
}

// GetLastError returns the most recent non-empty error message for the network (tcp or handshake)
func GetLastError(network string) string {
    b := getBuffer(network)
    b.mu.RLock()
    defer b.mu.RUnlock()
    for i := len(b.items)-1; i >= 0; i-- {
        if b.items[i].HandshakeError != "" {
            return b.items[i].HandshakeError
        }
        if b.items[i].TcpError != "" {
            return b.items[i].TcpError
        }
    }
    return ""
}

// GetDialedPeers returns unique addresses dialed recently for the network
func GetDialedPeers(network string) []string {
    b := getBuffer(network)
    b.mu.RLock()
    defer b.mu.RUnlock()
    set := make(map[string]struct{})
    for _, it := range b.items {
        set[it.Address] = struct{}{}
    }
    out := make([]string, 0, len(set))
    for k := range set {
        out = append(out, k)
    }
    return out
}
