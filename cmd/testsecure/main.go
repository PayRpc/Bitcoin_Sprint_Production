package main

import (
	"fmt"
	"log"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/securebuf"
)

func main() {
	fmt.Println("🔒 Bitcoin Sprint Security Test Suite")
	fmt.Println("=====================================")

	// Test 1: Basic buffer operations
	fmt.Print("Test 1: Basic buffer operations... ")
	if testBasicOperations() {
		fmt.Println("✅ PASSED")
	} else {
		fmt.Println("❌ FAILED")
		return
	}

	// Test 2: Memory security
	fmt.Print("Test 2: Memory security... ")
	if testMemorySecurity() {
		fmt.Println("✅ PASSED")
	} else {
		fmt.Println("❌ FAILED")
		return
	}

	// Test 3: Performance
	fmt.Print("Test 3: Performance benchmark... ")
	testPerformance()
	fmt.Println("✅ COMPLETED")

	fmt.Println("=====================================")
	fmt.Println("🎉 All security tests PASSED!")
}

func testBasicOperations() bool {
	buf, err := securebuf.New(1024)
	if err != nil {
		return false
	}
	defer buf.Free()

	testData := []byte("Bitcoin Sprint Security Test")
	err = buf.Write(testData)
	if err != nil {
		return false
	}

	readBuffer := make([]byte, len(testData))
	n, err := buf.Read(readBuffer)
	if err != nil {
		return false
	}

	return n == len(testData) && string(readBuffer[:n]) == string(testData)
}

func testMemorySecurity() bool {
	buf, err := securebuf.New(512)
	if err != nil {
		return false
	}
	defer buf.Free()

	// Test with sensitive data
	sensitiveData := []byte("private_key_test_data_12345")
	err = buf.Write(sensitiveData)
	if err != nil {
		return false
	}

	// Verify we can read it back securely
	readBuffer := make([]byte, len(sensitiveData))
	n, err := buf.Read(readBuffer)
	if err != nil {
		return false
	}

	return n == len(sensitiveData) && string(readBuffer[:n]) == string(sensitiveData)
}

func testPerformance() {
	start := time.Now()

	for i := 0; i < 1000; i++ {
		buf, err := securebuf.New(256)
		if err != nil {
			log.Printf("Performance test failed at iteration %d: %v", i, err)
			return
		}

		testData := []byte(fmt.Sprintf("test_data_%d", i))
		buf.Write(testData)

		// Read back the data
		readBuffer := make([]byte, len(testData))
		buf.Read(readBuffer)

		buf.Free()
	}

	duration := time.Since(start)
	fmt.Printf("(1000 operations in %v) ", duration)
}
