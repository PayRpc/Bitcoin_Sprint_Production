// Package api provides the main HTTP API server for Bitcoin Sprint
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

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
	addr := fmt.Sprintf("%s:%d", s.cfg.APIHost, s.cfg.APIPort)
	s.logger.Info("Starting API server", zap.String("addr", addr))

	// Initialize mux if missing
	if s.httpMux == nil {
		s.httpMux = http.NewServeMux()
	}

	// Core routes (public)
	s.httpMux.HandleFunc("/health", s.healthHandler)
	s.httpMux.HandleFunc("/version", s.versionHandler)

	// Competitive advantage and universal API routes
	s.RegisterSprintValueRoutes()

	// Chain-aware router (e.g., /v1/btc/latest)
	s.httpMux.HandleFunc("/v1/", s.chainAwareHandler)

	// Enterprise endpoints
	s.InitializeEnterpriseManager()
	if s.enterpriseManager != nil {
		// System
		s.httpMux.HandleFunc("/api/v1/enterprise/system/fingerprint", s.enterpriseManager.handleSystemFingerprint)
		s.httpMux.HandleFunc("/api/v1/enterprise/system/temperature", s.enterpriseManager.handleCPUTemperature)
		// Entropy
		s.httpMux.HandleFunc("/api/v1/enterprise/entropy/fast", s.enterpriseManager.handleFastEntropy)
		s.httpMux.HandleFunc("/api/v1/enterprise/entropy/hybrid", s.enterpriseManager.handleHybridEntropy)
		// Secure buffer
		s.httpMux.HandleFunc("/api/v1/enterprise/buffer/new", s.enterpriseManager.handleNewSecureBuffer)
		// Audit
		s.httpMux.HandleFunc("/api/v1/enterprise/security/audit-status", s.enterpriseManager.handleAuditStatus)
		s.httpMux.HandleFunc("/api/v1/enterprise/security/audit/enable", s.enterpriseManager.handleEnableAudit)
		s.httpMux.HandleFunc("/api/v1/enterprise/security/audit/disable", s.enterpriseManager.handleDisableAudit)
	}

	// Wrap with security middleware
	handler := s.securityMiddleware(s.httpMux)

	s.srv = &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Graceful shutdown watcher
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.srv.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("HTTP server shutdown error", zap.Error(err))
		}
	}()

	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Error("HTTP server error", zap.Error(err))
	}
}
