//go:build cgo
// +build cgo

package api

import "go.uber.org/zap"

// RegisterBloomEndpoints appends bloom-specific endpoints when cgo is enabled
func (esm *EnterpriseSecurityManager) RegisterBloomEndpoints() {
	esm.logger.Info("Bloom Filter Enterprise endpoints available (CGO enabled):",
		zap.Strings("bloom_endpoints", []string{
			"POST /api/v1/enterprise/bloom/new",
			"POST /api/v1/enterprise/bloom/insert-utxo",
			"POST /api/v1/enterprise/bloom/check-utxo",
			"GET /api/v1/enterprise/bloom/stats",
		}))
}
