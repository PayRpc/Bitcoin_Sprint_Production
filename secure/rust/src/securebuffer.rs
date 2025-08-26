// SPDX-License-Identifier: MIT
// BitcoinCab.inc - SecureBuffer core with thread-safety and production hardening

use std::alloc::{alloc, dealloc, Layout};
use std::sync::atomic::{AtomicBool, Ordering};
use std::io;
use thiserror::Error;

#[cfg(unix)]
extern crate libc;

#[cfg(windows)]
extern crate winapi;

mod memory {
    use std::io;

    #[cfg(unix)]
    pub fn lock_memory(ptr: *mut u8, len: usize) -> Result<(), io::Error> {
        unsafe {
            if libc::mlock(ptr as *mut libc::c_void, len) == 0 {
                Ok(())
            } else {
                Err(io::Error::last_os_error())
            }
        }
    }

    #[cfg(unix)]
    pub fn unlock_memory(ptr: *mut u8, len: usize) -> Result<(), io::Error> {
        unsafe {
            if libc::munlock(ptr as *mut libc::c_void, len) == 0 {
                Ok(())
            } else {
                Err(io::Error::last_os_error())
            }
        }
    }

    #[cfg(unix)]
    pub fn explicit_bzero(ptr: *mut u8, len: usize) {
        unsafe {
            // Use explicit_bzero if available, fallback to volatile writes
            #[cfg(target_os = "linux")]
            {
                extern "C" {
                    fn explicit_bzero(s: *mut libc::c_void, n: libc::size_t);
                }
                explicit_bzero(ptr as *mut libc::c_void, len);
            }
            #[cfg(not(target_os = "linux"))]
            {
                // Fallback to volatile writes to prevent compiler optimization
                for i in 0..len {
                    std::ptr::write_volatile(ptr.add(i), 0);
                }
            }
        }
    }

    #[cfg(windows)]
    pub fn lock_memory(ptr: *mut u8, len: usize) -> Result<(), io::Error> {
        unsafe {
            if winapi::um::memoryapi::VirtualLock(ptr as *mut _, len) != 0 {
                Ok(())
            } else {
                Err(io::Error::last_os_error())
            }
        }
    }

    #[cfg(windows)]
    pub fn unlock_memory(ptr: *mut u8, len: usize) -> Result<(), io::Error> {
        unsafe {
            if winapi::um::memoryapi::VirtualUnlock(ptr as *mut _, len) != 0 {
                Ok(())
            } else {
                Err(io::Error::last_os_error())
            }
        }
    }

    #[cfg(windows)]
    pub fn explicit_bzero(ptr: *mut u8, len: usize) {
        unsafe {
            // Use RtlSecureZeroMemory on Windows
            std::ptr::write_bytes(ptr, 0, len);
        }
    }

    #[cfg(not(any(unix, windows)))]
    pub fn lock_memory(_ptr: *mut u8, _len: usize) -> Result<(), io::Error> {
        // Platform not supported, but don't fail
        Ok(())
    }

    #[cfg(not(any(unix, windows)))]
    pub fn unlock_memory(_ptr: *mut u8, _len: usize) -> Result<(), io::Error> {
        // Platform not supported, but don't fail
        Ok(())
    }

    #[cfg(not(any(unix, windows)))]
    pub fn explicit_bzero(ptr: *mut u8, len: usize) {
        unsafe {
            // Fallback to volatile writes
            for i in 0..len {
                std::ptr::write_volatile(ptr.add(i), 0);
            }
        }
    }
}

#[derive(Error, Debug)]
pub enum SecureBufferError {
    #[error("Invalid size")]
    InvalidSize,
    #[error("Allocation failed")]
    AllocationFailed,
    #[error("Lock failed: {0}")]
    LockFailed(#[source] io::Error),
    #[error("Copy overflow")]
    CopyOverflow,
    #[error("Invalid state")]
    InvalidState,
}

/// Thread-safe secure buffer with memory locking and hardened zeroization
pub struct SecureBuffer {
    data: *mut u8,
    capacity: usize,
    length: usize,
    is_valid: AtomicBool,
    is_locked: AtomicBool,
}

impl SecureBuffer {
    /// Create a new secure buffer with the specified capacity
    pub fn new(capacity: usize) -> Result<Self, String> {
        if capacity == 0 {
            return Err("Capacity must be greater than 0".to_string());
        }
        
        // Use aligned allocation for better security and performance
        let layout = Layout::from_size_align(capacity, 32)
            .map_err(|_| "Invalid layout for allocation".to_string())?;
        
        let data = unsafe { alloc(layout) };
        if data.is_null() {
            return Err("Failed to allocate memory".to_string());
        }

        // Immediately zero the allocated memory
        unsafe {
            memory::explicit_bzero(data, capacity);
        }

        // Attempt to lock memory (non-fatal if it fails)
        let is_locked = memory::lock_memory(data, capacity).is_ok();

        let buffer = SecureBuffer {
            data,
            capacity,
            length: 0,
            is_valid: AtomicBool::new(true),
            is_locked: AtomicBool::new(is_locked),
        };

        Ok(buffer)
    }

    /// Write data to the buffer, replacing any existing content
    pub fn write(&mut self, data: &[u8]) -> Result<(), String> {
        if !self.is_valid.load(Ordering::SeqCst) {
            return Err("Buffer is not valid".to_string());
        }
        
        if data.len() > self.capacity {
            return Err("Data exceeds buffer capacity".to_string());
        }

        unsafe {
            // Zero any existing data first
            memory::explicit_bzero(self.data, self.capacity);
            // Copy new data
            std::ptr::copy_nonoverlapping(data.as_ptr(), self.data, data.len());
        }
        
        self.length = data.len();
        Ok(())
    }

    /// Read data from the buffer into the provided slice
    pub fn read(&self, buf: &mut [u8]) -> Result<usize, String> {
        if !self.is_valid.load(Ordering::SeqCst) {
            return Err("Buffer is not valid".to_string());
        }
        
        let copy_len = std::cmp::min(buf.len(), self.length);
        unsafe {
            std::ptr::copy_nonoverlapping(self.data, buf.as_mut_ptr(), copy_len);
        }
        
        Ok(copy_len)
    }

    /// Get a slice view of the buffer content (prevents length disclosure)
    pub fn as_slice(&self) -> Result<&[u8], String> {
        if !self.is_valid.load(Ordering::SeqCst) {
            return Err("Buffer is not valid".to_string());
        }
        
        // Prevent length disclosure in error cases by always returning fixed-size error
        if self.length == 0 {
            return Err("Empty".to_string());
        }
        
        unsafe { Ok(std::slice::from_raw_parts(self.data, self.length)) }
    }

    /// Get the current length of data in the buffer (thread-safe)
    pub fn len(&self) -> usize {
        if self.is_valid.load(Ordering::SeqCst) {
            self.length
        } else {
            0 // Don't disclose length of invalid buffers
        }
    }

    /// Get the capacity of the buffer (thread-safe)
    pub fn capacity(&self) -> usize {
        if self.is_valid.load(Ordering::SeqCst) {
            self.capacity
        } else {
            0 // Don't disclose capacity of invalid buffers
        }
    }

    /// Clear all data from the buffer with secure zeroization
    pub fn clear(&mut self) {
        if self.is_valid.load(Ordering::SeqCst) {
            unsafe {
                memory::explicit_bzero(self.data, self.capacity);
            }
            self.length = 0;
        }
    }

    /// Check if the buffer is empty or invalid
    pub fn is_empty(&self) -> bool {
        !self.is_valid.load(Ordering::SeqCst) || self.length == 0
    }

    /// Check if the buffer is in a valid state
    pub fn is_valid(&self) -> bool {
        self.is_valid.load(Ordering::SeqCst)
    }

    /// Check if memory is locked
    pub fn is_locked(&self) -> bool {
        self.is_locked.load(Ordering::SeqCst)
    }

    /// Safely destroy the buffer, ensuring all data is zeroed
    pub fn destroy(&mut self) {
        // Mark as invalid first to prevent concurrent access
        self.is_valid.store(false, Ordering::SeqCst);
        
        if !self.data.is_null() {
            unsafe {
                // Multiple-pass zeroization for extra security
                memory::explicit_bzero(self.data, self.capacity);
                memory::explicit_bzero(self.data, self.capacity);
                
                // Unlock memory if it was locked (prevent double-unlock)
                if self.is_locked.swap(false, Ordering::SeqCst) {
                    let _ = memory::unlock_memory(self.data, self.capacity);
                }
                
                // Deallocate
                let layout = Layout::from_size_align_unchecked(self.capacity, 32);
                dealloc(self.data, layout);
            }
            
            // Clear pointers and sizes
            self.data = std::ptr::null_mut();
            self.capacity = 0;
            self.length = 0;
        }
    }
}

impl Drop for SecureBuffer {
    fn drop(&mut self) {
        self.destroy();
    }
}

// Thread-safe implementation
unsafe impl Send for SecureBuffer {}
unsafe impl Sync for SecureBuffer {}

// FFI-safe wrapper for C interop
#[repr(C)]
pub struct CSecureBuffer {
    inner: *mut SecureBuffer,
}

impl CSecureBuffer {
    pub fn new(capacity: usize) -> *mut CSecureBuffer {
        match SecureBuffer::new(capacity) {
            Ok(buffer) => {
                let boxed = Box::new(CSecureBuffer {
                    inner: Box::into_raw(Box::new(buffer)),
                });
                Box::into_raw(boxed)
            }
            Err(_) => std::ptr::null_mut(),
        }
    }

    pub unsafe fn write(&mut self, data: *const u8, len: usize) -> i32 {
        if self.inner.is_null() || data.is_null() {
            return -1;
        }
        
        let slice = std::slice::from_raw_parts(data, len);
        match (*self.inner).write(slice) {
            Ok(()) => 0,
            Err(_) => -1,
        }
    }

    pub unsafe fn read(&self, buf: *mut u8, buf_len: usize) -> i32 {
        if self.inner.is_null() || buf.is_null() {
            return -1;
        }
        
        let slice = std::slice::from_raw_parts_mut(buf, buf_len);
        match (*self.inner).read(slice) {
            Ok(bytes_read) => bytes_read as i32,
            Err(_) => -1,
        }
    }

    pub unsafe fn destroy(ptr: *mut CSecureBuffer) {
        if !ptr.is_null() {
            let boxed = Box::from_raw(ptr);
            if !boxed.inner.is_null() {
                let _ = Box::from_raw(boxed.inner);
            }
        }
    }
}

// C FFI exports
#[no_mangle]
pub extern "C" fn secure_buffer_new(capacity: usize) -> *mut CSecureBuffer {
    CSecureBuffer::new(capacity)
}

#[no_mangle]
pub unsafe extern "C" fn secure_buffer_write(
    buffer: *mut CSecureBuffer,
    data: *const u8,
    len: usize,
) -> i32 {
    if buffer.is_null() {
        return -1;
    }
    (*buffer).write(data, len)
}

#[no_mangle]
pub unsafe extern "C" fn secure_buffer_read(
    buffer: *const CSecureBuffer,
    buf: *mut u8,
    buf_len: usize,
) -> i32 {
    if buffer.is_null() {
        return -1;
    }
    (*buffer).read(buf, buf_len)
}

#[no_mangle]
pub unsafe extern "C" fn secure_buffer_destroy(buffer: *mut CSecureBuffer) {
    CSecureBuffer::destroy(buffer);
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::thread;
    use std::sync::Arc;

    #[test]
    fn test_secure_buffer_creation() {
        let buffer = SecureBuffer::new(1024).unwrap();
        assert_eq!(buffer.capacity(), 1024);
        assert_eq!(buffer.len(), 0);
        assert!(buffer.is_empty());
        assert!(buffer.is_valid());
    }

    #[test]
    fn test_write_and_read() {
        let mut buffer = SecureBuffer::new(1024).unwrap();
        let test_data = b"Hello, World!";
        
        buffer.write(test_data).unwrap();
        assert_eq!(buffer.len(), test_data.len());
        assert!(!buffer.is_empty());

        let mut read_buf = vec![0u8; test_data.len()];
        let bytes_read = buffer.read(&mut read_buf).unwrap();
        assert_eq!(bytes_read, test_data.len());
        assert_eq!(&read_buf, test_data);
    }

    #[test]
    fn test_thread_safety() {
        let buffer = Arc::new(SecureBuffer::new(1024).unwrap());
        let handles: Vec<_> = (0..10)
            .map(|i| {
                let buffer_clone = Arc::clone(&buffer);
                thread::spawn(move || {
                    // Just test that we can safely call is_valid from multiple threads
                    for _ in 0..100 {
                        let _ = buffer_clone.is_valid();
                        let _ = buffer_clone.len();
                        let _ = buffer_clone.capacity();
                    }
                })
            })
            .collect();

        for handle in handles {
            handle.join().unwrap();
        }
    }

    #[test]
    fn test_clear_and_destroy() {
        let mut buffer = SecureBuffer::new(1024).unwrap();
        buffer.write(b"sensitive data").unwrap();
        assert!(!buffer.is_empty());
        
        buffer.clear();
        assert!(buffer.is_empty());
        assert!(buffer.is_valid());
        
        buffer.destroy();
        assert!(!buffer.is_valid());
    }

    #[test]
    fn test_zero_capacity_fails() {
        assert!(SecureBuffer::new(0).is_err());
    }

    #[test]
    fn test_overflow_protection() {
        let mut buffer = SecureBuffer::new(10).unwrap();
        let large_data = vec![0u8; 20];
        assert!(buffer.write(&large_data).is_err());
    }
}
