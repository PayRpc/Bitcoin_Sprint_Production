package main

import (
	"fmt"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
)

func main() {
	cfg := config.Load()

	fmt.Println("=== RPC Configuration Test ===")
	fmt.Printf("RPC Enabled: %v\n", cfg.RPCEnabled)
	fmt.Printf("RPC URL: %s\n", cfg.RPCURL)
	fmt.Printf("RPC Username: %s\n", cfg.RPCUsername)
	fmt.Printf("RPC Password: %s\n", cfg.RPCPassword)
	fmt.Printf("RPC Timeout: %v\n", cfg.RPCTimeout)
	fmt.Printf("RPC Batch Size: %d\n", cfg.RPCBatchSize)
	fmt.Printf("RPC Workers: %d\n", cfg.RPCWorkers)
	fmt.Printf("RPC Retry Attempts: %d\n", cfg.RPCRetryAttempts)
	fmt.Printf("RPC Retry Max Wait: %v\n", cfg.RPCRetryMaxWait)
	fmt.Printf("RPC Skip Mempool: %v\n", cfg.RPCSkipMempool)
	fmt.Printf("RPC Failed TX File: %s\n", cfg.RPCFailedTxFile)
	fmt.Printf("RPC Last ID File: %s\n", cfg.RPCLastIDFile)
	fmt.Printf("RPC Message Topic: %s\n", cfg.RPCMessageTopic)
}
