# Block Deduplication System

## Overview

The Block Deduplication System prevents duplicate block processing in Bitcoin Sprint's relay system. By detecting and dropping duplicate block announcements from multiple nodes, it reduces CPU usage, memory consumption, and improves response time stability.

## Implementation

The system consists of:

1. **Deduper Component** - A fixed-capacity, TTL-based deduplication cache that:
   - Tracks recently seen block hashes
   - Automatically evicts older entries when capacity is exceeded
   - Performs periodic cleanup of expired entries
   - Maintains thread safety for concurrent access

2. **Integration Points**:
   - Integrated with both Ethereum and Solana relays
   - Intercepts block announcements at the earliest point in the pipeline
   - Records metrics on suppressed duplicates

## Benefits

- **Reduced Processing Load**: Eliminates redundant block processing, as the same block is often announced by multiple peers
- **Consistent Performance**: Prevents spikes in CPU/memory usage from duplicate processing
- **Metrics Visibility**: Tracks how many duplicates were suppressed
- **Memory-Bounded**: Fixed capacity with TTL ensures memory usage remains constant

## Configuration

The deduper is configured with:

- **Capacity**: 4096-8192 entries (configurable)
- **TTL**: 3-5 minutes default time-to-live for entries
- **Cleanup Interval**: Automatic cleanup runs every 1 minute

## Metrics

The system records the following Prometheus metrics:

- `relay_duplicate_blocks_suppressed_total{network="ethereum"}`: Count of duplicate Ethereum blocks suppressed
- `relay_duplicate_blocks_suppressed_total{network="solana"}`: Count of duplicate Solana blocks suppressed

## Testing

Run the deduplication test script to:

1. Visualize the deduplication process with:
   ```
   .\test-dedupe.ps1
   ```

2. Run a performance benchmark with:
   ```
   .\test-dedupe.ps1 -RunBenchmark -BlockCount 1000 -DuplicatePercent 70
   ```
