package messaging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Message represents a blockchain transaction message
type Message struct {
	ID        string                 `json:"id"`
	Topic     string                 `json:"topic"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	BlockHash string                 `json:"block_hash,omitempty"`
}

// BitcoinRPCConfig holds node connection and operational parameters
type BitcoinRPCConfig struct {
	URL      string        `json:"url"`
	Username string        `json:"username"`
	Password string        `json:"password"`
	Timeout  time.Duration `json:"timeout"`

	MaxBlocks     int `json:"max_blocks"`
	MaxTxPerBlock int `json:"max_tx_per_block"`
	MaxTxWorkers  int `json:"max_tx_workers"`
	BatchSize     int `json:"batch_size"`

	Topic string `json:"topic"`

	RetryAttempts int           `json:"retry_attempts"`
	RetryMaxWait  time.Duration `json:"retry_max_wait"`

	SkipMempool bool `json:"skip_mempool"`

	FailedTxFile string `json:"failed_tx_file"`
	LastIDFile   string `json:"last_id_file"`
	LastID       string `json:"last_id"`
}

// backfillMetrics provides monitoring
type backfillMetrics struct {
	messagesBackfilled prometheus.Counter
	txsSkipped        prometheus.Counter
	rpcErrors         prometheus.Counter
	rpcCalls          prometheus.Counter
	batchRequests     prometheus.Counter
	processTime       prometheus.Histogram
	failedTxsTotal    prometheus.Counter
}

// newMetrics initializes Prometheus metrics
func newMetrics() *backfillMetrics {
	return &backfillMetrics{
		messagesBackfilled: promauto.NewCounter(prometheus.CounterOpts{
			Name: "bitcoin_backfill_messages_total",
			Help: "Total number of messages backfilled",
		}),
		txsSkipped: promauto.NewCounter(prometheus.CounterOpts{
			Name: "bitcoin_backfill_transactions_skipped_total",
			Help: "Total number of transactions skipped (duplicates)",
		}),
		rpcErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "bitcoin_backfill_rpc_errors_total",
			Help: "Total number of RPC errors encountered",
		}),
		rpcCalls: promauto.NewCounter(prometheus.CounterOpts{
			Name: "bitcoin_backfill_rpc_calls_total",
			Help: "Total number of RPC calls made",
		}),
		batchRequests: promauto.NewCounter(prometheus.CounterOpts{
			Name: "bitcoin_backfill_batch_requests_total",
			Help: "Total number of batch RPC requests",
		}),
		processTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "bitcoin_backfill_process_duration_seconds",
			Help:    "Time spent processing transactions",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
		}),
		failedTxsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "bitcoin_backfill_failed_transactions_total",
			Help: "Total number of persistent failed transactions",
		}),
	}
}

// ValidateConfig ensures configuration parameters
func (cfg *BitcoinRPCConfig) ValidateConfig() error {
	if cfg.URL == "" {
		return fmt.Errorf("URL is required")
	}
	if cfg.Username == "" || cfg.Password == "" {
		return fmt.Errorf("username and password are required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxBlocks <= 0 {
		cfg.MaxBlocks = 100
	}
	if cfg.MaxTxPerBlock <= 0 {
		cfg.MaxTxPerBlock = 10000
	}
	if cfg.MaxTxWorkers <= 0 {
		cfg.MaxTxWorkers = 10
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	if cfg.RetryAttempts <= 0 {
		cfg.RetryAttempts = 3
	}
	if cfg.RetryMaxWait <= 0 {
		cfg.RetryMaxWait = 5 * time.Minute
	}
	return nil
}

// BitcoinBackfill orchestrates transaction retrieval
func BitcoinBackfill(ctx context.Context, cfg BitcoinRPCConfig) ([]Message, string, []string, error) {
	if err := cfg.ValidateConfig(); err != nil {
		return nil, "", nil, fmt.Errorf("invalid configuration: %w", err)
	}

	client := &http.Client{Timeout: cfg.Timeout}
	messages := make([]Message, 0)
	seenTxs := make(map[string]struct{})
	failedTxs := loadFailedTxs(cfg.FailedTxFile)
	lastID := loadLastID(cfg.LastIDFile)
	metrics := newMetrics()

	log.Printf("Starting Bitcoin backfill - Last ID: %s, Failed TXs: %d", lastID, len(failedTxs))

	// RPC call with retry
	rpcCall := func(method string, params []interface{}) (json.RawMessage, error) {
		start := time.Now()
		defer func() {
			metrics.processTime.Observe(time.Since(start).Seconds())
		}()

		var result json.RawMessage
		operation := func() error {
			metrics.rpcCalls.Inc()

			reqBody, err := json.Marshal(map[string]interface{}{
				"jsonrpc": "1.0",
				"id":      fmt.Sprintf("backfill-%d", time.Now().UnixNano()),
				"method":  method,
				"params":  params,
			})
			if err != nil {
				return fmt.Errorf("marshal RPC request: %w", err)
			}

			req, err := http.NewRequestWithContext(ctx, "POST", cfg.URL, bytes.NewReader(reqBody))
			if err != nil {
				return fmt.Errorf("create RPC request: %w", err)
			}
			req.SetBasicAuth(cfg.Username, cfg.Password)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", "bitcoin-backfill/1.0")

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("execute RPC: %w", err)
			}
			defer resp.Body.Close()

			var rpcResp struct {
				Result json.RawMessage `json:"result"`
				Error  *struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
				return fmt.Errorf("decode RPC response: %w", err)
			}
			if rpcResp.Error != nil {
				return fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
			}
			result = rpcResp.Result
			return nil
		}

		b := backoff.NewExponentialBackOff()
		b.MaxElapsedTime = cfg.RetryMaxWait
		b.InitialInterval = 100 * time.Millisecond
		b.MaxInterval = 10 * time.Second
		b.Multiplier = 2.0

		if err := backoff.Retry(operation, backoff.WithMaxRetries(backoff.WithContext(b, ctx), uint64(cfg.RetryAttempts))); err != nil {
			metrics.rpcErrors.Inc()
			return nil, fmt.Errorf("RPC call failed after %d attempts: %w", cfg.RetryAttempts, err)
		}
		return result, nil
	}

	// Batch RPC processing
	batchRpcCall := func(txIDs []string) ([]json.RawMessage, []string) {
		if len(txIDs) == 0 {
			return nil, nil
		}

		metrics.batchRequests.Inc()
		start := time.Now()
		defer func() { metrics.processTime.Observe(time.Since(start).Seconds()) }()

		batch := make([]map[string]interface{}, 0, len(txIDs))
		for i, txid := range txIDs {
			batch = append(batch, map[string]interface{}{
				"jsonrpc": "1.0",
				"id":      fmt.Sprintf("batch-%d-%s", i, txid),
				"method":  "getrawtransaction",
				"params":  []interface{}{txid, true},
			})
		}

		reqBody, err := json.Marshal(batch)
		if err != nil {
			metrics.rpcErrors.Inc()
			log.Printf("Failed to marshal batch request: %v", err)
			return nil, txIDs
		}

		req, err := http.NewRequestWithContext(ctx, "POST", cfg.URL, bytes.NewReader(reqBody))
		if err != nil {
			metrics.rpcErrors.Inc()
			log.Printf("Failed to create batch request: %v", err)
			return nil, txIDs
		}
		req.SetBasicAuth(cfg.Username, cfg.Password)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "bitcoin-backfill/1.0")

		resp, err := client.Do(req)
		if err != nil {
			metrics.rpcErrors.Inc()
			log.Printf("Batch RPC call failed: %v", err)
			return nil, txIDs
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			metrics.rpcErrors.Inc()
			body, _ := io.ReadAll(resp.Body)
			log.Printf("HTTP error %d: %s", resp.StatusCode, string(body))
			return nil, txIDs
		}

		var batchResp []struct {
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
			ID string `json:"id"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
			metrics.rpcErrors.Inc()
			log.Printf("Failed to decode batch response: %v", err)
			return nil, txIDs
		}

		results := make([]json.RawMessage, 0, len(txIDs))
		failed := make([]string, 0)
		failedSet := make(map[string]struct{})

		// Create a map to track request order
		requestOrder := make(map[string]int)
		for i, txid := range txIDs {
			requestOrder[fmt.Sprintf("batch-%d-%s", i, txid)] = i
		}

		// Process responses in any order using the ID field
		responseMap := make(map[int]json.RawMessage)
		for _, r := range batchResp {
			if r.Error != nil {
				// Extract txid from the ID field
				if idx, ok := requestOrder[r.ID]; ok && idx < len(txIDs) {
					failed = append(failed, txIDs[idx])
					failedSet[txIDs[idx]] = struct{}{}
					metrics.failedTxsTotal.Inc()
				}
				metrics.rpcErrors.Inc()
				log.Printf("Transaction failed: %s", r.Error.Message)
				continue
			}
			// Map successful response to its original position
			if idx, ok := requestOrder[r.ID]; ok {
				responseMap[idx] = r.Result
			}
		}

		// Build results in original request order
		for i := range txIDs {
			if result, ok := responseMap[i]; ok {
				results = append(results, result)
			}
		}

		log.Printf("Batch processed: %d success, %d failed", len(results), len(failed))
		return results, failed
	}

	// Process mempool if enabled
	if !cfg.SkipMempool {
		log.Printf("Processing mempool transactions...")
		rawMempool, err := rpcCall("getrawmempool", nil)
		if err != nil {
			log.Printf("Failed to fetch mempool (continuing): %v", err)
		} else {
			var txIDs []string
			if err := json.Unmarshal(rawMempool, &txIDs); err != nil {
				log.Printf("Failed to unmarshal mempool: %v", err)
			} else {
				allTxIDs := append(txIDs, failedTxs...)
				failedTxs = fetchTxs(ctx, batchRpcCall, allTxIDs, &messages, seenTxs, lastID, cfg, metrics, time.Time{})
				log.Printf("Mempool processed: %d transactions, %d messages generated", len(allTxIDs), len(messages))
			}
		}
	}

	// Sequential block processing
	startBlockHash := lastID
	if startBlockHash == "" {
		if chainInfo, err := rpcCall("getblockchaininfo", nil); err == nil {
			var chainData map[string]interface{}
			if err := json.Unmarshal(chainInfo, &chainData); err == nil {
				if hash, ok := chainData["bestblockhash"].(string); ok {
					startBlockHash = hash
					log.Printf("Starting from best block: %s", hash)
				}
			}
		}
	}

	currentHash := startBlockHash
	lastProcessedBlock := currentHash
	blocksProcessed := 0

	for currentHash != "" && blocksProcessed < cfg.MaxBlocks {
		select {
		case <-ctx.Done():
			log.Printf("Context cancelled, saving state...")
			saveFailedTxs(failedTxs, cfg.FailedTxFile)
			saveLastID(lastProcessedBlock, cfg.LastIDFile)
			return messages, lastProcessedBlock, failedTxs, ctx.Err()
		default:
		}

		blockRaw, err := rpcCall("getblock", []interface{}{currentHash, 2})
		if err != nil {
			log.Printf("Failed to fetch block %s: %v", currentHash, err)
			break
		}

		var block map[string]interface{}
		if err := json.Unmarshal(blockRaw, &block); err != nil {
			log.Printf("Failed to unmarshal block %s: %v", currentHash, err)
			break
		}

		blockTime := time.Unix(int64(block["time"].(float64)), 0).UTC()
		blockHeight := int64(block["height"].(float64))
		log.Printf("Processing block %d (%s) at %s", blockHeight, currentHash, blockTime.Format(time.RFC3339))

		if txs, ok := block["tx"].([]interface{}); ok {
			txIDs := getTxIDs(txs, cfg.MaxTxPerBlock)
			if len(txIDs) > 0 {
				newFailed := fetchTxs(ctx, batchRpcCall, txIDs, &messages, seenTxs, lastID, cfg, metrics, blockTime)
				failedTxs = append(failedTxs, newFailed...)
			}
		}

		lastProcessedBlock = currentHash
		blocksProcessed++

		if nextHash, ok := block["nextblockhash"].(string); ok {
			currentHash = nextHash
		} else {
			log.Printf("Reached tip of blockchain at block %d", blockHeight)
			break
		}

		if blocksProcessed%10 == 0 {
			saveLastID(lastProcessedBlock, cfg.LastIDFile)
		}
	}

	saveFailedTxs(failedTxs, cfg.FailedTxFile)
	saveLastID(lastProcessedBlock, cfg.LastIDFile)

	log.Printf("Backfill complete: %d messages, %d blocks, %d failed",
		len(messages), blocksProcessed, len(failedTxs))

	return messages, lastProcessedBlock, failedTxs, nil
}

// fetchTxs processes transactions concurrently
func fetchTxs(ctx context.Context, batchRpcCall func([]string) ([]json.RawMessage, []string),
	txIDs []string, messages *[]Message, seenTxs map[string]struct{}, lastID string,
	cfg BitcoinRPCConfig, metrics *backfillMetrics, blockTime time.Time) []string {

	if len(txIDs) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, cfg.MaxTxWorkers)
	mu := sync.Mutex{}
	allFailed := make([]string, 0)

	for i := 0; i < len(txIDs); i += cfg.BatchSize {
		end := i + cfg.BatchSize
		if end > len(txIDs) {
			end = len(txIDs)
		}

		batchTxIDs := make([]string, 0, end-i)
		for _, txid := range txIDs[i:end] {
			if txid == lastID {
				continue
			}
			if _, seen := seenTxs[txid]; seen {
				metrics.txsSkipped.Inc()
				continue
			}
			batchTxIDs = append(batchTxIDs, txid)
		}

		if len(batchTxIDs) == 0 {
			continue
		}

		wg.Add(1)
		go func(batch []string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			results, failed := batchRpcCall(batch)

			mu.Lock()
			defer mu.Unlock()
			allFailed = append(allFailed, failed...)

			// Map results to txIDs
			failedSet := make(map[string]struct{})
			for _, f := range failed {
				failedSet[f] = struct{}{}
			}

			for j, txid := range batch {
				if _, isFailed := failedSet[txid]; isFailed {
					continue
				}
				if j >= len(results) {
					break
				}

				seenTxs[txid] = struct{}{}
				var tx map[string]interface{}
				if err := json.Unmarshal(results[j], &tx); err != nil {
					allFailed = append(allFailed, txid)
					continue
				}

				msg := Message{
					ID:    txid,
					Topic: cfg.Topic,
					Data:  tx,
				}

				if !blockTime.IsZero() {
					msg.Timestamp = blockTime
					if blockHash, ok := tx["blockhash"].(string); ok {
						msg.BlockHash = blockHash
					}
				} else {
					msg.Timestamp = time.Now().UTC()
				}

				*messages = append(*messages, msg)
				metrics.messagesBackfilled.Inc()
			}

		}(batchTxIDs)
	}

	wg.Wait()
	return allFailed
}

// getTxIDs extracts transaction IDs from block data
func getTxIDs(txs []interface{}, maxTx int) []string {
	txIDs := make([]string, 0, len(txs))
	for i, tx := range txs {
		if i >= maxTx {
			break
		}
		switch v := tx.(type) {
		case string:
			txIDs = append(txIDs, v)
		case map[string]interface{}:
			if txid, ok := v["txid"].(string); ok {
				txIDs = append(txIDs, txid)
			}
		}
	}
	return txIDs
}

// File persistence utilities
func loadFailedTxs(filename string) []string {
	if filename == "" {
		return nil
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}
	var txs []string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			txs = append(txs, line)
		}
	}
	return txs
}

func saveFailedTxs(txs []string, filename string) {
	if filename == "" {
		return
	}
	data := strings.Join(txs, "\n")
	if err := os.WriteFile(filename, []byte(data), 0644); err != nil {
		log.Printf("Failed to save failed transactions: %v", err)
	}
}

func loadLastID(filename string) string {
	if filename == "" {
		return ""
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func saveLastID(id, filename string) {
	if filename == "" {
		return
	}
	if err := os.WriteFile(filename, []byte(id), 0644); err != nil {
		log.Printf("Failed to save last ID: %v", err)
	}
}

// BitcoinRPCGetBlockCount performs a simple RPC call to get the current block count
func BitcoinRPCGetBlockCount(cfg BitcoinRPCConfig) (int, error) {
	client := &http.Client{Timeout: cfg.Timeout}

	reqBody, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "1.0",
		"id":      "health-check",
		"method":  "getblockcount",
		"params":  []interface{}{},
	})
	if err != nil {
		return 0, fmt.Errorf("marshal RPC request: %w", err)
	}

	req, err := http.NewRequest("POST", cfg.URL, bytes.NewReader(reqBody))
	if err != nil {
		return 0, fmt.Errorf("create RPC request: %w", err)
	}
	req.SetBasicAuth(cfg.Username, cfg.Password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "bitcoin-backfill/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("execute RPC: %w", err)
	}
	defer resp.Body.Close()

	var rpcResp struct {
		Result int `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return 0, fmt.Errorf("decode RPC response: %w", err)
	}
	if rpcResp.Error != nil {
		return 0, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}
