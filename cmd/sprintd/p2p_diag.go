package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// AttemptRecord holds one connection attempt result
type AttemptRecord struct {
	Address          string    `json:"address"`
	Timestamp        time.Time `json:"timestamp"`
	TcpSuccess       bool      `json:"tcp_success"`
	TcpError         string    `json:"tcp_error,omitempty"`
	HandshakeSuccess bool      `json:"handshake_success"`
	HandshakeError   string    `json:"handshake_error,omitempty"`
}

// Rolling buffer per protocol
type diagBuffer struct {
	Attempts    []AttemptRecord
	LastError   string
	DialedPeers []string
	mu          sync.Mutex
}

var (
	diagState = map[string]*diagBuffer{
		"bitcoin":  {Attempts: make([]AttemptRecord, 0, 50), DialedPeers: make([]string, 0, 20)},
		"ethereum": {Attempts: make([]AttemptRecord, 0, 50), DialedPeers: make([]string, 0, 20)},
		"solana":   {Attempts: make([]AttemptRecord, 0, 50), DialedPeers: make([]string, 0, 20)},
	}
)

// RecordAttempt safely appends to buffer for given chain
func RecordAttempt(protocol string, rec AttemptRecord) {
	buf, ok := diagState[protocol]
	if !ok {
		return
	}
	buf.mu.Lock()
	defer buf.mu.Unlock()

	if len(buf.Attempts) >= 50 {
		buf.Attempts = buf.Attempts[1:] // drop oldest
	}
	buf.Attempts = append(buf.Attempts, rec)

	if rec.HandshakeError != "" || rec.TcpError != "" {
		buf.LastError = rec.HandshakeError
		if buf.LastError == "" {
			buf.LastError = rec.TcpError
		} else if rec.TcpError != "" {
			buf.LastError += "; " + rec.TcpError
		}
	}
	if rec.Address != "" {
		buf.DialedPeers = append(buf.DialedPeers, rec.Address)
		if len(buf.DialedPeers) > 20 {
			buf.DialedPeers = buf.DialedPeers[1:]
		}
	}
}

// SetLastError sets the last error message for a protocol in diagnostics
func SetLastError(protocol string, msg string) {
	buf, ok := diagState[protocol]
	if !ok {
		return
	}
	buf.mu.Lock()
	buf.LastError = msg
	buf.mu.Unlock()
}

// snapshotBuffer returns a copy-safe view of the diag buffer for JSON
func snapshotBuffer(protocol string) map[string]interface{} {
	buf, ok := diagState[protocol]
	if !ok || buf == nil {
		return map[string]interface{}{
			"connection_attempts": []AttemptRecord{},
			"last_error":          "",
			"dialed_peers":        []string{},
		}
	}
	buf.mu.Lock()
	attempts := make([]AttemptRecord, len(buf.Attempts))
	copy(attempts, buf.Attempts)
	peers := make([]string, len(buf.DialedPeers))
	copy(peers, buf.DialedPeers)
	lastErr := buf.LastError
	buf.mu.Unlock()
	return map[string]interface{}{
		"connection_attempts": attempts,
		"last_error":          lastErr,
		"dialed_peers":        peers,
	}
}

// p2pDiagHandler returns peer diagnostics
func (s *Server) p2pDiagHandler(w http.ResponseWriter, r *http.Request) {
	// Build response from existing p2pClients map
	clients := map[string]interface{}{}

	// Known protocols
	protocols := []ProtocolType{ProtocolBitcoin, ProtocolEthereum, ProtocolSolana}
	for _, p := range protocols {
		var count int
		ids := []string{}
		if c := s.p2pClients[p]; c != nil {
			count = c.GetPeerCount()
			ids = c.GetPeerIDs()
		}
		snap := snapshotBuffer(string(p))
		backendStatus := "online"
		if count == 0 {
			// Graceful degradation signal for UI
			backendStatus = "fallback_rpc"
		}
		clients[string(p)] = map[string]interface{}{
			"peer_count": count,
			"peer_ids":   ids,
			"backend_status": backendStatus,
			// merge snapshot fields
			"connection_attempts": snap["connection_attempts"],
			"last_error":          snap["last_error"],
			"dialed_peers":        snap["dialed_peers"],
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"p2p_clients": clients,
		"ts":          time.Now().UTC().Format(time.RFC3339),
	})
}

