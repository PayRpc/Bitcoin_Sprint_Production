package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	diag "github.com/PayRpc/Bitcoin-Sprint/internal/p2p/diag"
)

// EthereumRelay implements RelayClient for Ethereum network using JSON-RPC WebSocket
type EthereumRelay struct {
	cfg    config.Config
	logger *zap.Logger

	// WebSocket connections
	connections []*websocket.Conn
	connMu      sync.RWMutex
	connected   atomic.Bool

	// Block streaming
	blockChan chan blocks.BlockEvent

	// Configuration
	relayConfig RelayConfig

	// Health and metrics
	health    *HealthStatus
	healthMu  sync.RWMutex
	metrics   *RelayMetrics
	metricsMu sync.RWMutex

	// Request tracking
	requestID   int64
	pendingReqs map[int64]chan *EthereumResponse
	reqMu       sync.RWMutex

	// Subscription management
	subscriptions map[string]chan *EthereumNotification
	subMu         sync.RWMutex
	// Handshake timers for diagnostics
	handshakeTimers   map[string]*time.Timer
	handshakeTimersMu sync.Mutex
}

// EthereumResponse represents a JSON-RPC response
type EthereumResponse struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *EthereumError  `json:"error,omitempty"`
}

// EthereumError represents a JSON-RPC error
type EthereumError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// EthereumNotification represents a subscription notification
type EthereumNotification struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// EthereumBlock represents an Ethereum block
type EthereumBlock struct {
	Number       string   `json:"number"`
	Hash         string   `json:"hash"`
	ParentHash   string   `json:"parentHash"`
	Timestamp    string   `json:"timestamp"`
	Size         string   `json:"size"`
	GasUsed      string   `json:"gasUsed"`
	GasLimit     string   `json:"gasLimit"`
	Transactions []string `json:"transactions"`
}

// EthereumNetworkInfo represents Ethereum network information
type EthereumNetworkInfo struct {
	ChainID     string `json:"chainId"`
	NetworkID   string `json:"networkId"`
	BlockNumber string `json:"blockNumber"`
	GasPrice    string `json:"gasPrice"`
	PeerCount   string `json:"peerCount"`
	Syncing     bool   `json:"syncing"`
}

// NewEthereumRelay creates a new Ethereum relay client
func NewEthereumRelay(cfg config.Config, logger *zap.Logger) *EthereumRelay {
	relayConfig := RelayConfig{
		Network:           "ethereum",
		Endpoints:         []string{"18.138.108.67:30303", "3.209.45.79:30303", "34.255.23.113:30303"},
		Timeout:           30 * time.Second,
		RetryAttempts:     3,
		RetryDelay:        5 * time.Second,
		MaxConcurrency:    4,
		BufferSize:        1000,
		EnableCompression: true,
	}

	return &EthereumRelay{
		cfg:           cfg,
		logger:        logger,
		relayConfig:   relayConfig,
		blockChan:     make(chan blocks.BlockEvent, 1000),
		pendingReqs:   make(map[int64]chan *EthereumResponse),
		subscriptions: make(map[string]chan *EthereumNotification),
		health: &HealthStatus{
			IsHealthy:       false,
			ConnectionState: "disconnected",
		},
		metrics: &RelayMetrics{},
	handshakeTimers: make(map[string]*time.Timer),
	}
}

// Connect establishes WebSocket connections to Ethereum nodes
func (er *EthereumRelay) Connect(ctx context.Context) error {
	if er.connected.Load() {
		return nil
	}

	er.logger.Info("Connecting to Ethereum network",
		zap.Strings("endpoints", er.relayConfig.Endpoints))

	for _, endpoint := range er.relayConfig.Endpoints {
		go er.connectToEndpoint(ctx, endpoint)
	}

	er.connected.Store(true)
	er.updateHealth(true, "connected", nil)

	return nil
}

// Disconnect closes all WebSocket connections
func (er *EthereumRelay) Disconnect() error {
	if !er.connected.Load() {
		return nil
	}

	er.connMu.Lock()
	defer er.connMu.Unlock()

	for _, conn := range er.connections {
		conn.Close()
	}
	er.connections = nil

	er.connected.Store(false)
	er.updateHealth(false, "disconnected", nil)

	er.logger.Info("Disconnected from Ethereum network")
	return nil
}

// IsConnected returns true if connected to at least one endpoint
func (er *EthereumRelay) IsConnected() bool {
	er.connMu.RLock()
	defer er.connMu.RUnlock()
	return er.connected.Load() && len(er.connections) > 0
}

// StreamBlocks streams Ethereum blocks
func (er *EthereumRelay) StreamBlocks(ctx context.Context, blockChan chan<- blocks.BlockEvent) error {
	if !er.IsConnected() {
		return fmt.Errorf("not connected to Ethereum network")
	}

	// Subscribe to new block headers
	if err := er.subscribeToBlocks(ctx); err != nil {
		return fmt.Errorf("failed to subscribe to blocks: %w", err)
	}

	// Forward blocks from internal channel to provided channel
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case block := <-er.blockChan:
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

// GetLatestBlock returns the latest Ethereum block
func (er *EthereumRelay) GetLatestBlock() (*blocks.BlockEvent, error) {
	if !er.IsConnected() {
		return nil, fmt.Errorf("not connected to Ethereum network")
	}

	// Make JSON-RPC call to get latest block
	response, err := er.makeRequest("eth_getBlockByNumber", []interface{}{"latest", false})
	if err != nil {
		return nil, fmt.Errorf("failed to get latest block: %w", err)
	}

	var ethBlock EthereumBlock
	if err := json.Unmarshal(response.Result, &ethBlock); err != nil {
		return nil, fmt.Errorf("failed to parse block: %w", err)
	}

	return er.convertToBlockEvent(&ethBlock), nil
}

// GetBlockByHash retrieves an Ethereum block by hash
func (er *EthereumRelay) GetBlockByHash(hash string) (*blocks.BlockEvent, error) {
	if !er.IsConnected() {
		return nil, fmt.Errorf("not connected to Ethereum network")
	}

	response, err := er.makeRequest("eth_getBlockByHash", []interface{}{hash, false})
	if err != nil {
		return nil, fmt.Errorf("failed to get block by hash: %w", err)
	}

	var ethBlock EthereumBlock
	if err := json.Unmarshal(response.Result, &ethBlock); err != nil {
		return nil, fmt.Errorf("failed to parse block: %w", err)
	}

	return er.convertToBlockEvent(&ethBlock), nil
}

// GetBlockByHeight retrieves an Ethereum block by height
func (er *EthereumRelay) GetBlockByHeight(height uint64) (*blocks.BlockEvent, error) {
	if !er.IsConnected() {
		return nil, fmt.Errorf("not connected to Ethereum network")
	}

	blockNumber := fmt.Sprintf("0x%x", height)
	response, err := er.makeRequest("eth_getBlockByNumber", []interface{}{blockNumber, false})
	if err != nil {
		return nil, fmt.Errorf("failed to get block by height: %w", err)
	}

	var ethBlock EthereumBlock
	if err := json.Unmarshal(response.Result, &ethBlock); err != nil {
		return nil, fmt.Errorf("failed to parse block: %w", err)
	}

	return er.convertToBlockEvent(&ethBlock), nil
}

// GetNetworkInfo returns Ethereum network information
func (er *EthereumRelay) GetNetworkInfo() (*NetworkInfo, error) {
	if !er.IsConnected() {
		return nil, fmt.Errorf("not connected to Ethereum network")
	}

	// Get network info via multiple JSON-RPC calls
	chainIDResp, _ := er.makeRequest("eth_chainId", []interface{}{})
	blockNumberResp, _ := er.makeRequest("eth_blockNumber", []interface{}{})
	peerCountResp, _ := er.makeRequest("net_peerCount", []interface{}{})

	networkInfo := &NetworkInfo{
		Network:   "ethereum",
		Timestamp: time.Now(),
	}

	if chainIDResp != nil {
		var chainID string
		json.Unmarshal(chainIDResp.Result, &chainID)
		networkInfo.ChainID = chainID
	}

	if blockNumberResp != nil {
		var blockNumber string
		json.Unmarshal(blockNumberResp.Result, &blockNumber)
		// Convert hex to decimal for height
		if height, err := parseHexNumber(blockNumber); err == nil {
			networkInfo.BlockHeight = height
		}
	}

	if peerCountResp != nil {
		var peerCount string
		json.Unmarshal(peerCountResp.Result, &peerCount)
		if count, err := parseHexNumber(peerCount); err == nil {
			networkInfo.PeerCount = int(count)
		}
	}

	return networkInfo, nil
}

// GetPeerCount returns the number of connected peers
func (er *EthereumRelay) GetPeerCount() int {
	response, err := er.makeRequest("net_peerCount", []interface{}{})
	if err != nil {
		return 0
	}

	var peerCount string
	if err := json.Unmarshal(response.Result, &peerCount); err != nil {
		return 0
	}

	if count, err := parseHexNumber(peerCount); err == nil {
		return int(count)
	}

	return 0
}

// GetSyncStatus returns Ethereum synchronization status
func (er *EthereumRelay) GetSyncStatus() (*SyncStatus, error) {
	response, err := er.makeRequest("eth_syncing", []interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to get sync status: %w", err)
	}

	var syncing interface{}
	if err := json.Unmarshal(response.Result, &syncing); err != nil {
		return nil, fmt.Errorf("failed to parse sync status: %w", err)
	}

	// If syncing is false, node is synced
	if isSyncing, ok := syncing.(bool); ok && !isSyncing {
		return &SyncStatus{
			IsSyncing:    false,
			SyncProgress: 1.0,
		}, nil
	}

	// Otherwise, parse sync progress
	syncData, ok := syncing.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected sync status format")
	}

	status := &SyncStatus{IsSyncing: true}

	if currentBlock, ok := syncData["currentBlock"].(string); ok {
		if current, err := parseHexNumber(currentBlock); err == nil {
			status.CurrentHeight = current
		}
	}

	if highestBlock, ok := syncData["highestBlock"].(string); ok {
		if highest, err := parseHexNumber(highestBlock); err == nil {
			status.HighestHeight = highest
		}
	}

	if status.HighestHeight > 0 {
		status.SyncProgress = float64(status.CurrentHeight) / float64(status.HighestHeight)
	}

	return status, nil
}

// GetHealth returns Ethereum relay health status
func (er *EthereumRelay) GetHealth() (*HealthStatus, error) {
	er.healthMu.RLock()
	defer er.healthMu.RUnlock()

	healthCopy := *er.health
	return &healthCopy, nil
}

// GetMetrics returns Ethereum relay metrics
func (er *EthereumRelay) GetMetrics() (*RelayMetrics, error) {
	er.metricsMu.RLock()
	defer er.metricsMu.RUnlock()

	metricsCopy := *er.metrics
	return &metricsCopy, nil
}

// SupportsFeature checks if Ethereum relay supports a specific feature
func (er *EthereumRelay) SupportsFeature(feature Feature) bool {
	supportedFeatures := map[Feature]bool{
		FeatureBlockStreaming:  true,
		FeatureTransactionPool: true,
		FeatureHistoricalData:  true,
		FeatureSmartContracts:  true,
		FeatureStateQueries:    true,
		FeatureEventLogs:       true,
		FeatureWebSocket:       true,
		FeatureGraphQL:         false,
		FeatureREST:            true,
		FeatureCompactBlocks:   false,
	}

	return supportedFeatures[feature]
}

// GetSupportedFeatures returns all supported features
func (er *EthereumRelay) GetSupportedFeatures() []Feature {
	return []Feature{
		FeatureBlockStreaming,
		FeatureTransactionPool,
		FeatureHistoricalData,
		FeatureSmartContracts,
		FeatureStateQueries,
		FeatureEventLogs,
		FeatureWebSocket,
		FeatureREST,
	}
}

// UpdateConfig updates the relay configuration
func (er *EthereumRelay) UpdateConfig(cfg RelayConfig) error {
	er.relayConfig = cfg
	return nil
}

// GetConfig returns the current relay configuration
func (er *EthereumRelay) GetConfig() RelayConfig {
	return er.relayConfig
}

// Helper methods

// connectToEndpoint establishes a WebSocket connection to an endpoint
func (er *EthereumRelay) connectToEndpoint(ctx context.Context, endpoint string) {
	u, err := url.Parse(endpoint)
	if err != nil {
		er.logger.Warn("Invalid endpoint URL",
			zap.String("endpoint", endpoint),
			zap.Error(err))
		return
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		er.logger.Warn("Failed to connect to Ethereum endpoint",
			zap.String("endpoint", endpoint),
			zap.Error(err))
		// record failed TCP attempt
		diag.RecordAttempt("ethereum", diag.AttemptRecord{
			Timestamp:  time.Now(),
			Address:    endpoint,
			TcpSuccess: false,
			TcpError:   err.Error(),
		})
		return
	}

	er.connMu.Lock()
	er.connections = append(er.connections, conn)
	er.connMu.Unlock()

	er.logger.Info("Connected to Ethereum endpoint", zap.String("endpoint", endpoint))

	// record successful TCP/connect attempt (handshake not yet confirmed)
	diag.RecordAttempt("ethereum", diag.AttemptRecord{
		Timestamp:        time.Now(),
		Address:          endpoint,
		TcpSuccess:       true,
		HandshakeSuccess: false,
		HandshakeError:   "",
	})

	// Start message handler (pass endpoint + done channel so handler can mark handshake when first valid response arrives)
	done := make(chan struct{})
	go er.handleMessages(conn, endpoint, done)

	// Start handshake timer (10s). If no valid JSON-RPC response arrives we record handshake failure.
	handshakeTimer := time.AfterFunc(10*time.Second, func() {
		diag.RecordAttempt("ethereum", diag.AttemptRecord{
			Timestamp:        time.Now(),
			Address:          endpoint,
			TcpSuccess:       true,
			HandshakeSuccess: false,
			HandshakeError:   "timeout: no JSON-RPC response",
		})
		// signal done if not already closed
		select {
		case <-done:
		default:
			close(done)
		}
	})

	// Save timer reference so it can be cancelled on handshake success or early failure
	er.handshakeTimersMu.Lock()
	er.handshakeTimers[endpoint] = handshakeTimer
	er.handshakeTimersMu.Unlock()
}

// handleMessages handles incoming WebSocket messages
func (er *EthereumRelay) handleMessages(conn *websocket.Conn, endpoint string, done chan struct{}) {
	defer conn.Close()

	handshakeDone := false

	for {
		_, message, err := conn.ReadMessage()
			if err != nil {
				er.logger.Warn("WebSocket read error", zap.Error(err))
				// if handshake never completed, record failure with the read error and cancel timer
				if !handshakeDone {
					diag.RecordAttempt("ethereum", diag.AttemptRecord{
						Timestamp:        time.Now(),
						Address:          endpoint,
						TcpSuccess:       true,
						HandshakeSuccess: false,
						HandshakeError:   err.Error(),
					})
					// cancel any handshake timer
					er.handshakeTimersMu.Lock()
					if t, ok := er.handshakeTimers[endpoint]; ok {
						t.Stop()
						delete(er.handshakeTimers, endpoint)
					}
					er.handshakeTimersMu.Unlock()
					// signal done to the watcher
					select {
					case <-done:
					default:
						close(done)
					}
				}
				break
			}

		// Parse message as JSON-RPC response or notification
		var response EthereumResponse
		if err := json.Unmarshal(message, &response); err == nil && response.ID > 0 {
			// On first valid JSON-RPC response, mark handshake success for diagnostics
				if !handshakeDone {
					handshakeDone = true
					diag.RecordAttempt("ethereum", diag.AttemptRecord{
						Timestamp:        time.Now(),
						Address:          endpoint,
						TcpSuccess:       true,
						HandshakeSuccess: true,
						HandshakeError:   "",
					})
					// Cancel handshake timer if present
					er.handshakeTimersMu.Lock()
					if t, ok := er.handshakeTimers[endpoint]; ok {
						t.Stop()
						delete(er.handshakeTimers, endpoint)
					}
					er.handshakeTimersMu.Unlock()
					// signal done to the watcher
					select {
					case <-done:
					default:
						close(done)
					}
				}
			// Handle response
			er.handleResponse(&response)
		} else {
			// Handle notification
			var notification EthereumNotification
			if err := json.Unmarshal(message, &notification); err == nil {
				er.handleNotification(&notification)
			}
		}
	}
}

// makeRequest makes a JSON-RPC request
func (er *EthereumRelay) makeRequest(method string, params []interface{}) (*EthereumResponse, error) {
	requestID := atomic.AddInt64(&er.requestID, 1)

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

	// Get a connection
	er.connMu.RLock()
	if len(er.connections) == 0 {
		er.connMu.RUnlock()
		return nil, fmt.Errorf("no active connections")
	}
	conn := er.connections[0] // Use first connection
	er.connMu.RUnlock()

	// Create response channel
	responseChan := make(chan *EthereumResponse, 1)
	er.reqMu.Lock()
	er.pendingReqs[requestID] = responseChan
	er.reqMu.Unlock()

	// Send request
	if err := conn.WriteMessage(websocket.TextMessage, requestData); err != nil {
		er.reqMu.Lock()
		delete(er.pendingReqs, requestID)
		er.reqMu.Unlock()
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Wait for response
	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(er.relayConfig.Timeout):
		er.reqMu.Lock()
		delete(er.pendingReqs, requestID)
		er.reqMu.Unlock()
		return nil, fmt.Errorf("request timeout")
	}
}

// handleResponse handles JSON-RPC responses
func (er *EthereumRelay) handleResponse(response *EthereumResponse) {
	er.reqMu.Lock()
	responseChan, exists := er.pendingReqs[response.ID]
	if exists {
		delete(er.pendingReqs, response.ID)
	}
	er.reqMu.Unlock()

	if exists {
		select {
		case responseChan <- response:
		default:
		}
	}
}

// handleNotification handles subscription notifications
func (er *EthereumRelay) handleNotification(notification *EthereumNotification) {
	// Handle block notifications specifically
	if notification.Method == "eth_subscription" {
		// Parse subscription params and extract block data
		er.handleBlockNotification(notification)
	}
}

// subscribeToBlocks subscribes to new block headers
func (er *EthereumRelay) subscribeToBlocks(ctx context.Context) error {
	// Subscribe to new block headers
	_, err := er.makeRequest("eth_subscribe", []interface{}{"newHeads"})
	return err
}

// handleBlockNotification processes block notifications
func (er *EthereumRelay) handleBlockNotification(notification *EthereumNotification) {
	// Parse notification and extract block data
	// Convert to BlockEvent and send to channel
	blockEvent := blocks.BlockEvent{
		Hash:       "0x" + "0000000000000000000000000000000000000000000000000000000000000000", // placeholder
		Height:     850000,                                                                    // placeholder
		Timestamp:  time.Now(),
		DetectedAt: time.Now(),
		Source:     "ethereum-relay",
		Tier:       "enterprise",
	}

	select {
	case er.blockChan <- blockEvent:
	default:
		// Channel full, drop block
	}
}

// convertToBlockEvent converts EthereumBlock to BlockEvent
func (er *EthereumRelay) convertToBlockEvent(ethBlock *EthereumBlock) *blocks.BlockEvent {
	event := &blocks.BlockEvent{
		Hash:       ethBlock.Hash,
		DetectedAt: time.Now(),
		Source:     "ethereum-relay",
		Tier:       "enterprise",
	}

	if height, err := parseHexNumber(ethBlock.Number); err == nil {
		event.Height = uint32(height)
	}

	if timestamp, err := parseHexNumber(ethBlock.Timestamp); err == nil {
		event.Timestamp = time.Unix(int64(timestamp), 0)
	}

	return event
}

// updateHealth updates the health status
func (er *EthereumRelay) updateHealth(healthy bool, state string, err error) {
	er.healthMu.Lock()
	defer er.healthMu.Unlock()

	er.health.IsHealthy = healthy
	er.health.LastSeen = time.Now()
	er.health.ConnectionState = state
	if err != nil {
		er.health.ErrorMessage = err.Error()
		er.health.ErrorCount++
	} else {
		er.health.ErrorMessage = ""
	}
}

// parseHexNumber parses a hex string to uint64
func parseHexNumber(hex string) (uint64, error) {
	if len(hex) < 3 || hex[:2] != "0x" {
		return 0, fmt.Errorf("invalid hex format")
	}

	var result uint64
	if _, err := fmt.Sscanf(hex, "0x%x", &result); err != nil {
		return 0, err
	}

	return result, nil
}
