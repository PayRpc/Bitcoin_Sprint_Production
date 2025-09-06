package solana

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Processor implements ChainProcessor interface for Solana
type Processor struct {
	logger      *zap.Logger
	chainState  *SolanaChainState
}

// NewProcessor creates a new Solana processor
func NewProcessor(logger *zap.Logger) *Processor {
	return &Processor{
		logger:     logger,
		chainState: NewSolanaChainState(),
	}
}

// ProcessBlock processes a Solana block (slot)
func (p *Processor) ProcessBlock(ctx context.Context, block interface{}) error {
	solBlock, ok := block.(*SolanaBlock)
	if !ok {
		return fmt.Errorf("invalid block type for Solana processor")
	}

	start := time.Now()
	defer func() {
		p.logger.Debug("Solana block processing completed",
			zap.String("blockhash", solBlock.Blockhash),
			zap.Uint64("slot", solBlock.Slot),
			zap.Duration("duration", time.Since(start)))
	}()

	// Extract and process transactions
	transactions, err := p.ExtractTransactions(ctx, block)
	if err != nil {
		return fmt.Errorf("failed to extract transactions: %w", err)
	}

	// Process validator rewards
	rewards, err := p.processValidatorRewards(ctx, solBlock)
	if err != nil {
		return fmt.Errorf("failed to process validator rewards: %w", err)
	}

	// Process program interactions
	programInteractions, err := p.extractProgramInteractions(ctx, solBlock)
	if err != nil {
		return fmt.Errorf("failed to extract program interactions: %w", err)
	}

	// Update chain state
	if err := p.UpdateChainState(ctx, block); err != nil {
		return fmt.Errorf("failed to update chain state: %w", err)
	}

	p.logger.Info("Solana block processed successfully",
		zap.String("blockhash", solBlock.Blockhash),
		zap.Uint64("slot", solBlock.Slot),
		zap.Int("transactions", len(transactions)),
		zap.Int("rewards", len(rewards)),
		zap.Int("program_interactions", len(programInteractions)))

	return nil
}

// ExtractTransactions extracts transactions from a Solana block
func (p *Processor) ExtractTransactions(ctx context.Context, block interface{}) ([]interface{}, error) {
	solBlock, ok := block.(*SolanaBlock)
	if !ok {
		return nil, fmt.Errorf("invalid block type for Solana processor")
	}

	transactions := make([]interface{}, 0, len(solBlock.Transactions))
	
	for i, tx := range solBlock.Transactions {
		// Process each transaction
		processedTx, err := p.processTransaction(ctx, tx, solBlock)
		if err != nil {
			p.logger.Warn("Failed to process transaction",
				zap.Strings("signatures", tx.Signatures),
				zap.Int("tx_index", i),
				zap.Error(err))
			continue
		}
		
		transactions = append(transactions, processedTx)
	}

	return transactions, nil
}

// UpdateChainState updates the Solana chain state
func (p *Processor) UpdateChainState(ctx context.Context, block interface{}) error {
	solBlock, ok := block.(*SolanaBlock)
	if !ok {
		return fmt.Errorf("invalid block type for Solana processor")
	}

	// Update latest slot info
	p.chainState.LatestSlot = solBlock.Slot
	p.chainState.LatestBlockhash = solBlock.Blockhash
	
	if solBlock.BlockTime != nil {
		p.chainState.LatestBlockTime = time.Unix(*solBlock.BlockTime, 0)
	}

	if solBlock.BlockHeight != nil {
		p.chainState.LatestBlockHeight = *solBlock.BlockHeight
	}

	// Update slot statistics
	p.chainState.TotalSlots++
	
	// Calculate slot timing
	if p.chainState.PreviousSlotTime != nil && solBlock.BlockTime != nil {
		currentTime := time.Unix(*solBlock.BlockTime, 0)
		slotDuration := currentTime.Sub(*p.chainState.PreviousSlotTime)
		
		// Update average slot time
		if p.chainState.AverageSlotTime == 0 {
			p.chainState.AverageSlotTime = slotDuration
		} else {
			// Simple moving average
			p.chainState.AverageSlotTime = (p.chainState.AverageSlotTime + slotDuration) / 2
		}
	}

	if solBlock.BlockTime != nil {
		blockTime := time.Unix(*solBlock.BlockTime, 0)
		p.chainState.PreviousSlotTime = &blockTime
	}

	// Update transaction count
	p.chainState.TotalTransactions += uint64(solBlock.TransactionCount)

	// Update fee statistics
	totalFees := p.calculateTotalFees(solBlock)
	p.chainState.TotalFees += totalFees

	// Track validator rewards
	if solBlock.Rewards != nil {
		for _, reward := range solBlock.Rewards {
			p.chainState.TotalRewards += uint64(reward.Lamports)
		}
		p.chainState.RewardEvents += uint64(len(solBlock.Rewards))
	}

	return nil
}

// processTransaction processes a single Solana transaction
func (p *Processor) processTransaction(ctx context.Context, tx SolanaTransaction, block *SolanaBlock) (*ProcessedSolTx, error) {
	processedTx := &ProcessedSolTx{
		Signatures:      tx.Signatures,
		AccountKeys:     tx.Message.AccountKeys,
		RecentBlockhash: tx.Message.RecentBlockhash,
		Instructions:    make([]ProcessedSolInstruction, len(tx.Message.Instructions)),
		Blockhash:       block.Blockhash,
		Slot:            block.Slot,
	}

	if block.BlockTime != nil {
		processedTx.Timestamp = time.Unix(*block.BlockTime, 0)
	}

	// Process instructions
	for i, instruction := range tx.Message.Instructions {
		processedInstr := ProcessedSolInstruction{
			ProgramIdIndex: instruction.ProgramIdIndex,
			Accounts:       instruction.Accounts,
			Data:           instruction.Data,
		}

		// Get program ID if available
		if instruction.ProgramIdIndex < len(tx.Message.AccountKeys) {
			processedInstr.ProgramId = tx.Message.AccountKeys[instruction.ProgramIdIndex]
		}

		processedTx.Instructions[i] = processedInstr
	}

	// Extract metadata if available
	if tx.Meta != nil {
		processedTx.Fee = tx.Meta.Fee
		processedTx.Success = tx.Meta.Err == nil
		processedTx.PreBalances = tx.Meta.PreBalances
		processedTx.PostBalances = tx.Meta.PostBalances
		processedTx.LogMessages = tx.Meta.LogMessages
	}

	return processedTx, nil
}

// processValidatorRewards processes validator rewards from the block
func (p *Processor) processValidatorRewards(ctx context.Context, block *SolanaBlock) ([]ProcessedSolReward, error) {
	if block.Rewards == nil {
		return nil, nil
	}

	rewards := make([]ProcessedSolReward, 0, len(block.Rewards))
	
	for _, reward := range block.Rewards {
		processedReward := ProcessedSolReward{
			ValidatorPubkey: reward.Pubkey,
			Lamports:        reward.Lamports,
			PostBalance:     reward.PostBalance,
			RewardType:      reward.RewardType,
			Commission:      reward.Commission,
			Slot:            block.Slot,
			Blockhash:       block.Blockhash,
		}

		if block.BlockTime != nil {
			processedReward.Timestamp = time.Unix(*block.BlockTime, 0)
		}

		rewards = append(rewards, processedReward)
	}

	return rewards, nil
}

// extractProgramInteractions extracts program interactions from transactions
func (p *Processor) extractProgramInteractions(ctx context.Context, block *SolanaBlock) ([]ProgramInteraction, error) {
	interactions := []ProgramInteraction{}
	
	for _, tx := range block.Transactions {
		for _, instruction := range tx.Message.Instructions {
			if instruction.ProgramIdIndex < len(tx.Message.AccountKeys) {
				programId := tx.Message.AccountKeys[instruction.ProgramIdIndex]
				
				interaction := ProgramInteraction{
					ProgramId:      programId,
					InstructionData: instruction.Data,
					Accounts:       make([]string, 0, len(instruction.Accounts)),
					Slot:           block.Slot,
					Blockhash:      block.Blockhash,
				}

				if block.BlockTime != nil {
					interaction.Timestamp = time.Unix(*block.BlockTime, 0)
				}

				// Add account pubkeys
				for _, accountIndex := range instruction.Accounts {
					if accountIndex < len(tx.Message.AccountKeys) {
						interaction.Accounts = append(interaction.Accounts, 
							tx.Message.AccountKeys[accountIndex])
					}
				}

				interactions = append(interactions, interaction)
			}
		}
	}

	return interactions, nil
}

// calculateTotalFees calculates total fees for all transactions in the block
func (p *Processor) calculateTotalFees(block *SolanaBlock) uint64 {
	totalFees := uint64(0)
	
	for _, tx := range block.Transactions {
		if tx.Meta != nil {
			totalFees += tx.Meta.Fee
		}
	}
	
	return totalFees
}

// GetChainState returns the current chain state
func (p *Processor) GetChainState() *SolanaChainState {
	return p.chainState
}

// SolanaChainState represents the current state of the Solana chain
type SolanaChainState struct {
	LatestSlot        uint64         `json:"latest_slot"`
	LatestBlockhash   string         `json:"latest_blockhash"`
	LatestBlockHeight uint64         `json:"latest_block_height"`
	LatestBlockTime   time.Time      `json:"latest_block_time"`
	TotalSlots        uint64         `json:"total_slots"`
	TotalTransactions uint64         `json:"total_transactions"`
	TotalFees         uint64         `json:"total_fees"`
	TotalRewards      uint64         `json:"total_rewards"`
	RewardEvents      uint64         `json:"reward_events"`
	AverageSlotTime   time.Duration  `json:"average_slot_time"`
	PreviousSlotTime  *time.Time     `json:"previous_slot_time,omitempty"`
}

// NewSolanaChainState creates a new Solana chain state
func NewSolanaChainState() *SolanaChainState {
	return &SolanaChainState{}
}

// ProcessedSolTx represents a processed Solana transaction
type ProcessedSolTx struct {
	Signatures      []string                   `json:"signatures"`
	AccountKeys     []string                   `json:"account_keys"`
	RecentBlockhash string                     `json:"recent_blockhash"`
	Instructions    []ProcessedSolInstruction  `json:"instructions"`
	Fee             uint64                     `json:"fee"`
	Success         bool                       `json:"success"`
	PreBalances     []uint64                   `json:"pre_balances,omitempty"`
	PostBalances    []uint64                   `json:"post_balances,omitempty"`
	LogMessages     []string                   `json:"log_messages,omitempty"`
	Blockhash       string                     `json:"blockhash"`
	Slot            uint64                     `json:"slot"`
	Timestamp       time.Time                  `json:"timestamp"`
}

// ProcessedSolInstruction represents a processed Solana instruction
type ProcessedSolInstruction struct {
	ProgramIdIndex int      `json:"program_id_index"`
	ProgramId      string   `json:"program_id"`
	Accounts       []int    `json:"accounts"`
	Data           string   `json:"data"`
}

// ProcessedSolReward represents a processed Solana validator reward
type ProcessedSolReward struct {
	ValidatorPubkey string     `json:"validator_pubkey"`
	Lamports        int64      `json:"lamports"`
	PostBalance     uint64     `json:"post_balance"`
	RewardType      string     `json:"reward_type"`
	Commission      *uint8     `json:"commission,omitempty"`
	Slot            uint64     `json:"slot"`
	Blockhash       string     `json:"blockhash"`
	Timestamp       time.Time  `json:"timestamp"`
}

// ProgramInteraction represents a Solana program interaction
type ProgramInteraction struct {
	ProgramId       string     `json:"program_id"`
	InstructionData string     `json:"instruction_data"`
	Accounts        []string   `json:"accounts"`
	Slot            uint64     `json:"slot"`
	Blockhash       string     `json:"blockhash"`
	Timestamp       time.Time  `json:"timestamp"`
}
