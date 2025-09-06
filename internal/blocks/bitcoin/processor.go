package bitcoin

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Processor implements ChainProcessor interface for Bitcoin
type Processor struct {
	logger *zap.Logger
}

// NewProcessor creates a new Bitcoin processor
func NewProcessor(logger *zap.Logger) *Processor {
	return &Processor{
		logger: logger,
	}
}

// ProcessBlock processes a Bitcoin block
func (p *Processor) ProcessBlock(ctx context.Context, block interface{}) error {
	bitcoinBlock, ok := block.(*BitcoinBlock)
	if !ok {
		return fmt.Errorf("invalid block type for Bitcoin processor")
	}

	startTime := time.Now()

	p.logger.Info("Processing Bitcoin block",
		zap.String("hash", bitcoinBlock.Hash),
		zap.Uint64("height", bitcoinBlock.Height),
		zap.Int("tx_count", bitcoinBlock.TransactionCount))

	// Process transactions
	if bitcoinBlock.Transactions != nil {
		for i, tx := range bitcoinBlock.Transactions {
			if err := p.processTransaction(ctx, tx); err != nil {
				return fmt.Errorf("failed to process transaction %d: %w", i, err)
			}
		}
	}

	// Update chain state would go here
	if err := p.UpdateChainState(ctx, block); err != nil {
		return fmt.Errorf("failed to update chain state: %w", err)
	}

	processingTime := time.Since(startTime)
	p.logger.Info("Bitcoin block processed successfully",
		zap.String("hash", bitcoinBlock.Hash),
		zap.Duration("processing_time", processingTime))

	return nil
}

// ExtractTransactions extracts transactions from a Bitcoin block
func (p *Processor) ExtractTransactions(ctx context.Context, block interface{}) ([]interface{}, error) {
	bitcoinBlock, ok := block.(*BitcoinBlock)
	if !ok {
		return nil, fmt.Errorf("invalid block type for Bitcoin processor")
	}

	if bitcoinBlock.Transactions == nil {
		return []interface{}{}, nil
	}

	transactions := make([]interface{}, len(bitcoinBlock.Transactions))
	for i, tx := range bitcoinBlock.Transactions {
		transactions[i] = tx
	}

	p.logger.Debug("Extracted Bitcoin transactions",
		zap.String("block_hash", bitcoinBlock.Hash),
		zap.Int("tx_count", len(transactions)))

	return transactions, nil
}

// UpdateChainState updates the Bitcoin chain state
func (p *Processor) UpdateChainState(ctx context.Context, block interface{}) error {
	bitcoinBlock, ok := block.(*BitcoinBlock)
	if !ok {
		return fmt.Errorf("invalid block type for Bitcoin processor")
	}

	// This would typically update:
	// - UTXO set
	// - Chain tip
	// - Difficulty adjustments
	// - Mempool cleanup
	// For now, we'll just log the update

	p.logger.Debug("Updating Bitcoin chain state",
		zap.String("hash", bitcoinBlock.Hash),
		zap.Uint64("height", bitcoinBlock.Height),
		zap.Float64("difficulty", bitcoinBlock.Difficulty))

	// Simulate chain state update processing
	select {
	case <-time.After(10 * time.Millisecond):
		// Chain state updated
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// processTransaction processes a single Bitcoin transaction
func (p *Processor) processTransaction(ctx context.Context, tx BitcoinTx) error {
	p.logger.Debug("Processing Bitcoin transaction",
		zap.String("hash", tx.Hash),
		zap.Int("size", tx.Size),
		zap.Int64("fee", tx.Fee))

	// Transaction processing logic would go here:
	// - Validate inputs/outputs
	// - Update UTXO set
	// - Calculate fees
	// - Update mempool

	// Simulate processing time
	select {
	case <-time.After(time.Millisecond):
		// Transaction processed
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}
