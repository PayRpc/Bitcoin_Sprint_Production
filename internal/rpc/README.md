# Enhanced RPC Service

The enhanced RPC service provides high-performance Bitcoin RPC operations using a generic processing engine with the following improvements:

## Features

- **Engine-Based Processing**: Uses a bounded worker pool with configurable concurrency
- **Persistent State Management**: Tracks last processed block and failed transactions
- **LRU Caching**: Efficient caching of processed messages
- **Deduplication**: Prevents reprocessing of seen transactions
- **Batch Operations**: Optimized batch RPC calls with retry logic
- **Comprehensive Metrics**: Prometheus-compatible metrics for monitoring
- **Graceful Shutdown**: Proper cleanup and state persistence

## Architecture

### Core Components

1. **Engine**: Generic task processing engine with worker pools
2. **BitcoinTask**: Specialized task for Bitcoin RPC operations
3. **StateStore**: Persistent storage for processing state
4. **SeenStore**: Deduplication store for processed items
5. **ResultCache**: LRU cache for recent results

### Task Types

- **BitcoinTask**: Processes blocks and transactions from Bitcoin RPC
- **CustomRPCTask**: Template for custom RPC operations
- **BatchRPCTask**: Handles multiple RPC queries efficiently

## Configuration

The service uses the existing RPC configuration from `config.go`:

```go
RPCEnabled         bool          // Enable/disable RPC operations
RPCURL             string        // Bitcoin RPC endpoint
RPCUsername        string        // RPC authentication
RPCPassword        string        // RPC authentication
RPCTimeout         time.Duration // Request timeout
RPCBatchSize       int           // Batch processing size
RPCRetryAttempts   int           // Retry attempts on failure
RPCRetryMaxWait    time.Duration // Maximum retry wait time
RPCSkipMempool     bool          // Skip mempool processing
RPCFailedTxFile    string        // Failed transaction log
```

## Usage Examples

### Basic Backfill

```go
// Create and start the service
rpcService, err := rpc.NewEnhancedRPCService(cfg, logger)
if err := rpcService.Start(ctx); err != nil {
    log.Fatal(err)
}

// Submit a backfill task
err = rpcService.SubmitBackfillTask("backfill-1", 100)
```

### Custom RPC Task

```go
// Create a custom task
task := rpc.NewCustomRPCTask("custom-1", rpcURL, user, pass, "getblockchaininfo")

// Submit to engine
engine.Submit(task)
```

### Batch Operations

```go
queries := []string{"getblockhash 100", "getblockhash 101"}
batchTask := rpc.NewBatchRPCTask("batch-1", rpcURL, user, pass, queries)
engine.Submit(batchTask)
```

## Metrics

The service exposes metrics at `:9091/metrics`:

- `engine_tasks_queued_total`: Total tasks queued
- `engine_tasks_processed_total`: Total tasks processed
- `engine_task_errors_total`: Total task errors
- `engine_task_processing_seconds`: Task processing duration
- `engine_messages_produced_total`: Messages produced
- `engine_queue_drops_total`: Tasks dropped due to full queue

## Performance Benefits

1. **Concurrent Processing**: Multiple workers process tasks simultaneously
2. **Batch RPC Calls**: Reduces network overhead with batched requests
3. **Intelligent Caching**: Avoids reprocessing recent data
4. **Retry Logic**: Automatic retry with exponential backoff
5. **State Persistence**: Resumes processing from last known state
6. **Memory Efficiency**: LRU cache prevents memory bloat

## Integration

The enhanced RPC service is automatically integrated into the main Bitcoin Sprint application and starts alongside other services. It provides:

- Improved backfill performance
- Better error handling and recovery
- Enhanced monitoring and observability
- Scalable architecture for future extensions

## Future Enhancements

- WebSocket support for real-time updates
- Multi-chain RPC support
- Advanced query optimization
- Machine learning-based performance tuning
