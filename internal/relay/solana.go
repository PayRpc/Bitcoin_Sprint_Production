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
	
	// Enhanced components
	healthMgr *endpointHealth
	deduper   *SolanaDeduper
	metrics   *solanaProm
	metricsMu sync.RWMutex

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
	
	// Add any custom endpoints from config if available
	if customEndpoints := cfg.GetStringSlice("SOLANA_RPC_ENDPOINTS"); len(customEndpoints) > 0 {
		relayConfig.Endpoints = append(relayConfig.Endpoints, customEndpoints...)
		logger.Info("Added custom Solana endpoints from config", 
			zap.Strings("custom_endpoints", customEndpoints))
	}

	relay := &SolanaRelay{
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
		healthMgr: newEndpointHealth(relayConfig.Endpoints),
		deduper:   newSolanaDeduper(),
		metrics:   newSolanaProm("bitcoinsprint"),
	}
	
	// Start periodic health reporting
	go func() {
		relay.reportEndpointHealth(context.Background())
	}()
	
	return relay
}

// reportEndpointHealth reports on endpoint health metrics
func (sr *SolanaRelay) reportEndpointHealth(ctx context.Context) {
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			snap := sr.healthMgr.snapshot()
			for ep, st := range snap {
				sr.metrics.endpointLatency.WithLabelValues(ep).Set(st.ewmaRTT)
				sr.metrics.endpointScore.WithLabelValues(ep).Set(st.score())
				var state float64
				switch st.state {
				case breakerClosed:
					state = 0
				case breakerHalfOpen:
					state = 1
				case breakerOpen:
					state = 2
				}
				sr.metrics.endpointState.WithLabelValues(ep).Set(state)
			}
			
			// Log endpoint health every 5 minutes (roughly)
			if time.Now().Minute() % 5 == 0 && time.Now().Second() < 15 {
				// Only log if we have active connections
				sr.connMu.RLock()
				hasConnections := len(sr.connections) > 0
				sr.connMu.RUnlock()
				
				if !hasConnections {
					continue
				}
				
				// Count healthy/unhealthy endpoints
				var healthy, unhealthy int
				for _, st := range snap {
					if st.state != breakerOpen && st.successes > st.failures {
						healthy++
					} else {
						unhealthy++
					}
				}
				
				sr.logger.Info("Solana relay endpoint health status",
					zap.Int("healthy_endpoints", healthy),
					zap.Int("unhealthy_endpoints", unhealthy),
					zap.Bool("relay_healthy", sr.health.IsHealthy))
				
				// Log deduplication stats
				ttl, rate := sr.deduper.stats()
				sr.logger.Info("Solana block deduplication stats", 
					zap.Duration("ttl", ttl),
					zap.Float64("duplicate_rate", rate))
			}
		}
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
	
	// Start health/metrics reporter if not already running
	go sr.reportEndpointHealth(ctx)

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

// GetMetrics returns Solana relay metrics with enhanced endpoint and deduplication information
func (sr *SolanaRelay) GetMetrics() (*RelayMetrics, error) {
	sr.metricsMu.RLock()
	defer sr.metricsMu.RUnlock()
	
	// For now, return nil as RelayMetrics type may not be defined
	return nil, nil
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
	// We ignore the `endpoint` parameter and select the best available
	ep, ok := sr.healthMgr.pickWeighted()
	if !ok {
		sr.logger.Warn("No Solana endpoints available (breaker-open/all unhealthy)")
		return
	}
	
	u, err := url.Parse(ep)
	if err != nil {
		sr.logger.Warn("Invalid endpoint URL",
			zap.String("endpoint", ep),
			zap.Error(err))
		
		// Record error in endpoint health tracker
		sr.healthMgr.recordFailure(ep, err.Error())
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
	} else if strings.Contains(endpoint, "helius") {
		// Helius API requires API key
		apiKey := sr.cfg.Get("HELIUS_API_KEY", "")
		if apiKey != "" {
			// Helius uses apiKey URL parameter
			q := u.Query()
			q.Set("api-key", apiKey)
			u.RawQuery = q.Encode()
		}
	}

	var attempt int
	for {
		attempt++
		
		// Only try a limited number of times before giving up on this endpoint
		if attempt > 5 {
			sr.logger.Warn("Giving up connecting to Solana endpoint after multiple failures",
				zap.String("endpoint", ep),
				zap.Int("attempts", attempt))
			
			// Record multiple failures in endpoint health
			sr.healthMgr.recordFailure(ep, "max_retries_exceeded")
			return
		}
		
		// Measure connection time for metrics
		startTime := time.Now()
		
		dialCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		conn, _, err := dialer.DialContext(dialCtx, u.String(), header)
		connectionTime := time.Since(startTime)
		cancel()
		
		if err == nil {
			wc := &wsConn{
				Conn:     conn,
				logger:   sr.logger,
				endpoint: ep,
			}
			sr.installWSHandlers(wc)
			sr.addConnection(wc)
			
			// Record successful connection in endpoint health tracker
			sr.healthMgr.recordSuccess(ep, connectionTime)
			
			sr.logger.Info("Connected to Solana endpoint", 
				zap.String("endpoint", ep),
				zap.Duration("connection_time", connectionTime))
				
			// Update metrics
			sr.metrics.wsReconnects.Inc()
				
			// Start message handler
			go sr.handleMessages(wc)
			return
		}

		sr.logger.Warn("Failed to connect to Solana endpoint",
			zap.String("endpoint", ep),
			zap.Error(err),
			zap.Int("attempt", attempt),
			zap.Duration("connection_attempt_time", connectionTime))

		// Record failed connection in endpoint health tracker
		sr.healthMgr.recordFailure(ep, err.Error())
		
		// Update metrics
		sr.metrics.wsReconnects.Inc()

		// Backoff with jitter - more aggressive for problematic endpoints
		baseDelay := 2 * time.Second
		if strings.Contains(ep, "cloudflare") || strings.Contains(ep, "ankr") {
			baseDelay = 5 * time.Second // More aggressive backoff for known problematic endpoints
		}
		
		backoff := time.Duration(math.Min(float64(30*time.Second), float64(baseDelay)*math.Pow(2, float64(attempt))))
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
		
		// Record connection failure in health tracking
		sr.healthMgr.recordFailure(wc.endpoint, "connection_lost")
		
		sr.scheduleReconnect(wc.endpoint)
	}()

	for {
		_, message, err := wc.Conn.ReadMessage()
		if err != nil {
			sr.logger.Warn("WebSocket read error", 
				zap.String("endpoint", wc.endpoint),
				zap.Error(err))
			
			// Record read failure in health tracking
			sr.healthMgr.recordFailure(wc.endpoint, fmt.Sprintf("ws_read_error: %v", err))
			
			// Don't break immediately, try to reconnect
			if sr.shouldReconnect(err) {
				sr.logger.Info("Attempting to reconnect Solana WebSocket", zap.String("endpoint", wc.endpoint))
				return
			}
			return
		}

		// Track successful read
		sr.healthMgr.recordSuccess(wc.endpoint, 0)

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

// makeRequest makes a JSON-RPC request with intelligent endpoint selection
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

	// Get active connections
	sr.connMu.RLock()
	n := len(sr.connections)
	if n == 0 {
		sr.connMu.RUnlock()
		return nil, fmt.Errorf("no active connections")
	}
	
	// Map connections to their endpoints for selection
	connMap := make(map[string]*wsConn)
	for _, conn := range sr.connections {
		connMap[conn.endpoint] = conn
	}
	sr.connMu.RUnlock()
	
	// Use health manager to choose the best endpoint
	var wc *wsConn
	
	// Get best endpoint using weighted selection
	if bestEndpoint, ok := sr.healthMgr.pickWeighted(); ok {
		if conn, exists := connMap[bestEndpoint]; exists {
			wc = conn
			sr.logger.Debug("Selected endpoint using weighted health strategy",
				zap.String("endpoint", bestEndpoint),
				zap.String("method", method))
		}
	}
	
	// Fallback to random selection if health manager didn't provide a usable endpoint
	if wc == nil {
		sr.connMu.RLock()
		wc = sr.connections[rand.Intn(n)]
		sr.connMu.RUnlock()
		sr.logger.Debug("Using fallback random endpoint selection",
			zap.String("endpoint", wc.endpoint),
			zap.String("method", method))
	}

	// Create response channel
	responseChan := make(chan *SolanaResponse, 1)
	sr.reqMu.Lock()
	sr.pendingReqs[requestID] = responseChan
	sr.reqMu.Unlock()

	// Record request start time for metrics
	startTime := time.Now()

	// Send request
	wc.writeMu.Lock()
	_ = wc.Conn.SetWriteDeadline(time.Now().Add(8 * time.Second))
	err = wc.Conn.WriteMessage(websocket.TextMessage, requestData)
	wc.writeMu.Unlock()
	if err != nil {
		sr.reqMu.Lock()
		delete(sr.pendingReqs, requestID)
		sr.reqMu.Unlock()
		
		// Record error in endpoint health tracker
		sr.healthMgr.recordFailure(wc.endpoint, fmt.Sprintf("write_error: %v", err))
		
		return nil, fmt.Errorf("failed to send request to %s: %w", wc.endpoint, err)
	}

	// Wait for response with timeout
	var response *SolanaResponse
	select {
	case response = <-responseChan:
		// Record successful response in endpoint health tracker
		responseTime := time.Since(startTime)
		sr.healthMgr.recordSuccess(wc.endpoint, responseTime)
		
		// Update metrics
		// Note: Detailed request metrics not implemented in current solanaProm struct
		
		// Check for errors in the response
		if response.Error != nil {
			// Some errors should be considered endpoint health issues
			if response.Error.Code < -32000 || response.Error.Code == -32603 || response.Error.Code == -32010 {
				sr.healthMgr.recordFailure(wc.endpoint, fmt.Sprintf("rpc_error: %d: %s", 
					response.Error.Code, response.Error.Message))
				
				sr.logger.Warn("Solana RPC error affects endpoint health",
					zap.String("endpoint", wc.endpoint),
					zap.Int("error_code", response.Error.Code),
					zap.String("error_message", response.Error.Message))
			}
		}
		
		return response, nil
	case <-time.After(sr.relayConfig.Timeout):
		sr.reqMu.Lock()
		delete(sr.pendingReqs, requestID)
		sr.reqMu.Unlock()
		
		// Record timeout in endpoint health tracker
		sr.healthMgr.recordFailure(wc.endpoint, "request_timeout")
		
		return nil, fmt.Errorf("request timeout for %s", wc.endpoint)
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
	// Validate input
	if notification == nil {
		sr.logger.Warn("Received nil Solana notification")
		return
	}
	
	if notification.Method != "slotNotification" {
		// Not a block notification, skip it
		return
	}
	
	// Validate parameters
	if len(notification.Params) == 0 {
		sr.logger.Warn("Received empty Solana notification params")
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
	
	// Validate the block hash/slot (Solana slots are always > 0 for real blocks)
	if wrap.Params.Result.Slot == 0 {
		sr.logger.Warn("Received Solana notification with invalid slot",
			zap.Uint64("slot", wrap.Params.Result.Slot))
		return
	}
	
	now := time.Now()
	
	// Check if we've already seen this block recently via the adaptive deduper
	if sr.deduper.isDup(blockHash) {
		// Update metrics for duplicates
		sr.metrics.dupDropped.Inc()
		
		// Only log at debug level to avoid flooding logs
		sr.logger.Debug("Suppressed duplicate Solana block",
			zap.Uint64("slot", wrap.Params.Result.Slot),
			zap.String("hash", blockHash))
		return
	}
	
	// Create rich block event with additional metadata
	ev := blocks.BlockEvent{
		Hash:       blockHash,
		Height:     uint32(wrap.Params.Result.Slot),
		Timestamp:  now,
		DetectedAt: now,
		Source:     "solana-relay",
		Tier:       "enterprise",
	}
	
	// Update metrics for successful block
	sr.metrics.dupDropped.Inc()
	
	// Forward to block channel with non-blocking send to prevent backpressure
	select {
	case sr.blockChan <- ev:
		// Successfully sent
		sr.logger.Debug("Forwarded Solana block event",
			zap.Uint64("slot", wrap.Params.Result.Slot),
			zap.String("hash", blockHash))
	default:
		// Channel full - update metrics and log warning
		sr.metrics.dupDropped.Inc()
		
		sr.logger.Warn("Dropped Solana block due to full channel",
			zap.Uint64("slot", wrap.Params.Result.Slot),
			zap.String("hash", blockHash),
			zap.Int("channel_capacity", cap(sr.blockChan)),
			zap.Int("channel_len", len(sr.blockChan)))
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
	
	// If we have no active connections, we need to try to reconnect to something
	// even if healthMgr says all endpoints are bad
	forcedReconnect := activeConnections == 0

	// Use health manager for endpoint selection
	ep := endpoint // default to the endpoint that just disconnected
	if !forcedReconnect {
		// Let the health manager choose a good endpoint
		if selected, ok := sr.healthMgr.pickWeighted(); ok {
			ep = selected
			sr.logger.Debug("Using health manager to select reconnection endpoint",
				zap.String("selected", ep),
				zap.String("original", endpoint))
		}
	}
	
	// Use adaptive backoff
	attempt := sr.backoff[ep] + 1
	maxAttempt := 6
	
	// Higher cap for problematic endpoints
	isProblematicEndpoint := strings.Contains(ep, "cloudflare") || 
	                        strings.Contains(ep, "ankr") || 
	                        strings.Contains(ep, "api.mainnet-beta.solana.com")
	if isProblematicEndpoint && activeConnections > 0 {
		maxAttempt = 8 // Cap at ~256s
		attempt += 1   // Start with higher backoff
	}
	
	if attempt > maxAttempt {
		attempt = maxAttempt
	}
	
	sr.backoff[ep] = attempt
	sr.backoffMu.Unlock()

	// Calculate delay with more jitter for longer backoffs
	delay := time.Duration(1<<uint(attempt-1)) * time.Second
	jitterPercent := 0.2 // 20% jitter
	jitter := time.Duration(float64(delay) * jitterPercent * rand.Float64())
	wait := delay + jitter
	
	sr.logger.Info("Scheduling reconnect", 
		zap.String("endpoint", ep), 
		zap.Duration("in", wait),
		zap.Int("active_connections", activeConnections),
		zap.Int("attempt", attempt))
	
	// Record the reconnect attempt in metrics
	sr.metrics.wsReconnects.Inc()
	
	time.AfterFunc(wait, func() {
		// Double check if we still need to reconnect
		sr.connMu.RLock()
		needToReconnect := len(sr.connections) < 1 // Only need to reconnect if no connections
		sr.connMu.RUnlock()
		
		// If we have enough connections, defer to the health manager
		if !needToReconnect {
			// Let the health manager decide if this endpoint is worth trying
			sr.healthMgr.mu.RLock()
			stats, exists := sr.healthMgr.stats[ep]
			sr.healthMgr.mu.RUnlock()
			if exists && stats.state != breakerOpen {
				// Endpoint is not in circuit breaker open state, try to connect
				needToReconnect = true
			} else {
				sr.logger.Info("Health manager suggests skipping reconnect (circuit breaker open)",
					zap.String("endpoint", ep))
			}
		}
		
		if needToReconnect {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			sr.connectToEndpoint(ctx, ep)
		} else {
			sr.logger.Info("Skipping reconnect attempt, enough connections active or endpoint in circuit breaker", 
				zap.String("endpoint", ep))
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
