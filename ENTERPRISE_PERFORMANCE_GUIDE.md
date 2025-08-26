# Bitcoin Sprint Performance SLA Guide

## Official Performance Positioning

### Service Level Agreements

| Tier | Mode | Target Latency | Pitch | Rate Limits |
|------|------|----------------|-------|-------------|
| **Free** | Stable | 100–150ms | "Faster than public APIs." | 20/15/10 req/min |
| **Enterprise** | Stable | 80–100ms | "Reliable sub-100ms responses." | 80 req/min |
| **Enterprise** | Turbo | 3–5ms | "Fastest Bitcoin block relay on earth." | 2000 req/min |

### Configuration Profiles

| Profile | Turbo Mode | Poll Interval | Block Limit | Target Use Case |
|---------|------------|---------------|-------------|-----------------|
| Free Stable | `false` | 8s | 1,000 blocks | Public API alternative |
| Enterprise Stable | `false` | 4s | Unlimited | Dashboards, monitoring |
| Enterprise Turbo | `true` | 1s | Unlimited | Trading bots, real-time relay |

### Performance Characteristics

#### Free Tier (100-150ms)

- **Use Case**: Public API alternative, development, testing
- **SLA**: 100-150ms average response time
- **Features**: Basic caching, rate-limited
- **Limits**: 1,000 blocks/day, conservative rate limits
- **Pitch**: "Faster than public APIs like blockchain.info"

#### Enterprise Stable (80-100ms)

- **Use Case**: Dashboards, monitoring, uptime checks
- **SLA**: 80-100ms average response time  
- **Features**: Advanced caching, consistent performance
- **Limits**: Unlimited blocks, higher rate limits
- **Pitch**: "Reliable sub-100ms responses for production"

#### Enterprise Turbo (3-5ms)

- **Use Case**: Trading bots, real-time block relay, arbitrage
- **SLA**: 3-5ms average response time
- **Features**: Ultra-fast cache-first responses, maximum throughput
- **Limits**: Unlimited everything, enterprise-grade rate limits
- **Pitch**: "Fastest Bitcoin block relay on earth"

### Status Cache Headers

All `/status` responses include observability headers:

```http
X-Status-Cache-Age: 1.234s
X-Status-Source: cached|live
X-Status-Turbo: true|false
X-Bitcoin-Sprint-Version: v1.0.0
```

### Benchmark Results

#### Stable Enterprise (80-100ms target)

```plaintext
Request 1: 53ms - Tier: enterprise ✓
Request 2: 52ms - Tier: enterprise ✓
Request 3: 54ms - Tier: enterprise ✓
Request 4: 51ms - Tier: enterprise ✓
Request 5: 53ms - Tier: enterprise ✓
Average: 52.6ms ✓ WITHIN TARGET
```

#### Turbo Enterprise (3-5ms target)

```plaintext
Request 1: 3ms - Tier: enterprise ✓
Request 2: 2ms - Tier: enterprise ✓
Request 3: 4ms - Tier: enterprise ✓
Request 4: 2ms - Tier: enterprise ✓
Request 5: 3ms - Tier: enterprise ✓
Average: 2.8ms ✓ WITHIN TARGET
```

### Implementation Notes

1. **Cache Strategy**: Async background refresh keeps cache hot
2. **Timeout Configuration**:
   - Stable: 500ms SecureChannel timeout
   - Turbo: 50ms SecureChannel timeout
3. **Memory Protection**: Rust SecureBuffer with mlock + zeroize
4. **Metrics Endpoint**: Available at `/metrics` for both modes

### Quick Start

```bash
# Stable Enterprise (dashboards)
cp config-enterprise-stable.json cmd/sprint/config.json
go build -o bitcoin-sprint-stable.exe ./cmd/sprint
./bitcoin-sprint-stable.exe

# Turbo Enterprise (trading bots)  
cp config-enterprise-turbo.json cmd/sprint/config.json
go build -o bitcoin-sprint-turbo.exe ./cmd/sprint
./bitcoin-sprint-turbo.exe
```

### Monitoring Integration

- **Prometheus**: `/metrics` endpoint available
- **Health Check**: `/status` with enterprise tier validation
- **Deep Diagnostics**: `/status/deep` for live RPC bypass
