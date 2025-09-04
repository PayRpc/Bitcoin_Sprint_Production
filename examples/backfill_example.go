//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/messaging"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("Failed to initialize logger", err)
	}
	defer logger.Sync()

	fmt.Println("=== Bitcoin Sprint Backfill Example ===")

	// Load configuration
	cfg := config.Load()

	// Check if RPC is enabled
	if !cfg.RPCEnabled {
		fmt.Println("❌ RPC backfill is disabled. Enable with RPC_ENABLED=true")
		fmt.Println("   Set the following environment variables:")
		fmt.Println("   - RPC_ENABLED=true")
		fmt.Println("   - RPC_URL=http://127.0.0.1:8332")
		fmt.Println("   - RPC_USERNAME=sprint")
		fmt.Println("   - RPC_PASSWORD=sprint_password_2025")
		return
	}

	// Create backfill service
	backfillService := messaging.NewBackfillService(cfg, nil, nil, logger)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	fmt.Println("🚀 Starting one-time backfill operation...")
	fmt.Printf("   RPC URL: %s\n", cfg.RPCURL)
	fmt.Printf("   Batch Size: %d\n", cfg.RPCBatchSize)
	fmt.Printf("   Workers: %d\n", cfg.RPCWorkers)
	fmt.Println()

	startTime := time.Now()

	// Run one-time backfill
	messages, lastID, failedTxs, err := backfillService.RunOnce(ctx)
	if err != nil {
		fmt.Printf("❌ Backfill failed: %v\n", err)
		return
	}

	duration := time.Since(startTime)

	// Display results
	fmt.Println("✅ Backfill completed successfully!")
	fmt.Println()
	fmt.Printf("📊 Results:\n")
	fmt.Printf("   Messages Processed: %d\n", len(messages))
	fmt.Printf("   Last Block ID: %s\n", lastID)
	fmt.Printf("   Failed Transactions: %d\n", len(failedTxs))
	fmt.Printf("   Duration: %.2f seconds\n", duration.Seconds())
	fmt.Printf("   Messages/second: %.1f\n", float64(len(messages))/duration.Seconds())

	if len(messages) > 0 {
		fmt.Println()
		fmt.Println("📝 Sample Messages:")
		for i, msg := range messages {
			if i >= 5 { // Show only first 5
				break
			}
			fmt.Printf("   %d. TX: %s | Block: %s | Time: %s\n",
				i+1, msg.ID[:16]+"...", msg.BlockHash[:16]+"...", msg.Timestamp.Format("15:04:05"))
		}
	}

	if len(failedTxs) > 0 {
		fmt.Println()
		fmt.Println("⚠️  Failed Transactions (first 5):")
		for i, txid := range failedTxs {
			if i >= 5 {
				break
			}
			fmt.Printf("   %s\n", txid)
		}
	}

	fmt.Println()
	fmt.Println("💡 Tips:")
	fmt.Println("   - Use RPC_FAILED_TX_FILE to persist failed transactions")
	fmt.Println("   - Use RPC_LAST_ID_FILE to resume from last processed block")
	fmt.Println("   - Adjust RPC_BATCH_SIZE for optimal performance")
	fmt.Println("   - Monitor with Prometheus metrics at /metrics")
}
