package securebuf

/*
#cgo LDFLAGS: -L. -lsecurebuffer
#include <stdlib.h>
#include <stdint.h>

// Rust FFI exports
extern void* secure_buffer_new(size_t capacity);
extern int secure_buffer_write(void* buf, const uint8_t* data, size_t len);
extern int secure_buffer_read(const void* buf, uint8_t* out, size_t out_len);
extern void secure_buffer_destroy(void* buf);
*/
import "C"
import (
	"errors"
	"unsafe"
)

// Buffer wraps the Rust SecureBuffer via FFI
type Buffer struct {
	ptr unsafe.Pointer
}

// New creates a new secure buffer with locked memory
func New(capacity int) (*Buffer, error) {
	if capacity <= 0 {
		return nil, errors.New("invalid capacity")
	}
	ptr := C.secure_buffer_new(C.size_t(capacity))
	if ptr == nil {
		return nil, errors.New("secure_buffer_new failed")
	}
	return &Buffer{ptr: ptr}, nil
}

// Write securely copies data into the buffer (zeroizes old contents first)
func (b *Buffer) Write(data []byte) error {
	if b.ptr == nil {
		return errors.New("buffer is nil")
	}
	if len(data) == 0 {
		return nil
	}
	res := C.secure_buffer_write(
		b.ptr,
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
	)
	if res != 0 {
		return errors.New("secure_buffer_write failed")
	}
	return nil
}

// Read copies data out of the buffer into dst
func (b *Buffer) Read(dst []byte) (int, error) {
	if b.ptr == nil {
		return 0, errors.New("buffer is nil")
	}
	if len(dst) == 0 {
		return 0, nil
	}
	res := C.secure_buffer_read(
		b.ptr,
		(*C.uint8_t)(unsafe.Pointer(&dst[0])),
		C.size_t(len(dst)),
	)
	if res < 0 {
		return 0, errors.New("secure_buffer_read failed")
	}
	return int(res), nil
}

// Free securely destroys the buffer, zeroizing and unlocking memory
func (b *Buffer) Free() {
	if b.ptr != nil {
		C.secure_buffer_destroy(b.ptr)
		b.ptr = nil
	}
}
