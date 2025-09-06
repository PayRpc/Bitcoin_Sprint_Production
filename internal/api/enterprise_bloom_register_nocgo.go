//go:build !cgo
// +build !cgo

package api

import "go.uber.org/zap"

// RegisterBloomEndpoints is a no-op when cgo is disabled
func (esm *EnterpriseSecurityManager) RegisterBloomEndpoints() {
	esm.logger.Info("Bloom Filter endpoints unavailable (CGO disabled)",
		zap.String("note", "Enable CGO and rebuild to access bloom filter functionality"))
}
