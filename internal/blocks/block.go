package blocks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Chain represents supported blockchain networks
type Chain string

const (
	ChainBitcoin  Chain = "bitcoin"
	ChainEthereum Chain = "ethereum"
	ChainSolana   Chain = "solana"
	ChainLitecoin Chain = "litecoin"
	ChainDogecoin Chain = "dogecoin"
)

// BlockStatus represents the processing status of a block
type BlockStatus string

const (
	StatusPending    BlockStatus = "pending"
	StatusProcessing BlockStatus = "processing"
	StatusProcessed  BlockStatus = "processed"
	StatusFailed     BlockStatus = "failed"
	StatusOrphaned   BlockStatus = "orphaned"
)

// BlockEvent represents a generic blockchain event for the relay system
type BlockEvent struct {
	Hash        string    `json:"hash"`
	Height      uint32    `json:"height"`
	Timestamp   time.Time `json:"timestamp"`
	DetectedAt  time.Time `json:"detected_at"`
	RelayTimeMs float64   `json:"relay_time_ms"`
	Source      string    `json:"source"`
	TxID        string    `json:"txid,omitempty"`
	Tier        string    `json:"tier"`
	IsHeader    bool      `json:"is_header,omitempty"`
	Chain       Chain     `json:"chain"`
	Status      BlockStatus `json:"status"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}

// BitcoinBlock represents a Bitcoin block with all relevant data
type BitcoinBlock struct {
	Hash              string            `json:"hash"`
	Height            uint64            `json:"height"`
	Version           int32             `json:"version"`
	PreviousBlockHash string            `json:"previous_block_hash"`
	MerkleRoot        string            `json:"merkle_root"`
	Timestamp         time.Time         `json:"timestamp"`
	Nonce             uint32            `json:"nonce"`
	Difficulty        float64           `json:"difficulty"`
	ChainWork         string            `json:"chain_work"`
	Size              int64             `json:"size"`
	Weight            int64             `json:"weight"`
	TransactionCount  int               `json:"transaction_count"`
	Transactions      []BitcoinTx       `json:"transactions,omitempty"`
	Confirmations     int               `json:"confirmations"`
	Status            BlockStatus       `json:"status"`
	ProcessingTime    time.Duration     `json:"processing_time"`
	ValidationErrors  []string          `json:"validation_errors,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// EthereumBlock represents an Ethereum block with all relevant data
type EthereumBlock struct {
	Hash             string            `json:"hash"`
	Number           uint64            `json:"number"`
	ParentHash       string            `json:"parent_hash"`
	StateRoot        string            `json:"state_root"`
	TransactionsRoot string            `json:"transactions_root"`
	ReceiptsRoot     string            `json:"receipts_root"`
	Timestamp        time.Time         `json:"timestamp"`
	GasLimit         uint64            `json:"gas_limit"`
	GasUsed          uint64            `json:"gas_used"`
	BaseFeePerGas    *big.Int          `json:"base_fee_per_gas,omitempty"`
	Size             int64             `json:"size"`
	TransactionCount int               `json:"transaction_count"`
	Transactions     []EthereumTx      `json:"transactions,omitempty"`
	Uncles           []string          `json:"uncles,omitempty"`
	Status           BlockStatus       `json:"status"`
	ProcessingTime   time.Duration     `json:"processing_time"`
	ValidationErrors []string          `json:"validation_errors,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// SolanaBlock represents a Solana block with all relevant data
type SolanaBlock struct {
	Hash             string            `json:"hash"`
	Slot             uint64            `json:"slot"`
	ParentSlot       uint64            `json:"parent_slot"`
	Timestamp        time.Time         `json:"timestamp"`
	TransactionCount int               `json:"transaction_count"`
	Transactions     []SolanaTx        `json:"transactions,omitempty"`
	Status           BlockStatus       `json:"status"`
	ProcessingTime   time.Duration     `json:"processing_time"`
	ValidationErrors []string          `json:"validation_errors,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// Transaction interfaces for different chains
type BitcoinTx struct {
	Hash     string  `json:"hash"`
	Size     int     `json:"size"`
	Fee      int64   `json:"fee"`
	Inputs   int     `json:"inputs"`
	Outputs  int     `json:"outputs"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type EthereumTx struct {
	Hash     string   `json:"hash"`
	From     string   `json:"from"`
	To       *string  `json:"to,omitempty"`
	Value    *big.Int `json:"value"`
	Gas      uint64   `json:"gas"`
	GasPrice *big.Int `json:"gas_price"`
	Nonce    uint64   `json:"nonce"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type SolanaTx struct {
	Signature string `json:"signature"`
	Fee       uint64 `json:"fee"`
	Success   bool   `json:"success"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// BlockProcessor handles enterprise-grade block processing workflows
type BlockProcessor struct {
	logger        *zap.Logger
	validators    map[Chain]BlockValidator
	processors    map[Chain]ChainProcessor
	metrics       *ProcessingMetrics
	cache         BlockCache
	mu            sync.RWMutex
	config        ProcessorConfig
	shutdownChan  chan struct{}
	processingWG  sync.WaitGroup
}

// BlockValidator interface for chain-specific validation
type BlockValidator interface {
	ValidateBlock(ctx context.Context, block interface{}) error
	ValidateTransactions(ctx context.Context, block interface{}) error
	CheckConsensus(ctx context.Context, block interface{}) error
}

// ChainProcessor interface for chain-specific processing
type ChainProcessor interface {
	ProcessBlock(ctx context.Context, block interface{}) error
	ExtractTransactions(ctx context.Context, block interface{}) ([]interface{}, error)
	UpdateChainState(ctx context.Context, block interface{}) error
}

// BlockCache interface for high-performance block caching
type BlockCache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration) error
	Delete(key string) error
	Stats() CacheStats
}

// ProcessorConfig holds configuration for block processing
type ProcessorConfig struct {
	MaxConcurrentBlocks  int           `json:"max_concurrent_blocks"`
	ProcessingTimeout    time.Duration `json:"processing_timeout"`
	ValidationTimeout    time.Duration `json:"validation_timeout"`
	RetryAttempts        int           `json:"retry_attempts"`
	RetryDelay           time.Duration `json:"retry_delay"`
	CacheSize            int           `json:"cache_size"`
	CacheTTL             time.Duration `json:"cache_ttl"`
	EnableMetrics        bool          `json:"enable_metrics"`
	EnableCompression    bool          `json:"enable_compression"`
	MetricsInterval      time.Duration `json:"metrics_interval"`
}

// ProcessingMetrics tracks block processing performance
type ProcessingMetrics struct {
	mu                    sync.RWMutex
	TotalBlocks          int64     `json:"total_blocks"`
	ProcessedBlocks      int64     `json:"processed_blocks"`
	FailedBlocks         int64     `json:"failed_blocks"`
	OrphanedBlocks       int64     `json:"orphaned_blocks"`
	AverageProcessingTime time.Duration `json:"average_processing_time"`
	ThroughputPerSecond  float64   `json:"throughput_per_second"`
	LastProcessedAt      time.Time `json:"last_processed_at"`
	ChainStats           map[Chain]*ChainMetrics `json:"chain_stats"`
}

// ChainMetrics tracks per-chain processing statistics
type ChainMetrics struct {
	BlocksProcessed      int64         `json:"blocks_processed"`
	AverageBlockTime     time.Duration `json:"average_block_time"`
	AverageProcessingTime time.Duration `json:"average_processing_time"`
	LastBlockHeight      uint64        `json:"last_block_height"`
	LastBlockHash        string        `json:"last_block_hash"`
	ValidationFailures   int64         `json:"validation_failures"`
	ProcessingFailures   int64         `json:"processing_failures"`
}

// CacheStats provides cache performance metrics
type CacheStats struct {
	Hits     int64 `json:"hits"`
	Misses   int64 `json:"misses"`
	Evictions int64 `json:"evictions"`
	Size     int   `json:"size"`
	MaxSize  int   `json:"max_size"`
}

// NewBlockProcessor creates a production-ready block processor
func NewBlockProcessor(config ProcessorConfig, logger *zap.Logger) *BlockProcessor {
	processor := &BlockProcessor{
		logger:       logger,
		validators:   make(map[Chain]BlockValidator),
		processors:   make(map[Chain]ChainProcessor),
		config:       config,
		shutdownChan: make(chan struct{}),
		metrics: &ProcessingMetrics{
			ChainStats: make(map[Chain]*ChainMetrics),
		},
	}

	// Initialize chain-specific metrics
	for _, chain := range []Chain{ChainBitcoin, ChainEthereum, ChainSolana} {
		processor.metrics.ChainStats[chain] = &ChainMetrics{}
	}

	return processor
}

// ProcessBlockEvent processes a generic block event with chain-specific handling
func (bp *BlockProcessor) ProcessBlockEvent(ctx context.Context, event *BlockEvent) error {
	startTime := time.Now()
	
	bp.logger.Info("Processing block event",
		zap.String("chain", string(event.Chain)),
		zap.String("hash", event.Hash),
		zap.Uint32("height", event.Height))

	// Update metrics
	bp.updateMetrics(func(m *ProcessingMetrics) {
		m.TotalBlocks++
	})

	// Get chain-specific processor
	processor, exists := bp.processors[event.Chain]
	if !exists {
		return fmt.Errorf("no processor configured for chain: %s", event.Chain)
	}

	// Create processing context with timeout
	processCtx, cancel := context.WithTimeout(ctx, bp.config.ProcessingTimeout)
	defer cancel()

	// Convert event to chain-specific block structure
	blockData, err := bp.convertEventToBlock(event)
	if err != nil {
		bp.recordProcessingFailure(event.Chain, err)
		return fmt.Errorf("failed to convert event to block: %w", err)
	}

	// Validate block
	if validator, exists := bp.validators[event.Chain]; exists {
		if err := validator.ValidateBlock(processCtx, blockData); err != nil {
			bp.recordValidationFailure(event.Chain, err)
			return fmt.Errorf("block validation failed: %w", err)
		}
	}

	// Process block
	if err := processor.ProcessBlock(processCtx, blockData); err != nil {
		bp.recordProcessingFailure(event.Chain, err)
		return fmt.Errorf("block processing failed: %w", err)
	}

	// Record successful processing
	processingTime := time.Since(startTime)
	bp.recordProcessingSuccess(event.Chain, processingTime)
	
	bp.logger.Info("Block processed successfully",
		zap.String("chain", string(event.Chain)),
		zap.String("hash", event.Hash),
		zap.Duration("processing_time", processingTime))

	return nil
}

// RegisterValidator registers a chain-specific block validator
func (bp *BlockProcessor) RegisterValidator(chain Chain, validator BlockValidator) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.validators[chain] = validator
	bp.logger.Info("Registered block validator", zap.String("chain", string(chain)))
}

// RegisterProcessor registers a chain-specific block processor
func (bp *BlockProcessor) RegisterProcessor(chain Chain, processor ChainProcessor) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.processors[chain] = processor
	bp.logger.Info("Registered block processor", zap.String("chain", string(chain)))
}

// GetMetrics returns current processing metrics
func (bp *BlockProcessor) GetMetrics() *ProcessingMetrics {
	bp.metrics.mu.RLock()
	defer bp.metrics.mu.RUnlock()
	
	// Deep copy metrics to avoid race conditions
	metrics := &ProcessingMetrics{
		TotalBlocks:           bp.metrics.TotalBlocks,
		ProcessedBlocks:       bp.metrics.ProcessedBlocks,
		FailedBlocks:          bp.metrics.FailedBlocks,
		OrphanedBlocks:        bp.metrics.OrphanedBlocks,
		AverageProcessingTime: bp.metrics.AverageProcessingTime,
		ThroughputPerSecond:   bp.metrics.ThroughputPerSecond,
		LastProcessedAt:       bp.metrics.LastProcessedAt,
		ChainStats:            make(map[Chain]*ChainMetrics),
	}
	
	for chain, stats := range bp.metrics.ChainStats {
		metrics.ChainStats[chain] = &ChainMetrics{
			BlocksProcessed:       stats.BlocksProcessed,
			AverageBlockTime:      stats.AverageBlockTime,
			AverageProcessingTime: stats.AverageProcessingTime,
			LastBlockHeight:       stats.LastBlockHeight,
			LastBlockHash:         stats.LastBlockHash,
			ValidationFailures:    stats.ValidationFailures,
			ProcessingFailures:    stats.ProcessingFailures,
		}
	}
	
	return metrics
}

// Shutdown gracefully shuts down the block processor
func (bp *BlockProcessor) Shutdown(ctx context.Context) error {
	bp.logger.Info("Shutting down block processor")
	
	close(bp.shutdownChan)
	
	// Wait for all processing to complete with timeout
	done := make(chan struct{})
	go func() {
		bp.processingWG.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		bp.logger.Info("Block processor shutdown complete")
		return nil
	case <-ctx.Done():
		bp.logger.Warn("Block processor shutdown timed out")
		return ctx.Err()
	}
}

// Helper methods for metrics and internal operations

func (bp *BlockProcessor) updateMetrics(fn func(*ProcessingMetrics)) {
	bp.metrics.mu.Lock()
	defer bp.metrics.mu.Unlock()
	fn(bp.metrics)
}

func (bp *BlockProcessor) recordProcessingSuccess(chain Chain, processingTime time.Duration) {
	bp.updateMetrics(func(m *ProcessingMetrics) {
		m.ProcessedBlocks++
		m.LastProcessedAt = time.Now()
		
		if chainStats, exists := m.ChainStats[chain]; exists {
			chainStats.BlocksProcessed++
			chainStats.AverageProcessingTime = calculateAverageTime(
				chainStats.AverageProcessingTime,
				processingTime,
				chainStats.BlocksProcessed,
			)
		}
	})
}

func (bp *BlockProcessor) recordProcessingFailure(chain Chain, err error) {
	bp.logger.Error("Block processing failed",
		zap.String("chain", string(chain)),
		zap.Error(err))
	
	bp.updateMetrics(func(m *ProcessingMetrics) {
		m.FailedBlocks++
		if chainStats, exists := m.ChainStats[chain]; exists {
			chainStats.ProcessingFailures++
		}
	})
}

func (bp *BlockProcessor) recordValidationFailure(chain Chain, err error) {
	bp.logger.Error("Block validation failed",
		zap.String("chain", string(chain)),
		zap.Error(err))
	
	bp.updateMetrics(func(m *ProcessingMetrics) {
		if chainStats, exists := m.ChainStats[chain]; exists {
			chainStats.ValidationFailures++
		}
	})
}

func (bp *BlockProcessor) convertEventToBlock(event *BlockEvent) (interface{}, error) {
	switch event.Chain {
	case ChainBitcoin:
		return &BitcoinBlock{
			Hash:      event.Hash,
			Height:    uint64(event.Height),
			Timestamp: event.Timestamp,
			Status:    StatusPending,
		}, nil
	case ChainEthereum:
		return &EthereumBlock{
			Hash:      event.Hash,
			Number:    uint64(event.Height),
			Timestamp: event.Timestamp,
			Status:    StatusPending,
		}, nil
	case ChainSolana:
		return &SolanaBlock{
			Hash:      event.Hash,
			Slot:      uint64(event.Height),
			Timestamp: event.Timestamp,
			Status:    StatusPending,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported chain: %s", event.Chain)
	}
}

// Utility functions

func calculateAverageTime(currentAvg, newTime time.Duration, count int64) time.Duration {
	if count <= 1 {
		return newTime
	}
	total := currentAvg*time.Duration(count-1) + newTime
	return total / time.Duration(count)
}

// BlockHash calculates a standardized hash for any block type
func BlockHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// ValidateChain checks if a chain is supported
func ValidateChain(chain Chain) error {
	switch chain {
	case ChainBitcoin, ChainEthereum, ChainSolana, ChainLitecoin, ChainDogecoin:
		return nil
	default:
		return fmt.Errorf("unsupported chain: %s", chain)
	}
}

// SerializeBlock serializes a block to JSON with compression support
func SerializeBlock(block interface{}, compress bool) ([]byte, error) {
	data, err := json.Marshal(block)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal block: %w", err)
	}
	
	// TODO: Add compression support if enabled
	if compress {
		// Implement compression here when needed
	}
	
	return data, nil
}
