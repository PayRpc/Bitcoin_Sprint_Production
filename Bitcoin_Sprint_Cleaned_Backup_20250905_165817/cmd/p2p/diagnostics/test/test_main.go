package main

import (
	"context"
	"fmt"
	"log"

	"github.com/PayRpc/Bitcoin-Sprint/cmd/p2p/diagnostics"
	"go.uber.org/zap"
)

func main() {
	// Create logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("Failed to create logger:", err)
	}
	defer logger.Sync()

	// Create diagnostics recorder
	recorder := diagnostics.NewRecorder(1000, logger)
	defer recorder.Close()

	ctx := context.Background()

	fmt.Println("=== Testing Basic Event Recording ===")

	// Test basic event recording
	err = recorder.RecordPeerConnection(ctx, "peer1", "192.168.1.100:8333")
	if err != nil {
		fmt.Printf("Failed to record peer connection: %v\n", err)
		return
	}
	fmt.Println("✓ Recorded peer connection")

	// Test message recording
	err = recorder.RecordMessage(ctx, "peer1", "version", "outbound", 150)
	if err != nil {
		fmt.Printf("Failed to record message: %v\n", err)
		return
	}
	fmt.Println("✓ Recorded message")

	// Test error recording
	testErr := fmt.Errorf("connection timeout")
	err = recorder.RecordError(ctx, "peer1", testErr, "network")
	if err != nil {
		fmt.Printf("Failed to record error: %v\n", err)
		return
	}
	fmt.Println("✓ Recorded error")

	fmt.Println("\n=== Testing Statistics ===")

	// Get statistics
	stats, err := recorder.GetStats(ctx)
	if err != nil {
		fmt.Printf("Failed to get stats: %v\n", err)
		return
	}

	fmt.Printf("Total Events: %d\n", stats.TotalEvents)
	fmt.Printf("Active Peers: %d\n", stats.ActivePeers)
	fmt.Printf("Error Rate: %.2f%%\n", stats.ErrorRate*100)

	fmt.Println("\n=== Testing Health Check ===")

	// Test health check
	health, err := recorder.HealthCheck(ctx)
	if err != nil {
		fmt.Printf("Failed to get health: %v\n", err)
		return
	}

	fmt.Printf("Health Status: %s\n", health.Status)
	fmt.Printf("Message: %s\n", health.Message)
	fmt.Printf("Event Count: %d/%d\n", health.EventCount, health.MaxEvents)

	fmt.Println("\n=== Testing Event Retrieval ===")

	// Get recent events
	events, err := recorder.GetEvents(ctx, 10, diagnostics.SeverityDebug)
	if err != nil {
		fmt.Printf("Failed to get events: %v\n", err)
		return
	}

	fmt.Printf("Retrieved %d events\n", len(events))
	for i, event := range events {
		fmt.Printf("  %d. [%s] %s: %s\n", i+1, event.Severity.String(), event.EventType, event.Message)
	}

	fmt.Println("\n=== Testing Data Export ===")

	// Test data export
	exportData, err := recorder.ExportEvents(ctx)
	if err != nil {
		fmt.Printf("Failed to export events: %v\n", err)
		return
	}

	fmt.Printf("Exported %d bytes of data\n", len(exportData))

	fmt.Println("\n=== Testing Configuration ===")

	// Test custom configuration (using default for now since NewRecorderWithConfig doesn't exist)
	config := diagnostics.DefaultRecorderConfig()
	fmt.Printf("Default MaxEvents: %d\n", config.MaxEvents)
	fmt.Printf("Default CleanupInterval: %v\n", config.CleanupInterval)
	fmt.Printf("Default EventRetention: %v\n", config.EventRetention)

	fmt.Println("✓ Retrieved default configuration")

	fmt.Println("\n=== All Tests Passed! ===")
	fmt.Println("The diagnostics module is working correctly.")
}
