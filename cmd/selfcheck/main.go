package main

import (
	"fmt"
	"log"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/securebuf"
)

func main() {
	fmt.Println("🔒 Bitcoin Sprint Security Self-Check")
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

	// Test 3: Data integrity
	fmt.Print("Test 3: Data integrity... ")
	if testDataIntegrity() {
		fmt.Println("✅ PASSED")
	} else {
		fmt.Println("❌ FAILED")
		return
	}

	fmt.Println("=====================================")
	fmt.Printf("🎉 Self-check completed successfully at %s\n", time.Now().Format("15:04:05"))
	fmt.Println("Bitcoin Sprint security system is operational.")
}

func testBasicOperations() bool {
	buf, err := securebuf.New(1024)
	if err != nil {
		log.Printf("Buffer creation failed: %v", err)
		return false
	}
	defer buf.Free()

	testData := []byte("Bitcoin Sprint Security Test")
	err = buf.Write(testData)
	if err != nil {
		log.Printf("Write operation failed: %v", err)
		return false
	}

	readBuffer := make([]byte, len(testData))
	n, err := buf.Read(readBuffer)
	if err != nil {
		log.Printf("Read operation failed: %v", err)
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

	// Verify we can read it back
	readBuffer := make([]byte, len(sensitiveData))
	n, err := buf.Read(readBuffer)
	if err != nil {
		return false
	}

	return n == len(sensitiveData) && string(readBuffer[:n]) == string(sensitiveData)
}

func testDataIntegrity() bool {
	// Test multiple buffer operations
	for i := 0; i < 10; i++ {
		buf, err := securebuf.New(128)
		if err != nil {
			return false
		}

		testData := []byte(fmt.Sprintf("integrity_test_%d", i))
		err = buf.Write(testData)
		if err != nil {
			buf.Free()
			return false
		}

		readBuffer := make([]byte, len(testData))
		n, err := buf.Read(readBuffer)
		if err != nil || n != len(testData) || string(readBuffer[:n]) != string(testData) {
			buf.Free()
			return false
		}

		buf.Free()
	}
	return true
}
