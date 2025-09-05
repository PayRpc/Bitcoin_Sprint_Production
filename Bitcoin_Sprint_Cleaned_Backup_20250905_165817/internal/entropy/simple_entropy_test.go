package entropy

import (
	"bytes"
	"testing"
)

// Test the simple Go-based entropy functions
func TestSimpleEntropy(t *testing.T) {
	data, err := SimpleEntropy()
	if err != nil {
		t.Fatalf("SimpleEntropy failed: %v", err)
	}

	if len(data) != 32 {
		t.Errorf("Expected 32 bytes, got %d", len(data))
	}

	// Test that successive calls produce different results
	data2, err := SimpleEntropy()
	if err != nil {
		t.Fatalf("Second SimpleEntropy failed: %v", err)
	}

	if bytes.Equal(data, data2) {
		t.Error("Successive calls to SimpleEntropy produced identical results")
	}
}

func TestTimingEntropy(t *testing.T) {
	data, err := TimingEntropy()
	if err != nil {
		t.Fatalf("TimingEntropy failed: %v", err)
	}

	if len(data) != 32 {
		t.Errorf("Expected 32 bytes, got %d", len(data))
	}

	// Test variability
	data2, err := TimingEntropy()
	if err != nil {
		t.Fatalf("Second TimingEntropy failed: %v", err)
	}

	if bytes.Equal(data, data2) {
		t.Error("Successive calls to TimingEntropy produced identical results")
	}
}

func TestEnhancedEntropy(t *testing.T) {
	data, err := EnhancedEntropy()
	if err != nil {
		t.Fatalf("EnhancedEntropy failed: %v", err)
	}

	if len(data) != 32 {
		t.Errorf("Expected 32 bytes, got %d", len(data))
	}

	// Test variability
	data2, err := EnhancedEntropy()
	if err != nil {
		t.Fatalf("Second EnhancedEntropy failed: %v", err)
	}

	if bytes.Equal(data, data2) {
		t.Error("Successive calls to EnhancedEntropy produced identical results")
	}
}

func TestSimpleEntropyDistribution(t *testing.T) {
	// Test that entropy has reasonable distribution
	const samples = 100
	results := make([][]byte, samples)

	for i := 0; i < samples; i++ {
		var err error
		results[i], err = EnhancedEntropy()
		if err != nil {
			t.Fatalf("EnhancedEntropy failed at sample %d: %v", i, err)
		}
	}

	// Check that we don't have duplicates
	for i := 0; i < samples; i++ {
		for j := i + 1; j < samples; j++ {
			if bytes.Equal(results[i], results[j]) {
				t.Errorf("Found duplicate entropy at samples %d and %d", i, j)
			}
		}
	}

	// Basic randomness check - count zero bytes
	zeroCount := 0
	totalBytes := samples * 32

	for _, result := range results {
		for _, b := range result {
			if b == 0 {
				zeroCount++
			}
		}
	}

	// Should have roughly 1/256 zero bytes (about 0.4%)
	expectedZeros := totalBytes / 256
	tolerance := expectedZeros / 2 // 50% tolerance

	if zeroCount < expectedZeros-tolerance || zeroCount > expectedZeros+tolerance {
		t.Logf("Zero byte count: %d, expected around %d (±%d)", zeroCount, expectedZeros, tolerance)
		// This is just a warning, not a failure
	}
}

func BenchmarkSimpleEntropy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := SimpleEntropy()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTimingEntropy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := TimingEntropy()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEnhancedEntropy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := EnhancedEntropy()
		if err != nil {
			b.Fatal(err)
		}
	}
}
