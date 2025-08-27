// Package entropy provides simple Go-based entropy sources
package entropy

import (
	"crypto/rand"
	"crypto/sha256"
	"time"
	"unsafe"
)

// SimpleEntropyGo generates 32 bytes of entropy using Go's crypto/rand
func SimpleEntropyGo() ([]byte, error) {
	data := make([]byte, 32)
	_, err := rand.Read(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// TimingEntropy generates entropy by mixing crypto/rand with timing jitter
func TimingEntropy() ([]byte, error) {
	// Start with crypto/rand
	baseData := make([]byte, 32)
	_, err := rand.Read(baseData)
	if err != nil {
		return nil, err
	}

	// Add timing jitter
	start := time.Now()
	for i := 0; i < 1000; i++ {
		_ = time.Now().UnixNano()
	}
	end := time.Now()

	// Mix timing data
	timingBytes := (*[8]byte)(unsafe.Pointer(&end))[:8]
	
	// Combine with hash
	hasher := sha256.New()
	hasher.Write(baseData)
	hasher.Write(timingBytes)
	hasher.Write([]byte(start.String()))
	
	return hasher.Sum(nil), nil
}

// EnhancedEntropyGo provides multiple entropy sources mixed together
func EnhancedEntropyGo() ([]byte, error) {
	// Multiple entropy sources
	sources := make([][]byte, 0, 3)
	
	// Source 1: crypto/rand
	data1 := make([]byte, 32)
	if _, err := rand.Read(data1); err != nil {
		return nil, err
	}
	sources = append(sources, data1)
	
	// Source 2: timing-based
	timing, err := TimingEntropy()
	if err != nil {
		return nil, err
	}
	sources = append(sources, timing)
	
	// Source 3: memory address randomness
	addr := uintptr(unsafe.Pointer(&sources))
	addrBytes := (*[8]byte)(unsafe.Pointer(&addr))[:8]
	
	// Final mixing
	hasher := sha256.New()
	for _, source := range sources {
		hasher.Write(source)
	}
	hasher.Write(addrBytes)
	hasher.Write([]byte(time.Now().String()))
	
	return hasher.Sum(nil), nil
}
