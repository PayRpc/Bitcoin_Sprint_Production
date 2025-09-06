package bitcoin

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Validator implements BlockValidator interface for Bitcoin
type Validator struct {
	logger *zap.Logger
}

// NewValidator creates a new Bitcoin validator
func NewValidator(logger *zap.Logger) *Validator {
	return &Validator{
		logger: logger,
	}
}

// ValidateBlock validates a Bitcoin block
func (v *Validator) ValidateBlock(ctx context.Context, block interface{}) error {
	bitcoinBlock, ok := block.(*BitcoinBlock)
	if !ok {
		return fmt.Errorf("invalid block type for Bitcoin validator")
	}

	// Basic validation checks
	if bitcoinBlock.Hash == "" {
		return fmt.Errorf("missing block hash")
	}

	if bitcoinBlock.Height == 0 {
		return fmt.Errorf("invalid block height")
	}

	if bitcoinBlock.Timestamp.IsZero() {
		return fmt.Errorf("missing block timestamp")
	}

	// Check if block is too far in the future (within 2 hours)
	if bitcoinBlock.Timestamp.After(time.Now().Add(2 * time.Hour)) {
		return fmt.Errorf("block timestamp too far in future")
	}

	// Validate hash format (should be 64 hex characters)
	if len(bitcoinBlock.Hash) != 64 {
		return fmt.Errorf("invalid hash length: expected 64, got %d", len(bitcoinBlock.Hash))
	}

	if _, err := hex.DecodeString(bitcoinBlock.Hash); err != nil {
		return fmt.Errorf("invalid hash format: %w", err)
	}

	// Validate previous block hash if present
	if bitcoinBlock.PreviousBlockHash != "" {
		if len(bitcoinBlock.PreviousBlockHash) != 64 {
			return fmt.Errorf("invalid previous block hash length")
		}
		if _, err := hex.DecodeString(bitcoinBlock.PreviousBlockHash); err != nil {
			return fmt.Errorf("invalid previous block hash format: %w", err)
		}
	}

	// Validate merkle root if present
	if bitcoinBlock.MerkleRoot != "" {
		if len(bitcoinBlock.MerkleRoot) != 64 {
			return fmt.Errorf("invalid merkle root length")
		}
		if _, err := hex.DecodeString(bitcoinBlock.MerkleRoot); err != nil {
			return fmt.Errorf("invalid merkle root format: %w", err)
		}
	}

	// Validate transaction count consistency
	if bitcoinBlock.Transactions != nil && len(bitcoinBlock.Transactions) != bitcoinBlock.TransactionCount {
		return fmt.Errorf("transaction count mismatch: expected %d, got %d", 
			bitcoinBlock.TransactionCount, len(bitcoinBlock.Transactions))
	}

	v.logger.Debug("Bitcoin block validation passed",
		zap.String("hash", bitcoinBlock.Hash),
		zap.Uint64("height", bitcoinBlock.Height),
		zap.Int("tx_count", bitcoinBlock.TransactionCount))

	return nil
}

// ValidateTransactions validates all transactions in a Bitcoin block
func (v *Validator) ValidateTransactions(ctx context.Context, block interface{}) error {
	bitcoinBlock, ok := block.(*BitcoinBlock)
	if !ok {
		return fmt.Errorf("invalid block type for Bitcoin validator")
	}

	if bitcoinBlock.Transactions == nil {
		return nil // No transactions to validate
	}

	for i, tx := range bitcoinBlock.Transactions {
		if err := v.validateTransaction(ctx, tx); err != nil {
			return fmt.Errorf("transaction %d validation failed: %w", i, err)
		}
	}

	return nil
}

// CheckConsensus performs Bitcoin-specific consensus checks
func (v *Validator) CheckConsensus(ctx context.Context, block interface{}) error {
	bitcoinBlock, ok := block.(*BitcoinBlock)
	if !ok {
		return fmt.Errorf("invalid block type for Bitcoin validator")
	}

	// Validate proof of work (simplified check)
	if bitcoinBlock.Difficulty <= 0 {
		return fmt.Errorf("invalid difficulty: %f", bitcoinBlock.Difficulty)
	}

	// Check block size limits
	if bitcoinBlock.Size > 4000000 { // 4MB limit
		return fmt.Errorf("block size exceeds limit: %d bytes", bitcoinBlock.Size)
	}

	// Check weight (for SegWit)
	if bitcoinBlock.Weight > 4000000 { // 4M weight units
		return fmt.Errorf("block weight exceeds limit: %d", bitcoinBlock.Weight)
	}

	return nil
}

// validateTransaction validates a single Bitcoin transaction
func (v *Validator) validateTransaction(ctx context.Context, tx BitcoinTx) error {
	if tx.Hash == "" {
		return fmt.Errorf("missing transaction hash")
	}

	if len(tx.Hash) != 64 {
		return fmt.Errorf("invalid transaction hash length")
	}

	if _, err := hex.DecodeString(tx.Hash); err != nil {
		return fmt.Errorf("invalid transaction hash format: %w", err)
	}

	if tx.Size <= 0 {
		return fmt.Errorf("invalid transaction size: %d", tx.Size)
	}

	if tx.Inputs < 1 && tx.Outputs < 1 {
		return fmt.Errorf("transaction must have at least one input or output")
	}

	return nil
}

// BitcoinBlock represents Bitcoin block structure for validation
type BitcoinBlock struct {
	Hash              string    `json:"hash"`
	Height            uint64    `json:"height"`
	PreviousBlockHash string    `json:"previous_block_hash"`
	MerkleRoot        string    `json:"merkle_root"`
	Timestamp         time.Time `json:"timestamp"`
	Difficulty        float64   `json:"difficulty"`
	Size              int64     `json:"size"`
	Weight            int64     `json:"weight"`
	TransactionCount  int       `json:"transaction_count"`
	Transactions      []BitcoinTx `json:"transactions,omitempty"`
}

// BitcoinTx represents a Bitcoin transaction
type BitcoinTx struct {
	Hash    string `json:"hash"`
	Size    int    `json:"size"`
	Fee     int64  `json:"fee"`
	Inputs  int    `json:"inputs"`
	Outputs int    `json:"outputs"`
}
