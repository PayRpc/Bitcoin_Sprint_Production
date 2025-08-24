//go:build !demo

package main

import (
"log"
"time"
)

// StartBlockPoller (Production Version) - Polls real Bitcoin Core nodes
func (s *Sprint) StartBlockPoller() {
var lastHash string
var consecutiveErrors int

log.Printf("Starting Bitcoin Core polling with %d nodes", len(s.config.RPCNodes))

// Default to 1s interval if not specified for optimal performance
interval := time.Duration(s.config.PollInterval) * time.Second
if interval == 0 {
interval = 1 * time.Second // ultra-fast default for better performance
}

ticker := time.NewTicker(interval)
defer ticker.Stop()

for {
select {
case <-s.ctx.Done():
return
case <-ticker.C:
var (
hash   string
height int
node   string
err    error
)

if s.config.TurboMode {
hash, height, node, err = s.getBestBlockTurbo()
} else {
hash, height, node, err = s.getBestBlock()
}

if err != nil {
consecutiveErrors++
if consecutiveErrors > 5 {
log.Printf("❌ Multiple consecutive errors: %v", err)
}
continue
}

consecutiveErrors = 0

// Check if we have a new block
if hash != "" && hash != lastHash {
s.OnNewBlock(hash, height, node)
lastHash = hash
}
}
}
}
