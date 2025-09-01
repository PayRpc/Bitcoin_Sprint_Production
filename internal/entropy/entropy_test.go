package entropy

import (
	"bytes"
	"testing"
)

// TestFastEntropyBasic tests basic FastEntropy functionality
func TestFastEntropyBasic(t *testing.T) {
	entropy, err := FastEntropy()
	if err != nil {
		t.Fatalf("FastEntropy failed: %v", err)
	}
	if len(entropy) != 32 {
		t.Fatalf("Expected 32 bytes, got %d", len(entropy))
	}

	// Check that it's not all zeros
	allZero := true
	for _, b := range entropy {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Fatal("Entropy should not be all zeros")
	}
}

// TestHybridEntropyBasic tests basic HybridEntropy functionality
func TestHybridEntropyBasic(t *testing.T) {
	entropy, err := HybridEntropy()
	if err != nil {
		t.Fatalf("HybridEntropy failed: %v", err)
	}

	if len(entropy) != 32 {
		t.Fatalf("Expected 32 bytes, got %d", len(entropy))
	}

	// Check that it's not all zeros
	allZero := true
	for _, b := range entropy {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Fatal("Entropy should not be all zeros")
	}
}

// TestEntropyVariability tests that different calls produce different results
func TestEntropyVariability(t *testing.T) {
	entropy1, err := FastEntropy()
	if err != nil {
		t.Fatalf("First FastEntropy failed: %v", err)
	}
	entropy2, err := FastEntropy()
	if err != nil {
		t.Fatalf("Second FastEntropy failed: %v", err)
	}

	if bytes.Equal(entropy1, entropy2) {
		t.Error("FastEntropy should produce different values on different calls")
	}
}
