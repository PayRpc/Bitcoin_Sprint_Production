//go:build !cgo
// +build !cgo

// Package secure provides memory-locked secure buffers (fallback implementation)
package secure

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
)

// SecureBuffer provides memory storage for sensitive data (fallback implementation)
type SecureBuffer struct {
	data []byte
}

// NewSecureBuffer creates a new buffer of the specified size
func NewSecureBuffer(size int) *SecureBuffer {
	if size <= 0 {
		return nil
	}
	return &SecureBuffer{
		data: make([]byte, size),
	}
}

// Free releases the buffer and zeros its memory
func (sb *SecureBuffer) Free() {
	if sb.data != nil {
		for i := range sb.data {
			sb.data[i] = 0
		}
		sb.data = nil
	}
}

// Copy copies data into the buffer
func (sb *SecureBuffer) Copy(src []byte) bool {
	if sb.data == nil || len(src) > len(sb.data) {
		return false
	}
	copy(sb.data, src)
	// Zero remaining bytes
	for i := len(src); i < len(sb.data); i++ {
		sb.data[i] = 0
	}
	return true
}

// Data returns the buffer data (fallback - not memory locked)
func (sb *SecureBuffer) Data() []byte {
	if sb.data == nil {
		return nil
	}
	return sb.data
}

// String returns the buffer as a string
func (sb *SecureBuffer) String() string {
	if sb.data == nil {
		return ""
	}
	// Find first null byte
	end := len(sb.data)
	for i, b := range sb.data {
		if b == 0 {
			end = i
			break
		}
	}
	return string(sb.data[:end])
}

// WithBytes executes a function with access to the buffer data
func (sb *SecureBuffer) WithBytes(fn func([]byte) error) error {
	if sb.data == nil {
		return fmt.Errorf("buffer is nil")
	}
	return fn(sb.data)
}

// HMACHex computes HMAC-SHA256 and returns hex encoding
func (sb *SecureBuffer) HMACHex(data []byte) string {
	if sb.data == nil {
		return ""
	}
	mac := hmac.New(sha256.New, sb.data)
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// SetBasicAuthHeader sets HTTP Basic Auth using the secure buffer as password
func SetBasicAuthHeader(req *http.Request, username string, password *SecureBuffer) {
	if password == nil || password.data == nil {
		req.SetBasicAuth(username, "")
		return
	}

	// Build "user:pass" bytes
	userBytes := []byte(username)
	passBytes := password.data

	// Find actual length of password (up to first null byte)
	passLen := len(passBytes)
	for i, b := range passBytes {
		if b == 0 {
			passLen = i
			break
		}
	}

	// Create combined buffer
	combined := make([]byte, len(userBytes)+1+passLen)
	copy(combined, userBytes)
	combined[len(userBytes)] = ':'
	copy(combined[len(userBytes)+1:], passBytes[:passLen])

	// Encode and set header
	encoded := base64.StdEncoding.EncodeToString(combined)
	req.Header.Set("Authorization", "Basic "+encoded)

	// Zero the temporary buffer
	for i := range combined {
		combined[i] = 0
	}
}
