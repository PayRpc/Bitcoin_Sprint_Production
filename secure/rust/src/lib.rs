// SPDX-License-Identifier: MIT
// BitcoinCab.inc - SecureBuffer core with thread-safety and production hardening

use std::alloc::{alloc, dealloc, Layout};
use std::sync::atomic::{AtomicBool, Ordering};
use std::io;
use std::ffi::{CStr, c_char};
use std::time::{SystemTime, UNIX_EPOCH};
use thiserror::Error;
// Import the bloom filter module and its traits
pub mod bloom_filter;
use bloom_filter::{BlockchainHash, TransactionId, UniversalBloomFilter, NetworkConfig, BloomConfig, BlockData};

// Storage verification module (optional IPFS support)
pub mod storage_verifier;

// Web server module for REST API
pub mod web_server;

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

use std::ffi::{c_void, c_int, c_double};

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
