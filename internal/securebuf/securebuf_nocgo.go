//go:build !cgo
// +build !cgo

package securebuf

import (
	"fmt"
)

// Stub implementations for non-CGO builds
// This allows the application to compile and run without Rust dependencies

// SecureBuffer represents a disabled secure buffer
type SecureBuffer struct {
	data []byte
}

// NewSecureBuffer creates a new stub secure buffer
func NewSecureBuffer(capacity int) (*SecureBuffer, error) {
	return &SecureBuffer{
		data: make([]byte, capacity),
	}, nil
}

// NewSecureBufferWithLevel creates a new stub secure buffer with security level
func NewSecureBufferWithLevel(capacity int, level int) (*SecureBuffer, error) {
	return NewSecureBuffer(capacity)
}

// Write writes data to the buffer
func (sb *SecureBuffer) Write(data []byte) (int, error) {
	if len(data) > len(sb.data) {
		return 0, fmt.Errorf("data too large for buffer")
	}
	copy(sb.data, data)
	return len(data), nil
}

// Read reads data from the buffer
func (sb *SecureBuffer) Read(data []byte) (int, error) {
	n := copy(data, sb.data)
	return n, nil
}

// Capacity returns the buffer capacity
func (sb *SecureBuffer) Capacity() int {
	return len(sb.data)
}

// Len returns the current data length
func (sb *SecureBuffer) Len() int {
	return len(sb.data)
}

// Lock locks the buffer (stub - no actual locking)
func (sb *SecureBuffer) Lock() error {
	return nil
}

// Unlock unlocks the buffer (stub - no actual unlocking)
func (sb *SecureBuffer) Unlock() error {
	return nil
}

// IsLocked checks if buffer is locked (always false in stub)
func (sb *SecureBuffer) IsLocked() bool {
	return false
}

// Zeroize clears the buffer
func (sb *SecureBuffer) Zeroize() error {
	for i := range sb.data {
		sb.data[i] = 0
	}
	return nil
}

// Free frees the buffer
func (sb *SecureBuffer) Free() error {
	sb.data = nil
	return nil
}

// IntegrityCheck performs integrity check (always passes in stub)
func (sb *SecureBuffer) IntegrityCheck() bool {
	return true
}

// IsHardwareBacked checks if buffer is hardware backed (always false in stub)
func (sb *SecureBuffer) IsHardwareBacked() bool {
	return false
}

// IsTampered checks if buffer is tampered (always false in stub)
func (sb *SecureBuffer) IsTampered() bool {
	return false
}

// HMAC functions
func HMACHex(key, data []byte) (string, error) {
	return fmt.Sprintf("%x", data), nil
}

func HMACBase64URL(key, data []byte) (string, error) {
	return "stubbed_hmac", nil
}

// Enterprise functions - only include what's NOT in enterprise_api.go
func BindToHardware() error {
	return fmt.Errorf("hardware binding not available in stub mode")
}

func EnableTamperDetection() error {
	return fmt.Errorf("tamper detection not available in stub mode")
}

func EnableSideChannelProtection() error {
	return fmt.Errorf("side channel protection not available in stub mode")
}

func ValidatePolicyCompliance() (bool, error) {
	return true, nil
}

func GetSecurityAuditLog() (string, error) {
	return "no audit log in stub mode", nil
}
