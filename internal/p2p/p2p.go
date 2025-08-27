package p2p

import (
"context"
"fmt"
"time"

"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
"github.com/PayRpc/Bitcoin-Sprint/internal/config"
"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
"go.uber.org/zap"
)

// Client manages P2P connections
type Client struct {
logger    *zap.Logger
blockChan chan blocks.BlockEvent
mempool   *mempool.Mempool
config    config.Config
ctx       context.Context
cancel    context.CancelFunc
}

// New creates a P2P client 
func New(cfg config.Config, blockChan chan blocks.BlockEvent, mem *mempool.Mempool, logger *zap.Logger) (*Client, error) {
ctx, cancel := context.WithCancel(context.Background())
return &Client{
logger:    logger,
blockChan: blockChan,
mempool:   mem,
config:    cfg,
ctx:       ctx,
cancel:    cancel,
}, nil
}

// Run starts the P2P client
func (c *Client) Run() {
c.logger.Info("P2P client starting")
ticker := time.NewTicker(30 * time.Second)
defer ticker.Stop()

for {
select {
case <-c.ctx.Done():
c.logger.Info("P2P client stopped")
return
case t := <-ticker.C:
c.blockChan <- blocks.BlockEvent{
Hash:      fmt.Sprintf("simulated_block_%d", t.Unix()),
Height:    0,
Timestamp: t,
Source:    "p2p_simulation",
}
}
}
}

// Stop halts the P2P client
func (c *Client) Stop() {
c.cancel()
}

// NewDirect creates a direct P2P connection
func NewDirect(ctx context.Context, addr string, blockChan chan blocks.BlockEvent, logger *zap.Logger) error {
logger.Info("Starting direct P2P connection", zap.String("addr", addr))

go func() {
ticker := time.NewTicker(30 * time.Second)
defer ticker.Stop()

for {
select {
case <-ctx.Done():
logger.Info("Direct P2P connection stopped")
return
case t := <-ticker.C:
blockChan <- blocks.BlockEvent{
Hash:      fmt.Sprintf("direct_block_%d", t.Unix()),
Height:    0,
Timestamp: t,
Source:    "direct_p2p",
}
}
}
}()

return nil
}

// NewMemoryWatcher creates a memory watcher
func NewMemoryWatcher(ctx context.Context, blockChan chan blocks.BlockEvent, logger *zap.Logger) {
logger.Info("Starting memory watcher")

go func() {
ticker := time.NewTicker(10 * time.Second)
defer ticker.Stop()

for {
select {
case <-ctx.Done():
logger.Info("Memory watcher stopped")
return
case <-ticker.C:
// Monitor memory and send events if needed
logger.Debug("Memory watcher tick")
}
}
}()
}
