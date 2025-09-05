package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
	"github.com/PayRpc/Bitcoin-Sprint/internal/metrics"
	"go.uber.org/zap"
)

// BackfillService provides comprehensive historical data processing with persistent deduplication
type BackfillService struct {
	cfg         config.Config
	logger      *zap.Logger
	blockChan   chan blocks.BlockEvent
	mem         *mempool.Mempool
	processedTx map[string]bool // In-memory deduplication
	processedMu sync.RWMutex
	running     bool
	stopChan    chan struct{}
	lastID      string
	lastIDMu    sync.RWMutex
	failedTxs   []string
	failedMu    sync.RWMutex
	metrics     *BackfillMetrics
}

// BackfillMetrics tracks backfill performance and health
type BackfillMetrics struct {
	MessagesProcessed int64
	BlocksProcessed   int64
	TxsProcessed      int64
	DuplicatesSkipped int64
	FailedTxs         int64
	LastBackfillTime  time.Time
	BackfillDuration  time.Duration
	mu                sync.RWMutex
}

// NewBackfillService creates a new backfill service with persistent state
func NewBackfillService(cfg config.Config, blockChan chan blocks.BlockEvent, mem *mempool.Mempool, logger *zap.Logger) *BackfillService {
	bs := &BackfillService{
		cfg:         cfg,
		logger:      logger,
		blockChan:   blockChan,
		mem:         mem,
		processedTx: make(map[string]bool),
		stopChan:    make(chan struct{}),
		metrics:     &BackfillMetrics{},
	}

	// Load persistent state
	bs.loadPersistentState()

	return bs
}

// Start begins the backfill process with retry logic
func (bs *BackfillService) Start(ctx context.Context) error {
	if !bs.cfg.RPCEnabled {
		bs.logger.Info("RPC backfill disabled, skipping")
		return nil
	}

	bs.running = true
	bs.logger.Info("Starting backfill service",
		zap.String("rpc_url", bs.cfg.RPCURL),
		zap.Bool("skip_mempool", bs.cfg.RPCSkipMempool),
		zap.Int("batch_size", bs.cfg.RPCBatchSize),
		zap.String("last_id", bs.lastID))

	// Start background backfill process
	go bs.runBackfill(ctx)

	return nil
}

// Stop halts the backfill process and saves state
func (bs *BackfillService) Stop() {
	if !bs.running {
		return
	}

	bs.running = false
	close(bs.stopChan)

	// Save persistent state
	bs.savePersistentState()

	bs.logger.Info("Backfill service stopped")
}

// GetMetrics returns current backfill metrics
func (bs *BackfillService) GetMetrics() BackfillMetrics {
	bs.metrics.mu.RLock()
	defer bs.metrics.mu.RUnlock()

	return BackfillMetrics{
		MessagesProcessed: bs.metrics.MessagesProcessed,
		BlocksProcessed:   bs.metrics.BlocksProcessed,
		TxsProcessed:      bs.metrics.TxsProcessed,
		DuplicatesSkipped: bs.metrics.DuplicatesSkipped,
		FailedTxs:         bs.metrics.FailedTxs,
		LastBackfillTime:  bs.metrics.LastBackfillTime,
		BackfillDuration:  bs.metrics.BackfillDuration,
	}
}

// GetProcessedTxCount returns the number of processed transactions
func (bs *BackfillService) GetProcessedTxCount() int {
	bs.processedMu.RLock()
	defer bs.processedMu.RUnlock()
	return len(bs.processedTx)
}

// GetFailedTxs returns list of failed transaction IDs
func (bs *BackfillService) GetFailedTxs() []string {
	bs.failedMu.RLock()
	defer bs.failedMu.RUnlock()

	failed := make([]string, len(bs.failedTxs))
	copy(failed, bs.failedTxs)
	return failed
}

// runBackfill executes the backfill process with comprehensive error handling
func (bs *BackfillService) runBackfill(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	retryCount := 0
	maxRetries := bs.cfg.RPCRetryAttempts

	for bs.running {
		select {
		case <-ctx.Done():
			bs.logger.Info("Context cancelled, stopping backfill")
			return
		case <-bs.stopChan:
			bs.logger.Info("Stop signal received, stopping backfill")
			return
		case <-ticker.C:
			startTime := time.Now()

			bs.logger.Info("Starting scheduled backfill",
				zap.String("last_id", bs.lastID),
				zap.Int("retry_count", retryCount))

			// Execute backfill with retry logic
			err := bs.executeBackfill(ctx)
			duration := time.Since(startTime)

			// Update metrics
			bs.metrics.mu.Lock()
			bs.metrics.LastBackfillTime = time.Now()
			bs.metrics.BackfillDuration = duration
			bs.metrics.mu.Unlock()

			if err != nil {
				retryCount++
				bs.logger.Error("Backfill failed",
					zap.Error(err),
					zap.Int("retry_count", retryCount),
					zap.Duration("duration", duration))

				if retryCount >= maxRetries {
					bs.logger.Error("Max retries exceeded, stopping backfill")
					return
				}

				// Exponential backoff
				backoff := time.Duration(retryCount) * time.Minute
				if backoff > bs.cfg.RPCRetryMaxWait {
					backoff = bs.cfg.RPCRetryMaxWait
				}

				bs.logger.Info("Retrying backfill",
					zap.Duration("backoff", backoff))
				time.Sleep(backoff)
				continue
			}

			// Success - reset retry count
			retryCount = 0
			bs.logger.Info("Backfill completed successfully",
				zap.Duration("duration", duration))
		}
	}
}

// executeBackfill performs the actual backfill operation
func (bs *BackfillService) executeBackfill(ctx context.Context) error {
	rpcCfg := BitcoinRPCConfig{
		URL:           bs.cfg.RPCURL,
		Username:      bs.cfg.RPCUsername,
		Password:      bs.cfg.RPCPassword,
		Timeout:       bs.cfg.RPCTimeout,
		MaxBlocks:     100,
		MaxTxPerBlock: 10000,
		MaxTxWorkers:  bs.cfg.RPCWorkers,
		BatchSize:     bs.cfg.RPCBatchSize,
		Topic:         bs.cfg.RPCMessageTopic,
		RetryAttempts: bs.cfg.RPCRetryAttempts,
		RetryMaxWait:  bs.cfg.RPCRetryMaxWait,
		SkipMempool:   bs.cfg.RPCSkipMempool,
		FailedTxFile:  bs.cfg.RPCFailedTxFile,
		LastIDFile:    bs.cfg.RPCLastIDFile,
	}

	// Set last ID for continuation
	rpcCfg.LastID = bs.getLastID()

	messages, lastID, failedTxs, err := BitcoinBackfill(ctx, rpcCfg)
	if err != nil {
		return fmt.Errorf("bitcoin backfill failed: %w", err)
	}

	// Process messages and update state
	processedCount := bs.processMessages(messages)

	// Update failed transactions
	bs.updateFailedTxs(failedTxs)

	// Update last ID for next run
	if lastID != "" {
		bs.setLastID(lastID)
	}

	// Update metrics
	bs.metrics.mu.Lock()
	bs.metrics.MessagesProcessed += int64(len(messages))
	bs.metrics.TxsProcessed += int64(processedCount)
	bs.metrics.FailedTxs += int64(len(failedTxs))
	bs.metrics.mu.Unlock()

	bs.logger.Info("Backfill execution completed",
		zap.Int("messages", len(messages)),
		zap.Int("processed_txs", processedCount),
		zap.Int("failed_txs", len(failedTxs)),
		zap.String("last_id", lastID))

	return nil
}

// processMessages processes backfill messages into block events
func (bs *BackfillService) processMessages(messages []Message) int {
	processedCount := 0

	for _, msg := range messages {
		// Extract transaction data
		txID, ok := msg.Data["txid"].(string)
		if !ok {
			bs.logger.Warn("Message missing txid", zap.String("id", msg.ID))
			continue
		}

		// Check for duplicates
		if bs.isTxProcessed(txID) {
			bs.metrics.mu.Lock()
			bs.metrics.DuplicatesSkipped++
			bs.metrics.mu.Unlock()

			bs.logger.Debug("Duplicate transaction skipped",
				zap.String("txid", txID))
			continue
		}

		// Mark as processed
		bs.markTxProcessed(txID)
		processedCount++

		// Extract block information
		if blockHash, ok := msg.Data["blockhash"].(string); ok {
			source := "rpc-backfill"

			bs.logger.Info("Processing block transaction",
				zap.String("txid", txID),
				zap.String("block_hash", blockHash),
				zap.String("source", source))

			blockEvent := blocks.BlockEvent{
				Hash:        blockHash,
				Timestamp:   msg.Timestamp,
				DetectedAt:  time.Now(),
				RelayTimeMs: 0, // Historical data
				Source:      source,
				TxID:        txID,
			}

			// Send to existing block channel (non-blocking)
			select {
			case bs.blockChan <- blockEvent:
				bs.logger.Debug("Backfill block event sent",
					zap.String("txid", txID),
					zap.String("block_hash", blockHash))

				// Update metrics
				bs.metrics.mu.Lock()
				bs.metrics.BlocksProcessed++
				bs.metrics.mu.Unlock()

				// Record successful processing metric
				metrics.BlocksProcessed.WithLabelValues(source).Inc()
			default:
				bs.logger.Warn("Block channel full, skipping backfill event",
					zap.String("txid", txID))
			}
		}
	}

	return processedCount
}

// updateFailedTxs updates the list of failed transactions
func (bs *BackfillService) updateFailedTxs(failedTxs []string) {
	bs.failedMu.Lock()
	defer bs.failedMu.Unlock()

	// Append new failed transactions
	bs.failedTxs = append(bs.failedTxs, failedTxs...)

	// Save to file if configured
	if bs.cfg.RPCFailedTxFile != "" {
		bs.saveFailedTxsToFile()
	}
}

// saveFailedTxsToFile saves failed transactions to persistent storage
func (bs *BackfillService) saveFailedTxsToFile() {
	if bs.cfg.RPCFailedTxFile == "" {
		return
	}

	bs.failedMu.RLock()
	defer bs.failedMu.RUnlock()

	data, err := json.MarshalIndent(bs.failedTxs, "", "  ")
	if err != nil {
		bs.logger.Error("Failed to marshal failed txs", zap.Error(err))
		return
	}

	if err := os.WriteFile(bs.cfg.RPCFailedTxFile, data, 0644); err != nil {
		bs.logger.Error("Failed to save failed txs to file", zap.Error(err))
	}
}

// loadPersistentState loads processed transactions and last ID from disk
func (bs *BackfillService) loadPersistentState() {
	// Load last ID
	if bs.cfg.RPCLastIDFile != "" {
		if data, err := os.ReadFile(bs.cfg.RPCLastIDFile); err == nil {
			bs.lastID = strings.TrimSpace(string(data))
			bs.logger.Info("Loaded last ID from file",
				zap.String("last_id", bs.lastID))
		}
	}

	// Load processed transactions
	processedFile := filepath.Join(filepath.Dir(bs.cfg.RPCLastIDFile), "processed_txs.json")
	if data, err := os.ReadFile(processedFile); err == nil {
		var processed []string
		if err := json.Unmarshal(data, &processed); err == nil {
			bs.processedMu.Lock()
			for _, txID := range processed {
				bs.processedTx[txID] = true
			}
			bs.processedMu.Unlock()

			bs.logger.Info("Loaded processed transactions",
				zap.Int("count", len(processed)))
		}
	}

	// Load failed transactions
	if bs.cfg.RPCFailedTxFile != "" {
		if data, err := os.ReadFile(bs.cfg.RPCFailedTxFile); err == nil {
			var failed []string
			if err := json.Unmarshal(data, &failed); err == nil {
				bs.failedMu.Lock()
				bs.failedTxs = failed
				bs.failedMu.Unlock()

				bs.logger.Info("Loaded failed transactions",
					zap.Int("count", len(failed)))
			}
		}
	}
}

// savePersistentState saves current state to disk
func (bs *BackfillService) savePersistentState() {
	// Save last ID
	if bs.cfg.RPCLastIDFile != "" {
		if err := os.WriteFile(bs.cfg.RPCLastIDFile, []byte(bs.lastID), 0644); err != nil {
			bs.logger.Error("Failed to save last ID", zap.Error(err))
		}
	}

	// Save processed transactions
	processedFile := filepath.Join(filepath.Dir(bs.cfg.RPCLastIDFile), "processed_txs.json")
	bs.processedMu.RLock()
	processed := make([]string, 0, len(bs.processedTx))
	for txID := range bs.processedTx {
		processed = append(processed, txID)
	}
	bs.processedMu.RUnlock()

	if data, err := json.MarshalIndent(processed, "", "  "); err == nil {
		if err := os.WriteFile(processedFile, data, 0644); err != nil {
			bs.logger.Error("Failed to save processed txs", zap.Error(err))
		}
	}
}

// isTxProcessed checks if a transaction has been processed
func (bs *BackfillService) isTxProcessed(txID string) bool {
	bs.processedMu.RLock()
	defer bs.processedMu.RUnlock()
	return bs.processedTx[txID]
}

// markTxProcessed marks a transaction as processed
func (bs *BackfillService) markTxProcessed(txID string) {
	bs.processedMu.Lock()
	defer bs.processedMu.Unlock()
	bs.processedTx[txID] = true
}

// getLastID returns the last processed ID
func (bs *BackfillService) getLastID() string {
	bs.lastIDMu.RLock()
	defer bs.lastIDMu.RUnlock()
	return bs.lastID
}

// setLastID sets the last processed ID
func (bs *BackfillService) setLastID(lastID string) {
	bs.lastIDMu.Lock()
	defer bs.lastIDMu.Unlock()
	bs.lastID = lastID
}

// RunOnce executes a one-time backfill operation
func (bs *BackfillService) RunOnce(ctx context.Context) ([]Message, string, []string, error) {
	if !bs.cfg.RPCEnabled {
		return nil, "", nil, fmt.Errorf("RPC backfill disabled")
	}

	rpcCfg := BitcoinRPCConfig{
		URL:           bs.cfg.RPCURL,
		Username:      bs.cfg.RPCUsername,
		Password:      bs.cfg.RPCPassword,
		Timeout:       bs.cfg.RPCTimeout,
		MaxBlocks:     100,
		MaxTxPerBlock: 10000,
		MaxTxWorkers:  bs.cfg.RPCWorkers,
		BatchSize:     bs.cfg.RPCBatchSize,
		Topic:         bs.cfg.RPCMessageTopic,
		RetryAttempts: bs.cfg.RPCRetryAttempts,
		RetryMaxWait:  bs.cfg.RPCRetryMaxWait,
		SkipMempool:   bs.cfg.RPCSkipMempool,
		FailedTxFile:  bs.cfg.RPCFailedTxFile,
		LastIDFile:    bs.cfg.RPCLastIDFile,
		LastID:        bs.getLastID(),
	}

	messages, lastID, failedTxs, err := BitcoinBackfill(ctx, rpcCfg)
	if err != nil {
		return nil, "", nil, err
	}

	// Process messages
	processedCount := bs.processMessages(messages)

	// Update state
	bs.updateFailedTxs(failedTxs)
	if lastID != "" {
		bs.setLastID(lastID)
	}

	bs.logger.Info("One-time backfill completed",
		zap.Int("messages", len(messages)),
		zap.Int("processed_txs", processedCount),
		zap.Int("failed_txs", len(failedTxs)),
		zap.String("last_id", lastID))

	return messages, lastID, failedTxs, nil
}

// HealthCheck performs a health check on the backfill service
func (bs *BackfillService) HealthCheck() error {
	if !bs.running {
		return fmt.Errorf("backfill service is not running")
	}

	// Check if we can access the RPC endpoint
	rpcCfg := BitcoinRPCConfig{
		URL:      bs.cfg.RPCURL,
		Username: bs.cfg.RPCUsername,
		Password: bs.cfg.RPCPassword,
		Timeout:  5 * time.Second,
	}

	// Simple RPC call to test connectivity
	_, err := BitcoinRPCGetBlockCount(rpcCfg)
	if err != nil {
		return fmt.Errorf("RPC connectivity check failed: %w", err)
	}

	return nil
}

// GetStatus returns the current status of the backfill service
func (bs *BackfillService) GetStatus() map[string]interface{} {
	metrics := bs.GetMetrics()

	return map[string]interface{}{
		"running":             bs.running,
		"rpc_enabled":         bs.cfg.RPCEnabled,
		"last_id":             bs.getLastID(),
		"processed_tx_count":  bs.GetProcessedTxCount(),
		"failed_tx_count":     len(bs.GetFailedTxs()),
		"messages_processed":  metrics.MessagesProcessed,
		"blocks_processed":    metrics.BlocksProcessed,
		"txs_processed":       metrics.TxsProcessed,
		"duplicates_skipped":  metrics.DuplicatesSkipped,
		"last_backfill_time":  metrics.LastBackfillTime,
		"backfill_duration":   metrics.BackfillDuration,
	}
}
