// Package api provides the main HTTP API server for Bitcoin Sprint
package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/cache"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
	"go.uber.org/zap"
)

// ===== ROUTE REGISTRATION =====

// RegisterSprintValueRoutes registers the competitive advantage endpoints
func (s *Server) RegisterSprintValueRoutes() {
	// Universal multi-chain API endpoints demonstrating Sprint's competitive advantages
	
	// Core value proposition endpoints
	if s.httpMux != nil {
		// Universal chain endpoint - single API for all chains
		s.httpMux.HandleFunc("/api/v1/universal/", s.universalChainHandler)
		
		// Performance monitoring endpoints
		s.httpMux.HandleFunc("/api/v1/sprint/latency-stats", s.latencyStatsHandler)
		s.httpMux.HandleFunc("/api/v1/sprint/cache-stats", s.cacheStatsHandler)
		s.httpMux.HandleFunc("/api/v1/sprint/tier-comparison", s.tierComparisonHandler)
		
		// Value demonstration endpoint
		s.httpMux.HandleFunc("/api/v1/sprint/value", SprintValueHandler)
		
		s.logger.Info("Sprint competitive advantage routes registered",
			zap.String("universal_endpoint", "/api/v1/universal/{chain}/{method}"),
			zap.String("value_props", "flat_p99,unified_api,predictive_cache,enterprise_tiers"))
	}
}

// ===== SERVER LIFECYCLE METHODS =====

// InitializeEnterpriseManager initializes the enterprise security manager
func (s *Server) InitializeEnterpriseManager() {
	if s.enterpriseManager == nil {
		s.enterpriseManager = NewEnterpriseSecurityManager(s, s.logger)
		s.enterpriseManager.RegisterEnterpriseRoutes()
	}
}

// Run starts the API server and blocks until shutdown
func (s *Server) Run(ctx context.Context) {
	s.logger.Info("Starting API server",
		zap.String("addr", fmt.Sprintf("%s:%d", s.cfg.APIHost, s.cfg.APIPort)))
	
	// TODO: Implement proper server startup logic
	// This is a placeholder until the API module is fully restructured
	s.logger.Info("API server run method called - implement server startup")
}
