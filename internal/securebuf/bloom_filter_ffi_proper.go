//go:build cgo
// +build cgo

package securebuf

// Bitcoin Bloom Filter FFI - High-performance UTXO filtering with proper C bindings

/*
#cgo LDFLAGS: -L../../secure/rust/target/x86_64-pc-windows-gnu/release -lsecurebuffer
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

// BitcoinBloomFilterProper represents a high-performance Bitcoin Bloom filter
type BitcoinBloomFilterProper struct {
	handle unsafe.Pointer
}

// BloomFilterStatsProper contains bloom filter performance statistics
type BloomFilterStatsProper struct {
	ItemCount          uint64  `json:"item_count"`
	FalsePositiveRate  float64 `json:"false_positive_rate"`
	TimestampEntries   uint64  `json:"timestamp_entries"`
	AverageAgeSeconds  float64 `json:"average_age_seconds"`
}

// NewBitcoinBloomFilterProper creates a new Bitcoin Bloom filter using the universal bloom filter
func NewBitcoinBloomFilterProper(sizeBits uint64, numHashes uint8, tweak uint32, flags uint8, maxAgeSeconds uint64, batchSize uint64) (*BitcoinBloomFilterProper, error) {
	// Use the basic bloom filter configuration
	config := C.BloomConfig{
		network:           C.CString("bitcoin"),
		size:              C.uint64_t(sizeBits),
		num_hashes:        C.uint8_t(numHashes),
		tweak:             C.uint32_t(tweak),
		flags:             C.uint8_t(flags),
		max_age_seconds:   C.uint64_t(maxAgeSeconds),
		batch_size:        C.uint64_t(batchSize),
		enable_compression: false,
		enable_metrics:   true,
	}
	
	var err C.BloomFilterErrorCode
	handle := C.bloom_filter_new(&config, &err)
	C.free(unsafe.Pointer(config.network))

	if handle == nil || err != C.BLOOM_OK {
		return nil, fmt.Errorf("failed to create Bitcoin Bloom filter: error code %d", err)
	}

	filter := &BitcoinBloomFilterProper{
		handle: unsafe.Pointer(handle),
	}

	runtime.SetFinalizer(filter, (*BitcoinBloomFilterProper).finalizer)
	return filter, nil
}

// NewBitcoinBloomFilterDefaultProper creates a Bitcoin Bloom filter with optimized defaults
func NewBitcoinBloomFilterDefaultProper() (*BitcoinBloomFilterProper, error) {
	// Use default configuration optimized for Bitcoin
	return NewBitcoinBloomFilterProper(1000000, 10, 0, 0, 86400, 1000)
}

// InsertUTXO adds a UTXO to the bloom filter
func (bf *BitcoinBloomFilterProper) InsertUTXO(txHash []byte, outputIndex uint32) error {
	if bf.handle == nil {
		return errors.New("bloom filter is null")
	}

	// Combine txHash and outputIndex for the key
	key := make([]byte, len(txHash)+4)
	copy(key, txHash)
	key[len(txHash)] = byte(outputIndex)
	key[len(txHash)+1] = byte(outputIndex >> 8)
	key[len(txHash)+2] = byte(outputIndex >> 16)
	key[len(txHash)+3] = byte(outputIndex >> 24)

	result := C.bloom_filter_insert(
		(*C.UniversalBloomFilter)(bf.handle),
		(*C.uint8_t)(unsafe.Pointer(&key[0])),
		C.uint64_t(len(key)),
	)

	if !result {
		return errors.New("failed to insert UTXO into bloom filter")
	}
	return nil
}

// ContainsUTXO checks if a UTXO might be in the bloom filter
func (bf *BitcoinBloomFilterProper) ContainsUTXO(txHash []byte, outputIndex uint32) (bool, error) {
	if bf.handle == nil {
		return false, errors.New("bloom filter is null")
	}

	// Combine txHash and outputIndex for the key
	key := make([]byte, len(txHash)+4)
	copy(key, txHash)
	key[len(txHash)] = byte(outputIndex)
	key[len(txHash)+1] = byte(outputIndex >> 8)
	key[len(txHash)+2] = byte(outputIndex >> 16)
	key[len(txHash)+3] = byte(outputIndex >> 24)

	result := C.bloom_filter_contains(
		(*C.UniversalBloomFilter)(bf.handle),
		(*C.uint8_t)(unsafe.Pointer(&key[0])),
		C.uint64_t(len(key)),
	)

	return bool(result), nil
}

// GetStats returns bloom filter statistics
func (bf *BitcoinBloomFilterProper) GetStats() (*BloomFilterStatsProper, error) {
	if bf.handle == nil {
		return nil, errors.New("bloom filter is null")
	}

	count := C.bloom_filter_count((*C.UniversalBloomFilter)(bf.handle))
	rate := C.bloom_filter_false_positive_rate((*C.UniversalBloomFilter)(bf.handle))

	return &BloomFilterStatsProper{
		ItemCount:         uint64(count),
		FalsePositiveRate: float64(rate),
		TimestampEntries:  uint64(count), // Simplified
		AverageAgeSeconds: 0,             // Not available in basic interface
	}, nil
}

// Free releases the bloom filter resources
func (bf *BitcoinBloomFilterProper) Free() {
	if bf.handle != nil {
		C.bloom_filter_free((*C.UniversalBloomFilter)(bf.handle))
		bf.handle = nil
		runtime.SetFinalizer(bf, nil)
	}
}

func (bf *BitcoinBloomFilterProper) finalizer() {
	if bf.handle != nil {
		C.bloom_filter_free((*C.UniversalBloomFilter)(bf.handle))
		bf.handle = nil
	}
}
