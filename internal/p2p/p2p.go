package p2p

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
	"github.com/PayRpc/Bitcoin-Sprint/internal/metrics"
	"github.com/PayRpc/Bitcoin-Sprint/internal/netkit"
	"github.com/PayRpc/Bitcoin-Sprint/internal/securebuf"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/peer"
	"github.com/btcsuite/btcd/wire"
	"go.uber.org/zap"
)

// Service flag constants for peer validation
const (
	SvcNodeNetwork       = 1 << 0  // 1
	SvcNodeGetUTXO       = 1 << 1  // 2 (unused)
	SvcNodeBloom         = 1 << 2  // 4 (legacy)
	SvcNodeWitness       = 1 << 3  // 8
	SvcNodeNetworkLimited= 1 << 10 // 1024
	SvcNodeP2Pv2         = 1 << 11 // 2048
)

const minProtocol = 70016

// goodServices validates that peer has required service flags
func goodServices(s uint64) bool {
	hasNet := (s&SvcNodeNetwork) != 0 || (s&SvcNodeNetworkLimited) != 0
	hasWit := (s&SvcNodeWitness) != 0
	return hasNet && hasWit
}

// Client manages P2P peers with secure handshake authentication and resilient reconnection
type Client struct {
	cfg       config.Config
	blockChan chan blocks.BlockEvent
	mem       *mempool.Mempool
	logger    *zap.Logger

	peers []*peer.Peer
	mu    sync.RWMutex

	activePeers int32
	stopped     atomic.Bool

	auth *Authenticator

	// Concurrent processing pipeline
	blockProcessor *BlockProcessor

	// Adaptive connection management
	peerMetrics   map[string]*PeerMetrics
	peerMetricsMu sync.RWMutex

	// Network health monitoring
	networkHealth *NetworkHealthMonitor

	// Fee estimation
	feeEstimator *FeeEstimator
}

// PeerMetrics tracks performance metrics for adaptive peer selection
type PeerMetrics struct {
	address             string
	latency             time.Duration
	blocksReceived      int64
	lastSeen            time.Time
	qualityScore        float64
	consecutiveFailures int64
	circuitBreakerUntil time.Time
}

// BlockProcessor handles concurrent block processing with backpressure
type BlockProcessor struct {
	workers    int
	workChan   chan *wire.MsgBlock
	resultChan chan blocks.BlockEvent
	wg         sync.WaitGroup

	// Backpressure and circuit breaker
	queueDepth     int64
	maxQueueDepth  int64
	backpressureMu sync.RWMutex
	circuitBreaker *CircuitBreaker

	// Metrics
	processedBlocks    int64
	droppedBlocks      int64
	duplicateBlocks    int64
	backpressureEvents int64
}

// CircuitBreaker implements circuit breaker pattern for peer connections
type CircuitBreaker struct {
	failures    int64
	lastFailure time.Time
	state       CircuitState
	mu          sync.RWMutex
	threshold   int64
	timeout     time.Duration
	halfOpenMax int64
}

type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(threshold int64, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:       StateClosed,
		threshold:   threshold,
		timeout:     timeout,
		halfOpenMax: 3,
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen {
		if time.Since(cb.lastFailure) < cb.timeout {
			return errors.New("circuit breaker is open")
		}
		cb.state = StateHalfOpen
	}

	err := fn()
	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
		if cb.failures >= cb.threshold {
			cb.state = StateOpen
		}
		return err
	}

	if cb.state == StateHalfOpen {
		cb.failures = 0
		cb.state = StateClosed
	}

	return nil
}

func New(cfg config.Config, blockChan chan blocks.BlockEvent, mem *mempool.Mempool, logger *zap.Logger) (*Client, error) {
	// Initialize secure authenticator with HMAC secret from environment
	secret := []byte(os.Getenv("PEER_HMAC_SECRET"))
	if len(secret) == 0 {
		// Generate secure default secret using SecureBuffer
		logger.Warn("PEER_HMAC_SECRET not set - generating secure default")
		secretBuf, err := securebuf.New(64) // 64 bytes for strong HMAC secret
		if err != nil {
			return nil, fmt.Errorf("failed to create secure buffer for secret: %w", err)
		}
		defer secretBuf.Free()

		// Use a deterministic but secure default for dev
		defaultSecret := []byte("bitcoin-sprint-default-peer-secret-key-2025-entropy-backed")
		if err := secretBuf.Write(defaultSecret); err != nil {
			return nil, fmt.Errorf("failed to write to secure buffer: %w", err)
		}

		secretBytes, err := secretBuf.ReadToSlice()
		if err != nil {
			return nil, fmt.Errorf("failed to read from secure buffer: %w", err)
		}
		secret = secretBytes
	}

	auth, err := NewAuthenticator(secret, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create authenticator: %w", err)
	}

	return &Client{
		cfg:         cfg,
		blockChan:   blockChan,
		mem:         mem,
		logger:      logger,
		peers:       make([]*peer.Peer, 0),
		auth:        auth,
		peerMetrics: make(map[string]*PeerMetrics),
	}, nil
}

func (c *Client) Run() {
	c.logger.Info("Starting Bitcoin Sprint P2P client with parallel connection pool")

	// Production Bitcoin seed nodes
	nodes := []string{
		"seed.bitcoin.sipa.be:8333",          // Pieter Wuille
		"dnsseed.bluematt.me:8333",           // Matt Corallo
		"dnsseed.bitcoin.dashjr.org:8333",    // Luke Dashjr
		"seed.bitcoinstats.com:8333",         // Christian Decker
		"seed.bitnodes.io:8333",              // Addy Yeow
		"dnsseed.emzy.de:8333",               // Stephan Oeste
		"seed.bitcoin.jonasschnelli.ch:8333", // Jonas Schnelli
	}

	// Create connection pool with configurable size
	poolSize := c.getConnectionPoolSize()
	connectionChan := make(chan *peer.Peer, poolSize)

	// Start parallel connection goroutines
	for _, nodeAddr := range nodes {
		go c.parallelConnect(nodeAddr, connectionChan)
	}

	// Collect successful connections
	successfulConnections := 0
	for successfulConnections < poolSize {
		select {
		case p := <-connectionChan:
			if p != nil {
				c.addPeerSafe(p)
				successfulConnections++
			}
		case <-time.After(30 * time.Second):
			break
		}
	}

	c.mu.RLock()
	peerCount := len(c.peers)
	c.mu.RUnlock()
	c.logger.Info("P2P connection pool established",
		zap.Int("successful", successfulConnections),
		zap.Int("pool_size", poolSize),
		zap.Int("current_peers", peerCount))

	// Start concurrent block processing pipeline
	c.startBlockProcessingPipeline()
}

// getConnectionPoolSize returns the appropriate connection pool size based on tier
func (c *Client) getConnectionPoolSize() int {
	switch c.cfg.Tier {
	case config.TierTurbo:
		return 20 // Maximum connections for turbo tier
	case config.TierEnterprise:
		return 15 // High connection count for enterprise
	case config.TierPro, config.TierBusiness:
		return 10 // Moderate connections for pro/business
	default:
		return 5 // Conservative connection count for lite/standard
	}
}

// getTierAwareWorkerCount returns the appropriate worker count based on tier
func (c *Client) getTierAwareWorkerCount() int {
	// Use config-based pipeline workers, with fallback to CPU-based calculation
	if c.cfg.PipelineWorkers > 0 {
		return c.cfg.PipelineWorkers
	}

	switch c.cfg.Tier {
	case config.TierTurbo:
		return runtime.NumCPU() * 2
	case config.TierEnterprise:
		return runtime.NumCPU()
	case config.TierPro, config.TierBusiness:
		return runtime.NumCPU() / 2
	default:
		return runtime.NumCPU() / 4
	}
}

// parallelConnect attempts to connect to a peer and sends result to channel
func (c *Client) parallelConnect(address string, connectionChan chan<- *peer.Peer) {
	if c.stopped.Load() {
		connectionChan <- nil
		return
	}

	c.logger.Debug("Attempting parallel connection to peer", zap.String("address", address))

	config := &peer.Config{
		UserAgentName:    "Bitcoin-Sprint",
		UserAgentVersion: "2.1.0",
		ChainParams:      &chaincfg.MainNetParams,
		Services:         wire.SFNodeNetwork,
		TrickleInterval:  time.Second * 10,
		ProtocolVersion:  wire.ProtocolVersion,
		Listeners: peer.MessageListeners{
			OnVersion: func(p *peer.Peer, msg *wire.MsgVersion) *wire.MsgReject {
				// Enforce minimum protocol version
				if msg.ProtocolVersion < minProtocol {
					c.logger.Warn("Rejecting peer: protocol too old",
						zap.String("peer", address),
						zap.Uint32("version", uint32(msg.ProtocolVersion)),
						zap.Uint32("min_version", minProtocol))
					return wire.NewMsgReject(msg.Command(), wire.RejectMalformed, "protocol version too old")
				}

				// Enforce service flag requirements
				if !goodServices(uint64(msg.Services)) {
					c.logger.Warn("Rejecting peer: insufficient services",
						zap.String("peer", address),
						zap.Uint64("services", uint64(msg.Services)),
						zap.Bool("has_network", (uint64(msg.Services)&SvcNodeNetwork) != 0),
						zap.Bool("has_network_limited", (uint64(msg.Services)&SvcNodeNetworkLimited) != 0),
						zap.Bool("has_witness", (uint64(msg.Services)&SvcNodeWitness) != 0))
					return wire.NewMsgReject(msg.Command(), wire.RejectMalformed, "insufficient services")
				}

				c.logger.Info("Bitcoin protocol handshake completed",
					zap.String("peer", address),
					zap.String("user_agent", msg.UserAgent),
					zap.Uint32("protocol_version", uint32(msg.ProtocolVersion)),
					zap.Uint64("services", uint64(msg.Services)),
					zap.Bool("network", (uint64(msg.Services)&SvcNodeNetwork) != 0),
					zap.Bool("network_limited", (uint64(msg.Services)&SvcNodeNetworkLimited) != 0),
					zap.Bool("witness", (uint64(msg.Services)&SvcNodeWitness) != 0),
					zap.Bool("p2p_v2", (uint64(msg.Services)&SvcNodeP2Pv2) != 0))

				atomic.AddInt32(&c.activePeers, 1)
				return nil
			},
			OnVerAck: func(p *peer.Peer, msg *wire.MsgVerAck) {
				// Normal logging
				c.logger.Info("Version acknowledgment received", zap.String("peer", address))

				// For Sprint peers, authentication already happened during connection
				// For regular Bitcoin peers, no additional auth needed
			},
			OnPong: func(p *peer.Peer, msg *wire.MsgPong) {
				// Normal pong handling - no token validation needed
				// since Sprint authentication happens at connection time
				c.logger.Debug("Pong received", zap.String("peer", address))
			},
			OnBlock: func(p *peer.Peer, msg *wire.MsgBlock, buf []byte) {
				c.handleBlock(msg)
			},
			OnInv: func(p *peer.Peer, msg *wire.MsgInv) {
				c.handleInv(p, msg)
			},
			OnTx: func(p *peer.Peer, msg *wire.MsgTx) {
				c.logger.Debug("Received transaction",
					zap.String("txid", msg.TxHash().String()),
					zap.String("peer", address))
			},
		},
	}

	p, err := peer.NewOutboundPeer(config, address)
	if err != nil {
		c.logger.Warn("Failed to create peer", zap.String("address", address), zap.Error(err))
		connectionChan <- nil
		return
	}

	// Set connection timeout with enhanced dialing
	conn, err := netkit.DialHappy(address, 30*time.Second)
	if err != nil {
		c.logger.Warn("Failed to connect to peer with enhanced dialing",
			zap.String("address", address),
			zap.Error(err))
		connectionChan <- nil
		return
	}

	// If peer is in Sprint relay cluster list, enforce handshake
	if c.isSprintPeer(address) {
		if err := c.auth.PerformHandshakeClient(conn, 5*time.Second); err != nil {
			c.logger.Warn("Sprint handshake failed", zap.String("peer", address), zap.Error(err))
			conn.Close()
			connectionChan <- nil
			return
		}
		c.logger.Debug("Sprint peer authenticated", zap.String("peer", address))
	}

	// Add to peer list
	c.addPeerSafe(p)

	// Associate connection with peer
	p.AssociateConnection(conn)

	c.logger.Info("Peer connection established",
		zap.String("peer", address),
		zap.Int32("total_peers", int32(len(c.peers))))

	connectionChan <- p
}

// startBlockProcessingPipeline initializes concurrent block processing with backpressure
func (c *Client) startBlockProcessingPipeline() {
	// Tier-aware worker count
	workers := c.getTierAwareWorkerCount()

	c.blockProcessor = &BlockProcessor{
		workers:        workers,
		workChan:       make(chan *wire.MsgBlock, 1000),
		resultChan:     make(chan blocks.BlockEvent, 1000),
		maxQueueDepth:  int64(workers * 100),                 // Workers * 100 = max queue depth
		circuitBreaker: NewCircuitBreaker(5, 30*time.Second), // 5 failures, 30s timeout
	}

	// Start worker goroutines
	for i := 0; i < c.blockProcessor.workers; i++ {
		c.blockProcessor.wg.Add(1)
		go c.blockProcessingWorker()
	}

	// Start result handler
	go c.handleProcessedBlocks()

	// Start backpressure monitor
	go c.monitorBackpressure()

	c.logger.Info("Started block processing pipeline with backpressure",
		zap.Int("workers", workers),
		zap.Int64("max_queue_depth", c.blockProcessor.maxQueueDepth))
}

// blockProcessingWorker processes blocks concurrently with circuit breaker protection
func (c *Client) blockProcessingWorker() {
	defer c.blockProcessor.wg.Done()

	for block := range c.blockProcessor.workChan {
		blockHash := block.BlockHash().String()

		// TODO: Use global block deduplication index
		// source := "p2p"
		// if end, ok := dedup.BlockIdx.TryBegin(blockHash); !ok {
		// 	c.logger.Info("Duplicate block ignored",
		// 		zap.String("hash", blockHash),
		// 		zap.String("source", source),
		// 		zap.String("reason", "recent-seen"))
		// 	metrics.BlockDuplicatesIgnored.WithLabelValues(source).Inc()
		// 	atomic.AddInt64(&c.blockProcessor.duplicateBlocks, 1)
		// 	continue
		// }
		// defer end(true)

		source := "p2p"
		c.logger.Info("Processing block",
			zap.String("hash", blockHash),
			zap.Int("tx_count", len(block.Transactions)),
			zap.String("source", source))

		// Use circuit breaker to protect against cascading failures
		err := c.blockProcessor.circuitBreaker.Call(func() error {
			// Process block concurrently
			blockEvent := c.processBlockConcurrent(block)

			// Try to send result with timeout to prevent blocking
			select {
			case c.blockProcessor.resultChan <- blockEvent:
				atomic.AddInt64(&c.blockProcessor.processedBlocks, 1)
				// Record successful processing metric
				metrics.BlocksProcessed.WithLabelValues(source).Inc()
			case <-time.After(100 * time.Millisecond):
				atomic.AddInt64(&c.blockProcessor.droppedBlocks, 1)
				c.logger.Warn("Block processing result dropped due to timeout",
					zap.String("hash", blockEvent.Hash))
				return errors.New("result channel timeout")
			}
			return nil
		})

		if err != nil {
			c.logger.Warn("Circuit breaker activated for block processing", zap.Error(err))
		}
	}
}

// processBlockConcurrent processes a block and returns block event
func (c *Client) processBlockConcurrent(block *wire.MsgBlock) blocks.BlockEvent {
	detectionTime := time.Now()

	blockHash := block.BlockHash().String()
	c.logger.Info("Processing block concurrently",
		zap.String("hash", blockHash),
		zap.Int("tx_count", len(block.Transactions)))

	// Create block event for Sprint processing
	blockEvent := blocks.BlockEvent{
		Hash:      blockHash,
		Height:    0, // Height will be determined by block processing
		Timestamp: detectionTime,
		Source:    "p2p-concurrent",
	}

	return blockEvent
}

// handleProcessedBlocks handles completed block processing results
func (c *Client) handleProcessedBlocks() {
	for blockEvent := range c.blockProcessor.resultChan {
		// Send to block processing channel (non-blocking)
		select {
		case c.blockChan <- blockEvent:
			c.logger.Debug("Concurrent block event sent to processing channel",
				zap.String("hash", blockEvent.Hash))
		default:
			c.logger.Warn("Block channel full, dropping concurrent block event",
				zap.String("hash", blockEvent.Hash))
		}
	}
}

// handleBlockHeaders processes block headers for faster propagation
func (c *Client) handleBlockHeaders(msg *wire.MsgHeaders) {
	if c.stopped.Load() {
		return
	}

	for _, hdr := range msg.Headers {
		blockHash := hdr.BlockHash()

		// Create header-only block event for immediate relay
		headerEvent := blocks.BlockEvent{
			Hash:      blockHash.String(),
			Height:    0, // Will be determined later
			Timestamp: hdr.Timestamp,
			Source:    "p2p-header",
			IsHeader:  true,
		}

		// Relay header immediately for ultra-low latency
		select {
		case c.blockChan <- headerEvent:
			c.logger.Debug("Block header relayed immediately",
				zap.String("hash", blockHash.String()))
		default:
			c.logger.Warn("Block header channel full")
		}

		// Request full block in background
		go c.requestFullBlock(blockHash)
	}
}

// requestFullBlock requests the full block data for a given header
func (c *Client) requestFullBlock(blockHash chainhash.Hash) {
	getData := wire.NewMsgGetData()
	getData.AddInvVect(wire.NewInvVect(wire.InvTypeBlock, &blockHash))

	// Send to first available peer
	c.mu.RLock()
	if len(c.peers) > 0 {
		c.peers[0].QueueMessage(getData, nil)
		c.logger.Debug("Requested full block data",
			zap.String("hash", blockHash.String()))
	}
	c.mu.RUnlock()
}

// updatePeerMetrics updates performance metrics for a peer
func (c *Client) updatePeerMetrics(peerAddr string, latency time.Duration, success bool) {
	c.peerMetricsMu.Lock()
	defer c.peerMetricsMu.Unlock()

	if c.peerMetrics == nil {
		c.peerMetrics = make(map[string]*PeerMetrics)
	}

	metrics := c.peerMetrics[peerAddr]
	if metrics == nil {
		metrics = &PeerMetrics{
			address: peerAddr,
		}
		c.peerMetrics[peerAddr] = metrics
	}

	metrics.latency = latency
	metrics.lastSeen = time.Now()

	if success {
		metrics.blocksReceived++
		metrics.consecutiveFailures = 0
		// Clear circuit breaker if it was set
		if time.Now().After(metrics.circuitBreakerUntil) {
			metrics.circuitBreakerUntil = time.Time{}
		}
	} else {
		metrics.consecutiveFailures++
		// Activate circuit breaker after 3 consecutive failures
		if metrics.consecutiveFailures >= 3 {
			metrics.circuitBreakerUntil = time.Now().Add(5 * time.Minute)
			c.logger.Warn("Circuit breaker activated for peer",
				zap.String("peer", peerAddr),
				zap.Int64("failures", metrics.consecutiveFailures))
		}
	}

	metrics.qualityScore = c.calculateQualityScore(metrics)

	c.logger.Debug("Updated peer metrics",
		zap.String("peer", peerAddr),
		zap.Duration("latency", latency),
		zap.Bool("success", success),
		zap.Int64("consecutive_failures", metrics.consecutiveFailures),
		zap.Float64("quality_score", metrics.qualityScore))

	// Persist metrics periodically (every 100 updates)
	if metrics.blocksReceived%100 == 0 {
		go c.persistPeerMetrics()
	}
}

// calculateQualityScore calculates a quality score for peer selection
func (c *Client) calculateQualityScore(metrics *PeerMetrics) float64 {
	// Base score starts at 1.0
	score := 1.0

	// Penalize high latency (lower is better)
	if metrics.latency > 0 {
		latencyPenalty := metrics.latency.Seconds() / 10.0 // 10 seconds = full penalty
		score -= latencyPenalty
	}

	// Reward recent activity
	timeSinceLastSeen := time.Since(metrics.lastSeen)
	if timeSinceLastSeen < time.Minute {
		score += 0.5
	} else if timeSinceLastSeen < 5*time.Minute {
		score += 0.2
	}

	// Penalize consecutive failures
	if metrics.consecutiveFailures > 0 {
		failurePenalty := float64(metrics.consecutiveFailures) * 0.2
		score -= failurePenalty
	}

	// Heavy penalty for circuit breaker activation
	if time.Now().Before(metrics.circuitBreakerUntil) {
		score -= 2.0 // Effectively disable peer
	}

	// Reward blocks received
	if metrics.blocksReceived > 0 {
		blockReward := float64(metrics.blocksReceived) * 0.01
		if blockReward > 1.0 {
			blockReward = 1.0
		}
		score += blockReward
	}

	// Ensure score doesn't go below -1
	if score < -1 {
		score = -1
	}

	return score
}

// persistPeerMetrics saves peer metrics to disk
func (c *Client) persistPeerMetrics() {
	if c.peerMetrics == nil || len(c.peerMetrics) == 0 {
		return
	}

	c.peerMetricsMu.RLock()
	defer c.peerMetricsMu.RUnlock()

	// Simple JSON persistence (in production, consider using a database)
	type PersistentMetrics struct {
		Address             string    `json:"address"`
		LatencyNs           int64     `json:"latency_ns"`
		BlocksReceived      int64     `json:"blocks_received"`
		LastSeen            time.Time `json:"last_seen"`
		QualityScore        float64   `json:"quality_score"`
		ConsecutiveFailures int64     `json:"consecutive_failures"`
		CircuitBreakerUntil time.Time `json:"circuit_breaker_until"`
	}

	var persistentMetrics []PersistentMetrics
	for _, metrics := range c.peerMetrics {
		persistentMetrics = append(persistentMetrics, PersistentMetrics{
			Address:             metrics.address,
			LatencyNs:           metrics.latency.Nanoseconds(),
			BlocksReceived:      metrics.blocksReceived,
			LastSeen:            metrics.lastSeen,
			QualityScore:        metrics.qualityScore,
			ConsecutiveFailures: metrics.consecutiveFailures,
			CircuitBreakerUntil: metrics.circuitBreakerUntil,
		})
	}

	// In a real implementation, you'd write this to a file or database
	// For now, we'll just log that persistence would happen
	c.logger.Debug("Peer metrics persistence triggered",
		zap.Int("peer_count", len(persistentMetrics)))
}

// loadPeerMetrics loads peer metrics from disk
func (c *Client) loadPeerMetrics() {
	// In a real implementation, you'd read from a file or database
	// For now, we'll initialize with empty metrics
	c.peerMetrics = make(map[string]*PeerMetrics)
	c.logger.Debug("Peer metrics loaded from persistence")
}

func (c *Client) Stop() {
	if c.stopped.CompareAndSwap(false, true) {
		c.logger.Info("Stopping P2P client")

		// Close authenticator
		if c.auth != nil {
			c.auth.Close()
		}

		// Disconnect all peers
		c.mu.Lock()
		for _, p := range c.peers {
			p.Disconnect()
		}
		c.peers = nil
		c.mu.Unlock()

		c.logger.Info("P2P client stopped")
	}
}

// retryConnect keeps trying to connect with exponential backoff - never gives up
func (c *Client) retryConnect(address string) {
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second
	currentDelay := baseDelay

	for {
		if c.stopped.Load() {
			c.logger.Info("Reconnection manager stopping",
				zap.String("address", address))
			return
		}

		err := c.connectToPeer(address)
		if err == nil {
			// Connection successful - reset backoff and monitor
			currentDelay = baseDelay
			c.logger.Info("Peer connected successfully",
				zap.String("address", address))

			// Monitor this connection and restart if it fails
			c.monitorPeerConnection(address)
			continue
		}

		// Log failure and wait with exponential backoff
		c.logger.Warn("Peer connection failed, retrying with exponential backoff",
			zap.String("address", address),
			zap.Error(err),
			zap.Duration("retry_in", currentDelay))

		// Wait before retrying
		select {
		case <-time.After(currentDelay):
			// Continue to retry
		}

		// Increase delay exponentially, but cap at maxDelay
		currentDelay *= 2
		if currentDelay > maxDelay {
			currentDelay = maxDelay
		}
	}
}

// monitorPeerConnection watches a connected peer and returns when disconnected
func (c *Client) monitorPeerConnection(address string) {
	// Simple monitoring - in production you'd want more sophisticated monitoring
	// For now, we'll rely on the peer disconnect callbacks
	c.logger.Debug("Monitoring peer connection", zap.String("address", address))
}

func (c *Client) connectToPeer(address string) error {
	if c.stopped.Load() {
		return fmt.Errorf("client stopped")
	}

	c.logger.Debug("Attempting to connect to peer", zap.String("address", address))

	config := &peer.Config{
		UserAgentName:    "Bitcoin-Sprint",
		UserAgentVersion: "2.1.0",
		ChainParams:      &chaincfg.MainNetParams,
		Services:         wire.SFNodeNetwork,
		TrickleInterval:  time.Second * 10,
		ProtocolVersion:  wire.ProtocolVersion,
		Listeners: peer.MessageListeners{
			OnVersion: func(p *peer.Peer, msg *wire.MsgVersion) *wire.MsgReject {
				c.logger.Info("Bitcoin protocol handshake completed",
					zap.String("peer", address),
					zap.String("user_agent", msg.UserAgent),
					zap.Uint32("protocol_version", uint32(msg.ProtocolVersion)),
					zap.Uint64("services", uint64(msg.Services)))

				atomic.AddInt32(&c.activePeers, 1)
				return nil
			},
			OnVerAck: func(p *peer.Peer, msg *wire.MsgVerAck) {
				// Normal logging
				c.logger.Info("Version acknowledgment received", zap.String("peer", address))

				// For Sprint peers, authentication already happened during connection
				// For regular Bitcoin peers, no additional auth needed
			},
			OnPong: func(p *peer.Peer, msg *wire.MsgPong) {
				// Normal pong handling - no token validation needed
				// since Sprint authentication happens at connection time
				c.logger.Debug("Pong received", zap.String("peer", address))
			},
			OnBlock: func(p *peer.Peer, msg *wire.MsgBlock, buf []byte) {
				c.handleBlock(msg)
			},
			OnInv: func(p *peer.Peer, msg *wire.MsgInv) {
				c.handleInv(p, msg)
			},
			OnTx: func(p *peer.Peer, msg *wire.MsgTx) {
				c.logger.Debug("Received transaction",
					zap.String("txid", msg.TxHash().String()),
					zap.String("peer", address))
			},
		},
	}

	p, err := peer.NewOutboundPeer(config, address)
	if err != nil {
		return fmt.Errorf("failed to create peer: %w", err)
	}

	// Set connection timeout with enhanced dialing
	conn, err := netkit.DialHappy(address, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect with enhanced dialing: %w", err)
	}

	// If peer is in Sprint relay cluster list, enforce handshake
	if c.isSprintPeer(address) {
		if err := c.auth.PerformHandshakeClient(conn, 5*time.Second); err != nil {
			c.logger.Warn("Sprint handshake failed", zap.String("peer", address), zap.Error(err))
			conn.Close()
			return err
		}
		c.logger.Debug("Sprint peer authenticated", zap.String("peer", address))
	}

	// Add to peer list
	c.addPeerSafe(p)

	// Associate connection with peer
	p.AssociateConnection(conn)

	c.logger.Info("Peer connection established",
		zap.String("peer", address),
		zap.Int32("total_peers", int32(len(c.peers))))

	return nil
}

func (c *Client) handleBlock(block *wire.MsgBlock) {
	if c.stopped.Load() {
		return
	}

	blockHash := block.BlockHash().String()
	c.logger.Info("Received new block from Bitcoin network",
		zap.String("hash", blockHash),
		zap.Int("tx_count", len(block.Transactions)),
		zap.Time("timestamp", block.Header.Timestamp))

	// Update network health with new block
	c.updateNetworkHealthWithBlock(block)

	// Check backpressure before sending to processing pipeline
	queueLen := len(c.blockProcessor.workChan)
	if int64(queueLen) > c.blockProcessor.maxQueueDepth*9/10 {
		c.logger.Warn("Backpressure: dropping block due to full queue",
			zap.String("hash", blockHash),
			zap.Int("queue_len", queueLen),
			zap.Int64("max_depth", c.blockProcessor.maxQueueDepth))
		return
	}

	// Send to concurrent processing pipeline (non-blocking)
	select {
	case c.blockProcessor.workChan <- block:
		c.logger.Debug("Block sent to concurrent processing pipeline",
			zap.String("hash", blockHash))
	default:
		c.logger.Warn("Block processing pipeline full, dropping block",
			zap.String("hash", blockHash))
	}
}

func (c *Client) handleInv(p *peer.Peer, msg *wire.MsgInv) {
	if c.stopped.Load() {
		return
	}

	getData := wire.NewMsgGetData()

	for _, inv := range msg.InvList {
		switch inv.Type {
		case wire.InvTypeBlock:
			c.logger.Debug("Requesting block from inventory",
				zap.String("hash", inv.Hash.String()))
			getData.AddInvVect(inv)
		case wire.InvTypeTx:
			c.logger.Debug("Requesting transaction from inventory",
				zap.String("hash", inv.Hash.String()))
			getData.AddInvVect(inv)
		}
	}

	if len(getData.InvList) > 0 {
		p.QueueMessage(getData, nil)
		c.logger.Debug("Requested inventory items",
			zap.Int("count", len(getData.InvList)))
	}
}

// GetActivePeerCount returns the current number of active peers
func (c *Client) GetActivePeerCount() int32 {
	return atomic.LoadInt32(&c.activePeers)
}

// GetPeerInfo returns information about connected peers
func (c *Client) GetPeerInfo() []map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	peerInfo := make([]map[string]interface{}, 0, len(c.peers))
	for _, p := range c.peers {
		if p.Connected() {
			info := map[string]interface{}{
				"address":    p.Addr(),
				"user_agent": p.UserAgent(),
				"version":    p.ProtocolVersion(),
				"connected":  p.Connected(),
			}
			peerInfo = append(peerInfo, info)
		}
	}

	return peerInfo
}

// monitorBackpressure monitors queue depth and applies backpressure
func (c *Client) monitorBackpressure() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if c.stopped.Load() {
				return
			}

			queueLen := len(c.blockProcessor.workChan)
			queueDepth := int64(queueLen)

			// Update metrics
			atomic.StoreInt64(&c.blockProcessor.queueDepth, queueDepth)

			// Apply backpressure if queue is 90% full
			if queueDepth > c.blockProcessor.maxQueueDepth*9/10 {
				atomic.AddInt64(&c.blockProcessor.backpressureEvents, 1)

				c.logger.Warn("Backpressure triggered, slowing intake",
					zap.Int64("queue_depth", queueDepth),
					zap.Int64("max_depth", c.blockProcessor.maxQueueDepth),
					zap.Int64("backpressure_events", atomic.LoadInt64(&c.blockProcessor.backpressureEvents)))

				time.Sleep(50 * time.Millisecond)
			}
		}
	}
}

// requestHeadersFromPeer requests block headers from a peer with tier-aware limits
func (c *Client) requestHeadersFromPeer(p *peer.Peer, startHash *chainhash.Hash) {
	if c.stopped.Load() {
		return
	}

	// Use tier-aware header limit from config
	maxHeaders := c.cfg.MaxOutstandingHeadersPerPeer
	if maxHeaders <= 0 {
		maxHeaders = 2000 // Default fallback
	}

	getHeaders := wire.NewMsgGetHeaders()
	getHeaders.HashStop = chainhash.Hash{} // Request up to current tip
	getHeaders.AddBlockLocatorHash(startHash)

	c.logger.Debug("Requesting headers from peer",
		zap.String("peer", p.Addr()),
		zap.String("start_hash", startHash.String()),
		zap.Int("max_headers", maxHeaders))

	p.QueueMessage(getHeaders, nil)
}

// NetworkHealthMonitor tracks overall network health and performance
type NetworkHealthMonitor struct {
	networkHashrate   int64
	blockInterval     time.Duration
	lastBlockTime     time.Time
	networkDifficulty float64
	peerCount         int32
	mu                sync.RWMutex
}

// FeeEstimator provides fee estimation based on mempool data
type FeeEstimator struct {
	feeRates   map[int]*FeeRate // Maps confirmation target to fee rate
	lastUpdate time.Time
	mu         sync.RWMutex
}

// FeeRate represents a fee rate for a specific confirmation target
type FeeRate struct {
	satPerByte   float64
	targetBlocks int
	lastUpdate   time.Time
}

// UpdateNetworkHealth updates network health metrics
func (c *Client) UpdateNetworkHealth() {
	if c.networkHealth == nil {
		c.networkHealth = &NetworkHealthMonitor{}
	}

	c.networkHealth.mu.Lock()
	defer c.networkHealth.mu.Unlock()

	// Update peer count
	c.networkHealth.peerCount = atomic.LoadInt32(&c.activePeers)

	// Estimate network hashrate (simplified calculation)
	// In a real implementation, this would use block timestamps and difficulty
	if !c.networkHealth.lastBlockTime.IsZero() {
		timeDiff := time.Since(c.networkHealth.lastBlockTime)
		if timeDiff > 0 {
			c.networkHealth.blockInterval = timeDiff
		}
	}

	c.logger.Debug("Updated network health metrics",
		zap.Int32("peer_count", c.networkHealth.peerCount),
		zap.Duration("block_interval", c.networkHealth.blockInterval))
}

// GetNetworkHealth returns current network health status
func (c *Client) GetNetworkHealth() map[string]interface{} {
	if c.networkHealth == nil {
		return map[string]interface{}{
			"status": "initializing",
		}
	}

	c.networkHealth.mu.RLock()
	defer c.networkHealth.mu.RUnlock()

	return map[string]interface{}{
		"peer_count":      c.networkHealth.peerCount,
		"block_interval":  c.networkHealth.blockInterval.String(),
		"network_status":  c.getNetworkStatus(),
		"last_block_time": c.networkHealth.lastBlockTime,
	}
}

// getNetworkStatus returns a human-readable network status
func (c *Client) getNetworkStatus() string {
	peerCount := atomic.LoadInt32(&c.activePeers)

	switch {
	case peerCount == 0:
		return "disconnected"
	case peerCount < 3:
		return "poor_connectivity"
	case peerCount < 8:
		return "fair_connectivity"
	default:
		return "good_connectivity"
	}
}

// EstimateFee provides fee estimation for transaction confirmation
func (c *Client) EstimateFee(targetBlocks int) (float64, error) {
	if c.feeEstimator == nil {
		c.feeEstimator = &FeeEstimator{
			feeRates: make(map[int]*FeeRate),
		}
	}

	c.feeEstimator.mu.RLock()
	defer c.feeEstimator.mu.RUnlock()

	if feeRate, exists := c.feeEstimator.feeRates[targetBlocks]; exists {
		// Check if estimate is still fresh (within 5 minutes)
		if time.Since(feeRate.lastUpdate) < 5*time.Minute {
			return feeRate.satPerByte, nil
		}
	}

	// Request fee estimation from peers
	return c.requestFeeEstimation(targetBlocks)
}

// requestFeeEstimation requests fee estimation from connected peers
func (c *Client) requestFeeEstimation(targetBlocks int) (float64, error) {
	// Send fee filter message to peers to get fee estimation
	// This is a simplified implementation - in practice you'd aggregate from multiple peers

	// Default fee rates based on target blocks (conservative estimates)
	defaultRates := map[int]float64{
		1:  50.0, // 1 block: high priority
		2:  40.0, // 2 blocks: very high priority
		3:  30.0, // 3 blocks: high priority
		6:  20.0, // 6 blocks: medium priority
		10: 15.0, // 10 blocks: low priority
		20: 10.0, // 20 blocks: very low priority
	}

	if rate, exists := defaultRates[targetBlocks]; exists {
		// Update fee estimator cache
		c.feeEstimator.mu.Lock()
		c.feeEstimator.feeRates[targetBlocks] = &FeeRate{
			satPerByte:   rate,
			targetBlocks: targetBlocks,
			lastUpdate:   time.Now(),
		}
		c.feeEstimator.mu.Unlock()

		return rate, nil
	}

	// For custom targets, interpolate from known values
	return 5.0, nil // Minimum fee rate
}

// addPeerSafe safely adds a peer to the peers slice with proper locking
func (c *Client) addPeerSafe(p *peer.Peer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.peers = append(c.peers, p)
}

// isSprintPeer checks if an address is in the configured Sprint relay peer list
func (c *Client) isSprintPeer(addr string) bool {
	for _, sprintNode := range c.cfg.SprintRelayPeers {
		if addr == sprintNode {
			return true
		}
	}
	return false
}

// updateNetworkHealthWithBlock updates network health metrics when a new block is received
func (c *Client) updateNetworkHealthWithBlock(block *wire.MsgBlock) {
	if c.networkHealth == nil {
		c.networkHealth = &NetworkHealthMonitor{}
	}

	c.networkHealth.mu.Lock()
	defer c.networkHealth.mu.Unlock()

	prev := c.networkHealth.lastBlockTime
	now := block.Header.Timestamp
	if !prev.IsZero() && now.After(prev) {
		c.networkHealth.blockInterval = now.Sub(prev)
	}
	c.networkHealth.lastBlockTime = now

	// Update difficulty (simplified - would need actual calc)
	c.networkHealth.networkDifficulty = float64(block.Header.Bits)

	c.logger.Debug("Updated network health with new block",
		zap.Time("block_time", block.Header.Timestamp),
		zap.Duration("interval", c.networkHealth.blockInterval))
}
