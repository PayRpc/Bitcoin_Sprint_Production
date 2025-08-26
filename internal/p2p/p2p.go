package p2p

import (
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/peer"
	"github.com/btcsuite/btcd/wire"
	"go.uber.org/zap"
)

// Client manages P2P peers with secure handshake authentication
type Client struct {
	cfg       config.Config
	blockChan chan blocks.BlockEvent
	mem       *mempool.Mempool
	logger    *zap.Logger

	peers []*peer.Peer
	mu    sync.RWMutex

	activePeers int32
	stopped     atomic.Bool

	// Secure handshake authenticator
	auth *Authenticator
}

func New(cfg config.Config, blockChan chan blocks.BlockEvent, mem *mempool.Mempool, logger *zap.Logger) (*Client, error) {
	// Initialize secure authenticator with HMAC secret from environment
	secret := []byte(os.Getenv("PEER_HMAC_SECRET"))
	if len(secret) == 0 {
		// Use default secret for development (in production, this should always be set)
		secret = []byte("bitcoin-sprint-default-peer-secret-key-2025")
		logger.Warn("Using default PEER_HMAC_SECRET - set environment variable for production")
	}

	auth, err := NewAuthenticator(secret, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create authenticator: %w", err)
	}

	return &Client{
		cfg:       cfg,
		blockChan: blockChan,
		mem:       mem,
		logger:    logger,
		peers:     make([]*peer.Peer, 0),
		auth:      auth,
	}, nil
}

func (c *Client) Run() {
	c.logger.Info("Starting P2P client with secure handshake authentication")

	// Add some hardcoded Bitcoin nodes for demo
	nodes := []string{
		"seed.bitcoin.sipa.be:8333",
		"dnsseed.bitcoin.dashjr.org:8333",
		"seed.bitcoinstats.com:8333",
	}

	for _, nodeAddr := range nodes {
		go c.connectToPeer(nodeAddr)
	}
}

func (c *Client) Stop() {
	if c.stopped.CompareAndSwap(false, true) {
		c.logger.Info("Stopping P2P client")

		// Close authenticator
		if c.auth != nil {
			c.auth.Close()
		}

		c.mu.Lock()
		for _, p := range c.peers {
			p.Disconnect()
		}
		c.peers = nil
		c.mu.Unlock()
	}
}

func (c *Client) connectToPeer(address string) {
	if c.stopped.Load() {
		return
	}

	c.logger.Info("Connecting to peer", zap.String("address", address))

	// First, establish TCP connection for secure handshake
	conn, err := net.DialTimeout("tcp", address, 30*time.Second)
	if err != nil {
		c.logger.Warn("Failed to connect to peer",
			zap.String("address", address),
			zap.Error(err))
		return
	}

	// Perform secure handshake
	if err := c.auth.DoOutbound(conn); err != nil {
		c.logger.Warn("Outbound handshake failed",
			zap.String("address", address),
			zap.Error(err))
		conn.Close()
		return
	}

	c.logger.Info("Secure handshake completed", zap.String("address", address))

	// Now proceed with Bitcoin protocol handshake
	c.createBitcoinPeer(address, conn)
}

func (c *Client) createBitcoinPeer(address string, conn net.Conn) {
	// Create peer configuration
	config := &peer.Config{
		UserAgentName:    "Bitcoin-Sprint",
		UserAgentVersion: "1.0.0",
		ChainParams:      &chaincfg.MainNetParams,
		Services:         wire.SFNodeNetwork,
		TrickleInterval:  time.Second * 10,

		Listeners: peer.MessageListeners{
			OnBlock: func(p *peer.Peer, msg *wire.MsgBlock, buf []byte) {
				c.handleBlock(msg)
			},
			OnInv: func(p *peer.Peer, msg *wire.MsgInv) {
				c.handleInv(p, msg)
			},
			OnVersion: func(p *peer.Peer, msg *wire.MsgVersion) *wire.MsgReject {
				c.logger.Info("Bitcoin protocol handshake completed",
					zap.String("address", address),
					zap.String("user_agent", msg.UserAgent))
				atomic.AddInt32(&c.activePeers, 1)
				return nil
			},
		},
	}

	// Create peer with existing authenticated connection
	p, err := peer.NewOutboundPeer(config, address)
	if err != nil {
		c.logger.Error("Failed to create peer",
			zap.String("address", address),
			zap.Error(err))
		conn.Close()
		return
	}

	// Store peer
	c.mu.Lock()
	c.peers = append(c.peers, p)
	c.mu.Unlock()

	// Start peer with authenticated connection
	go p.AssociateConnection(conn)

	c.logger.Info("Peer connected and authenticated", zap.String("address", address))
}

func (c *Client) handleInboundConnection(conn net.Conn) {
	defer conn.Close()

	// Perform secure handshake for inbound connection
	if err := c.auth.DoInbound(conn); err != nil {
		c.logger.Warn("Inbound handshake failed", zap.Error(err))
		return
	}

	c.logger.Info("Inbound peer authenticated")

	// Continue with Bitcoin protocol handshake
	// This would be implemented similarly to outbound connections
	// For now, we'll just log the successful authentication
}

func (c *Client) handleBlock(block *wire.MsgBlock) {
	if c.stopped.Load() {
		return
	}

	c.logger.Info("Received block",
		zap.String("hash", block.BlockHash().String()),
		zap.Int("tx_count", len(block.Transactions)))

	blockEvent := blocks.BlockEvent{
		Hash:      block.BlockHash().String(),
		Height:    0, // We'd need to track height separately
		Timestamp: block.Header.Timestamp,
		Source:    "p2p-authenticated",
	}

	select {
	case c.blockChan <- blockEvent:
		// Block sent successfully
	default:
		// Channel full, skip
		c.logger.Warn("Block channel full, skipping block")
	}
}

func (c *Client) handleInv(p *peer.Peer, msg *wire.MsgInv) {
	// Request blocks when we receive inventory messages
	getData := wire.NewMsgGetData()

	for _, inv := range msg.InvList {
		if inv.Type == wire.InvTypeBlock {
			getData.AddInvVect(inv)
		}
	}

	if len(getData.InvList) > 0 {
		p.QueueMessage(getData, nil)
	}
}

func (c *Client) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"active_peers":         atomic.LoadInt32(&c.activePeers),
		"total_peers":          len(c.peers),
		"authentication":       "HMAC-SHA256",
		"handshake_protection": "replay+timestamp",
	}
}

// StartListener starts listening for inbound peer connections
func (c *Client) StartListener(port int) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to start peer listener: %w", err)
	}

	c.logger.Info("Started secure peer listener", zap.Int("port", port))

	go func() {
		defer listener.Close()
		for !c.stopped.Load() {
			conn, err := listener.Accept()
			if err != nil {
				if !c.stopped.Load() {
					c.logger.Warn("Failed to accept connection", zap.Error(err))
				}
				continue
			}

			go c.handleInboundConnection(conn)
		}
	}()

	return nil
}
