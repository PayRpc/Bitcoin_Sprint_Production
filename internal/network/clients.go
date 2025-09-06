package network

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// EthereumClient provides high-performance Ethereum connectivity
type EthereumClient struct {
	manager *NetworkManager
	logger  *zap.Logger
	
	// Connection management
	mu               sync.RWMutex
	activeEndpoints  []*NodeEndpoint
	currentEndpoint  *NodeEndpoint
	
	// Request tracking
	requestCounter   int64
	lastEndpointSwitch time.Time
	switchThreshold   time.Duration
	
	// Performance optimization
	connectionPool   map[string]*http.Client
	responseCache    map[string]*CachedResponse
	cacheMutex       sync.RWMutex
	cacheExpiration  time.Duration
}

// SolanaClient provides high-performance Solana connectivity
type SolanaClient struct {
	manager *NetworkManager
	logger  *zap.Logger
	
	// Connection management
	mu               sync.RWMutex
	activeEndpoints  []*NodeEndpoint
	currentEndpoint  *NodeEndpoint
	
	// Request tracking
	requestCounter   int64
	lastEndpointSwitch time.Time
	switchThreshold   time.Duration
	
	// Performance optimization
	connectionPool   map[string]*http.Client
	responseCache    map[string]*CachedResponse
	cacheMutex       sync.RWMutex
	cacheExpiration  time.Duration
}

// BitcoinClient provides high-performance Bitcoin P2P connectivity
type BitcoinClient struct {
	manager *NetworkManager
	logger  *zap.Logger
	
	// P2P connections
	mu               sync.RWMutex
	activeConnections map[string]*BitcoinP2PConnection
	healthyPeers     []*NodeEndpoint
	primaryPeer      *NodeEndpoint
	
	// Connection pool management
	maxConnections   int
	connectionHealth map[string]*ConnectionHealth
	reconnectDelay   time.Duration
}

// CachedResponse represents a cached API response
type CachedResponse struct {
	Data      []byte    `json:"data"`
	Timestamp time.Time `json:"timestamp"`
	TTL       time.Duration `json:"ttl"`
}

// ConnectionHealth tracks individual connection health
type ConnectionHealth struct {
	LastSeen       time.Time     `json:"last_seen"`
	MessageCount   int64         `json:"message_count"`
	ErrorCount     int64         `json:"error_count"`
	Latency        time.Duration `json:"latency"`
	IsResponsive   bool          `json:"is_responsive"`
}

// BitcoinP2PConnection represents a Bitcoin P2P connection
type BitcoinP2PConnection struct {
	Endpoint     *NodeEndpoint     `json:"endpoint"`
	Connected    bool              `json:"connected"`
	LastMessage  time.Time         `json:"last_message"`
	Version      int32             `json:"version"`
	Services     uint64            `json:"services"`
	UserAgent    string            `json:"user_agent"`
	StartHeight  int32             `json:"start_height"`
	Health       *ConnectionHealth `json:"health"`
}

// NewEthereumClient creates a new Ethereum client
func NewEthereumClient(manager *NetworkManager, logger *zap.Logger) *EthereumClient {
	return &EthereumClient{
		manager:          manager,
		logger:           logger,
		connectionPool:   make(map[string]*http.Client),
		responseCache:    make(map[string]*CachedResponse),
		switchThreshold:  30 * time.Second,
		cacheExpiration:  5 * time.Second,
	}
}

// NewSolanaClient creates a new Solana client
func NewSolanaClient(manager *NetworkManager, logger *zap.Logger) *SolanaClient {
	return &SolanaClient{
		manager:          manager,
		logger:           logger,
		connectionPool:   make(map[string]*http.Client),
		responseCache:    make(map[string]*CachedResponse),
		switchThreshold:  30 * time.Second,
		cacheExpiration:  3 * time.Second,
	}
}

// NewBitcoinClient creates a new Bitcoin client
func NewBitcoinClient(manager *NetworkManager, logger *zap.Logger) *BitcoinClient {
	return &BitcoinClient{
		manager:           manager,
		logger:            logger,
		activeConnections: make(map[string]*BitcoinP2PConnection),
		connectionHealth:  make(map[string]*ConnectionHealth),
		maxConnections:    10,
		reconnectDelay:    5 * time.Second,
	}
}

// Initialize sets up the Ethereum client
func (ec *EthereumClient) Initialize(ctx context.Context) error {
	ec.logger.Info("Initializing Ethereum client")
	
	// Get healthy endpoints
	ec.refreshEndpoints()
	
	// Setup connection pools
	ec.setupConnectionPools()
	
	// Start performance monitoring
	go ec.monitorPerformance(ctx)
	
	// Start cache cleanup
	go ec.cleanupCache(ctx)
	
	return nil
}

// Initialize sets up the Solana client
func (sc *SolanaClient) Initialize(ctx context.Context) error {
	sc.logger.Info("Initializing Solana client")
	
	// Get healthy endpoints
	sc.refreshEndpoints()
	
	// Setup connection pools
	sc.setupConnectionPools()
	
	// Start performance monitoring
	go sc.monitorPerformance(ctx)
	
	// Start cache cleanup
	go sc.cleanupCache(ctx)
	
	return nil
}

// Initialize sets up the Bitcoin client
func (bc *BitcoinClient) Initialize(ctx context.Context) error {
	bc.logger.Info("Initializing Bitcoin client")
	
	// Get healthy P2P endpoints
	bc.refreshPeers()
	
	// Establish P2P connections
	go bc.maintainP2PConnections(ctx)
	
	// Monitor connection health
	go bc.monitorConnectionHealth(ctx)
	
	return nil
}

// CallMethod makes an Ethereum RPC call with intelligent endpoint selection
func (ec *EthereumClient) CallMethod(ctx context.Context, method string, params []interface{}) (json.RawMessage, error) {
	// Check cache first
	cacheKey := ec.getCacheKey(method, params)
	if cached := ec.getFromCache(cacheKey); cached != nil {
		return cached, nil
	}
	
	// Select best endpoint
	endpoint, err := ec.selectBestEndpoint()
	if err != nil {
		return nil, fmt.Errorf("no healthy endpoints available: %w", err)
	}
	
	// Prepare request
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	}
	
	// Execute request with retries
	result, err := ec.executeRequest(ctx, endpoint, request)
	if err != nil {
		// Try backup endpoint
		if backup := ec.getBackupEndpoint(endpoint); backup != nil {
			result, err = ec.executeRequest(ctx, backup, request)
		}
	}
	
	if err == nil && result != nil {
		// Cache successful response
		ec.cacheResponse(cacheKey, result)
	}
	
	return result, err
}

// CallMethod makes a Solana RPC call with intelligent endpoint selection
func (sc *SolanaClient) CallMethod(ctx context.Context, method string, params []interface{}) (json.RawMessage, error) {
	// Check cache first
	cacheKey := sc.getCacheKey(method, params)
	if cached := sc.getFromCache(cacheKey); cached != nil {
		return cached, nil
	}
	
	// Select best endpoint
	endpoint, err := sc.selectBestEndpoint()
	if err != nil {
		return nil, fmt.Errorf("no healthy endpoints available: %w", err)
	}
	
	// Prepare request
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	}
	
	// Execute request with retries
	result, err := sc.executeRequest(ctx, endpoint, request)
	if err != nil {
		// Try backup endpoint
		if backup := sc.getBackupEndpoint(endpoint); backup != nil {
			result, err = sc.executeRequest(ctx, backup, request)
		}
	}
	
	if err == nil && result != nil {
		// Cache successful response
		sc.cacheResponse(cacheKey, result)
	}
	
	return result, err
}

// GetBlockData retrieves block data from Bitcoin P2P network
func (bc *BitcoinClient) GetBlockData(ctx context.Context, blockHash string) ([]byte, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	// Find best peer for block request
	peer := bc.selectBestPeer()
	if peer == nil {
		return nil, fmt.Errorf("no healthy Bitcoin peers available")
	}
	
	// Request block data through P2P
	return bc.requestBlockFromPeer(ctx, peer, blockHash)
}

// selectBestEndpoint chooses the optimal Ethereum endpoint
func (ec *EthereumClient) selectBestEndpoint() (*NodeEndpoint, error) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	
	if len(ec.activeEndpoints) == 0 {
		ec.refreshEndpoints()
		if len(ec.activeEndpoints) == 0 {
			return nil, fmt.Errorf("no active Ethereum endpoints")
		}
	}
	
	// Use current endpoint if it's performing well
	if ec.currentEndpoint != nil && 
		ec.currentEndpoint.IsHealthy && 
		time.Since(ec.lastEndpointSwitch) < ec.switchThreshold {
		return ec.currentEndpoint, nil
	}
	
	// Find best performing endpoint
	best := ec.manager.selectBestEndpoint(ec.activeEndpoints)
	if best != ec.currentEndpoint {
		ec.currentEndpoint = best
		ec.lastEndpointSwitch = time.Now()
		ec.logger.Info("Switched to new Ethereum endpoint",
			zap.String("url", best.URL),
			zap.String("provider", best.Provider))
	}
	
	return best, nil
}

// selectBestEndpoint chooses the optimal Solana endpoint
func (sc *SolanaClient) selectBestEndpoint() (*NodeEndpoint, error) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	
	if len(sc.activeEndpoints) == 0 {
		sc.refreshEndpoints()
		if len(sc.activeEndpoints) == 0 {
			return nil, fmt.Errorf("no active Solana endpoints")
		}
	}
	
	// Use current endpoint if it's performing well
	if sc.currentEndpoint != nil && 
		sc.currentEndpoint.IsHealthy && 
		time.Since(sc.lastEndpointSwitch) < sc.switchThreshold {
		return sc.currentEndpoint, nil
	}
	
	// Find best performing endpoint
	best := sc.manager.selectBestEndpoint(sc.activeEndpoints)
	if best != sc.currentEndpoint {
		sc.currentEndpoint = best
		sc.lastEndpointSwitch = time.Now()
		sc.logger.Info("Switched to new Solana endpoint",
			zap.String("url", best.URL),
			zap.String("provider", best.Provider))
	}
	
	return best, nil
}

// selectBestPeer chooses the optimal Bitcoin peer
func (bc *BitcoinClient) selectBestPeer() *BitcoinP2PConnection {
	var best *BitcoinP2PConnection
	var bestScore float64
	
	for _, conn := range bc.activeConnections {
		if !conn.Connected || !conn.Health.IsResponsive {
			continue
		}
		
		// Calculate peer score based on health metrics
		score := bc.calculatePeerScore(conn)
		if score > bestScore {
			bestScore = score
			best = conn
		}
	}
	
	return best
}

// executeRequest performs HTTP request with performance tracking
func (ec *EthereumClient) executeRequest(ctx context.Context, endpoint *NodeEndpoint, request map[string]interface{}) (json.RawMessage, error) {
	start := time.Now()
	
	// Get HTTP client for endpoint
	client := ec.getHTTPClient(endpoint)
	
	// Serialize request
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	
	// Create HTTP request
	url := "https://" + endpoint.URL
	if endpoint.URL[:4] == "http" {
		url = endpoint.URL
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	if endpoint.CustomHeaders != nil {
		for key, value := range endpoint.CustomHeaders {
			httpReq.Header.Set(key, value)
		}
	}
	
	// Execute request
	resp, err := client.Do(httpReq)
	if err != nil {
		// Track failure
		ec.manager.metricsChan <- &MetricsUpdate{
			Endpoint:    endpoint,
			RequestTime: time.Since(start),
			Success:     false,
			Timestamp:   time.Now(),
		}
		return nil, err
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	// Track success
	ec.manager.metricsChan <- &MetricsUpdate{
		Endpoint:    endpoint,
		RequestTime: time.Since(start),
		Success:     true,
		Timestamp:   time.Now(),
	}
	
	// Parse JSON response
	var jsonResp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return nil, err
	}
	
	if jsonResp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", jsonResp.Error.Code, jsonResp.Error.Message)
	}
	
	return jsonResp.Result, nil
}

// executeRequest performs HTTP request for Solana
func (sc *SolanaClient) executeRequest(ctx context.Context, endpoint *NodeEndpoint, request map[string]interface{}) (json.RawMessage, error) {
	start := time.Now()
	
	// Get HTTP client for endpoint
	client := sc.getHTTPClient(endpoint)
	
	// Serialize request
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	
	// Create HTTP request
	url := endpoint.URL
	if endpoint.URL[:4] != "http" {
		url = "https://" + endpoint.URL
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	if endpoint.CustomHeaders != nil {
		for key, value := range endpoint.CustomHeaders {
			httpReq.Header.Set(key, value)
		}
	}
	
	// Execute request
	resp, err := client.Do(httpReq)
	if err != nil {
		// Track failure
		sc.manager.metricsChan <- &MetricsUpdate{
			Endpoint:    endpoint,
			RequestTime: time.Since(start),
			Success:     false,
			Timestamp:   time.Now(),
		}
		return nil, err
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	// Track success
	sc.manager.metricsChan <- &MetricsUpdate{
		Endpoint:    endpoint,
		RequestTime: time.Since(start),
		Success:     true,
		Timestamp:   time.Now(),
	}
	
	// Parse JSON response
	var jsonResp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return nil, err
	}
	
	if jsonResp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", jsonResp.Error.Code, jsonResp.Error.Message)
	}
	
	return jsonResp.Result, nil
}

// Helper methods for caching, monitoring, and connection management
func (ec *EthereumClient) getCacheKey(method string, params []interface{}) string {
	data, _ := json.Marshal(map[string]interface{}{"method": method, "params": params})
	return fmt.Sprintf("eth:%x", data)
}

func (sc *SolanaClient) getCacheKey(method string, params []interface{}) string {
	data, _ := json.Marshal(map[string]interface{}{"method": method, "params": params})
	return fmt.Sprintf("sol:%x", data)
}

func (ec *EthereumClient) getFromCache(key string) json.RawMessage {
	ec.cacheMutex.RLock()
	defer ec.cacheMutex.RUnlock()
	
	if cached, exists := ec.responseCache[key]; exists {
		if time.Since(cached.Timestamp) < cached.TTL {
			return cached.Data
		}
		// Remove expired cache entry
		delete(ec.responseCache, key)
	}
	
	return nil
}

func (sc *SolanaClient) getFromCache(key string) json.RawMessage {
	sc.cacheMutex.RLock()
	defer sc.cacheMutex.RUnlock()
	
	if cached, exists := sc.responseCache[key]; exists {
		if time.Since(cached.Timestamp) < cached.TTL {
			return cached.Data
		}
		// Remove expired cache entry
		delete(sc.responseCache, key)
	}
	
	return nil
}

func (ec *EthereumClient) cacheResponse(key string, data json.RawMessage) {
	ec.cacheMutex.Lock()
	defer ec.cacheMutex.Unlock()
	
	ec.responseCache[key] = &CachedResponse{
		Data:      data,
		Timestamp: time.Now(),
		TTL:       ec.cacheExpiration,
	}
}

func (sc *SolanaClient) cacheResponse(key string, data json.RawMessage) {
	sc.cacheMutex.Lock()
	defer sc.cacheMutex.Unlock()
	
	sc.responseCache[key] = &CachedResponse{
		Data:      data,
		Timestamp: time.Now(),
		TTL:       sc.cacheExpiration,
	}
}

// Additional helper methods would be implemented here for:
// - refreshEndpoints()
// - setupConnectionPools()
// - getHTTPClient()
// - getBackupEndpoint()
// - monitorPerformance()
// - cleanupCache()
// - maintainP2PConnections()
// - monitorConnectionHealth()
// - requestBlockFromPeer()
// - calculatePeerScore()
// etc.

// refreshEndpoints updates the list of active endpoints
func (ec *EthereumClient) refreshEndpoints() {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	
	ec.activeEndpoints = ec.manager.GetHealthyEndpoints(Ethereum)
	ec.logger.Info("Refreshed Ethereum endpoints",
		zap.Int("healthy_count", len(ec.activeEndpoints)))
}

func (sc *SolanaClient) refreshEndpoints() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	sc.activeEndpoints = sc.manager.GetHealthyEndpoints(Solana)
	sc.logger.Info("Refreshed Solana endpoints",
		zap.Int("healthy_count", len(sc.activeEndpoints)))
}

func (bc *BitcoinClient) refreshPeers() {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	bc.healthyPeers = bc.manager.GetHealthyEndpoints(Bitcoin)
	bc.logger.Info("Refreshed Bitcoin peers",
		zap.Int("healthy_count", len(bc.healthyPeers)))
}

// setupConnectionPools creates HTTP connection pools
func (ec *EthereumClient) setupConnectionPools() {
	for _, endpoint := range ec.activeEndpoints {
		if endpoint.Type == RPC {
			key := fmt.Sprintf("%s:%s", endpoint.Network, endpoint.URL)
			ec.connectionPool[key] = ec.manager.httpClients[ec.manager.getEndpointKey(endpoint.Network, endpoint)]
		}
	}
}

func (sc *SolanaClient) setupConnectionPools() {
	for _, endpoint := range sc.activeEndpoints {
		if endpoint.Type == RPC {
			key := fmt.Sprintf("%s:%s", endpoint.Network, endpoint.URL)
			sc.connectionPool[key] = sc.manager.httpClients[sc.manager.getEndpointKey(endpoint.Network, endpoint)]
		}
	}
}

// getHTTPClient returns the HTTP client for an endpoint
func (ec *EthereumClient) getHTTPClient(endpoint *NodeEndpoint) *http.Client {
	key := fmt.Sprintf("%s:%s", endpoint.Network, endpoint.URL)
	if client, exists := ec.connectionPool[key]; exists {
		return client
	}
	
	// Return manager's client as fallback
	managerKey := ec.manager.getEndpointKey(endpoint.Network, endpoint)
	if client, exists := ec.manager.httpClients[managerKey]; exists {
		return client
	}
	
	// Create a basic client as last resort
	return &http.Client{Timeout: 30 * time.Second}
}

func (sc *SolanaClient) getHTTPClient(endpoint *NodeEndpoint) *http.Client {
	key := fmt.Sprintf("%s:%s", endpoint.Network, endpoint.URL)
	if client, exists := sc.connectionPool[key]; exists {
		return client
	}
	
	// Return manager's client as fallback
	managerKey := sc.manager.getEndpointKey(endpoint.Network, endpoint)
	if client, exists := sc.manager.httpClients[managerKey]; exists {
		return client
	}
	
	// Create a basic client as last resort
	return &http.Client{Timeout: 30 * time.Second}
}

// getBackupEndpoint returns a backup endpoint if primary fails
func (ec *EthereumClient) getBackupEndpoint(failed *NodeEndpoint) *NodeEndpoint {
	for _, endpoint := range ec.activeEndpoints {
		if endpoint != failed && endpoint.IsHealthy && endpoint.Type == failed.Type {
			return endpoint
		}
	}
	return nil
}

func (sc *SolanaClient) getBackupEndpoint(failed *NodeEndpoint) *NodeEndpoint {
	for _, endpoint := range sc.activeEndpoints {
		if endpoint != failed && endpoint.IsHealthy && endpoint.Type == failed.Type {
			return endpoint
		}
	}
	return nil
}

// monitorPerformance monitors endpoint performance
func (ec *EthereumClient) monitorPerformance(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ec.refreshEndpoints()
		}
	}
}

func (sc *SolanaClient) monitorPerformance(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sc.refreshEndpoints()
		}
	}
}

// cleanupCache removes expired cache entries
func (ec *EthereumClient) cleanupCache(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ec.cacheMutex.Lock()
			for key, cached := range ec.responseCache {
				if time.Since(cached.Timestamp) > cached.TTL {
					delete(ec.responseCache, key)
				}
			}
			ec.cacheMutex.Unlock()
		}
	}
}

func (sc *SolanaClient) cleanupCache(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sc.cacheMutex.Lock()
			for key, cached := range sc.responseCache {
				if time.Since(cached.Timestamp) > cached.TTL {
					delete(sc.responseCache, key)
				}
			}
			sc.cacheMutex.Unlock()
		}
	}
}

// Bitcoin-specific methods
func (bc *BitcoinClient) maintainP2PConnections(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			bc.refreshPeers()
			// Additional P2P connection maintenance would go here
		}
	}
}

func (bc *BitcoinClient) monitorConnectionHealth(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Monitor P2P connection health
			bc.mu.RLock()
			healthyCount := 0
			for _, conn := range bc.activeConnections {
				if conn.Connected && conn.Health.IsResponsive {
					healthyCount++
				}
			}
			bc.mu.RUnlock()
			
			bc.logger.Debug("Bitcoin P2P health check",
				zap.Int("healthy_connections", healthyCount),
				zap.Int("total_connections", len(bc.activeConnections)))
		}
	}
}

func (bc *BitcoinClient) requestBlockFromPeer(ctx context.Context, peer *BitcoinP2PConnection, blockHash string) ([]byte, error) {
	// Placeholder for actual P2P block request implementation
	// This would involve Bitcoin protocol messages
	return nil, fmt.Errorf("P2P block request not yet implemented")
}

func (bc *BitcoinClient) calculatePeerScore(conn *BitcoinP2PConnection) float64 {
	if !conn.Connected || !conn.Health.IsResponsive {
		return 0.0
	}
	
	// Calculate score based on various factors
	latencyScore := 1.0
	if conn.Health.Latency > 0 {
		latencyScore = 1.0 / (float64(conn.Health.Latency.Milliseconds()) / 100.0)
		if latencyScore > 1.0 {
			latencyScore = 1.0
		}
	}
	
	errorRate := 0.0
	if conn.Health.MessageCount > 0 {
		errorRate = float64(conn.Health.ErrorCount) / float64(conn.Health.MessageCount)
	}
	
	reliabilityScore := 1.0 - errorRate
	if reliabilityScore < 0 {
		reliabilityScore = 0
	}
	
	return (latencyScore * 0.6) + (reliabilityScore * 0.4)
}
