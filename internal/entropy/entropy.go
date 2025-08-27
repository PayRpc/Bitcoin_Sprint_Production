// Package entropy provides Go-based entropy functions
package entropy

// FastEntropy returns fast entropy using Go-based implementation
func FastEntropy() ([]byte, error) {
	return SimpleEntropy()
}

// HybridEntropy returns enhanced entropy using Go-based implementation  
func HybridEntropy() ([]byte, error) {
	return EnhancedEntropy()
}
