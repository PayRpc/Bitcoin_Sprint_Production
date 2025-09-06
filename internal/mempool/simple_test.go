package mempool

import (
	"fmt"
	"testing"
	"time"
)

func TestMempool_BasicOperations_Simple(t *testing.T) {
	mempool := New()
	defer mempool.Stop()

	// Test Add and Contains
	mempool.Add("tx1")
	if !mempool.Contains("tx1") {
		t.Errorf("Expected mempool to contain tx1")
	}
	if mempool.Contains("tx2") {
		t.Errorf("Expected mempool to not contain tx2")
	}

	// Test Size
	if mempool.Size() != 1 {
		t.Errorf("Expected size 1, got %d", mempool.Size())
	}

	// Test All
	txids := mempool.All()
	if len(txids) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(txids))
	}
	found := false
	for _, txid := range txids {
		if txid == "tx1" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected to find tx1 in All() result")
	}

	// Test Remove
	removed := mempool.Remove("tx1")
	if !removed {
		t.Errorf("Expected Remove to return true")
	}
	if mempool.Contains("tx1") {
		t.Errorf("Expected mempool to not contain tx1 after removal")
	}
	if mempool.Size() != 0 {
		t.Errorf("Expected size 0 after removal, got %d", mempool.Size())
	}

	fmt.Println("✅ Basic operations test passed")
}

func TestMempool_WithDetails_Simple(t *testing.T) {
	mempool := New()
	defer mempool.Stop()

	// Test AddWithDetails
	mempool.AddWithDetails("tx1", 250, 1, 0.001)

	entry, found := mempool.Get("tx1")
	if !found {
		t.Errorf("Expected to find tx1")
	}
	if entry.TxID != "tx1" {
		t.Errorf("Expected TxID tx1, got %s", entry.TxID)
	}
	if entry.Size != 250 {
		t.Errorf("Expected Size 250, got %d", entry.Size)
	}
	if entry.Priority != 1 {
		t.Errorf("Expected Priority 1, got %d", entry.Priority)
	}
	if entry.FeeRate != 0.001 {
		t.Errorf("Expected FeeRate 0.001, got %f", entry.FeeRate)
	}
	if entry.AddedAt.IsZero() {
		t.Errorf("Expected AddedAt to be set")
	}
	if entry.ExpiresAt.IsZero() {
		t.Errorf("Expected ExpiresAt to be set")
	}

	fmt.Println("✅ Details test passed")
}

func TestMempool_Expiry_Simple(t *testing.T) {
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
	if !mempool.Contains("tx1") {
		t.Errorf("Expected mempool to contain tx1")
	}
	if mempool.Size() != 1 {
		t.Errorf("Expected size 1, got %d", mempool.Size())
	}

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)

	// Transaction should be expired and cleaned up
	if mempool.Contains("tx1") {
		t.Errorf("Expected tx1 to be expired")
	}
	if mempool.Size() != 0 {
		t.Errorf("Expected size 0 after expiry, got %d", mempool.Size())
	}

	fmt.Println("✅ Expiry test passed")
}

func TestMempool_Stats_Simple(t *testing.T) {
	config := Config{
		MaxSize:         1000,
		ExpiryTime:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		ShardCount:      8,
	}

	mempool := NewWithConfig(config)
	defer mempool.Stop()

	// Add transactions
	for i := 0; i < 20; i++ {
		mempool.Add(fmt.Sprintf("tx%d", i))
	}

	if mempool.Size() != 20 {
		t.Errorf("Expected size 20, got %d", mempool.Size())
	}

	// Test stats
	stats := mempool.Stats()
	if stats["size"] != 20 {
		t.Errorf("Expected stats size 20, got %v", stats["size"])
	}
	if stats["shard_count"] != 8 {
		t.Errorf("Expected shard_count 8, got %v", stats["shard_count"])
	}
	if stats["shards"] == nil {
		t.Errorf("Expected shards stats to be populated")
	}

	fmt.Println("✅ Stats test passed")
}

func TestMempool_Clear_Simple(t *testing.T) {
	mempool := New()
	defer mempool.Stop()

	// Add some transactions
	for i := 0; i < 10; i++ {
		mempool.Add(fmt.Sprintf("tx%d", i))
	}

	if mempool.Size() != 10 {
		t.Errorf("Expected size 10, got %d", mempool.Size())
	}

	// Clear mempool
	mempool.Clear()

	if mempool.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", mempool.Size())
	}
	
	txids := mempool.All()
	if len(txids) != 0 {
		t.Errorf("Expected no transactions after clear, got %d", len(txids))
	}

	fmt.Println("✅ Clear test passed")
}
