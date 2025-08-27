// Package api provides Bloom Filter functionality with Rust FFI integration
package api

/*
#cgo CFLAGS: -I../../secure/rust/include
#cgo LDFLAGS: -L../../secure/rust/target/release -lsecurebuffer -lws2_32 -luserenv -lntdll -lbcrypt -lmsvcrt -lkernel32 -lstdc++
#include "../../secure/rust/include/bloom_filter.h"
#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
)

// ===== BLOOM FILTER MANAGER IMPLEMENTATION =====

// BloomFilterManager manages the Rust Bloom Filter integration
type BloomFilterManager struct {
	filterHandle C.UniversalBloomFilterHandle
	config       config.Config
	mu           sync.RWMutex
	isEnabled    bool
}

// NewBloomFilterManager creates a new Bloom Filter manager
func NewBloomFilterManager(cfg config.Config) *BloomFilterManager {
	manager := &BloomFilterManager{
		config:    cfg,
		isEnabled: false,
	}

	// Initialize the Bloom Filter based on tier
	if err := manager.initializeFilter(); err != nil {
		// Log error but don't fail - Bloom Filter is optional
		fmt.Printf("Bloom Filter initialization failed: %v\n", err)
	}

	return manager
}

// initializeFilter initializes the Rust Bloom Filter with tier-appropriate settings
func (bfm *BloomFilterManager) initializeFilter() error {
	bfm.mu.Lock()
	defer bfm.mu.Unlock()

	var filterHandle C.UniversalBloomFilterHandle

	// Configure filter based on tier
	switch bfm.config.Tier {
	case config.TierTurbo, config.TierEnterprise:
		// High-performance configuration for premium tiers
		networkName := C.CString("bitcoin")
		defer C.free(unsafe.Pointer(networkName))

		filterHandle = C.universal_bloom_filter_new(
			C.size_t(100000),  // Large filter size
			C.uint8_t(7),      // More hash functions
			C.uint32_t(0),     // Random tweak
			C.uint8_t(0),      // Flags
			C.uint64_t(86400), // 24 hour max age
			C.size_t(8192),    // Large batch size
			networkName,
		)

	case config.TierBusiness:
		// Balanced configuration for business tier
		networkName := C.CString("bitcoin")
		defer C.free(unsafe.Pointer(networkName))

		filterHandle = C.universal_bloom_filter_new(
			C.size_t(50000),   // Medium filter size
			C.uint8_t(5),      // Standard hash functions
			C.uint32_t(0),     // Random tweak
			C.uint8_t(0),      // Flags
			C.uint64_t(86400), // 24 hour max age
			C.size_t(4096),    // Medium batch size
			networkName,
		)

	case config.TierPro:
		// Standard configuration for pro tier
		networkName := C.CString("bitcoin")
		defer C.free(unsafe.Pointer(networkName))

		filterHandle = C.universal_bloom_filter_new(
			C.size_t(36000),   // Standard Bitcoin Core size
			C.uint8_t(5),      // Standard hash functions
			C.uint32_t(0),     // Random tweak
			C.uint8_t(0),      // Flags
			C.uint64_t(86400), // 24 hour max age
			C.size_t(2048),    // Standard batch size
			networkName,
		)

	default: // Free tier
		// Memory-optimized configuration for free tier
		networkName := C.CString("bitcoin")
		defer C.free(unsafe.Pointer(networkName))

		filterHandle = C.universal_bloom_filter_new(
			C.size_t(18000),   // Smaller filter size
			C.uint8_t(3),      // Fewer hash functions
			C.uint32_t(0),     // Random tweak
			C.uint8_t(0),      // Flags
			C.uint64_t(86400), // 24 hour max age
			C.size_t(1024),    // Smaller batch size
			networkName,
		)
	}

	if filterHandle == nil {
		return fmt.Errorf("failed to create Bloom Filter")
	}

	bfm.filterHandle = filterHandle
	bfm.isEnabled = true
	return nil
}

// ContainsUTXO checks if a UTXO exists in the Bloom Filter
func (bfm *BloomFilterManager) ContainsUTXO(txid []byte, vout uint32) (bool, error) {
	if !bfm.isEnabled || bfm.filterHandle == nil {
		return false, fmt.Errorf("Bloom Filter not enabled")
	}

	bfm.mu.RLock()
	defer bfm.mu.RUnlock()

	if len(txid) != 32 {
		return false, fmt.Errorf("invalid TXID length: expected 32 bytes, got %d", len(txid))
	}

	result := C.universal_bloom_filter_contains_utxo(
		bfm.filterHandle,
		(*C.uint8_t)(unsafe.Pointer(&txid[0])),
		C.uint32_t(vout),
	)

	if result < 0 {
		return false, fmt.Errorf("Bloom Filter query failed with error code: %d", int(result))
	}

	return result == 1, nil
}

// InsertUTXO inserts a UTXO into the Bloom Filter
func (bfm *BloomFilterManager) InsertUTXO(txid []byte, vout uint32) error {
	if !bfm.isEnabled || bfm.filterHandle == nil {
		return fmt.Errorf("Bloom Filter not enabled")
	}

	bfm.mu.Lock()
	defer bfm.mu.Unlock()

	if len(txid) != 32 {
		return fmt.Errorf("invalid TXID length: expected 32 bytes, got %d", len(txid))
	}

	result := C.universal_bloom_filter_insert_utxo(
		bfm.filterHandle,
		(*C.uint8_t)(unsafe.Pointer(&txid[0])),
		C.uint32_t(vout),
	)

	if result != 0 {
		return fmt.Errorf("Bloom Filter insert failed with error code: %d", int(result))
	}

	return nil
}

// LoadBlock loads all transactions from a block into the Bloom Filter
func (bfm *BloomFilterManager) LoadBlock(blockData []byte) error {
	if !bfm.isEnabled || bfm.filterHandle == nil {
		return fmt.Errorf("Bloom Filter not enabled")
	}

	bfm.mu.Lock()
	defer bfm.mu.Unlock()

	if len(blockData) == 0 {
		return fmt.Errorf("empty block data")
	}

	result := C.universal_bloom_filter_load_block(
		bfm.filterHandle,
		(*C.uint8_t)(unsafe.Pointer(&blockData[0])),
		C.size_t(len(blockData)),
	)

	if result != 0 {
		return fmt.Errorf("Bloom Filter load block failed with error code: %d", int(result))
	}

	return nil
}

// IsEnabled returns whether the Bloom Filter is enabled
func (bfm *BloomFilterManager) IsEnabled() bool {
	bfm.mu.RLock()
	defer bfm.mu.RUnlock()
	return bfm.isEnabled
}

// Cleanup performs maintenance on the Bloom Filter
func (bfm *BloomFilterManager) Cleanup() error {
	if !bfm.isEnabled || bfm.filterHandle == nil {
		return fmt.Errorf("Bloom Filter not enabled")
	}

	bfm.mu.Lock()
	defer bfm.mu.Unlock()

	result := C.universal_bloom_filter_cleanup(bfm.filterHandle)
	if result != 0 {
		return fmt.Errorf("Bloom Filter cleanup failed with error code: %d", int(result))
	}

	return nil
}
