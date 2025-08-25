//go:build cgo
// +build cgo

// Package secure provides memory-locked secure buffers via Rust FFI
package secure

/*
#cgo windows LDFLAGS: -L${SRCDIR}/../../secure/rust/target/release -lsecurebuffer
#cgo linux   LDFLAGS: -L${SRCDIR}/../../secure/rust/target/release -lsecurebuffer
#cgo darwin  LDFLAGS: -L${SRCDIR}/../../secure/rust/target/release -lsecurebuffer

#include "../../secure/rust/include/securebuffer.h"
*/
import "C"
import (
	"encoding/base64"
	"fmt"
	"net/http"
	"runtime"
	"unsafe"
)

// SecureBuffer provides memory-locked storage for sensitive data
type SecureBuffer struct {
	ptr *C.SecureBuffer
}

// NewSecureBuffer creates a new memory-locked buffer of the specified size
func NewSecureBuffer(size int) *SecureBuffer {
	if size <= 0 {
		return nil
	}

	ptr := C.securebuffer_new(C.size_t(size))
	if ptr == nil {
		return nil
	}

	sb := &SecureBuffer{ptr: ptr}
	runtime.SetFinalizer(sb, (*SecureBuffer).Free)
	return sb
}

// Free releases the secure buffer and zeros its memory
func (sb *SecureBuffer) Free() {
	if sb.ptr != nil {
		C.securebuffer_free(sb.ptr)
		sb.ptr = nil
		runtime.SetFinalizer(sb, nil)
	}
}

// Copy copies data into the secure buffer
func (sb *SecureBuffer) Copy(data []byte) bool {
	if sb.ptr == nil || len(data) == 0 {
		return false
	}

	return bool(C.securebuffer_copy(sb.ptr, (*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data))))
}

// Data returns a slice view of the secure buffer's contents
// WARNING: The returned slice is only valid while the SecureBuffer exists
func (sb *SecureBuffer) Data() []byte {
	if sb.ptr == nil {
		return nil
	}

	length := int(C.securebuffer_len(sb.ptr))
	if length == 0 {
		return nil
	}

	ptr := C.securebuffer_data(sb.ptr)
	if ptr == nil {
		return nil
	}

	return unsafe.Slice((*byte)(unsafe.Pointer(ptr)), length)
}

// String returns the secure buffer contents as a string
// WARNING: This creates a copy in Go string memory
func (sb *SecureBuffer) String() string {
	if sb == nil || sb.ptr == nil {
		return ""
	}
	data := sb.Data()
	if data == nil {
		return ""
	}
	return string(data)
}

// WithBytes calls fn with the secure buffer's bytes while avoiding creating a Go string.
// The callback must not retain the slice after it returns.
func (sb *SecureBuffer) WithBytes(fn func([]byte) error) error {
	if sb == nil || sb.ptr == nil {
		return fmt.Errorf("secure buffer is nil")
	}
	data := sb.Data()
	if data == nil {
		return fmt.Errorf("secure buffer empty")
	}
	return fn(data)
}

// HMACHex computes HMAC-SHA256 over data using the secret in the SecureBuffer and returns hex string
func (sb *SecureBuffer) HMACHex(data []byte) string {
	if sb.ptr == nil || len(data) == 0 {
		return ""
	}
	cstr := C.securebuffer_hmac_hex(sb.ptr, (*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)))
	if cstr == nil {
		return ""
	}
	defer C.securebuffer_free_cstr(cstr)
	return C.GoString(cstr)
}

// HMACBase64URL computes HMAC-SHA256 and returns base64url (no padding)
func (sb *SecureBuffer) HMACBase64URL(data []byte) string {
	if sb.ptr == nil || len(data) == 0 {
		return ""
	}
	cstr := C.securebuffer_hmac_base64url(sb.ptr, (*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)))
	if cstr == nil {
		return ""
	}
	defer C.securebuffer_free_cstr(cstr)
	return C.GoString(cstr)
}

// SetBasicAuthHeader sets the Authorization header for req using username and a SecureBuffer password.
// This minimizes the lifetime of any plaintext password in Go.
func SetBasicAuthHeader(req *http.Request, user string, pass *SecureBuffer) {
	if req == nil || pass == nil {
		return
	}
	// Build "user:pass" bytes
	data := make([]byte, 0, len(user)+1+int(C.securebuffer_len(pass.ptr)))
	data = append(data, []byte(user)...)
	data = append(data, ':')
	// Append password bytes directly from secure buffer
	p := pass.Data()
	if p != nil {
		data = append(data, p...)
	}
	// Encode header
	encoded := "Basic " + base64.StdEncoding.EncodeToString(data)
	req.Header.Set("Authorization", encoded)
	// Zero temporary slices
	for i := range data {
		data[i] = 0
	}
}

// SelfCheck performs a self-test of the SecureBuffer functionality
// Returns true if SecureBuffer is working correctly (memory locking and zeroization)
func SelfCheck() bool {
	return bool(C.securebuffer_self_check())
}
