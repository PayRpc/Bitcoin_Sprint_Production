// Package api provides enterprise security and audit endpoints
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/securebuf"
	"go.uber.org/zap"
)

// ===== ENTERPRISE SECURITY API ENDPOINTS =====

// EnterpriseSecurityManager handles enterprise security features
type EnterpriseSecurityManager struct {
	logger *zap.Logger
	server *Server
}

// NewEnterpriseSecurityManager creates a new enterprise security manager
func NewEnterpriseSecurityManager(server *Server, logger *zap.Logger) *EnterpriseSecurityManager {
	return &EnterpriseSecurityManager{
		logger: logger,
		server: server,
	}
}

// RegisterEnterpriseRoutes registers all enterprise security endpoints
func (esm *EnterpriseSecurityManager) RegisterEnterpriseRoutes() {
	// These would be registered with the router when available
	// For now, document the enterprise endpoints
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
		}))
	
	// Register bloom endpoints if CGO is enabled
	esm.RegisterBloomEndpoints()
}

// === ENTROPY ENDPOINTS ===

// FastEntropyRequest represents the request structure for fast entropy
type FastEntropyRequest struct {
	Size int `json:"size"`
}

// FastEntropyResponse represents the response structure for fast entropy
type FastEntropyResponse struct {
	Entropy   string    `json:"entropy"`
	Size      int       `json:"size"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
}

// handleFastEntropy generates fast entropy using hardware sources
func (esm *EnterpriseSecurityManager) handleFastEntropy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		esm.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req FastEntropyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		esm.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate size
	if req.Size <= 0 || req.Size > 1024 {
		esm.jsonError(w, http.StatusBadRequest, "Size must be between 1 and 1024 bytes")
		return
	}

	// Generate fast entropy
	entropy, err := securebuf.FastEntropy()
	if err != nil {
		esm.logger.Error("Failed to generate fast entropy", zap.Error(err))
		esm.jsonError(w, http.StatusInternalServerError, "Failed to generate entropy")
		return
	}

	// Trim to requested size if needed
	if len(entropy) > req.Size {
		entropy = entropy[:req.Size]
	}

	response := FastEntropyResponse{
		Entropy:   fmt.Sprintf("%x", entropy),
		Size:      len(entropy),
		Timestamp: time.Now(),
		Source:    "hardware",
	}

	esm.jsonResponse(w, http.StatusOK, response)
}

// HybridEntropyRequest represents the request for hybrid entropy with Bitcoin headers
type HybridEntropyRequest struct {
	Headers []string `json:"headers"` // Hex-encoded Bitcoin block headers
}

// HybridEntropyResponse represents the response for hybrid entropy
type HybridEntropyResponse struct {
	Entropy     string    `json:"entropy"`
	HeaderCount int       `json:"header_count"`
	Timestamp   time.Time `json:"timestamp"`
	Source      string    `json:"source"`
}

// handleHybridEntropy generates entropy using system sources mixed with Bitcoin headers
func (esm *EnterpriseSecurityManager) handleHybridEntropy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		esm.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req HybridEntropyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		esm.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Convert hex headers to byte arrays
	headers := make([][]byte, len(req.Headers))
	for i, hexHeader := range req.Headers {
		header := make([]byte, 80) // Bitcoin headers are 80 bytes
		if len(hexHeader) != 160 { // 80 bytes * 2 hex chars
			esm.jsonError(w, http.StatusBadRequest, fmt.Sprintf("Header %d must be 160 hex characters (80 bytes)", i))
			return
		}

		for j := 0; j < 80; j++ {
			hexByte := hexHeader[j*2 : j*2+2]
			if n, err := strconv.ParseUint(hexByte, 16, 8); err != nil {
				esm.jsonError(w, http.StatusBadRequest, fmt.Sprintf("Invalid hex in header %d", i))
				return
			} else {
				header[j] = byte(n)
			}
		}
		headers[i] = header
	}

	// Generate hybrid entropy
	entropy, err := securebuf.HybridEntropy(headers)
	if err != nil {
		esm.logger.Error("Failed to generate hybrid entropy", zap.Error(err))
		esm.jsonError(w, http.StatusInternalServerError, "Failed to generate hybrid entropy")
		return
	}

	response := HybridEntropyResponse{
		Entropy:     fmt.Sprintf("%x", entropy),
		HeaderCount: len(headers),
		Timestamp:   time.Now(),
		Source:      "hybrid",
	}

	esm.jsonResponse(w, http.StatusOK, response)
}

// === SYSTEM INFORMATION ENDPOINTS ===

// SystemFingerprintResponse represents the system fingerprint response
type SystemFingerprintResponse struct {
	Fingerprint string    `json:"fingerprint"`
	Timestamp   time.Time `json:"timestamp"`
}

// handleSystemFingerprint gets unique system hardware fingerprint
func (esm *EnterpriseSecurityManager) handleSystemFingerprint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		esm.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	fingerprint, err := securebuf.SystemFingerprint()
	if err != nil {
		esm.logger.Error("Failed to get system fingerprint", zap.Error(err))
		esm.jsonError(w, http.StatusInternalServerError, "Failed to get system fingerprint")
		return
	}

	response := SystemFingerprintResponse{
		Fingerprint: fmt.Sprintf("%x", fingerprint),
		Timestamp:   time.Now(),
	}

	esm.jsonResponse(w, http.StatusOK, response)
}

// CPUTemperatureResponse represents the CPU temperature response
type CPUTemperatureResponse struct {
	Temperature float64   `json:"temperature_celsius"`
	Timestamp   time.Time `json:"timestamp"`
}

// handleCPUTemperature gets current CPU temperature for entropy purposes
func (esm *EnterpriseSecurityManager) handleCPUTemperature(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		esm.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	temperature, err := securebuf.GetCPUTemperature()
	if err != nil {
		esm.logger.Error("Failed to get CPU temperature", zap.Error(err))
		esm.jsonError(w, http.StatusInternalServerError, "Failed to get CPU temperature")
		return
	}

	response := CPUTemperatureResponse{
		Temperature: float64(temperature),
		Timestamp:   time.Now(),
	}

	esm.jsonResponse(w, http.StatusOK, response)
}

// === SECURE BUFFER ENDPOINTS ===

// SecureBufferRequest represents a request to create a secure buffer
type SecureBufferRequest struct {
	Size            int    `json:"size"`
	SecurityLevel   string `json:"security_level"`
	FillWithEntropy bool   `json:"fill_with_entropy"`
}

// SecureBufferResponse represents the response for secure buffer operations
type SecureBufferResponse struct {
	BufferID      string    `json:"buffer_id"`
	Size          int       `json:"size"`
	SecurityLevel string    `json:"security_level"`
	Timestamp     time.Time `json:"timestamp"`
}

// handleNewSecureBuffer creates a new secure buffer with specified parameters
func (esm *EnterpriseSecurityManager) handleNewSecureBuffer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		esm.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req SecureBufferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		esm.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate size
	if req.Size <= 0 || req.Size > 10*1024*1024 { // Max 10MB
		esm.jsonError(w, http.StatusBadRequest, "Size must be between 1 byte and 10MB")
		return
	}

	// Parse security level
	var secLevel securebuf.SecurityLevel
	switch req.SecurityLevel {
	case "standard":
		secLevel = securebuf.SecurityStandard
	case "high":
		secLevel = securebuf.SecurityHigh
	case "enterprise":
		secLevel = securebuf.SecurityEnterprise
	case "forensic_resistant":
		secLevel = securebuf.SecurityForensicResistant
	case "hardware":
		secLevel = securebuf.SecurityHardware
	default:
		secLevel = securebuf.SecurityStandard
	}

	// Create buffer (handle both Buffer and EnterpriseBuffer types)
	if req.FillWithEntropy {
		buf, err := securebuf.NewWithFastEntropy(req.Size)
		if err != nil {
			esm.logger.Error("Failed to create secure buffer", zap.Error(err))
			esm.jsonError(w, http.StatusInternalServerError, "Failed to create secure buffer")
			return
		}
		defer buf.Free()
	} else {
		entBuf, err := securebuf.NewWithSecurityLevel(req.Size, secLevel)
		if err != nil {
			esm.logger.Error("Failed to create secure buffer", zap.Error(err))
			esm.jsonError(w, http.StatusInternalServerError, "Failed to create secure buffer")
			return
		}
		defer entBuf.Free()
	}

	response := SecureBufferResponse{
		BufferID:      fmt.Sprintf("buf_%d", time.Now().UnixNano()),
		Size:          req.Size,
		SecurityLevel: req.SecurityLevel,
		Timestamp:     time.Now(),
	}

	esm.jsonResponse(w, http.StatusOK, response)
}

// === AUDIT AND COMPLIANCE ENDPOINTS ===

// AuditStatusResponse represents the audit logging status
type AuditStatusResponse struct {
	Enabled   bool      `json:"enabled"`
	Timestamp time.Time `json:"timestamp"`
}

// handleAuditStatus gets current audit logging status
func (esm *EnterpriseSecurityManager) handleAuditStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		esm.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	enabled := securebuf.IsAuditLoggingEnabled()

	response := AuditStatusResponse{
		Enabled:   enabled,
		Timestamp: time.Now(),
	}

	esm.jsonResponse(w, http.StatusOK, response)
}

// AuditEnableRequest represents a request to enable audit logging
type AuditEnableRequest struct {
	LogPath string `json:"log_path"`
}

// handleEnableAudit enables enterprise audit logging
func (esm *EnterpriseSecurityManager) handleEnableAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		esm.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req AuditEnableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		esm.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.LogPath == "" {
		req.LogPath = "/var/log/bitcoin-sprint/enterprise-audit.log"
	}

	if err := securebuf.EnableAuditLogging(req.LogPath); err != nil {
		esm.logger.Error("Failed to enable audit logging", zap.Error(err))
		esm.jsonError(w, http.StatusInternalServerError, "Failed to enable audit logging")
		return
	}

	esm.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":    "enabled",
		"log_path":  req.LogPath,
		"timestamp": time.Now(),
	})
}

// handleDisableAudit disables enterprise audit logging
func (esm *EnterpriseSecurityManager) handleDisableAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		esm.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if err := securebuf.DisableAuditLogging(); err != nil {
		esm.logger.Error("Failed to disable audit logging", zap.Error(err))
		esm.jsonError(w, http.StatusInternalServerError, "Failed to disable audit logging")
		return
	}

	esm.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":    "disabled",
		"timestamp": time.Now(),
	})
}

// EnterprisePolicy represents an enterprise security policy
type EnterprisePolicy struct {
	MaxBufferLifetime       int  `json:"max_buffer_lifetime"`
	RequireMemoryLock       bool `json:"require_memory_lock"`
	EnableTamperDetection   bool `json:"enable_tamper_detection"`
	AuditAllOperations      bool `json:"audit_all_operations"`
	SideChannelProtection   bool `json:"side_channel_protection"`
	HardwareBindingRequired bool `json:"hardware_binding_required"`
}

// handleSetPolicy sets enterprise security policy
func (esm *EnterpriseSecurityManager) handleSetPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		esm.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var policy EnterprisePolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		esm.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Convert to JSON string for the securebuf API
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		esm.jsonError(w, http.StatusInternalServerError, "Failed to serialize policy")
		return
	}

	if err := securebuf.SetEnterprisePolicy(string(policyJSON)); err != nil {
		esm.logger.Error("Failed to set enterprise policy", zap.Error(err))
		esm.jsonError(w, http.StatusInternalServerError, "Failed to set enterprise policy")
		return
	}

	esm.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":    "policy_set",
		"policy":    policy,
		"timestamp": time.Now(),
	})
}

// handleComplianceReport gets enterprise compliance report
func (esm *EnterpriseSecurityManager) handleComplianceReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		esm.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	report, err := securebuf.GetComplianceReport()
	if err != nil {
		esm.logger.Error("Failed to get compliance report", zap.Error(err))
		esm.jsonError(w, http.StatusInternalServerError, "Failed to get compliance report")
		return
	}

	// Parse the JSON report
	var reportData interface{}
	if err := json.Unmarshal([]byte(report), &reportData); err != nil {
		// If parsing fails, return as string
		esm.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"report":    report,
			"timestamp": time.Now(),
		})
		return
	}

	esm.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"report":    reportData,
		"timestamp": time.Now(),
	})
}

// === BITCOIN BLOOM FILTER ENDPOINTS ===
// Bloom filter endpoints are available only when built with cgo. See enterprise_bloom_cgo.go.

// === UTILITY METHODS ===

// jsonResponse sends a JSON response
func (esm *EnterpriseSecurityManager) jsonResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// jsonError sends a JSON error response
func (esm *EnterpriseSecurityManager) jsonError(w http.ResponseWriter, statusCode int, message string) {
	esm.jsonResponse(w, statusCode, map[string]string{
		"error":     message,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
