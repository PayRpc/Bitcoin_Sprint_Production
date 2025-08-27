// Package api provides utility functions and interfaces
package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/securebuf"
)

// ===== UTILITY INTERFACES AND FUNCTIONS =====

// Clock interface for testable time operations
type Clock interface {
	Now() time.Time
}

// RealClock implements Clock using the real system time
type RealClock struct{}

// Now returns the current time
func (RealClock) Now() time.Time {
	return time.Now()
}

// RandomReader interface for testable random operations
type RandomReader interface {
	Read(p []byte) (n int, err error)
}

// RealRandomReader implements RandomReader using crypto/rand
type RealRandomReader struct{}

// Read reads random bytes
func (RealRandomReader) Read(p []byte) (n int, err error) {
	return rand.Read(p)
}

// hashKey creates a SHA256 hash of the key
func hashKey(key string) string {
	hasher := sha256.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

// exceedsKeyGenRateLimit checks if the client has exceeded the rate limit for key generation
func (s *Server) exceedsKeyGenRateLimit(clientIP string) bool {
	// Get key generation limit from config (use free tier as default for new users)
	var keyGenLimit int = 10 // fallback default
	if rateLimit, exists := s.cfg.RateLimits[config.TierFree]; exists {
		keyGenLimit = rateLimit.KeyGenerationPerHour
	}

	// Convert hourly limit to refill rate (tokens per second)
	refillRate := float64(keyGenLimit) / 3600.0
	return !s.rateLimiter.Allow(clientIP+":keygen", float64(keyGenLimit), refillRate)
}

// generateSecureRandomKey generates a secure random key using the securebuf package
func (s *Server) generateSecureRandomKey() (string, error) {
	// Use a larger key size for better security
	const keySize = 32

	// Create secure buffer
	keyBuf, err := securebuf.New(keySize)
	if err != nil {
		return "", fmt.Errorf("failed to create secure buffer: %w", err)
	}
	defer keyBuf.Free() // Ensure memory is wiped

	// Generate random bytes
	keyBytes := make([]byte, keySize)
	if _, err := s.randReader.Read(keyBytes); err != nil {
		return "", fmt.Errorf("failed to generate random data: %w", err)
	}

	// Write to secure buffer
	if err := keyBuf.Write(keyBytes); err != nil {
		return "", fmt.Errorf("failed to write to secure buffer: %w", err)
	}

	// Clear the original slice to remove it from memory
	for i := range keyBytes {
		keyBytes[i] = 0
	}

	// Read from secure buffer
	finalKeyBytes, err := keyBuf.ReadToSlice()
	if err != nil {
		return "", fmt.Errorf("failed to read from secure buffer: %w", err)
	}

	// Convert to hex string
	newKey := hex.EncodeToString(finalKeyBytes)

	// Clear the final key bytes too
	for i := range finalKeyBytes {
		finalKeyBytes[i] = 0
	}

	return newKey, nil
}
