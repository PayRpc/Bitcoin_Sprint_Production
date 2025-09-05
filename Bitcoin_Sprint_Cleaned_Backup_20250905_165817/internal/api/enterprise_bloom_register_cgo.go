//go:build cgo
// +build cgo

package api

import "go.uber.org/zap"

// RegisterEnterpriseRoutes appends bloom endpoints when cgo is enabled
func (esm *EnterpriseSecurityManager) RegisterEnterpriseRoutes() {
    esm.logger.Info("Enterprise Security API endpoints available:",
        zap.Strings("endpoints", []string{
            "POST /api/v1/enterprise/entropy/fast",
            "POST /api/v1/enterprise/entropy/hybrid",
            "GET /api/v1/enterprise/system/fingerprint",
            "GET /api/v1/enterprise/system/temperature",
            "POST /api/v1/enterprise/buffer/new",
            "GET /api/v1/enterprise/security/audit-status",
            "POST /api/v1/enterprise/security/audit/enable",
            "POST /api/v1/enterprise/security/audit/disable",
            "POST /api/v1/enterprise/security/policy",
            "GET /api/v1/enterprise/security/compliance-report",
            // Bloom endpoints available only with cgo
            "POST /api/v1/enterprise/bloom/new",
            "POST /api/v1/enterprise/bloom/insert-utxo",
            "POST /api/v1/enterprise/bloom/check-utxo",
            "GET /api/v1/enterprise/bloom/stats",
        }))
}
