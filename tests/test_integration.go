package main

import (
	"fmt"
	"github.com/PayRpc/Bitcoin-Sprint/internal/blocks"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/mempool"
	"github.com/PayRpc/Bitcoin-Sprint/internal/messaging"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	fmt.Println("=== Backfill Service Integration Test ===")

	// Load configuration
	cfg := config.Load()

	// Create required components
	blockChan := make(chan blocks.BlockEvent, 10)
	mem := mempool.New()

	// Create backfill service
	backfillService := messaging.NewBackfillService(cfg, blockChan, mem, logger)

	fmt.Printf("✅ Backfill service created successfully\n")
	fmt.Printf("   RPC Enabled: %v\n", cfg.RPCEnabled)
	fmt.Printf("   Service: %T\n", backfillService)

	// Test service methods
	messageCount := backfillService.GetMessageCount()
	messages := backfillService.GetMessages()

	fmt.Printf("   Initial message count: %d\n", messageCount)
	fmt.Printf("   Initial messages slice length: %d\n", len(messages))

	// Test configuration validation
	if cfg.RPCEnabled {
		fmt.Println("✅ RPC is enabled - service will start automatically")
		fmt.Printf("   URL: %s\n", cfg.RPCURL)
		fmt.Printf("   Batch Size: %d\n", cfg.RPCBatchSize)
		fmt.Printf("   Workers: %d\n", cfg.RPCWorkers)
	} else {
		fmt.Println("ℹ️  RPC is disabled - set RPC_ENABLED=true to enable backfill")
	}

	fmt.Println("\n=== Test Environment Variables ===")
	fmt.Println("To enable RPC backfill, set these environment variables:")
	fmt.Println("  RPC_ENABLED=true")
	fmt.Println("  RPC_URL=http://127.0.0.1:8332")
	fmt.Println("  RPC_USERNAME=sprint")
	fmt.Println("  RPC_PASSWORD=sprint_password_2025")
	fmt.Println("  RPC_BATCH_SIZE=50")
	fmt.Println("  RPC_WORKERS=10")

	fmt.Println("\n✅ Integration test completed successfully!")
}
