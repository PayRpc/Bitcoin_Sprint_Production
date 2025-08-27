// SPDX-License-Identifier: MIT
// Bitcoin Sprint - SecureBuffer Entropy Integration

use crate::{SecureBuffer, CSecureBuffer};
use crate::entropy;

impl SecureBuffer {
    /// Fill SecureBuffer with fast entropy (OS RNG + timing jitter)
    pub fn fill_with_fast_entropy(&mut self) -> Result<(), String> {
        if !self.is_valid() {
            return Err("Buffer is not valid".to_string());
        }
        
        let entropy_data = entropy::fast_entropy();
        self.write(&entropy_data)
    }

    /// Fill SecureBuffer with hybrid entropy (OS RNG + Bitcoin headers + jitter)
    pub fn fill_with_hybrid_entropy(&mut self, headers: &[Vec<u8>]) -> Result<(), String> {
        if !self.is_valid() {
            return Err("Buffer is not valid".to_string());
        }
        
        let entropy_data = entropy::hybrid_entropy(headers);
        self.write(&entropy_data)
    }

    /// Fill SecureBuffer with enterprise-grade entropy
    pub fn fill_with_enterprise_entropy(&mut self, headers: &[Vec<u8>], additional_data: &[u8]) -> Result<(), String> {
        if !self.is_valid() {
            return Err("Buffer is not valid".to_string());
        }
        
        let entropy_data = entropy::enterprise_entropy(headers, additional_data);
        self.write(&entropy_data)
    }

    /// Create a new SecureBuffer pre-filled with fast entropy
    pub fn new_with_fast_entropy(capacity: usize) -> Result<Self, String> {
        let mut buffer = Self::new(capacity)?;
        
        // Fill with entropy up to buffer capacity
        let chunks = (capacity + 31) / 32; // Round up to cover full capacity
        let mut offset = 0;
        
        for _ in 0..chunks {
            let entropy_chunk = entropy::fast_entropy();
            let remaining = capacity - offset;
            let write_len = std::cmp::min(32, remaining);
            
            if offset == 0 {
                // First write - use the write method which clears first
                buffer.write(&entropy_chunk[..write_len])?;
            } else {
                // Subsequent writes - append to existing entropy
                unsafe {
                    if offset + write_len <= buffer.capacity {
                        std::ptr::copy_nonoverlapping(
                            entropy_chunk.as_ptr(),
                            buffer.data.add(offset),
                            write_len
                        );
                        buffer.length = offset + write_len;
                    }
                }
            }
            
            offset += write_len;
            if offset >= capacity {
                break;
            }
        }
        
        Ok(buffer)
    }

    /// Create a new SecureBuffer pre-filled with hybrid entropy
    pub fn new_with_hybrid_entropy(capacity: usize, headers: &[Vec<u8>]) -> Result<Self, String> {
        let mut buffer = Self::new(capacity)?;
        
        // Fill with hybrid entropy up to buffer capacity
        let chunks = (capacity + 31) / 32;
        let mut offset = 0;
        
        for _ in 0..chunks {
            let entropy_chunk = entropy::hybrid_entropy(headers);
            let remaining = capacity - offset;
            let write_len = std::cmp::min(32, remaining);
            
            if offset == 0 {
                buffer.write(&entropy_chunk[..write_len])?;
            } else {
                unsafe {
                    if offset + write_len <= buffer.capacity {
                        std::ptr::copy_nonoverlapping(
                            entropy_chunk.as_ptr(),
                            buffer.data.add(offset),
                            write_len
                        );
                        buffer.length = offset + write_len;
                    }
                }
            }
            
            offset += write_len;
            if offset >= capacity {
                break;
            }
        }
        
        Ok(buffer)
    }

    /// Refresh buffer contents with new entropy (preserves capacity)
    pub fn refresh_entropy(&mut self) -> Result<(), String> {
        if !self.is_valid() {
            return Err("Buffer is not valid".to_string());
        }
        
        // Generate new entropy and overwrite existing content
        let entropy_data = entropy::fast_entropy();
        let write_len = std::cmp::min(entropy_data.len(), self.capacity);
        
        unsafe {
            // Clear existing content
            std::ptr::write_bytes(self.data, 0, self.capacity);
            // Write new entropy
            std::ptr::copy_nonoverlapping(entropy_data.as_ptr(), self.data, write_len);
        }
        
        self.length = write_len;
        Ok(())
    }

    /// Mix additional entropy into existing buffer content
    pub fn mix_entropy(&mut self, headers: &[Vec<u8>]) -> Result<(), String> {
        if !self.is_valid() {
            return Err("Buffer is not valid".to_string());
        }
        
        if self.length == 0 {
            return self.fill_with_hybrid_entropy(headers);
        }
        
        // Generate new entropy
        let new_entropy = entropy::hybrid_entropy(headers);
        let mix_len = std::cmp::min(new_entropy.len(), self.length);
        
        // XOR with existing content
        unsafe {
            for i in 0..mix_len {
                let existing = *self.data.add(i);
                *self.data.add(i) = existing ^ new_entropy[i];
            }
        }
        
        Ok(())
    }
}

// FFI exports for Go integration
#[no_mangle]
pub unsafe extern "C" fn securebuffer_fill_fast_entropy(buffer: *mut CSecureBuffer) -> i32 {
    if buffer.is_null() {
        return -1;
    }
    
    let c_buffer = &mut *buffer;
    if c_buffer.inner.is_null() {
        return -1;
    }
    
    match (*c_buffer.inner).fill_with_fast_entropy() {
        Ok(()) => 0,
        Err(_) => -1,
    }
}

#[no_mangle]
pub unsafe extern "C" fn securebuffer_fill_hybrid_entropy(
    buffer: *mut CSecureBuffer,
    headers_ptr: *const u8,
    headers_len: usize,
    header_count: usize,
) -> i32 {
    if buffer.is_null() || headers_ptr.is_null() {
        return -1;
    }
    
    let c_buffer = &mut *buffer;
    if c_buffer.inner.is_null() {
        return -1;
    }
    
    // Parse headers from flattened byte array
    // Each header is assumed to be 80 bytes (Bitcoin block header size)
    let header_size = if headers_len > 0 && header_count > 0 {
        headers_len / header_count
    } else {
        80 // Default to Bitcoin header size
    };
    
    let mut headers = Vec::new();
    for i in 0..header_count {
        let start = i * header_size;
        let end = std::cmp::min(start + header_size, headers_len);
        if start < headers_len {
            let header_slice = std::slice::from_raw_parts(headers_ptr.add(start), end - start);
            headers.push(header_slice.to_vec());
        }
    }
    
    match (*c_buffer.inner).fill_with_hybrid_entropy(&headers) {
        Ok(()) => 0,
        Err(_) => -1,
    }
}

#[no_mangle]
pub unsafe extern "C" fn securebuffer_fill_enterprise_entropy(
    buffer: *mut CSecureBuffer,
    headers_ptr: *const u8,
    headers_len: usize,
    header_count: usize,
    additional_data_ptr: *const u8,
    additional_data_len: usize,
) -> i32 {
    if buffer.is_null() {
        return -1;
    }
    
    let c_buffer = &mut *buffer;
    if c_buffer.inner.is_null() {
        return -1;
    }
    
    // Parse headers
    let header_size = if headers_len > 0 && header_count > 0 {
        headers_len / header_count
    } else {
        80
    };
    
    let mut headers = Vec::new();
    if !headers_ptr.is_null() {
        for i in 0..header_count {
            let start = i * header_size;
            let end = std::cmp::min(start + header_size, headers_len);
            if start < headers_len {
                let header_slice = std::slice::from_raw_parts(headers_ptr.add(start), end - start);
                headers.push(header_slice.to_vec());
            }
        }
    }
    
    // Parse additional data
    let additional_data = if !additional_data_ptr.is_null() && additional_data_len > 0 {
        std::slice::from_raw_parts(additional_data_ptr, additional_data_len)
    } else {
        &[]
    };
    
    match (*c_buffer.inner).fill_with_enterprise_entropy(&headers, additional_data) {
        Ok(()) => 0,
        Err(_) => -1,
    }
}

#[no_mangle]
pub unsafe extern "C" fn securebuffer_new_with_fast_entropy(capacity: usize) -> *mut CSecureBuffer {
    match SecureBuffer::new_with_fast_entropy(capacity) {
        Ok(buffer) => {
            let boxed = Box::new(CSecureBuffer {
                inner: Box::into_raw(Box::new(buffer)),
            });
            Box::into_raw(boxed)
        }
        Err(_) => std::ptr::null_mut(),
    }
}

#[no_mangle]
pub unsafe extern "C" fn securebuffer_new_with_hybrid_entropy(
    capacity: usize,
    headers_ptr: *const u8,
    headers_len: usize,
    header_count: usize,
) -> *mut CSecureBuffer {
    let header_size = if headers_len > 0 && header_count > 0 {
        headers_len / header_count
    } else {
        80
    };
    
    let mut headers = Vec::new();
    if !headers_ptr.is_null() {
        for i in 0..header_count {
            let start = i * header_size;
            let end = std::cmp::min(start + header_size, headers_len);
            if start < headers_len {
                let header_slice = std::slice::from_raw_parts(headers_ptr.add(start), end - start);
                headers.push(header_slice.to_vec());
            }
        }
    }
    
    match SecureBuffer::new_with_hybrid_entropy(capacity, &headers) {
        Ok(buffer) => {
            let boxed = Box::new(CSecureBuffer {
                inner: Box::into_raw(Box::new(buffer)),
            });
            Box::into_raw(boxed)
        }
        Err(_) => std::ptr::null_mut(),
    }
}

#[no_mangle]
pub unsafe extern "C" fn securebuffer_refresh_entropy(buffer: *mut CSecureBuffer) -> i32 {
    if buffer.is_null() {
        return -1;
    }
    
    let c_buffer = &mut *buffer;
    if c_buffer.inner.is_null() {
        return -1;
    }
    
    match (*c_buffer.inner).refresh_entropy() {
        Ok(()) => 0,
        Err(_) => -1,
    }
}

#[no_mangle]
pub unsafe extern "C" fn securebuffer_mix_entropy(
    buffer: *mut CSecureBuffer,
    headers_ptr: *const u8,
    headers_len: usize,
    header_count: usize,
) -> i32 {
    if buffer.is_null() {
        return -1;
    }
    
    let c_buffer = &mut *buffer;
    if c_buffer.inner.is_null() {
        return -1;
    }
    
    let header_size = if headers_len > 0 && header_count > 0 {
        headers_len / header_count
    } else {
        80
    };
    
    let mut headers = Vec::new();
    if !headers_ptr.is_null() {
        for i in 0..header_count {
            let start = i * header_size;
            let end = std::cmp::min(start + header_size, headers_len);
            if start < headers_len {
                let header_slice = std::slice::from_raw_parts(headers_ptr.add(start), end - start);
                headers.push(header_slice.to_vec());
            }
        }
    }
    
    match (*c_buffer.inner).mix_entropy(&headers) {
        Ok(()) => 0,
        Err(_) => -1,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_fill_with_fast_entropy() {
        let mut buffer = SecureBuffer::new(32).unwrap();
        assert!(buffer.fill_with_fast_entropy().is_ok());
        assert_eq!(buffer.len(), 32);
        
        // Should not be all zeros
        let mut data = vec![0u8; 32];
        assert!(buffer.read(&mut data).is_ok());
        assert_ne!(data, vec![0u8; 32]);
    }

    #[test]
    fn test_fill_with_hybrid_entropy() {
        let mut buffer = SecureBuffer::new(32).unwrap();
        let mock_headers = vec![vec![0u8; 80], vec![1u8; 80]];
        
        assert!(buffer.fill_with_hybrid_entropy(&mock_headers).is_ok());
        assert_eq!(buffer.len(), 32);
    }

    #[test]
    fn test_new_with_fast_entropy() {
        let buffer = SecureBuffer::new_with_fast_entropy(64).unwrap();
        assert_eq!(buffer.len(), 64);
        assert_eq!(buffer.capacity(), 64);
        
        // Should contain entropy data
        let mut data = vec![0u8; 64];
        assert!(buffer.read(&mut data).is_ok());
        
        // Very unlikely to be all zeros
        assert_ne!(data, vec![0u8; 64]);
    }

    #[test]
    fn test_refresh_entropy() {
        let mut buffer = SecureBuffer::new(32).unwrap();
        buffer.write(b"initial data").unwrap();
        
        let mut initial_data = vec![0u8; 32];
        buffer.read(&mut initial_data).unwrap();
        
        assert!(buffer.refresh_entropy().is_ok());
        
        let mut new_data = vec![0u8; 32];
        buffer.read(&mut new_data).unwrap();
        
        // Should be different after refresh
        assert_ne!(initial_data, new_data);
    }

    #[test]
    fn test_mix_entropy() {
        let mut buffer = SecureBuffer::new(32).unwrap();
        buffer.write(&[0xAA; 32]).unwrap();
        
        let mut initial_data = vec![0u8; 32];
        buffer.read(&mut initial_data).unwrap();
        
        let mock_headers = vec![vec![0x55; 80]];
        assert!(buffer.mix_entropy(&mock_headers).is_ok());
        
        let mut mixed_data = vec![0u8; 32];
        buffer.read(&mut mixed_data).unwrap();
        
        // Should be different after mixing
        assert_ne!(initial_data, mixed_data);
    }
}
