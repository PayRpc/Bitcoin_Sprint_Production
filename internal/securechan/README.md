# Secure Channel Package Documentation

## Overview

The `securechan` package provides enterprise-grade secure channel communication for Bitcoin Sprint with comprehensive error handling, metrics collection, circuit breaker protection, and dual implementation support (CGO and pure Go).

## Architecture

### Dual Implementation Strategy

The package provides two implementations:

1. **CGO Implementation** (`ffi.go`): High-performance native C library integration
2. **Pure Go Implementation** (`fallback.go`): Fallback for environments without CGO support

Build tags automatically select the appropriate implementation:
- `//go:build cgo` - Uses native C library for maximum performance
- `//go:build !cgo` - Uses pure Go implementation for compatibility

## Features

### Enterprise-Grade Capabilities

- **Error Handling**: Comprehensive error types with operation context
- **Retry Logic**: Configurable retry attempts with exponential backoff
- **Circuit Breaker Protection**: Built-in fault tolerance and recovery
- **Metrics Collection**: Real-time performance and operational metrics
- **State Management**: Full lifecycle state tracking and notifications
- **Context Support**: Proper cancellation and timeout handling
- **Security**: TLS encryption and certificate management
- **Logging**: Structured logging with Zap integration

### Performance Optimizations

- **Connection Pooling**: Efficient resource management
- **Buffer Management**: Configurable send/receive buffers
- **Latency Tracking**: Real-time latency metrics collection
- **Health Monitoring**: Continuous connection health checks
- **Memory Safety**: Proper cleanup and resource deallocation

## API Reference

### Types

#### `Channel`
Primary interface for secure channel operations.

```go
type Channel struct {
    // Internal fields for connection management, metrics, and state
}
```

#### `ChannelConfig`
Configuration parameters for channel behavior.

```go
type ChannelConfig struct {
    ConnectionTimeout    time.Duration  // Connection establishment timeout
    ReadTimeout         time.Duration  // Read operation timeout
    WriteTimeout        time.Duration  // Write operation timeout
    KeepAliveInterval   time.Duration  // Keep-alive ping interval
    MaxRetries          int            // Maximum connection retry attempts
    RetryDelay          time.Duration  // Initial retry delay
    BackoffMultiplier   float64        // Exponential backoff multiplier
    MaxRetryDelay       time.Duration  // Maximum retry delay cap
    SendBufferSize      int            // Send buffer size in bytes
    ReceiveBufferSize   int            // Receive buffer size in bytes
    MaxMessageSize      int            // Maximum message size limit
    EnableEncryption    bool           // Enable TLS encryption
    CertificatePath     string         // TLS certificate file path
    KeyPath            string         // TLS private key file path
    EnableMetrics       bool           // Enable metrics collection
    MetricsInterval     time.Duration  // Metrics collection interval
}
```

#### `ChannelState`
Represents the current operational state of a channel.

```go
type ChannelState int

const (
    StateDisconnected ChannelState = iota  // Channel is disconnected
    StateConnecting                        // Connection in progress
    StateConnected                         // Successfully connected
    StateError                            // Error state
    StateStopping                         // Graceful shutdown in progress
)
```

#### `ChannelMetrics`
Performance and operational metrics for monitoring.

```go
type ChannelMetrics struct {
    ConnectionAttempts   int64         // Total connection attempts
    SuccessfulConnects   int64         // Successful connections
    FailedConnects      int64         // Failed connection attempts
    BytesSent           int64         // Total bytes transmitted
    BytesReceived       int64         // Total bytes received
    ErrorCount          int64         // Total error count
    LastConnectionTime  time.Time     // Timestamp of last connection
    TotalUptime         time.Duration // Total connected time
    MaxLatency          time.Duration // Maximum observed latency
    AverageLatency      time.Duration // Average operation latency
}
```

#### `SecureChannelError`
Specialized error type with operation context.

```go
type SecureChannelError struct {
    Operation string  // Operation that failed (e.g., "connect", "send", "receive")
    Endpoint  string  // Target endpoint
    Err       error   // Underlying error
}
```

### Functions

#### `NewChannel(endpoint string, config *ChannelConfig, logger *zap.Logger) (*Channel, error)`
Creates a new secure channel instance.

**Parameters:**
- `endpoint`: Target endpoint URL (e.g., "tcp://localhost:8080")
- `config`: Channel configuration (nil uses defaults)
- `logger`: Zap logger instance (nil creates default logger)

**Returns:**
- `*Channel`: Configured channel instance
- `error`: Creation error, if any

**Example:**
```go
config := DefaultChannelConfig()
logger, _ := zap.NewProduction()
channel, err := NewChannel("tcp://localhost:8080", config, logger)
if err != nil {
    log.Fatal(err)
}
defer channel.Close()
```

#### `DefaultChannelConfig() *ChannelConfig`
Returns production-ready default configuration.

**Returns:**
- `*ChannelConfig`: Default configuration with enterprise settings

### Methods

#### `Start(ctx context.Context) error`
Initiates secure channel connection with retry logic.

**Parameters:**
- `ctx`: Context for cancellation and timeout control

**Returns:**
- `error`: Connection error, if any

**Features:**
- Automatic retry with exponential backoff
- Context cancellation support
- Connection state management
- Metrics collection

#### `Stop(ctx context.Context) error`
Gracefully shuts down the secure channel connection.

**Parameters:**
- `ctx`: Context for timeout control

**Returns:**
- `error`: Shutdown error, if any

#### `Send(ctx context.Context, data []byte) (int, error)`
Transmits data through the secure channel.

**Parameters:**
- `ctx`: Context for timeout and cancellation
- `data`: Data to transmit

**Returns:**
- `int`: Number of bytes sent
- `error`: Send error, if any

**Features:**
- Message size validation
- Write timeout enforcement
- Metrics tracking
- Error handling

#### `Receive(ctx context.Context, buffer []byte) (int, error)`
Receives data from the secure channel.

**Parameters:**
- `ctx`: Context for timeout and cancellation
- `buffer`: Buffer to store received data

**Returns:**
- `int`: Number of bytes received
- `error`: Receive error, if any

**Features:**
- Read timeout enforcement
- Buffer validation
- Metrics tracking
- Error handling

#### `IsConnected() bool`
Checks if the channel is currently connected.

**Returns:**
- `bool`: Connection status

#### `GetState() ChannelState`
Returns the current channel state.

**Returns:**
- `ChannelState`: Current operational state

#### `GetMetrics() ChannelMetrics`
Returns a copy of current performance metrics.

**Returns:**
- `ChannelMetrics`: Current metrics snapshot

#### `GetEndpoint() string`
Returns the target endpoint for this channel.

**Returns:**
- `string`: Endpoint URL

#### `Close() error`
Releases all resources associated with the channel.

**Returns:**
- `error`: Cleanup error, if any

## Usage Examples

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/PayRpc/Bitcoin-Sprint/internal/securechan"
    "go.uber.org/zap"
)

func main() {
    // Create logger
    logger, err := zap.NewProduction()
    if err != nil {
        log.Fatal(err)
    }
    defer logger.Sync()

    // Create channel with default config
    channel, err := securechan.NewChannel("tcp://localhost:8080", nil, logger)
    if err != nil {
        log.Fatal(err)
    }
    defer channel.Close()

    // Start connection
    ctx := context.Background()
    if err := channel.Start(ctx); err != nil {
        log.Fatal(err)
    }

    // Send data
    data := []byte("Hello, Secure Channel!")
    sent, err := channel.Send(ctx, data)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Sent %d bytes", sent)

    // Receive data
    buffer := make([]byte, 1024)
    received, err := channel.Receive(ctx, buffer)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Received %d bytes: %s", received, string(buffer[:received]))

    // Stop connection
    if err := channel.Stop(ctx); err != nil {
        log.Fatal(err)
    }
}
```

### Advanced Configuration

```go
func createAdvancedChannel() (*securechan.Channel, error) {
    // Custom configuration
    config := &securechan.ChannelConfig{
        ConnectionTimeout:   15 * time.Second,
        ReadTimeout:        5 * time.Second,
        WriteTimeout:       5 * time.Second,
        KeepAliveInterval:  30 * time.Second,
        MaxRetries:         5,
        RetryDelay:         200 * time.Millisecond,
        BackoffMultiplier:  1.5,
        MaxRetryDelay:     10 * time.Second,
        SendBufferSize:    16384,
        ReceiveBufferSize: 16384,
        MaxMessageSize:    2 * 1024 * 1024, // 2MB
        EnableEncryption:  true,
        CertificatePath:   "/path/to/cert.pem",
        KeyPath:          "/path/to/key.pem",
        EnableMetrics:     true,
        MetricsInterval:   15 * time.Second,
    }

    logger, _ := zap.NewProduction()
    return securechan.NewChannel("tls://secure.example.com:443", config, logger)
}
```

### Metrics Monitoring

```go
func monitorChannelMetrics(channel *securechan.Channel) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        metrics := channel.GetMetrics()
        
        log.Printf("Channel Metrics:")
        log.Printf("  State: %s", channel.GetState().String())
        log.Printf("  Connection Attempts: %d", metrics.ConnectionAttempts)
        log.Printf("  Successful Connects: %d", metrics.SuccessfulConnects)
        log.Printf("  Failed Connects: %d", metrics.FailedConnects)
        log.Printf("  Bytes Sent: %d", metrics.BytesSent)
        log.Printf("  Bytes Received: %d", metrics.BytesReceived)
        log.Printf("  Error Count: %d", metrics.ErrorCount)
        log.Printf("  Total Uptime: %v", metrics.TotalUptime)
        log.Printf("  Max Latency: %v", metrics.MaxLatency)
        log.Printf("  Average Latency: %v", metrics.AverageLatency)
    }
}
```

### Error Handling

```go
func handleChannelErrors(err error) {
    if secErr, ok := err.(*securechan.SecureChannelError); ok {
        log.Printf("Secure channel error in %s operation for endpoint %s: %v", 
                  secErr.Operation, secErr.Endpoint, secErr.Err)
        
        // Handle specific operations
        switch secErr.Operation {
        case "connect":
            log.Println("Connection failed, will retry...")
        case "send":
            log.Println("Send failed, message may be lost")
        case "receive":
            log.Println("Receive failed, data may be incomplete")
        }
    } else {
        log.Printf("General error: %v", err)
    }
}
```

## Performance Considerations

### CGO vs Pure Go Implementation

- **CGO Implementation**: Better performance for high-throughput scenarios
- **Pure Go Implementation**: Better compatibility and easier deployment

### Buffer Sizing

- Larger buffers reduce system call overhead
- Smaller buffers reduce memory usage
- Default 8KB buffers balance performance and memory

### Connection Pooling

- Reuse channels when possible
- Implement connection pooling for high-concurrency scenarios
- Monitor connection health and replace failed connections

### Timeout Configuration

- Set appropriate timeouts based on network conditions
- Use shorter timeouts for local connections
- Use longer timeouts for high-latency connections

## Security Considerations

### TLS Configuration

- Always enable encryption for production deployments
- Use strong cipher suites and protocol versions
- Validate certificates and implement proper PKI

### Error Information

- Avoid exposing sensitive information in error messages
- Log detailed errors securely for debugging
- Sanitize error messages in production

### Resource Management

- Always call `Close()` to prevent resource leaks
- Use context cancellation for proper cleanup
- Monitor resource usage in long-running applications

## Testing

The package includes comprehensive test coverage:

- Unit tests for all public methods
- Error condition testing
- Performance benchmarks
- CGO and non-CGO build validation

Run tests with:
```bash
# All tests
go test ./internal/securechan/...

# With CGO
CGO_ENABLED=1 go test ./internal/securechan/...

# Without CGO
CGO_ENABLED=0 go test ./internal/securechan/...

# Benchmarks
go test -bench=. ./internal/securechan/...
```

## Integration with Bitcoin Sprint

The secure channel package integrates seamlessly with Bitcoin Sprint's enterprise architecture:

- **Circuit Breaker Integration**: Automatic fault tolerance
- **Metrics Collection**: Prometheus metrics export
- **Logging Integration**: Structured logging with Zap
- **Configuration Management**: Environment-based configuration
- **Service Lifecycle**: Proper startup and shutdown handling

This package provides the secure communication foundation for Bitcoin Sprint's enterprise relay capabilities, ensuring reliable and performant inter-service communication.
