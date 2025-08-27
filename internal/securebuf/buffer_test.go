package securebuf

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuffer_New(t *testing.T) {
	tests := []struct {
		name      string
		capacity  int
		expectErr bool
	}{
		{"valid capacity", 1024, false},
		{"zero capacity", 0, true},
		{"negative capacity", -1, true},
		{"large capacity", 1024 * 1024, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := New(tt.capacity)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, buf)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, buf)
				assert.Equal(t, tt.capacity, buf.Capacity())
				assert.Equal(t, 0, buf.Len())
				buf.Close()
			}
		})
	}
}

func TestBuffer_Write_Read(t *testing.T) {
	buf, err := New(1024)
	require.NoError(t, err)
	require.NotNil(t, buf)
	defer buf.Close()

	testData := []byte("Hello, Secure World!")
	err = buf.Write(testData)
	assert.NoError(t, err)
	assert.Equal(t, len(testData), buf.Len())

	// Read into slice
	readData := make([]byte, len(testData))
	n, err := buf.Read(readData)
	assert.NoError(t, err)
	assert.Equal(t, len(testData), n)
	assert.Equal(t, testData, readData)
}

func TestBuffer_ReadToSlice(t *testing.T) {
	buf, err := New(1024)
	require.NoError(t, err)
	require.NotNil(t, buf)
	defer buf.Close()

	testData := []byte("Test data for ReadToSlice")
	err = buf.Write(testData)
	assert.NoError(t, err)

	data, err := buf.ReadToSlice()
	assert.NoError(t, err)
	assert.Equal(t, testData, data)
}

func TestBuffer_AppendSecure(t *testing.T) {
	buf, err := New(1024)
	require.NoError(t, err)
	require.NotNil(t, buf)
	defer buf.Close()

	part1 := []byte("Hello")
	part2 := []byte(" World")

	err = buf.AppendSecure(part1)
	assert.NoError(t, err)
	assert.Equal(t, len(part1), buf.Len())

	err = buf.AppendSecure(part2)
	assert.NoError(t, err)
	assert.Equal(t, len(part1)+len(part2), buf.Len())

	data, err := buf.ReadToSlice()
	assert.NoError(t, err)
	assert.Equal(t, []byte("Hello World"), data)
}

func TestBuffer_AppendSecure_Overflow(t *testing.T) {
	buf, err := New(10)
	require.NoError(t, err)
	require.NotNil(t, buf)
	defer buf.Close()

	// Fill buffer
	err = buf.AppendSecure([]byte("1234567890"))
	assert.NoError(t, err)

	// Try to append more than capacity allows
	err = buf.AppendSecure([]byte("more data"))
	assert.Error(t, err)
}

func TestBuffer_Zeroize(t *testing.T) {
	buf, err := New(1024)
	require.NoError(t, err)
	require.NotNil(t, buf)
	defer buf.Close()

	testData := []byte("Sensitive data")
	err = buf.Write(testData)
	assert.NoError(t, err)

	err = buf.Zeroize()
	assert.NoError(t, err)
	assert.Equal(t, 0, buf.Len())

	// Verify data is zeroized
	data, err := buf.ReadToSlice()
	assert.NoError(t, err)
	assert.Equal(t, []byte{}, data)
}

func TestBuffer_Clone(t *testing.T) {
	buf, err := New(1024)
	require.NoError(t, err)
	require.NotNil(t, buf)
	defer buf.Close()

	testData := []byte("Data to clone")
	err = buf.Write(testData)
	assert.NoError(t, err)

	clone, err := buf.Clone()
	assert.NoError(t, err)
	assert.NotNil(t, clone)
	defer clone.Close()

	assert.Equal(t, buf.Len(), clone.Len())
	assert.Equal(t, buf.Capacity(), clone.Capacity())

	cloneData, err := clone.ReadToSlice()
	assert.NoError(t, err)
	assert.Equal(t, testData, cloneData)
}

func TestBuffer_IntegrityCheck(t *testing.T) {
	buf, err := New(1024)
	require.NoError(t, err)
	require.NotNil(t, buf)
	defer buf.Close()

	assert.True(t, buf.IntegrityCheck())

	err = buf.Write([]byte("test"))
	assert.NoError(t, err)
	assert.True(t, buf.IntegrityCheck())
}

func TestBuffer_LockMemory(t *testing.T) {
	buf, err := New(1024)
	require.NoError(t, err)
	require.NotNil(t, buf)
	defer buf.Close()

	// Note: In Go fallback mode, LockMemory will return an error
	err = buf.LockMemory()
	if err != nil {
		// This is expected in Go fallback mode
		assert.Contains(t, err.Error(), "not available")
		assert.False(t, buf.IsLocked())
	} else {
		// If CGO is enabled and memory locking works
		assert.True(t, buf.IsLocked())

		err = buf.UnlockMemory()
		assert.NoError(t, err)
		assert.False(t, buf.IsLocked())
	}
}

func TestBuffer_Write_Overflow(t *testing.T) {
	buf, err := New(10)
	require.NoError(t, err)
	require.NotNil(t, buf)
	defer buf.Close()

	// Try to write more than capacity
	err = buf.Write([]byte("this is more than 10 bytes"))
	assert.Error(t, err)
}

func TestBuffer_Read_Empty(t *testing.T) {
	buf, err := New(1024)
	require.NoError(t, err)
	require.NotNil(t, buf)
	defer buf.Close()

	data, err := buf.ReadToSlice()
	assert.NoError(t, err)
	assert.Equal(t, []byte{}, data)
}

func TestBuffer_Read_Partial(t *testing.T) {
	buf, err := New(1024)
	require.NoError(t, err)
	require.NotNil(t, buf)
	defer buf.Close()

	testData := []byte("Hello World")
	err = buf.Write(testData)
	assert.NoError(t, err)

	// Read only part of the data
	partial := make([]byte, 5)
	n, err := buf.Read(partial)
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, []byte("Hello"), partial)
}

func TestBuffer_Close(t *testing.T) {
	buf, err := New(1024)
	require.NoError(t, err)
	require.NotNil(t, buf)

	err = buf.Close()
	assert.NoError(t, err)

	// Buffer should be unusable after close
	err = buf.Write([]byte("test"))
	assert.Error(t, err)
}

func TestBuffer_Nil_Safety(t *testing.T) {
	var buf *Buffer

	err := buf.Write([]byte("test"))
	assert.Error(t, err)

	n, err := buf.Read([]byte{})
	assert.Error(t, err)
	assert.Equal(t, 0, n)

	data, err := buf.ReadToSlice()
	assert.Error(t, err)
	assert.Nil(t, data)

	assert.Equal(t, 0, buf.Len())
	assert.Equal(t, 0, buf.Capacity())

	err = buf.Zeroize()
	assert.Error(t, err)

	err = buf.Close()
	assert.NoError(t, err) // Close on nil should be safe
}

func TestBuffer_Large_Data(t *testing.T) {
	capacity := 1024 * 1024 // 1MB
	buf, err := New(capacity)
	require.NoError(t, err)
	require.NotNil(t, buf)
	defer buf.Close()

	// Create large test data
	largeData := bytes.Repeat([]byte("A"), capacity)
	err = buf.Write(largeData)
	assert.NoError(t, err)
	assert.Equal(t, capacity, buf.Len())

	readData, err := buf.ReadToSlice()
	assert.NoError(t, err)
	assert.Equal(t, largeData, readData)
}
