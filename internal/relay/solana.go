package relay

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/netx"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// SolanaRelay implements RelayClient for Solana network using WebSocket + QUIC
type SolanaRelay struct {
	cfg    config.Config
	logger *zap.Logger

	// WebSocket connections
	connections []*wsConn
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
	
	// Block deduplication
	deduper    *BlockDeduper

	// Request tracking
	requestID   int64
	pendingReqs map[int64]chan *SolanaResponse
	reqMu       sync.RWMutex

	// Subscription management
	subscriptions map[string]chan *SolanaNotification
	subMu         sync.RWMutex

	// backoff per endpoint
	backoffMu sync.Mutex
	backoff   map[string]int
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
	Slot              uint64        `json:"slot"`
	BlockHash         string        `json:"blockhash"`
	PreviousBlockhash string        `json:"previousBlockhash"`
	BlockTime         *int64        `json:"blockTime"`
	BlockHeight       uint64        `json:"blockHeight"`
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
	Slot              uint64           `json:"slot"`
	BlockHeight       uint64           `json:"blockHeight"`
	EpochInfo         *SolanaEpochInfo `json:"epochInfo"`
	Version           *SolanaVersion   `json:"version"`
	TotalSupply       uint64           `json:"totalSupply"`
	CirculatingSupply uint64           `json:"circulatingSupply"`
}

// SolanaEpochInfo represents epoch information
type SolanaEpochInfo struct {
	Epoch            uint64  `json:"epoch"`
	SlotIndex        uint64  `json:"slotIndex"`
	SlotsInEpoch     uint64  `json:"slotsInEpoch"`
	AbsoluteSlot     uint64  `json:"absoluteSlot"`
	BlockHeight      uint64  `json:"blockHeight"`
	TransactionCount *uint64 `json:"transactionCount"`
}

// SolanaVersion represents version information
type SolanaVersion struct {
	SolanaCore string `json:"solana-core"`
	FeatureSet uint32 `json:"feature-set"`
}

// NewSolanaRelay creates a new Solana relay client
func NewSolanaRelay(cfg config.Config, logger *zap.Logger) *SolanaRelay {
	relayConfig := RelayConfig{
		Network:           "solana",
		Endpoints:         []string{"wss://api.mainnet-beta.solana.com", "wss://solana.publicnode.com", "wss://rpc.ankr.com/solana"},
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
		backoff:       make(map[string]int),
		health: &HealthStatus{
			IsHealthy:       false,
			ConnectionState: "disconnected",
		},
		metrics: &RelayMetrics{},
		deduper: NewBlockDeduper(4096, 3*time.Minute), // Solana-specific deduper
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

	return nil
}

// Disconnect closes all WebSocket connections
func (sr *SolanaRelay) Disconnect() error {
	if !sr.connected.Load() {
		return nil
	}

	sr.connMu.Lock()
	defer sr.connMu.Unlock()

	for _, wc := range sr.connections {
		_ = wc.Conn.Close()
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
	return len(sr.connections) > 0
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
		"encoding":                       "json",
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
		"encoding":                       "json",
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
	_, _ = sr.makeRequest("getEpochInfo", []interface{}{})

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

	// Use a websocket dialer that respects a custom resolver and TLS
	dialer := websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  20 * time.Second,
		TLSClientConfig:   &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: false},
		NetDialContext:    netx.DialerWithResolver(),
		EnableCompression: true,
	}

	// Base headers for all endpoints
	header := http.Header{}
	header.Set("Origin", "https://bitcoinsprint.com")
	header.Set("User-Agent", "BitcoinSprint/2.2 (+https://bitcoinsprint.com)")
	header.Set("Pragma", "no-cache")
	header.Set("Cache-Control", "no-cache")
	
	// Endpoint-specific configuration
	if strings.Contains(endpoint, "cloudflare") {
		// Cloudflare requires specific headers
		header.Set("Origin", "https://www.cloudflare-eth.com")
		header.Set("CF-Access-Client-Id", sr.cfg.Get("CF_ACCESS_CLIENT_ID", ""))
		header.Set("CF-Access-Client-Secret", sr.cfg.Get("CF_ACCESS_CLIENT_SECRET", ""))
	} else if strings.Contains(endpoint, "ankr") {
		// Ankr API requires JWT or API key
		apiKey := sr.cfg.Get("ANKR_API_KEY", "")
		if apiKey != "" {
			header.Set("Authorization", "Bearer "+apiKey)
		}
		header.Set("Origin", "https://www.ankr.com")
	}

	var attempt int
	for {
		attempt++
		dialCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		conn, _, err := dialer.DialContext(dialCtx, u.String(), header)
		cancel()
		if err == nil {
			wc := &wsConn{
				Conn:     conn,
				logger:   sr.logger,
				endpoint: endpoint,
			}
			sr.installWSHandlers(wc)
			sr.addConnection(wc)
			sr.logger.Info("Connected to Solana endpoint", zap.String("endpoint", endpoint))
			// Start message handler
			go sr.handleMessages(wc)
			return
		}

		sr.logger.Warn("Failed to connect to Solana endpoint",
			zap.String("endpoint", endpoint),
			zap.Error(err),
			zap.Int("attempt", attempt))

		// Backoff with jitter
		backoff := time.Duration(math.Min(float64(30*time.Second), float64(2*time.Second)*math.Pow(2, float64(attempt))))
		jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
		wait := backoff + jitter

		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
			// retry
		}
	}
}

func (sr *SolanaRelay) installWSHandlers(wc *wsConn) {
	// Set a more aggressive initial read deadline
	_ = wc.Conn.SetReadDeadline(time.Now().Add(45 * time.Second))
	
	// Enhanced pong handler with logging
	wc.Conn.SetPongHandler(func(data string) error {
		_ = wc.Conn.SetReadDeadline(time.Now().Add(45 * time.Second))
		sr.logger.Debug("Received pong", 
			zap.String("endpoint", wc.endpoint),
			zap.String("data", data))
		return nil
	})
	
	// Enhanced ping loop with more frequent pings and heartbeat subscription refresh
	go func() {
		pingTicker := time.NewTicker(15 * time.Second)  // More frequent pings
		heartbeatTicker := time.NewTicker(50 * time.Second) // Send heartbeat before timeout
		defer pingTicker.Stop()
		defer heartbeatTicker.Stop()
		
		for {
			select {
			case <-pingTicker.C:
				// Verify connection is still in active set
				sr.connMu.RLock()
				alive := false
				for _, c := range sr.connections {
					if c == wc {
						alive = true
						break
					}
				}
				sr.connMu.RUnlock()
				
				if !alive {
					return
				}
				
				// Send ping with timestamp
				wc.writeMu.Lock()
				_ = wc.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
				pingData := fmt.Sprintf("ping-%d", time.Now().Unix())
				err := wc.Conn.WriteControl(websocket.PingMessage, []byte(pingData), time.Now().Add(5*time.Second))
				wc.writeMu.Unlock()
				
				if err != nil {
					sr.logger.Warn("Ping failed", 
						zap.String("endpoint", wc.endpoint), 
						zap.Error(err))
					return
				}
				
			case <-heartbeatTicker.C:
				// Send a heartbeat message to keep connection alive
				// This is especially important for publicnode.com which has a 60s timeout
				sr.sendHeartbeat(wc)
			}
		}
	}()
}

// sendHeartbeat sends a lightweight RPC call to keep the connection active
func (sr *SolanaRelay) sendHeartbeat(wc *wsConn) {
	// For Solana connections: refresh subscriptions or send a lightweight call
	requestData := []byte(`{"jsonrpc":"2.0","method":"getHealth","params":[],"id":0}`)
	
	wc.writeMu.Lock()
	_ = wc.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	err := wc.Conn.WriteMessage(websocket.TextMessage, requestData)
	wc.writeMu.Unlock()
	
	if err != nil {
		sr.logger.Warn("Failed to send heartbeat", 
			zap.String("endpoint", wc.endpoint), 
			zap.Error(err))
	} else {
		sr.logger.Debug("Sent heartbeat to keep connection alive", 
			zap.String("endpoint", wc.endpoint))
	}
}

// handleMessages handles incoming WebSocket messages
func (sr *SolanaRelay) handleMessages(wc *wsConn) {
	defer func() {
		_ = wc.Conn.Close()
		sr.removeConnection(wc)
		sr.updateHealth(sr.IsConnected(), "connection_lost", nil)
		sr.logger.Warn("Solana WebSocket handler exited", zap.String("endpoint", wc.endpoint))
		sr.scheduleReconnect(wc.endpoint)
	}()

	for {
		_, message, err := wc.Conn.ReadMessage()
		if err != nil {
			sr.logger.Warn("WebSocket read error", zap.Error(err))
			// Don't break immediately, try to reconnect
			if sr.shouldReconnect(err) {
				sr.logger.Info("Attempting to reconnect Solana WebSocket", zap.String("endpoint", wc.endpoint))
				return
			}
			return
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

func (sr *SolanaRelay) addConnection(wc *wsConn) {
	sr.connMu.Lock()
	defer sr.connMu.Unlock()
	sr.connections = append(sr.connections, wc)
	if len(sr.connections) == 1 {
		sr.connected.Store(true)
		sr.updateHealth(true, "connected", nil)
	}
}

func (sr *SolanaRelay) removeConnection(wc *wsConn) {
	sr.connMu.Lock()
	defer sr.connMu.Unlock()
	out := sr.connections[:0]
	for _, c := range sr.connections {
		if c != wc {
			out = append(out, c)
		}
	}
	sr.connections = out
	if len(sr.connections) == 0 {
		sr.connected.Store(false)
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
	n := len(sr.connections)
	if n == 0 {
		sr.connMu.RUnlock()
		return nil, fmt.Errorf("no active connections")
	}
	wc := sr.connections[rand.Intn(n)]
	sr.connMu.RUnlock()

	// Create response channel
	responseChan := make(chan *SolanaResponse, 1)
	sr.reqMu.Lock()
	sr.pendingReqs[requestID] = responseChan
	sr.reqMu.Unlock()

	// Send request
	wc.writeMu.Lock()
	_ = wc.Conn.SetWriteDeadline(time.Now().Add(8 * time.Second))
	err = wc.Conn.WriteMessage(websocket.TextMessage, requestData)
	wc.writeMu.Unlock()
	if err != nil {
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
	if notification.Method != "slotNotification" {
		return
	}
	// payload: {"jsonrpc":"2.0","method":"slotNotification","params":{"result":{"parent":N,"root":N,"slot":N},"subscription":ID}}
	var wrap struct {
		Method string `json:"method"`
		Params struct {
			Subscription int `json:"subscription"`
			Result       struct {
				Parent uint64 `json:"parent"`
				Root   uint64 `json:"root"`
				Slot   uint64 `json:"slot"`
			} `json:"result"`
		} `json:"params"`
	}
	if err := json.Unmarshal(notification.Params, &wrap.Params); err != nil {
		sr.logger.Warn("Failed to parse slotNotification params", zap.Error(err))
		return
	}
	
	// Create block hash from the slot
	blockHash := fmt.Sprintf("slot:%d", wrap.Params.Result.Slot)
	
	// Check if we've already seen this block recently via the relay's deduper
	if sr.deduper != nil && sr.deduper.Seen(blockHash, time.Now(), "solana") {
		sr.logger.Debug("Suppressed duplicate Solana block",
			zap.Uint64("slot", wrap.Params.Result.Slot),
			zap.String("hash", blockHash))
		return
	}
	
	ev := blocks.BlockEvent{
		Hash:       blockHash,
		Height:     uint32(wrap.Params.Result.Slot),
		Timestamp:  time.Now(),
		DetectedAt: time.Now(),
		Source:     "solana-relay",
		Tier:       "enterprise",
	}
	
	select {
	case sr.blockChan <- ev:
	default:
	}
}

// subscribeToBlocks subscribes to slot updates (Solana's equivalent of blocks)
func (sr *SolanaRelay) subscribeToBlocks(ctx context.Context) error {
	// Subscribe to slot notifications
	_, err := sr.makeRequest("slotSubscribe", []interface{}{})
	return err
}

// scheduleReconnect schedules reconnect with exponential backoff per endpoint
func (sr *SolanaRelay) scheduleReconnect(endpoint string) {
	sr.backoffMu.Lock()
	
	// Check how many connections we still have
	sr.connMu.RLock()
	activeConnections := len(sr.connections)
	sr.connMu.RUnlock()
	
	// If this is a Cloudflare or Ankr endpoint and we have at least one working connection,
	// use a longer backoff to avoid unnecessary reconnection attempts
	isProblematicEndpoint := strings.Contains(endpoint, "cloudflare") || 
	                        strings.Contains(endpoint, "ankr") || 
	                        strings.Contains(endpoint, "api.mainnet-beta.solana.com")
	
	var attempt int
	if isProblematicEndpoint && activeConnections > 0 {
		// Use higher starting backoff for problematic endpoints if we have other working connections
		attempt = sr.backoff[endpoint] + 2
		if attempt > 8 {
			attempt = 8 // Cap at ~256s for problematic endpoints
		}
	} else {
		// Standard backoff for primary endpoints or when we have no connections
		attempt = sr.backoff[endpoint] + 1
		if attempt > 6 {
			attempt = 6 // Cap at ~32s
		}
	}
	
	sr.backoff[endpoint] = attempt
	sr.backoffMu.Unlock()

	// Calculate delay with more jitter for longer backoffs
	delay := time.Duration(1<<uint(attempt-1)) * time.Second
	jitterPercent := 0.2 // 20% jitter
	jitter := time.Duration(float64(delay) * jitterPercent * rand.Float64())
	wait := delay + jitter
	
	sr.logger.Info("Scheduling reconnect", 
		zap.String("endpoint", endpoint), 
		zap.Duration("in", wait),
		zap.Int("active_connections", activeConnections),
		zap.Int("attempt", attempt))
	
	time.AfterFunc(wait, func() {
		// Double check if we still need to reconnect
		sr.connMu.RLock()
		needToReconnect := true
		
		// If we have enough connections and this is a problematic endpoint,
		// we can skip reconnection attempt
		if isProblematicEndpoint && len(sr.connections) >= 1 {
			// Only skip every other attempt to ensure we keep trying occasionally
			if sr.backoff[endpoint]%2 == 0 {
				needToReconnect = false
			}
		}
		sr.connMu.RUnlock()
		
		if needToReconnect {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			sr.connectToEndpoint(ctx, endpoint)
		} else {
			sr.logger.Info("Skipping reconnect attempt, enough connections active", 
				zap.String("endpoint", endpoint))
		}
	})
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

// shouldReconnect determines if we should attempt to reconnect based on the error
func (sr *SolanaRelay) shouldReconnect(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific WebSocket close codes that indicate temporary issues
	if closeErr, ok := err.(*websocket.CloseError); ok {
		switch closeErr.Code {
		case websocket.CloseAbnormalClosure,
			 websocket.CloseGoingAway,
			 websocket.CloseInternalServerErr,
			 websocket.CloseTryAgainLater:
			return true
		}
	}

	// Reconnect on network-related errors
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		   strings.Contains(errStr, "connection reset") ||
		   strings.Contains(errStr, "broken pipe") ||
		   strings.Contains(errStr, "bad handshake") ||
		   strings.Contains(errStr, "tls") ||
		   strings.Contains(errStr, "lookup") ||
		   strings.Contains(errStr, "network is unreachable")
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
