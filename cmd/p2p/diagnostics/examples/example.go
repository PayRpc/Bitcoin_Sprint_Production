package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/cmd/p2p/diagnostics"
	"go.uber.org/zap"
)

func main() {
	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("Failed to create logger:", err)
	}
	defer logger.Sync()

	// Create a diagnostics recorder with max 100 events
	recorder := diagnostics.NewRecorder(100, logger)
	defer recorder.Close()

	ctx := context.Background()

	// Example 1: Record a peer connection
	err = recorder.RecordPeerConnection(ctx, "peer123", "192.168.1.100:8333")
	if err != nil {
		logger.Error("Failed to record peer connection", zap.Error(err))
	}

	// Example 2: Record a message exchange
	err = recorder.RecordMessage(ctx, "peer123", "version", "outbound", 1024)
	if err != nil {
		logger.Error("Failed to record message", zap.Error(err))
	}

	// Example 3: Record an error
	testErr := fmt.Errorf("connection timeout")
	err = recorder.RecordError(ctx, "peer123", testErr, "handshake")
	if err != nil {
		logger.Error("Failed to record error", zap.Error(err))
	}

	// Example 4: Get recent events
	events, err := recorder.GetEvents(ctx, 10, diagnostics.SeverityDebug)
	if err != nil {
		logger.Error("Failed to get events", zap.Error(err))
	} else {
		logger.Info("Retrieved events", zap.Int("count", len(events)))
		for i, event := range events {
			logger.Info("Event",
				zap.Int("index", i),
				zap.String("type", event.EventType),
				zap.String("message", event.Message),
				zap.String("severity", event.Severity.String()),
			)
		}
	}

	// Example 5: Get statistics
	stats, err := recorder.GetStats(ctx)
	if err != nil {
		logger.Error("Failed to get stats", zap.Error(err))
	} else {
		logger.Info("Diagnostics Statistics",
			zap.Int64("total_events", stats.TotalEvents),
			zap.Int("active_peers", stats.ActivePeers),
		)
	}

	// Wait a bit to see cleanup in action
	logger.Info("Waiting for cleanup routine...")
	time.Sleep(2 * time.Second)

	// Check events again
	events, err = recorder.GetEvents(ctx, 10, diagnostics.SeverityDebug)
	if err != nil {
		logger.Error("Failed to get events after cleanup", zap.Error(err))
	} else {
		logger.Info("Events after cleanup", zap.Int("count", len(events)))
	}

	logger.Info("Diagnostics example completed successfully!")
}
