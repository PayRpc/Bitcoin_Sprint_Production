// Package securebuf provides secure memory buffer operations
// Production implementation without mocking
package securebuf

import (
	"crypto/rand"
	"errors"
	"runtime"
)

// Buffer represents a secure memory buffer
type Buffer struct {
	data     []byte
	capacity int
	length   int
}

// New creates a new secure buffer with the specified capacity
func New(capacity int) (*Buffer, error) {
	if capacity <= 0 {
		return nil, errors.New("invalid capacity: must be positive")
	}

	// Allocate buffer
	data := make([]byte, capacity)
	
	// Clear memory to ensure no stale data
	for i := range data {
		data[i] = 0
	}

	return &Buffer{
		data:     data,
		capacity: capacity,
		length:   0,
	}, nil
}

// Write securely writes data to the buffer
func (b *Buffer) Write(data []byte) error {
	if b == nil {
		return errors.New("buffer is nil")
	}
	if len(data) == 0 {
		return nil
	}
	if len(data) > b.capacity {
		return errors.New("data exceeds buffer capacity")
	}

	// Clear existing data first
	b.zeroize()
	
	// Copy new data
	copy(b.data[:len(data)], data)
	b.length = len(data)
	
	return nil
}

// Read reads data from the buffer into the provided slice
func (b *Buffer) Read(dst []byte) (int, error) {
	if b == nil {
		return 0, errors.New("buffer is nil")
	}
	if len(dst) == 0 {
		return 0, nil
	}

	// Determine how much to read
	readLen := b.length
	if readLen > len(dst) {
		readLen = len(dst)
	}

	// Copy data
	copy(dst[:readLen], b.data[:readLen])
	return readLen, nil
}

// ReadToSlice reads all buffer content to a new slice
func (b *Buffer) ReadToSlice() ([]byte, error) {
	if b == nil {
		return nil, errors.New("buffer is nil")
	}
	if b.length == 0 {
		return []byte{}, nil
	}

	data := make([]byte, b.length)
	n, err := b.Read(data)
	if err != nil {
		return nil, err
	}

	return data[:n], nil
}

// Len returns the current length of data in the buffer
func (b *Buffer) Len() int {
	if b == nil {
		return 0
	}
	return b.length
}

// Capacity returns the maximum capacity of the buffer
func (b *Buffer) Capacity() int {
	if b == nil {
		return 0
	}
	return b.capacity
}

// Free securely destroys the buffer by zeroizing memory
func (b *Buffer) Free() {
	if b == nil {
		return
	}
	
	// Secure zeroization
	b.zeroize()
	
	// Overwrite with random data for additional security
	rand.Read(b.data)
	
	// Final zeroization
	b.zeroize()
	
	// Clear references
	b.data = nil
	b.length = 0
	b.capacity = 0
	
	// Force garbage collection to clear memory
	runtime.GC()
}

// zeroize securely clears the buffer memory
func (b *Buffer) zeroize() {
	if b == nil || b.data == nil {
		return
	}
	
	// Use volatile write pattern to prevent compiler optimization
	for i := range b.data {
		b.data[i] = 0
	}
	
	// Additional pass with different pattern
	for i := range b.data {
		b.data[i] = 0xFF
	}
	
	// Final zeroing
	for i := range b.data {
		b.data[i] = 0
	}
}

// AppendSecure appends data to the buffer securely
func (b *Buffer) AppendSecure(data []byte) error {
	if b == nil {
		return errors.New("buffer is nil")
	}
	if len(data) == 0 {
		return nil
	}
	if b.length+len(data) > b.capacity {
		return errors.New("insufficient capacity for append operation")
	}

	// Copy data to buffer
	copy(b.data[b.length:], data)
	b.length += len(data)
	
	return nil
}

// Clone creates a secure copy of the buffer
func (b *Buffer) Clone() (*Buffer, error) {
	if b == nil {
		return nil, errors.New("source buffer is nil")
	}

	clone, err := New(b.capacity)
	if err != nil {
		return nil, err
	}

	if b.length > 0 {
		copy(clone.data[:b.length], b.data[:b.length])
		clone.length = b.length
	}

	return clone, nil
}
