package api

import (
	"context"
	"fmt"
	"time"
)

// handleSolanaRequest handles Solana-specific requests using the real relay
func (s *Server) handleSolanaRequest(method string, start time.Time) map[string]interface{} {
	response := map[string]interface{}{
		"chain":     "solana",
		"method":    method,
		"timestamp": start.Unix(),
		"data":      nil,
		"error":     nil,
	}

	// Ensure Solana relay is connected
	if s.solanaRelay != nil && !s.solanaRelay.IsConnected() {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()
		if err := s.solanaRelay.Connect(ctx); err != nil {
			response["error"] = fmt.Sprintf("Failed to connect to Solana network: %v", err)
			return response
		}
	}

	// Handle specific methods
	switch method {
	case "ping":
		ok := true
		if s.solanaRelay != nil && !s.solanaRelay.IsConnected() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := s.solanaRelay.Connect(ctx); err != nil {
				ok = false
				response["error"] = fmt.Sprintf("Ping failed: %v", err)
			}
		}
		response["data"] = map[string]interface{}{
			"ok":         ok,
			"peer_count": s.solanaRelay.GetPeerCount(),
		}
	case "latest", "latest_block":
		if block, err := s.solanaRelay.GetLatestBlock(); err != nil {
			response["error"] = fmt.Sprintf("Failed to get latest block: %v", err)
		} else {
			response["data"] = block
		}
	case "status", "network_info":
		if info, err := s.solanaRelay.GetNetworkInfo(); err != nil {
			response["error"] = fmt.Sprintf("Failed to get network info: %v", err)
		} else {
			response["data"] = info
		}
	case "peers", "peer_count":
		peerCount := s.solanaRelay.GetPeerCount()
		response["data"] = map[string]interface{}{
			"peer_count": peerCount,
		}
	case "sync", "sync_status":
		if status, err := s.solanaRelay.GetSyncStatus(); err != nil {
			response["error"] = fmt.Sprintf("Failed to get sync status: %v", err)
		} else {
			response["data"] = status
		}
	default:
		response["error"] = fmt.Sprintf("Unknown method: %s", method)
	}

	// Add performance metrics
	response["performance"] = map[string]interface{}{
		"response_time": fmt.Sprintf("%.2fms", float64(time.Since(start).Nanoseconds())/1e6),
		"real_data":     true,
		"network":       "solana_mainnet",
	}

	return response
}
