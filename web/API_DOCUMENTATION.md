# Multi-Chain Sprint API Documentation

This document provides complete documentation for the Multi-Chain Sprint API endpoints supporting Bitcoin, Ethereum, Solana, Cosmos, Polkadot and other major blockchains.

## Authentication

All API endpoints require an API key, which must be provided in the `X-API-KEY` header:

```http
X-API-KEY: your-api-key-here
```

Alternatively, you can use the standard Bearer token format in the Authorization header:

```http
Authorization: Bearer your-api-key-here
```

## Tier System

Bitcoin Sprint uses a tiered access system:

- **FREE**: Basic access with limited rate and daily block limits
- **PRO**: Enhanced access with higher limits and additional endpoints
- **ENTERPRISE**: Full access to all features including streaming, mempool data, and analytics
- **ENTERPRISE_PLUS**: Unlimited access with dedicated infrastructure

## Endpoints

### Health and Status

#### GET `/api/health`

Check the health of the Bitcoin Sprint service.

**Response Example:**

```json
{
  "status": "ok",
  "timestamp": "2025-08-26T09:45:00Z",
  "uptime": 3600,
  "version": "2.1.0",
  "checks": {
    "database": {
      "status": "ok",
      "responseTime": 5.23
    },
    "api": {
      "status": "ok",
      "responseTime": 12.45
    },
    "bitcoind": {
      "status": "ok",
      "responseTime": 85.67,
      "chain": "main",
      "blocks": 850000,
      "connections": 8
    }
  }
}
```

#### GET `/api/maintenance`

Check if the service is in maintenance mode.

**Response Example:**

```json
{
  "maintenance": false,
  "message": "Bitcoin Sprint API is operating normally.",
  "estimatedResolution": null
}
```

### Block Data

#### GET `/api/latest`

Get the latest blocks from the blockchain.

**Query Parameters:**

- `limit` (optional): Number of blocks to return (default: 10, max: 100)
- `offset` (optional): Number of blocks to skip (default: 0)

**Response Example:**

```json
{
  "blocks": [
    {
      "hash": "000000000000000000024bead8df69990852c202db0e0097c1a12ea637d7e96d",
      "height": 850000,
      "timestamp": 1692896553,
      "txCount": 1985,
      "size": 1458735,
      "weight": 3993124,
      "confirmations": 1
    },
    {
      "hash": "00000000000000000001ebcd6175a227d980765b4af0ada581f6af1e73819ffe",
      "height": 849999,
      "timestamp": 1692895901,
      "txCount": 2105,
      "size": 1512476,
      "weight": 3998453,
      "confirmations": 2
    }
    // ... more blocks
  ],
  "status": "success",
  "executionTime": 25.34
}
```

#### GET `/api/v1/blocks/latest`

Enhanced version of the latest blocks endpoint with additional metadata.

**Query Parameters:**

- `limit` (optional): Number of blocks to return (default: 10, max: 100)
- `offset` (optional): Number of blocks to skip (default: 0)

**Response Example:**

```json
{
  "blocks": [
    {
      "hash": "000000000000000000024bead8df69990852c202db0e0097c1a12ea637d7e96d",
      "height": 850000,
      "timestamp": 1692896553,
      "txCount": 1985,
      "size": 1458735,
      "weight": 3993124,
      "confirmations": 1
    }
    // ... more blocks
  ],
  "count": 10,
  "offset": 0,
  "limit": 10,
  "status": "success",
  "timestamp": "2025-08-26T09:45:00Z"
}
```

#### GET `/api/v1/blocks/[hash]`

Get detailed information about a specific block by its hash.

**Response Example:**

```json
{
  "block": {
    "hash": "000000000000000000024bead8df69990852c202db0e0097c1a12ea637d7e96d",
    "confirmations": 1,
    "size": 1458735,
    "weight": 5834940,
    "height": 850000,
    "version": 536870912,
    "versionHex": "20000000",
    "merkleroot": "7fe07e76f5314e30d704e39bc3911e13e74a5e772f83418df80e169e05c36929",
    "time": 1692896553,
    "mediantime": 1692896253,
    "nonce": 1094540634,
    "bits": "1703a5b3",
    "difficulty": 53311599263588.1,
    "chainwork": "00000000000000000000000000000000000000001ec9886c4754f9a8e8778a6",
    "nTx": 1985,
    "previousblockhash": "00000000000000000001ebcd6175a227d980765b4af0ada581f6af1e73819ffe",
    "nextblockhash": "000000000000000000036b8a2c2e68de45b4e144d9fffd3a92311d272b62cffe",
    "strippedsize": 1166988,
    "transactions": [
      "7fe07e76f5314e30d704e39bc3911e13e74a5e772f83418df80e169e05c36929",
      // ... more transaction IDs (first 10 shown)
    ],
    "transactionCount": 1985,
    "totalFees": "0.55824973",
    "miner": "Foundry USA",
    "avgFeeRate": 12,
    "avgFeePerTx": 28123
  },
  "status": "success",
  "timestamp": "2025-08-26T09:45:00Z"
}
```

#### GET `/api/v1/blocks/height/[height]`

Get detailed information about a specific block by its height.

**Response Example:**
Same format as the block hash endpoint above.

### Transaction Data

#### GET `/api/v1/transaction/lookup`

Look up details of a specific transaction by its ID.

**Query Parameters:**

- `txid` (required): The transaction ID to look up

**Response Example:**

```json
{
  "transaction": {
    "txid": "7fe07e76f5314e30d704e39bc3911e13e74a5e772f83418df80e169e05c36929",
    "hash": "7fe07e76f5314e30d704e39bc3911e13e74a5e772f83418df80e169e05c36929",
    "version": 2,
    "size": 245,
    "weight": 980,
    "locktime": 0,
    "vin": [
      {
        "txid": "8f2334f54d37c9c27f518dc98bbba439439165e74a0e84132a67e526f12edad2",
        "vout": 1,
        "scriptSig": {
          "asm": "3045022100... [signature data]",
          "hex": "483045022100..."
        },
        "sequence": 4294967295
      }
    ],
    "vout": [
      {
        "value": 1.25000000,
        "n": 0,
        "scriptPubKey": {
          "asm": "OP_DUP OP_HASH160 [pubkey hash] OP_EQUALVERIFY OP_CHECKSIG",
          "hex": "76a914...88ac",
          "address": "bc1q8c6fshw2dlwun7ekn9qwf37cu2rn755upcp6n7",
          "type": "witness_v0_keyhash"
        }
      }
    ],
    "hex": "0200000001...",
    "blockhash": "000000000000000000024bead8df69990852c202db0e0097c1a12ea637d7e96d",
    "confirmations": 1,
    "time": 1692896553,
    "blocktime": 1692896553,
    "fee": "0.00015342",
    "fee_sat": 15342,
    "fee_per_vbyte": 14
  },
  "status": "success",
  "timestamp": "2025-08-26T09:45:00Z"
}
```

### Mempool Data

#### GET `/api/v1/mempool/summary`

Get a summary of the current mempool state.

**Response Example:**

```json
{
  "mempool": {
    "txCount": 12543,
    "size": 6271500,
    "bytes": 6271500,
    "usage": 6271500,
    "totalFees": "0.25621458",
    "feeRateDistribution": {
      "1-2": 987,
      "3-5": 2345,
      "6-10": 4521,
      "11-20": 2876,
      "21-50": 1456,
      "51+": 358
    },
    "minFeeRate": 1,
    "maxFeeRate": 87,
    "medianFeeRate": 8
  },
  "feeEstimates": {
    "fastestFee": 25,
    "halfHourFee": 15,
    "hourFee": 8,
    "economyFee": 3,
    "minimumFee": 1
  },
  "status": "success",
  "timestamp": "2025-08-26T09:45:00Z"
}
```

For Enterprise tier customers, additional fee estimation data is provided:

```json
{
  "feeEstimates": {
    "fastestFee": 25,
    "halfHourFee": 15,
    "hourFee": 8,
    "economyFee": 3,
    "minimumFee": 1,
    "confirmedBlocks": {
      "2": 28,
      "3": 22,
      "6": 12,
      "12": 7,
      "24": 4
    },
    "confirmedMinutes": {
      "20": 28,
      "40": 22,
      "60": 12,
      "120": 7,
      "240": 4
    },
    "historicalRates": {
      "1h": 18,
      "6h": 15,
      "12h": 12,
      "24h": 10
    }
  }
}
```

### Analytics

#### GET `/api/v1/analytics/summary`

Get comprehensive analytics about the Bitcoin network and API usage (Enterprise tier only).

**Response Example:**

```json
{
  "summary": {
    "currentBlockHeight": 850053,
    "totalPeers": 12,
    "blocksToday": 87,
    "uptimeSeconds": 43652,
    "turboModeEnabled": true,
    "apiVersion": "v1",
    "analyticsFeatures": [
      "Real-time block monitoring",
      "Peer connection tracking",
      "Performance metrics",
      "Predictive analytics"
    ]
  },
  "blockchainMetrics": {
    "averageBlockTime": 595.23,
    "averageBlockSize": 1325642,
    "averageTxPerBlock": 2534,
    "difficulty": 53392145230588.1,
    "hashrate": {
      "estimate": 567.24,
      "timestamp": "2025-08-26T09:45:00Z"
    }
  },
  "networkHealth": {
    "peerCount": 12,
    "peerVersions": {
      "24.0": 6,
      "23.0": 4,
      "22.0": 1,
      "other": 1
    },
    "geographicDistribution": {
      "North America": 5,
      "Europe": 4,
      "Asia": 2,
      "Other": 1
    },
    "connectionStability": "0.9854"
  },
  "performanceMetrics": {
    "systemLoad": {
      "cpu": "35.42",
      "memory": "72.18",
      "disk": "42.65"
    },
    "responseTimeMs": {
      "p50": 32,
      "p90": 78,
      "p99": 145
    },
    "errorRate": "0.0023",
    "successfulRequests": 985432,
    "failedRequests": 234
  },
  "apiUsage": {
    "requestsToday": 87542,
    "uniqueEndpoints": 8,
    "averageResponseTime": 123,
    "errorRate": "0.0254",
    "mostPopularEndpoint": "/api/v1/blocks"
  },
  "timestamp": "2025-08-26T09:45:00Z",
  "refreshInterval": 300,
  "status": "success"
}
```

### License Information

#### GET `/api/v1/license/info`

Get information about the current license and its limits.

**Response Example:**

```json
{
  "license": {
    "tier": "ENTERPRISE",
    "valid": true,
    "email": "customer@example.com",
    "company": "Example Corp",
    "key_id": "usr_12345abcde"
  },
  "limits": {
    "rate_limit_per_minute": 20000,
    "blocks_per_day": -1,
    "endpoints_available": 7,
    "mempool_access": true,
    "predictive_features": true,
    "stream_access": true
  },
  "usage_today": {
    "requests": 8754,
    "blocks_delivered": 1245,
    "last_request": "2025-08-26T09:45:00Z"
  },
  "performance": {
    "target_latency_ms": 200,
    "turbo_mode": true,
    "dedicated_infra": false
  },
  "api_version": "v1",
  "documentation": "https://docs.bitcoin-sprint.com/v1"
}
```

## Rate Limits

Rate limits are applied based on the user's tier:

- FREE: 100 requests per minute
- PRO: 2,000 requests per minute
- ENTERPRISE: 20,000 requests per minute
- ENTERPRISE_PLUS: 100,000 requests per minute

When the rate limit is exceeded, the API returns a 429 status code with a message indicating when the limit will reset.

## Error Responses

All error responses follow a consistent format:

```json
{
  "error": "Error Type",
  "message": "Detailed error message",
  "status": "error"
}
```

Common error types:

- Invalid API Key
- Rate Limit Exceeded
- Insufficient Tier
- Invalid Parameters
- Resource Not Found
- Internal Server Error

## Streaming API

Real-time updates are available via Server-Sent Events (SSE) at `/api/stream`. This endpoint requires authentication and supports different event types based on the user's tier.

**Query Parameters:**

- `events` (optional): Event types to listen for (default: "blocks")
- `mempool` (optional): Include mempool events (Enterprise tier only)

## License

This API and its documentation are proprietary and confidential. Unauthorized use, reproduction, or distribution is prohibited.

© 2025 Bitcoin Sprint. All Rights Reserved.
