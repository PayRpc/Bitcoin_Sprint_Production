# P2P Diagnostics Module

A comprehensive diagnostics and monitoring system for Bitcoin P2P network operations.

## Overview

The diagnostics module provides real-time monitoring, event recording, and statistical analysis for P2P network operations in the Bitcoin Sprint system. It offers production-ready features including:

- **Event Recording**: Track all P2P operations with detailed metadata
- **Statistics**: Real-time analytics on network health and performance
- **Severity Filtering**: Configurable event filtering by severity levels
- **Automatic Cleanup**: Memory-efficient event management with automatic cleanup
- **Concurrency Safety**: Thread-safe operations for high-throughput environments

## Features

### Event Types
- **Peer Connections/Disconnections**: Track peer lifecycle events
- **Message Exchanges**: Monitor P2P protocol messages
- **Error Tracking**: Record and analyze network errors
- **Custom Events**: Support for application-specific diagnostic events

### Severity Levels
- `DEBUG`: Detailed debugging information
- `INFO`: General informational messages
- `WARNING`: Warning conditions that don't affect operation
- `ERROR`: Error conditions that may affect operation
- `CRITICAL`: Critical errors requiring immediate attention

### Advanced Features
- **Health Monitoring**: Continuous health checks with status reporting
- **Configuration Management**: Flexible configuration with sensible defaults
- **Data Export**: Export diagnostic data in JSON format for analysis
- **Performance Benchmarking**: Built-in benchmarking for performance analysis
- **Resource Management**: Automatic cleanup with configurable retention policies

## Usage

### Basic Setup

```go
import (
    "context"
    "github.com/PayRpc/Bitcoin-Sprint/cmd/p2p/diagnostics"
    "go.uber.org/zap"
)

// Create logger
logger, _ := zap.NewDevelopment()

// Create recorder with max 1000 events
recorder := diagnostics.NewRecorder(1000, logger)
defer recorder.Close()

ctx := context.Background()
```

### Advanced Configuration

```go
// Create custom configuration
config := &diagnostics.RecorderConfig{
    MaxEvents:         10000,              // Maximum events to store
    CleanupInterval:   5 * time.Minute,    // Cleanup frequency
    RetentionPeriod:   24 * time.Hour,     // How long to keep events
    EnableMetrics:     true,               // Enable metrics collection
    EnableHealthCheck: true,               // Enable health monitoring
    LogLevel:          "info",             // Logging level
    ExportFormat:      "json",             // Export format
}

recorder := diagnostics.NewRecorderWithConfig(config, logger)
defer recorder.Close()

// Or use default configuration and customize
config := diagnostics.DefaultRecorderConfig()
config.MaxEvents = 5000
recorder := diagnostics.NewRecorderWithConfig(config, logger)
```

### Recording Events

```go
// Record peer connection
err := recorder.RecordPeerConnection(ctx, "peer123", "192.168.1.100:8333")

// Record message exchange
err := recorder.RecordMessage(ctx, "peer123", "version", "outbound", 1024)

// Record error
err := recorder.RecordError(ctx, "peer123", fmt.Errorf("timeout"), "handshake")

// Record custom event
event := &diagnostics.DiagnosticEvent{
    EventType: "custom_event",
    Message:   "Custom diagnostic message",
    Severity:  diagnostics.SeverityInfo,
    Metadata: map[string]interface{}{
        "key": "value",
    },
}
err := recorder.RecordEvent(ctx, event)
```

### Retrieving Events

```go
// Get recent events (last 50, minimum severity INFO)
events, err := recorder.GetEvents(ctx, 50, diagnostics.SeverityInfo)

// Get all events regardless of severity
allEvents, err := recorder.GetEvents(ctx, 100, diagnostics.SeverityDebug)
```

### Statistics and Monitoring

```go
// Get comprehensive statistics
stats, err := recorder.GetStats(ctx)

fmt.Printf("Total Events: %d\n", stats.TotalEvents)
fmt.Printf("Active Peers: %d\n", stats.ActivePeers)
fmt.Printf("Error Rate: %.2f%%\n", stats.ErrorRate*100)

// Events by type
for eventType, count := range stats.EventsByType {
    fmt.Printf("%s: %d events\n", eventType, count)
}

// Events by severity
for severity, count := range stats.EventsBySeverity {
    fmt.Printf("%s: %d events\n", severity.String(), count)
}
```

### Health Monitoring

```go
// Perform health check
health, err := recorder.HealthCheck(ctx)
if err != nil {
    log.Printf("Health check failed: %v", err)
} else {
    fmt.Printf("Health Status: %s\n", health.Status)
    fmt.Printf("Message: %s\n", health.Message)
    fmt.Printf("Event Count: %d/%d\n", health.EventCount, health.MaxEvents)
    fmt.Printf("Total Events: %d\n", health.TotalEvents)
}

// Continuous health monitoring
go func() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            health, err := recorder.HealthCheck(ctx)
            if err != nil {
                logger.Error("Health check failed", zap.Error(err))
                continue
            }

            if health.Status != "healthy" {
                logger.Warn("Unhealthy diagnostics state",
                    zap.String("status", health.Status),
                    zap.String("message", health.Message))
            }
        }
    }
}()
```

### Data Export

```go
// Export all events in JSON format
exportData, err := recorder.ExportEvents(ctx)
if err != nil {
    log.Printf("Export failed: %v", err)
} else {
    // Save to file
    err := os.WriteFile("diagnostics_export.json", exportData, 0644)
    if err != nil {
        log.Printf("Failed to save export file: %v", err)
    } else {
        log.Printf("Exported %d bytes of diagnostic data", len(exportData))
    }
}
```

### Cleanup and Management

```go
// Clear all events
err := recorder.ClearEvents(ctx)

// Events are automatically cleaned up after 24 hours
// Manual cleanup can be triggered
recorder.cleanupOldEvents()
```

### Performance Benchmarking

```go
// Run performance benchmarks
go test -bench=. ./cmd/p2p/diagnostics/

// Example benchmark test
func BenchmarkEventRecording(b *testing.B) {
    recorder := diagnostics.NewRecorder(10000, zap.NewNop())
    defer recorder.Close()

    ctx := context.Background()
    event := &diagnostics.DiagnosticEvent{
        EventType: "benchmark_test",
        Message:   "Performance test event",
        Severity:  diagnostics.SeverityInfo,
    }

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            recorder.RecordEvent(ctx, event)
        }
    })
}

// Record performance metrics in application code
start := time.Now()
result := performNetworkOperation()
duration := time.Since(start)

event := &diagnostics.DiagnosticEvent{
    EventType: "performance_metric",
    Message:   fmt.Sprintf("Operation completed in %v", duration),
    Severity:  diagnostics.SeverityInfo,
    Metadata: map[string]interface{}{
        "operation":     "network_request",
        "duration_ms":   duration.Milliseconds(),
        "success":       result.Success,
    },
}
recorder.RecordEvent(ctx, event)
```

## Architecture

### Core Components

1. **Recorder**: Main component that handles event recording and management
2. **DiagnosticEvent**: Structured event data with metadata
3. **DiagnosticStats**: Statistical information about recorded events
4. **Severity**: Event severity classification system

### Thread Safety

The diagnostics module is fully thread-safe and can be used concurrently from multiple goroutines:

```go
// Safe for concurrent use
go func() {
    recorder.RecordPeerConnection(ctx, "peer1", "addr1")
}()

go func() {
    recorder.RecordMessage(ctx, "peer2", "ping", "inbound", 32)
}()
```

### Memory Management

- Configurable maximum number of events
- Automatic cleanup of events older than 24 hours
- Efficient storage with minimal memory overhead
- Background cleanup routine prevents memory leaks

## Integration

### With Existing P2P Code

```go
// In your P2P client
type P2PClient struct {
    diagnostics *diagnostics.Recorder
    // ... other fields
}

func (p *P2PClient) handlePeerConnection(peerID, address string) {
    // Existing connection logic...

    // Add diagnostics
    p.diagnostics.RecordPeerConnection(ctx, peerID, address)
}

func (p *P2PClient) handleMessage(peerID, msgType, direction string, size int) {
    // Existing message handling...

    // Add diagnostics
    p.diagnostics.RecordMessage(ctx, peerID, msgType, direction, size)
}
```

### Advanced Integration Patterns

#### Enhanced P2P Client with Health Monitoring

```go
type EnhancedP2PClient struct {
    diagnostics *diagnostics.Recorder
    logger      *zap.Logger
    healthChan  chan diagnostics.HealthStatus
}

func NewEnhancedP2PClient(logger *zap.Logger) *EnhancedP2PClient {
    config := diagnostics.DefaultRecorderConfig()
    config.EnableHealthCheck = true
    recorder := diagnostics.NewRecorderWithConfig(config, logger)

    return &EnhancedP2PClient{
        diagnostics: recorder,
        logger:      logger,
        healthChan:  make(chan diagnostics.HealthStatus, 10),
    }
}

func (c *EnhancedP2PClient) StartHealthMonitoring(ctx context.Context) {
    go c.healthMonitor(ctx)
    go c.periodicHealthCheck(ctx)
}

func (c *EnhancedP2PClient) healthMonitor(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case health := <-c.healthChan:
            if health.Status == "error" {
                c.logger.Error("Critical diagnostics issue",
                    zap.String("message", health.Message))
                // Trigger alerts or recovery procedures
            }
        }
    }
}

func (c *EnhancedP2PClient) periodicHealthCheck(ctx context.Context) {
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
                c.logger.Warn("Health channel full")
            }
        }
    }
}
```

#### Performance-Aware Operations

```go
func (c *EnhancedP2PClient) ConnectToPeerWithDiagnostics(ctx context.Context, peerID, address string) error {
    start := time.Now()

    // Record connection attempt
    c.diagnostics.RecordEvent(ctx, &diagnostics.DiagnosticEvent{
        EventType: "connection_start",
        PeerID:    peerID,
        Message:   "Starting peer connection",
        Severity:  diagnostics.SeverityDebug,
        Metadata: map[string]interface{}{
            "address": address,
            "start_time": start,
        },
    })

    // Perform connection
    err := c.actualConnect(peerID, address)
    duration := time.Since(start)

    if err != nil {
        c.diagnostics.RecordError(ctx, peerID, err, "connection")
        return err
    }

    // Record successful connection with performance data
    c.diagnostics.RecordPeerConnection(ctx, peerID, address)
    c.diagnostics.RecordEvent(ctx, &diagnostics.DiagnosticEvent{
        EventType: "connection_performance",
        PeerID:    peerID,
        Message:   fmt.Sprintf("Connection established in %v", duration),
        Severity:  diagnostics.SeverityInfo,
        Metadata: map[string]interface{}{
            "duration_ms": duration.Milliseconds(),
            "address":     address,
        },
    })

    return nil
}
```

#### Comprehensive Error Handling

```go
func (c *EnhancedP2PClient) HandleErrorWithContext(ctx context.Context, peerID string, err error, operation string, additionalContext map[string]interface{}) {
    // Create rich error event
    metadata := map[string]interface{}{
        "operation": operation,
        "error_type": fmt.Sprintf("%T", err),
        "timestamp":  time.Now(),
    }

    // Add additional context
    for k, v := range additionalContext {
        metadata[k] = v
    }

    // Record the error
    if recordErr := c.diagnostics.RecordError(ctx, peerID, err, operation); recordErr != nil {
        c.logger.Error("Failed to record error", zap.Error(recordErr))
        return
    }

    // Create detailed diagnostic event
    event := &diagnostics.DiagnosticEvent{
        EventType: "detailed_error",
        PeerID:    peerID,
        Message:   fmt.Sprintf("Error in %s: %v", operation, err),
        Severity:  diagnostics.SeverityError,
        Metadata:  metadata,
    }

    if recordErr := c.diagnostics.RecordEvent(ctx, event); recordErr != nil {
        c.logger.Error("Failed to record detailed error", zap.Error(recordErr))
    }
}
```

### Configuration

```go
// Development: More verbose logging, smaller event buffer
devRecorder := diagnostics.NewRecorder(100, devLogger)

// Production: Optimized for performance, larger buffer
prodRecorder := diagnostics.NewRecorder(10000, prodLogger)
```

## Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| MaxEvents | 1000 | Maximum number of events to store |
| CleanupInterval | 5m | How often to run cleanup |
| RetentionPeriod | 24h | How long to keep events |
| EnableMetrics | true | Enable metrics collection |
| EnableHealthCheck | true | Enable health monitoring |
| LogLevel | "info" | Logging level |
| ExportFormat | "json" | Export format |

## Best Practices

### Resource Management
- Always call `Close()` when done with the recorder
- Configure appropriate `MaxEvents` based on your use case
- Set reasonable `RetentionPeriod` to prevent unbounded growth
- Monitor memory usage in production environments

### Performance Considerations
- Use appropriate severity levels to control log volume
- Consider using sampling for high-frequency events
- Batch similar events when possible
- Use background health monitoring judiciously

### Error Handling
- Always check return values from recorder methods
- Use appropriate severity levels for different error types
- Include relevant context in error metadata
- Implement proper fallback behavior when diagnostics fail

### Monitoring Integration
- Export data periodically for long-term storage
- Set up alerts based on health check status
- Monitor error rates and performance metrics
- Integrate with existing monitoring systems

### Production Deployment
```go
// Production configuration example
config := &diagnostics.RecorderConfig{
    MaxEvents:         50000,             // Larger buffer for production
    CleanupInterval:   10 * time.Minute,  // Less frequent cleanup
    RetentionPeriod:   72 * time.Hour,    // Keep data longer
    EnableMetrics:     true,
    EnableHealthCheck: true,
    LogLevel:          "warn",            // Less verbose in production
    ExportFormat:      "json",
}

recorder := diagnostics.NewRecorderWithConfig(config, prodLogger)
```

## Performance Considerations

- **Memory Usage**: ~1-2KB per event (configurable buffer size)
- **CPU Overhead**: Minimal impact on P2P operations
- **Cleanup**: Automatic background cleanup prevents memory growth
- **Concurrency**: Optimized for high-throughput P2P environments

## Error Handling

The diagnostics module provides comprehensive error handling:

```go
// All operations return errors for proper handling
if err := recorder.RecordEvent(ctx, event); err != nil {
    // Handle recording error
    log.Printf("Failed to record event: %v", err)
}

// Check recorder status
if err := recorder.GetStats(ctx); err != nil {
    // Recorder may be closed or unavailable
    log.Printf("Recorder unavailable: %v", err)
}
```

## Testing

The module includes comprehensive unit tests covering:

- Basic event recording and retrieval
- Concurrent access patterns
- Memory limits and cleanup
- Error conditions and edge cases
- Statistics accuracy
- Helper method functionality
- Health monitoring capabilities
- Data export functionality

Run tests with:
```bash
go test ./cmd/p2p/diagnostics -v
```

Run benchmarks with:
```bash
go test -bench=. ./cmd/p2p/diagnostics/
```

Run coverage analysis:
```bash
go test -cover ./cmd/p2p/diagnostics/
```

### Example Benchmark Results

```
BenchmarkEventRecording-8         1000000    1234 ns/op    456 B/op    12 allocs/op
BenchmarkConcurrentRecording-8     500000    2345 ns/op    678 B/op    23 allocs/op
BenchmarkStatsRetrieval-8         2000000     678 ns/op     89 B/op     3 allocs/op
```

### Test Categories

- **Unit Tests**: Test individual functions and methods
- **Integration Tests**: Test interaction between components
- **Concurrency Tests**: Test thread safety and race conditions
- **Performance Tests**: Benchmark critical operations
- **Edge Case Tests**: Test boundary conditions and error scenarios

## API Reference

### Types

- `DiagnosticEvent`: Event data structure
- `Severity`: Severity level enumeration
- `DiagnosticStats`: Statistical information
- `RecorderConfig`: Configuration structure
- `HealthStatus`: Health check result structure
- `NetworkDiagnostic`: Network health diagnostic structure

### Core Methods

- `NewRecorder(maxEvents int, logger *zap.Logger) *Recorder`
- `NewRecorderWithConfig(config *RecorderConfig, logger *zap.Logger) *Recorder`
- `DefaultRecorderConfig() *RecorderConfig`
- `RecordEvent(ctx context.Context, event *DiagnosticEvent) error`
- `GetEvents(ctx context.Context, limit int, minSeverity Severity) ([]*DiagnosticEvent, error)`
- `GetStats(ctx context.Context) (*DiagnosticStats, error)`
- `HealthCheck(ctx context.Context) (*HealthStatus, error)`
- `ExportEvents(ctx context.Context) ([]byte, error)`
- `ClearEvents(ctx context.Context) error`
- `Close() error`

### Helper Methods

- `RecordPeerConnection(ctx context.Context, peerID, address string) error`
- `RecordPeerDisconnection(ctx context.Context, peerID, address, reason string) error`
- `RecordMessage(ctx context.Context, peerID, msgType, direction string, size int) error`
- `RecordError(ctx context.Context, peerID string, err error, operation string) error`

## Examples

See the `examples/` directory for comprehensive usage examples:

- `example.go`: Basic usage patterns and simple integration
- `integration_example.go`: Integration with existing P2P clients and network health monitoring
- `advanced_integration_example.go`: Advanced patterns with health monitoring, performance analysis, and comprehensive reporting

### Basic Example

```go
package main

import (
    "context"
    "log"
    "github.com/PayRpc/Bitcoin-Sprint/cmd/p2p/diagnostics"
    "go.uber.org/zap"
)

func main() {
    logger, _ := zap.NewDevelopment()
    recorder := diagnostics.NewRecorder(1000, logger)
    defer recorder.Close()

    ctx := context.Background()

    // Record some events
    recorder.RecordPeerConnection(ctx, "peer1", "192.168.1.100:8333")
    recorder.RecordMessage(ctx, "peer1", "version", "outbound", 150)

    // Get statistics
    stats, _ := recorder.GetStats(ctx)
    log.Printf("Recorded %d events", stats.TotalEvents)
}
```

### Advanced Example with Health Monitoring

```go
func main() {
    logger, _ := zap.NewDevelopment()

    // Custom configuration
    config := &diagnostics.RecorderConfig{
        MaxEvents:         10000,
        CleanupInterval:   5 * time.Minute,
        RetentionPeriod:   24 * time.Hour,
        EnableHealthCheck: true,
    }

    recorder := diagnostics.NewRecorderWithConfig(config, logger)
    defer recorder.Close()

    ctx := context.Background()

    // Start health monitoring
    go func() {
        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                health, err := recorder.HealthCheck(ctx)
                if err != nil {
                    log.Printf("Health check failed: %v", err)
                    continue
                }

                if health.Status != "healthy" {
                    log.Printf("Health issue: %s", health.Message)
                }
            }
        }
    }()

    // Use recorder for diagnostics...
}
```

## Future Enhancements

- **Metrics Export**: Integration with Prometheus metrics
- **Alerting**: Configurable alerts based on event patterns
- **Persistence**: Optional event persistence to disk/database
- **Visualization**: Web dashboard for real-time monitoring
- **Performance Profiling**: Detailed performance metrics
