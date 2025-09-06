package relay

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"go.uber.org/zap"
)

// RelayClient defines the universal interface for blockchain relay operations
// This interface abstracts different blockchain networks (Bitcoin, Ethereum, Solana, etc.)
type RelayClient interface {
	// Core relay operations
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool

	// Block streaming
	StreamBlocks(ctx context.Context, blockChan chan<- blocks.BlockEvent) error
	GetLatestBlock() (*blocks.BlockEvent, error)
	GetBlockByHash(hash string) (*blocks.BlockEvent, error)
	GetBlockByHeight(height uint64) (*blocks.BlockEvent, error)

	// Network information
	GetNetworkInfo() (*NetworkInfo, error)
	GetPeerCount() int
	GetSyncStatus() (*SyncStatus, error)

	// Health and metrics
	GetHealth() (*HealthStatus, error)
	GetMetrics() (*RelayMetrics, error)

	// Network-specific operations
	SupportsFeature(feature Feature) bool
	GetSupportedFeatures() []Feature

	// Configuration
	UpdateConfig(cfg RelayConfig) error
	GetConfig() RelayConfig
}

// RelayDispatcher manages multiple relay clients and routes requests
type RelayDispatcher struct {
	clients    map[string]RelayClient
	logger     *zap.Logger
	cfg        config.Config
	mu         sync.RWMutex
	deduper    *BlockDeduper
	dedupeStop chan struct{}
}

// NetworkInfo contains network-specific information
type NetworkInfo struct {
	Network         string    `json:"network"`
	ChainID         string    `json:"chain_id,omitempty"`
	BlockHeight     uint64    `json:"block_height"`
	BlockHash       string    `json:"block_hash"`
	NetworkHashrate *string   `json:"network_hashrate,omitempty"`
	Difficulty      *string   `json:"difficulty,omitempty"`
	PeerCount       int       `json:"peer_count"`
	Timestamp       time.Time `json:"timestamp"`
}

// SyncStatus represents the synchronization status
type SyncStatus struct {
	IsSyncing              bool           `json:"is_syncing"`
	CurrentHeight          uint64         `json:"current_height"`
	HighestHeight          uint64         `json:"highest_height"`
	SyncProgress           float64        `json:"sync_progress"`
	EstimatedTimeRemaining *time.Duration `json:"estimated_time_remaining,omitempty"`
}

// HealthStatus represents the health of a relay client
type HealthStatus struct {
	IsHealthy       bool          `json:"is_healthy"`
	LastSeen        time.Time     `json:"last_seen"`
	ErrorCount      int64         `json:"error_count"`
	Latency         time.Duration `json:"latency"`
	ConnectionState string        `json:"connection_state"`
	ErrorMessage    string        `json:"error_message,omitempty"`
}

// RelayMetrics contains performance metrics
type RelayMetrics struct {
	BlocksReceived    int64         `json:"blocks_received"`
	BlocksPerSecond   float64       `json:"blocks_per_second"`
	AverageLatency    time.Duration `json:"average_latency"`
	ErrorRate         float64       `json:"error_rate"`
	ConnectionUptime  time.Duration `json:"connection_uptime"`
	BytesReceived     int64         `json:"bytes_received"`
	BytesTransmitted  int64         `json:"bytes_transmitted"`
	LastBlockReceived time.Time     `json:"last_block_received"`
}

// Feature represents supported relay features
type Feature string

const (
	FeatureBlockStreaming  Feature = "block_streaming"
	FeatureTransactionPool Feature = "transaction_pool"
	FeatureHistoricalData  Feature = "historical_data"
	FeatureSmartContracts  Feature = "smart_contracts"
	FeatureStateQueries    Feature = "state_queries"
	FeatureEventLogs       Feature = "event_logs"
	FeatureCompactBlocks   Feature = "compact_blocks"
	FeatureWebSocket       Feature = "websocket"
	FeatureGraphQL         Feature = "graphql"
	FeatureREST            Feature = "rest"
)

// RelayConfig contains relay-specific configuration
type RelayConfig struct {
	Network           string            `json:"network"`
	Endpoints         []string          `json:"endpoints"`
	Timeout           time.Duration     `json:"timeout"`
	RetryAttempts     int               `json:"retry_attempts"`
	RetryDelay        time.Duration     `json:"retry_delay"`
	MaxConcurrency    int               `json:"max_concurrency"`
	BufferSize        int               `json:"buffer_size"`
	EnableCompression bool              `json:"enable_compression"`
	CustomHeaders     map[string]string `json:"custom_headers,omitempty"`
	AuthToken         string            `json:"auth_token,omitempty"`
	TLSConfig         *TLSConfig        `json:"tls_config,omitempty"`
}

// TLSConfig contains TLS-specific settings
type TLSConfig struct {
	Enabled            bool   `json:"enabled"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
	CertFile           string `json:"cert_file,omitempty"`
	KeyFile            string `json:"key_file,omitempty"`
	CAFile             string `json:"ca_file,omitempty"`
}

// NewRelayDispatcher creates a new relay dispatcher
func NewRelayDispatcher(cfg config.Config, logger *zap.Logger) *RelayDispatcher {
	dispatcher := &RelayDispatcher{
		clients:    make(map[string]RelayClient),
		logger:     logger,
		cfg:        cfg,
		deduper:    NewBlockDeduper(8192, 5*time.Minute), // 8K capacity with 5min TTL
		dedupeStop: make(chan struct{}),
	}

	// Start background cleanup for the deduper
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				dispatcher.deduper.Cleanup()
			case <-dispatcher.dedupeStop:
				return
			}
		}
	}()

	return dispatcher
}

// RegisterClient registers a relay client for a specific network
func (d *RelayDispatcher) RegisterClient(network string, client RelayClient) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.clients[network]; exists {
		return fmt.Errorf("relay client for network %s already registered", network)
	}

	d.clients[network] = client
	d.logger.Info("Registered relay client", zap.String("network", network))
	return nil
}

// GetClient returns a relay client for the specified network
func (d *RelayDispatcher) GetClient(network string) (RelayClient, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	client, exists := d.clients[network]
	if !exists {
		return nil, fmt.Errorf("no relay client registered for network %s", network)
	}

	return client, nil
}

// GetSupportedNetworks returns a list of supported networks
func (d *RelayDispatcher) GetSupportedNetworks() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	networks := make([]string, 0, len(d.clients))
	for network := range d.clients {
		networks = append(networks, network)
	}

	return networks
}

// StreamAllBlocks streams blocks from all registered relay clients
func (d *RelayDispatcher) StreamAllBlocks(ctx context.Context, blockChan chan<- blocks.BlockEvent) error {
	d.mu.RLock()
	clients := make(map[string]RelayClient)
	for network, client := range d.clients {
		clients[network] = client
	}
	d.mu.RUnlock()

	for network, client := range clients {
		go func(net string, cli RelayClient) {
			if err := cli.StreamBlocks(ctx, blockChan); err != nil {
				d.logger.Error("Failed to stream blocks",
					zap.String("network", net),
					zap.Error(err))
			}
		}(network, client)
	}

	return nil
}

// GetHealthStatus returns health status for all registered clients
func (d *RelayDispatcher) GetHealthStatus() map[string]*HealthStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()

	status := make(map[string]*HealthStatus)
	for network, client := range d.clients {
		if health, err := client.GetHealth(); err == nil {
			status[network] = health
		} else {
			status[network] = &HealthStatus{
				IsHealthy:       false,
				ErrorMessage:    err.Error(),
				ConnectionState: "error",
			}
		}
	}

	return status
}

// GetMetrics returns metrics for all registered clients
func (d *RelayDispatcher) GetMetrics() map[string]*RelayMetrics {
	d.mu.RLock()
	defer d.mu.RUnlock()

	metrics := make(map[string]*RelayMetrics)
	for network, client := range d.clients {
		if metric, err := client.GetMetrics(); err == nil {
			metrics[network] = metric
		}
	}

	return metrics
}

// Shutdown gracefully shuts down all relay clients
func (d *RelayDispatcher) Shutdown(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Stop deduper cleanup goroutine
	if d.dedupeStop != nil {
		close(d.dedupeStop)
		d.dedupeStop = nil
	}

	for network, client := range d.clients {
		if err := client.Disconnect(); err != nil {
			d.logger.Warn("Error disconnecting relay client",
				zap.String("network", network),
				zap.Error(err))
		}
	}

	d.logger.Info("Relay dispatcher shutdown complete")
	return nil
}
