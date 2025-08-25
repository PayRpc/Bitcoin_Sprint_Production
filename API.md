# Bitcoin Sprint Customer API Documentation

## Overview

Bitcoin Sprint provides a comprehensive REST API for enterprise customers to access Bitcoin block data, analytics, and system information programmatically.

**Base URL**: `http://localhost:8080` (configurable via APIBase setting)  
**API Version**: v1  
**Authentication**: Rate-limited public access  

## Available Endpoints

### 📊 System Status APIs

#### GET `/status`
Returns current system status, license information, and performance metrics.

**Example Response:**
```json
{
  "tier": "Enterprise",
  "license_key": "btc_****_****_1234",
  "valid": true,
  "blocks_sent_today": 150,
  "block_limit": 1000,
  "peers_connected": 8,
  "uptime_seconds": 3600,
  "version": "1.2.0",
  "turbo_mode_enabled": true
}
```

#### GET `/metrics`
Performance metrics and detailed system statistics.

#### GET `/predictive`
Predictive analytics and trend data.

#### GET `/stream`
Server-sent events stream for real-time metrics.

---

### 🧱 Block Information APIs

#### GET `/api/v1/blocks/`
Get latest block information and available endpoints.

**Example Response:**
```json
{
  "latest_height": 850000,
  "latest_hash": "00000000000000000002a7c4c1e48d76c5a37902165a270156b7a8d72728a054",
  "timestamp": "2024-08-24T15:30:00Z",
  "api_version": "v1",
  "endpoints": [
    "/api/v1/blocks/{height}",
    "/api/v1/blocks/range/{start}/{end}"
  ]
}
```

#### GET `/api/v1/blocks/{height}`
Get information about a specific block by height.

**Parameters:**
- `height` (path): Block height number

**Example:**
```bash
curl "http://localhost:8080/api/v1/blocks/850000"
```

**Response:**
```json
{
  "requested_height": "850000",
  "latest_height": 850001,
  "latest_hash": "00000000000000000002a7c4c1e48d76c5a37902165a270156b7a8d72728a054",
  "timestamp": "2024-08-24T15:30:00Z",
  "message": "Single block lookup - integrate with your Bitcoin node for full block data"
}
```

#### GET `/api/v1/blocks/range/{start}/{end}`
Get information about a range of blocks.

**Parameters:**
- `start` (path): Starting block height
- `end` (path): Ending block height

**Example:**
```bash
curl "http://localhost:8080/api/v1/blocks/range/850000/850010"
```

---

### 🔑 License Management APIs

#### GET `/api/v1/license/info`
Get detailed license information and usage statistics.

**Example Response:**
```json
{
  "tier": "Enterprise",
  "valid": true,
  "block_limit": 1000,
  "expires_at": 1735689600,
  "blocks_today": 150,
  "api_version": "v1"
}
```

---

### 📈 Analytics APIs

#### GET `/api/v1/analytics/summary`
Get comprehensive analytics summary.

**Example Response:**
```json
{
  "current_block_height": 850000,
  "total_peers": 8,
  "blocks_sent_today": 150,
  "uptime_seconds": 3600,
  "turbo_mode": true,
  "api_version": "v1",
  "analytics_features": [
    "Real-time block monitoring",
    "Peer connection tracking", 
    "Performance metrics",
    "Predictive analytics"
  ]
}
```

---

## Rate Limiting

All API endpoints are rate-limited to prevent abuse:
- **Default**: Standard rate limiting per IP
- **Turbo Mode**: Higher rate limits for Enterprise customers

When rate limit is exceeded, you'll receive:
```json
HTTP 429 Too Many Requests
"Rate limit exceeded"
```

## Error Handling

The API uses standard HTTP status codes:
- `200` - Success
- `400` - Bad Request (invalid parameters)
- `429` - Rate Limit Exceeded
- `500` - Internal Server Error

## Integration Examples

### Python Example
```python
import requests

# Get latest block info
response = requests.get("http://localhost:8080/api/v1/blocks/")
latest = response.json()
print(f"Latest block: {latest['latest_height']}")

# Get license info
license_info = requests.get("http://localhost:8080/api/v1/license/info").json()
print(f"Blocks remaining: {license_info['block_limit'] - license_info['blocks_today']}")
```

### JavaScript Example
```javascript
// Get analytics summary
fetch('http://localhost:8080/api/v1/analytics/summary')
  .then(response => response.json())
  .then(data => {
    console.log(`Current block: ${data.current_block_height}`);
    console.log(`Peers connected: ${data.total_peers}`);
  });
```

### cURL Examples
```bash
# Get system status
curl "http://localhost:8080/status"

# Get specific block
curl "http://localhost:8080/api/v1/blocks/850000"

# Get block range
curl "http://localhost:8080/api/v1/blocks/range/850000/850010"

# Get license info
curl "http://localhost:8080/api/v1/license/info"

# Stream real-time metrics
curl "http://localhost:8080/stream"
```

## Custom API Development

Need a custom endpoint for your specific use case? Bitcoin Sprint's modular architecture makes it easy to add new APIs:

1. **Block History APIs** - Access historical block data
2. **Transaction APIs** - Search and analyze transactions  
3. **Mempool APIs** - Monitor unconfirmed transactions
4. **Alert APIs** - Set up custom notifications
5. **Webhook APIs** - Push notifications to your systems

Contact our development team for custom API development and integration support.

---

## Support

- **Documentation**: This file and inline code comments
- **Issues**: Use GitHub issues for bug reports
- **Custom Development**: Contact for enterprise API extensions
