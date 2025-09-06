//go:build cgo
// +build cgo

package securebuf

// Bitcoin Bloom Filter FFI - Ultra-high performance UTXO filtering
// This integrates the Rust Universal Bloom Filter for Bitcoin operations

/*
#cgo LDFLAGS: -lsecurebuffer
#include "../../secure/rust/include/bloom_filter.h"
#include "../../secure/rust/include/securebuffer.h"
#include <stdlib.h>
#include <stdint.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"
)

// BitcoinBloomFilter represents a high-performance Bitcoin Bloom filter
type BitcoinBloomFilter struct {
	handle unsafe.Pointer
}

// BloomFilterStats contains bloom filter performance statistics
type BloomFilterStats struct {
	ItemCount          uint64  `json:"item_count"`
	FalsePositiveCount uint64  `json:"false_positive_count"`
	TheoreticalFPRate  float64 `json:"theoretical_fp_rate"`
	MemoryUsageBytes   uint64  `json:"memory_usage_bytes"`
	TimestampEntries   uint64  `json:"timestamp_entries"`
	AverageAgeSeconds  float64 `json:"average_age_seconds"`
}

// NewBitcoinBloomFilter creates a new Bitcoin Bloom filter with custom parameters
func NewBitcoinBloomFilter(sizeBits uint64, numHashes uint8, tweak uint32, flags uint8, maxAgeSeconds uint64, batchSize uint64) (*BitcoinBloomFilter, error) {
	handle := C.bitcoin_bloom_filter_new(
		C.size_t(sizeBits),
		C.uint8_t(numHashes),
		C.uint32_t(tweak),
		C.uint8_t(flags),
		C.uint64_t(maxAgeSeconds),
		C.size_t(batchSize),
	)

	if handle == nil {
		return nil, errors.New("failed to create Bitcoin Bloom filter")
	}

	filter := &BitcoinBloomFilter{
		handle: handle,
	}

	runtime.SetFinalizer(filter, (*BitcoinBloomFilter).finalizer)
	return filter, nil
}

// NewBitcoinBloomFilterDefault creates a Bitcoin Bloom filter with optimized defaults
func NewBitcoinBloomFilterDefault() (*BitcoinBloomFilter, error) {
	handle := C.bitcoin_bloom_filter_new_default()
	if handle == nil {
		return nil, errors.New("failed to create default Bitcoin Bloom filter")
	}

	filter := &BitcoinBloomFilter{
		handle: handle,
	}

	runtime.SetFinalizer(filter, (*BitcoinBloomFilter).finalizer)
	return filter, nil
}

// InsertUTXO inserts a single UTXO (transaction ID + output index) into the filter
func (bf *BitcoinBloomFilter) InsertUTXO(txid []byte, vout uint32) error {
	if bf == nil || bf.handle == nil {
		return errors.New("bloom filter is nil or destroyed")
	}

	if len(txid) != 32 {
		return errors.New("transaction ID must be exactly 32 bytes")
	}

	result := C.bitcoin_bloom_filter_insert_utxo(
		bf.handle,
		(*C.uint8_t)(unsafe.Pointer(&txid[0])),
		C.uint32_t(vout),
	)

	if result != 0 {
		return fmt.Errorf("failed to insert UTXO: error %d", result)
	}

	return nil
}

// InsertUTXOBatch inserts multiple UTXOs in a single high-performance operation
func (bf *BitcoinBloomFilter) InsertUTXOBatch(txids [][]byte, vouts []uint32) error {
	if bf == nil || bf.handle == nil {
		return errors.New("bloom filter is nil or destroyed")
	}

	if len(txids) != len(vouts) {
		return errors.New("txids and vouts slices must have the same length")
	}

	if len(txids) == 0 {
		return nil // Nothing to insert
	}

	// Validate all txids are 32 bytes
	for i, txid := range txids {
		if len(txid) != 32 {
			return fmt.Errorf("transaction ID at index %d must be exactly 32 bytes", i)
		}
	}

	// Flatten txids into a single byte array (32 bytes per txid)
	flatTxids := make([]byte, len(txids)*32)
	for i, txid := range txids {
		copy(flatTxids[i*32:], txid)
	}

	result := C.bitcoin_bloom_filter_insert_batch(
		bf.handle,
		(*C.uint8_t)(unsafe.Pointer(&flatTxids[0])),
		(*C.uint32_t)(unsafe.Pointer(&vouts[0])),
		C.size_t(len(txids)),
	)

	if result != 0 {
		return fmt.Errorf("failed to insert UTXO batch: error %d", result)
	}

	return nil
}

// ContainsUTXO checks if a UTXO exists in the bloom filter
func (bf *BitcoinBloomFilter) ContainsUTXO(txid []byte, vout uint32) (bool, error) {
	if bf == nil || bf.handle == nil {
		return false, errors.New("bloom filter is nil or destroyed")
	}

	if len(txid) != 32 {
		return false, errors.New("transaction ID must be exactly 32 bytes")
	}

	result := C.bitcoin_bloom_filter_contains_utxo(
		bf.handle,
		(*C.uint8_t)(unsafe.Pointer(&txid[0])),
		C.uint32_t(vout),
	)

	switch result {
	case 0:
		return false, nil // Definitely not present
	case 1:
		return true, nil // Probably present (could be false positive)
	default:
		return false, fmt.Errorf("error checking UTXO: error %d", result)
	}
}

// ContainsUTXOBatch checks multiple UTXOs in a single high-performance operation
func (bf *BitcoinBloomFilter) ContainsUTXOBatch(txids [][]byte, vouts []uint32) ([]bool, error) {
	if bf == nil || bf.handle == nil {
		return nil, errors.New("bloom filter is nil or destroyed")
	}

	if len(txids) != len(vouts) {
		return nil, errors.New("txids and vouts slices must have the same length")
	}

	if len(txids) == 0 {
		return []bool{}, nil
	}

	// Validate all txids are 32 bytes
	for i, txid := range txids {
		if len(txid) != 32 {
			return nil, fmt.Errorf("transaction ID at index %d must be exactly 32 bytes", i)
		}
	}

	// Flatten txids into a single byte array
	flatTxids := make([]byte, len(txids)*32)
	for i, txid := range txids {
		copy(flatTxids[i*32:], txid)
	}

	// Prepare results array
	results := make([]bool, len(txids))

	result := C.bitcoin_bloom_filter_contains_batch(
		bf.handle,
		(*C.uint8_t)(unsafe.Pointer(&flatTxids[0])),
		(*C.uint32_t)(unsafe.Pointer(&vouts[0])),
		C.size_t(len(txids)),
		(*C.bool)(unsafe.Pointer(&results[0])),
	)

	if result != 0 {
		return nil, fmt.Errorf("failed to check UTXO batch: error %d", result)
	}

	return results, nil
}

// LoadBlock loads an entire Bitcoin block into the bloom filter
func (bf *BitcoinBloomFilter) LoadBlock(blockData []byte) error {
	if bf == nil || bf.handle == nil {
		return errors.New("bloom filter is nil or destroyed")
	}

	if len(blockData) == 0 {
		return errors.New("block data cannot be empty")
	}

	result := C.bitcoin_bloom_filter_load_block(
		bf.handle,
		(*C.uint8_t)(unsafe.Pointer(&blockData[0])),
		C.size_t(len(blockData)),
	)

	if result != 0 {
		return fmt.Errorf("failed to load block: error %d", result)
	}

	return nil
}

// GetStats returns comprehensive statistics about the bloom filter
func (bf *BitcoinBloomFilter) GetStats() (*BloomFilterStats, error) {
	if bf == nil || bf.handle == nil {
		return nil, errors.New("bloom filter is nil or destroyed")
	}

	var (
		itemCount          C.uint64_t
		falsePositiveCount C.uint64_t
		theoreticalFPRate  C.double
		memoryUsageBytes   C.size_t
		timestampEntries   C.size_t
		averageAgeSeconds  C.double
	)

	result := C.bitcoin_bloom_filter_get_stats(
		bf.handle,
		&itemCount,
		&falsePositiveCount,
		&theoreticalFPRate,
		&memoryUsageBytes,
		&timestampEntries,
		&averageAgeSeconds,
	)

	if result != 0 {
		return nil, fmt.Errorf("failed to get bloom filter stats: error %d", result)
	}

	return &BloomFilterStats{
		ItemCount:          uint64(itemCount),
		FalsePositiveCount: uint64(falsePositiveCount),
		TheoreticalFPRate:  float64(theoreticalFPRate),
		MemoryUsageBytes:   uint64(memoryUsageBytes),
		TimestampEntries:   uint64(timestampEntries),
		AverageAgeSeconds:  float64(averageAgeSeconds),
	}, nil
}

// GetFalsePositiveRate returns the theoretical false positive rate
func (bf *BitcoinBloomFilter) GetFalsePositiveRate() (float64, error) {
	if bf == nil || bf.handle == nil {
		return 0, errors.New("bloom filter is nil or destroyed")
	}

	rate := C.bitcoin_bloom_filter_false_positive_rate(bf.handle)
	if rate < 0 {
		return 0, errors.New("failed to get false positive rate")
	}

	return float64(rate), nil
}

// Cleanup removes old entries to maintain performance
func (bf *BitcoinBloomFilter) Cleanup() error {
	if bf == nil || bf.handle == nil {
		return errors.New("bloom filter is nil or destroyed")
	}

	result := C.bitcoin_bloom_filter_cleanup(bf.handle)
	if result != 0 {
		return fmt.Errorf("failed to cleanup bloom filter: error %d", result)
	}

	return nil
}

// AutoCleanup performs automatic cleanup if needed (call periodically)
func (bf *BitcoinBloomFilter) AutoCleanup() error {
	if bf == nil || bf.handle == nil {
		return errors.New("bloom filter is nil or destroyed")
	}

	result := C.bitcoin_bloom_filter_auto_cleanup(bf.handle)
	if result != 0 {
		return fmt.Errorf("failed to auto-cleanup bloom filter: error %d", result)
	}

	return nil
}

// Free releases the bloom filter and securely wipes its memory
func (bf *BitcoinBloomFilter) Free() {
	if bf != nil && bf.handle != nil {
		C.bitcoin_bloom_filter_destroy(bf.handle)
		bf.handle = nil
		runtime.SetFinalizer(bf, nil)
	}
}

// finalizer is called by the garbage collector
func (bf *BitcoinBloomFilter) finalizer() {
	bf.Free()
}
