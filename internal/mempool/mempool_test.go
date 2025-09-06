package mempool

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMempool_BasicOperations(t *testing.T) {
	mempool := New()
	defer mempool.Stop()

	// Test Add and Contains
	mempool.Add("tx1")
	assert.True(t, mempool.Contains("tx1"))
	assert.False(t, mempool.Contains("tx2"))

	// Test Size
	assert.Equal(t, 1, mempool.Size())

	// Test All
	txids := mempool.All()
	assert.Len(t, txids, 1)
	assert.Contains(t, txids, "tx1")

	// Test Remove
	removed := mempool.Remove("tx1")
	assert.True(t, removed)
	assert.False(t, mempool.Contains("tx1"))
	assert.Equal(t, 0, mempool.Size())

	// Test Remove non-existent
	removed = mempool.Remove("tx_nonexistent")
	assert.False(t, removed)
}

func TestMempool_WithDetails(t *testing.T) {
	mempool := New()
	defer mempool.Stop()

	// Test AddWithDetails
	mempool.AddWithDetails("tx1", 250, 1, 0.001)

	entry, found := mempool.Get("tx1")
	require.True(t, found)
	assert.Equal(t, "tx1", entry.TxID)
	assert.Equal(t, 250, entry.Size)
	assert.Equal(t, 1, entry.Priority)
	assert.Equal(t, 0.001, entry.FeeRate)
	assert.False(t, entry.AddedAt.IsZero())
	assert.False(t, entry.ExpiresAt.IsZero())
}

func TestMempool_WithConfig(t *testing.T) {
	config := Config{
		MaxSize:         10,
		ExpiryTime:      1 * time.Second,
		CleanupInterval: 100 * time.Millisecond,
		ShardCount:      4,
	}

	mempool := NewWithConfig(config)
	defer mempool.Stop()

	// Test max size limit
	for i := 0; i < 15; i++ {
		mempool.Add(fmt.Sprintf("tx%d", i))
	}

	// Should not exceed max size
	assert.LessOrEqual(t, mempool.Size(), 10)
}

func TestMempool_WithMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewMempoolMetrics(reg)
	config := DefaultConfig()
	config.ExpiryTime = 100 * time.Millisecond
	config.CleanupInterval = 50 * time.Millisecond

	mempool := NewWithMetricsAndConfig(config, metrics)
	defer mempool.Stop()

	// Add some transactions
	mempool.AddWithDetails("tx1", 100, 1, 0.001)
	mempool.AddWithDetails("tx2", 200, 2, 0.002)

	// Check metrics
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, metricFamilies)

	// Wait for expiry and cleanup
	time.Sleep(200 * time.Millisecond)

	// Transactions should be expired
	assert.False(t, mempool.Contains("tx1"))
	assert.False(t, mempool.Contains("tx2"))
}

func TestMempool_Expiry(t *testing.T) {
	config := Config{
		MaxSize:         100,
		ExpiryTime:      50 * time.Millisecond,
		CleanupInterval: 25 * time.Millisecond,
		ShardCount:      2,
	}

	mempool := NewWithConfig(config)
	defer mempool.Stop()

	// Add transaction
	mempool.Add("tx1")
	assert.True(t, mempool.Contains("tx1"))
	assert.Equal(t, 1, mempool.Size())

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)

	// Transaction should be expired and cleaned up
	assert.False(t, mempool.Contains("tx1"))
	assert.Equal(t, 0, mempool.Size())
}

func TestMempool_Sharding(t *testing.T) {
	config := Config{
		MaxSize:         1000,
		ExpiryTime:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		ShardCount:      8,
	}

	mempool := NewWithConfig(config)
	defer mempool.Stop()

	// Add many transactions to test shard distribution
	txCount := 100
	for i := 0; i < txCount; i++ {
		mempool.Add(fmt.Sprintf("tx%d", i))
	}

	assert.Equal(t, txCount, mempool.Size())

	// Verify all transactions are retrievable
	for i := 0; i < txCount; i++ {
		txid := fmt.Sprintf("tx%d", i)
		assert.True(t, mempool.Contains(txid), "Transaction %s should exist", txid)
	}

	// Test stats
	stats := mempool.Stats()
	assert.Equal(t, txCount, stats["size"])
	assert.Equal(t, 8, stats["shard_count"])
	assert.NotNil(t, stats["shards"])
}

func TestMempool_ConcurrentAccess(t *testing.T) {
	mempool := New()
	defer mempool.Stop()

	const numGoroutines = 50
	const numOpsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // readers, writers, removers

	// Writers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				txid := fmt.Sprintf("tx%d_%d", id, j)
				mempool.AddWithDetails(txid, j*10, id, float64(j)*0.001)
			}
		}(i)
	}

	// Readers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				txid := fmt.Sprintf("tx%d_%d", id, j)
				mempool.Contains(txid)
				mempool.Get(txid)
			}
		}(i)
	}

	// Removers (remove some transactions)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine/2; j++ {
				txid := fmt.Sprintf("tx%d_%d", id, j*2)
				mempool.Remove(txid)
			}
		}(i)
	}

	wg.Wait()

	// Verify mempool is still functional
	mempool.Add("final_tx")
	assert.True(t, mempool.Contains("final_tx"))
}

func TestMempool_AllEntries(t *testing.T) {
	mempool := New()
	defer mempool.Stop()

	// Add transactions with different details
	mempool.AddWithDetails("tx1", 100, 1, 0.001)
	mempool.AddWithDetails("tx2", 200, 2, 0.002)
	mempool.AddWithDetails("tx3", 300, 3, 0.003)

	entries := mempool.AllEntries()
	assert.Len(t, entries, 3)

	// Verify entries contain correct data
	txidToEntry := make(map[string]*TransactionEntry)
	for _, entry := range entries {
		txidToEntry[entry.TxID] = entry
	}

	assert.Equal(t, 100, txidToEntry["tx1"].Size)
	assert.Equal(t, 1, txidToEntry["tx1"].Priority)
	assert.Equal(t, 0.001, txidToEntry["tx1"].FeeRate)

	assert.Equal(t, 200, txidToEntry["tx2"].Size)
	assert.Equal(t, 2, txidToEntry["tx2"].Priority)
	assert.Equal(t, 0.002, txidToEntry["tx2"].FeeRate)
}

func TestMempool_Clear(t *testing.T) {
	mempool := New()
	defer mempool.Stop()

	// Add some transactions
	for i := 0; i < 10; i++ {
		mempool.Add(fmt.Sprintf("tx%d", i))
	}

	assert.Equal(t, 10, mempool.Size())

	// Clear mempool
	mempool.Clear()

	assert.Equal(t, 0, mempool.Size())
	assert.Empty(t, mempool.All())
}

func TestMempool_Stop(t *testing.T) {
	mempool := New()

	// Add some transactions
	mempool.Add("tx1")
	mempool.Add("tx2")

	assert.Equal(t, 2, mempool.Size())

	// Stop mempool
	err := mempool.Stop()
	require.NoError(t, err)

	// Try to stop again (should return error)
	err = mempool.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already stopped")

	// Operations after stop should not add new transactions
	mempool.Add("tx3")
	// Size might still be 2 depending on timing
}

func TestMempool_DefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, 100000, config.MaxSize)
	assert.Equal(t, 5*time.Minute, config.ExpiryTime)
	assert.Equal(t, 30*time.Second, config.CleanupInterval)
	assert.Equal(t, 16, config.ShardCount)
}

func TestMempool_StatsWithShards(t *testing.T) {
	config := Config{
		MaxSize:         50,
		ExpiryTime:      1 * time.Minute,
		CleanupInterval: 10 * time.Second,
		ShardCount:      4,
	}

	mempool := NewWithConfig(config)
	defer mempool.Stop()

	// Add transactions
	for i := 0; i < 20; i++ {
		mempool.Add(fmt.Sprintf("tx%d", i))
	}

	stats := mempool.Stats()
	
	assert.Equal(t, 20, stats["size"])
	assert.Equal(t, 50, stats["max_size"])
	assert.Equal(t, 4, stats["shard_count"])
	assert.Equal(t, "1m0s", stats["expiry_time"])
	assert.Equal(t, "10s", stats["cleanup_interval"])

	// Check shard stats
	shardStats := stats["shards"].([]map[string]interface{})
	assert.Len(t, shardStats, 4)

	totalShardSize := 0
	for _, shardStat := range shardStats {
		shardSize := shardStat["size"].(int)
		totalShardSize += shardSize
		assert.GreaterOrEqual(t, shardSize, 0)
	}

	assert.Equal(t, 20, totalShardSize)
}

func TestMempool_UpdateExistingTransaction(t *testing.T) {
	mempool := New()
	defer mempool.Stop()

	// Add transaction
	mempool.AddWithDetails("tx1", 100, 1, 0.001)
	assert.Equal(t, 1, mempool.Size())

	// Add same transaction again with different details
	mempool.AddWithDetails("tx1", 200, 2, 0.002)
	assert.Equal(t, 1, mempool.Size()) // Size should not change

	// Verify details were updated
	entry, found := mempool.Get("tx1")
	require.True(t, found)
	assert.Equal(t, 200, entry.Size)
	assert.Equal(t, 2, entry.Priority)
	assert.Equal(t, 0.002, entry.FeeRate)
}

func BenchmarkMempool_Add(b *testing.B) {
	mempool := New()
	defer mempool.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			txid := fmt.Sprintf("tx%d", i)
			mempool.Add(txid)
			i++
		}
	})
}

func BenchmarkMempool_Contains(b *testing.B) {
	mempool := New()
	defer mempool.Stop()

	// Pre-populate mempool
	for i := 0; i < 10000; i++ {
		mempool.Add(fmt.Sprintf("tx%d", i))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			txid := fmt.Sprintf("tx%d", i%10000)
			mempool.Contains(txid)
			i++
		}
	})
}

func BenchmarkMempool_Get(b *testing.B) {
	mempool := New()
	defer mempool.Stop()

	// Pre-populate mempool
	for i := 0; i < 10000; i++ {
		mempool.AddWithDetails(fmt.Sprintf("tx%d", i), i*10, i%10, float64(i)*0.0001)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			txid := fmt.Sprintf("tx%d", i%10000)
			mempool.Get(txid)
			i++
		}
	})
}
