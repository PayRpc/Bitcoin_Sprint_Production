# Bitcoin Sprint - Storage Verifier Web API

A production-ready REST API for the Bitcoin Sprint enhanced storage verification system with cryptographic proofs, rate limiting, and challenge management.

## Features

- **RESTful API**: Clean HTTP endpoints for storage verification
- **Rate Limiting**: Per-provider sliding window rate limiting
- **Challenge Management**: UUID-based challenge tracking with expiration
- **Request Validation**: Comprehensive input validation for all protocols
- **Metrics Endpoint**: Real-time metrics and health monitoring
- **Async Processing**: High-performance async/await runtime
- **Thread Safety**: Thread-safe shared state with Arc and Mutex

## API Endpoints

### POST /verify

Generate and verify storage challenges for files.

**Request Body:**

```json
{
  "file_id": "string",
  "provider": "ipfs|arweave|filecoin|bitcoin",
  "protocol": "string",
  "file_size": 1024
}
```

**Response:**

```json
{
  "verified": true,
  "timestamp": 1234567890,
  "signature": "hex_string",
  "challenge_id": "uuid_string",
  "verification_score": 0.95
}
```

### GET /health

Health check endpoint.

**Response:**

```json
{
  "status": "healthy",
  "timestamp": 1234567890,
  "uptime_seconds": 3600
}
```

### GET /metrics

System metrics and statistics.

**Response:**

```json
{
  "active_challenges": 5,
  "total_verifications": 100,
  "rate_limited_requests": 2,
  "uptime_seconds": 3600,
  "memory_usage_mb": 150
}
```

## Building and Running

### Prerequisites

- Rust 1.70+ with Cargo
- The `web-server` feature enabled

### Build the Web Server

```bash
cargo build --bin storage_verifier_server --features web-server
```

### Run the Server

```bash
cargo run --bin storage_verifier_server --features web-server
```

The server will start on `http://0.0.0.0:8080`

### Build for Production

```bash
cargo build --release --bin storage_verifier_server --features web-server
```

## Configuration

The server uses the following default configuration:

- **Host**: 0.0.0.0
- **Port**: 8080
- **Workers**: 8 (configurable via environment)
- **Rate Limit**: 100 requests per minute per provider
- **Challenge Expiration**: 300 seconds (5 minutes)

## Rate Limiting

The API implements sliding window rate limiting:

- **Window Size**: 60 seconds
- **Max Requests**: 100 per provider per window
- **Provider Identification**: Based on request source

## Challenge Management

- **Challenge Format**: UUID v4
- **Expiration**: Automatic cleanup after 5 minutes
- **Thread Safety**: Concurrent access protected with Mutex

## Error Handling

All endpoints return appropriate HTTP status codes:

- `200 OK`: Successful request
- `400 Bad Request`: Invalid input parameters
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server-side error

Error responses include detailed error messages in JSON format.

## Security Features

- **Input Validation**: All request parameters are validated
- **Rate Limiting**: Prevents abuse and DoS attacks
- **Challenge Expiration**: Prevents replay attacks
- **Thread Safety**: Safe concurrent access to shared state

## Integration

The web server integrates with the existing `StorageVerifier` from the securebuffer library, providing:

- Cryptographic proof generation
- Multi-protocol support (IPFS, Arweave, Filecoin, Bitcoin)
- Storage verification with entropy analysis
- Bloom filter-based duplicate detection

## Testing

### Health Check

```bash
curl http://localhost:8080/health
```

### Verification Request

```bash
curl -X POST http://localhost:8080/verify \
  -H "Content-Type: application/json" \
  -d '{
    "file_id": "test-file-123",
    "provider": "ipfs",
    "protocol": "ipfs",
    "file_size": 1048576
  }'
```

### Metrics

```bash
curl http://localhost:8080/metrics
```

## Architecture

The web server is built with:

- **Actix Web 4.x**: High-performance async web framework
- **Tokio Runtime**: Async runtime for concurrent processing
- **Serde**: JSON serialization/deserialization
- **UUID**: Unique identifier generation
- **Log/Env_logger**: Structured logging

## Production Deployment

For production deployment:

1. Use `--release` build flag for optimization
2. Configure reverse proxy (nginx/apache) for SSL termination
3. Set appropriate environment variables for configuration
4. Monitor logs and metrics endpoints
5. Implement proper backup and recovery procedures

## License

This project is licensed under the MIT License.
