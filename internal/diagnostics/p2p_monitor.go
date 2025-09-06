package diagnostics

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AttemptRecord holds one connection attempt result
type AttemptRecord struct {
	Address          string        `json:"address"`
	Timestamp        time.Time     `json:"timestamp"`
	TcpSuccess       bool          `json:"tcp_success"`
	TcpError         string        `json:"tcp_error,omitempty"`
	HandshakeSuccess bool          `json:"handshake_success"`
	HandshakeError   string        `json:"handshake_error,omitempty"`
	ConnectLatency   time.Duration `json:"connect_latency,omitempty"`
	ResponseTime     time.Duration `json:"response_time,omitempty"`
}

// P2PBuffer holds connection attempt history per protocol
type P2PBuffer struct {
	Attempts    []AttemptRecord `json:"attempts"`
	DialedPeers []string        `json:"dialed_peers"`
	mu          sync.RWMutex
}

// P2PMonitor provides production-ready P2P diagnostics
type P2PMonitor struct {
	diagnostics map[string]*P2PBuffer
	logger      *zap.Logger
	mu          sync.RWMutex
}

// NewP2PMonitor creates a production P2P monitor
func NewP2PMonitor(logger *zap.Logger) *P2PMonitor {
	return &P2PMonitor{
		diagnostics: map[string]*P2PBuffer{
			"bitcoin":  {Attempts: make([]AttemptRecord, 0, 50), DialedPeers: make([]string, 0, 20)},
			"ethereum": {Attempts: make([]AttemptRecord, 0, 50), DialedPeers: make([]string, 0, 20)},
			"solana":   {Attempts: make([]AttemptRecord, 0, 50), DialedPeers: make([]string, 0, 20)},
		},
		logger: logger,
	}
}

// RecordAttempt safely records a connection attempt
func (pm *P2PMonitor) RecordAttempt(protocol string, rec AttemptRecord) {
	pm.mu.RLock()
	buf, ok := pm.diagnostics[protocol]
	pm.mu.RUnlock()

	if !ok {
		pm.logger.Warn("Unknown protocol", zap.String("protocol", protocol))
		return
	}

	buf.mu.Lock()
	defer buf.mu.Unlock()

	// Maintain circular buffer
	if len(buf.Attempts) >= 50 {
		buf.Attempts = buf.Attempts[1:]
	}
	buf.Attempts = append(buf.Attempts, rec)
}

// GetStatusSnapshot returns current diagnostic data
func (pm *P2PMonitor) GetStatusSnapshot(protocol string) map[string]interface{} {
	pm.mu.RLock()
	buf, ok := pm.diagnostics[protocol]
	pm.mu.RUnlock()

	if !ok {
		return map[string]interface{}{
			"connection_attempts": []AttemptRecord{},
			"last_error":          "Unknown protocol",
			"dialed_peers":        []string{},
		}
	}

	buf.mu.RLock()
	defer buf.mu.RUnlock()

	attempts := make([]AttemptRecord, len(buf.Attempts))
	copy(attempts, buf.Attempts)

	peers := make([]string, len(buf.DialedPeers))
	copy(peers, buf.DialedPeers)

	var lastErr string
	for i := len(attempts) - 1; i >= 0; i-- {
		if attempts[i].TcpError != "" || attempts[i].HandshakeError != "" {
			if attempts[i].TcpError != "" {
				lastErr = attempts[i].TcpError
			} else {
				lastErr = attempts[i].HandshakeError
			}
			break
		}
	}

	return map[string]interface{}{
		"connection_attempts": attempts,
		"last_error":          lastErr,
		"dialed_peers":        peers,
	}
}

// P2PDiagnosticsHandler provides HTTP endpoint for P2P diagnostics
func (pm *P2PMonitor) P2PDiagnosticsHandler(w http.ResponseWriter, r *http.Request) {
	protocols := []string{"bitcoin", "ethereum", "solana"}
	clients := make(map[string]interface{})

	for _, protocol := range protocols {
		snap := pm.GetStatusSnapshot(protocol)
		clients[protocol] = map[string]interface{}{
			"peer_count":          0, // Will be populated by actual P2P clients
			"peer_ids":            []string{},
			"backend_status":      "fallback_rpc",
			"connection_attempts": snap["connection_attempts"],
			"last_error":          snap["last_error"],
			"dialed_peers":        snap["dialed_peers"],
		}
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"p2p_clients": clients,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		pm.logger.Error("Failed to encode P2P diagnostics", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
