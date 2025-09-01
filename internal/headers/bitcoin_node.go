// internal/headers/bitcoin_node.go
package headers

import (
	"context"
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

// NewBitcoinNode creates a new Bitcoin node client
func NewBitcoinNode(rpcURL, rpcUser, rpcPass string) *BitcoinNode {
	return &BitcoinNode{
		rpcURL:  rpcURL,
		rpcUser: rpcUser,
		rpcPass: rpcPass,
		client: &http.Client{
			Timeout: 10 * time.Second,
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
