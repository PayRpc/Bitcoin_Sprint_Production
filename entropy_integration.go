package main

import (
	"fmt"
	"log"
	"os"

	"github.com/PayRpc/Bitcoin-Sprint/internal/entropy"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--test-entropy" {
		testEntropy()
		return
	}
	
	fmt.Println("Bitcoin Sprint - Entropy Integration Test")
	fmt.Println("Use --test-entropy flag to run entropy tests")
}

func testEntropy() {
	fmt.Println("=== Bitcoin Sprint Entropy Integration Test ===")
	
	// Test FastEntropy
	fmt.Print("Testing FastEntropy... ")
	fast, err := entropy.FastEntropy()
	if err != nil {
		log.Fatalf("FastEntropy failed: %v", err)
	}
	fmt.Printf("✓ Generated %d bytes\n", len(fast))
	fmt.Printf("Sample (first 8 bytes): %x\n", fast[:8])
	
	// Test HybridEntropy
	fmt.Print("Testing HybridEntropy... ")
	hybrid, err := entropy.HybridEntropy()
	if err != nil {
		log.Fatalf("HybridEntropy failed: %v", err)
	}
	fmt.Printf("✓ Generated %d bytes\n", len(hybrid))
	fmt.Printf("Sample (first 8 bytes): %x\n", hybrid[:8])
	
	// Test variability
	fmt.Print("Testing entropy variability... ")
	fast2, err := entropy.FastEntropy()
	if err != nil {
		log.Fatalf("Second FastEntropy failed: %v", err)
	}
	
	// Check they're different
	same := true
	for i := 0; i < len(fast) && i < len(fast2); i++ {
		if fast[i] != fast2[i] {
			same = false
			break
		}
	}
	
	if same {
		log.Fatal("✗ Entropy values are identical (bad!)")
	} else {
		fmt.Println("✓ Entropy values are different (good!)")
	}
	
	// Performance test
	fmt.Print("Performance test (1000 generations)... ")
	for i := 0; i < 1000; i++ {
		_, err := entropy.FastEntropy()
		if err != nil {
			log.Fatalf("Performance test failed at iteration %d: %v", i, err)
		}
	}
	fmt.Println("✓ Completed successfully")
	
	fmt.Println("\n🎉 All entropy tests passed!")
	fmt.Println("The Go-based entropy system is working correctly.")
}
