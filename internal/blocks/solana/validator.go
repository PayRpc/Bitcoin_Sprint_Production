package solana

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Validator implements BlockValidator interface for Solana
type Validator struct {
	logger *zap.Logger
}

// NewValidator creates a new Solana validator
func NewValidator(logger *zap.Logger) *Validator {
	return &Validator{
		logger: logger,
	}
}

// ValidateBlock validates a Solana block (slot)
func (v *Validator) ValidateBlock(ctx context.Context, block interface{}) error {
	solBlock, ok := block.(*SolanaBlock)
	if !ok {
		return fmt.Errorf("invalid block type for Solana validator")
	}

	// Basic validation checks
	if solBlock.Blockhash == "" {
		return fmt.Errorf("missing block hash")
	}

	if solBlock.Slot == 0 {
		return fmt.Errorf("invalid slot number")
	}

	if solBlock.BlockTime == nil {
		return fmt.Errorf("missing block time")
	}

	// Check if block is too far in the future
	blockTime := time.Unix(*solBlock.BlockTime, 0)
	if blockTime.After(time.Now().Add(10 * time.Minute)) {
		return fmt.Errorf("block time too far in future")
	}

	// Validate blockhash format (Solana uses base58, but we'll check basic format)
	if len(solBlock.Blockhash) < 32 || len(solBlock.Blockhash) > 44 {
		return fmt.Errorf("invalid blockhash length: %d", len(solBlock.Blockhash))
	}

	// Validate parent slot relationship
	if solBlock.ParentSlot != 0 && solBlock.ParentSlot >= solBlock.Slot {
		return fmt.Errorf("invalid parent slot: %d >= current slot: %d", 
			solBlock.ParentSlot, solBlock.Slot)
	}

	// Validate transaction count consistency
	if solBlock.Transactions != nil && len(solBlock.Transactions) != solBlock.TransactionCount {
		return fmt.Errorf("transaction count mismatch: expected %d, got %d", 
			solBlock.TransactionCount, len(solBlock.Transactions))
	}

	// Validate rewards structure
	if solBlock.Rewards != nil {
		for i, reward := range solBlock.Rewards {
			if reward.Pubkey == "" {
				return fmt.Errorf("reward %d missing pubkey", i)
			}
			if len(reward.Pubkey) < 32 || len(reward.Pubkey) > 44 {
				return fmt.Errorf("reward %d invalid pubkey length", i)
			}
		}
	}

	v.logger.Debug("Solana block validation passed",
		zap.String("blockhash", solBlock.Blockhash),
		zap.Uint64("slot", solBlock.Slot),
		zap.Int("tx_count", solBlock.TransactionCount))

	return nil
}

// ValidateTransactions validates all transactions in a Solana block
func (v *Validator) ValidateTransactions(ctx context.Context, block interface{}) error {
	solBlock, ok := block.(*SolanaBlock)
	if !ok {
		return fmt.Errorf("invalid block type for Solana validator")
	}

	if solBlock.Transactions == nil {
		return nil // No transactions to validate
	}

	for i, tx := range solBlock.Transactions {
		if err := v.validateTransaction(ctx, tx); err != nil {
			return fmt.Errorf("transaction %d validation failed: %w", i, err)
		}
	}

	return nil
}

// CheckConsensus performs Solana-specific consensus checks
func (v *Validator) CheckConsensus(ctx context.Context, block interface{}) error {
	solBlock, ok := block.(*SolanaBlock)
	if !ok {
		return fmt.Errorf("invalid block type for Solana validator")
	}

	// Check slot progression (Solana has ~400ms slot time)
	if solBlock.ParentSlot != 0 {
		// Allow some variance in slot timing
		if solBlock.Slot-solBlock.ParentSlot > 10 {
			v.logger.Warn("Large slot gap detected",
				zap.Uint64("current_slot", solBlock.Slot),
				zap.Uint64("parent_slot", solBlock.ParentSlot),
				zap.Uint64("gap", solBlock.Slot-solBlock.ParentSlot))
		}
	}

	// Validate block height progression
	if solBlock.BlockHeight != nil && solBlock.ParentSlot != 0 {
		// Block height should generally increase with slots
		if *solBlock.BlockHeight == 0 {
			return fmt.Errorf("invalid block height: cannot be zero")
		}
	}

	// Check transaction limits (Solana has specific limits)
	if solBlock.TransactionCount > 20000 { // Reasonable upper bound
		return fmt.Errorf("transaction count exceeds reasonable limit: %d", solBlock.TransactionCount)
	}

	return nil
}

// validateTransaction validates a single Solana transaction
func (v *Validator) validateTransaction(ctx context.Context, tx SolanaTransaction) error {
	// Validate signatures
	if len(tx.Signatures) == 0 {
		return fmt.Errorf("transaction missing signatures")
	}

	for i, sig := range tx.Signatures {
		if sig == "" {
			return fmt.Errorf("signature %d is empty", i)
		}
		// Solana signatures are base58 encoded, typically 87-88 characters
		if len(sig) < 80 || len(sig) > 90 {
			return fmt.Errorf("signature %d invalid length: %d", i, len(sig))
		}
	}

	// Validate message structure
	if tx.Message.AccountKeys == nil || len(tx.Message.AccountKeys) == 0 {
		return fmt.Errorf("transaction missing account keys")
	}

	// Validate account keys format
	for i, pubkey := range tx.Message.AccountKeys {
		if pubkey == "" {
			return fmt.Errorf("account key %d is empty", i)
		}
		if len(pubkey) < 32 || len(pubkey) > 44 {
			return fmt.Errorf("account key %d invalid length: %d", i, len(pubkey))
		}
	}

	// Validate instructions
	if tx.Message.Instructions != nil {
		for i, instruction := range tx.Message.Instructions {
			if err := v.validateInstruction(instruction, len(tx.Message.AccountKeys)); err != nil {
				return fmt.Errorf("instruction %d validation failed: %w", i, err)
			}
		}
	}

	// Validate recent blockhash
	if tx.Message.RecentBlockhash == "" {
		return fmt.Errorf("transaction missing recent blockhash")
	}

	if len(tx.Message.RecentBlockhash) < 32 || len(tx.Message.RecentBlockhash) > 44 {
		return fmt.Errorf("invalid recent blockhash length: %d", len(tx.Message.RecentBlockhash))
	}

	return nil
}

// validateInstruction validates a Solana instruction
func (v *Validator) validateInstruction(instruction SolanaInstruction, accountCount int) error {
	// Validate program ID index
	if instruction.ProgramIdIndex >= accountCount {
		return fmt.Errorf("program id index %d exceeds account count %d", 
			instruction.ProgramIdIndex, accountCount)
	}

	// Validate account indices
	for i, accountIndex := range instruction.Accounts {
		if accountIndex >= accountCount {
			return fmt.Errorf("account index %d at position %d exceeds account count %d", 
				accountIndex, i, accountCount)
		}
	}

	// Validate instruction data format
	if instruction.Data != "" {
		// Data should be base64 encoded
		if _, err := base64.StdEncoding.DecodeString(instruction.Data); err != nil {
			return fmt.Errorf("invalid instruction data encoding: %w", err)
		}
	}

	return nil
}

// SolanaBlock represents Solana block structure for validation
type SolanaBlock struct {
	Blockhash        string              `json:"blockhash"`
	Slot             uint64              `json:"slot"`
	ParentSlot       uint64              `json:"parent_slot"`
	BlockTime        *int64              `json:"block_time"`
	BlockHeight      *uint64             `json:"block_height,omitempty"`
	TransactionCount int                 `json:"transaction_count"`
	Transactions     []SolanaTransaction `json:"transactions,omitempty"`
	Rewards          []SolanaReward      `json:"rewards,omitempty"`
}

// SolanaTransaction represents a Solana transaction
type SolanaTransaction struct {
	Signatures []string       `json:"signatures"`
	Message    SolanaMessage  `json:"message"`
	Meta       *SolanaTxMeta  `json:"meta,omitempty"`
}

// SolanaMessage represents a Solana transaction message
type SolanaMessage struct {
	AccountKeys                []string            `json:"account_keys"`
	RecentBlockhash           string              `json:"recent_blockhash"`
	Instructions              []SolanaInstruction `json:"instructions"`
	Header                    SolanaMessageHeader `json:"header"`
}

// SolanaMessageHeader represents Solana message header
type SolanaMessageHeader struct {
	NumRequiredSignatures       int `json:"num_required_signatures"`
	NumReadonlySignedAccounts   int `json:"num_readonly_signed_accounts"`
	NumReadonlyUnsignedAccounts int `json:"num_readonly_unsigned_accounts"`
}

// SolanaInstruction represents a Solana instruction
type SolanaInstruction struct {
	ProgramIdIndex int    `json:"program_id_index"`
	Accounts       []int  `json:"accounts"`
	Data           string `json:"data"`
}

// SolanaTxMeta represents Solana transaction metadata
type SolanaTxMeta struct {
	Err               interface{}      `json:"err"`
	Fee               uint64           `json:"fee"`
	PreBalances       []uint64         `json:"pre_balances"`
	PostBalances      []uint64         `json:"post_balances"`
	InnerInstructions []interface{}    `json:"inner_instructions,omitempty"`
	LogMessages       []string         `json:"log_messages,omitempty"`
	PreTokenBalances  []interface{}    `json:"pre_token_balances,omitempty"`
	PostTokenBalances []interface{}    `json:"post_token_balances,omitempty"`
}

// SolanaReward represents a Solana validator reward
type SolanaReward struct {
	Pubkey      string  `json:"pubkey"`
	Lamports    int64   `json:"lamports"`
	PostBalance uint64  `json:"post_balance"`
	RewardType  string  `json:"reward_type"`
	Commission  *uint8  `json:"commission,omitempty"`
}
