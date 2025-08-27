//go:build nozmq
// +build nozmq

package zmq

import (
"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
"github.com/PayRpc/Bitcoin-Sprint/internal/config"
"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
"go.uber.org/zap"
)

// Client is a mock ZMQ client used when libzmq is not available
type Client struct {
logger *zap.Logger
}

// New returns a mock ZMQ client
func New(cfg config.Config, blockChan chan blocks.BlockEvent, mem *mempool.Mempool, logger *zap.Logger) *Client {
logger.Info("ZMQ disabled: running in mock mode (nozmq build tag)")
return &Client{logger: logger}
}

// Run starts the mock ZMQ client
func (c *Client) Run() {
c.logger.Info("Mock ZMQ client running (nozmq mode)")
}

// Stop stops the mock client
func (c *Client) Stop() {
c.logger.Info("Mock ZMQ client stopped (nozmq mode)")
}
