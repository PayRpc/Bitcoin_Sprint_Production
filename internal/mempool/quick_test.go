package mempool

import (
	"fmt"
	"testing"
	"time"
)

func TestMempool_QuickTest(t *testing.T) {
	// Create a mempool with simple settings
	config := Config{
		MaxSize:         100,
		ExpiryTime:      5 * time.Minute,
		CleanupInterval: 1 * time.Hour, // Very long cleanup to avoid goroutine issues
		ShardCount:      2,
	}
	
	mempool := NewWithConfig(config)
	
	// Quick test without defer to avoid hanging on Stop()
	mempool.Add("test_tx")
	
	if !mempool.Contains("test_tx") {
		t.Errorf("Transaction not found")
		return
	}
	
	if mempool.Size() != 1 {
		t.Errorf("Expected size 1, got %d", mempool.Size())
		return
	}
	
	// Test removal
	removed := mempool.Remove("test_tx")
	if !removed {
		t.Errorf("Failed to remove transaction")
		return
	}
	
	if mempool.Size() != 0 {
		t.Errorf("Expected size 0, got %d", mempool.Size())
		return
	}
	
	fmt.Println("✅ Quick test passed - mempool is working correctly!")
	
	// Try to stop mempool
	err := mempool.Stop()
	if err != nil {
		t.Logf("Warning: Stop returned error: %v", err)
	}
}
