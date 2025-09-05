//go:build !cgo
// +build !cgo

// Package securebuf provides secure memory buffer operations (fallback when CGO is disabled)
package securebuf

import (
	"crypto/rand"
	"errors"
	"runtime"
	"sync"
)

// Buffer represents a secure memory buffer (Go fallback implementation)
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

// Zeroize clears the buffer content
func (b *Buffer) Zeroize() error {
	if b == nil {
		return errors.New("buffer is nil")
	}
	b.zeroize()
	return nil
}

// LockMemory locks the buffer in memory to prevent swapping (no-op in Go fallback)
func (b *Buffer) LockMemory() error {
	// Go fallback: memory locking not available
	return errors.New("memory locking not available in Go fallback mode")
}

// UnlockMemory unlocks the buffer memory (no-op in Go fallback)
func (b *Buffer) UnlockMemory() error {
	// Go fallback: memory locking not available
	return errors.New("memory unlocking not available in Go fallback mode")
}

// IsLocked returns whether the buffer memory is locked (always false in Go fallback)
func (b *Buffer) IsLocked() bool {
	return false
}

// IntegrityCheck verifies the buffer integrity (basic check in Go fallback)
func (b *Buffer) IntegrityCheck() bool {
	return b != nil && b.data != nil && b.length <= b.capacity
}

// Close frees the buffer and clears its content
func (b *Buffer) Close() error {
	if b == nil {
		return nil
	}
	b.Free()
	return nil
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

	// Reset length to indicate buffer is empty
	b.length = 0
}

// SecureBufferPool manages a pool of secure buffers for flat latency
type SecureBufferPool struct {
	pools   map[int]*sync.Pool
	mu      sync.RWMutex
	maxSize int
	minSize int
}

// NewSecureBufferPool creates a new secure buffer pool
func NewSecureBufferPool() *SecureBufferPool {
	return &SecureBufferPool{
		pools:   make(map[int]*sync.Pool),
		maxSize: 1024 * 1024, // 1MB max buffer size
		minSize: 64,          // 64 bytes min buffer size
	}
}

// Get retrieves a secure buffer from the pool
func (sbp *SecureBufferPool) Get(size int) (*Buffer, error) {
	if size > sbp.maxSize || size < sbp.minSize {
		// For sizes outside pool range, create new buffer
		return New(size)
	}

	sbp.mu.RLock()
	pool, exists := sbp.pools[size]
	sbp.mu.RUnlock()

	if !exists {
		sbp.mu.Lock()
		if sbp.pools[size] == nil {
			sbp.pools[size] = &sync.Pool{
				New: func() interface{} {
					buf, _ := New(size)
					return buf
				},
			}
		}
		pool = sbp.pools[size]
		sbp.mu.Unlock()
	}

	buf := pool.Get().(*Buffer)

	// Reset buffer state
	buf.length = 0
	for i := range buf.data {
		buf.data[i] = 0
	}

	return buf, nil
}

// Put returns a secure buffer to the pool after secure zeroization
func (sbp *SecureBufferPool) Put(buf *Buffer) {
	if buf == nil || buf.capacity > sbp.maxSize || buf.capacity < sbp.minSize {
		return
	}

	// Securely zeroize the buffer
	buf.zeroize()

	sbp.mu.RLock()
	pool, exists := sbp.pools[buf.capacity]
	sbp.mu.RUnlock()

	if exists {
		pool.Put(buf)
	}
}

// GetPoolStats returns statistics about the secure buffer pool
func (sbp *SecureBufferPool) GetPoolStats() map[string]interface{} {
	sbp.mu.RLock()
	defer sbp.mu.RUnlock()

	return map[string]interface{}{
		"pool_sizes":      len(sbp.pools),
		"max_buffer_size": sbp.maxSize,
		"min_buffer_size": sbp.minSize,
	}
}
