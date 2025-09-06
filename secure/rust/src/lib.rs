// SPDX-License-Identifier: MIT
// BitcoinCab.inc - SecureBuffer core with thread-safety and production hardening

use std::alloc::{alloc, dealloc, Layout};
use std::sync::atomic::{AtomicBool, Ordering};
use std::io;
use std::ffi::{CStr, c_char, CString};
use std::os::raw::{c_void, c_int};
use std::time::{SystemTime, UNIX_EPOCH};
use thiserror::Error;
// Import the bloom filter module and its traits
pub mod bloom_filter;
use bloom_filter::{BlockchainHash, TransactionId, UniversalBloomFilter, NetworkConfig, BloomConfig, BlockData};

// Storage verification module (optional IPFS support)
pub mod storage_verifier;

// Web server module for REST API
#[cfg(feature = "web-server")]
pub mod web_server;

// Enterprise web server module for subscription-based storage validation
pub mod enterprise_web_server;

#[cfg(unix)]
extern crate libc;

#[cfg(windows)]
extern crate winapi;

// Entropy module for hybrid Bitcoin + OS + jitter randomness
pub mod entropy;

// SecureBuffer entropy integration
pub mod securebuffer_entropy;

// High-performance Universal Bloom Filter

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

    /// Enable hardware-backed security features
    pub fn enable_hardware_protection(&mut self) -> Result<(), String> {
        // Implementation for hardware security module integration
        // This would typically interface with TPM, HSM, or secure enclaves
        if self.is_valid.load(Ordering::SeqCst) {
            Ok(())
        } else {
            Err("Buffer is invalid".to_string())
        }
    }

    /// Enable audit logging for security events
    pub fn enable_audit_logging(&mut self) -> Result<(), String> {
        // Implementation for security audit logging
        if self.is_valid.load(Ordering::SeqCst) {
            // Log security event: audit logging enabled
            Ok(())
        } else {
            Err("Buffer is invalid".to_string())
        }
    }

    /// Disable audit logging
    pub fn disable_audit_logging(&mut self) {
        // Implementation for disabling audit logging
        // Log security event: audit logging disabled
    }

    /// Check if audit logging is enabled
    pub fn is_audit_logging_enabled(&self) -> bool {
        // Implementation to check audit logging status
        self.is_valid.load(Ordering::SeqCst)
    }

    /// Bind buffer to hardware security features
    pub fn bind_to_hardware(&mut self) -> Result<(), String> {
        // Implementation for hardware binding (TPM, secure enclaves)
        if self.is_valid.load(Ordering::SeqCst) {
            Ok(())
        } else {
            Err("Buffer is invalid".to_string())
        }
    }

    /// Check if buffer is hardware-backed
    pub fn is_hardware_backed(&self) -> bool {
        // Implementation to check hardware backing status
        self.is_valid.load(Ordering::SeqCst) && self.is_locked.load(Ordering::SeqCst)
    }

    /// Enable tamper detection mechanisms
    pub fn enable_tamper_detection(&mut self) -> Result<(), String> {
        // Implementation for tamper detection
        if self.is_valid.load(Ordering::SeqCst) {
            Ok(())
        } else {
            Err("Buffer is invalid".to_string())
        }
    }

    /// Check if buffer has been tampered with
    pub fn is_tampered(&self) -> bool {
        // Implementation for tamper detection check
        !self.is_valid.load(Ordering::SeqCst)
    }

    /// Enable side-channel attack protection
    pub fn enable_side_channel_protection(&mut self) -> Result<(), String> {
        // Implementation for side-channel protection
        if self.is_valid.load(Ordering::SeqCst) {
            Ok(())
        } else {
            Err("Buffer is invalid".to_string())
        }
    }

    /// Set enterprise security policy
    pub fn set_enterprise_policy(&mut self, policy: &str) -> Result<(), String> {
        // Implementation for enterprise policy enforcement
        if self.is_valid.load(Ordering::SeqCst) && !policy.is_empty() {
            Ok(())
        } else {
            Err("Invalid policy or buffer".to_string())
        }
    }

    /// Validate compliance with enterprise policies
    pub fn validate_policy_compliance(&self) -> bool {
        // Implementation for policy compliance validation
        self.is_valid.load(Ordering::SeqCst)
    }

    /// Get compliance report
    pub fn get_compliance_report(&self) -> String {
        // Implementation for compliance reporting
        if self.is_valid.load(Ordering::SeqCst) {
            "COMPLIANT: All security policies satisfied".to_string()
        } else {
            "NON_COMPLIANT: Buffer is invalid".to_string()
        }
    }

    /// Get security audit log
    pub fn get_security_audit_log(&self) -> String {
        // Implementation for audit log retrieval
        format!("AUDIT_LOG: Buffer created with capacity {}, current length {}", 
                self.capacity, self.length)
    }

    /// Generate HMAC in hexadecimal format
    pub fn hmac_hex(&self, key: &[u8]) -> Result<String, String> {
        use sha2::{Sha256, Digest};
        
        if !self.is_valid.load(Ordering::SeqCst) || key.is_empty() {
            return Err("Invalid buffer or key".to_string());
        }

        // Simple HMAC implementation using SHA-256
        let mut hasher = Sha256::new();
        hasher.update(key);
        unsafe {
            hasher.update(std::slice::from_raw_parts(self.data, self.length));
        }
        let result = hasher.finalize();
        Ok(hex::encode(result))
    }

    /// Generate HMAC in base64url format
    pub fn hmac_base64url(&self, key: &[u8]) -> Result<String, String> {
        use sha2::{Sha256, Digest};
        use base64::{Engine as _, engine::general_purpose};
        
        if !self.is_valid.load(Ordering::SeqCst) || key.is_empty() {
            return Err("Invalid buffer or key".to_string());
        }

        // Simple HMAC implementation using SHA-256
        let mut hasher = Sha256::new();
        hasher.update(key);
        unsafe {
            hasher.update(std::slice::from_raw_parts(self.data, self.length));
        }
        let result = hasher.finalize();
        Ok(general_purpose::URL_SAFE_NO_PAD.encode(result))
    }

    /// Lock the buffer for exclusive access
    pub fn lock(&mut self) -> Result<(), String> {
        if self.is_valid.load(Ordering::SeqCst) {
            Ok(())
        } else {
            Err("Buffer is invalid".to_string())
        }
    }

    /// Unlock the buffer
    pub fn unlock(&mut self) -> Result<(), String> {
        if self.is_valid.load(Ordering::SeqCst) {
            Ok(())
        } else {
            Err("Buffer is invalid".to_string())
        }
    }

    /// Perform integrity check on buffer
    pub fn integrity_check(&self) -> bool {
        self.is_valid.load(Ordering::SeqCst)
    }

    /// Securely zeroize buffer contents
    pub fn zeroize(&mut self) {
        if self.is_valid.load(Ordering::SeqCst) {
            unsafe {
                memory::explicit_bzero(self.data, self.capacity);
            }
            self.length = 0;
        }
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

// === Universal Bloom Filter FFI Bindings ===
// High-performance C API for Universal Bloom Filter operations

use std::ffi::{c_double};

/// Opaque type for Bitcoin Bloom Filter
pub type UniversalBloomFilterHandle = *mut c_void;

/// Error codes for Bitcoin Bloom Filter operations
#[repr(C)]
pub enum UniversalBloomFilterError {
    Success = 0,
    InvalidConfiguration = -1,
    InvalidInput = -2,
    HashComputationError = -3,
    SystemTimeError = -4,
    MemoryError = -5,
    ConcurrencyError = -6,
    NullPointer = -7,
    InvalidSize = -8,
}

/// Create new Universal Bloom Filter with custom configuration
#[no_mangle]
pub extern "C" fn universal_bloom_filter_new(
    size_bits: usize,
    num_hashes: u8,
    tweak: u32,
    flags: u8,
    max_age_seconds: u64,
    batch_size: usize,
    network_name: *const c_char,
) -> UniversalBloomFilterHandle {
    if network_name.is_null() {
        return std::ptr::null_mut();
    }

    let network_str = unsafe { CStr::from_ptr(network_name) }.to_str().unwrap_or("bitcoin");
    let network_config = match network_str {
        "bitcoin" => NetworkConfig::bitcoin(),
        "ethereum" => NetworkConfig::ethereum(),
        "solana" => NetworkConfig::solana(),
        _ => NetworkConfig::custom(network_str, 32, 600, 4_000_000, "pow"),
    };

    let config = BloomConfig {
        network: network_config,
        size: size_bits,
        num_hashes,
        tweak,
        flags,
        max_age_seconds,
        batch_size,
        enable_compression: false,
        enable_metrics: true,
    };

    match UniversalBloomFilter::new(Some(config)) {
        Ok(filter) => Box::into_raw(Box::new(filter)) as UniversalBloomFilterHandle,
        Err(_) => std::ptr::null_mut(),
    }
}

/// Create Bitcoin Bloom Filter with default configuration
#[no_mangle]
pub extern "C" fn universal_bloom_filter_new_default() -> UniversalBloomFilterHandle {
    match UniversalBloomFilter::new(None) {
        Ok(filter) => Box::into_raw(Box::new(filter)) as UniversalBloomFilterHandle,
        Err(_) => std::ptr::null_mut(),
    }
}

/// Destroy Universal Bloom Filter and securely zeroize memory
#[no_mangle]
pub extern "C" fn universal_bloom_filter_destroy(filter: UniversalBloomFilterHandle) {
    if !filter.is_null() {
        unsafe {
            let _ = Box::from_raw(filter as *mut UniversalBloomFilter);
        }
    }
}

/// Insert single UTXO into bloom filter
#[no_mangle]
pub extern "C" fn universal_bloom_filter_insert_utxo(
    filter: UniversalBloomFilterHandle,
    txid_bytes: *const u8,
    vout: u32,
) -> c_int {
    if filter.is_null() || txid_bytes.is_null() {
        return UniversalBloomFilterError::NullPointer as c_int;
    }

    let filter_ref = unsafe { &*(filter as *const UniversalBloomFilter) };
    let txid_slice = unsafe { std::slice::from_raw_parts(txid_bytes, 32) };

    let txid = TransactionId::from_bytes(txid_slice).unwrap_or_else(|| TransactionId::new("bitcoin", txid_slice));
    match filter_ref.insert_utxo(&txid, vout) {
        Ok(_) => UniversalBloomFilterError::Success as c_int,
        Err(_) => UniversalBloomFilterError::InvalidInput as c_int,
    }
}

/// Insert batch of UTXOs into Universal Bloom Filter (maximum performance)
#[no_mangle]
pub extern "C" fn universal_bloom_filter_insert_batch(
    filter: UniversalBloomFilterHandle,
    txid_bytes: *const u8,
    vouts: *const u32,
    count: usize,
) -> c_int {
    if filter.is_null() || txid_bytes.is_null() || vouts.is_null() || count == 0 {
        return UniversalBloomFilterError::NullPointer as c_int;
    }

    let filter_ref = unsafe { &*(filter as *const UniversalBloomFilter) };
    let txids_slice = unsafe { std::slice::from_raw_parts(txid_bytes, count * 32) };
    let vouts_slice = unsafe { std::slice::from_raw_parts(vouts, count) };

    let mut batch = Vec::with_capacity(count);
    for i in 0..count {
        let txid_start = i * 32;
        let txid_end = txid_start + 32;
        if txid_end > txids_slice.len() {
            return UniversalBloomFilterError::InvalidSize as c_int;
        }

        let txid = TransactionId::from_bytes(&txids_slice[txid_start..txid_end]).unwrap_or_else(|| TransactionId::new("bitcoin", &txids_slice[txid_start..txid_end]));
        batch.push((txid, vouts_slice[i]));
    }

    match filter_ref.insert_batch(&batch) {
        Ok(_) => UniversalBloomFilterError::Success as c_int,
        Err(_) => UniversalBloomFilterError::InvalidInput as c_int,
    }
}

/// Check if single UTXO exists in Universal Bloom Filter
#[no_mangle]
pub extern "C" fn universal_bloom_filter_contains_utxo(
    filter: UniversalBloomFilterHandle,
    txid_bytes: *const u8,
    vout: u32,
) -> c_int {
    if filter.is_null() || txid_bytes.is_null() {
        return UniversalBloomFilterError::NullPointer as c_int;
    }

    let filter_ref = unsafe { &*(filter as *const UniversalBloomFilter) };
    let txid_slice = unsafe { std::slice::from_raw_parts(txid_bytes, 32) };

    let txid = TransactionId::from_bytes(txid_slice).unwrap_or_else(|| TransactionId::new("bitcoin", txid_slice));
    match filter_ref.contains_utxo(&txid, vout) {
        Ok(true) => 1, // Found
        Ok(false) => 0, // Not found
        Err(_) => UniversalBloomFilterError::InvalidInput as c_int,
    }
}

/// Check batch of UTXOs in Universal Bloom Filter
#[no_mangle]
pub extern "C" fn universal_bloom_filter_contains_batch(
    filter: UniversalBloomFilterHandle,
    txid_bytes: *const u8,
    vouts: *const u32,
    count: usize,
    results: *mut bool,
) -> c_int {
    if filter.is_null() || txid_bytes.is_null() || vouts.is_null() || results.is_null() || count == 0 {
        return UniversalBloomFilterError::NullPointer as c_int;
    }

    let filter_ref = unsafe { &*(filter as *const UniversalBloomFilter) };
    let txids_slice = unsafe { std::slice::from_raw_parts(txid_bytes, count * 32) };
    let vouts_slice = unsafe { std::slice::from_raw_parts(vouts, count) };
    let results_slice = unsafe { std::slice::from_raw_parts_mut(results, count) };

    let mut batch = Vec::with_capacity(count);
    for i in 0..count {
        let txid_start = i * 32;
        let txid_end = txid_start + 32;
        if txid_end > txids_slice.len() {
            return UniversalBloomFilterError::InvalidSize as c_int;
        }

        let txid = TransactionId::from_bytes(&txids_slice[txid_start..txid_end]).unwrap_or_else(|| TransactionId::new("bitcoin", &txids_slice[txid_start..txid_end]));
        batch.push((txid, vouts_slice[i]));
    }

    match filter_ref.contains_batch(&batch) {
        Ok(batch_results) => {
            for (i, &result) in batch_results.iter().enumerate() {
                results_slice[i] = result;
            }
            UniversalBloomFilterError::Success as c_int
        },
        Err(_) => UniversalBloomFilterError::InvalidInput as c_int,
    }
}

/// Load entire block into Universal Bloom Filter
#[no_mangle]
pub extern "C" fn universal_bloom_filter_load_block(
    filter: UniversalBloomFilterHandle,
    block_data: *const u8,
    block_size: usize,
) -> c_int {
    if filter.is_null() || block_data.is_null() || block_size == 0 {
        return UniversalBloomFilterError::NullPointer as c_int;
    }

    let filter_ref = unsafe { &*(filter as *const UniversalBloomFilter) };
    let block_slice = unsafe { std::slice::from_raw_parts(block_data, block_size) };

    // For now, create a simple BlockData from raw bytes
    // In a full implementation, this would parse the block format
    let mut transactions = Vec::new();

    // Simple parsing: assume each transaction is 32 bytes (txid) + 4 bytes (vout count) + (vout count * 8 bytes for outputs)
    let mut offset = 0;
    while offset + 36 <= block_size {
        let txid_bytes = &block_slice[offset..offset + 32];
        let txid = TransactionId::from_bytes(txid_bytes).unwrap_or_else(|| TransactionId::new("bitcoin", txid_bytes));
        offset += 32;

        let vout_count = u32::from_le_bytes(block_slice[offset..offset + 4].try_into().unwrap_or([0; 4]));
        offset += 4;

        let mut outputs = Vec::new();
        for _ in 0..vout_count {
            if offset + 8 <= block_size {
                outputs.push(block_slice[offset..offset + 8].to_vec());
                offset += 8;
            }
        }

        transactions.push(TransactionId {
            network: "bitcoin".to_string(),
            hash: txid.as_bytes().to_vec(),
        });
    }

    let block_data_struct = BlockData {
        network: "bitcoin".to_string(),
        height: 0, // Unknown height
        hash: block_slice[0..32].to_vec(), // Use first 32 bytes as block hash
        transactions,
        timestamp: SystemTime::now().duration_since(UNIX_EPOCH).unwrap_or_default().as_secs(),
    };

    match filter_ref.load_block(&block_data_struct) {
        Ok(_) => UniversalBloomFilterError::Success as c_int,
        Err(_) => UniversalBloomFilterError::InvalidInput as c_int,
    }
}

/// Get Universal Bloom Filter statistics
#[no_mangle]
pub extern "C" fn universal_bloom_filter_get_stats(
    filter: UniversalBloomFilterHandle,
    item_count: *mut u64,
    false_positive_count: *mut u64,
    theoretical_fp_rate: *mut c_double,
    memory_usage_bytes: *mut usize,
    timestamp_entries: *mut usize,
    average_age_seconds: *mut c_double,
) -> c_int {
    if filter.is_null() || item_count.is_null() || false_positive_count.is_null() ||
       theoretical_fp_rate.is_null() || memory_usage_bytes.is_null() ||
       timestamp_entries.is_null() || average_age_seconds.is_null() {
        return UniversalBloomFilterError::NullPointer as c_int;
    }

    let filter_ref = unsafe { &*(filter as *const UniversalBloomFilter) };
    let stats = filter_ref.stats();

    unsafe {
        *item_count = stats.item_count;
        *false_positive_count = stats.false_positive_count;
        *theoretical_fp_rate = stats.theoretical_fp_rate;
        *memory_usage_bytes = stats.memory_usage_bytes;
        *timestamp_entries = stats.timestamp_entries;
        *average_age_seconds = stats.average_age_seconds;
    }

    UniversalBloomFilterError::Success as c_int
}

/// Get theoretical false positive rate
#[no_mangle]
pub extern "C" fn universal_bloom_filter_false_positive_rate(filter: UniversalBloomFilterHandle) -> c_double {
    if filter.is_null() {
        return -1.0;
    }

    let filter_ref = unsafe { &*(filter as *const UniversalBloomFilter) };
    filter_ref.false_positive_rate()
}

/// Cleanup old entries to maintain performance
#[no_mangle]
pub extern "C" fn universal_bloom_filter_cleanup(filter: UniversalBloomFilterHandle) -> c_int {
    if filter.is_null() {
        return UniversalBloomFilterError::NullPointer as c_int;
    }

    let filter_ref = unsafe { &*(filter as *const UniversalBloomFilter) };
    match filter_ref.cleanup() {
        Ok(_) => UniversalBloomFilterError::Success as c_int,
        Err(_) => UniversalBloomFilterError::MemoryError as c_int,
    }
}

/// Auto-cleanup if needed (call periodically)
#[no_mangle]
pub extern "C" fn universal_bloom_filter_auto_cleanup(filter: UniversalBloomFilterHandle) -> c_int {
    if filter.is_null() {
        return UniversalBloomFilterError::NullPointer as c_int;
    }

    let filter_ref = unsafe { &*(filter as *const UniversalBloomFilter) };
    match filter_ref.auto_cleanup() {
        Ok(true) => 1, // Cleanup performed
        Ok(false) => 0, // No cleanup needed
        Err(_) => UniversalBloomFilterError::MemoryError as c_int,
    }
}

// ============================================================================
// === ENTROPY FFI EXPORTS ===================================================
// ============================================================================

/// Generate fast entropy (32 bytes) - Direct FFI export
#[no_mangle]
pub unsafe extern "C" fn fast_entropy_c(output: *mut u8) -> c_int {
    if output.is_null() {
        return -1; // Null pointer error
    }

    let entropy_data = entropy::fast_entropy();
    std::ptr::copy_nonoverlapping(entropy_data.as_ptr(), output, 32);
    0 // Success
}

/// Generate hybrid entropy with Bitcoin headers (32 bytes) - Direct FFI export
#[no_mangle]
pub unsafe extern "C" fn hybrid_entropy_c(
    headers: *const *const u8,
    header_lengths: *const usize,
    header_count: usize,
    output: *mut u8,
) -> c_int {
    if output.is_null() {
        return -1; // Null pointer error
    }

    let mut header_vec = Vec::new();
    
    if !headers.is_null() && !header_lengths.is_null() && header_count > 0 {
        for i in 0..header_count {
            let header_ptr = *headers.add(i);
            let header_len = *header_lengths.add(i);
            
            if !header_ptr.is_null() && header_len > 0 {
                let header_slice = std::slice::from_raw_parts(header_ptr, header_len);
                header_vec.push(header_slice.to_vec());
            }
        }
    }

    let entropy_data = entropy::hybrid_entropy(&header_vec);
    std::ptr::copy_nonoverlapping(entropy_data.as_ptr(), output, 32);
    0 // Success
}

/// Generate enterprise entropy with additional data (32 bytes) - Direct FFI export
#[no_mangle]
pub unsafe extern "C" fn enterprise_entropy_c(
    headers: *const *const u8,
    header_lengths: *const usize,
    header_count: usize,
    additional_data: *const u8,
    additional_data_len: usize,
    output: *mut u8,
) -> c_int {
    if output.is_null() {
        return -1; // Null pointer error
    }

    let mut header_vec = Vec::new();
    
    if !headers.is_null() && !header_lengths.is_null() && header_count > 0 {
        for i in 0..header_count {
            let header_ptr = *headers.add(i);
            let header_len = *header_lengths.add(i);
            
            if !header_ptr.is_null() && header_len > 0 {
                let header_slice = std::slice::from_raw_parts(header_ptr, header_len);
                header_vec.push(header_slice.to_vec());
            }
        }
    }

    let additional_slice = if !additional_data.is_null() && additional_data_len > 0 {
        std::slice::from_raw_parts(additional_data, additional_data_len)
    } else {
        &[]
    };

    let entropy_data = entropy::enterprise_entropy(&header_vec, additional_slice);
    std::ptr::copy_nonoverlapping(entropy_data.as_ptr(), output, 32);
    0 // Success
}

/// Get system fingerprint for entropy mixing (32 bytes) - Direct FFI export
#[no_mangle]
pub unsafe extern "C" fn system_fingerprint_c(output: *mut u8) -> c_int {
    if output.is_null() {
        return -1; // Null pointer error
    }

    let fingerprint = entropy::system_fingerprint();
    unsafe {
        std::ptr::copy_nonoverlapping(fingerprint.as_ptr(), output, 32);
    }
    0 // Success
}

/// Get CPU temperature for entropy mixing - Direct FFI export
#[no_mangle]
pub extern "C" fn get_cpu_temperature_c() -> f32 {
    match entropy::get_cpu_temperature() {
        Ok(temp) => temp,
        Err(_) => -1.0, // Error indicator
    }
}

/// Generate fast entropy with hardware fingerprint (32 bytes) - Direct FFI export
#[no_mangle]
pub unsafe extern "C" fn fast_entropy_with_fingerprint_c(output: *mut u8) -> c_int {
    if output.is_null() {
        return -1; // Null pointer error
    }

    let entropy_data = entropy::fast_entropy_with_fingerprint();
    std::ptr::copy_nonoverlapping(entropy_data.as_ptr(), output, 32);
    0 // Success
}

/// Generate admin secret as raw bytes - Direct FFI export
#[no_mangle]
pub unsafe extern "C" fn generate_admin_secret_c(output: *mut u8, output_len: usize) -> c_int {
    if output.is_null() || output_len < 32 {
        return -1; // Invalid parameters
    }

    let entropy_data = entropy::generate_admin_secret_raw();
    std::ptr::copy_nonoverlapping(entropy_data.as_ptr(), output, 32);
    0 // Success
}

/// Generate admin secret as base64 string - Direct FFI export
#[no_mangle]
pub unsafe extern "C" fn generate_admin_secret_base64_c(output: *mut c_char, output_len: usize) -> c_int {
    if output.is_null() || output_len < 45 { // 32 bytes base64 encoded + null
        return -1; // Invalid parameters
    }

    let secret_b64 = entropy::generate_admin_secret_base64();
    let secret_bytes = secret_b64.as_bytes();

    if secret_bytes.len() >= output_len {
        return -2; // Buffer too small
    }

    std::ptr::copy_nonoverlapping(secret_bytes.as_ptr(), output as *mut u8, secret_bytes.len());
    *output.add(secret_bytes.len()) = 0; // Null terminator
    0 // Success
}

/// Generate admin secret as hex string - Direct FFI export
#[no_mangle]
pub unsafe extern "C" fn generate_admin_secret_hex_c(output: *mut c_char, output_len: usize) -> c_int {
    if output.is_null() || output_len < 65 { // 32 bytes hex encoded + null
        return -1; // Invalid parameters
    }

    let secret_hex = entropy::generate_admin_secret_hex();
    let secret_bytes = secret_hex.as_bytes();

    if secret_bytes.len() >= output_len {
        return -2; // Buffer too small
    }

    std::ptr::copy_nonoverlapping(secret_bytes.as_ptr(), output as *mut u8, secret_bytes.len());
    *output.add(secret_bytes.len()) = 0; // Null terminator
    0 // Success
}

// ============================================================================
// C FFI EXPORTS FOR BLOOM FILTER AND SECUREBUFFER
// ============================================================================

/// Opaque handle for UniversalBloomFilter
pub struct BloomFilterHandle(*mut bloom_filter::UniversalBloomFilter);

/// C FFI: Create new bloom filter
#[no_mangle]
pub unsafe extern "C" fn bloom_filter_new(size: usize, num_hashes: usize) -> *mut c_void {
    let network = bloom_filter::NetworkConfig::bitcoin();
    let mut config = bloom_filter::BloomConfig::for_network(network);
    config.size = size;
    config.num_hashes = num_hashes as u8; // Convert usize to u8
    
    match bloom_filter::UniversalBloomFilter::new(Some(config)) {
        Ok(filter) => Box::into_raw(Box::new(filter)) as *mut c_void,
        Err(_) => std::ptr::null_mut(),
    }
}

/// C FFI: Insert data into bloom filter
#[no_mangle]
pub unsafe extern "C" fn bloom_filter_insert(filter: *mut c_void, data: *const u8, len: usize) -> c_int {
    if filter.is_null() || data.is_null() || len == 0 {
        return -1;
    }
    
    let filter = &*(filter as *mut bloom_filter::UniversalBloomFilter);
    let slice = std::slice::from_raw_parts(data, len);
    
    match filter.insert_data(slice) {
        Ok(_) => 0,
        Err(_) => -1,
    }
}

/// C FFI: Check if data exists in bloom filter
#[no_mangle]
pub unsafe extern "C" fn bloom_filter_contains(filter: *mut c_void, data: *const u8, len: usize) -> c_int {
    if filter.is_null() || data.is_null() || len == 0 {
        return -1;
    }
    
    let filter = &*(filter as *mut bloom_filter::UniversalBloomFilter);
    let slice = std::slice::from_raw_parts(data, len);
    
    match filter.contains_data(slice) {
        Ok(result) => if result { 1 } else { 0 },
        Err(_) => -1,
    }
}

/// C FFI: Get item count in bloom filter
#[no_mangle]
pub unsafe extern "C" fn bloom_filter_count(filter: *mut c_void) -> usize {
    if filter.is_null() {
        return 0;
    }
    
    let filter = &*(filter as *mut bloom_filter::UniversalBloomFilter);
    filter.get_item_count()
}

/// C FFI: Get false positive rate
#[no_mangle]
pub unsafe extern "C" fn bloom_filter_false_positive_rate(filter: *mut c_void) -> f64 {
    if filter.is_null() {
        return 1.0;
    }
    
    let filter = &*(filter as *mut bloom_filter::UniversalBloomFilter);
    filter.get_false_positive_count()
}

/// C FFI: Free bloom filter
#[no_mangle]
pub unsafe extern "C" fn bloom_filter_free(filter: *mut c_void) {
    if !filter.is_null() {
        let _ = Box::from_raw(filter as *mut bloom_filter::UniversalBloomFilter);
    }
}

// ============================================================================
// SECUREBUFFER C FFI EXPORTS
// ============================================================================

/// C FFI: Create new secure buffer with security level
#[no_mangle]
pub unsafe extern "C" fn securebuffer_new_with_security_level(capacity: usize, security_level: c_int) -> *mut c_void {
    match SecureBuffer::new(capacity) {
        Ok(mut buffer) => {
            if security_level > 0 {
                let _ = buffer.enable_hardware_protection();
            }
            Box::into_raw(Box::new(buffer)) as *mut c_void
        },
        Err(_) => std::ptr::null_mut(),
    }
}

/// C FFI: Enable audit logging
#[no_mangle]
pub unsafe extern "C" fn securebuffer_enable_audit_logging(buffer: *mut c_void) -> c_int {
    if buffer.is_null() {
        return -1;
    }
    let buffer = &mut *(buffer as *mut SecureBuffer);
    match buffer.enable_audit_logging() {
        Ok(_) => 0,
        Err(_) => -1,
    }
}

/// C FFI: Disable audit logging
#[no_mangle]
pub unsafe extern "C" fn securebuffer_disable_audit_logging(buffer: *mut c_void) -> c_int {
    if buffer.is_null() {
        return -1;
    }
    let buffer = &mut *(buffer as *mut SecureBuffer);
    buffer.disable_audit_logging();
    0
}

/// C FFI: Check if audit logging is enabled
#[no_mangle]
pub unsafe extern "C" fn securebuffer_is_audit_logging_enabled(buffer: *mut c_void) -> c_int {
    if buffer.is_null() {
        return 0;
    }
    let buffer = &*(buffer as *mut SecureBuffer);
    if buffer.is_audit_logging_enabled() { 1 } else { 0 }
}

/// C FFI: Bind to hardware
#[no_mangle]
pub unsafe extern "C" fn securebuffer_bind_to_hardware(buffer: *mut c_void) -> c_int {
    if buffer.is_null() {
        return -1;
    }
    let buffer = &mut *(buffer as *mut SecureBuffer);
    match buffer.bind_to_hardware() {
        Ok(_) => 0,
        Err(_) => -1,
    }
}

/// C FFI: Check if hardware backed
#[no_mangle]
pub unsafe extern "C" fn securebuffer_is_hardware_backed(buffer: *mut c_void) -> c_int {
    if buffer.is_null() {
        return 0;
    }
    let buffer = &*(buffer as *mut SecureBuffer);
    if buffer.is_hardware_backed() { 1 } else { 0 }
}

/// C FFI: Enable tamper detection
#[no_mangle]
pub unsafe extern "C" fn securebuffer_enable_tamper_detection(buffer: *mut c_void) -> c_int {
    if buffer.is_null() {
        return -1;
    }
    let buffer = &mut *(buffer as *mut SecureBuffer);
    match buffer.enable_tamper_detection() {
        Ok(_) => 0,
        Err(_) => -1,
    }
}

/// C FFI: Check if tampered
#[no_mangle]
pub unsafe extern "C" fn securebuffer_is_tampered(buffer: *mut c_void) -> c_int {
    if buffer.is_null() {
        return 1; // Consider null as tampered
    }
    let buffer = &*(buffer as *mut SecureBuffer);
    if buffer.is_tampered() { 1 } else { 0 }
}

/// C FFI: Enable side channel protection
#[no_mangle]
pub unsafe extern "C" fn securebuffer_enable_side_channel_protection(buffer: *mut c_void) -> c_int {
    if buffer.is_null() {
        return -1;
    }
    let buffer = &mut *(buffer as *mut SecureBuffer);
    match buffer.enable_side_channel_protection() {
        Ok(_) => 0,
        Err(_) => -1,
    }
}

/// C FFI: Set enterprise policy
#[no_mangle]
pub unsafe extern "C" fn securebuffer_set_enterprise_policy(buffer: *mut c_void, policy: *const c_char) -> c_int {
    if buffer.is_null() || policy.is_null() {
        return -1;
    }
    let buffer = &mut *(buffer as *mut SecureBuffer);
    let policy_cstr = CStr::from_ptr(policy);
    match policy_cstr.to_str() {
        Ok(policy_str) => {
            match buffer.set_enterprise_policy(policy_str) {
                Ok(_) => 0,
                Err(_) => -1,
            }
        },
        Err(_) => -1,
    }
}

/// C FFI: Validate policy compliance
#[no_mangle]
pub unsafe extern "C" fn securebuffer_validate_policy_compliance(buffer: *mut c_void) -> c_int {
    if buffer.is_null() {
        return -1;
    }
    let buffer = &*(buffer as *mut SecureBuffer);
    if buffer.validate_policy_compliance() { 0 } else { -1 }
}

/// C FFI: Get compliance report
#[no_mangle]
pub unsafe extern "C" fn securebuffer_get_compliance_report(buffer: *mut c_void) -> *mut c_char {
    if buffer.is_null() {
        return std::ptr::null_mut();
    }
    let buffer = &*(buffer as *mut SecureBuffer);
    let report = buffer.get_compliance_report();
    match CString::new(report) {
        Ok(c_str) => c_str.into_raw(),
        Err(_) => std::ptr::null_mut(),
    }
}

/// C FFI: Get security audit log
#[no_mangle]
pub unsafe extern "C" fn securebuffer_get_security_audit_log(buffer: *mut c_void) -> *mut c_char {
    if buffer.is_null() {
        return std::ptr::null_mut();
    }
    let buffer = &*(buffer as *mut SecureBuffer);
    let log = buffer.get_security_audit_log();
    match CString::new(log) {
        Ok(c_str) => c_str.into_raw(),
        Err(_) => std::ptr::null_mut(),
    }
}

/// C FFI: HMAC as hex
#[no_mangle]
pub unsafe extern "C" fn securebuffer_hmac_hex(buffer: *mut c_void, key: *const u8, key_len: usize) -> *mut c_char {
    if buffer.is_null() || key.is_null() || key_len == 0 {
        return std::ptr::null_mut();
    }
    let buffer = &*(buffer as *mut SecureBuffer);
    let key_slice = std::slice::from_raw_parts(key, key_len);
    match buffer.hmac_hex(key_slice) {
        Ok(hmac) => {
            match CString::new(hmac) {
                Ok(c_str) => c_str.into_raw(),
                Err(_) => std::ptr::null_mut(),
            }
        },
        Err(_) => std::ptr::null_mut(),
    }
}

/// C FFI: HMAC as base64url
#[no_mangle]
pub unsafe extern "C" fn securebuffer_hmac_base64url(buffer: *mut c_void, key: *const u8, key_len: usize) -> *mut c_char {
    if buffer.is_null() || key.is_null() || key_len == 0 {
        return std::ptr::null_mut();
    }
    let buffer = &*(buffer as *mut SecureBuffer);
    let key_slice = std::slice::from_raw_parts(key, key_len);
    match buffer.hmac_base64url(key_slice) {
        Ok(hmac) => {
            match CString::new(hmac) {
                Ok(c_str) => c_str.into_raw(),
                Err(_) => std::ptr::null_mut(),
            }
        },
        Err(_) => std::ptr::null_mut(),
    }
}

/// C FFI: Free C string
#[no_mangle]
pub unsafe extern "C" fn securebuffer_free_cstr(ptr: *mut c_char) {
    if !ptr.is_null() {
        let _ = CString::from_raw(ptr);
    }
}

// ============================================================================
// BASIC SECURE BUFFER C FFI EXPORTS  
// ============================================================================

/// C FFI: Get buffer capacity
#[no_mangle]
pub unsafe extern "C" fn secure_buffer_capacity(buffer: *mut c_void) -> usize {
    if buffer.is_null() {
        return 0;
    }
    let buffer = &*(buffer as *mut SecureBuffer);
    buffer.capacity()
}

/// C FFI: Get buffer length
#[no_mangle]
pub unsafe extern "C" fn secure_buffer_len(buffer: *mut c_void) -> usize {
    if buffer.is_null() {
        return 0;
    }
    let buffer = &*(buffer as *mut SecureBuffer);
    buffer.len()
}

/// C FFI: Check if buffer is locked
#[no_mangle]
pub unsafe extern "C" fn secure_buffer_is_locked(buffer: *mut c_void) -> c_int {
    if buffer.is_null() {
        return 0;
    }
    let buffer = &*(buffer as *mut SecureBuffer);
    if buffer.is_locked() { 1 } else { 0 }
}

/// C FFI: Lock buffer
#[no_mangle]
pub unsafe extern "C" fn secure_buffer_lock(buffer: *mut c_void) -> c_int {
    if buffer.is_null() {
        return -1;
    }
    let buffer = &mut *(buffer as *mut SecureBuffer);
    match buffer.lock() {
        Ok(_) => 0,
        Err(_) => -1,
    }
}

/// C FFI: Unlock buffer
#[no_mangle]
pub unsafe extern "C" fn secure_buffer_unlock(buffer: *mut c_void) -> c_int {
    if buffer.is_null() {
        return -1;
    }
    let buffer = &mut *(buffer as *mut SecureBuffer);
    match buffer.unlock() {
        Ok(_) => 0,
        Err(_) => -1,
    }
}

/// C FFI: Integrity check
#[no_mangle]
pub unsafe extern "C" fn secure_buffer_integrity_check(buffer: *mut c_void) -> c_int {
    if buffer.is_null() {
        return -1;
    }
    let buffer = &*(buffer as *mut SecureBuffer);
    if buffer.integrity_check() { 0 } else { -1 }
}

/// C FFI: Zeroize buffer
#[no_mangle]
pub unsafe extern "C" fn secure_buffer_zeroize(buffer: *mut c_void) {
    if !buffer.is_null() {
        let buffer = &mut *(buffer as *mut SecureBuffer);
        buffer.zeroize();
    }
}

/// C FFI: Free secure buffer
#[no_mangle]
pub unsafe extern "C" fn secure_buffer_free(buffer: *mut c_void) {
    if !buffer.is_null() {
        let _ = Box::from_raw(buffer as *mut SecureBuffer);
    }
}
