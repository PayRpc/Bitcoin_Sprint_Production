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

// wsConn wraps websocket.Conn with thread-safe write operations
type wsConn struct {
	Conn     *websocket.Conn
	logger   *zap.Logger
	writeMu  sync.Mutex
	endpoint string
}

// WriteMessage sends a message through the WebSocket connection with thread safety
func (w *wsConn) WriteMessage(messageType int, data []byte) error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	return w.Conn.WriteMessage(messageType, data)
}

// ReadMessage reads a message from the WebSocket connection
func (w *wsConn) ReadMessage() (int, []byte, error) {
	return w.Conn.ReadMessage()
}

// Close closes the WebSocket connection
func (w *wsConn) Close() error {
	return w.Conn.Close()
}

// EthereumRelay implements RelayClient for Ethereum network using JSON-RPC WebSocket
type EthereumRelay struct {
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
	pendingReqs map[int64]chan *EthereumResponse
	reqMu       sync.RWMutex

	// Subscription management
	subscriptions map[string]chan *EthereumNotification
	subMu         sync.RWMutex

	// backoff per endpoint
	backoffMu sync.Mutex
	backoff   map[string]int // attempt counter
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
		Endpoints:         []string{"wss://ethereum.publicnode.com", "wss://cloudflare-eth.com", "wss://rpc.ankr.com/eth/ws"},
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
		connections:   make([]*wsConn, 0),
		blockChan:     make(chan blocks.BlockEvent, 1000),
		pendingReqs:   make(map[int64]chan *EthereumResponse),
		subscriptions: make(map[string]chan *EthereumNotification),
		backoff:       make(map[string]int),
		health: &HealthStatus{
			IsHealthy:       false,
			ConnectionState: "disconnected",
		},
		metrics: &RelayMetrics{},
		deduper: NewBlockDeduper(4096, 3*time.Minute), // Ethereum-specific deduper
	}
}

// Connect establishes WebSocket connections to Ethereum nodes
func (er *EthereumRelay) Connect(ctx context.Context) error {
	if er.connected.Load() {
		return nil
	}

	er.logger.Info("Connecting to Ethereum network",
		zap.Strings("endpoints", er.relayConfig.Endpoints))

	// Try to connect to all endpoints in parallel
	for _, endpoint := range er.relayConfig.Endpoints {
		go er.connectToEndpoint(ctx, endpoint)
	}

	// The connected flag will be set when the first connection succeeds
	// in addConnection, not immediately here

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

// addConnection adds a connection to the active set
func (er *EthereumRelay) addConnection(wc *wsConn) {
	er.connMu.Lock()
	defer er.connMu.Unlock()
	er.connections = append(er.connections, wc)
	if len(er.connections) == 1 {
		er.connected.Store(true)
		er.updateHealth(true, "connected", nil)
	}
}

// removeConnection removes a connection from the active set
func (er *EthereumRelay) removeConnection(wc *wsConn) {
	er.connMu.Lock()
	defer er.connMu.Unlock()
	out := er.connections[:0]
	for _, c := range er.connections {
		if c != wc {
			out = append(out, c)
		}
	}
	er.connections = out
	if len(er.connections) == 0 {
		er.connected.Store(false)
		er.updateHealth(false, "disconnected", nil)
	}
}

// scheduleReconnect schedules reconnect with exponential backoff per endpoint
func (er *EthereumRelay) scheduleReconnect(endpoint string) {
	er.backoffMu.Lock()
	
	// Check how many connections we still have
	er.connMu.RLock()
	activeConnections := len(er.connections)
	er.connMu.RUnlock()
	
	// If this is a Cloudflare or Ankr endpoint and we have at least one working connection,
	// use a longer backoff to avoid unnecessary reconnection attempts
	isProblematicEndpoint := strings.Contains(endpoint, "cloudflare") || 
	                        strings.Contains(endpoint, "ankr") || 
	                        strings.Contains(endpoint, "infura")
	
	var attempt int
	if isProblematicEndpoint && activeConnections > 0 {
		// Use higher starting backoff for problematic endpoints if we have other working connections
		attempt = er.backoff[endpoint] + 2
		if attempt > 8 {
			attempt = 8 // Cap at ~256s for problematic endpoints
		}
	} else {
		// Standard backoff for primary endpoints or when we have no connections
		attempt = er.backoff[endpoint] + 1
		if attempt > 6 {
			attempt = 6 // Cap at ~32s
		}
	}
	
	er.backoff[endpoint] = attempt
	er.backoffMu.Unlock()

	// Calculate delay with more jitter for longer backoffs
	delay := time.Duration(1<<uint(attempt-1)) * time.Second
	jitterPercent := 0.2 // 20% jitter
	jitter := time.Duration(float64(delay) * jitterPercent * rand.Float64())
	wait := delay + jitter
	
	er.logger.Info("Scheduling reconnect", 
		zap.String("endpoint", endpoint), 
		zap.Duration("in", wait),
		zap.Int("active_connections", activeConnections),
		zap.Int("attempt", attempt))
	
	time.AfterFunc(wait, func() {
		// Double check if we still need to reconnect
		er.connMu.RLock()
		needToReconnect := true
		
		// If we have enough connections and this is a problematic endpoint,
		// we can skip reconnection attempt
		if isProblematicEndpoint && len(er.connections) >= 1 {
			// Only skip every other attempt to ensure we keep trying occasionally
			if er.backoff[endpoint]%2 == 0 {
				needToReconnect = false
			}
		}
		er.connMu.RUnlock()
		
		if needToReconnect {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			er.connectToEndpoint(ctx, endpoint)
		} else {
			er.logger.Info("Skipping reconnect attempt, enough connections active", 
				zap.String("endpoint", endpoint))
		}
	})
}

// shouldReconnect determines if we should attempt to reconnect based on the error
func (er *EthereumRelay) shouldReconnect(err error) bool {
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

	// Create WebSocket dialer with resolver-aware NetDialContext
	dialer := websocket.Dialer{
		Proxy:             websocket.DefaultDialer.Proxy,
		HandshakeTimeout:  20 * time.Second, // Increased from 12 to 20 seconds
		TLSClientConfig:   &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: false},
		NetDialContext:    netx.DialerWithResolver(),
		EnableCompression: true,
	}

	// Base headers for all endpoints
	header := http.Header{}
	header.Set("Origin", "https://bitcoinsprint.com")
	header.Set("User-Agent", "BitcoinSprint/2.5.0 (+https://bitcoinsprint.com)")
	header.Set("Pragma", "no-cache")
	header.Set("Cache-Control", "no-cache")
	
	// Endpoint-specific configuration
	if strings.Contains(endpoint, "cloudflare") {
		// Cloudflare requires specific headers
		header.Set("Origin", "https://www.cloudflare-eth.com")
		header.Set("CF-Access-Client-Id", er.cfg.Get("CF_ACCESS_CLIENT_ID", ""))
		header.Set("CF-Access-Client-Secret", er.cfg.Get("CF_ACCESS_CLIENT_SECRET", ""))
	} else if strings.Contains(endpoint, "ankr") {
		// Ankr API requires JWT or API key
		apiKey := er.cfg.Get("ANKR_API_KEY", "")
		if apiKey != "" {
			header.Set("Authorization", "Bearer "+apiKey)
		}
		header.Set("Origin", "https://www.ankr.com")
	} else if strings.Contains(endpoint, "infura") {
		// Infura may require API key or project ID
		projectId := er.cfg.Get("INFURA_PROJECT_ID", "")
		if projectId != "" && !strings.Contains(endpoint, projectId) {
			// Only add if not already in the URL
			if strings.Contains(endpoint, "?") {
				u.RawQuery += "&projectId=" + projectId
			} else {
				u.RawQuery = "projectId=" + projectId
			}
		}
	}

	// Use exponential backoff with jitter for reconnect attempts
	er.backoffMu.Lock()
	attempt := er.backoff[endpoint]
	er.backoff[endpoint] = attempt + 1
	er.backoffMu.Unlock()

	for {
		// per-attempt timeout to avoid long DNS hangs
		dialCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		conn, resp, err := dialer.DialContext(dialCtx, u.String(), header)
		cancel()
		if err == nil {
			if resp != nil {
				resp.Body.Close()
			}
			wsConn := &wsConn{
				Conn:     conn,
				logger:   er.logger,
				endpoint: endpoint,
			}
			
			// Install ping/pong and heartbeat handlers
			er.installWSHandlers(wsConn)
			
			// Add connection to active set
			er.addConnection(wsConn)

			// Reset backoff on successful connection
			er.backoffMu.Lock()
			delete(er.backoff, endpoint)
			er.backoffMu.Unlock()

			er.logger.Info("Connected to Ethereum endpoint", zap.String("endpoint", endpoint))
			// Start message handler
			go er.handleMessages(wsConn)
			return
		}

		er.logger.Warn("Failed to connect to Ethereum endpoint",
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
			// try again
			attempt++
			er.backoffMu.Lock()
			er.backoff[endpoint] = attempt
			er.backoffMu.Unlock()
		}
	}
}

// installWSHandlers sets up ping/pong and heartbeat handlers for the WebSocket connection
func (er *EthereumRelay) installWSHandlers(wc *wsConn) {
	// Set a more aggressive initial read deadline
	_ = wc.Conn.SetReadDeadline(time.Now().Add(45 * time.Second))
	
	// Enhanced pong handler with logging
	wc.Conn.SetPongHandler(func(data string) error {
		_ = wc.Conn.SetReadDeadline(time.Now().Add(45 * time.Second))
		er.logger.Debug("Received pong", 
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
				er.connMu.RLock()
				alive := false
				for _, c := range er.connections {
					if c == wc {
						alive = true
						break
					}
				}
				er.connMu.RUnlock()
				
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
					er.logger.Warn("Ping failed", 
						zap.String("endpoint", wc.endpoint), 
						zap.Error(err))
					return
				}
				
			case <-heartbeatTicker.C:
				// Send a heartbeat message to keep connection alive
				er.sendHeartbeat(wc)
			}
		}
	}()
}

// sendHeartbeat sends a lightweight RPC call to keep the connection active
func (er *EthereumRelay) sendHeartbeat(wc *wsConn) {
	// For Ethereum connections: eth_blockNumber is very lightweight
	requestData := []byte(`{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":0}`)
	
	wc.writeMu.Lock()
	_ = wc.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	err := wc.Conn.WriteMessage(websocket.TextMessage, requestData)
	wc.writeMu.Unlock()
	
	if err != nil {
		er.logger.Warn("Failed to send heartbeat", 
			zap.String("endpoint", wc.endpoint), 
			zap.Error(err))
	} else {
		er.logger.Debug("Sent heartbeat to keep connection alive", 
			zap.String("endpoint", wc.endpoint))
	}
}

// handleMessages handles incoming WebSocket messages
func (er *EthereumRelay) handleMessages(conn *wsConn) {
	defer func() {
		conn.Close()
		// Remove connection from active set
		er.removeConnection(conn)
		// Schedule reconnect
		er.scheduleReconnect(conn.endpoint)
		// Mark as disconnected when handler exits
		er.updateHealth(er.IsConnected(), "connection_lost", nil)
		er.logger.Warn("Ethereum WebSocket handler exited", 
			zap.String("endpoint", conn.endpoint),
			zap.Int("remaining_connections", len(er.connections)))
	}()

	for {
		// Reset read deadline on each loop iteration
		_ = conn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		
		_, message, err := conn.ReadMessage()
		if err != nil {
			er.logger.Warn("WebSocket read error", 
				zap.String("endpoint", conn.endpoint),
				zap.Error(err))
			
			// Don't attempt to reconnect here, let the scheduleReconnect in the defer handle it
			return
		}

		// Parse message as JSON-RPC response or notification
		var response EthereumResponse
		if err := json.Unmarshal(message, &response); err == nil && response.ID > 0 {
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
	// Validate input
	if notification == nil || len(notification.Params) == 0 {
		er.logger.Warn("Received empty Ethereum notification")
		return
	}
	
	// Parse subscription notification
	var result struct {
		Subscription string          `json:"subscription"`
		Result       EthereumBlock   `json:"result"`
	}
	
	if err := json.Unmarshal(notification.Params, &result); err != nil {
		er.logger.Warn("Failed to parse Ethereum block notification", zap.Error(err))
		return
	}
	
	// Extract block info
	blockHash := result.Result.Hash
	
	// Validate the block hash (should be a proper hex string starting with 0x)
	if blockHash == "" || blockHash == "0x0000000000000000000000000000000000000000000000000000000000000000" || !strings.HasPrefix(blockHash, "0x") {
		er.logger.Warn("Received block notification with invalid hash", 
			zap.String("hash", blockHash),
			zap.String("number", result.Result.Number))
		return
	}
	
	// Check if we've already seen this block recently via the deduper
	// Safely handle the deduper
	if er.deduper != nil {
		if er.deduper.Seen(blockHash, time.Now(), "ethereum") {
			er.logger.Debug("Suppressed duplicate Ethereum block", 
				zap.String("hash", blockHash),
				zap.String("number", result.Result.Number))
			return
		}
	} else {
		// No deduper configured, but let's log this for debugging
		er.logger.Debug("Deduper not configured, processing all blocks", 
			zap.String("hash", blockHash))
	}
	
	// Convert to BlockEvent
	blockEvent := er.convertToBlockEvent(&result.Result)
	
	// Send to block channel
	select {
	case er.blockChan <- *blockEvent:
		// Successfully sent
	default:
		// Channel full, drop block
		er.logger.Warn("Block channel full, dropping block", 
			zap.String("hash", blockHash))
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

func (er *EthereumRelay) reconnect() error {
	er.logger.Info("Reconnecting Ethereum WebSocket")

	// Disconnect first to clean up
	er.Disconnect()

	// Wait a bit before reconnecting
	time.Sleep(2 * time.Second)

	// Try to connect again
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return er.Connect(ctx)
}
