//go:build cgo
// +build cgo

// Package cgobindings provides Go bindings for Rust entropy functions
package cgobindings

/*
#cgo CFLAGS: -I../../../secure/rust/include
#cgo LDFLAGS: -L../../../secure/rust/target/release -lsecurebuffer -lws2_32 -luserenv -lntdll -lbcrypt -lmsvcrt -lkernel32 -lstdc++ -lchkstk_ms

#include <securebuffer.h>
#include <stdlib.h>
*/
import "C"

import (
	"unsafe"
)

// FastEntropy calls Rust fast_entropy() via FFI
func FastEntropy() []byte {
	buffer := make([]byte, 32)
	C.securebuffer_fill_fast_entropy(
		unsafe.Pointer(&buffer[0]),
	)
	return buffer
}

// HybridEntropy calls Rust hybrid_entropy() with Bitcoin headers via FFI
func HybridEntropy(headers [][]byte) []byte {
	buffer := make([]byte, 32)

	// Prepare headers for FFI - concatenate all headers
	var headersPtr *C.uint8_t
	var totalLen C.size_t
	headerCount := len(headers)

	if headerCount > 0 {
		// Calculate total size
		totalSize := 0
		for _, header := range headers {
			totalSize += len(header)
		}

		if totalSize > 0 {
			headersPtr = (*C.uint8_t)(C.malloc(C.size_t(totalSize)))
			defer C.free(unsafe.Pointer(headersPtr))

			// Copy all headers into contiguous memory
			offset := 0
			headersSlice := (*[1 << 30]byte)(unsafe.Pointer(headersPtr))[:totalSize:totalSize]
			for _, header := range headers {
				copy(headersSlice[offset:], header)
				offset += len(header)
			}
			totalLen = C.size_t(totalSize)
		}
	}

	C.securebuffer_fill_hybrid_entropy(
		unsafe.Pointer(&buffer[0]),
		headersPtr,
		totalLen,
		C.size_t(headerCount),
	)

	return buffer
}
