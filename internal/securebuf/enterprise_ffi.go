//go:build cgo
// +build cgo

package securebuf

// Enterprise SecureBuffer FFI - Complete API integration
// This file provides access to all enterprise features from the Rust library

/*
#cgo LDFLAGS: -L../../secure/rust/target/x86_64-pc-windows-gnu/release -lsecurebuffer
#include "../../secure/rust/include/securebuffer.h"
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"
)

// SecurityLevel represents the security level for buffers
type SecurityLevel int

const (
	SecurityStandard          SecurityLevel = C.SECUREBUFFER_SECURITY_STANDARD
	SecurityHigh              SecurityLevel = C.SECUREBUFFER_SECURITY_HIGH
	SecurityEnterprise        SecurityLevel = C.SECUREBUFFER_SECURITY_ENTERPRISE
	SecurityForensicResistant SecurityLevel = C.SECUREBUFFER_SECURITY_FORENSIC_RESISTANT
	SecurityHardware          SecurityLevel = C.SECUREBUFFER_SECURITY_HARDWARE
)

// HashAlgorithm represents supported hash algorithms
type HashAlgorithm int

const (
	HashSHA256 HashAlgorithm = C.SECUREBUFFER_HASH_SHA256
	HashSHA512 HashAlgorithm = C.SECUREBUFFER_HASH_SHA512
	HashBLAKE3 HashAlgorithm = C.SECUREBUFFER_HASH_BLAKE3
)

// EnterpriseBuffer extends Buffer with enterprise features
type EnterpriseBuffer struct {
	*Buffer
	securityLevel SecurityLevel
}

// NewWithSecurityLevel creates a new secure buffer with specified security level
func NewWithSecurityLevel(capacity int, level SecurityLevel) (*EnterpriseBuffer, error) {
	if capacity <= 0 {
		return nil, errors.New("invalid capacity: must be positive")
	}

	handle := C.securebuffer_new_with_security_level(C.size_t(capacity), C.SecureBufferSecurityLevel(level))
	if handle == nil {
		return nil, errors.New("failed to allocate enterprise secure buffer")
	}

	buffer := &Buffer{
		handle: C.SecureBufferHandle(handle),
		locked: false,
	}

	enterprise := &EnterpriseBuffer{
		Buffer:        buffer,
		securityLevel: level,
	}

	runtime.SetFinalizer(enterprise, (*EnterpriseBuffer).finalizer)
	return enterprise, nil
}

// === ENTROPY INTEGRATION ===

// NewWithFastEntropy creates a buffer pre-filled with fast entropy
func NewWithFastEntropy(capacity int) (*Buffer, error) {
	if capacity <= 0 {
		return nil, errors.New("invalid capacity: must be positive")
	}

	handle := C.securebuffer_new_with_fast_entropy(C.size_t(capacity))
	if handle == nil {
		return nil, errors.New("failed to allocate buffer with fast entropy")
	}

	buffer := &Buffer{
		handle: C.SecureBufferHandle(handle),
		locked: false,
	}

	runtime.SetFinalizer(buffer, (*Buffer).finalizer)
	return buffer, nil
}

// NewWithHybridEntropy creates a buffer with Bitcoin header entropy
func NewWithHybridEntropy(capacity int, blockHeaders [][]byte) (*Buffer, error) {
	if capacity <= 0 {
		return nil, errors.New("invalid capacity: must be positive")
	}

	// Prepare headers for C call
	if len(blockHeaders) == 0 {
		// Call without headers
		handle := C.securebuffer_new_with_fast_entropy(C.size_t(capacity))
		if handle == nil {
			return nil, errors.New("failed to allocate buffer")
		}

		buffer := &Buffer{
			handle: C.SecureBufferHandle(handle),
			locked: false,
		}
		runtime.SetFinalizer(buffer, (*Buffer).finalizer)
		return buffer, nil
	}

	// Flatten headers for C API
	var totalLen int
	for _, header := range blockHeaders {
		totalLen += len(header)
	}

	flatHeaders := make([]byte, totalLen)
	offset := 0
	for _, header := range blockHeaders {
		copy(flatHeaders[offset:], header)
		offset += len(header)
	}

	handle := C.securebuffer_new_with_hybrid_entropy(
		C.size_t(capacity),
		(*C.uint8_t)(unsafe.Pointer(&flatHeaders[0])),
		C.size_t(totalLen),
		C.size_t(len(blockHeaders)),
	)

	if handle == nil {
		return nil, errors.New("failed to allocate buffer with hybrid entropy")
	}

	buffer := &Buffer{
		handle: C.SecureBufferHandle(handle),
		locked: false,
	}

	runtime.SetFinalizer(buffer, (*Buffer).finalizer)
	return buffer, nil
}

// FillWithFastEntropy fills an existing buffer with fast entropy
func (b *Buffer) FillWithFastEntropy() error {
	if b == nil || b.handle == nil {
		return errors.New("buffer is nil or freed")
	}

	result := C.securebuffer_fill_fast_entropy(unsafe.Pointer(b.handle))
	if result != 0 {
		return fmt.Errorf("failed to fill with fast entropy: error code %d", result)
	}

	return nil
}

// FillWithHybridEntropy fills buffer with Bitcoin header entropy
func (b *Buffer) FillWithHybridEntropy(blockHeaders [][]byte) error {
	if b == nil || b.handle == nil {
		return errors.New("buffer is nil or freed")
	}

	if len(blockHeaders) == 0 {
		return b.FillWithFastEntropy()
	}

	// Flatten headers
	var totalLen int
	for _, header := range blockHeaders {
		totalLen += len(header)
	}

	flatHeaders := make([]byte, totalLen)
	offset := 0
	for _, header := range blockHeaders {
		copy(flatHeaders[offset:], header)
		offset += len(header)
	}

	result := C.securebuffer_fill_hybrid_entropy(
		unsafe.Pointer(b.handle),
		(*C.uint8_t)(unsafe.Pointer(&flatHeaders[0])),
		C.size_t(totalLen),
		C.size_t(len(blockHeaders)),
	)

	if result != 0 {
		return fmt.Errorf("failed to fill with hybrid entropy: error code %d", result)
	}

	return nil
}

// RefreshEntropy refreshes buffer contents with new entropy
func (b *Buffer) RefreshEntropy() error {
	if b == nil || b.handle == nil {
		return errors.New("buffer is nil or freed")
	}

	result := C.securebuffer_refresh_entropy(unsafe.Pointer(b.handle))
	if result != 0 {
		return fmt.Errorf("failed to refresh entropy: error code %d", result)
	}

	return nil
}

// === CRYPTOGRAPHIC OPERATIONS ===

// HMACHex computes HMAC and returns hex string
func (b *Buffer) HMACHex(data []byte) (string, error) {
	if b == nil || b.handle == nil {
		return "", errors.New("buffer is nil or freed")
	}

	var dataPtr *C.uint8_t
	if len(data) > 0 {
		dataPtr = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	}

	cResult := C.securebuffer_hmac_hex(
		(*C.SecureBuffer)(unsafe.Pointer(b.handle)),
		dataPtr,
		C.size_t(len(data)),
	)

	if cResult == nil {
		return "", errors.New("failed to compute HMAC")
	}

	result := C.GoString(cResult)
	C.securebuffer_free_cstr(cResult)
	return result, nil
}

// HMACBase64URL computes HMAC and returns base64url string
func (b *Buffer) HMACBase64URL(data []byte) (string, error) {
	if b == nil || b.handle == nil {
		return "", errors.New("buffer is nil or freed")
	}

	var dataPtr *C.uint8_t
	if len(data) > 0 {
		dataPtr = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	}

	cResult := C.securebuffer_hmac_base64url(
		(*C.SecureBuffer)(unsafe.Pointer(b.handle)),
		dataPtr,
		C.size_t(len(data)),
	)

	if cResult == nil {
		return "", errors.New("failed to compute HMAC")
	}

	result := C.GoString(cResult)
	C.securebuffer_free_cstr(cResult)
	return result, nil
}

// === HARDWARE-BACKED SECURITY ===

// BindToHardware binds buffer to hardware security module
func (eb *EnterpriseBuffer) BindToHardware() error {
	if eb == nil || eb.Buffer == nil || eb.Buffer.handle == nil {
		return errors.New("buffer is nil or freed")
	}

	result := C.securebuffer_bind_to_hardware((*C.SecureBuffer)(unsafe.Pointer(eb.Buffer.handle)))
	if result != C.SECUREBUFFER_SUCCESS {
		return fmt.Errorf("failed to bind to hardware: error %d", result)
	}

	return nil
}

// IsHardwareBacked returns true if buffer is hardware-backed
func (eb *EnterpriseBuffer) IsHardwareBacked() bool {
	if eb == nil || eb.Buffer == nil || eb.Buffer.handle == nil {
		return false
	}

	return bool(C.securebuffer_is_hardware_backed((*C.SecureBuffer)(unsafe.Pointer(eb.Buffer.handle))))
}

// EnableSideChannelProtection enables side-channel attack protection
func (eb *EnterpriseBuffer) EnableSideChannelProtection() error {
	if eb == nil || eb.Buffer == nil || eb.Buffer.handle == nil {
		return errors.New("buffer is nil or freed")
	}

	result := C.securebuffer_enable_side_channel_protection((*C.SecureBuffer)(unsafe.Pointer(eb.Buffer.handle)))
	if result != C.SECUREBUFFER_SUCCESS {
		return fmt.Errorf("failed to enable side-channel protection: error %d", result)
	}

	return nil
}

// === TAMPER DETECTION ===

// EnableTamperDetection enables tamper detection on the buffer
func (eb *EnterpriseBuffer) EnableTamperDetection() error {
	if eb == nil || eb.Buffer == nil || eb.Buffer.handle == nil {
		return errors.New("buffer is nil or freed")
	}

	result := C.securebuffer_enable_tamper_detection((*C.SecureBuffer)(unsafe.Pointer(eb.Buffer.handle)))
	if result != C.SECUREBUFFER_SUCCESS {
		return fmt.Errorf("failed to enable tamper detection: error %d", result)
	}

	return nil
}

// IsTampered checks if buffer has been tampered with
func (eb *EnterpriseBuffer) IsTampered() bool {
	if eb == nil || eb.Buffer == nil || eb.Buffer.handle == nil {
		return true // Assume tampered if buffer is invalid
	}

	return bool(C.securebuffer_is_tampered((*C.SecureBuffer)(unsafe.Pointer(eb.Buffer.handle))))
}

// === AUDIT AND COMPLIANCE ===

// GetSecurityAuditLog returns security audit log for the buffer
func (eb *EnterpriseBuffer) GetSecurityAuditLog() (string, error) {
	if eb == nil || eb.Buffer == nil || eb.Buffer.handle == nil {
		return "", errors.New("buffer is nil or freed")
	}

	cResult := C.securebuffer_get_security_audit_log((*C.SecureBuffer)(unsafe.Pointer(eb.Buffer.handle)))
	if cResult == nil {
		return "", errors.New("failed to get audit log")
	}

	result := C.GoString(cResult)
	C.securebuffer_free_cstr(cResult)
	return result, nil
}

// ValidatePolicyCompliance validates buffer against enterprise policy
func (eb *EnterpriseBuffer) ValidatePolicyCompliance() error {
	if eb == nil || eb.Buffer == nil || eb.Buffer.handle == nil {
		return errors.New("buffer is nil or freed")
	}

	result := C.securebuffer_validate_policy_compliance((*C.SecureBuffer)(unsafe.Pointer(eb.Buffer.handle)))
	if result != C.SECUREBUFFER_SUCCESS {
		return fmt.Errorf("policy compliance validation failed: error %d", result)
	}

	return nil
}

// === GLOBAL SYSTEM FUNCTIONS ===

// EnableAuditLogging enables global audit logging
func EnableAuditLogging(logPath string) error {
	cLogPath := C.CString(logPath)
	defer C.free(unsafe.Pointer(cLogPath))

	result := C.securebuffer_enable_audit_logging(cLogPath)
	if result != C.SECUREBUFFER_SUCCESS {
		return fmt.Errorf("failed to enable audit logging: error %d", result)
	}

	return nil
}

// DisableAuditLogging disables global audit logging
func DisableAuditLogging() error {
	result := C.securebuffer_disable_audit_logging()
	if result != C.SECUREBUFFER_SUCCESS {
		return fmt.Errorf("failed to disable audit logging: error %d", result)
	}

	return nil
}

// IsAuditLoggingEnabled returns true if audit logging is enabled
func IsAuditLoggingEnabled() bool {
	return bool(C.securebuffer_is_audit_logging_enabled())
}

// GetComplianceReport returns global compliance report
func GetComplianceReport() (string, error) {
	cResult := C.securebuffer_get_compliance_report()
	if cResult == nil {
		return "", errors.New("failed to get compliance report")
	}

	result := C.GoString(cResult)
	C.securebuffer_free_cstr(cResult)
	return result, nil
}

// SetEnterprisePolicy sets enterprise security policy from JSON
func SetEnterprisePolicy(policyJSON string) error {
	cPolicy := C.CString(policyJSON)
	defer C.free(unsafe.Pointer(cPolicy))

	result := C.securebuffer_set_enterprise_policy(cPolicy)
	if result != C.SECUREBUFFER_SUCCESS {
		return fmt.Errorf("failed to set enterprise policy: error %d", result)
	}

	return nil
}

// === DIRECT ENTROPY FUNCTIONS ===

// FastEntropy returns 32 bytes of fast entropy
func FastEntropy() ([]byte, error) {
	output := make([]byte, 32)
	result := C.fast_entropy_c((*C.uchar)(unsafe.Pointer(&output[0])))
	if result != 0 {
		return nil, fmt.Errorf("failed to generate fast entropy: error %d", result)
	}
	return output, nil
}

// HybridEntropy returns entropy using Bitcoin headers
func HybridEntropy(blockHeaders [][]byte) ([]byte, error) {
	output := make([]byte, 32)

	if len(blockHeaders) == 0 {
		return FastEntropy()
	}

	// Prepare headers for C call
	headerPtrs := make([]*C.uchar, len(blockHeaders))
	headerLens := make([]C.size_t, len(blockHeaders))

	for i, header := range blockHeaders {
		if len(header) > 0 {
			headerPtrs[i] = (*C.uchar)(unsafe.Pointer(&header[0]))
			headerLens[i] = C.size_t(len(header))
		}
	}

	result := C.hybrid_entropy_c(
		(**C.uchar)(unsafe.Pointer(&headerPtrs[0])),
		(*C.size_t)(unsafe.Pointer(&headerLens[0])),
		C.size_t(len(blockHeaders)),
		(*C.uchar)(unsafe.Pointer(&output[0])),
	)

	if result != 0 {
		return nil, fmt.Errorf("failed to generate hybrid entropy: error %d", result)
	}

	return output, nil
}

// SystemFingerprint returns hardware fingerprint
func SystemFingerprint() ([]byte, error) {
	output := make([]byte, 32)
	result := C.system_fingerprint_c((*C.uchar)(unsafe.Pointer(&output[0])))
	if result != 0 {
		return nil, fmt.Errorf("failed to get system fingerprint: error %d", result)
	}
	return output, nil
}

// GetCPUTemperature returns CPU temperature for entropy
func GetCPUTemperature() (float32, error) {
	temp := C.get_cpu_temperature_c()
	if temp < 0 {
		return 0, errors.New("failed to get CPU temperature")
	}
	return float32(temp), nil
}

// finalizer for EnterpriseBuffer
func (eb *EnterpriseBuffer) finalizer() {
	if eb != nil && eb.Buffer != nil {
		eb.Buffer.finalizer()
	}
}
