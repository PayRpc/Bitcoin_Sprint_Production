package network

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// HealthChecker monitors endpoint health continuously
type HealthChecker struct {
	manager *NetworkManager
	logger  *zap.Logger
	
	// HTTP clients for health checks
	healthClients map[string]*http.Client
	
	// Control
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(manager *NetworkManager, logger *zap.Logger) *HealthChecker {
	return &HealthChecker{
		manager:       manager,
		logger:        logger,
		healthClients: make(map[string]*http.Client),
		stopChan:      make(chan struct{}),
	}
}

// Start begins health checking for all endpoints
func (hc *HealthChecker) Start(ctx context.Context) {
	hc.logger.Info("Starting health checker")
	
	// Create optimized health check clients
	hc.setupHealthClients()
	
	// Start health checks for each network
	for network := range hc.manager.endpoints {
		hc.wg.Add(1)
		go hc.monitorNetwork(ctx, network)
	}
	
	// Wait for shutdown
	<-ctx.Done()
	hc.Stop()
}

// setupHealthClients creates lightweight HTTP clients for health checks
func (hc *HealthChecker) setupHealthClients() {
	// Fast health check client with minimal timeout
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 10 * time.Second,
		}).DialContext,
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   5,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		DisableCompression:    true,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	
	client := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}
	
	hc.healthClients["default"] = client
}

// monitorNetwork monitors all endpoints for a specific network
func (hc *HealthChecker) monitorNetwork(ctx context.Context, network NetworkType) {
	defer hc.wg.Done()
	
	ticker := time.NewTicker(hc.manager.config.HealthCheckInterval)
	defer ticker.Stop()
	
	// Initial health check
	hc.checkNetworkHealth(network)
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-hc.stopChan:
			return
		case <-ticker.C:
			hc.checkNetworkHealth(network)
		}
	}
}

// checkNetworkHealth performs health checks on all endpoints for a network
func (hc *HealthChecker) checkNetworkHealth(network NetworkType) {
	endpoints := hc.manager.endpoints[network]
	
	// Check endpoints in parallel
	var wg sync.WaitGroup
	for _, endpoint := range endpoints {
		wg.Add(1)
		go func(ep *NodeEndpoint) {
			defer wg.Done()
			hc.checkEndpointHealth(ep)
		}(endpoint)
	}
	wg.Wait()
}

// checkEndpointHealth performs a health check on a single endpoint
func (hc *HealthChecker) checkEndpointHealth(endpoint *NodeEndpoint) {
	start := time.Now()
	isHealthy := false
	var err error
	
	switch endpoint.Type {
	case RPC:
		isHealthy, err = hc.checkRPCHealth(endpoint)
	case WebSocket:
		isHealthy, err = hc.checkWebSocketHealth(endpoint)
	case P2P:
		isHealthy, err = hc.checkP2PHealth(endpoint)
	}
	
	responseTime := time.Since(start)
	endpoint.ResponseTime = responseTime
	
	// Determine final health status
	if isHealthy {
		endpoint.FailureCount = 0
	} else {
		endpoint.FailureCount++
	}
	
	// Mark as unhealthy if too many failures
	finalHealth := isHealthy && endpoint.FailureCount < int64(hc.manager.config.MaxFailuresBeforeDown)
	
	// Send health update
	hc.manager.healthChan <- &HealthUpdate{
		Endpoint:  endpoint,
		IsHealthy: finalHealth,
		Error:     err,
		Timestamp: time.Now(),
	}
	
	// Send metrics update
	hc.manager.metricsChan <- &MetricsUpdate{
		Endpoint:    endpoint,
		RequestTime: responseTime,
		Success:     isHealthy,
		Timestamp:   time.Now(),
	}
}

// checkRPCHealth checks HTTP RPC endpoint health
func (hc *HealthChecker) checkRPCHealth(endpoint *NodeEndpoint) (bool, error) {
	client := hc.healthClients["default"]
	
	var healthURL string
	switch endpoint.Network {
	case Bitcoin:
		// Bitcoin RPC health check (if available)
		return hc.checkBitcoinRPCHealth(endpoint)
	case Ethereum:
		healthURL = "https://" + endpoint.URL
		return hc.checkEthereumRPCHealth(healthURL, client)
	case Solana:
		healthURL = endpoint.URL
		return hc.checkSolanaRPCHealth(healthURL, client)
	}
	
	return false, nil
}

// checkBitcoinRPCHealth checks Bitcoin RPC health
func (hc *HealthChecker) checkBitcoinRPCHealth(endpoint *NodeEndpoint) (bool, error) {
	// For Bitcoin, we mainly use P2P connections
	// RPC health can be checked if credentials are available
	return true, nil // Assume healthy for P2P-focused Bitcoin
}

// checkEthereumRPCHealth checks Ethereum RPC health
func (hc *HealthChecker) checkEthereumRPCHealth(url string, client *http.Client) (bool, error) {
	// Create a simple eth_blockNumber request
	payload := `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`
	
	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return false, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == 200, nil
}

// checkSolanaRPCHealth checks Solana RPC health
func (hc *HealthChecker) checkSolanaRPCHealth(url string, client *http.Client) (bool, error) {
	// Create a simple getHealth request
	payload := `{"jsonrpc":"2.0","id":1,"method":"getHealth"}`
	
	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return false, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == 200, nil
}

// checkWebSocketHealth checks WebSocket endpoint health
func (hc *HealthChecker) checkWebSocketHealth(endpoint *NodeEndpoint) (bool, error) {
	// For WebSocket, we can try a simple connection test
	conn, err := net.DialTimeout("tcp", endpoint.URL, 5*time.Second)
	if err != nil {
		return false, err
	}
	conn.Close()
	return true, nil
}

// checkP2PHealth checks P2P endpoint health
func (hc *HealthChecker) checkP2PHealth(endpoint *NodeEndpoint) (bool, error) {
	// For P2P, check if we can establish a TCP connection
	conn, err := net.DialTimeout("tcp", endpoint.URL, 5*time.Second)
	if err != nil {
		return false, err
	}
	conn.Close()
	return true, nil
}

// Stop gracefully shuts down the health checker
func (hc *HealthChecker) Stop() {
	hc.logger.Info("Stopping health checker")
	close(hc.stopChan)
	hc.wg.Wait()
	
	// Close health check clients
	for _, client := range hc.healthClients {
		if transport, ok := client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}
}
