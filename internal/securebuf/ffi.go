//go:build cgo
// +build cgo

package securebuf

// CGO implementation
// This file provides a bridge to the Rust-based secure buffer implementation

/*
#cgo LDFLAGS: -L../../secure/rust/target/x86_64-pc-windows-gnu/release -lsecurebuffer
#include <stdlib.h>
#include <stdint.h>
#include <string.h>

typedef void* SecureBufferHandle;

extern SecureBufferHandle secure_buffer_new(size_t capacity);
extern int secure_buffer_write(SecureBufferHandle buffer, const uint8_t* data, size_t len);
extern int secure_buffer_read(SecureBufferHandle buffer, uint8_t* dst, size_t len, size_t* bytes_read);
extern size_t secure_buffer_len(SecureBufferHandle buffer);
extern size_t secure_buffer_capacity(SecureBufferHandle buffer);
extern int secure_buffer_zeroize(SecureBufferHandle buffer);
extern int secure_buffer_lock(SecureBufferHandle buffer);
extern int secure_buffer_unlock(SecureBufferHandle buffer);
extern int secure_buffer_is_locked(SecureBufferHandle buffer);
extern int secure_buffer_integrity_check(SecureBufferHandle buffer);
extern void secure_buffer_free(SecureBufferHandle buffer);
*/
import "C"
import (
	"errors"
	"runtime"
	"unsafe"
)

// Buffer represents a secure memory buffer using the Rust implementation
type Buffer struct {
	handle C.SecureBufferHandle
	locked bool
}

// New creates a new secure buffer with the specified capacity
func New(capacity int) (*Buffer, error) {
	if capacity <= 0 {
		return nil, errors.New("invalid capacity: must be positive")
	}

	handle := C.secure_buffer_new(C.size_t(capacity))
	if handle == nil {
		return nil, errors.New("failed to allocate secure buffer")
	}

	buffer := &Buffer{
		handle: handle,
		locked: false,
	}

	// Use finalizer to ensure buffer is freed when garbage collected
	runtime.SetFinalizer(buffer, (*Buffer).finalizer)

	return buffer, nil
}

// Write securely writes data to the buffer
func (b *Buffer) Write(data []byte) error {
	if b == nil || b.handle == nil {
		return errors.New("buffer is nil or already freed")
	}
	if len(data) == 0 {
		return nil
	}

	dataPtr := (*C.uint8_t)(unsafe.Pointer(&data[0]))
	result := C.secure_buffer_write(b.handle, dataPtr, C.size_t(len(data)))
	if result != 0 {
		return errors.New("failed to write data to secure buffer")
	}

	return nil
}

// Read reads data from the buffer into the provided slice
func (b *Buffer) Read(dst []byte) (int, error) {
	if b == nil || b.handle == nil {
		return 0, errors.New("buffer is nil or already freed")
	}
	if len(dst) == 0 {
		return 0, nil
	}

	var bytesRead C.size_t
	dstPtr := (*C.uint8_t)(unsafe.Pointer(&dst[0]))
	result := C.secure_buffer_read(b.handle, dstPtr, C.size_t(len(dst)), &bytesRead)
	if result != 0 {
		return 0, errors.New("failed to read data from secure buffer")
	}

	return int(bytesRead), nil
}

// ReadToSlice reads all buffer content to a new slice
func (b *Buffer) ReadToSlice() ([]byte, error) {
	if b == nil || b.handle == nil {
		return nil, errors.New("buffer is nil or already freed")
	}

	length := b.Len()
	if length == 0 {
		return []byte{}, nil
	}

	data := make([]byte, length)
	n, err := b.Read(data)
	if err != nil {
		return nil, err
	}

	return data[:n], nil
}

// Len returns the current length of data in the buffer
func (b *Buffer) Len() int {
	if b == nil || b.handle == nil {
		return 0
	}
	return int(C.secure_buffer_len(b.handle))
}

// Capacity returns the maximum capacity of the buffer
func (b *Buffer) Capacity() int {
	if b == nil || b.handle == nil {
		return 0
	}
	return int(C.secure_buffer_capacity(b.handle))
}

// Zeroize clears the buffer content
func (b *Buffer) Zeroize() error {
	if b == nil || b.handle == nil {
		return errors.New("buffer is nil or already freed")
	}

	result := C.secure_buffer_zeroize(b.handle)
	if result != 0 {
		return errors.New("failed to zeroize secure buffer")
	}

	return nil
}

// LockMemory locks the buffer in memory to prevent swapping
func (b *Buffer) LockMemory() error {
	if b == nil || b.handle == nil {
		return errors.New("buffer is nil or already freed")
	}

	if b.locked {
		return nil // Already locked
	}

	result := C.secure_buffer_lock(b.handle)
	if result != 0 {
		return errors.New("failed to lock memory")
	}

	b.locked = true
	return nil
}

// UnlockMemory unlocks the buffer memory
func (b *Buffer) UnlockMemory() error {
	if b == nil || b.handle == nil {
		return errors.New("buffer is nil or already freed")
	}

	if !b.locked {
		return nil // Not locked
	}

	result := C.secure_buffer_unlock(b.handle)
	if result != 0 {
		return errors.New("failed to unlock memory")
	}

	b.locked = false
	return nil
}

// IsLocked returns whether the buffer memory is locked
func (b *Buffer) IsLocked() bool {
	if b == nil || b.handle == nil {
		return false
	}
	return C.secure_buffer_is_locked(b.handle) != 0
}

// IntegrityCheck verifies the buffer integrity
func (b *Buffer) IntegrityCheck() bool {
	if b == nil || b.handle == nil {
		return false
	}
	return C.secure_buffer_integrity_check(b.handle) != 0
}

// Free releases the buffer and securely wipes its content
func (b *Buffer) Free() {
	if b != nil && b.handle != nil {
		// Ensure memory is unlocked before freeing
		if b.locked {
			_ = b.UnlockMemory()
		}

		// Free the buffer
		C.secure_buffer_free(b.handle)
		b.handle = nil

		// Remove finalizer since we've manually freed
		runtime.SetFinalizer(b, nil)
	}
}

// finalizer is called by the garbage collector when the buffer is no longer referenced
func (b *Buffer) finalizer() {
	b.Free()
}
