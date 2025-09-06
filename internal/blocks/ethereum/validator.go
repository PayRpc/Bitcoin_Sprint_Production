package ethereum

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Validator implements BlockValidator interface for Ethereum
type Validator struct {
	logger *zap.Logger
}

// NewValidator creates a new Ethereum validator
func NewValidator(logger *zap.Logger) *Validator {
	return &Validator{
		logger: logger,
	}
}

// ValidateBlock validates an Ethereum block
func (v *Validator) ValidateBlock(ctx context.Context, block interface{}) error {
	ethBlock, ok := block.(*EthereumBlock)
	if !ok {
		return fmt.Errorf("invalid block type for Ethereum validator")
	}

	// Basic validation checks
	if ethBlock.Hash == "" {
		return fmt.Errorf("missing block hash")
	}

	if ethBlock.Number == 0 {
		return fmt.Errorf("invalid block number")
	}

	if ethBlock.Timestamp.IsZero() {
		return fmt.Errorf("missing block timestamp")
	}

	// Check if block is too far in the future
	if ethBlock.Timestamp.After(time.Now().Add(15 * time.Minute)) {
		return fmt.Errorf("block timestamp too far in future")
	}

	// Validate hash format (should be 66 characters with 0x prefix)
	if !strings.HasPrefix(ethBlock.Hash, "0x") || len(ethBlock.Hash) != 66 {
		return fmt.Errorf("invalid hash format: expected 0x + 64 hex chars, got %s", ethBlock.Hash)
	}

	if _, err := hex.DecodeString(ethBlock.Hash[2:]); err != nil {
		return fmt.Errorf("invalid hash format: %w", err)
	}

	// Validate parent hash
	if ethBlock.ParentHash != "" {
		if !strings.HasPrefix(ethBlock.ParentHash, "0x") || len(ethBlock.ParentHash) != 66 {
			return fmt.Errorf("invalid parent hash format")
		}
		if _, err := hex.DecodeString(ethBlock.ParentHash[2:]); err != nil {
			return fmt.Errorf("invalid parent hash format: %w", err)
		}
	}

	// Validate Merkle roots
	roots := []string{ethBlock.StateRoot, ethBlock.TransactionsRoot, ethBlock.ReceiptsRoot}
	rootNames := []string{"state_root", "transactions_root", "receipts_root"}
	
	for i, root := range roots {
		if root != "" {
			if !strings.HasPrefix(root, "0x") || len(root) != 66 {
				return fmt.Errorf("invalid %s format", rootNames[i])
			}
			if _, err := hex.DecodeString(root[2:]); err != nil {
				return fmt.Errorf("invalid %s format: %w", rootNames[i], err)
			}
		}
	}

	// Validate gas limits and usage
	if ethBlock.GasLimit == 0 {
		return fmt.Errorf("invalid gas limit: cannot be zero")
	}

	if ethBlock.GasUsed > ethBlock.GasLimit {
		return fmt.Errorf("gas used (%d) exceeds gas limit (%d)", ethBlock.GasUsed, ethBlock.GasLimit)
	}

	// Validate transaction count consistency
	if ethBlock.Transactions != nil && len(ethBlock.Transactions) != ethBlock.TransactionCount {
		return fmt.Errorf("transaction count mismatch: expected %d, got %d", 
			ethBlock.TransactionCount, len(ethBlock.Transactions))
	}

	v.logger.Debug("Ethereum block validation passed",
		zap.String("hash", ethBlock.Hash),
		zap.Uint64("number", ethBlock.Number),
		zap.Int("tx_count", ethBlock.TransactionCount))

	return nil
}

// ValidateTransactions validates all transactions in an Ethereum block
func (v *Validator) ValidateTransactions(ctx context.Context, block interface{}) error {
	ethBlock, ok := block.(*EthereumBlock)
	if !ok {
		return fmt.Errorf("invalid block type for Ethereum validator")
	}

	if ethBlock.Transactions == nil {
		return nil // No transactions to validate
	}

	for i, tx := range ethBlock.Transactions {
		if err := v.validateTransaction(ctx, tx); err != nil {
			return fmt.Errorf("transaction %d validation failed: %w", i, err)
		}
	}

	return nil
}

// CheckConsensus performs Ethereum-specific consensus checks
func (v *Validator) CheckConsensus(ctx context.Context, block interface{}) error {
	ethBlock, ok := block.(*EthereumBlock)
	if !ok {
		return fmt.Errorf("invalid block type for Ethereum validator")
	}

	// Check block size limits (approximate)
	if ethBlock.Size > 1000000 { // ~1MB limit
		return fmt.Errorf("block size exceeds reasonable limit: %d bytes", ethBlock.Size)
	}

	// Validate base fee (EIP-1559)
	if ethBlock.BaseFeePerGas != nil {
		if ethBlock.BaseFeePerGas.Sign() < 0 {
			return fmt.Errorf("base fee cannot be negative")
		}
	}

	// Check uncle blocks (should be reasonable count)
	if len(ethBlock.Uncles) > 2 {
		return fmt.Errorf("too many uncle blocks: %d", len(ethBlock.Uncles))
	}

	return nil
}

// validateTransaction validates a single Ethereum transaction
func (v *Validator) validateTransaction(ctx context.Context, tx EthereumTx) error {
	if tx.Hash == "" {
		return fmt.Errorf("missing transaction hash")
	}

	if !strings.HasPrefix(tx.Hash, "0x") || len(tx.Hash) != 66 {
		return fmt.Errorf("invalid transaction hash format")
	}

	if _, err := hex.DecodeString(tx.Hash[2:]); err != nil {
		return fmt.Errorf("invalid transaction hash format: %w", err)
	}

	// Validate addresses
	if tx.From == "" {
		return fmt.Errorf("missing from address")
	}

	if !isValidEthereumAddress(tx.From) {
		return fmt.Errorf("invalid from address format")
	}

	if tx.To != nil && !isValidEthereumAddress(*tx.To) {
		return fmt.Errorf("invalid to address format")
	}

	// Validate value
	if tx.Value != nil && tx.Value.Sign() < 0 {
		return fmt.Errorf("transaction value cannot be negative")
	}

	// Validate gas
	if tx.Gas == 0 {
		return fmt.Errorf("gas limit cannot be zero")
	}

	if tx.GasPrice != nil && tx.GasPrice.Sign() < 0 {
		return fmt.Errorf("gas price cannot be negative")
	}

	return nil
}

// isValidEthereumAddress checks if an address is a valid Ethereum address
func isValidEthereumAddress(addr string) bool {
	if !strings.HasPrefix(addr, "0x") {
		return false
	}
	
	if len(addr) != 42 { // 0x + 40 hex characters
		return false
	}
	
	_, err := hex.DecodeString(addr[2:])
	return err == nil
}

// EthereumBlock represents Ethereum block structure for validation
type EthereumBlock struct {
	Hash             string       `json:"hash"`
	Number           uint64       `json:"number"`
	ParentHash       string       `json:"parent_hash"`
	StateRoot        string       `json:"state_root"`
	TransactionsRoot string       `json:"transactions_root"`
	ReceiptsRoot     string       `json:"receipts_root"`
	Timestamp        time.Time    `json:"timestamp"`
	GasLimit         uint64       `json:"gas_limit"`
	GasUsed          uint64       `json:"gas_used"`
	BaseFeePerGas    *big.Int     `json:"base_fee_per_gas,omitempty"`
	Size             int64        `json:"size"`
	TransactionCount int          `json:"transaction_count"`
	Transactions     []EthereumTx `json:"transactions,omitempty"`
	Uncles           []string     `json:"uncles,omitempty"`
}

// EthereumTx represents an Ethereum transaction
type EthereumTx struct {
	Hash     string   `json:"hash"`
	From     string   `json:"from"`
	To       *string  `json:"to,omitempty"`
	Value    *big.Int `json:"value"`
	Gas      uint64   `json:"gas"`
	GasPrice *big.Int `json:"gas_price"`
	Nonce    uint64   `json:"nonce"`
}
