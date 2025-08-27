// Package entropy provides Go-based entropy functions
package entropy

import (
	"crypto/rand"
	"encoding/binary"
	"time"
)

// FastEntropy returns fast entropy using Go-based implementation
func FastEntropy() ([]byte, error) {
	return SimpleEntropy()
}

// HybridEntropy returns enhanced entropy using Go-based implementation
func HybridEntropy() ([]byte, error) {
	return EnhancedEntropy()
}

// FastEntropyRust returns fast entropy using Rust FFI implementation (fallback to Go)
func FastEntropyRust() ([]byte, error) {
	// Fallback to Go implementation when Rust FFI is not available
	return FastEntropy()
}

// HybridEntropyRust returns hybrid entropy using Rust FFI implementation (fallback to Go)
func HybridEntropyRust(headers [][]byte) ([]byte, error) {
	// Fallback to Go implementation when Rust FFI is not available
	return HybridEntropy()
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

// GetCPUTemperatureRust returns CPU temperature using Rust FFI implementation (fallback)
func GetCPUTemperatureRust() (float32, error) {
	// Fallback: return a mock temperature value
	// In a real implementation, this would read actual CPU temperature
	return 45.0, nil
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