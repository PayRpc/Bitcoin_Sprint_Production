// Bitcoin Sprint Enterprise Security Integration Demo
// This demonstrates how the SecureBuffer enterprise features integrate with the API layer

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/securebuf"
)

// ===== ENTERPRISE API DEMONSTRATION =====

// EnterpriseAPIDemo shows how the enterprise features integrate into the main API
type EnterpriseAPIDemo struct {
	mux *http.ServeMux
}

// NewEnterpriseAPIDemo creates a new demo server
func NewEnterpriseAPIDemo() *EnterpriseAPIDemo {
	demo := &EnterpriseAPIDemo{
		mux: http.NewServeMux(),
	}
	demo.registerRoutes()
	return demo
}

// registerRoutes registers all enterprise API endpoints
func (demo *EnterpriseAPIDemo) registerRoutes() {
	fmt.Println("🔐 Registering Enterprise Security API Routes:")

	// Entropy endpoints
	demo.mux.HandleFunc("/api/v1/enterprise/entropy/fast", demo.handleFastEntropy)
	demo.mux.HandleFunc("/api/v1/enterprise/entropy/hybrid", demo.handleHybridEntropy)

	// System information
	demo.mux.HandleFunc("/api/v1/enterprise/system/fingerprint", demo.handleSystemFingerprint)
	demo.mux.HandleFunc("/api/v1/enterprise/system/temperature", demo.handleCPUTemperature)

	// Buffer management
	demo.mux.HandleFunc("/api/v1/enterprise/buffer/new", demo.handleNewSecureBuffer)

	// Audit and compliance
	demo.mux.HandleFunc("/api/v1/enterprise/audit/status", demo.handleAuditStatus)
	demo.mux.HandleFunc("/api/v1/enterprise/audit/enable", demo.handleEnableAudit)
	demo.mux.HandleFunc("/api/v1/enterprise/audit/disable", demo.handleDisableAudit)
	demo.mux.HandleFunc("/api/v1/enterprise/policy", demo.handleSetPolicy)
	demo.mux.HandleFunc("/api/v1/enterprise/compliance", demo.handleComplianceReport)

	// Bitcoin bloom filters
	demo.mux.HandleFunc("/api/v1/enterprise/bloom/new", demo.handleNewBloomFilter)
	demo.mux.HandleFunc("/api/v1/enterprise/bloom/stats", demo.handleBloomStats)

	// Root endpoint
	demo.mux.HandleFunc("/", demo.handleRoot)

	endpoints := []string{
		"GET  /api/v1/enterprise/system/fingerprint - Get system hardware fingerprint",
		"GET  /api/v1/enterprise/system/temperature - Get CPU temperature for entropy",
		"POST /api/v1/enterprise/entropy/fast - Generate fast hardware entropy",
		"POST /api/v1/enterprise/entropy/hybrid - Generate entropy with Bitcoin headers",
		"POST /api/v1/enterprise/buffer/new - Create enterprise secure buffer",
		"GET  /api/v1/enterprise/audit/status - Check audit logging status",
		"POST /api/v1/enterprise/audit/enable - Enable enterprise audit logging",
		"POST /api/v1/enterprise/audit/disable - Disable enterprise audit logging",
		"POST /api/v1/enterprise/policy - Set enterprise security policy",
		"GET  /api/v1/enterprise/compliance - Get compliance report",
		"POST /api/v1/enterprise/bloom/new - Create Bitcoin bloom filter",
		"GET  /api/v1/enterprise/bloom/stats - Get bloom filter statistics",
	}

	for _, endpoint := range endpoints {
		fmt.Printf("   📍 %s\n", endpoint)
	}
}

// ===== API HANDLERS =====

func (demo *EnterpriseAPIDemo) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service":   "Bitcoin Sprint Enterprise Security API",
		"version":   "1.0.0",
		"status":    "running",
		"timestamp": time.Now(),
		"endpoints": []string{
			"/api/v1/enterprise/entropy/fast",
			"/api/v1/enterprise/entropy/hybrid",
			"/api/v1/enterprise/system/fingerprint",
			"/api/v1/enterprise/system/temperature",
			"/api/v1/enterprise/buffer/new",
			"/api/v1/enterprise/audit/status",
			"/api/v1/enterprise/policy",
			"/api/v1/enterprise/compliance",
			"/api/v1/enterprise/bloom/new",
		},
	})
}

func (demo *EnterpriseAPIDemo) handleFastEntropy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		demo.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	entropy, err := securebuf.FastEntropy()
	if err != nil {
		demo.jsonError(w, http.StatusInternalServerError, "Failed to generate entropy")
		return
	}

	demo.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"entropy":   fmt.Sprintf("%x", entropy),
		"size":      len(entropy),
		"source":    "hardware",
		"timestamp": time.Now(),
	})
}

func (demo *EnterpriseAPIDemo) handleHybridEntropy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		demo.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// For demo, use mock headers
	mockHeaders := [][]byte{
		make([]byte, 80),
		make([]byte, 80),
	}

	entropy, err := securebuf.HybridEntropy(mockHeaders)
	if err != nil {
		demo.jsonError(w, http.StatusInternalServerError, "Failed to generate hybrid entropy")
		return
	}

	demo.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"entropy":      fmt.Sprintf("%x", entropy),
		"size":         len(entropy),
		"headers_used": len(mockHeaders),
		"source":       "hybrid",
		"timestamp":    time.Now(),
	})
}

func (demo *EnterpriseAPIDemo) handleSystemFingerprint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		demo.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	fingerprint, err := securebuf.SystemFingerprint()
	if err != nil {
		demo.jsonError(w, http.StatusInternalServerError, "Failed to get system fingerprint")
		return
	}

	demo.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"fingerprint": fmt.Sprintf("%x", fingerprint),
		"timestamp":   time.Now(),
	})
}

func (demo *EnterpriseAPIDemo) handleCPUTemperature(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		demo.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	temperature, err := securebuf.GetCPUTemperature()
	if err != nil {
		demo.jsonError(w, http.StatusInternalServerError, "Failed to get CPU temperature")
		return
	}

	demo.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"temperature_celsius": temperature,
		"timestamp":           time.Now(),
	})
}

func (demo *EnterpriseAPIDemo) handleNewSecureBuffer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		demo.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// For demo, create a buffer with fast entropy
	buffer, err := securebuf.NewWithFastEntropy(256)
	if err != nil {
		demo.jsonError(w, http.StatusInternalServerError, "Failed to create secure buffer")
		return
	}
	defer buffer.Free()

	demo.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"buffer_id":      fmt.Sprintf("buf_%d", time.Now().UnixNano()),
		"size":           buffer.Capacity(),
		"security_level": "enterprise",
		"entropy_filled": true,
		"timestamp":      time.Now(),
	})
}

func (demo *EnterpriseAPIDemo) handleAuditStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		demo.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	enabled := securebuf.IsAuditLoggingEnabled()

	demo.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"audit_enabled": enabled,
		"timestamp":     time.Now(),
	})
}

func (demo *EnterpriseAPIDemo) handleEnableAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		demo.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	logPath := "/tmp/bitcoin-sprint-audit.log"
	if err := securebuf.EnableAuditLogging(logPath); err != nil {
		demo.jsonError(w, http.StatusInternalServerError, "Failed to enable audit logging")
		return
	}

	demo.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":    "enabled",
		"log_path":  logPath,
		"timestamp": time.Now(),
	})
}

func (demo *EnterpriseAPIDemo) handleDisableAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		demo.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if err := securebuf.DisableAuditLogging(); err != nil {
		demo.jsonError(w, http.StatusInternalServerError, "Failed to disable audit logging")
		return
	}

	demo.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":    "disabled",
		"timestamp": time.Now(),
	})
}

func (demo *EnterpriseAPIDemo) handleSetPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		demo.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	policyJSON := `{
		"security_level": "enterprise",
		"max_buffer_lifetime": 86400,
		"require_memory_lock": true,
		"enable_tamper_detection": true,
		"audit_all_operations": true
	}`

	if err := securebuf.SetEnterprisePolicy(policyJSON); err != nil {
		demo.jsonError(w, http.StatusInternalServerError, "Failed to set enterprise policy")
		return
	}

	demo.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":    "policy_set",
		"timestamp": time.Now(),
	})
}

func (demo *EnterpriseAPIDemo) handleComplianceReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		demo.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	report, err := securebuf.GetComplianceReport()
	if err != nil {
		demo.jsonError(w, http.StatusInternalServerError, "Failed to get compliance report")
		return
	}

	demo.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"compliance_report": report,
		"timestamp":         time.Now(),
	})
}

func (demo *EnterpriseAPIDemo) handleNewBloomFilter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		demo.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	filter, err := securebuf.NewBitcoinBloomFilterDefault()
	if err != nil {
		demo.jsonError(w, http.StatusInternalServerError, "Failed to create bloom filter")
		return
	}
	defer filter.Free()

	demo.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"filter_id": fmt.Sprintf("bloom_%d", time.Now().UnixNano()),
		"type":      "bitcoin_utxo",
		"optimized": true,
		"timestamp": time.Now(),
	})
}

func (demo *EnterpriseAPIDemo) handleBloomStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		demo.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	filter, err := securebuf.NewBitcoinBloomFilterDefault()
	if err != nil {
		demo.jsonError(w, http.StatusInternalServerError, "Failed to create bloom filter")
		return
	}
	defer filter.Free()

	stats, err := filter.GetStats()
	if err != nil {
		demo.jsonError(w, http.StatusInternalServerError, "Failed to get bloom filter stats")
		return
	}

	demo.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"item_count":             stats.ItemCount,
		"false_positive_rate":    stats.TheoreticalFPRate,
		"memory_usage_bytes":     stats.MemoryUsageBytes,
		"optimal_hash_functions": stats.OptimalHashFunctions,
		"timestamp":              time.Now(),
	})
}

// ===== UTILITY METHODS =====

func (demo *EnterpriseAPIDemo) jsonResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (demo *EnterpriseAPIDemo) jsonError(w http.ResponseWriter, statusCode int, message string) {
	demo.jsonResponse(w, statusCode, map[string]interface{}{
		"error":     message,
		"timestamp": time.Now(),
	})
}

// ServeHTTP implements http.Handler
func (demo *EnterpriseAPIDemo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	demo.mux.ServeHTTP(w, r)
}

// ===== MAIN DEMONSTRATION =====

func main() {
	fmt.Println("🚀 Bitcoin Sprint Enterprise Security API Integration Demo")
	fmt.Println("===========================================================")

	// Create the enterprise API demo
	apiDemo := NewEnterpriseAPIDemo()

	// Start the server
	addr := ":9090"
	fmt.Printf("\n🌐 Starting Enterprise API Server on %s\n", addr)
	fmt.Println("\n📚 Available Endpoints:")
	fmt.Println("   🏠 GET  / - API information")
	fmt.Println("   🔐 POST /api/v1/enterprise/entropy/fast - Fast entropy generation")
	fmt.Println("   🔗 POST /api/v1/enterprise/entropy/hybrid - Hybrid entropy with Bitcoin headers")
	fmt.Println("   🖥️  GET  /api/v1/enterprise/system/fingerprint - System hardware fingerprint")
	fmt.Println("   🌡️  GET  /api/v1/enterprise/system/temperature - CPU temperature")
	fmt.Println("   🛡️  POST /api/v1/enterprise/buffer/new - Create secure buffer")
	fmt.Println("   📋 GET  /api/v1/enterprise/audit/status - Audit status")
	fmt.Println("   ✅ POST /api/v1/enterprise/audit/enable - Enable audit logging")
	fmt.Println("   ❌ POST /api/v1/enterprise/audit/disable - Disable audit logging")
	fmt.Println("   ⚙️  POST /api/v1/enterprise/policy - Set security policy")
	fmt.Println("   📊 GET  /api/v1/enterprise/compliance - Compliance report")
	fmt.Println("   🌸 POST /api/v1/enterprise/bloom/new - Create Bitcoin bloom filter")
	fmt.Println("   📈 GET  /api/v1/enterprise/bloom/stats - Bloom filter statistics")

	fmt.Printf("\n🔗 Server running at: http://localhost%s\n", addr)
	fmt.Println("💡 Try: curl http://localhost:9090/api/v1/enterprise/system/fingerprint")
	fmt.Println("💡 Try: curl -X POST http://localhost:9090/api/v1/enterprise/entropy/fast")

	if err := http.ListenAndServe(addr, apiDemo); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
