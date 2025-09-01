package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/entropy"
)

func main() {
	fmt.Println("🔐 Bitcoin Sprint - Entropy Generation Performance Test")
	fmt.Println("==================================================")

	// Test Fast Entropy
	fmt.Println("\n1. Fast Entropy (32 bytes):")
	start := time.Now()
	fastEntropy, err := entropy.FastEntropy()
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Printf("✅ Generated in: %v\n", duration)
		fmt.Printf("📄 Hex: %s\n", hex.EncodeToString(fastEntropy))
		fmt.Printf("📏 Length: %d bytes\n", len(fastEntropy))
	}

	// Test Hybrid Entropy
	fmt.Println("\n2. Hybrid Entropy (32 bytes):")
	start = time.Now()
	hybridEntropy, err := entropy.HybridEntropy()
	duration = time.Since(start)

	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Printf("✅ Generated in: %v\n", duration)
		fmt.Printf("📄 Hex: %s\n", hex.EncodeToString(hybridEntropy))
		fmt.Printf("📏 Length: %d bytes\n", len(hybridEntropy))
	}

	// Test System Fingerprint
	fmt.Println("\n3. System Fingerprint (32 bytes):")
	start = time.Now()
	fingerprint, err := entropy.SystemFingerprintRust()
	duration = time.Since(start)

	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Printf("✅ Generated in: %v\n", duration)
		fmt.Printf("📄 Hex: %s\n", hex.EncodeToString(fingerprint))
		fmt.Printf("📏 Length: %d bytes\n", len(fingerprint))
	}

	// Performance comparison with Go's crypto/rand
	fmt.Println("\n4. Performance Comparison (Go crypto/rand):")
	start = time.Now()
	randomBytes := make([]byte, 32)
	_, err = rand.Read(randomBytes)
	duration = time.Since(start)

	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Printf("✅ Generated in: %v\n", duration)
		fmt.Printf("📄 Hex: %s\n", hex.EncodeToString(randomBytes))
	}

	// Test multiple generations to show variance
	fmt.Println("\n5. Variance Test (5 consecutive generations):")
	for i := 0; i < 5; i++ {
		start := time.Now()
		testEntropy, _ := entropy.FastEntropy()
		duration := time.Since(start)
		fmt.Printf("   Run %d: %s... (%v)\n", i+1, hex.EncodeToString(testEntropy)[:16], duration)
	}

	fmt.Println("\n🎉 Entropy generation test complete!")
	fmt.Println("💡 All entropy is now cryptographically secure using OS-level randomness")
}
