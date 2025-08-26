package main

import (
	"fmt"

	"github.com/PayRpc/Bitcoin-Sprint/pkg/secure"
)

/*
#include <stdlib.h>
extern bool securebuffer_self_check();
*/
import "C"

func main() {
	// Test direct C call
	fmt.Printf("Direct C Self-Check: %v\n", bool(C.securebuffer_self_check()))

	// Test Go wrapper
	fmt.Printf("Go Wrapper Self-Check: %v\n", secure.SelfCheck())

	// Test manual allocation
	buf := secure.NewSecureBuffer(32)
	if buf == nil {
		fmt.Println("Failed to create SecureBuffer")
		return
	}
	defer buf.Free()

	testData := []byte("test data for verification")
	if !buf.Copy(testData) {
		fmt.Println("Failed to copy data")
		return
	}

	fmt.Printf("Buffer created and data written successfully\n")
	fmt.Printf("Buffer contents: %s\n", string(buf.Data()))
}
