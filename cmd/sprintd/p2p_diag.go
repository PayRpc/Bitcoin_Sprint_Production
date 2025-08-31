package main

import (
	"net/http"
)

// p2pDiagHandler returns lightweight diagnostics for P2P clients
func (s *Server) p2pDiagHandler(w http.ResponseWriter, r *http.Request) {
	result := make(map[string]interface{})
	clients := make(map[string]interface{})

	for proto, uc := range s.p2pClients {
		if uc == nil {
			clients[string(proto)] = map[string]interface{}{"peer_count": 0}
			continue
		}

		clients[string(proto)] = map[string]interface{}{
			"peer_count": uc.GetPeerCount(),
			"peer_ids":   uc.GetPeerIDs(),
		}
	}

	result["p2p_clients"] = clients

	s.jsonResponse(w, http.StatusOK, result)
}
