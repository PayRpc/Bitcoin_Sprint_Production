package blocks

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"compress/gzip"
	"github.com/hashicorp/golang-lru"
	"go.uber.org/zap"
)

// ProcessorMetrics tracks block processing performance
type ProcessingMetrics struct {
	mu                    sync.RWMutex
	TotalBlocks           int64                   `json:"total_blocks"`
	ProcessedBlocks       int64                   `json:"processed_blocks"`
	FailedBlocks          int64                   `json:"failed_blocks"`
	OrphanedBlocks        int64                   `json:"orphaned_blocks"`
	ValidationErrors      int64                   `json:"validation_errors"`
	ProcessingErrors      int64                   `json:"processing_errors"`
	TotalProcessingTime   time.Duration           `json:"total_processing_time"`
	AvgProcessingTime     time.Duration           `json:"avg_processing_time"`
	AverageProcessingTime time.Duration           `json:"average_processing_time"`
	ThroughputPerSecond   float64                 `json:"throughput_per_second"`
	QueuedBlocks          int64                   `json:"queued_blocks"`
	RejectedBlocks        int64                   `json:"rejected_blocks"`
	RetryAttempts         int64                   `json:"retry_attempts"`
	CacheHits             int64                   `json:"cache_hits"`
	CacheMisses           int64                   `json:"cache_misses"`
	LastProcessedAt       time.Time               `json:"last_processed_at"`
	ChainStats            map[Chain]*ChainMetrics `json:"chain_stats"`
} // Chain represents supported blockchain networks
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
	Hash        string      `json:"hash"`
	Height      uint32      `json:"height"`
	Timestamp   time.Time   `json:"timestamp"`
	DetectedAt  time.Time   `json:"detected_at"`
	RelayTimeMs float64     `json:"relay_time_ms"`
	Source      string      `json:"source"`
	TxID        string      `json:"txid,omitempty"`
	Tier        string      `json:"tier"`
	IsHeader    bool        `json:"is_header,omitempty"`
	Chain       Chain       `json:"chain"`
	Status      BlockStatus `json:"status"`
	ProcessedAt *time.Time  `json:"processed_at,omitempty"`
}

// ErrAlreadyProcessing indicates a duplicate in-flight block event.
var ErrAlreadyProcessing = errors.New("block already processing")

// BitcoinBlock represents a Bitcoin block with all relevant data
type BitcoinBlock struct {
	Hash              string                 `json:"hash"`
	Height            uint64                 `json:"height"`
	Version           int32                  `json:"version"`
	PreviousBlockHash string                 `json:"previous_block_hash"`
	MerkleRoot        string                 `json:"merkle_root"`
	Timestamp         time.Time              `json:"timestamp"`
	Nonce             uint32                 `json:"nonce"`
	Difficulty        float64                `json:"difficulty"`
	ChainWork         string                 `json:"chain_work"`
	Size              int64                  `json:"size"`
	Weight            int64                  `json:"weight"`
	TransactionCount  int                    `json:"transaction_count"`
	Transactions      []BitcoinTx            `json:"transactions,omitempty"`
	Confirmations     int                    `json:"confirmations"`
	Status            BlockStatus            `json:"status"`
	ProcessingTime    time.Duration          `json:"processing_time"`
	ValidationErrors  []string               `json:"validation_errors,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// EthereumBlock represents an Ethereum block with all relevant data
type EthereumBlock struct {
	Hash             string                 `json:"hash"`
	Number           uint64                 `json:"number"`
	ParentHash       string                 `json:"parent_hash"`
	StateRoot        string                 `json:"state_root"`
	TransactionsRoot string                 `json:"transactions_root"`
	ReceiptsRoot     string                 `json:"receipts_root"`
	Timestamp        time.Time              `json:"timestamp"`
	GasLimit         uint64                 `json:"gas_limit"`
	GasUsed          uint64                 `json:"gas_used"`
	BaseFeePerGas    *big.Int               `json:"base_fee_per_gas,omitempty"`
	Size             int64                  `json:"size"`
	TransactionCount int                    `json:"transaction_count"`
	Transactions     []EthereumTx           `json:"transactions,omitempty"`
	Uncles           []string               `json:"uncles,omitempty"`
	Status           BlockStatus            `json:"status"`
	ProcessingTime   time.Duration          `json:"processing_time"`
	ValidationErrors []string               `json:"validation_errors,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// SolanaBlock represents a Solana block with all relevant data
type SolanaBlock struct {
	Hash             string                 `json:"hash"`
	Slot             uint64                 `json:"slot"`
	ParentSlot       uint64                 `json:"parent_slot"`
	Timestamp        time.Time              `json:"timestamp"`
	TransactionCount int                    `json:"transaction_count"`
	Transactions     []SolanaTx             `json:"transactions,omitempty"`
	Status           BlockStatus            `json:"status"`
	ProcessingTime   time.Duration          `json:"processing_time"`
	ValidationErrors []string               `json:"validation_errors,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// Transaction interfaces for different chains
type BitcoinTx struct {
	Hash     string                 `json:"hash"`
	Size     int                    `json:"size"`
	Fee      int64                  `json:"fee"`
	Inputs   int                    `json:"inputs"`
	Outputs  int                    `json:"outputs"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type EthereumTx struct {
	Hash     string                 `json:"hash"`
	From     string                 `json:"from"`
	To       *string                `json:"to,omitempty"`
	Value    *big.Int               `json:"value"`
	Gas      uint64                 `json:"gas"`
	GasPrice *big.Int               `json:"gas_price"`
	Nonce    uint64                 `json:"nonce"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type SolanaTx struct {
	Signature string                 `json:"signature"`
	Fee       uint64                 `json:"fee"`
	Success   bool                   `json:"success"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// BlockProcessor handles enterprise-grade block processing workflows
type BlockProcessor struct {
	logger           *zap.Logger
	validators       map[Chain]BlockValidator
	processors       map[Chain]ChainProcessor
	metrics          *ProcessingMetrics
	cache            BlockCache
	mu               sync.RWMutex
	config           ProcessorConfig
	shutdownChan     chan struct{}
	processingWG     sync.WaitGroup
	inflight         sync.Map // key -> struct{}{} to dedupe concurrent processing
	inflightRequests *sync.Map
	blockCache       *lru.Cache
	processedBlocks  *sync.Map
	blockQueue       chan *BlockEvent
	stopChan         chan struct{}
	semaphore        chan struct{}
	resultCache      map[string]ProcessingResult
	dedupLock        *sync.RWMutex
	statusCache      *sync.Map
	procMetrics      *ProcessingMetrics
	lastMetricsTime  time.Time

	// Atomic counters
	totalProcessed      int64
	totalFailed         int64
	totalRetries        int64
	inFlightCount       int64
	dedupHits           int64
	cacheHits           int64
	cacheMisses         int64
	validationErrors    int64
	rejectedBlocks      int64
	totalProcessingTime int64
	avgProcessingTime   int64
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
	MaxConcurrentBlocks int           `json:"max_concurrent_blocks"`
	ProcessingTimeout   time.Duration `json:"processing_timeout"`
	ValidationTimeout   time.Duration `json:"validation_timeout"`
	RetryAttempts       int           `json:"retry_attempts"`
	RetryDelay          time.Duration `json:"retry_delay"`
	CacheSize           int           `json:"cache_size"`
	CacheTTL            time.Duration `json:"cache_ttl"`
	EnableMetrics       bool          `json:"enable_metrics"`
	EnableCompression   bool          `json:"enable_compression"`
	MetricsInterval     time.Duration `json:"metrics_interval"`
	EnableDedup         bool          `json:"enable_dedup"`
	EnableStatusCache   bool          `json:"enable_status_cache"`
}

// ChainMetrics tracks per-chain processing statistics
type ChainMetrics struct {
	BlocksProcessed       int64         `json:"blocks_processed"`
	AverageBlockTime      time.Duration `json:"average_block_time"`
	AverageProcessingTime time.Duration `json:"average_processing_time"`
	LastBlockHeight       uint64        `json:"last_block_height"`
	LastBlockHash         string        `json:"last_block_hash"`
	ValidationFailures    int64         `json:"validation_failures"`
	ProcessingFailures    int64         `json:"processing_failures"`
}

// CacheStats provides cache performance metrics
type CacheStats struct {
	Hits      int64 `json:"hits"`
	Misses    int64 `json:"misses"`
	Evictions int64 `json:"evictions"`
	Size      int   `json:"size"`
	MaxSize   int   `json:"max_size"`
}

// NewBlockProcessor creates a new BlockProcessor with the specified configuration
func NewBlockProcessor(config ProcessorConfig, logger *zap.Logger) (*BlockProcessor, error) {
	if config.MaxConcurrentBlocks <= 0 {
		return nil, fmt.Errorf("max concurrent blocks must be positive")
	}
	if config.ProcessingTimeout <= 0 {
		return nil, fmt.Errorf("processing timeout must be positive")
	}

	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	processor := &BlockProcessor{
		config:           config,
		logger:           logger,
		inflightRequests: new(sync.Map),
		procMetrics: &ProcessingMetrics{
			ProcessedBlocks:     0,
			FailedBlocks:        0,
			ValidationErrors:    0,
			ProcessingErrors:    0,
			TotalProcessingTime: 0,
			AvgProcessingTime:   0,
			QueuedBlocks:        0,
			RejectedBlocks:      0,
			RetryAttempts:       0,
			CacheHits:           0,
			CacheMisses:         0,
		},
		lastMetricsTime: time.Now(),
		blockQueue:      make(chan *BlockEvent, config.MaxConcurrentBlocks*2),
		stopChan:        make(chan struct{}),
		dedupLock:       new(sync.RWMutex),
		statusCache:     new(sync.Map),
	}

	if config.CacheSize > 0 {
		processor.blockCache, _ = lru.New(config.CacheSize)
	}

	processor.semaphore = make(chan struct{}, config.MaxConcurrentBlocks)
	processor.resultCache = make(map[string]ProcessingResult)

	// Initialize atomic counters
	atomic.StoreInt64(&processor.totalProcessed, 0)
	atomic.StoreInt64(&processor.totalFailed, 0)
	atomic.StoreInt64(&processor.totalRetries, 0)
	atomic.StoreInt64(&processor.avgProcessingTime, 0)
	atomic.StoreInt64(&processor.inFlightCount, 0)

	return processor, nil
}

// ProcessingStatus represents the status of a processed block
type ProcessingStatus struct {
	Completed   bool      `json:"completed"`
	Success     bool      `json:"success"`
	ErrorMsg    string    `json:"error_msg,omitempty"`
	CompletedAt time.Time `json:"completed_at"`
}

// ProcessingResult contains the result of a block processing operation
type ProcessingResult struct {
	BlockHash   string      `json:"block_hash"`
	BlockHeight int64       `json:"block_height"`
	Chain       Chain       `json:"chain"`
	Success     bool        `json:"success"`
	Error       string      `json:"error,omitempty"`
	ProcessedAt time.Time   `json:"processed_at"`
	Data        interface{} `json:"data,omitempty"`
}

// Error definitions for block processing
var (
	ErrInvalidBlock     = errors.New("invalid block data")
	ErrBlockInProgress  = errors.New("block already being processed")
	ErrDuplicateBlock   = errors.New("duplicate block detected")
	ErrProcessingFailed = errors.New("block processing failed")
	ErrValidationFailed = errors.New("block validation failed")
	ErrUnsupportedChain = errors.New("unsupported blockchain")
	ErrTimeout          = errors.New("processing timeout")
)

// isDuplicate checks if a block has already been processed
func (bp *BlockProcessor) isDuplicate(blockKey string) bool {
	bp.dedupLock.RLock()
	defer bp.dedupLock.RUnlock()

	_, exists := bp.processedBlocks.Load(blockKey)
	return exists
}

// markAsProcessed marks a block as processed for deduplication
func (bp *BlockProcessor) markAsProcessed(blockKey string) {
	bp.dedupLock.Lock()
	defer bp.dedupLock.Unlock()

	bp.processedBlocks.Store(blockKey, time.Now())
}

// updateProcessingStatus updates the processing status of a block
func (bp *BlockProcessor) updateProcessingStatus(blockKey string, status *ProcessingStatus) {
	bp.statusCache.Store(blockKey, status)
}

// getProcessingStatus retrieves the processing status of a block
func (bp *BlockProcessor) getProcessingStatus(blockKey string) (*ProcessingStatus, bool) {
	value, exists := bp.statusCache.Load(blockKey)
	if !exists {
		return nil, false
	}

	status, ok := value.(*ProcessingStatus)
	return status, ok
}

// ValidateBlock validates a block before processing
func (bp *BlockProcessor) ValidateBlock(ctx context.Context, event *BlockEvent) (bool, error) {
	// This is a stub implementation - replace with real validation logic
	if event == nil || event.Hash == "" || event.Height == 0 {
		return false, ErrInvalidBlock
	}
	return true, nil
}

// processBlockWithRetry processes a block with configurable retries
func (bp *BlockProcessor) processBlockWithRetry(ctx context.Context, event *BlockEvent) (*ProcessingResult, error) {
	var result *ProcessingResult
	var err error

	for attempt := 0; attempt <= bp.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			atomic.AddInt64(&bp.totalRetries, 1)
			bp.logger.Debug("Retrying block processing",
				zap.String("chain", string(event.Chain)),
				zap.Uint32("height", event.Height),
				zap.Int("attempt", attempt))

			// Apply exponential backoff
			select {
			case <-time.After(bp.config.RetryDelay * time.Duration(attempt)):
				// Continue with retry
			case <-ctx.Done():
				return nil, fmt.Errorf("processing aborted during retry: %w", ctx.Err())
			}
		}

		result, err = bp.processBlock(ctx, event)
		if err == nil {
			return result, nil
		}

		// Check if we should retry based on error type
		if errors.Is(err, ErrInvalidBlock) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			// Don't retry for validation errors or cancellations
			break
		}
	}

	return nil, fmt.Errorf("processing failed after %d attempts: %w", bp.config.RetryAttempts+1, err)
}

// processBlock is the core block processing function
func (bp *BlockProcessor) processBlock(ctx context.Context, event *BlockEvent) (*ProcessingResult, error) {
	// This is a stub implementation - replace with real processing logic
	result := &ProcessingResult{
		BlockHash:   event.Hash,
		BlockHeight: int64(event.Height),
		Chain:       event.Chain,
		Success:     true,
		ProcessedAt: time.Now(),
	}

	// Simulate processing
	select {
	case <-time.After(50 * time.Millisecond):
		// Processing completed
	case <-ctx.Done():
		return nil, fmt.Errorf("processing interrupted: %w", ctx.Err())
	}

	return result, nil
}

// CompressData compresses data using gzip
func CompressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)

	_, err := gw.Write(data)
	if err != nil {
		return nil, err
	}

	if err := gw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// DecompressData decompresses gzipped data
func DecompressData(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	return io.ReadAll(gr)
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
