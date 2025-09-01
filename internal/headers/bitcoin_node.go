// internal/headers/bitcoin_node.go
package headers

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// BitcoinNode implements the Node interface for Bitcoin Core RPC
type BitcoinNode struct {
	rpcURL  string
	rpcUser string
	rpcPass string
	client  *http.Client
}

// NewBitcoinNode creates a new Bitcoin node client with optimized connection settings
func NewBitcoinNode(rpcURL, rpcUser, rpcPass string) *BitcoinNode {
	// Create optimized transport for Bitcoin RPC calls
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     false, // Bitcoin RPC typically uses HTTP/1.1
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			// Allow self-signed certificates for local Bitcoin nodes
			InsecureSkipVerify: true,
		},
	}

	return &BitcoinNode{
		rpcURL:  rpcURL,
		rpcUser: rpcUser,
		rpcPass: rpcPass,
		client: &http.Client{
			Timeout:   15 * time.Second, // Longer timeout for RPC calls
			Transport: transport,
		},
	}
}

// GetBlockHeader fetches a Bitcoin block header by height
func (bn *BitcoinNode) GetBlockHeader(ctx context.Context, height int) (Header, error) {
	// For now, return a mock header - in production you'd implement actual RPC calls
	// This is a placeholder implementation

	// Mock header data (80 bytes for Bitcoin)
	mockHeader := make([]byte, 80)
	for i := range mockHeader {
		mockHeader[i] = byte(i % 256)
	}

	// Compute hash from mock header
	hash := DoubleSHA256(mockHeader)

	return Header{
		Hash:   hash,
		Height: uint32(height),
		Raw:    mockHeader,
	}, nil
}

// MockNode provides a simple mock implementation for testing
type MockNode struct {
	baseHeight int
}

// NewMockNode creates a mock node for testing
func NewMockNode(baseHeight int) *MockNode {
	return &MockNode{baseHeight: baseHeight}
}

// GetBlockHeader returns mock header data
func (mn *MockNode) GetBlockHeader(ctx context.Context, height int) (Header, error) {
	// Simulate some processing time
	select {
	case <-time.After(50 * time.Millisecond):
	case <-ctx.Done():
		return Header{}, ctx.Err()
	}

	// Create deterministic mock data based on height
	mockHeader := make([]byte, 80)
	for i := range mockHeader {
		mockHeader[i] = byte((height + i) % 256)
	}

	hash := DoubleSHA256(mockHeader)

	return Header{
		Hash:   hash,
		Height: uint32(height),
		Raw:    mockHeader,
	}, nil
}
