package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// GenericRelay implements RelayClient for generic blockchain networks using JSON-RPC
type GenericRelay struct {
	cfg    config.Config
	logger *zap.Logger

	// HTTP client for JSON-RPC calls
	httpClient *http.Client

	// WebSocket connections for real-time data
	wsConnections []*websocket.Conn
	wsConnMu      sync.RWMutex
	wsConnected   atomic.Bool

	// Block streaming
	blockChan chan blocks.BlockEvent

	// Configuration
	relayConfig RelayConfig

	// Health and metrics
	health    *HealthStatus
	healthMu  sync.RWMutex
	metrics   *RelayMetrics
	metricsMu sync.RWMutex

	// Request tracking for async operations
	requestID   int64
	pendingReqs map[int64]chan *GenericResponse
	reqMu       sync.RWMutex

	// Network-specific configuration
	networkType string
	rpcMethods  GenericRPCMethods
}

// GenericResponse represents a JSON-RPC response
type GenericResponse struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *GenericError   `json:"error,omitempty"`
}

// GenericError represents a JSON-RPC error
type GenericError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// GenericRPCMethods defines the RPC methods for different blockchain networks
type GenericRPCMethods struct {
	GetLatestBlock     string // e.g., "eth_getBlockByNumber", "getblock", "get_block"
	GetBlockByHash     string // e.g., "eth_getBlockByHash", "getblock", "get_block"
	GetBlockByHeight   string // e.g., "eth_getBlockByNumber", "getblockcount", "get_block_count"
	GetNetworkInfo     string // e.g., "net_version", "getnetworkinfo", "get_info"
	GetPeerCount       string // e.g., "net_peerCount", "getconnectioncount", "get_connections"
	GetSyncStatus      string // e.g., "eth_syncing", "getblockchaininfo", "get_sync_info"
	SubscribeBlocks    string // e.g., "eth_subscribe", "zmq", "subscribe"
	GetTransactionPool string // e.g., "txpool_status", "getrawmempool", "get_pending_txs"
}

// GenericBlock represents a generic blockchain block structure
type GenericBlock struct {
	Hash         string      `json:"hash"`
	Height       uint64      `json:"height,omitempty"`
	Number       interface{} `json:"number,omitempty"`    // Can be string or number
	Timestamp    interface{} `json:"timestamp,omitempty"` // Can be string or number
	Time         interface{} `json:"time,omitempty"`      // Alternative timestamp field
	ParentHash   string      `json:"parentHash,omitempty"`
	PreviousHash string      `json:"previousblockhash,omitempty"`
	Size         interface{} `json:"size,omitempty"`
	Transactions interface{} `json:"transactions,omitempty"` // Array or count
}

// NetworkTypeConfig defines supported generic network configurations
var NetworkTypeConfigs = map[string]GenericRPCMethods{
	"ethereum-like": {
		GetLatestBlock:     "eth_getBlockByNumber",
		GetBlockByHash:     "eth_getBlockByHash",
		GetBlockByHeight:   "eth_getBlockByNumber",
		GetNetworkInfo:     "net_version",
		GetPeerCount:       "net_peerCount",
		GetSyncStatus:      "eth_syncing",
		SubscribeBlocks:    "eth_subscribe",
		GetTransactionPool: "txpool_status",
	},
	"bitcoin-like": {
		GetLatestBlock:     "getbestblockhash",
		GetBlockByHash:     "getblock",
		GetBlockByHeight:   "getblockhash",
		GetNetworkInfo:     "getnetworkinfo",
		GetPeerCount:       "getconnectioncount",
		GetSyncStatus:      "getblockchaininfo",
		SubscribeBlocks:    "zmq",
		GetTransactionPool: "getrawmempool",
	},
	"cosmos-like": {
		GetLatestBlock:     "block",
		GetBlockByHash:     "block_by_hash",
		GetBlockByHeight:   "block",
		GetNetworkInfo:     "status",
		GetPeerCount:       "net_info",
		GetSyncStatus:      "status",
		SubscribeBlocks:    "subscribe",
		GetTransactionPool: "unconfirmed_txs",
	},
	"substrate-like": {
		GetLatestBlock:     "chain_getBlock",
		GetBlockByHash:     "chain_getBlock",
		GetBlockByHeight:   "chain_getBlockHash",
		GetNetworkInfo:     "system_chain",
		GetPeerCount:       "system_health",
		GetSyncStatus:      "system_syncState",
		SubscribeBlocks:    "chain_subscribeNewHeads",
		GetTransactionPool: "author_pendingExtrinsics",
	},
}

// NewGenericRelay creates a new generic relay client
func NewGenericRelay(cfg config.Config, logger *zap.Logger, networkType string) *GenericRelay {
	rpcMethods, exists := NetworkTypeConfigs[networkType]
	if !exists {
		// Default to ethereum-like if unknown
		rpcMethods = NetworkTypeConfigs["ethereum-like"]
		networkType = "ethereum-like"
	}

	relayConfig := RelayConfig{
		Network:           "generic-" + networkType,
		Endpoints:         []string{"http://localhost:8545", "ws://localhost:8546"}, // Default endpoints
		Timeout:           30 * time.Second,
		RetryAttempts:     3,
		RetryDelay:        2 * time.Second,
		MaxConcurrency:    4,
		BufferSize:        1000,
		EnableCompression: false,
	}

	return &GenericRelay{
		cfg:         cfg,
		logger:      logger,
		networkType: networkType,
		rpcMethods:  rpcMethods,
		relayConfig: relayConfig,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		blockChan:   make(chan blocks.BlockEvent, 1000),
		pendingReqs: make(map[int64]chan *GenericResponse),
		health: &HealthStatus{
			IsHealthy:       false,
			ConnectionState: "disconnected",
		},
		metrics: &RelayMetrics{},
	}
}

// Connect establishes connections to the generic blockchain network
func (gr *GenericRelay) Connect(ctx context.Context) error {
	gr.logger.Info("Connecting to generic blockchain network",
		zap.String("network_type", gr.networkType),
		zap.Strings("endpoints", gr.relayConfig.Endpoints))

	// Test HTTP connection first
	if err := gr.testHTTPConnection(); err != nil {
		gr.logger.Warn("HTTP connection test failed", zap.Error(err))
	}

	// Try to establish WebSocket connections for real-time data
	for _, endpoint := range gr.relayConfig.Endpoints {
		if strings.HasPrefix(endpoint, "ws://") || strings.HasPrefix(endpoint, "wss://") {
			go gr.connectWebSocket(ctx, endpoint)
		}
	}

	gr.updateHealth(true, "connected", nil)
	return nil
}

// Disconnect closes all connections
func (gr *GenericRelay) Disconnect() error {
	gr.wsConnMu.Lock()
	defer gr.wsConnMu.Unlock()

	for _, conn := range gr.wsConnections {
		conn.Close()
	}
	gr.wsConnections = nil

	gr.wsConnected.Store(false)
	gr.updateHealth(false, "disconnected", nil)

	gr.logger.Info("Disconnected from generic blockchain network")
	return nil
}

// IsConnected returns true if at least one connection is active
func (gr *GenericRelay) IsConnected() bool {
	// Check if HTTP is working by testing connection
	if err := gr.testHTTPConnection(); err == nil {
		return true
	}

	// Check WebSocket connections
	gr.wsConnMu.RLock()
	defer gr.wsConnMu.RUnlock()
	return gr.wsConnected.Load() && len(gr.wsConnections) > 0
}

// StreamBlocks streams blocks from the blockchain
func (gr *GenericRelay) StreamBlocks(ctx context.Context, blockChan chan<- blocks.BlockEvent) error {
	if !gr.IsConnected() {
		return fmt.Errorf("not connected to blockchain network")
	}

	// Start block polling since generic networks may not support streaming
	go gr.pollBlocks(ctx)

	// Forward blocks from internal channel to provided channel
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case block := <-gr.blockChan:
				select {
				case blockChan <- block:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return nil
}

// GetLatestBlock returns the latest block
func (gr *GenericRelay) GetLatestBlock() (*blocks.BlockEvent, error) {
	if !gr.IsConnected() {
		return nil, fmt.Errorf("not connected to blockchain network")
	}

	var params []interface{}
	if gr.networkType == "ethereum-like" {
		params = []interface{}{"latest", false}
	} else if gr.networkType == "bitcoin-like" {
		// For Bitcoin-like, first get the best block hash
		hashResp, err := gr.makeHTTPRequest(gr.rpcMethods.GetLatestBlock, []interface{}{})
		if err != nil {
			return nil, fmt.Errorf("failed to get latest block hash: %w", err)
		}

		var blockHash string
		if err := json.Unmarshal(hashResp.Result, &blockHash); err != nil {
			return nil, fmt.Errorf("failed to parse block hash: %w", err)
		}

		// Then get the block by hash
		return gr.GetBlockByHash(blockHash)
	}

	response, err := gr.makeHTTPRequest(gr.rpcMethods.GetLatestBlock, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest block: %w", err)
	}

	return gr.parseBlockResponse(response.Result)
}

// GetBlockByHash retrieves a block by its hash
func (gr *GenericRelay) GetBlockByHash(hash string) (*blocks.BlockEvent, error) {
	if !gr.IsConnected() {
		return nil, fmt.Errorf("not connected to blockchain network")
	}

	var params []interface{}
	if gr.networkType == "ethereum-like" {
		params = []interface{}{hash, false}
	} else {
		params = []interface{}{hash}
	}

	resp, err := gr.makeHTTPRequest(gr.rpcMethods.GetBlockByHash, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get block by hash: %w", err)
	}

	return gr.parseBlockResponse(resp.Result)
}

// GetBlockByHeight retrieves a block by its height
func (gr *GenericRelay) GetBlockByHeight(height uint64) (*blocks.BlockEvent, error) {
	if !gr.IsConnected() {
		return nil, fmt.Errorf("not connected to blockchain network")
	}

	var params []interface{}
	if gr.networkType == "ethereum-like" {
		params = []interface{}{fmt.Sprintf("0x%x", height), false}
	} else if gr.networkType == "bitcoin-like" {
		// For Bitcoin-like, first get block hash by height
		hashResp, err := gr.makeHTTPRequest("getblockhash", []interface{}{height})
		if err != nil {
			return nil, fmt.Errorf("failed to get block hash for height %d: %w", height, err)
		}

		var blockHash string
		if err := json.Unmarshal(hashResp.Result, &blockHash); err != nil {
			return nil, fmt.Errorf("failed to parse block hash: %w", err)
		}

		return gr.GetBlockByHash(blockHash)
	} else {
		params = []interface{}{height}
	}

	response, err := gr.makeHTTPRequest(gr.rpcMethods.GetBlockByHeight, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get block by height: %w", err)
	}

	return gr.parseBlockResponse(response.Result)
}

// GetNetworkInfo returns network information
func (gr *GenericRelay) GetNetworkInfo() (*NetworkInfo, error) {
	if !gr.IsConnected() {
		return nil, fmt.Errorf("not connected to blockchain network")
	}

	_, err := gr.makeHTTPRequest(gr.rpcMethods.GetNetworkInfo, []interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to get network info: %w", err)
	}

	networkInfo := &NetworkInfo{
		Network:     gr.relayConfig.Network,
		Timestamp:   time.Now(),
		PeerCount:   gr.GetPeerCount(),
		BlockHeight: 0, // Will be filled from latest block
	}

	// Try to get latest block height
	if latestBlock, err := gr.GetLatestBlock(); err == nil {
		networkInfo.BlockHeight = uint64(latestBlock.Height)
	}

	return networkInfo, nil
}

// GetPeerCount returns the number of connected peers
func (gr *GenericRelay) GetPeerCount() int {
	response, err := gr.makeHTTPRequest(gr.rpcMethods.GetPeerCount, []interface{}{})
	if err != nil {
		return 0
	}

	var peerCount interface{}
	if err := json.Unmarshal(response.Result, &peerCount); err != nil {
		return 0
	}

	// Handle different response formats
	switch v := peerCount.(type) {
	case float64:
		return int(v)
	case string:
		// For hex strings like "0x5"
		if strings.HasPrefix(v, "0x") {
			if val, err := fmt.Sscanf(v, "0x%x", &peerCount); err == nil && val == 1 {
				return peerCount.(int)
			}
		}
		return 0
	default:
		return 0
	}
}

// GetSyncStatus returns synchronization status
func (gr *GenericRelay) GetSyncStatus() (*SyncStatus, error) {
	response, err := gr.makeHTTPRequest(gr.rpcMethods.GetSyncStatus, []interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to get sync status: %w", err)
	}

	syncStatus := &SyncStatus{
		IsSyncing:     false,
		CurrentHeight: 0,
		HighestHeight: 0,
		SyncProgress:  1.0,
	}

	// Parse response based on network type
	if gr.networkType == "ethereum-like" {
		var syncResult interface{}
		if err := json.Unmarshal(response.Result, &syncResult); err == nil {
			if syncResult == false {
				// Not syncing
				syncStatus.IsSyncing = false
			} else if syncMap, ok := syncResult.(map[string]interface{}); ok {
				// Currently syncing
				syncStatus.IsSyncing = true
				if current, ok := syncMap["currentBlock"].(string); ok {
					if val, err := fmt.Sscanf(current, "0x%x", &syncStatus.CurrentHeight); err == nil && val == 1 {
						// Successfully parsed current height
					}
				}
				if highest, ok := syncMap["highestBlock"].(string); ok {
					if val, err := fmt.Sscanf(highest, "0x%x", &syncStatus.HighestHeight); err == nil && val == 1 {
						// Successfully parsed highest height
					}
				}
			}
		}
	}

	// Calculate sync progress
	if syncStatus.HighestHeight > 0 && syncStatus.CurrentHeight > 0 {
		syncStatus.SyncProgress = float64(syncStatus.CurrentHeight) / float64(syncStatus.HighestHeight)
	}

	return syncStatus, nil
}

// GetHealth returns relay health status
func (gr *GenericRelay) GetHealth() (*HealthStatus, error) {
	gr.healthMu.RLock()
	defer gr.healthMu.RUnlock()

	healthCopy := *gr.health
	return &healthCopy, nil
}

// GetMetrics returns relay metrics
func (gr *GenericRelay) GetMetrics() (*RelayMetrics, error) {
	gr.metricsMu.RLock()
	defer gr.metricsMu.RUnlock()

	metricsCopy := *gr.metrics
	return &metricsCopy, nil
}

// SupportsFeature checks if the generic relay supports a specific feature
func (gr *GenericRelay) SupportsFeature(feature Feature) bool {
	// Base features supported by most generic networks
	supportedFeatures := map[Feature]bool{
		FeatureBlockStreaming:  true, // Via polling
		FeatureTransactionPool: true, // Most networks support mempool
		FeatureHistoricalData:  true, // Most support historical queries
		FeatureSmartContracts:  gr.networkType == "ethereum-like" || gr.networkType == "substrate-like",
		FeatureStateQueries:    gr.networkType == "ethereum-like" || gr.networkType == "substrate-like",
		FeatureEventLogs:       gr.networkType == "ethereum-like",
		FeatureWebSocket:       true,  // Can attempt WebSocket connections
		FeatureGraphQL:         false, // Not commonly supported
		FeatureREST:            true,  // HTTP/JSON-RPC
		FeatureCompactBlocks:   gr.networkType == "bitcoin-like",
	}

	return supportedFeatures[feature]
}

// GetSupportedFeatures returns all supported features
func (gr *GenericRelay) GetSupportedFeatures() []Feature {
	features := []Feature{
		FeatureBlockStreaming,
		FeatureTransactionPool,
		FeatureHistoricalData,
		FeatureWebSocket,
		FeatureREST,
	}

	if gr.networkType == "ethereum-like" || gr.networkType == "substrate-like" {
		features = append(features, FeatureSmartContracts, FeatureStateQueries)
	}

	if gr.networkType == "ethereum-like" {
		features = append(features, FeatureEventLogs)
	}

	if gr.networkType == "bitcoin-like" {
		features = append(features, FeatureCompactBlocks)
	}

	return features
}

// UpdateConfig updates the relay configuration
func (gr *GenericRelay) UpdateConfig(cfg RelayConfig) error {
	gr.relayConfig = cfg
	return nil
}

// GetConfig returns the current relay configuration
func (gr *GenericRelay) GetConfig() RelayConfig {
	return gr.relayConfig
}

// Helper methods

// testHTTPConnection tests the HTTP connection to the blockchain
func (gr *GenericRelay) testHTTPConnection() error {
	// Find HTTP endpoint
	var httpEndpoint string
	for _, endpoint := range gr.relayConfig.Endpoints {
		if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
			httpEndpoint = endpoint
			break
		}
	}

	if httpEndpoint == "" {
		return fmt.Errorf("no HTTP endpoint configured")
	}

	// Make a simple test request
	_, err := gr.makeHTTPRequest(gr.rpcMethods.GetNetworkInfo, []interface{}{})
	return err
}

// makeHTTPRequest makes an HTTP JSON-RPC request
func (gr *GenericRelay) makeHTTPRequest(method string, params []interface{}) (*GenericResponse, error) {
	// Find HTTP endpoint
	var httpEndpoint string
	for _, endpoint := range gr.relayConfig.Endpoints {
		if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
			httpEndpoint = endpoint
			break
		}
	}

	if httpEndpoint == "" {
		return nil, fmt.Errorf("no HTTP endpoint configured")
	}

	requestID := atomic.AddInt64(&gr.requestID, 1)

	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      requestID,
	}

	requestData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := gr.httpClient.Post(httpEndpoint, "application/json", strings.NewReader(string(requestData)))
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	var response GenericResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("RPC error: %s", response.Error.Message)
	}

	return &response, nil
}

// connectWebSocket establishes a WebSocket connection
func (gr *GenericRelay) connectWebSocket(ctx context.Context, endpoint string) {
	u, err := url.Parse(endpoint)
	if err != nil {
		gr.logger.Warn("Invalid WebSocket endpoint URL",
			zap.String("endpoint", endpoint),
			zap.Error(err))
		return
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		gr.logger.Warn("Failed to connect to WebSocket endpoint",
			zap.String("endpoint", endpoint),
			zap.Error(err))
		return
	}

	gr.wsConnMu.Lock()
	gr.wsConnections = append(gr.wsConnections, conn)
	gr.wsConnMu.Unlock()
	gr.wsConnected.Store(true)

	gr.logger.Info("Connected to WebSocket endpoint", zap.String("endpoint", endpoint))

	// Start message handler
	go gr.handleWebSocketMessages(conn)
}

// handleWebSocketMessages handles incoming WebSocket messages
func (gr *GenericRelay) handleWebSocketMessages(conn *websocket.Conn) {
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			gr.logger.Warn("WebSocket read error", zap.Error(err))
			break
		}

		// Parse and handle the message (implementation depends on specific protocol)
		gr.logger.Debug("Received WebSocket message", zap.String("message", string(message)))
	}
}

// pollBlocks polls for new blocks when streaming is not available
func (gr *GenericRelay) pollBlocks(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second) // Poll every 5 seconds
	defer ticker.Stop()

	var lastBlockHeight uint32

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			block, err := gr.GetLatestBlock()
			if err != nil {
				gr.logger.Warn("Failed to poll latest block", zap.Error(err))
				continue
			}

			if block.Height > lastBlockHeight {
				lastBlockHeight = block.Height
				select {
				case gr.blockChan <- *block:
				default:
					// Channel full, drop block
				}
			}
		}
	}
}

// parseBlockResponse parses a generic block response
func (gr *GenericRelay) parseBlockResponse(result json.RawMessage) (*blocks.BlockEvent, error) {
	var genericBlock GenericBlock
	if err := json.Unmarshal(result, &genericBlock); err != nil {
		return nil, fmt.Errorf("failed to parse block: %w", err)
	}

	event := &blocks.BlockEvent{
		Hash:       genericBlock.Hash,
		DetectedAt: time.Now(),
		Source:     gr.relayConfig.Network,
		Tier:       "enterprise",
	}

	// Parse height from different possible fields
	if genericBlock.Height > 0 {
		event.Height = uint32(genericBlock.Height)
	} else if genericBlock.Number != nil {
		switch v := genericBlock.Number.(type) {
		case float64:
			event.Height = uint32(v)
		case string:
			if strings.HasPrefix(v, "0x") {
				var height uint64
				if n, err := fmt.Sscanf(v, "0x%x", &height); err == nil && n == 1 {
					event.Height = uint32(height)
				}
			}
		}
	}

	// Parse timestamp from different possible fields
	now := time.Now()
	if genericBlock.Timestamp != nil {
		switch v := genericBlock.Timestamp.(type) {
		case float64:
			event.Timestamp = time.Unix(int64(v), 0)
		case string:
			if strings.HasPrefix(v, "0x") {
				var timestamp int64
				if n, err := fmt.Sscanf(v, "0x%x", &timestamp); err == nil && n == 1 {
					event.Timestamp = time.Unix(timestamp, 0)
				}
			}
		}
	} else if genericBlock.Time != nil {
		switch v := genericBlock.Time.(type) {
		case float64:
			event.Timestamp = time.Unix(int64(v), 0)
		case string:
			if timestamp, err := time.Parse(time.RFC3339, v); err == nil {
				event.Timestamp = timestamp
			}
		}
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = now
	}

	return event, nil
}

// updateHealth updates the health status
func (gr *GenericRelay) updateHealth(healthy bool, state string, err error) {
	gr.healthMu.Lock()
	defer gr.healthMu.Unlock()

	gr.health.IsHealthy = healthy
	gr.health.LastSeen = time.Now()
	gr.health.ConnectionState = state
	if err != nil {
		gr.health.ErrorMessage = err.Error()
		gr.health.ErrorCount++
	} else {
		gr.health.ErrorMessage = ""
	}
}
