# Bitcoin Sprint Turbo Mode Implementation Guide

## Overview

Bitcoin Sprint now supports tier-based performance optimization with **Turbo Mode** for ultra-low latency block relay. This implementation provides configurable performance tiers from Free to Enterprise with automatic optimization based on the selected tier.

## Performance Tiers

| Tier | Block Relay Latency | Write Deadline | Buffer Size | Features |
|------|-------------------|----------------|-------------|----------|
| **Free** | <500ms | 2s | 512 | Basic TCP relay |
| **Pro** | <100ms | 1.5s | 1,280 | Enhanced relay |
| **Business** | <50ms | 1s | 1,536 | Priority processing |
| **Turbo** | <10ms | 500µs | 2,048 | Shared memory + zero-copy |
| **Enterprise** | <5ms | 200µs | 4,096 | Kernel bypass + dedicated infra |

## Quick Start

### 1. Activate Turbo Mode

Set the environment variable:
```bash
export TIER=turbo
```

Or create a `.env` file:
```bash
TIER=turbo
USE_SHARED_MEMORY=true
USE_DIRECT_P2P=true
USE_MEMORY_CHANNEL=true
OPTIMIZE_SYSTEM=true
```

### 2. Run Bitcoin Sprint

```bash
# Using the demo script (Windows)
.\demo-turbo-mode.ps1

# Or manually
go run cmd/sprintd/main.go
```

### 3. Verify Turbo Status

```bash
curl http://localhost:8080/api/turbo-status
```

## Implementation Details

### Configuration System

The `internal/config/config.go` now includes:

```go
type Tier string

const (
    TierFree       Tier = "free"
    TierPro        Tier = "pro"
    TierBusiness   Tier = "business"
    TierTurbo      Tier = "turbo"
    TierEnterprise Tier = "enterprise"
)

type Config struct {
    // ... existing fields ...
    
    // Turbo mode enhancements
    Tier             Tier
    WriteDeadline    time.Duration
    UseSharedMemory  bool
    BlockBufferSize  int
    EnableKernelBypass bool
}
```

### Tier-Aware Processing

The main application automatically selects processing mode based on tier:

- **Turbo/Enterprise**: Uses `runMemoryOptimizedSprint()` with shared memory and zero-copy operations
- **Free/Pro/Business**: Uses `runStandardSprint()` with conventional relay mechanisms

### Broadcaster with Tier-Aware Delivery

The `internal/broadcaster/broadcaster.go` implements:

- **Buffer overwrite strategy** for Turbo/Enterprise (never miss blocks)
- **Best effort delivery** for lower tiers
- **Performance monitoring** with detailed metrics
- **Dynamic buffer sizing** based on tier

### Zero-Copy Notifications

For Turbo and Enterprise tiers:
- Pre-allocated buffers to avoid heap allocations
- Strict deadline enforcement (500µs for Turbo, 200µs for Enterprise)
- Shared memory channels for ultra-low latency
- Fallback to standard mechanisms with timeout

## Environment Variables

| Variable | Description | Turbo Default | Enterprise Default |
|----------|-------------|---------------|-------------------|
| `TIER` | Performance tier | `turbo` | `enterprise` |
| `USE_SHARED_MEMORY` | Enable shared memory | `true` | `true` |
| `USE_DIRECT_P2P` | Enable direct P2P bypass | `true` | `true` |
| `USE_MEMORY_CHANNEL` | Enable memory channels | `true` | `true` |
| `OPTIMIZE_SYSTEM` | Apply system optimizations | `true` | `true` |
| `ENABLE_KERNEL_BYPASS` | Enable kernel bypass (future) | `false` | `false` |

## API Endpoints

### Turbo Status Endpoint

**GET** `/api/turbo-status`

Returns current tier configuration and performance targets:

```json
{
  "tier": "turbo",
  "turboModeEnabled": true,
  "writeDeadline": "500µs",
  "useSharedMemory": true,
  "blockBufferSize": 2048,
  "features": [
    "Shared Memory",
    "Direct P2P",
    "Memory Channel",
    "System Optimizations"
  ],
  "performanceTargets": {
    "blockRelayLatency": "<10ms (Turbo)",
    "writeDeadline": "500µs",
    "bufferStrategy": "Overwrite old events (never miss)",
    "peerNotification": "Zero-copy shared memory"
  }
}
```

## Performance Monitoring

### Logging

Turbo mode provides detailed performance logging:

```
⚡ Turbo mode enabled: writeDeadline=500µs sharedMemory=true blockBufferSize=2048
⚡ Starting memory-optimized sprint with tight deadlines
Turbo relay successful: elapsed=245µs blockHash=000000000...
```

### Metrics

The broadcaster tracks:
- Delivery success rate per tier
- Buffer utilization by tier
- Overwrite frequency for Turbo customers
- Average processing time

## Deployment Strategies

### Dedicated Turbo Infrastructure

For production Turbo deployments:

```bash
# High-performance Bitcoin node configuration
dbcache=8000
maxconnections=256
zmqpubhashblock=tcp://127.0.0.1:28332

# Route Turbo customers to dedicated nodes
BITCOIN_NODE=turbo-node-1.internal:8333
ZMQ_NODE=turbo-node-1.internal:28332
```

### Load Balancing

- **Free/Pro customers**: Standard infrastructure
- **Business customers**: Enhanced infrastructure
- **Turbo customers**: Dedicated low-latency infrastructure
- **Enterprise customers**: Dedicated infrastructure with SLA guarantees

## Future Enhancements

### Planned Features

1. **Kernel Bypass** (DPDK integration)
2. **User-space TCP stack** for Enterprise tier
3. **Hardware acceleration** for cryptographic operations
4. **Predictive pre-loading** based on mempool analysis
5. **Geographic routing** to nearest Turbo nodes

### Performance Targets

- **Turbo Goal**: Consistent sub-5ms block relay
- **Enterprise Goal**: Sub-2ms block relay with 99.9% SLA
- **Scalability**: Support 10,000+ concurrent Turbo connections

## Troubleshooting

### Common Issues

1. **High latency despite Turbo mode**
   - Check `TIER` environment variable
   - Verify shared memory is enabled
   - Monitor system resources

2. **Buffer overflows**
   - Increase `BlockBufferSize` in config
   - Check network connectivity
   - Monitor peer connection count

3. **Failed zero-copy operations**
   - Fallback to standard relay is automatic
   - Check logs for timeout warnings
   - Verify system optimization settings

### Performance Validation

```bash
# Check current tier
curl http://localhost:8080/api/turbo-status | jq '.tier'

# Monitor health
curl http://localhost:8080/api/health

# Check logs for Turbo activation
journalctl -u bitcoin-sprint | grep "Turbo mode enabled"
```

## License and Support

- **Free/Pro**: Community support
- **Business**: Email support with 48-hour response
- **Turbo**: Priority support with 24-hour response
- **Enterprise**: Dedicated support with 4-hour SLA

This implementation provides a solid foundation for tier-based performance optimization while maintaining backward compatibility and operational simplicity.
