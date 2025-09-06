package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/cmd/p2p/diagnostics"
	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/p2p"
	"go.uber.org/zap"
)

func main() {
	// Create logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Create diagnostics recorder
	recorder := diagnostics.NewRecorder(1000, logger)
	defer recorder.Close()

	ctx := context.Background()

	// Example 1: Enhanced P2P Client with Diagnostics
	enhancedClient := NewEnhancedP2PClient(recorder, logger)

	// Simulate some P2P operations with diagnostics
	enhancedClient.simulateP2POperations(ctx)

	// Example 2: Network Health Integration
	networkHealth := NewNetworkHealthWithDiagnostics(recorder, logger)
	networkHealth.monitorNetworkHealth(ctx)

	// Example 3: Performance Monitoring
	performanceMonitor := NewPerformanceMonitor(recorder, logger)
	performanceMonitor.monitorPerformance(ctx)

	// Get comprehensive diagnostics report
	report := generateDiagnosticsReport(ctx, recorder)
	fmt.Println("=== DIAGNOSTICS REPORT ===")
	fmt.Println(report)

	logger.Info("Enhanced P2P diagnostics example completed!")
}

// EnhancedP2PClient demonstrates integration with existing P2P client
type EnhancedP2PClient struct {
	diagnostics *diagnostics.Recorder
	logger      *zap.Logger
}

func NewEnhancedP2PClient(recorder *diagnostics.Recorder, logger *zap.Logger) *EnhancedP2PClient {
	return &EnhancedP2PClient{
		diagnostics: recorder,
		logger:      logger,
	}
}

func (c *EnhancedP2PClient) simulateP2POperations(ctx context.Context) {
	c.logger.Info("Starting enhanced P2P operations with diagnostics")

	// Simulate peer connections
	peers := []string{"peer1", "peer2", "peer3"}
	for _, peer := range peers {
		c.connectToPeer(ctx, peer, "127.0.0.1:8333")
		time.Sleep(100 * time.Millisecond)
	}

	// Simulate message exchanges
	for _, peer := range peers {
		c.handleMessage(ctx, peer, "version", "inbound", 150)
		c.handleMessage(ctx, peer, "ping", "outbound", 32)
		time.Sleep(50 * time.Millisecond)
	}

	// Simulate some errors
	c.handleError(ctx, "peer1", fmt.Errorf("connection timeout"), "network")
	c.handleError(ctx, "peer2", fmt.Errorf("invalid handshake"), "protocol")

	// Simulate disconnections
	for _, peer := range peers {
		c.disconnectPeer(ctx, peer, "normal_shutdown")
		time.Sleep(100 * time.Millisecond)
	}
}

func (c *EnhancedP2PClient) connectToPeer(ctx context.Context, peerID, address string) {
	// Record the connection attempt
	if err := c.diagnostics.RecordPeerConnection(ctx, peerID, address); err != nil {
		c.logger.Error("Failed to record peer connection", zap.Error(err))
		return
	}

	c.logger.Info("Peer connected with diagnostics",
		zap.String("peer", peerID),
		zap.String("address", address))
}

func (c *EnhancedP2PClient) disconnectPeer(ctx context.Context, peerID, reason string) {
	// Record the disconnection
	if err := c.diagnostics.RecordPeerDisconnection(ctx, peerID, "127.0.0.1:8333", reason); err != nil {
		c.logger.Error("Failed to record peer disconnection", zap.Error(err))
		return
	}

	c.logger.Info("Peer disconnected with diagnostics",
		zap.String("peer", peerID),
		zap.String("reason", reason))
}

func (c *EnhancedP2PClient) handleMessage(ctx context.Context, peerID, msgType, direction string, size int) {
	// Record the message
	if err := c.diagnostics.RecordMessage(ctx, peerID, msgType, direction, size); err != nil {
		c.logger.Error("Failed to record message", zap.Error(err))
		return
	}

	c.logger.Debug("Message handled with diagnostics",
		zap.String("peer", peerID),
		zap.String("type", msgType),
		zap.String("direction", direction),
		zap.Int("size", size))
}

func (c *EnhancedP2PClient) handleError(ctx context.Context, peerID string, err error, operation string) {
	// Record the error
	if recordErr := c.diagnostics.RecordError(ctx, peerID, err, operation); recordErr != nil {
		c.logger.Error("Failed to record error", zap.Error(recordErr))
		return
	}

	c.logger.Warn("Error recorded in diagnostics",
		zap.String("peer", peerID),
		zap.String("operation", operation),
		zap.Error(err))
}

// NetworkHealthWithDiagnostics integrates with existing network health monitoring
type NetworkHealthWithDiagnostics struct {
	diagnostics *diagnostics.Recorder
	logger      *zap.Logger
}

func NewNetworkHealthWithDiagnostics(recorder *diagnostics.Recorder, logger *zap.Logger) *NetworkHealthWithDiagnostics {
	return &NetworkHealthWithDiagnostics{
		diagnostics: recorder,
		logger:      logger,
	}
}

func (nh *NetworkHealthWithDiagnostics) monitorNetworkHealth(ctx context.Context) {
	nh.logger.Info("Starting network health monitoring with diagnostics")

	// Simulate network health checks
	go nh.periodicHealthCheck(ctx)

	// Simulate block reception
	go nh.simulateBlockReception(ctx)

	time.Sleep(2 * time.Second) // Let monitoring run briefly
}

func (nh *NetworkHealthWithDiagnostics) periodicHealthCheck(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Record network health check
			event := &diagnostics.DiagnosticEvent{
				EventType: "network_health_check",
				Message:   "Periodic network health assessment",
				Severity:  diagnostics.SeverityInfo,
				Metadata: map[string]interface{}{
					"check_type": "connectivity",
					"peer_count": 8,
					"latency_ms": 45,
				},
			}

			if err := nh.diagnostics.RecordEvent(ctx, event); err != nil {
				nh.logger.Error("Failed to record health check", zap.Error(err))
			}
		}
	}
}

func (nh *NetworkHealthWithDiagnostics) simulateBlockReception(ctx context.Context) {
	time.Sleep(1 * time.Second)

	// Record block reception
	event := &diagnostics.DiagnosticEvent{
		EventType: "block_received",
		Message:   "New block received from network",
		Severity:  diagnostics.SeverityInfo,
		Metadata: map[string]interface{}{
			"block_hash":   "0000000000000000000123456789abcdef",
			"block_height": 850000,
			"tx_count":     2500,
			"block_size":   1250000,
		},
	}

	if err := nh.diagnostics.RecordEvent(ctx, event); err != nil {
		nh.logger.Error("Failed to record block reception", zap.Error(err))
	}
}

// PerformanceMonitor demonstrates performance diagnostics
type PerformanceMonitor struct {
	diagnostics *diagnostics.Recorder
	logger      *zap.Logger
}

func NewPerformanceMonitor(recorder *diagnostics.Recorder, logger *zap.Logger) *PerformanceMonitor {
	return &PerformanceMonitor{
		diagnostics: recorder,
		logger:      logger,
	}
}

func (pm *PerformanceMonitor) monitorPerformance(ctx context.Context) {
	pm.logger.Info("Starting performance monitoring with diagnostics")

	// Simulate performance metrics
	metrics := []struct {
		operation string
		duration  time.Duration
		success   bool
	}{
		{"block_validation", 120 * time.Millisecond, true},
		{"transaction_verification", 45 * time.Millisecond, true},
		{"mempool_update", 15 * time.Millisecond, true},
		{"peer_sync", 2000 * time.Millisecond, false}, // Slow operation
	}

	for _, metric := range metrics {
		pm.recordPerformanceMetric(ctx, metric.operation, metric.duration, metric.success)
		time.Sleep(100 * time.Millisecond)
	}
}

func (pm *PerformanceMonitor) recordPerformanceMetric(ctx context.Context, operation string, duration time.Duration, success bool) {
	severity := diagnostics.SeverityInfo
	message := fmt.Sprintf("Performance: %s completed in %v", operation, duration)

	if !success {
		severity = diagnostics.SeverityWarning
		message = fmt.Sprintf("Performance: %s failed after %v", operation, duration)
	} else if duration > 1000*time.Millisecond {
		severity = diagnostics.SeverityWarning
		message = fmt.Sprintf("Performance: %s is slow (%v)", operation, duration)
	}

	event := &diagnostics.DiagnosticEvent{
		EventType: "performance_metric",
		Message:   message,
		Severity:  severity,
		Metadata: map[string]interface{}{
			"operation":   operation,
			"duration_ms": duration.Milliseconds(),
			"success":     success,
		},
	}

	if err := pm.diagnostics.RecordEvent(ctx, event); err != nil {
		pm.logger.Error("Failed to record performance metric", zap.Error(err))
	}
}

// generateDiagnosticsReport creates a comprehensive diagnostics report
func generateDiagnosticsReport(ctx context.Context, recorder *diagnostics.Recorder) string {
	report := "Bitcoin Sprint P2P Diagnostics Report\n"
	report += "===================================\n\n"

	// Get statistics
	stats, err := recorder.GetStats(ctx)
	if err != nil {
		return fmt.Sprintf("Error getting stats: %v", err)
	}

	report += fmt.Sprintf("Total Events Recorded: %d\n", stats.TotalEvents)
	report += fmt.Sprintf("Active Peers: %d\n", stats.ActivePeers)
	report += fmt.Sprintf("Error Rate: %.2f%%\n\n", stats.ErrorRate*100)

	// Events by type
	report += "Events by Type:\n"
	for eventType, count := range stats.EventsByType {
		report += fmt.Sprintf("  %s: %d\n", eventType, count)
	}
	report += "\n"

	// Events by severity
	report += "Events by Severity:\n"
	for severity, count := range stats.EventsBySeverity {
		report += fmt.Sprintf("  %s: %d\n", severity.String(), count)
	}
	report += "\n"

	// Recent events
	events, err := recorder.GetEvents(ctx, 10, diagnostics.SeverityDebug)
	if err == nil && len(events) > 0 {
		report += "Recent Events:\n"
		for i, event := range events {
			if i >= 5 { // Show only last 5
				break
			}
			report += fmt.Sprintf("  [%s] %s: %s\n",
				event.Timestamp.Format("15:04:05"),
				event.Severity.String(),
				event.Message)
		}
	}

	return report
}
