# Bitcoin Sprint - Enhanced Storage Verification Service

A production-ready Rust Actix Web service for entropy-based storage verification with advanced connection management, circuit breaker patterns, and enterprise-grade reliability features.

## 🚀 Features

### Connection Management

- **Connection Pooling**: Intelligent HTTP client with configurable connection limits
- **Timeout Management**: Configurable connect, request, and idle timeouts
- **Keep-Alive**: Persistent connections with configurable keep-alive intervals
- **Resource Limits**: Maximum connections per host and idle connection management

### Reliability Patterns

- **Circuit Breaker**: Automatic failure detection and recovery
- **Retry Logic**: Exponential backoff with configurable retry policies
- **Rate Limiting**: Per-provider rate limiting with cleanup
- **Health Monitoring**: Real-time connection pool and circuit breaker status

### Enterprise Features

- **Graceful Shutdown**: 30-second graceful shutdown with connection draining
- **Structured Logging**: Comprehensive logging with request tracing
- **Metrics Endpoint**: Real-time metrics and health status
- **Input Validation**: Robust request validation with detailed error responses

## 🏗️ Architecture

```text
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Actix Web     │────│  Circuit Breaker │────│  HTTP Client    │
│   Server        │    │                  │    │  (reqwest)      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌──────────────────┐
                    │  Storage         │
                    │  Verifier        │
                    │  (Entropy)       │
                    └──────────────────┘
```

## 📋 Configuration

### Connection Configuration

```rust
let connection_config = ConnectionConfig {
    max_connections: 100,
    connect_timeout: Duration::from_secs(10),
    request_timeout: Duration::from_secs(30),
    pool_idle_timeout: Duration::from_secs(90),
    max_idle_connections: 10,
    keep_alive: Duration::from_secs(60),
};
```

### Circuit Breaker Configuration

```rust
let circuit_breaker = CircuitBreaker::new(
    5,                          // Failure threshold
    Duration::from_secs(60)     // Recovery timeout
);
```

### Rate Limiting Configuration

```rust
let rate_limiter = RateLimiter::new(
    10,     // Max requests per window
    60      // Window duration in seconds
);
```

## 🔧 API Endpoints

### POST /verify

Verify storage proof with connection management.

**Request:**

```json
{
    "file_id": "bitcoin_block_800000.dat",
    "provider": "decentralized_provider",
    "protocol": "ipfs",
    "file_size": 1048576
}
```

**Response:**

```json
{
    "verified": true,
    "timestamp": 1640995200,
    "signature": "0x...",
    "challenge_id": "uuid-v4",
    "verification_score": 0.85,
    "connection_health": {
        "pool_size": 100,
        "active_connections": 5,
        "idle_connections": 3,
        "circuit_breaker_state": "CLOSED"
    }
}
```

### GET /health

Service health check with connection status.

### GET /metrics

Real-time metrics and connection pool statistics.

## 🚀 Running the Service

### Prerequisites

- Rust 1.70+
- Cargo

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd bitcoin-sprint

# Build the service
cargo build --release

# Run the service
cargo run --release
```

### Docker

```bash
# Build Docker image
docker build -t bitcoin-sprint-storage-verifier .

# Run with Docker
docker run -p 8080:8080 bitcoin-sprint-storage-verifier
```

## 🔍 Connection Management Details

### Connection Pooling Strategy

- **Max Connections**: 100 total connections across all hosts
- **Per-Host Limit**: 10 idle connections per host
- **Idle Timeout**: 90 seconds before closing idle connections
- **Keep-Alive**: 60-second TCP keep-alive intervals

### Circuit Breaker States

- **CLOSED**: Normal operation, requests pass through
- **OPEN**: Service unavailable, requests fail fast
- **HALF-OPEN**: Testing recovery, limited requests allowed

### Retry Strategy

- **Exponential Backoff**: Base delay with exponential growth
- **Max Retries**: Configurable maximum retry attempts
- **Jitter**: Randomization to prevent thundering herd

### Timeout Configuration

- **Connect Timeout**: 10 seconds for initial connection
- **Request Timeout**: 30 seconds for complete request/response
- **Idle Timeout**: 90 seconds for connection pool cleanup

## 📊 Monitoring

### Health Check Response

```json
{
    "status": "healthy",
    "timestamp": 1640995200,
    "service": "entropy-storage-verifier",
    "connection_health": {
        "pool_size": 100,
        "active_connections": 5,
        "idle_connections": 3,
        "circuit_breaker_state": "CLOSED"
    }
}
```

### Metrics Response

```json
{
    "active_challenges": 12,
    "timestamp": 1640995200,
    "connection_health": {
        "pool_size": 100,
        "active_connections": 5,
        "idle_connections": 3,
        "circuit_breaker_state": "CLOSED"
    }
}
```

## 🛡️ Security Features

- **Rate Limiting**: Prevents abuse and ensures fair resource usage
- **Input Validation**: Comprehensive validation of all request parameters
- **Timeout Protection**: Prevents resource exhaustion from slow requests
- **Circuit Breaker**: Automatic protection against cascading failures
- **Connection Limits**: Prevents connection pool exhaustion

## 🔧 Configuration Options

### Environment Variables

```bash
# Server Configuration
RUST_LOG=info
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
WORKERS=8

# Connection Configuration
MAX_CONNECTIONS=100
CONNECT_TIMEOUT_SECS=10
REQUEST_TIMEOUT_SECS=30
POOL_IDLE_TIMEOUT_SECS=90
MAX_IDLE_CONNECTIONS=10
KEEP_ALIVE_SECS=60

# Circuit Breaker
FAILURE_THRESHOLD=5
RECOVERY_TIMEOUT_SECS=60

# Rate Limiting
MAX_REQUESTS_PER_WINDOW=10
RATE_WINDOW_SECS=60
```

## 📈 Performance Optimizations

1. **Connection Reuse**: Persistent connections reduce handshake overhead
2. **Request Batching**: Efficient handling of concurrent requests
3. **Memory Pooling**: Reuse of memory buffers for request processing
4. **Async Processing**: Non-blocking I/O for high concurrency
5. **Resource Cleanup**: Automatic cleanup of expired connections and challenges

## 🧪 Testing

### Unit Tests

```bash
cargo test
```

### Integration Tests

```bash
cargo test --test integration
```

### Load Testing

```bash
# Using wrk for load testing
wrk -t12 -c400 -d30s http://localhost:8080/health

# Using hey for simple load testing
hey -n 1000 -c 10 http://localhost:8080/health
```

## 📝 Logging

The service uses structured logging with the following levels:

- **ERROR**: Critical errors requiring immediate attention
- **WARN**: Warning conditions that should be monitored
- **INFO**: General information about service operation
- **DEBUG**: Detailed debugging information
- **TRACE**: Very detailed execution tracing

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🆘 Support

For support and questions:

- Create an issue on GitHub
- Check the documentation
- Review the logs for error details

## 🔄 Migration from Previous Version

If migrating from the previous version:

1. **Update Dependencies**: Update to the new Cargo.toml dependencies
2. **Configuration**: Review and update connection configuration
3. **API Changes**: Note the addition of `connection_health` in responses
4. **Monitoring**: Update monitoring to use new health endpoints

The new version is backward compatible for the `/verify` endpoint but provides enhanced reliability and monitoring capabilities.
