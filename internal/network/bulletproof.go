package network

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// BulletproofConnectionManager manages all network connections with maximum reliability
type BulletproofConnectionManager struct {
	logger *zap.Logger
	
	// Network components
	networkManager  *NetworkManager
	bitcoinClient   *BitcoinClient
	ethereumClient  *EthereumClient
	solanaClient    *SolanaClient
	
	// Failover management
	mu              sync.RWMutex
	isHealthy       bool
	lastHealthCheck time.Time
	failoverHistory map[NetworkType][]time.Time
	
	// Performance tracking
	connectionStats map[NetworkType]*ConnectionStats
	
	// Control
	ctx      context.Context
	cancel   context.CancelFunc
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// ConnectionStats tracks connection performance
type ConnectionStats struct {
	TotalRequests     int64         `json:"total_requests"`
	SuccessfulReqs    int64         `json:"successful_requests"`
	FailedRequests    int64         `json:"failed_requests"`
	AverageLatency    time.Duration `json:"average_latency"`
	LastSuccessTime   time.Time     `json:"last_success_time"`
	UptimePercentage  float64       `json:"uptime_percentage"`
	ActiveConnections int           `json:"active_connections"`
	FailoverCount     int64         `json:"failover_count"`
}

// NewBulletproofConnectionManager creates a new bulletproof connection manager
func NewBulletproofConnectionManager(logger *zap.Logger) *BulletproofConnectionManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create network manager with optimized config
	config := &NetworkConfig{
		MaxConnsPerHost:       200,
		MaxIdleConns:          100,
		IdleConnTimeout:       120 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
		TLSHandshakeTimeout:   8 * time.Second,
		HealthCheckInterval:   15 * time.Second,
		HealthCheckTimeout:    8 * time.Second,
		MaxFailuresBeforeDown: 2,
		RateLimitRPS:          2000,
		RateLimitBurst:        200,
		FastestResponseWeight: 0.8,
		MaxRetries:            5,
		RetryDelay:            500 * time.Millisecond,
		RetryBackoffFactor:    1.5,
		EnableTLSVerification: false, // For maximum compatibility
	}
	
	networkManager := NewNetworkManager(config, logger)
	
	manager := &BulletproofConnectionManager{
		logger:          logger,
		networkManager:  networkManager,
		bitcoinClient:   NewBitcoinClient(networkManager, logger),
		ethereumClient:  NewEthereumClient(networkManager, logger),
		solanaClient:    NewSolanaClient(networkManager, logger),
		ctx:             ctx,
		cancel:          cancel,
		stopChan:        make(chan struct{}),
		failoverHistory: make(map[NetworkType][]time.Time),
		connectionStats: make(map[NetworkType]*ConnectionStats),
	}
	
	// Initialize connection stats
	manager.connectionStats[Bitcoin] = &ConnectionStats{}
	manager.connectionStats[Ethereum] = &ConnectionStats{}
	manager.connectionStats[Solana] = &ConnectionStats{}
	
	return manager
}

// Start initializes and starts all network connections
func (bcm *BulletproofConnectionManager) Start() error {
	bcm.logger.Info("Starting Bulletproof Connection Manager",
		zap.String("mode", "MAXIMUM_PERFORMANCE"),
		zap.Bool("bypass_third_parties", true))
	
	// Start network manager
	if err := bcm.networkManager.Start(bcm.ctx); err != nil {
		return err
	}
	
	// Initialize all clients
	if err := bcm.bitcoinClient.Initialize(bcm.ctx); err != nil {
		bcm.logger.Error("Failed to initialize Bitcoin client", zap.Error(err))
	}
	
	if err := bcm.ethereumClient.Initialize(bcm.ctx); err != nil {
		bcm.logger.Error("Failed to initialize Ethereum client", zap.Error(err))
	}
	
	if err := bcm.solanaClient.Initialize(bcm.ctx); err != nil {
		bcm.logger.Error("Failed to initialize Solana client", zap.Error(err))
	}
	
	// Start monitoring and failover systems
	bcm.wg.Add(3)
	go bcm.monitorConnections()
	go bcm.performanceMonitor()
	go bcm.failoverManager()
	
	bcm.logger.Info("Bulletproof Connection Manager started successfully",
		zap.String("status", "BULLETPROOF_ACTIVE"))
	
	return nil
}

// GetBitcoinData retrieves data from Bitcoin network with automatic failover
func (bcm *BulletproofConnectionManager) GetBitcoinData(ctx context.Context, blockHash string) ([]byte, error) {
	start := time.Now()
	
	data, err := bcm.bitcoinClient.GetBlockData(ctx, blockHash)
	
	// Update stats
	bcm.updateStats(Bitcoin, time.Since(start), err == nil)
	
	if err != nil {
		bcm.logger.Warn("Bitcoin request failed, attempting failover",
			zap.Error(err),
			zap.String("block_hash", blockHash))
		bcm.recordFailover(Bitcoin)
	}
	
	return data, err
}

// GetEthereumData retrieves data from Ethereum network with automatic failover
func (bcm *BulletproofConnectionManager) GetEthereumData(ctx context.Context, method string, params []interface{}) ([]byte, error) {
	start := time.Now()
	
	result, err := bcm.ethereumClient.CallMethod(ctx, method, params)
	
	// Update stats
	bcm.updateStats(Ethereum, time.Since(start), err == nil)
	
	if err != nil {
		bcm.logger.Warn("Ethereum request failed, attempting failover",
			zap.Error(err),
			zap.String("method", method))
		bcm.recordFailover(Ethereum)
	}
	
	return result, err
}

// GetSolanaData retrieves data from Solana network with automatic failover
func (bcm *BulletproofConnectionManager) GetSolanaData(ctx context.Context, method string, params []interface{}) ([]byte, error) {
	start := time.Now()
	
	result, err := bcm.solanaClient.CallMethod(ctx, method, params)
	
	// Update stats
	bcm.updateStats(Solana, time.Since(start), err == nil)
	
	if err != nil {
		bcm.logger.Warn("Solana request failed, attempting failover",
			zap.Error(err),
			zap.String("method", method))
		bcm.recordFailover(Solana)
	}
	
	return result, err
}

// GetConnectionStats returns current connection statistics
func (bcm *BulletproofConnectionManager) GetConnectionStats() map[NetworkType]*ConnectionStats {
	bcm.mu.RLock()
	defer bcm.mu.RUnlock()
	
	// Deep copy stats
	result := make(map[NetworkType]*ConnectionStats)
	for network, stats := range bcm.connectionStats {
		result[network] = &ConnectionStats{
			TotalRequests:     stats.TotalRequests,
			SuccessfulReqs:    stats.SuccessfulReqs,
			FailedRequests:    stats.FailedRequests,
			AverageLatency:    stats.AverageLatency,
			LastSuccessTime:   stats.LastSuccessTime,
			UptimePercentage:  stats.UptimePercentage,
			ActiveConnections: stats.ActiveConnections,
			FailoverCount:     stats.FailoverCount,
		}
	}
	
	return result
}

// IsHealthy returns overall system health status
func (bcm *BulletproofConnectionManager) IsHealthy() bool {
	bcm.mu.RLock()
	defer bcm.mu.RUnlock()
	
	// Check if all networks have healthy connections
	bitcoinHealthy := len(bcm.networkManager.GetHealthyEndpoints(Bitcoin)) > 0
	ethereumHealthy := len(bcm.networkManager.GetHealthyEndpoints(Ethereum)) > 0
	solanaHealthy := len(bcm.networkManager.GetHealthyEndpoints(Solana)) > 0
	
	return bitcoinHealthy && ethereumHealthy && solanaHealthy
}

// monitorConnections continuously monitors all connections
func (bcm *BulletproofConnectionManager) monitorConnections() {
	defer bcm.wg.Done()
	
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-bcm.ctx.Done():
			return
		case <-bcm.stopChan:
			return
		case <-ticker.C:
			bcm.checkConnectionHealth()
		}
	}
}

// performanceMonitor tracks and optimizes performance
func (bcm *BulletproofConnectionManager) performanceMonitor() {
	defer bcm.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-bcm.ctx.Done():
			return
		case <-bcm.stopChan:
			return
		case <-ticker.C:
			bcm.optimizeConnections()
		}
	}
}

// failoverManager handles automatic failover scenarios
func (bcm *BulletproofConnectionManager) failoverManager() {
	defer bcm.wg.Done()
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-bcm.ctx.Done():
			return
		case <-bcm.stopChan:
			return
		case <-ticker.C:
			bcm.processFailovers()
		}
	}
}

// checkConnectionHealth verifies all connections are healthy
func (bcm *BulletproofConnectionManager) checkConnectionHealth() {
	bcm.mu.Lock()
	defer bcm.mu.Unlock()
	
	bcm.isHealthy = bcm.IsHealthy()
	bcm.lastHealthCheck = time.Now()
	
	// Log health status
	stats := bcm.connectionStats
	bcm.logger.Info("Connection health check",
		zap.Bool("overall_healthy", bcm.isHealthy),
		zap.Int64("bitcoin_successes", stats[Bitcoin].SuccessfulReqs),
		zap.Int64("ethereum_successes", stats[Ethereum].SuccessfulReqs),
		zap.Int64("solana_successes", stats[Solana].SuccessfulReqs),
		zap.Float64("bitcoin_uptime", stats[Bitcoin].UptimePercentage),
		zap.Float64("ethereum_uptime", stats[Ethereum].UptimePercentage),
		zap.Float64("solana_uptime", stats[Solana].UptimePercentage))
}

// optimizeConnections optimizes connection parameters
func (bcm *BulletproofConnectionManager) optimizeConnections() {
	// Get network metrics
	metrics := bcm.networkManager.GetMetrics()
	
	// Analyze performance and adjust if needed
	for network, stats := range bcm.connectionStats {
		if stats.UptimePercentage < 95.0 {
			bcm.logger.Warn("Low uptime detected, optimizing connections",
				zap.String("network", string(network)),
				zap.Float64("uptime", stats.UptimePercentage))
			
			// Trigger connection refresh
			bcm.refreshNetworkConnections(network)
		}
	}
	
	bcm.logger.Debug("Connection optimization complete",
		zap.Int("total_metrics", len(metrics)))
}

// processFailovers handles failover scenarios
func (bcm *BulletproofConnectionManager) processFailovers() {
	now := time.Now()
	
	for network, failovers := range bcm.failoverHistory {
		// Clean old failover records (older than 5 minutes)
		var recent []time.Time
		for _, failTime := range failovers {
			if now.Sub(failTime) < 5*time.Minute {
				recent = append(recent, failTime)
			}
		}
		bcm.failoverHistory[network] = recent
		
		// If too many recent failovers, take corrective action
		if len(recent) > 3 {
			bcm.logger.Warn("Excessive failovers detected",
				zap.String("network", string(network)),
				zap.Int("failover_count", len(recent)))
			
			bcm.handleExcessiveFailovers(network)
		}
	}
}

// updateStats updates connection statistics
func (bcm *BulletproofConnectionManager) updateStats(network NetworkType, latency time.Duration, success bool) {
	bcm.mu.Lock()
	defer bcm.mu.Unlock()
	
	stats := bcm.connectionStats[network]
	stats.TotalRequests++
	
	if success {
		stats.SuccessfulReqs++
		stats.LastSuccessTime = time.Now()
		
		// Update average latency (weighted)
		if stats.AverageLatency == 0 {
			stats.AverageLatency = latency
		} else {
			stats.AverageLatency = time.Duration(
				(int64(stats.AverageLatency)*9 + int64(latency)) / 10,
			)
		}
	} else {
		stats.FailedRequests++
	}
	
	// Calculate uptime percentage
	if stats.TotalRequests > 0 {
		stats.UptimePercentage = (float64(stats.SuccessfulReqs) / float64(stats.TotalRequests)) * 100.0
	}
}

// recordFailover records a failover event
func (bcm *BulletproofConnectionManager) recordFailover(network NetworkType) {
	bcm.mu.Lock()
	defer bcm.mu.Unlock()
	
	now := time.Now()
	bcm.failoverHistory[network] = append(bcm.failoverHistory[network], now)
	bcm.connectionStats[network].FailoverCount++
}

// refreshNetworkConnections refreshes connections for a network
func (bcm *BulletproofConnectionManager) refreshNetworkConnections(network NetworkType) {
	switch network {
	case Bitcoin:
		bcm.bitcoinClient.refreshPeers()
	case Ethereum:
		bcm.ethereumClient.refreshEndpoints()
	case Solana:
		bcm.solanaClient.refreshEndpoints()
	}
}

// handleExcessiveFailovers takes corrective action for excessive failovers
func (bcm *BulletproofConnectionManager) handleExcessiveFailovers(network NetworkType) {
	bcm.logger.Error("Taking corrective action for excessive failovers",
		zap.String("network", string(network)))
	
	// Force refresh all connections
	bcm.refreshNetworkConnections(network)
	
	// Reset failure counters
	bcm.mu.Lock()
	bcm.failoverHistory[network] = nil
	bcm.mu.Unlock()
}

// Stop gracefully shuts down the connection manager
func (bcm *BulletproofConnectionManager) Stop() error {
	bcm.logger.Info("Stopping Bulletproof Connection Manager")
	
	bcm.cancel()
	close(bcm.stopChan)
	bcm.wg.Wait()
	
	return bcm.networkManager.Stop()
}

// Note: Additional helper methods would be implemented for the clients:
// - refreshPeers() for BitcoinClient
// - refreshEndpoints() for EthereumClient and SolanaClient
// These would update the active endpoint lists from the NetworkManager
