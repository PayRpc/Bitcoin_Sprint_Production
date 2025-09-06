package ethereum

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Processor implements ChainProcessor interface for Ethereum
type Processor struct {
	logger      *zap.Logger
	chainState  *EthereumChainState
}

// NewProcessor creates a new Ethereum processor
func NewProcessor(logger *zap.Logger) *Processor {
	return &Processor{
		logger:     logger,
		chainState: NewEthereumChainState(),
	}
}

// ProcessBlock processes an Ethereum block
func (p *Processor) ProcessBlock(ctx context.Context, block interface{}) error {
	ethBlock, ok := block.(*EthereumBlock)
	if !ok {
		return fmt.Errorf("invalid block type for Ethereum processor")
	}

	start := time.Now()
	defer func() {
		p.logger.Debug("Ethereum block processing completed",
			zap.String("hash", ethBlock.Hash),
			zap.Uint64("number", ethBlock.Number),
			zap.Duration("duration", time.Since(start)))
	}()

	// Extract and process transactions
	transactions, err := p.ExtractTransactions(ctx, block)
	if err != nil {
		return fmt.Errorf("failed to extract transactions: %w", err)
	}

	// Process smart contract events
	events, err := p.extractSmartContractEvents(ctx, ethBlock)
	if err != nil {
		return fmt.Errorf("failed to extract smart contract events: %w", err)
	}

	// Update chain state
	if err := p.UpdateChainState(ctx, block); err != nil {
		return fmt.Errorf("failed to update chain state: %w", err)
	}

	p.logger.Info("Ethereum block processed successfully",
		zap.String("hash", ethBlock.Hash),
		zap.Uint64("number", ethBlock.Number),
		zap.Int("transactions", len(transactions)),
		zap.Int("events", len(events)),
		zap.Uint64("gas_used", ethBlock.GasUsed))

	return nil
}

// ExtractTransactions extracts transactions from an Ethereum block
func (p *Processor) ExtractTransactions(ctx context.Context, block interface{}) ([]interface{}, error) {
	ethBlock, ok := block.(*EthereumBlock)
	if !ok {
		return nil, fmt.Errorf("invalid block type for Ethereum processor")
	}

	transactions := make([]interface{}, 0, len(ethBlock.Transactions))
	
	for i, tx := range ethBlock.Transactions {
		// Process each transaction
		processedTx, err := p.processTransaction(ctx, tx, ethBlock)
		if err != nil {
			p.logger.Warn("Failed to process transaction",
				zap.String("tx_hash", tx.Hash),
				zap.Int("tx_index", i),
				zap.Error(err))
			continue
		}
		
		transactions = append(transactions, processedTx)
	}

	return transactions, nil
}

// UpdateChainState updates the Ethereum chain state
func (p *Processor) UpdateChainState(ctx context.Context, block interface{}) error {
	ethBlock, ok := block.(*EthereumBlock)
	if !ok {
		return fmt.Errorf("invalid block type for Ethereum processor")
	}

	// Update latest block info
	p.chainState.LatestBlockHash = ethBlock.Hash
	p.chainState.LatestBlockNumber = ethBlock.Number
	p.chainState.LatestBlockTime = ethBlock.Timestamp

	// Update gas statistics
	p.chainState.TotalGasUsed += ethBlock.GasUsed
	p.chainState.BlockCount++

	// Calculate average gas usage
	if p.chainState.BlockCount > 0 {
		p.chainState.AverageGasUsed = p.chainState.TotalGasUsed / p.chainState.BlockCount
	}

	// Update transaction count
	p.chainState.TotalTransactions += uint64(ethBlock.TransactionCount)

	// Track base fee changes (EIP-1559)
	if ethBlock.BaseFeePerGas != nil {
		p.chainState.CurrentBaseFee = ethBlock.BaseFeePerGas
	}

	return nil
}

// processTransaction processes a single Ethereum transaction
func (p *Processor) processTransaction(ctx context.Context, tx EthereumTx, block *EthereumBlock) (*ProcessedEthTx, error) {
	processedTx := &ProcessedEthTx{
		Hash:        tx.Hash,
		From:        tx.From,
		To:          tx.To,
		Value:       tx.Value,
		Gas:         tx.Gas,
		GasPrice:    tx.GasPrice,
		Nonce:       tx.Nonce,
		BlockHash:   block.Hash,
		BlockNumber: block.Number,
		Timestamp:   block.Timestamp,
	}

	// Determine transaction type
	if tx.To == nil {
		processedTx.Type = "contract_creation"
	} else if tx.Value != nil && tx.Value.Cmp(big.NewInt(0)) > 0 {
		processedTx.Type = "value_transfer"
	} else {
		processedTx.Type = "contract_call"
	}

	// Calculate effective gas price
	if block.BaseFeePerGas != nil && tx.GasPrice != nil {
		// For legacy transactions, effective gas price is gas price
		// For EIP-1559 transactions, would need priority fee calculation
		processedTx.EffectiveGasPrice = tx.GasPrice
	}

	return processedTx, nil
}

// extractSmartContractEvents extracts smart contract events from the block
func (p *Processor) extractSmartContractEvents(ctx context.Context, block *EthereumBlock) ([]SmartContractEvent, error) {
	var events []SmartContractEvent

	// For each transaction, check if it's a contract interaction
	for _, tx := range block.Transactions {
		if tx.To != nil {
			// This would typically require receipt data to get actual events
			// For now, we'll create placeholder events for contract calls
			if p.isContractAddress(*tx.To) {
				event := SmartContractEvent{
					TxHash:          tx.Hash,
					ContractAddress: *tx.To,
					BlockNumber:     block.Number,
					BlockHash:       block.Hash,
					Timestamp:       block.Timestamp,
					EventType:       "contract_interaction",
				}
				events = append(events, event)
			}
		}
	}

	return events, nil
}

// isContractAddress checks if an address is likely a contract address
// This is a simplified check - in practice, you'd query the blockchain
func (p *Processor) isContractAddress(address string) bool {
	// Simple heuristic: if address doesn't start with common EOA patterns
	// In practice, you'd check if the address has code
	return !strings.HasSuffix(address, "0000") // Very basic heuristic
}

// GetChainState returns the current chain state
func (p *Processor) GetChainState() *EthereumChainState {
	return p.chainState
}

// EthereumChainState represents the current state of the Ethereum chain
type EthereumChainState struct {
	LatestBlockHash     string    `json:"latest_block_hash"`
	LatestBlockNumber   uint64    `json:"latest_block_number"`
	LatestBlockTime     time.Time `json:"latest_block_time"`
	TotalGasUsed        uint64    `json:"total_gas_used"`
	AverageGasUsed      uint64    `json:"average_gas_used"`
	BlockCount          uint64    `json:"block_count"`
	TotalTransactions   uint64    `json:"total_transactions"`
	CurrentBaseFee      *big.Int  `json:"current_base_fee,omitempty"`
}

// NewEthereumChainState creates a new Ethereum chain state
func NewEthereumChainState() *EthereumChainState {
	return &EthereumChainState{
		CurrentBaseFee: big.NewInt(0),
	}
}

// ProcessedEthTx represents a processed Ethereum transaction
type ProcessedEthTx struct {
	Hash               string    `json:"hash"`
	From               string    `json:"from"`
	To                 *string   `json:"to,omitempty"`
	Value              *big.Int  `json:"value"`
	Gas                uint64    `json:"gas"`
	GasPrice           *big.Int  `json:"gas_price"`
	EffectiveGasPrice  *big.Int  `json:"effective_gas_price,omitempty"`
	Nonce              uint64    `json:"nonce"`
	Type               string    `json:"type"`
	BlockHash          string    `json:"block_hash"`
	BlockNumber        uint64    `json:"block_number"`
	Timestamp          time.Time `json:"timestamp"`
}

// SmartContractEvent represents a smart contract event
type SmartContractEvent struct {
	TxHash          string    `json:"tx_hash"`
	ContractAddress string    `json:"contract_address"`
	BlockNumber     uint64    `json:"block_number"`
	BlockHash       string    `json:"block_hash"`
	Timestamp       time.Time `json:"timestamp"`
	EventType       string    `json:"event_type"`
	Data            string    `json:"data,omitempty"`
}
