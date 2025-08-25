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
