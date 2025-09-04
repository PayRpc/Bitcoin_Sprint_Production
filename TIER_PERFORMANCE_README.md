# Bitcoin Sprint Tier Performance Comparison

This directory contains tools for comprehensive performance testing and comparison across all Bitcoin Sprint tiers.

## Available Tools

### 1. `tier-performance-comparison.ps1`
**Main benchmarking script** - Tests all tiers systematically with configurable parameters.

**Usage:**
```powershell
# Test all tiers with default settings (30s per tier, 5 concurrent requests)
.\tier-performance-comparison.ps1

# Test specific tiers only
.\tier-performance-comparison.ps1 -Tiers "free", "pro", "enterprise"

# Custom test duration and concurrency
.\tier-performance-comparison.ps1 -TestDuration 60 -ConcurrentRequests 10

# Skip build step (if already built)
.\tier-performance-comparison.ps1 -SkipBuild

# Keep application running after test
.\tier-performance-comparison.ps1 -KeepRunning
```

### 2. `run-tier-test.ps1`
**Quick test runner** - Simplified interface for common test scenarios.

**Usage:**
```powershell
# Fast test (10 seconds per tier)
.\run-tier-test.ps1 -Fast

# Full comprehensive test (60 seconds per tier)
.\run-tier-test.ps1 -Full

# Standard test (30 seconds per tier)
.\run-tier-test.ps1

# Custom settings
.\run-tier-test.ps1 -Custom -Duration 45 -Concurrent 8
```

### 3. `analyze-performance.ps1`
**Performance analysis tool** - Analyzes test results and provides insights.

**Usage:**
```powershell
# Analyze latest results automatically
.\analyze-performance.ps1

# Analyze specific results file
.\analyze-performance.ps1 -ResultsFile "tier-results-20241201.json"

# Generate detailed report
.\analyze-performance.ps1 -GenerateReport

# Compare with baseline
.\analyze-performance.ps1 -CompareWithBaseline
```

## Tier Specifications

| Tier | Rate Limit | Expected Latency | Cost |
|------|------------|------------------|------|
| **Free** | 1 req/sec | 2000-5000ms | $0 |
| **Pro** | 10 req/sec | 500-1200ms | $10/month |
| **Business** | 50 req/sec | 200-800ms | $50/month |
| **Turbo** | 100 req/sec | 50-300ms | $100/month |
| **Enterprise** | 500 req/sec | 10-150ms | $500/month |

## Quick Start

1. **Run a fast comparison test:**
   ```powershell
   .\run-tier-test.ps1 -Fast
   ```

2. **Run comprehensive analysis:**
   ```powershell
   .\run-tier-test.ps1 -Full
   .\analyze-performance.ps1 -GenerateReport
   ```

3. **Test specific tiers:**
   ```powershell
   .\tier-performance-comparison.ps1 -Tiers "free", "enterprise" -TestDuration 60
   ```

## Test Results

The tools will provide:
- **Latency metrics**: Average, P95, P99, min/max
- **Throughput**: Requests per second
- **Success rates**: Percentage of successful requests
- **Error analysis**: Common failure patterns
- **Performance grades**: How each tier performs vs expectations
- **Cost-benefit analysis**: Performance vs pricing comparison

## Output Files

- `tier-performance-results-YYYYMMDD-HHMMSS.json`: Raw test data
- `tier-performance-analysis-YYYYMMDD-HHMMSS.json`: Analysis report
- Console output with real-time results and recommendations

## Tips

- **Fast tests** are good for quick comparisons
- **Full tests** provide more accurate results but take longer
- **Enterprise tier** typically shows 10-100x better performance than Free tier
- Results are saved automatically for later analysis
- Use `-SkipBuild` if you've already built the application

## Troubleshooting

- Ensure all `.env.tier` files exist for the tiers you want to test
- Make sure the application builds successfully before testing
- Check that port 8080 is available
- For high-concurrency tests, ensure sufficient system resources
