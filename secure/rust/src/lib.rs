// SPDX-License-Identifier: MIT
// Copyright (c) 2025 BitcoinCab.inc

use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Mutex;
use zeroize::Zeroize;
use lazy_static::lazy_static;
use std::io;
use thiserror::Error;

mod platform {
    #[cfg(unix)]
    pub fn lock_memory(ptr: *mut u8, len: usize) -> Result<(), std::io::Error> {
        unsafe {
            let mut retries = 3;
            while retries > 0 {
                if libc::mlock(ptr as *mut libc::c_void, len) == 0 {
                    return Ok(());
                }
                let err = std::io::Error::last_os_error();
                if err.raw_os_error() != Some(libc::EINTR) {
                    return Err(err);
                }
                retries -= 1;
            }
            Err(std::io::Error::new(std::io::ErrorKind::Other, "mlock retry limit exceeded"))
        }
    }

    #[cfg(unix)]
    pub fn unlock_memory(ptr: *mut u8, len: usize) -> Result<(), std::io::Error> {
        unsafe {
            if libc::munlock(ptr as *mut libc::c_void, len) == 0 {
                Ok(())
            } else {
                Err(std::io::Error::last_os_error())
            }
        }
    }

    #[cfg(not(unix))]
    pub fn lock_memory(_ptr: *mut u8, _len: usize) -> Result<(), std::io::Error> {
        Ok(()) // No-op on non-Unix systems
    }

    #[cfg(not(unix))]
    pub fn unlock_memory(_ptr: *mut u8, _len: usize) -> Result<(), std::io::Error> {
        Ok(()) // No-op on non-Unix systems
    }
}

lazy_static! {
    static ref MEMORY_POOL: Mutex<Vec<(usize, Vec<u8>)>> = Mutex::new(Vec::with_capacity(32));
}

#[derive(Error, Debug)]
pub enum SecureBufferError {
    #[error("Invalid buffer size: {0}")]
    InvalidSize(String),
    #[error("Memory allocation failed")]
    AllocationFailed,
    #[error("Memory locking failed: {0}")]
    LockFailed(#[source] io::Error),
    #[error("Memory unlocking failed: {0}")]
    UnlockFailed(#[source] io::Error),
    #[error("Buffer is invalid or was cleaned")]
    InvalidState,
    #[error("Copy operation failed: source length {0} exceeds buffer length {1}")]
    CopyOverflow(usize, usize),
}

#[derive(Debug)]
pub struct SecureBuffer {
    buffer: Vec<u8>,
    lock_count: AtomicUsize,
    is_valid: bool,
}

impl SecureBuffer {
    /// Creates a new secure buffer with the specified size.
    ///
    /// The buffer is allocated, zeroed, and locked in memory to prevent swapping to disk.
    /// Size must be non-zero and less than 1 GiB to prevent excessive memory usage.
    ///
    /// # Errors
    /// Returns `SecureBufferError::InvalidSize` if size is 0 or exceeds 1 GiB.
    /// Returns `SecureBufferError::AllocationFailed` if memory allocation fails.
    /// Returns `SecureBufferError::LockFailed` if memory locking fails.
    pub fn new(size: usize) -> Result<Self, SecureBufferError> {
        if size == 0 || size > 1 << 30 {
            return Err(SecureBufferError::InvalidSize(format!(
                "size {} is out of bounds (0 < size <= 1 GiB)", size
            )));
        }

        // Try to reuse a buffer from the pool
        let mut buffer = {
            let mut pool = MEMORY_POOL.lock().map_err(|_| SecureBufferError::AllocationFailed)?;
            pool.iter().position(|(s, _)| *s == size)
                .map(|i| pool.swap_remove(i).1)
                .unwrap_or_else(|| vec![0u8; size])
        };

        if buffer.capacity() != size {
            buffer = vec![0u8; size];
            if buffer.capacity() != size {
                return Err(SecureBufferError::AllocationFailed);
            }
        }

        let mut secure_buf = Self {
            buffer,
            lock_count: AtomicUsize::new(0),
            is_valid: true,
        };

        // Lock the memory
        platform::lock_memory(secure_buf.buffer.as_mut_ptr(), size)
            .map_err(SecureBufferError::LockFailed)?;
        secure_buf.lock_count.store(1, Ordering::Release);

        Ok(secure_buf)
    }

    /// Creates a secure buffer initialized with the provided data.
    ///
    /// The buffer is allocated to match the size of the input slice, data is copied securely,
    /// and the memory is locked to prevent swapping.
    ///
    /// # Errors
    /// Returns errors as per `new` if allocation or locking fails.
    pub fn from_slice(data: &[u8]) -> Result<Self, SecureBufferError> {
        let mut buf = Self::new(data.len())?;
        buf.copy_from_slice(data)?;
        Ok(buf)
    }

    /// Copies data into the buffer securely.
    ///
    /// Overwrites the buffer with the provided data, ensuring no sensitive data is left
    /// in memory. The operation is bounds-checked.
    ///
    /// # Errors
    /// Returns `SecureBufferError::InvalidState` if the buffer is invalid.
    /// Returns `SecureBufferError::CopyOverflow` if the input data is larger than the buffer.
    pub fn copy_from_slice(&mut self, data: &[u8]) -> Result<(), SecureBufferError> {
        if !self.is_valid {
            return Err(SecureBufferError::InvalidState);
        }
        if data.len() > self.buffer.len() {
            return Err(SecureBufferError::CopyOverflow(data.len(), self.buffer.len()));
        }
        self.buffer[..data.len()].copy_from_slice(data);
        // Zeroize any remaining bytes
        if data.len() < self.buffer.len() {
            self.buffer[data.len()..].zeroize();
        }
        Ok(())
    }

    /// Resizes the buffer to a new size, preserving as much data as possible.
    ///
    /// If the new size is smaller, data is truncated. If larger, the new space is zeroed.
    /// The new buffer is re-locked in memory.
    ///
    /// # Errors
    /// Returns errors as per `new` for invalid sizes, allocation, or locking failures.
    pub fn resize(&mut self, new_size: usize) -> Result<(), SecureBufferError> {
        if !self.is_valid {
            return Err(SecureBufferError::InvalidState);
        }
        if new_size == 0 || new_size > 1 << 30 {
            return Err(SecureBufferError::InvalidSize(format!(
                "new size {} is out of bounds (0 < size <= 1 GiB)", new_size
            )));
        }

        // Unlock current memory
        if self.lock_count.load(Ordering::Acquire) > 0 {
            platform::unlock_memory(self.buffer.as_mut_ptr(), self.buffer.len())
                .map_err(SecureBufferError::UnlockFailed)?;
            self.lock_count.store(0, Ordering::Release);
        }

        // Resize buffer
        let old_len = self.buffer.len();
        self.buffer.resize(new_size, 0);
        if self.buffer.capacity() < new_size {
            self.buffer = vec![0u8; new_size];
            if self.buffer.capacity() != new_size {
                return Err(SecureBufferError::AllocationFailed);
            }
        }

        // Zeroize new space if expanded
        if new_size > old_len {
            self.buffer[old_len..].zeroize();
        }

        // Re-lock memory
        platform::lock_memory(self.buffer.as_mut_ptr(), new_size)
            .map_err(SecureBufferError::LockFailed)?;
        self.lock_count.store(1, Ordering::Release);

        Ok(())
    }

    /// Returns an immutable reference to the buffer contents.
    ///
    /// Returns `None` if the buffer is invalid or was cleaned.
    pub fn as_slice(&self) -> Option<&[u8]> {
        if !self.is_valid {
            None
        } else {
            Some(&self.buffer)
        }
    }

    /// Returns a mutable reference to the buffer contents.
    ///
    /// Returns `None` if the buffer is invalid or was cleaned.
    pub fn as_mut_slice(&mut self) -> Option<&mut [u8]> {
        if !self.is_valid {
            None
        } else {
            Some(&mut self.buffer)
        }
    }

    /// Compares buffer contents with another slice in constant time to prevent timing attacks.
    ///
    /// Returns `false` if the buffer is invalid or lengths differ.
    pub fn constant_time_eq(&self, other: &[u8]) -> bool {
        if !self.is_valid || self.buffer.len() != other.len() {
            return false;
        }

        // Constant-time length comparison
        let len_diff = (self.buffer.len() ^ other.len()) as u8;
        let mut result = len_diff;

        // Compare bytes
        for (a, b) in self.buffer.iter().zip(other.iter()) {
            result |= a ^ b;
        }
        result == 0
    }

    /// Securely clears the buffer contents by zeroing memory.
    pub fn clean(&mut self) {
        if self.is_valid {
            self.buffer.zeroize();
        }
    }

    /// Returns the size of the buffer.
    pub fn len(&self) -> usize {
        self.buffer.len()
    }

    /// Checks if the buffer is empty.
    pub fn is_empty(&self) -> bool {
        self.buffer.is_empty()
    }

    /// Checks if the buffer is still valid.
    pub fn is_valid(&self) -> bool {
        self.is_valid
    }
}

unsafe impl Send for SecureBuffer {}
unsafe impl Sync for SecureBuffer {}

impl Clone for SecureBuffer {
    /// Creates a new secure buffer with a copy of the current buffer's contents.
    ///
    /// The new buffer is independently locked and managed.
    fn clone(&self) -> Self {
        let mut new_buf = SecureBuffer::new(self.buffer.len())
            .expect("Failed to allocate new buffer during clone");
        if let Some(src) = self.as_slice() {
            new_buf.copy_from_slice(src).expect("Failed to copy during clone");
        }
        new_buf
    }
}

impl Drop for SecureBuffer {
    fn drop(&mut self) {
        self.is_valid = false;
        self.clean();

        // Unlock the memory
        if self.lock_count.load(Ordering::Acquire) > 0 {
            let _ = platform::unlock_memory(self.buffer.as_mut_ptr(), self.buffer.len())
                .map_err(|e| eprintln!("Failed to unlock memory: {}", e));
            self.lock_count.store(0, Ordering::Release);
        }

        // Return buffer to pool if size is reasonable and pool isn't full
        if self.buffer.len() <= 1 << 20 {
            if let Ok(mut pool) = MEMORY_POOL.lock() {
                if pool.len() < 32 {
                    let mut empty_buf = Vec::new();
                    std::mem::swap(&mut empty_buf, &mut self.buffer);
                    empty_buf.zeroize();
                    empty_buf.clear();
                    pool.push((empty_buf.capacity(), empty_buf));
                }
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_new_secure_buffer() {
        let buffer = SecureBuffer::new(1024).expect("Failed to create secure buffer");
        assert_eq!(buffer.len(), 1024);
        assert!(buffer.is_valid());
        assert!(buffer.as_slice().is_some());
    }

    #[test]
    fn test_invalid_size() {
        assert!(matches!(
            SecureBuffer::new(0),
            Err(SecureBufferError::InvalidSize(_))
        ));
        assert!(matches!(
            SecureBuffer::new(1 << 31),
            Err(SecureBufferError::InvalidSize(_))
        ));
    }

    #[test]
    fn test_from_slice() {
        let data = [1, 2, 3, 4];
        let buffer = SecureBuffer::from_slice(&data).expect("Failed to create from slice");
        assert!(buffer.constant_time_eq(&data));
        assert_eq!(buffer.len(), 4);
    }

    #[test]
    fn test_copy_from_slice() {
        let mut buffer = SecureBuffer::new(4).unwrap();
        let data = [1, 2, 3, 4];
        buffer.copy_from_slice(&data).unwrap();
        assert!(buffer.constant_time_eq(&data));

        let too_large = [1, 2, 3, 4, 5];
        assert!(matches!(
            buffer.copy_from_slice(&too_large),
            Err(SecureBufferError::CopyOverflow(5, 4))
        ));

        buffer.clean();
        assert!(matches!(
            buffer.copy_from_slice(&data),
            Err(SecureBufferError::InvalidState)
        ));
    }

    #[test]
    fn test_resize() {
        let mut buffer = SecureBuffer::new(4).unwrap();
        buffer.copy_from_slice(&[1, 2, 3, 4]).unwrap();

        // Resize smaller
        buffer.resize(2).unwrap();
        assert_eq!(buffer.len(), 2);
        assert!(buffer.constant_time_eq(&[1, 2]));

        // Resize larger
        buffer.resize(4).unwrap();
        assert_eq!(buffer.len(), 4);
        assert!(buffer.constant_time_eq(&[1, 2, 0, 0]));
    }

    #[test]
    fn test_constant_time_eq() {
        let mut buffer = SecureBuffer::new(4).unwrap();
        buffer.copy_from_slice(&[1, 2, 3, 4]).unwrap();

        assert!(buffer.constant_time_eq(&[1, 2, 3, 4]));
        assert!(!buffer.constant_time_eq(&[1, 2, 3, 5]));
        assert!(!buffer.constant_time_eq(&[1, 2, 3])); // Different length

        buffer.clean();
        assert!(!buffer.constant_time_eq(&[0, 0, 0, 0]));
    }

    #[test]
    fn test_clean() {
        let mut buffer = SecureBuffer::new(4).unwrap();
        buffer.copy_from_slice(&[1, 2, 3, 4]).unwrap();
        buffer.clean();
        assert!(buffer.constant_time_eq(&[0, 0, 0, 0]));
        assert!(!buffer.is_valid());
        assert!(buffer.as_slice().is_none());
    }

    #[test]
    fn test_clone() {
        let mut buffer = SecureBuffer::new(4).unwrap();
        buffer.copy_from_slice(&[1, 2, 3, 4]).unwrap();
        let clone = buffer.clone();
        assert!(clone.constant_time_eq(&[1, 2, 3, 4]));
        assert_ne!(buffer.as_slice().unwrap().as_ptr(), clone.as_slice().unwrap().as_ptr());
    }

    #[test]
    fn test_memory_pool() {
        let buffer = SecureBuffer::new(1024).unwrap();
        drop(buffer);
        let pool = MEMORY_POOL.lock().unwrap();
        assert_eq!(pool.len(), 1);
        assert_eq!(pool[0].0, 1024);

        let new_buffer = SecureBuffer::new(1024).unwrap();
        assert_eq!(new_buffer.len(), 1024);
        drop(new_buffer);
        let pool = MEMORY_POOL.lock().unwrap();
        assert_eq!(pool.len(), 1); // Reused, so pool size doesn't grow
    }
}
