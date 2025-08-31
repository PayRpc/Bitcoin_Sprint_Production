package main

import (
	"net/http"
	diagpkg "github.com/PayRpc/Bitcoin-Sprint/internal/p2p/diag"
)

// p2pDiagHandler returns lightweight diagnostics for P2P clients
func (s *Server) p2pDiagHandler(w http.ResponseWriter, r *http.Request) {
	result := make(map[string]interface{})
	clients := make(map[string]interface{})

	for proto, uc := range s.p2pClients {
		netName := string(proto)
		entry := map[string]interface{}{
			"peer_count": uc.GetPeerCount(),
			"peer_ids":   uc.GetPeerIDs(),
			// default placeholders for new fields
			"connection_attempts": diagpkg.GetAttempts(netName),
			"last_error":          diagpkg.GetLastError(netName),
			"dialed_peers":        diagpkg.GetDialedPeers(netName),
		}
		clients[netName] = entry
	}

	result["p2p_clients"] = clients

	s.jsonResponse(w, http.StatusOK, result)
}
