# Bitcoin Sprint - Tier Configuration Guide

This guide explains the 5 official tiers available in Bitcoin Sprint and how to test them.

## Available Tiers

### 1. Free Tier (`TIER=free`)
- **Target**: Development and basic testing
- **Rate Limits**: 1 req/sec, 1000 req/hour
- **Resources**: Limited concurrent streams (1), 10MB data limit
- **Features**: Basic monitoring, no advanced optimizations
- **Use Case**: Getting started, basic functionality testing

### 2. Pro Tier (`TIER=pro`)
- **Target**: Professional users and small teams
- **Rate Limits**: 10 req/sec, 10,000 req/hour
- **Resources**: 5 concurrent streams, 100MB data limit
- **Features**: Direct P2P, basic entropy monitoring
- **Use Case**: Production applications with moderate load

### 3. Business Tier (`TIER=business`)
- **Target**: Business applications and medium teams
- **Rate Limits**: 50 req/sec, 50,000 req/hour
- **Resources**: 20 concurrent streams, 500MB data limit
- **Features**: Memory channels, enhanced entropy monitoring, hardware fingerprinting
- **Use Case**: High-traffic business applications

### 4. Turbo Tier (`TIER=turbo`)
- **Target**: High-performance applications
- **Rate Limits**: 100 req/sec, 100,000 req/hour
- **Resources**: 50 concurrent streams, 1GB data limit
- **Features**: Ultra-low latency (1-3ms), shared memory, kernel bypass
- **Use Case**: Real-time trading, high-frequency applications

### 5. Enterprise Tier (`TIER=enterprise`)
- **Target**: Enterprise-grade applications
- **Rate Limits**: 500 req/sec, 500,000 req/hour
- **Resources**: 100 concurrent streams, 5GB data limit
- **Features**: Maximum performance, TLS/mTLS, advanced security, hardware monitoring
- **Use Case**: Large-scale enterprise deployments

## How to Switch Tiers

### Method 1: Using the Tier Switcher (Recommended)

```bash
# Using PowerShell (recommended for Windows)
.\switch-tier.ps1 turbo
.\switch-tier.ps1 enterprise
.\switch-tier.ps1 free
.\switch-tier.ps1 pro
```

### Method 2: Manual Environment Setup

```bash
# Copy the desired tier configuration
cp .env.turbo .env
cp .env.enterprise .env

# Or set environment variable directly
export TIER=turbo
export TIER=enterprise
```

### Method 3: Runtime Environment Variable

```bash
# Set tier at runtime
TIER=turbo .\start-dev.ps1
TIER=enterprise go run cmd/sprintd/main.go
```

## Testing Each Tier

### Performance Testing

```bash
# Test latency for each tier
TIER=free ./benchmark-latency.sh
TIER=turbo ./benchmark-latency.sh
TIER=enterprise ./benchmark-latency.sh
```

### Load Testing

```bash
# Test rate limits for each tier
TIER=free ./load-test.sh --requests 1000
TIER=pro ./load-test.sh --requests 10000
TIER=enterprise ./load-test.sh --requests 100000
```

### Feature Testing

```bash
# Test tier-specific features
TIER=turbo ./test-turbo-features.sh
TIER=enterprise ./test-enterprise-security.sh
```

## Configuration Files

Each tier has its own environment file:

- `.env.free` - Free tier configuration
- `.env.pro` - Pro tier configuration
- `.env.business` - Business tier configuration
- `.env.turbo` - Turbo tier configuration
- `.env.enterprise` - Enterprise tier configuration

## Key Differences Between Tiers

| Feature | Free | Pro | Business | Turbo | Enterprise |
|---------|------|-----|----------|-------|------------|
| Requests/sec | 1 | 10 | 50 | 100 | 500 |
| Concurrent Streams | 1 | 5 | 20 | 50 | 100 |
| Data Limit (MB) | 10 | 100 | 500 | 1000 | 5000 |
| Direct P2P | No | Yes | Yes | Yes | Yes |
| Memory Channel | No | No | Yes | Yes | Yes |
| Kernel Bypass | No | No | No | No | Yes |
| Hardware Monitoring | No | No | Yes | Yes | Yes |
| TLS/mTLS | No | No | No | No | Yes |

## Monitoring Tier Performance

Each tier includes different monitoring capabilities:

```bash
# Check current tier status
curl http://localhost:8080/status

# Monitor performance metrics
curl http://localhost:9090/metrics

# Check entropy monitoring (Business+ tiers)
curl http://localhost:8080/entropy-status
```

## Troubleshooting

### Common Issues

1. **Tier not applying**: Make sure to restart the application after switching tiers
2. **Rate limits not working**: Check that the correct environment file is active
3. **Performance not improving**: Verify that tier-specific optimizations are enabled

### Logs

Each tier logs to a separate file:
- `logs/free_tier.log`
- `logs/pro_tier.log`
- `logs/business_tier.log`
- `logs/turbo.log`
- `logs/enterprise_tier.log`

## Development Notes

- All tiers share the same codebase but apply different configurations
- Turbo and Enterprise tiers include performance optimizations that may require specific hardware
- Rate limits are enforced at the application level
- Database files are tier-specific to avoid conflicts during testing
