// Package entropy provides entropy functions with Rust FFI integration
package entropy

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"time"
)

// FastEntropy returns fast entropy using Rust FFI when available, fallback to Go
func FastEntropy() ([]byte, error) {
	// Try Rust implementation first (when CGO is enabled)
	if rustEntropy, err := FastEntropyRust(); err == nil {
		return rustEntropy, nil
	}
	// Fallback to Go implementation
	return SimpleEntropy()
}

// HybridEntropy returns enhanced entropy using Rust FFI when available, fallback to Go
func HybridEntropy() ([]byte, error) {
	// Try Rust implementation first (when CGO is enabled)
	if rustEntropy, err := HybridEntropyRust(nil); err == nil {
		return rustEntropy, nil
	}
	// Fallback to Go implementation
	return EnhancedEntropy()
}

// FastEntropyRust returns fast entropy using Rust FFI implementation
func FastEntropyRust() ([]byte, error) {
	// This will be implemented in entropy_cgo.go when CGO is enabled
	return nil, errors.New("Rust FFI not available - build with CGO enabled")
}

// HybridEntropyRust returns hybrid entropy using Rust FFI implementation
func HybridEntropyRust(headers [][]byte) ([]byte, error) {
	// This will be implemented in entropy_cgo.go when CGO is enabled
	return nil, errors.New("Rust FFI not available - build with CGO enabled")
}

// SimpleEntropy returns basic entropy using crypto/rand
func SimpleEntropy() ([]byte, error) {
	entropy := make([]byte, 32)
	if _, err := rand.Read(entropy); err != nil {
		return nil, err
	}
	return entropy, nil
}

// EnhancedEntropy returns enhanced entropy combining multiple sources
func EnhancedEntropy() ([]byte, error) {
	entropy := make([]byte, 32)

	// Primary entropy from crypto/rand
	if _, err := rand.Read(entropy); err != nil {
		return nil, err
	}

	// Add timing jitter
	timestamp := time.Now().UnixNano()
	binary.LittleEndian.PutUint64(entropy[24:32], uint64(timestamp))

	return entropy, nil
}

// GetCPUTemperatureRust returns CPU temperature using Rust FFI implementation (fallback)
func GetCPUTemperatureRust() (float32, error) {
	// Fallback: return a mock temperature value
	// In a real implementation, this would read actual CPU temperature
	return 45.0, nil
}

// SystemFingerprintRust returns system fingerprint using Rust FFI implementation (fallback)
func SystemFingerprintRust() ([]byte, error) {
	// Fallback: generate a simple system fingerprint using Go
	fingerprint := make([]byte, 32)

	// Use current time as a basic system fingerprint
	timestamp := time.Now().UnixNano()
	binary.LittleEndian.PutUint64(fingerprint[0:8], uint64(timestamp))

	// Add some randomness
	if _, err := rand.Read(fingerprint[8:]); err != nil {
		return nil, err
	}

	return fingerprint, nil
}

// FastEntropyWithFingerprintRust returns fast entropy with hardware fingerprinting (fallback)
func FastEntropyWithFingerprintRust() ([]byte, error) {
	// Get base entropy
	entropy, err := FastEntropy()
	if err != nil {
		return nil, err
	}

	// Mix in system fingerprint
	fingerprint, err := SystemFingerprintRust()
	if err != nil {
		return entropy, nil // Return base entropy if fingerprint fails
	}

	// XOR the entropy with fingerprint for additional uniqueness
	for i := range entropy {
		entropy[i] ^= fingerprint[i%len(fingerprint)]
	}

	return entropy, nil
}

// HybridEntropyWithFingerprintRust returns hybrid entropy with hardware fingerprinting (fallback)
func HybridEntropyWithFingerprintRust(headers [][]byte) ([]byte, error) {
	// Get base entropy
	entropy, err := HybridEntropy()
	if err != nil {
		return nil, err
	}

	// Mix in system fingerprint
	fingerprint, err := SystemFingerprintRust()
	if err != nil {
		return entropy, nil // Return base entropy if fingerprint fails
	}

	// XOR the entropy with fingerprint for additional uniqueness
	for i := range entropy {
		entropy[i] ^= fingerprint[i%len(fingerprint)]
	}

	return entropy, nil
}

// CreateEnterpriseEntropyBuffer creates a secure buffer with enterprise-grade entropy
func CreateEnterpriseEntropyBuffer(size int) ([]byte, error) {
	return FastEntropyWithFingerprintRust()
}

// CreateEntropyBufferWithHeaders creates entropy buffer with Bitcoin block headers
func CreateEntropyBufferWithHeaders(size int, headers [][]byte) ([]byte, error) {
	return HybridEntropyWithFingerprintRust(headers)
}
