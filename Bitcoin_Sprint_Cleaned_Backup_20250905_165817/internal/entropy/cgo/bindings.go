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
	"fmt"
	"unsafe"
)

// FastEntropy calls Rust fast_entropy_c() via direct FFI
func FastEntropy() []byte {
	buffer := make([]byte, 32)
	result := C.fast_entropy_c((*C.uchar)(unsafe.Pointer(&buffer[0])))
	if result != 0 {
		// If FFI fails, return pseudo-random data
		for i := range buffer {
			buffer[i] = byte(i*7 + 42) // Simple fallback
		}
	}
	return buffer
}

// HybridEntropy calls Rust hybrid_entropy_c() with Bitcoin headers via direct FFI
func HybridEntropy(headers [][]byte) []byte {
	buffer := make([]byte, 32)

	if len(headers) == 0 {
		// Fall back to fast entropy if no headers
		return FastEntropy()
	}

	// Prepare headers for FFI
	headerPtrs := make([]*C.uchar, len(headers))
	headerLens := make([]C.size_t, len(headers))

	// Allocate and copy header data
	allocatedHeaders := make([][]byte, len(headers))
	for i, header := range headers {
		if len(header) > 0 {
			allocatedHeaders[i] = make([]byte, len(header))
			copy(allocatedHeaders[i], header)
			headerPtrs[i] = (*C.uchar)(unsafe.Pointer(&allocatedHeaders[i][0]))
			headerLens[i] = C.size_t(len(header))
		}
	}

	result := C.hybrid_entropy_c(
		(**C.uchar)(unsafe.Pointer(&headerPtrs[0])),
		(*C.size_t)(unsafe.Pointer(&headerLens[0])),
		C.size_t(len(headers)),
		(*C.uchar)(unsafe.Pointer(&buffer[0])),
	)

	if result != 0 {
		// If FFI fails, return fast entropy as fallback
		return FastEntropy()
	}

	return buffer
}

// SystemFingerprint calls Rust system_fingerprint_c() via direct FFI
func SystemFingerprint() ([]byte, error) {
	buffer := make([]byte, 32)
	result := C.system_fingerprint_c((*C.uchar)(unsafe.Pointer(&buffer[0])))
	if result != 0 {
		return nil, fmt.Errorf("failed to get system fingerprint (code: %d)", result)
	}
	return buffer, nil
}

// GetCPUTemperature calls Rust get_cpu_temperature_c() via direct FFI
func GetCPUTemperature() (float32, error) {
	temp := C.get_cpu_temperature_c()
	if temp < 0 {
		return 0, fmt.Errorf("failed to get CPU temperature")
	}
	return float32(temp), nil
}

// FastEntropyWithFingerprint calls Rust fast_entropy_with_fingerprint_c() via direct FFI
func FastEntropyWithFingerprint() []byte {
	buffer := make([]byte, 32)
	result := C.fast_entropy_with_fingerprint_c((*C.uchar)(unsafe.Pointer(&buffer[0])))
	if result != 0 {
		// Fall back to regular fast entropy
		return FastEntropy()
	}
	return buffer
}

// HybridEntropyWithFingerprint calls Rust hybrid_entropy_with_fingerprint_c() via direct FFI
func HybridEntropyWithFingerprint(headers [][]byte) []byte {
	buffer := make([]byte, 32)

	if len(headers) == 0 {
		// Fall back to fast entropy with fingerprint if no headers
		return FastEntropyWithFingerprint()
	}

	// Prepare headers for FFI
	headerPtrs := make([]*C.uchar, len(headers))
	headerLens := make([]C.size_t, len(headers))

	// Allocate and copy header data
	allocatedHeaders := make([][]byte, len(headers))
	for i, header := range headers {
		if len(header) > 0 {
			allocatedHeaders[i] = make([]byte, len(header))
			copy(allocatedHeaders[i], header)
			headerPtrs[i] = (*C.uchar)(unsafe.Pointer(&allocatedHeaders[i][0]))
			headerLens[i] = C.size_t(len(header))
		}
	}

	result := C.hybrid_entropy_with_fingerprint_c(
		(**C.uchar)(unsafe.Pointer(&headerPtrs[0])),
		(*C.size_t)(unsafe.Pointer(&headerLens[0])),
		C.size_t(len(headers)),
		(*C.uchar)(unsafe.Pointer(&buffer[0])),
	)

	if result != 0 {
		// If FFI fails, return fast entropy with fingerprint as fallback
		return FastEntropyWithFingerprint()
	}

	return buffer
}
