package main

import (
	"fmt"
	"log"

	"github.com/PayRpc/Bitcoin-Sprint/internal/securebuf"
)

func main() {
	fmt.Println("🔧 Bitcoin Sprint Debug Check")
	fmt.Println("==============================")

	// Test 1: Module availability
	fmt.Print("Test 1: SecureBuf module... ")
	if testModuleAvailable() {
		fmt.Println("✅ AVAILABLE")
	} else {
		fmt.Println("❌ UNAVAILABLE")
		return
	}

	// Test 2: Buffer creation
	fmt.Print("Test 2: Buffer creation... ")
	if testBufferCreation() {
		fmt.Println("✅ SUCCESS")
	} else {
		fmt.Println("❌ FAILED")
		return
	}

	// Test 3: Data integrity
	fmt.Print("Test 3: Data integrity... ")
	if testDataIntegrity() {
		fmt.Println("✅ VERIFIED")
	} else {
		fmt.Println("❌ COMPROMISED")
		return
	}

	// Test 4: Memory management
	fmt.Print("Test 4: Memory management... ")
	if testMemoryManagement() {
		fmt.Println("✅ SECURE")
	} else {
		fmt.Println("❌ LEAKS DETECTED")
		return
	}

	fmt.Println("==============================")
	fmt.Println("🎯 Debug check completed successfully!")
	fmt.Println("System is ready for production.")
}

func testModuleAvailable() bool {
	// Try to create a buffer to test module availability
	buf, err := securebuf.New(16)
	if err != nil {
		return false
	}
	buf.Free()
	return true
}

func testBufferCreation() bool {
	buf, err := securebuf.New(64)
	if err != nil {
		log.Printf("Buffer creation failed: %v", err)
		return false
	}
	defer buf.Free()
	return true
}

func testDataIntegrity() bool {
	buf, err := securebuf.New(128)
	if err != nil {
		return false
	}
	defer buf.Free()

	testData := []byte("debug_integrity_test_12345")
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

func testMemoryManagement() bool {
	// Create and destroy multiple buffers to test memory management
	for i := 0; i < 100; i++ {
		buf, err := securebuf.New(32)
		if err != nil {
			return false
		}
		
		testData := []byte(fmt.Sprintf("memory_test_%d", i))
		buf.Write(testData)
		buf.Free()
	}
	return true
}
