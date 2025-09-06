//go:build !cgo
// +build !cgo

package securebuf

import (
	"errors"
)

// BitcoinBloomFilter represents a high-performance Bitcoin Bloom filter (disabled for non-CGO builds)
type BitcoinBloomFilter struct {
	disabled bool
}

// BloomFilterStats contains bloom filter performance statistics
type BloomFilterStats struct {
	ItemCount          uint64  `json:"item_count"`
	FalsePositiveRate  float64 `json:"false_positive_rate"`
	MemoryUsageBytes   uint64  `json:"memory_usage_bytes"`
	CompressionEnabled bool    `json:"compression_enabled"`
	TimestampEntries   uint64  `json:"timestamp_entries"`
	AverageAgeSeconds  float64 `json:"average_age_seconds"`
}

// NewBitcoinBloomFilter creates a new Bitcoin Bloom filter with custom parameters (disabled in non-CGO builds)
func NewBitcoinBloomFilter(sizeBits uint64, numHashes uint8, tweak uint32, flags uint8, maxAgeSeconds uint64, batchSize uint64) (*BitcoinBloomFilter, error) {
	return nil, errors.New("Bitcoin Bloom filter not available in non-CGO builds")
}

// NewBitcoinBloomFilterDefault creates a Bitcoin Bloom filter with optimized defaults (disabled in non-CGO builds)
func NewBitcoinBloomFilterDefault() (*BitcoinBloomFilter, error) {
	return nil, errors.New("Bitcoin Bloom filter not available in non-CGO builds")
}

// All other methods return appropriate errors or defaults for disabled builds
func (bf *BitcoinBloomFilter) InsertUTXO(txHash []byte, outputIndex uint32, scriptPubKey []byte, amount uint64) error {
	return errors.New("Bitcoin Bloom filter not available in non-CGO builds")
}

func (bf *BitcoinBloomFilter) InsertBatch(entries []UTXOEntry) ([]bool, error) {
	return nil, errors.New("Bitcoin Bloom filter not available in non-CGO builds")
}

func (bf *BitcoinBloomFilter) ContainsUTXO(txHash []byte, outputIndex uint32, scriptPubKey []byte) (bool, error) {
	return false, errors.New("Bitcoin Bloom filter not available in non-CGO builds")
}

func (bf *BitcoinBloomFilter) ContainsBatch(entries []UTXOEntry) ([]bool, error) {
	return nil, errors.New("Bitcoin Bloom filter not available in non-CGO builds")
}

func (bf *BitcoinBloomFilter) LoadBlock(blockData []byte, blockHeight uint64) error {
	return errors.New("Bitcoin Bloom filter not available in non-CGO builds")
}

func (bf *BitcoinBloomFilter) GetStats() (*BloomFilterStats, error) {
	return &BloomFilterStats{}, errors.New("Bitcoin Bloom filter not available in non-CGO builds")
}

func (bf *BitcoinBloomFilter) GetFalsePositiveRate() float64 {
	return 0.0
}

func (bf *BitcoinBloomFilter) Cleanup() error {
	return errors.New("Bitcoin Bloom filter not available in non-CGO builds")
}

func (bf *BitcoinBloomFilter) AutoCleanup() error {
	return errors.New("Bitcoin Bloom filter not available in non-CGO builds")
}

func (bf *BitcoinBloomFilter) finalizer() {
	// No-op for disabled builds
}

// UTXOEntry represents a UTXO entry for batch operations
type UTXOEntry struct {
	TxHash       []byte `json:"tx_hash"`
	OutputIndex  uint32 `json:"output_index"`
	ScriptPubKey []byte `json:"script_pub_key"`
	Amount       uint64 `json:"amount"`
}
