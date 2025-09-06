package network

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// NetworkType represents the blockchain network
type NetworkType string

const (
	Bitcoin  NetworkType = "bitcoin"
	Ethereum NetworkType = "ethereum"
	Solana   NetworkType = "solana"
)

// ConnectionType represents the connection method
type ConnectionType string

const (
	P2P       ConnectionType = "p2p"
	RPC       ConnectionType = "rpc"
	WebSocket ConnectionType = "websocket"
)

// NodeEndpoint represents a network endpoint
type NodeEndpoint struct {
	URL            string         `json:"url"`
	Type           ConnectionType `json:"type"`
	Network        NetworkType    `json:"network"`
	Priority       int            `json:"priority"`
	IsHealthy      bool           `json:"is_healthy"`
	ResponseTime   time.Duration  `json:"response_time"`
	LastChecked    time.Time      `json:"last_checked"`
	FailureCount   int64          `json:"failure_count"`
	SuccessCount   int64          `json:"success_count"`
	Region         string         `json:"region"`
	Provider       string         `json:"provider"`
	CustomHeaders  map[string]string `json:"custom_headers,omitempty"`
}

// NetworkManager manages all blockchain network connections
type NetworkManager struct {
	logger    *zap.Logger
	mu        sync.RWMutex
	endpoints map[NetworkType][]*NodeEndpoint
	
	// Connection pools
	httpClients    map[string]*http.Client
	wsConnections  map[string]interface{} // WebSocket connections
	p2pConnections map[string]interface{} // P2P connections
	
	// Health monitoring
	healthChecker  *HealthChecker
	rateLimiters   map[string]*rate.Limiter
	
	// Performance tracking
	responseMetrics map[string]*ResponseMetrics
	
	// Configuration
	config *NetworkConfig
	
	// Control channels
	stopChan    chan struct{}
	healthChan  chan *HealthUpdate
	metricsChan chan *MetricsUpdate
}

// NetworkConfig holds configuration for network connections
type NetworkConfig struct {
	// Connection settings
	MaxConnsPerHost       int           `json:"max_conns_per_host"`
	MaxIdleConns          int           `json:"max_idle_conns"`
	IdleConnTimeout       time.Duration `json:"idle_conn_timeout"`
	ResponseHeaderTimeout time.Duration `json:"response_header_timeout"`
	TLSHandshakeTimeout   time.Duration `json:"tls_handshake_timeout"`
	
	// Health check settings
	HealthCheckInterval   time.Duration `json:"health_check_interval"`
	HealthCheckTimeout    time.Duration `json:"health_check_timeout"`
	MaxFailuresBeforeDown int           `json:"max_failures_before_down"`
	
	// Performance settings
	RateLimitRPS          int           `json:"rate_limit_rps"`
	RateLimitBurst        int           `json:"rate_limit_burst"`
	FastestResponseWeight float64       `json:"fastest_response_weight"`
	
	// Retry settings
	MaxRetries         int           `json:"max_retries"`
	RetryDelay         time.Duration `json:"retry_delay"`
	RetryBackoffFactor float64       `json:"retry_backoff_factor"`
	
	// Security settings
	EnableTLSVerification bool `json:"enable_tls_verification"`
	CustomCA              string `json:"custom_ca,omitempty"`
}

// ResponseMetrics tracks response performance
type ResponseMetrics struct {
	TotalRequests    int64         `json:"total_requests"`
	SuccessfulReqs   int64         `json:"successful_requests"`
	FailedReqs       int64         `json:"failed_requests"`
	AvgResponseTime  time.Duration `json:"avg_response_time"`
	MinResponseTime  time.Duration `json:"min_response_time"`
	MaxResponseTime  time.Duration `json:"max_response_time"`
	LastResponseTime time.Duration `json:"last_response_time"`
	Uptime           float64       `json:"uptime_percentage"`
	ErrorRate        float64       `json:"error_rate"`
}

// HealthUpdate represents a health status change
type HealthUpdate struct {
	Endpoint   *NodeEndpoint `json:"endpoint"`
	IsHealthy  bool          `json:"is_healthy"`
	Error      error         `json:"error,omitempty"`
	Timestamp  time.Time     `json:"timestamp"`
}

// MetricsUpdate represents a metrics update
type MetricsUpdate struct {
	Endpoint    *NodeEndpoint     `json:"endpoint"`
	Metrics     *ResponseMetrics  `json:"metrics"`
	RequestTime time.Duration     `json:"request_time"`
	Success     bool              `json:"success"`
	Timestamp   time.Time         `json:"timestamp"`
}

// NewNetworkManager creates a new network manager
func NewNetworkManager(config *NetworkConfig, logger *zap.Logger) *NetworkManager {
	if config == nil {
		config = DefaultNetworkConfig()
	}
	
	nm := &NetworkManager{
		logger:          logger,
		endpoints:       make(map[NetworkType][]*NodeEndpoint),
		httpClients:     make(map[string]*http.Client),
		wsConnections:   make(map[string]interface{}),
		p2pConnections:  make(map[string]interface{}),
		rateLimiters:    make(map[string]*rate.Limiter),
		responseMetrics: make(map[string]*ResponseMetrics),
		config:          config,
		stopChan:        make(chan struct{}),
		healthChan:      make(chan *HealthUpdate, 100),
		metricsChan:     make(chan *MetricsUpdate, 100),
	}
	
	nm.healthChecker = NewHealthChecker(nm, logger)
	nm.initializeEndpoints()
	nm.setupHTTPClients()
	
	return nm
}

// DefaultNetworkConfig returns default configuration
func DefaultNetworkConfig() *NetworkConfig {
	return &NetworkConfig{
		MaxConnsPerHost:       100,
		MaxIdleConns:          50,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		HealthCheckInterval:   30 * time.Second,
		HealthCheckTimeout:    10 * time.Second,
		MaxFailuresBeforeDown: 3,
		RateLimitRPS:          1000,
		RateLimitBurst:        100,
		FastestResponseWeight: 0.7,
		MaxRetries:            3,
		RetryDelay:            1 * time.Second,
		RetryBackoffFactor:    2.0,
		EnableTLSVerification: true,
	}
}

// initializeEndpoints sets up direct node endpoints for all networks
func (nm *NetworkManager) initializeEndpoints() {
	// Bitcoin direct node endpoints (bypassing third parties)
	bitcoinEndpoints := []*NodeEndpoint{
		{URL: "seed.bitcoin.sipa.be:8333", Type: P2P, Network: Bitcoin, Priority: 1, Provider: "Pieter Wuille", Region: "Global"},
		{URL: "seed.bitcoinstats.com:8333", Type: P2P, Network: Bitcoin, Priority: 1, Provider: "BitcoinStats", Region: "Global"},
		{URL: "dnsseed.bitcoin.dashjr.org:8333", Type: P2P, Network: Bitcoin, Priority: 1, Provider: "Luke Dashjr", Region: "Global"},
		{URL: "dnsseed.emzy.de:8333", Type: P2P, Network: Bitcoin, Priority: 1, Provider: "Emzy", Region: "Europe"},
		{URL: "seed.bitcoin.jonasschnelli.ch:8333", Type: P2P, Network: Bitcoin, Priority: 1, Provider: "Jonas Schnelli", Region: "Europe"},
		{URL: "seed.bitnodes.io:8333", Type: P2P, Network: Bitcoin, Priority: 1, Provider: "Bitnodes", Region: "Global"},
		{URL: "dnsseed.bluematt.me:8333", Type: P2P, Network: Bitcoin, Priority: 1, Provider: "BlueMatt", Region: "Global"},
		{URL: "seed.btc.petertodd.org:8333", Type: P2P, Network: Bitcoin, Priority: 2, Provider: "Peter Todd", Region: "Global"},
		// Additional high-performance Bitcoin nodes
		{URL: "btc-node1.coinbase.com:8333", Type: P2P, Network: Bitcoin, Priority: 3, Provider: "Coinbase", Region: "US"},
		{URL: "btc-node2.coinbase.com:8333", Type: P2P, Network: Bitcoin, Priority: 3, Provider: "Coinbase", Region: "US"},
	}
	
	// Ethereum direct node endpoints (bypassing Infura/Alchemy)
	ethereumEndpoints := []*NodeEndpoint{
		// Major validator/infrastructure providers running full nodes
		{URL: "ethereum-mainnet.public.blastapi.io", Type: RPC, Network: Ethereum, Priority: 1, Provider: "Blast API", Region: "Global"},
		{URL: "rpc.ankr.com/eth", Type: RPC, Network: Ethereum, Priority: 1, Provider: "Ankr", Region: "Global"},
		{URL: "eth.public-rpc.com", Type: RPC, Network: Ethereum, Priority: 1, Provider: "Public RPC", Region: "Global"},
		{URL: "ethereum.blockpi.network/v1/rpc/public", Type: RPC, Network: Ethereum, Priority: 1, Provider: "BlockPI", Region: "Global"},
		{URL: "eth-mainnet.nodereal.io/v1/1659dfb40aa24bbb8153a677b98064d7", Type: RPC, Network: Ethereum, Priority: 2, Provider: "NodeReal", Region: "Asia"},
		{URL: "rpc.flashbots.net", Type: RPC, Network: Ethereum, Priority: 2, Provider: "Flashbots", Region: "Global"},
		{URL: "eth-mainnet.gateway.pokt.network/v1/5f3453978e354ab992842753", Type: RPC, Network: Ethereum, Priority: 2, Provider: "Pocket Network", Region: "Global"},
		{URL: "mainnet.eth.cloud.ava.do", Type: RPC, Network: Ethereum, Priority: 3, Provider: "AVADO", Region: "Global"},
		// WebSocket connections for real-time data
		{URL: "wss://ethereum.blockpi.network/v1/ws/public", Type: WebSocket, Network: Ethereum, Priority: 1, Provider: "BlockPI WS", Region: "Global"},
		{URL: "wss://eth-mainnet.nodereal.io/ws/v1/1659dfb40aa24bbb8153a677b98064d7", Type: WebSocket, Network: Ethereum, Priority: 2, Provider: "NodeReal WS", Region: "Asia"},
	}
	
	// Solana direct RPC endpoints (bypassing third parties)
	solanaEndpoints := []*NodeEndpoint{
		// High-performance Solana RPC providers
		{URL: "https://solana-mainnet.public.blastapi.io", Type: RPC, Network: Solana, Priority: 1, Provider: "Blast API", Region: "Global"},
		{URL: "https://rpc.ankr.com/solana", Type: RPC, Network: Solana, Priority: 1, Provider: "Ankr", Region: "Global"},
		{URL: "https://solana.blockpi.network/v1/rpc/public", Type: RPC, Network: Solana, Priority: 1, Provider: "BlockPI", Region: "Global"},
		{URL: "https://solana-mainnet.gateway.pokt.network/v1/5f3453978e354ab992842753", Type: RPC, Network: Solana, Priority: 2, Provider: "Pocket Network", Region: "Global"},
		{URL: "https://api.mainnet-beta.solana.com", Type: RPC, Network: Solana, Priority: 2, Provider: "Solana Labs", Region: "Global"},
		{URL: "https://solana-api.projectserum.com", Type: RPC, Network: Solana, Priority: 3, Provider: "Serum", Region: "Global"},
		{URL: "https://ssc-dao.genesysgo.net", Type: RPC, Network: Solana, Priority: 3, Provider: "GenesysGo", Region: "Global"},
		// WebSocket connections for real-time updates
		{URL: "wss://solana.blockpi.network/v1/ws/public", Type: WebSocket, Network: Solana, Priority: 1, Provider: "BlockPI WS", Region: "Global"},
		{URL: "wss://api.mainnet-beta.solana.com", Type: WebSocket, Network: Solana, Priority: 2, Provider: "Solana Labs WS", Region: "Global"},
	}
	
	nm.endpoints[Bitcoin] = bitcoinEndpoints
	nm.endpoints[Ethereum] = ethereumEndpoints
	nm.endpoints[Solana] = solanaEndpoints
	
	// Initialize metrics for all endpoints
	for network, endpoints := range nm.endpoints {
		for _, endpoint := range endpoints {
			key := nm.getEndpointKey(network, endpoint)
			nm.responseMetrics[key] = &ResponseMetrics{
				MinResponseTime: time.Hour, // Start with high value
			}
			nm.rateLimiters[key] = rate.NewLimiter(rate.Limit(nm.config.RateLimitRPS), nm.config.RateLimitBurst)
		}
	}
}

// setupHTTPClients creates optimized HTTP clients for each endpoint
func (nm *NetworkManager) setupHTTPClients() {
	for network, endpoints := range nm.endpoints {
		for _, endpoint := range endpoints {
			if endpoint.Type == RPC {
				key := nm.getEndpointKey(network, endpoint)
				
				// Create custom transport for maximum performance
				transport := &http.Transport{
					DialContext: (&net.Dialer{
						Timeout:   5 * time.Second,
						KeepAlive: 30 * time.Second,
					}).DialContext,
					MaxIdleConns:          nm.config.MaxIdleConns,
					MaxIdleConnsPerHost:   nm.config.MaxConnsPerHost,
					IdleConnTimeout:       nm.config.IdleConnTimeout,
					TLSHandshakeTimeout:   nm.config.TLSHandshakeTimeout,
					ResponseHeaderTimeout: nm.config.ResponseHeaderTimeout,
					DisableCompression:    false,
					ForceAttemptHTTP2:     true,
				}
				
				// Configure TLS
				if !nm.config.EnableTLSVerification {
					transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
				}
				
				// Create HTTP client
				client := &http.Client{
					Transport: transport,
					Timeout:   nm.config.ResponseHeaderTimeout,
				}
				
				nm.httpClients[key] = client
			}
		}
	}
}

// Start begins the network manager operations
func (nm *NetworkManager) Start(ctx context.Context) error {
	nm.logger.Info("Starting Network Manager",
		zap.Int("bitcoin_endpoints", len(nm.endpoints[Bitcoin])),
		zap.Int("ethereum_endpoints", len(nm.endpoints[Ethereum])),
		zap.Int("solana_endpoints", len(nm.endpoints[Solana])))
	
	// Start health checker
	go nm.healthChecker.Start(ctx)
	
	// Start metrics processor
	go nm.processMetrics(ctx)
	
	// Start health update processor
	go nm.processHealthUpdates(ctx)
	
	return nil
}

// GetBestEndpoint returns the best available endpoint for a network
func (nm *NetworkManager) GetBestEndpoint(network NetworkType, connType ConnectionType) (*NodeEndpoint, error) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	
	endpoints := nm.endpoints[network]
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints configured for network %s", network)
	}
	
	// Filter by connection type and health
	var candidates []*NodeEndpoint
	for _, endpoint := range endpoints {
		if endpoint.Type == connType && endpoint.IsHealthy {
			candidates = append(candidates, endpoint)
		}
	}
	
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no healthy endpoints available for network %s type %s", network, connType)
	}
	
	// Sort by performance and priority
	best := nm.selectBestEndpoint(candidates)
	return best, nil
}

// selectBestEndpoint uses sophisticated algorithm to pick the best endpoint
func (nm *NetworkManager) selectBestEndpoint(candidates []*NodeEndpoint) *NodeEndpoint {
	if len(candidates) == 1 {
		return candidates[0]
	}
	
	var best *NodeEndpoint
	bestScore := float64(-1)
	
	for _, endpoint := range candidates {
		score := nm.calculateEndpointScore(endpoint)
		if score > bestScore {
			bestScore = score
			best = endpoint
		}
	}
	
	return best
}

// calculateEndpointScore calculates a comprehensive score for an endpoint
func (nm *NetworkManager) calculateEndpointScore(endpoint *NodeEndpoint) float64 {
	key := nm.getEndpointKey(endpoint.Network, endpoint)
	metrics := nm.responseMetrics[key]
	
	// Base score from priority (higher priority = higher score)
	priorityScore := float64(10-endpoint.Priority) / 10.0
	
	// Performance score (faster = higher score)
	var performanceScore float64
	if metrics.AvgResponseTime > 0 {
		// Convert response time to score (faster = higher)
		performanceScore = 1.0 / (float64(metrics.AvgResponseTime.Milliseconds()) / 1000.0)
		if performanceScore > 1.0 {
			performanceScore = 1.0
		}
	} else {
		performanceScore = 0.5 // Default for new endpoints
	}
	
	// Reliability score
	reliabilityScore := 1.0 - metrics.ErrorRate
	if reliabilityScore < 0 {
		reliabilityScore = 0
	}
	
	// Uptime score
	uptimeScore := metrics.Uptime / 100.0
	
	// Recent performance weight
	recentSuccess := float64(1.0)
	if endpoint.FailureCount > 0 {
		recentSuccess = float64(endpoint.SuccessCount) / float64(endpoint.SuccessCount+endpoint.FailureCount)
	}
	
	// Combine scores with weights
	finalScore := (priorityScore * 0.2) +
		(performanceScore * nm.config.FastestResponseWeight) +
		(reliabilityScore * 0.15) +
		(uptimeScore * 0.1) +
		(recentSuccess * 0.05)
	
	return finalScore
}

// GetHealthyEndpoints returns all healthy endpoints for a network
func (nm *NetworkManager) GetHealthyEndpoints(network NetworkType) []*NodeEndpoint {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	
	var healthy []*NodeEndpoint
	for _, endpoint := range nm.endpoints[network] {
		if endpoint.IsHealthy {
			healthy = append(healthy, endpoint)
		}
	}
	
	return healthy
}

// GetMetrics returns performance metrics for all endpoints
func (nm *NetworkManager) GetMetrics() map[string]*ResponseMetrics {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	
	// Deep copy metrics
	result := make(map[string]*ResponseMetrics)
	for key, metrics := range nm.responseMetrics {
		result[key] = &ResponseMetrics{
			TotalRequests:    metrics.TotalRequests,
			SuccessfulReqs:   metrics.SuccessfulReqs,
			FailedReqs:       metrics.FailedReqs,
			AvgResponseTime:  metrics.AvgResponseTime,
			MinResponseTime:  metrics.MinResponseTime,
			MaxResponseTime:  metrics.MaxResponseTime,
			LastResponseTime: metrics.LastResponseTime,
			Uptime:           metrics.Uptime,
			ErrorRate:        metrics.ErrorRate,
		}
	}
	
	return result
}

// processMetrics handles metrics updates
func (nm *NetworkManager) processMetrics(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-nm.stopChan:
			return
		case update := <-nm.metricsChan:
			nm.updateMetrics(update)
		}
	}
}

// processHealthUpdates handles health status changes
func (nm *NetworkManager) processHealthUpdates(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-nm.stopChan:
			return
		case update := <-nm.healthChan:
			nm.updateHealth(update)
		}
	}
}

// updateMetrics updates endpoint metrics
func (nm *NetworkManager) updateMetrics(update *MetricsUpdate) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	
	key := nm.getEndpointKey(update.Endpoint.Network, update.Endpoint)
	metrics := nm.responseMetrics[key]
	
	atomic.AddInt64(&metrics.TotalRequests, 1)
	
	if update.Success {
		atomic.AddInt64(&metrics.SuccessfulReqs, 1)
		atomic.AddInt64(&update.Endpoint.SuccessCount, 1)
		
		// Update response times
		metrics.LastResponseTime = update.RequestTime
		if update.RequestTime < metrics.MinResponseTime || metrics.MinResponseTime == 0 {
			metrics.MinResponseTime = update.RequestTime
		}
		if update.RequestTime > metrics.MaxResponseTime {
			metrics.MaxResponseTime = update.RequestTime
		}
		
		// Update average (weighted)
		if metrics.AvgResponseTime == 0 {
			metrics.AvgResponseTime = update.RequestTime
		} else {
			metrics.AvgResponseTime = time.Duration(
				(int64(metrics.AvgResponseTime)*9 + int64(update.RequestTime)) / 10,
			)
		}
	} else {
		atomic.AddInt64(&metrics.FailedReqs, 1)
		atomic.AddInt64(&update.Endpoint.FailureCount, 1)
	}
	
	// Recalculate derived metrics
	total := float64(metrics.TotalRequests)
	if total > 0 {
		metrics.ErrorRate = float64(metrics.FailedReqs) / total
		metrics.Uptime = (float64(metrics.SuccessfulReqs) / total) * 100.0
	}
}

// updateHealth updates endpoint health status
func (nm *NetworkManager) updateHealth(update *HealthUpdate) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	
	update.Endpoint.IsHealthy = update.IsHealthy
	update.Endpoint.LastChecked = update.Timestamp
	
	if !update.IsHealthy {
		nm.logger.Warn("Endpoint marked unhealthy",
			zap.String("network", string(update.Endpoint.Network)),
			zap.String("url", update.Endpoint.URL),
			zap.Error(update.Error))
	} else {
		nm.logger.Info("Endpoint marked healthy",
			zap.String("network", string(update.Endpoint.Network)),
			zap.String("url", update.Endpoint.URL))
	}
}

// getEndpointKey generates a unique key for an endpoint
func (nm *NetworkManager) getEndpointKey(network NetworkType, endpoint *NodeEndpoint) string {
	return fmt.Sprintf("%s:%s:%s", network, endpoint.Type, endpoint.URL)
}

// Stop gracefully shuts down the network manager
func (nm *NetworkManager) Stop() error {
	nm.logger.Info("Stopping Network Manager")
	close(nm.stopChan)
	
	// Close all HTTP clients
	for _, client := range nm.httpClients {
		if transport, ok := client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}
	
	return nil
}
