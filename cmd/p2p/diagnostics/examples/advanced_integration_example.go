package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/cmd/p2p/diagnostics"
	"go.uber.org/zap"
)

// AdvancedP2PClient demonstrates advanced diagnostics integration patterns
type AdvancedP2PClient struct {
	diagnostics *diagnostics.Recorder
	logger      *zap.Logger
	config      *diagnostics.RecorderConfig
	healthChan  chan diagnostics.HealthStatus
}

// NewAdvancedP2PClient creates a new advanced P2P client with full diagnostics integration
func NewAdvancedP2PClient(logger *zap.Logger) *AdvancedP2PClient {
	// Create custom configuration
	config := &diagnostics.RecorderConfig{
		MaxEvents:         10000,
		CleanupInterval:   5 * time.Minute,
		RetentionPeriod:   24 * time.Hour,
		EnableMetrics:     true,
		EnableHealthCheck: true,
		LogLevel:          "info",
		ExportFormat:      "json",
	}

	recorder := diagnostics.NewRecorderWithConfig(config, logger)

	return &AdvancedP2PClient{
		diagnostics: recorder,
		logger:      logger,
		config:      config,
		healthChan:  make(chan diagnostics.HealthStatus, 10),
	}
}

// StartHealthMonitoring starts continuous health monitoring
func (c *AdvancedP2PClient) StartHealthMonitoring(ctx context.Context) {
	go c.healthMonitor(ctx)
	go c.periodicHealthCheck(ctx)
}

// healthMonitor listens for health status updates
func (c *AdvancedP2PClient) healthMonitor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case health := <-c.healthChan:
			c.handleHealthUpdate(ctx, health)
		}
	}
}

// periodicHealthCheck performs regular health assessments
func (c *AdvancedP2PClient) periodicHealthCheck(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			health, err := c.diagnostics.HealthCheck(ctx)
			if err != nil {
				c.logger.Error("Health check failed", zap.Error(err))
				continue
			}

			select {
			case c.healthChan <- *health:
			default:
				c.logger.Warn("Health channel full, dropping health update")
			}
		}
	}
}

// handleHealthUpdate processes health status updates
func (c *AdvancedP2PClient) handleHealthUpdate(ctx context.Context, health diagnostics.HealthStatus) {
	c.logger.Info("Health status update",
		zap.String("status", health.Status),
		zap.String("message", health.Message),
		zap.Int("event_count", health.EventCount),
		zap.Int64("total_events", health.TotalEvents))

	// Alert on critical health issues
	if health.Status == "error" || health.Status == "critical" {
		c.diagnostics.RecordEvent(ctx, &diagnostics.DiagnosticEvent{
			EventType: "health_critical",
			Message:   fmt.Sprintf("Critical health issue: %s", health.Message),
			Severity:  diagnostics.SeverityCritical,
			Metadata: map[string]interface{}{
				"health_status": health.Status,
				"event_count":   health.EventCount,
				"max_events":    health.MaxEvents,
			},
		})
	}
}

// SimulateComplexP2POperations demonstrates complex P2P operations with diagnostics
func (c *AdvancedP2PClient) SimulateComplexP2POperations(ctx context.Context) {
	c.logger.Info("Starting complex P2P operations simulation")

	// Phase 1: Initial peer discovery
	c.simulatePeerDiscovery(ctx)

	// Phase 2: Connection establishment with retry logic
	c.simulateConnectionEstablishment(ctx)

	// Phase 3: Message exchange patterns
	c.simulateMessageExchange(ctx)

	// Phase 4: Error scenarios and recovery
	c.simulateErrorScenarios(ctx)

	// Phase 5: Performance testing
	c.simulatePerformanceTest(ctx)

	// Phase 6: Graceful shutdown
	c.simulateGracefulShutdown(ctx)
}

func (c *AdvancedP2PClient) simulatePeerDiscovery(ctx context.Context) {
	c.logger.Info("Phase 1: Peer discovery")

	peers := []string{"peer1", "peer2", "peer3", "peer4", "peer5"}
	addresses := []string{
		"192.168.1.100:8333",
		"192.168.1.101:8333",
		"192.168.1.102:8333",
		"192.168.1.103:8333",
		"192.168.1.104:8333",
	}

	for i, peer := range peers {
		// Record peer discovery
		c.diagnostics.RecordEvent(ctx, &diagnostics.DiagnosticEvent{
			EventType: "peer_discovered",
			PeerID:    peer,
			Message:   fmt.Sprintf("Discovered peer %s at %s", peer, addresses[i]),
			Severity:  diagnostics.SeverityInfo,
			Metadata: map[string]interface{}{
				"discovery_method": "dns_seed",
				"address":          addresses[i],
				"services":         []string{"NODE_NETWORK", "NODE_WITNESS"},
			},
		})

		time.Sleep(200 * time.Millisecond)
	}
}

func (c *AdvancedP2PClient) simulateConnectionEstablishment(ctx context.Context) {
	c.logger.Info("Phase 2: Connection establishment")

	peers := []string{"peer1", "peer2", "peer3", "peer4", "peer5"}
	addresses := []string{
		"192.168.1.100:8333",
		"192.168.1.101:8333",
		"192.168.1.102:8333",
		"192.168.1.103:8333",
		"192.168.1.104:8333",
	}

	for i, peer := range peers {
		// Simulate connection with retry logic
		maxRetries := 3
		for attempt := 1; attempt <= maxRetries; attempt++ {
			c.diagnostics.RecordEvent(ctx, &diagnostics.DiagnosticEvent{
				EventType: "connection_attempt",
				PeerID:    peer,
				Message:   fmt.Sprintf("Connection attempt %d/%d to %s", attempt, maxRetries, addresses[i]),
				Severity:  diagnostics.SeverityDebug,
				Metadata: map[string]interface{}{
					"attempt":     attempt,
					"max_retries": maxRetries,
					"address":     addresses[i],
				},
			})

			// Simulate connection delay
			time.Sleep(300 * time.Millisecond)

			// Simulate occasional connection failures
			if attempt == 2 && peer == "peer3" {
				c.diagnostics.RecordError(ctx, peer, fmt.Errorf("connection refused"), "connection")
				continue
			}

			// Successful connection
			if err := c.diagnostics.RecordPeerConnection(ctx, peer, addresses[i]); err != nil {
				c.logger.Error("Failed to record peer connection", zap.Error(err))
			}
			break
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func (c *AdvancedP2PClient) simulateMessageExchange(ctx context.Context) {
	c.logger.Info("Phase 3: Message exchange patterns")

	peers := []string{"peer1", "peer2", "peer4", "peer5"} // peer3 failed to connect
	messagePatterns := []struct {
		messageType string
		direction   string
		size        int
		frequency   int
	}{
		{"version", "outbound", 150, 1},
		{"verack", "inbound", 24, 1},
		{"ping", "outbound", 32, 5},
		{"pong", "inbound", 32, 5},
		{"getblocks", "outbound", 1000, 2},
		{"inv", "inbound", 500, 10},
		{"getdata", "outbound", 500, 8},
		{"block", "inbound", 1000000, 3},
		{"tx", "inbound", 250, 15},
	}

	for _, pattern := range messagePatterns {
		for i := 0; i < pattern.frequency; i++ {
			peer := peers[i%len(peers)]

			if err := c.diagnostics.RecordMessage(ctx, peer, pattern.messageType, pattern.direction, pattern.size); err != nil {
				c.logger.Error("Failed to record message", zap.Error(err))
			}

			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (c *AdvancedP2PClient) simulateErrorScenarios(ctx context.Context) {
	c.logger.Info("Phase 4: Error scenarios and recovery")

	peers := []string{"peer1", "peer2", "peer4", "peer5"}
	errorScenarios := []struct {
		peer      string
		error     error
		operation string
		severity  diagnostics.Severity
	}{
		{"peer1", fmt.Errorf("invalid block header"), "block_validation", diagnostics.SeverityWarning},
		{"peer2", fmt.Errorf("connection timeout"), "network", diagnostics.SeverityError},
		{"peer4", fmt.Errorf("invalid transaction"), "tx_validation", diagnostics.SeverityWarning},
		{"peer5", fmt.Errorf("peer misbehaving"), "protocol", diagnostics.SeverityError},
	}

	for _, scenario := range errorScenarios {
		// Record the error
		if err := c.diagnostics.RecordError(ctx, scenario.peer, scenario.error, scenario.operation); err != nil {
			c.logger.Error("Failed to record error", zap.Error(err))
		}

		// Simulate recovery attempt
		time.Sleep(500 * time.Millisecond)

		c.diagnostics.RecordEvent(ctx, &diagnostics.DiagnosticEvent{
			EventType: "recovery_attempt",
			PeerID:    scenario.peer,
			Message:   fmt.Sprintf("Attempting recovery from %s error", scenario.operation),
			Severity:  diagnostics.SeverityInfo,
			Metadata: map[string]interface{}{
				"error_operation": scenario.operation,
				"recovery_type":   "reconnection",
			},
		})

		time.Sleep(300 * time.Millisecond)
	}
}

func (c *AdvancedP2PClient) simulatePerformanceTest(ctx context.Context) {
	c.logger.Info("Phase 5: Performance testing")

	// Simulate high-frequency operations
	start := time.Now()
	operations := 1000

	for i := 0; i < operations; i++ {
		peer := fmt.Sprintf("perf-peer-%d", i%10)
		messageType := "ping"

		if err := c.diagnostics.RecordMessage(ctx, peer, messageType, "outbound", 32); err != nil {
			c.logger.Error("Failed to record performance message", zap.Error(err))
		}
	}

	duration := time.Since(start)
	opsPerSecond := float64(operations) / duration.Seconds()

	c.diagnostics.RecordEvent(ctx, &diagnostics.DiagnosticEvent{
		EventType: "performance_test",
		Message:   fmt.Sprintf("Performance test completed: %d operations in %v (%.0f ops/sec)", operations, duration, opsPerSecond),
		Severity:  diagnostics.SeverityInfo,
		Metadata: map[string]interface{}{
			"operations":     operations,
			"duration_ms":    duration.Milliseconds(),
			"ops_per_second": opsPerSecond,
		},
	})
}

func (c *AdvancedP2PClient) simulateGracefulShutdown(ctx context.Context) {
	c.logger.Info("Phase 6: Graceful shutdown")

	peers := []string{"peer1", "peer2", "peer4", "peer5"}

	for _, peer := range peers {
		if err := c.diagnostics.RecordPeerDisconnection(ctx, peer, "127.0.0.1:8333", "shutdown"); err != nil {
			c.logger.Error("Failed to record peer disconnection", zap.Error(err))
		}

		time.Sleep(200 * time.Millisecond)
	}

	// Record shutdown event
	c.diagnostics.RecordEvent(ctx, &diagnostics.DiagnosticEvent{
		EventType: "system_shutdown",
		Message:   "P2P client shutting down gracefully",
		Severity:  diagnostics.SeverityInfo,
		Metadata: map[string]interface{}{
			"shutdown_type": "graceful",
			"peers_disconnected": len(peers),
		},
	})
}

// GenerateComprehensiveReport creates a detailed diagnostics report
func (c *AdvancedP2PClient) GenerateComprehensiveReport(ctx context.Context) error {
	report := &AdvancedDiagnosticReport{
		Timestamp: time.Now(),
	}

	// Get basic statistics
	stats, err := c.diagnostics.GetStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}
	report.Statistics = *stats

	// Get recent events
	events, err := c.diagnostics.GetEvents(ctx, 100, diagnostics.SeverityDebug)
	if err != nil {
		return fmt.Errorf("failed to get events: %w", err)
	}
	report.RecentEvents = events

	// Get health status
	health, err := c.diagnostics.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("failed to get health: %w", err)
	}
	report.HealthStatus = *health

	// Analyze performance
	report.PerformanceAnalysis = c.analyzePerformance(ctx, events)

	// Export to file
	return c.exportReport(report)
}

// AdvancedDiagnosticReport represents a comprehensive diagnostic report
type AdvancedDiagnosticReport struct {
	Timestamp           time.Time                    `json:"timestamp"`
	Statistics          diagnostics.DiagnosticStats `json:"statistics"`
	RecentEvents        []*diagnostics.DiagnosticEvent `json:"recent_events"`
	HealthStatus        diagnostics.HealthStatus    `json:"health_status"`
	PerformanceAnalysis PerformanceAnalysis         `json:"performance_analysis"`
}

// PerformanceAnalysis contains performance metrics and analysis
type PerformanceAnalysis struct {
	AverageEventRate float64           `json:"average_event_rate"`
	ErrorRate        float64           `json:"error_rate"`
	TopErrorTypes    map[string]int    `json:"top_error_types"`
	PeakEventPeriods []EventPeriod     `json:"peak_event_periods"`
}

// EventPeriod represents a period of high event activity
type EventPeriod struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	EventCount int      `json:"event_count"`
}

func (c *AdvancedP2PClient) analyzePerformance(ctx context.Context, events []*diagnostics.DiagnosticEvent) PerformanceAnalysis {
	analysis := PerformanceAnalysis{
		TopErrorTypes: make(map[string]int),
	}

	if len(events) == 0 {
		return analysis
	}

	// Calculate event rate
	duration := time.Since(events[len(events)-1].Timestamp)
	if duration > 0 {
		analysis.AverageEventRate = float64(len(events)) / duration.Seconds()
	}

	// Analyze errors
	errorCount := 0
	for _, event := range events {
		if event.Severity >= diagnostics.SeverityError {
			errorCount++
			if event.Metadata != nil {
				if operation, ok := event.Metadata["operation"].(string); ok {
					analysis.TopErrorTypes[operation]++
				}
			}
		}
	}

	if len(events) > 0 {
		analysis.ErrorRate = float64(errorCount) / float64(len(events))
	}

	// Find peak periods (simplified)
	if len(events) > 10 {
		analysis.PeakEventPeriods = []EventPeriod{
			{
				StartTime: events[0].Timestamp,
				EndTime:   events[len(events)-1].Timestamp,
				EventCount: len(events),
			},
		}
	}

	return analysis
}

func (c *AdvancedP2PClient) exportReport(report *AdvancedDiagnosticReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	filename := fmt.Sprintf("diagnostic_report_%s.json", time.Now().Format("20060102_150405"))
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	c.logger.Info("Diagnostic report exported",
		zap.String("filename", filename),
		zap.Int("file_size", len(data)))

	return nil
}

// Cleanup performs cleanup operations
func (c *AdvancedP2PClient) Cleanup() {
	if c.diagnostics != nil {
		c.diagnostics.Close()
	}
	close(c.healthChan)
}

func main() {
	// Create logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("Failed to create logger:", err)
	}
	defer logger.Sync()

	// Create advanced P2P client
	client := NewAdvancedP2PClient(logger)
	defer client.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Start health monitoring
	client.StartHealthMonitoring(ctx)

	// Run complex P2P operations simulation
	client.SimulateComplexP2POperations(ctx)

	// Generate comprehensive report
	if err := client.GenerateComprehensiveReport(ctx); err != nil {
		logger.Error("Failed to generate report", zap.Error(err))
	}

	// Export events data
	exportData, err := client.diagnostics.ExportEvents(ctx)
	if err != nil {
		logger.Error("Failed to export events", zap.Error(err))
	} else {
		logger.Info("Events exported successfully",
			zap.Int("data_size_bytes", len(exportData)))
	}

	logger.Info("Advanced P2P diagnostics example completed successfully!")
}
