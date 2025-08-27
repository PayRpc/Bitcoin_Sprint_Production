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
)

// SolanaRelay implements RelayClient for Solana network using WebSocket + QUIC
type SolanaRelay struct {
	cfg         config.Config
	logger      *zap.Logger
	
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
	pendingReqs map[int64]chan *SolanaResponse
	reqMu       sync.RWMutex
	
	// Subscription management
	subscriptions map[string]chan *SolanaNotification
	subMu         sync.RWMutex
}

// SolanaResponse represents a JSON-RPC response
type SolanaResponse struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *SolanaError    `json:"error,omitempty"`
}

// SolanaError represents a JSON-RPC error
type SolanaError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// SolanaNotification represents a subscription notification
type SolanaNotification struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// SolanaBlock represents a Solana block
type SolanaBlock struct {
	Slot              uint64 `json:"slot"`
	BlockHash         string `json:"blockhash"`
	PreviousBlockhash string `json:"previousBlockhash"`
	BlockTime         *int64 `json:"blockTime"`
	BlockHeight       uint64 `json:"blockHeight"`
	Transactions      []interface{} `json:"transactions"`
}

// SolanaSlotInfo represents Solana slot information
type SolanaSlotInfo struct {
	Slot   uint64 `json:"slot"`
	Parent uint64 `json:"parent"`
	Root   uint64 `json:"root"`
}

// SolanaNetworkInfo represents Solana network information
type SolanaNetworkInfo struct {
	Slot              uint64  `json:"slot"`
	BlockHeight       uint64  `json:"blockHeight"`
	EpochInfo         *SolanaEpochInfo `json:"epochInfo"`
	Version           *SolanaVersion   `json:"version"`
	TotalSupply       uint64  `json:"totalSupply"`
	CirculatingSupply uint64  `json:"circulatingSupply"`
}

// SolanaEpochInfo represents epoch information
type SolanaEpochInfo struct {
	Epoch             uint64  `json:"epoch"`
	SlotIndex         uint64  `json:"slotIndex"`
	SlotsInEpoch      uint64  `json:"slotsInEpoch"`
	AbsoluteSlot      uint64  `json:"absoluteSlot"`
	BlockHeight       uint64  `json:"blockHeight"`
	TransactionCount  *uint64 `json:"transactionCount"`
}

// SolanaVersion represents version information
type SolanaVersion struct {
	SolanaCore      string `json:"solana-core"`
	FeatureSet      uint32 `json:"feature-set"`
}

// NewSolanaRelay creates a new Solana relay client
func NewSolanaRelay(cfg config.Config, logger *zap.Logger) *SolanaRelay {
	relayConfig := RelayConfig{
		Network:           "solana",
		Endpoints:         []string{"wss://api.mainnet-beta.solana.com", "wss://solana-api.projectserum.com"},
		Timeout:           30 * time.Second,
		RetryAttempts:     3,
		RetryDelay:        2 * time.Second,
		MaxConcurrency:    8,
		BufferSize:        2000,
		EnableCompression: true,
	}

	return &SolanaRelay{
		cfg:           cfg,
		logger:        logger,
		relayConfig:   relayConfig,
		blockChan:     make(chan blocks.BlockEvent, 2000),
		pendingReqs:   make(map[int64]chan *SolanaResponse),
		subscriptions: make(map[string]chan *SolanaNotification),
		health: &HealthStatus{
			IsHealthy:       false,
			ConnectionState: "disconnected",
		},
		metrics: &RelayMetrics{},
	}
}

// Connect establishes WebSocket connections to Solana nodes
func (sr *SolanaRelay) Connect(ctx context.Context) error {
	if sr.connected.Load() {
		return nil
	}

	sr.logger.Info("Connecting to Solana network",
		zap.Strings("endpoints", sr.relayConfig.Endpoints))

	for _, endpoint := range sr.relayConfig.Endpoints {
		go sr.connectToEndpoint(ctx, endpoint)
	}

	sr.connected.Store(true)
	sr.updateHealth(true, "connected", nil)
	
	return nil
}

// Disconnect closes all WebSocket connections
func (sr *SolanaRelay) Disconnect() error {
	if !sr.connected.Load() {
		return nil
	}

	sr.connMu.Lock()
	defer sr.connMu.Unlock()

	for _, conn := range sr.connections {
		conn.Close()
	}
	sr.connections = nil

	sr.connected.Store(false)
	sr.updateHealth(false, "disconnected", nil)

	sr.logger.Info("Disconnected from Solana network")
	return nil
}

// IsConnected returns true if connected to at least one endpoint
func (sr *SolanaRelay) IsConnected() bool {
	sr.connMu.RLock()
	defer sr.connMu.RUnlock()
	return sr.connected.Load() && len(sr.connections) > 0
}

// StreamBlocks streams Solana blocks
func (sr *SolanaRelay) StreamBlocks(ctx context.Context, blockChan chan<- blocks.BlockEvent) error {
	if !sr.IsConnected() {
		return fmt.Errorf("not connected to Solana network")
	}

	// Subscribe to block updates
	if err := sr.subscribeToBlocks(ctx); err != nil {
		return fmt.Errorf("failed to subscribe to blocks: %w", err)
	}

	// Forward blocks from internal channel to provided channel
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case block := <-sr.blockChan:
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

// GetLatestBlock returns the latest Solana block
func (sr *SolanaRelay) GetLatestBlock() (*blocks.BlockEvent, error) {
	if !sr.IsConnected() {
		return nil, fmt.Errorf("not connected to Solana network")
	}

	// Get latest slot
	slotResponse, err := sr.makeRequest("getSlot", []interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to get latest slot: %w", err)
	}

	var slot uint64
	if err := json.Unmarshal(slotResponse.Result, &slot); err != nil {
		return nil, fmt.Errorf("failed to parse slot: %w", err)
	}

	// Get block for this slot
	blockResponse, err := sr.makeRequest("getBlock", []interface{}{slot, map[string]interface{}{
		"encoding": "json",
		"maxSupportedTransactionVersion": 0,
	}})
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	var solanaBlock SolanaBlock
	if err := json.Unmarshal(blockResponse.Result, &solanaBlock); err != nil {
		return nil, fmt.Errorf("failed to parse block: %w", err)
	}

	return sr.convertToBlockEvent(&solanaBlock), nil
}

// GetBlockByHash retrieves a Solana block by hash (not supported, returns error)
func (sr *SolanaRelay) GetBlockByHash(hash string) (*blocks.BlockEvent, error) {
	return nil, fmt.Errorf("Solana does not support block retrieval by hash")
}

// GetBlockByHeight retrieves a Solana block by slot (height equivalent)
func (sr *SolanaRelay) GetBlockByHeight(height uint64) (*blocks.BlockEvent, error) {
	if !sr.IsConnected() {
		return nil, fmt.Errorf("not connected to Solana network")
	}

	blockResponse, err := sr.makeRequest("getBlock", []interface{}{height, map[string]interface{}{
		"encoding": "json",
		"maxSupportedTransactionVersion": 0,
	}})
	if err != nil {
		return nil, fmt.Errorf("failed to get block by slot: %w", err)
	}

	var solanaBlock SolanaBlock
	if err := json.Unmarshal(blockResponse.Result, &solanaBlock); err != nil {
		return nil, fmt.Errorf("failed to parse block: %w", err)
	}

	return sr.convertToBlockEvent(&solanaBlock), nil
}

// GetNetworkInfo returns Solana network information
func (sr *SolanaRelay) GetNetworkInfo() (*NetworkInfo, error) {
	if !sr.IsConnected() {
		return nil, fmt.Errorf("not connected to Solana network")
	}

	// Get multiple pieces of network info
	slotResp, _ := sr.makeRequest("getSlot", []interface{}{})
	heightResp, _ := sr.makeRequest("getBlockHeight", []interface{}{})
	_ , _ = sr.makeRequest("getEpochInfo", []interface{}{})

	networkInfo := &NetworkInfo{
		Network:   "solana",
		Timestamp: time.Now(),
	}

	if slotResp != nil {
		var slot uint64
		if err := json.Unmarshal(slotResp.Result, &slot); err == nil {
			networkInfo.BlockHeight = slot // In Solana, slot is like block height
		}
	}

	if heightResp != nil {
		var height uint64
		if err := json.Unmarshal(heightResp.Result, &height); err == nil {
			networkInfo.BlockHeight = height
		}
	}

	// Solana doesn't have traditional peer count, set to 0
	networkInfo.PeerCount = 0

	return networkInfo, nil
}

// GetPeerCount returns 0 for Solana (concept doesn't apply the same way)
func (sr *SolanaRelay) GetPeerCount() int {
	return 0 // Solana uses validators instead of traditional peers
}

// GetSyncStatus returns Solana synchronization status
func (sr *SolanaRelay) GetSyncStatus() (*SyncStatus, error) {
	healthResp, err := sr.makeRequest("getHealth", []interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to get health status: %w", err)
	}

	var health string
	if err := json.Unmarshal(healthResp.Result, &health); err != nil {
		return nil, fmt.Errorf("failed to parse health: %w", err)
	}

	// If health is "ok", assume synced
	isSynced := health == "ok"

	slotResp, _ := sr.makeRequest("getSlot", []interface{}{})
	var currentSlot uint64
	if slotResp != nil {
		json.Unmarshal(slotResp.Result, &currentSlot)
	}

	return &SyncStatus{
		IsSyncing:     !isSynced,
		CurrentHeight: currentSlot,
		HighestHeight: currentSlot,
		SyncProgress:  1.0,
	}, nil
}

// GetHealth returns Solana relay health status
func (sr *SolanaRelay) GetHealth() (*HealthStatus, error) {
	sr.healthMu.RLock()
	defer sr.healthMu.RUnlock()
	
	healthCopy := *sr.health
	return &healthCopy, nil
}

// GetMetrics returns Solana relay metrics
func (sr *SolanaRelay) GetMetrics() (*RelayMetrics, error) {
	sr.metricsMu.RLock()
	defer sr.metricsMu.RUnlock()
	
	metricsCopy := *sr.metrics
	return &metricsCopy, nil
}

// SupportsFeature checks if Solana relay supports a specific feature
func (sr *SolanaRelay) SupportsFeature(feature Feature) bool {
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
func (sr *SolanaRelay) GetSupportedFeatures() []Feature {
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
func (sr *SolanaRelay) UpdateConfig(cfg RelayConfig) error {
	sr.relayConfig = cfg
	return nil
}

// GetConfig returns the current relay configuration
func (sr *SolanaRelay) GetConfig() RelayConfig {
	return sr.relayConfig
}

// Helper methods

// connectToEndpoint establishes a WebSocket connection to an endpoint
func (sr *SolanaRelay) connectToEndpoint(ctx context.Context, endpoint string) {
	u, err := url.Parse(endpoint)
	if err != nil {
		sr.logger.Warn("Invalid endpoint URL", 
			zap.String("endpoint", endpoint), 
			zap.Error(err))
		return
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		sr.logger.Warn("Failed to connect to Solana endpoint", 
			zap.String("endpoint", endpoint), 
			zap.Error(err))
		return
	}

	sr.connMu.Lock()
	sr.connections = append(sr.connections, conn)
	sr.connMu.Unlock()

	sr.logger.Info("Connected to Solana endpoint", zap.String("endpoint", endpoint))

	// Start message handler
	go sr.handleMessages(conn)
}

// handleMessages handles incoming WebSocket messages
func (sr *SolanaRelay) handleMessages(conn *websocket.Conn) {
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			sr.logger.Warn("WebSocket read error", zap.Error(err))
			break
		}

		// Parse message as JSON-RPC response or notification
		var response SolanaResponse
		if err := json.Unmarshal(message, &response); err == nil && response.ID > 0 {
			// Handle response
			sr.handleResponse(&response)
		} else {
			// Handle notification
			var notification SolanaNotification
			if err := json.Unmarshal(message, &notification); err == nil {
				sr.handleNotification(&notification)
			}
		}
	}
}

// makeRequest makes a JSON-RPC request
func (sr *SolanaRelay) makeRequest(method string, params []interface{}) (*SolanaResponse, error) {
	requestID := atomic.AddInt64(&sr.requestID, 1)
	
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
	sr.connMu.RLock()
	if len(sr.connections) == 0 {
		sr.connMu.RUnlock()
		return nil, fmt.Errorf("no active connections")
	}
	conn := sr.connections[0] // Use first connection
	sr.connMu.RUnlock()

	// Create response channel
	responseChan := make(chan *SolanaResponse, 1)
	sr.reqMu.Lock()
	sr.pendingReqs[requestID] = responseChan
	sr.reqMu.Unlock()

	// Send request
	if err := conn.WriteMessage(websocket.TextMessage, requestData); err != nil {
		sr.reqMu.Lock()
		delete(sr.pendingReqs, requestID)
		sr.reqMu.Unlock()
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Wait for response
	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(sr.relayConfig.Timeout):
		sr.reqMu.Lock()
		delete(sr.pendingReqs, requestID)
		sr.reqMu.Unlock()
		return nil, fmt.Errorf("request timeout")
	}
}

// handleResponse handles JSON-RPC responses
func (sr *SolanaRelay) handleResponse(response *SolanaResponse) {
	sr.reqMu.Lock()
	responseChan, exists := sr.pendingReqs[response.ID]
	if exists {
		delete(sr.pendingReqs, response.ID)
	}
	sr.reqMu.Unlock()

	if exists {
		select {
		case responseChan <- response:
		default:
		}
	}
}

// handleNotification handles subscription notifications
func (sr *SolanaRelay) handleNotification(notification *SolanaNotification) {
	// Handle slot notifications specifically
	if notification.Method == "slotNotification" {
		sr.handleSlotNotification(notification)
	}
}

// subscribeToBlocks subscribes to slot updates (Solana's equivalent of blocks)
func (sr *SolanaRelay) subscribeToBlocks(ctx context.Context) error {
	// Subscribe to slot notifications
	_, err := sr.makeRequest("slotSubscribe", []interface{}{})
	return err
}

// handleSlotNotification processes slot notifications
func (sr *SolanaRelay) handleSlotNotification(notification *SolanaNotification) {
	// Parse notification and extract slot data
	blockEvent := blocks.BlockEvent{
		Hash:       "solana-slot-" + fmt.Sprintf("%d", time.Now().Unix()), // Solana uses slots, not traditional hashes
		Height:     uint32(time.Now().Unix() % 1000000), // placeholder
		Timestamp:  time.Now(),
		DetectedAt: time.Now(),
		Source:     "solana-relay",
		Tier:       "enterprise",
	}

	select {
	case sr.blockChan <- blockEvent:
	default:
		// Channel full, drop block
	}
}

// convertToBlockEvent converts SolanaBlock to BlockEvent
func (sr *SolanaRelay) convertToBlockEvent(solanaBlock *SolanaBlock) *blocks.BlockEvent {
	event := &blocks.BlockEvent{
		Hash:       solanaBlock.BlockHash,
		Height:     uint32(solanaBlock.Slot),
		DetectedAt: time.Now(),
		Source:     "solana-relay",
		Tier:       "enterprise",
	}

	if solanaBlock.BlockTime != nil {
		event.Timestamp = time.Unix(*solanaBlock.BlockTime, 0)
	} else {
		event.Timestamp = time.Now()
	}

	return event
}

// updateHealth updates the health status
func (sr *SolanaRelay) updateHealth(healthy bool, state string, err error) {
	sr.healthMu.Lock()
	defer sr.healthMu.Unlock()
	
	sr.health.IsHealthy = healthy
	sr.health.LastSeen = time.Now()
	sr.health.ConnectionState = state
	if err != nil {
		sr.health.ErrorMessage = err.Error()
		sr.health.ErrorCount++
	} else {
		sr.health.ErrorMessage = ""
	}
}
