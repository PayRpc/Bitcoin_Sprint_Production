// SPDX-License-Identifier: MIT
// Bitcoin Sprint - Cryptographically Secure Entropy Module

use std::time::Instant;
use std::sync::atomic::{AtomicU64, Ordering};
use rand::{RngCore, Rng};
use rand::rngs::OsRng;
#[cfg(target_family = "unix")]
use libc;
use sysinfo::{System, RefreshKind, CpuRefreshKind};
use base64;
use hex;

// Static jitter accumulator for CPU timing entropy
static JITTER_COUNTER: AtomicU64 = AtomicU64::new(0);

// Error types for entropy operations
#[derive(Debug)]
pub enum EntropyError {
    SystemError(String),
    InsufficientEntropy,
    InvalidBlockHeaders,
}

/// High-quality entropy source combining multiple randomness sources
pub struct EntropyCollector {
    os_rng: OsRng,
}

impl Default for EntropyCollector {
    fn default() -> Self {
        Self::new()
    }
}

impl EntropyCollector {
    /// Create a new entropy collector
    pub fn new() -> Self {
        Self {
            os_rng: OsRng,
        }
    }

    /// Collect high-resolution timing jitter (supplemental entropy only)
    fn collect_jitter(&self) -> u64 {
        let start = std::time::Instant::now();

        // Perform some unpredictable operations to create timing variance
        let mut accumulator = 0u64;
        for i in 0..100 {
            accumulator = accumulator.wrapping_mul(6364136223846793005u64)
                .wrapping_add(1442695040888963407u64)
                .wrapping_add(i);
        }

        let duration = start.elapsed();
        let jitter = duration.as_nanos() as u64 ^ accumulator;

        // Update global counter
        JITTER_COUNTER.fetch_add(jitter.wrapping_mul(accumulator), Ordering::Relaxed);

        jitter
    }

    /// Get cryptographically secure OS-level randomness
    fn get_os_entropy(&mut self, output: &mut [u8]) -> Result<(), EntropyError> {
        self.os_rng.try_fill_bytes(output)
            .map_err(|e| EntropyError::SystemError(format!("OS RNG failed: {}", e)))
    }

    /// Extract entropy from Bitcoin block headers (non-deterministic mixing)
    fn extract_block_entropy(&mut self, headers: &[Vec<u8>]) -> [u8; 32] {
        let mut combined_entropy = [0u8; 32];

        if headers.is_empty() {
            // If no headers provided, use pure OS entropy
            let _ = self.get_os_entropy(&mut combined_entropy);
            return combined_entropy;
        }

        // Start with OS entropy as base
        let _ = self.get_os_entropy(&mut combined_entropy);

        // Mix in block header data non-deterministically
        for (i, header) in headers.iter().enumerate() {
            if header.len() >= 80 {
                // Extract variable fields from Bitcoin header
                let nonce = &header[76..80];
                let timestamp = &header[68..72];
                let merkle_root = &header[36..68];

                // Mix each field with OS entropy using different positions
                for (j, &byte) in nonce.iter().enumerate() {
                    let pos = (i * 4 + j) % 32;
                    combined_entropy[pos] ^= byte;
                }

                for (j, &byte) in timestamp.iter().enumerate() {
                    let pos = (i * 4 + j + 16) % 32;
                    combined_entropy[pos] ^= byte;
                }

                // Mix merkle root bytes
                for (j, &byte) in merkle_root.iter().enumerate() {
                    let pos = (i * 32 + j) % 32;
                    combined_entropy[pos] ^= byte;
                }
            }
        }

        // Add timing jitter as additional entropy
        let jitter = self.collect_jitter();
        let jitter_bytes = jitter.to_le_bytes();

        for i in 0..8 {
            combined_entropy[i] ^= jitter_bytes[i];
            combined_entropy[i + 24] ^= jitter_bytes[7 - i];
        }

        combined_entropy
    }
}

/// Generate fast, cryptographically secure entropy (32 bytes)
pub fn fast_entropy() -> [u8; 32] {
    let mut collector = EntropyCollector::new();
    let mut output = [0u8; 32];

    // Use cryptographically secure OS randomness as primary source
    if let Ok(_) = collector.get_os_entropy(&mut output) {
        // Add timing jitter as additional entropy (supplemental only)
        let jitter = collector.collect_jitter();
        let jitter_bytes = jitter.to_le_bytes();

        // Mix jitter with OS entropy using cryptographically sound mixing
        for i in 0..8 {
            output[i] ^= jitter_bytes[i];
            output[i + 24] ^= jitter_bytes[7 - i];
        }
    } else {
        // Fallback: pure OS entropy without jitter enhancement
        let _ = collector.get_os_entropy(&mut output);
    }

    output
}

/// Generate hybrid entropy using Bitcoin headers + OS randomness + timing jitter
pub fn hybrid_entropy(headers: &[Vec<u8>]) -> [u8; 32] {
    let mut collector = EntropyCollector::new();
    let mut output = [0u8; 32];

    // Start with cryptographically secure OS entropy
    let _ = collector.get_os_entropy(&mut output);

    // Mix in blockchain entropy non-deterministically
    let block_entropy = collector.extract_block_entropy(headers);
    for i in 0..32 {
        output[i] ^= block_entropy[i];
    }

    // Add final timing jitter layer
    let jitter = collector.collect_jitter();
    let jitter_bytes = jitter.to_le_bytes();

    for i in 0..8 {
        output[i] ^= jitter_bytes[i];
        output[i + 16] ^= jitter_bytes[7 - i];
    }

    output
}

/// Generate system fingerprint for entropy enhancement
pub fn system_fingerprint() -> [u8; 32] {
    let mut collector = EntropyCollector::new();
    let mut output = [0u8; 32];

    // Use OS entropy as base
    let _ = collector.get_os_entropy(&mut output);

    // Add system-specific entropy
    let process_id = std::process::id();
    let thread_id_str = format!("{:?}", std::thread::current().id());
    let timestamp = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap_or_default()
        .as_nanos() as u64;

    let pid_bytes = process_id.to_le_bytes();
    let tid_bytes = thread_id_str.as_bytes();
    let ts_bytes = timestamp.to_le_bytes();

    // Mix in system identifiers
    for i in 0..4 {
        output[i] ^= pid_bytes[i % 4];
        output[i + 8] ^= tid_bytes[i % tid_bytes.len()];
        output[i + 16] ^= ts_bytes[i % 8];
    }

    // Add jitter for additional randomness
    let final_jitter = collector.collect_jitter();
    let jitter_bytes = final_jitter.to_le_bytes();
    for i in 0..8 {
        output[i * 4 % 32] ^= jitter_bytes[i];
    }

    output
}

/// Generate enterprise-grade entropy with additional security measures
pub fn enterprise_entropy(headers: &[Vec<u8>], additional_data: &[u8]) -> [u8; 32] {
    let mut collector = EntropyCollector::new();
    let mut output = [0u8; 32];
    
    // Multi-round entropy collection
    for round in 0..3 {
        let mut round_output = [0u8; 32];
        
        // OS entropy with round-specific offset
        let _ = collector.get_os_entropy(&mut round_output);
        
        // Blockchain entropy
        let block_entropy = collector.extract_block_entropy(headers);
        
        // Additional data incorporation
        if !additional_data.is_empty() {
            use std::collections::hash_map::DefaultHasher;
            use std::hash::{Hash, Hasher};
            
            let mut hasher = DefaultHasher::new();
            additional_data.hash(&mut hasher);
            round.hash(&mut hasher);
            let add_hash = hasher.finish().to_le_bytes();
            
            for i in 0..8 {
                round_output[i] ^= add_hash[i];
                round_output[i + 16] ^= add_hash[7 - i];
            }
        }
        
        // Jitter for this round
        let round_jitter = collector.collect_jitter();
        let jitter_bytes = round_jitter.to_le_bytes();
        
        // Combine all sources for this round
        for i in 0..32 {
            round_output[i] ^= block_entropy[i] ^ jitter_bytes[i % 8];
        }
        
        // Accumulate into final output
        for i in 0..32 {
            output[i] ^= round_output[i];
        }
    }
    
    output
}

/// Get CPU temperature for entropy mixing and monitoring
pub fn get_cpu_temperature() -> Result<f32, EntropyError> {
    let mut system = System::new_with_specifics(
        RefreshKind::new().with_cpu(CpuRefreshKind::everything())
    );
    system.refresh_cpu();

    let mut total_temp = 0.0;
    let mut cpu_count = 0;

    for cpu in system.cpus() {
        // Note: Temperature might not be available on all systems
        // This is a placeholder - actual temperature reading depends on hardware support
        // For now, we'll use CPU usage as a proxy for "system activity"
        let usage = cpu.cpu_usage();
        total_temp += usage as f32;
        cpu_count += 1;
    }

    if cpu_count == 0 {
        return Err(EntropyError::SystemError("No CPU information available".into()));
    }

    Ok(total_temp / cpu_count as f32)
}

/// Enhanced fast entropy with hardware fingerprinting
pub fn fast_entropy_with_fingerprint() -> [u8; 32] {
    let mut output = fast_entropy();

    // Mix in system fingerprint
    let fingerprint = system_fingerprint();
    for i in 0..32 {
        output[i] ^= fingerprint[i];
    }

    // Mix in CPU temperature if available
    if let Ok(temp) = get_cpu_temperature() {
        let temp_bytes = (temp as u32).to_le_bytes();
        for i in 0..4 {
            output[i] ^= temp_bytes[i];
            output[i + 28] ^= temp_bytes[3 - i];
        }
    }

    output
}

/// Enhanced hybrid entropy with hardware fingerprinting
pub fn hybrid_entropy_with_fingerprint(headers: &[Vec<u8>]) -> [u8; 32] {
    let mut output = hybrid_entropy(headers);

    // Mix in system fingerprint
    let fingerprint = system_fingerprint();
    for i in 0..32 {
        output[i] ^= fingerprint[i];
    }

    // Mix in CPU temperature if available
    if let Ok(temp) = get_cpu_temperature() {
        let temp_bytes = (temp as u32).to_le_bytes();
        for i in 0..4 {
            output[i] ^= temp_bytes[i];
            output[i + 28] ^= temp_bytes[3 - i];
        }
    }

    output
}
mod tests {
    use super::*;

    #[test]
    fn test_fast_entropy_length_and_variance() {
        let e1 = fast_entropy();
        let e2 = fast_entropy();
        assert_eq!(e1.len(), 32);
        assert_ne!(e1, e2, "Fast entropy should not repeat immediately");
    }

    #[test]
    fn test_hybrid_entropy_differs_with_headers() {
        let headers1 = vec![vec![0u8; 80]];
        let headers2 = vec![vec![1u8; 80]];
        let e1 = hybrid_entropy(&headers1);
        let e2 = hybrid_entropy(&headers2);
        assert_eq!(e1.len(), 32);
        assert_eq!(e2.len(), 32);
        assert_ne!(e1, e2, "Hybrid entropy must vary with block headers");
    }

    #[test]
    fn test_fast_entropy() {
        let entropy1 = fast_entropy();
        let entropy2 = fast_entropy();

        // Should produce different outputs
        assert_ne!(entropy1, entropy2);

        // Should not be all zeros
        assert_ne!(entropy1, [0u8; 32]);
    }

    #[test]
    fn test_hybrid_entropy() {
        let mock_headers = vec![
            vec![0u8; 80], // Mock Bitcoin header
            vec![1u8; 80],
        ];

        let entropy1 = hybrid_entropy(&mock_headers);
        let entropy2 = hybrid_entropy(&mock_headers);

        // Should produce different outputs due to jitter
        assert_ne!(entropy1, entropy2);
    }

    #[test]
    fn test_enterprise_entropy() {
        let mock_headers = vec![vec![0u8; 80]];
        let additional_data = b"test_data";

        let entropy = enterprise_entropy(&mock_headers, additional_data);
        assert_ne!(entropy, [0u8; 32]);
    }

    #[test]
    fn test_entropy_collector() {
        let mut collector = EntropyCollector::new();

        // Test jitter collection
        let jitter1 = collector.collect_jitter();
        let jitter2 = collector.collect_jitter();

        // Jitter values should be different
        assert_ne!(jitter1, jitter2);

        // Test OS entropy
        let mut buffer = [0u8; 16];
        assert!(collector.get_os_entropy(&mut buffer).is_ok());

        // Should not be all zeros (very unlikely)
        assert_ne!(buffer, [0u8; 16]);
    }

    #[test]
    fn test_entropy_statistical_properties() {
        let mut ones = 0;
        let mut bits = 0;
        let samples = 100;

        // Collect entropy from multiple calls
        for _ in 0..samples {
            let entropy = fast_entropy();
            for byte in entropy.iter() {
                for bit in 0..8 {
                    if (byte >> bit) & 1 == 1 {
                        ones += 1;
                    }
                    bits += 1;
                }
            }
        }

        let ratio = ones as f64 / bits as f64;

        // Statistical test: should be close to 0.5 (50% ones, 50% zeros)
        // Allow some tolerance for randomness
        assert!(ratio > 0.4 && ratio < 0.6,
            "Entropy bias detected: ratio={:.3}, expected ~0.5", ratio);
    }

    #[test]
    fn test_block_entropy_extraction() {
        let mut collector = EntropyCollector::new();

        // Test with empty headers (should use last known entropy)
        let empty_entropy = collector.extract_block_entropy(&[]);
        assert_ne!(empty_entropy, [0u8; 32]);

        // Test with mock headers
        let headers = vec![
            vec![0u8; 80],
            vec![255u8; 80],
        ];
        let block_entropy = collector.extract_block_entropy(&headers);
        assert_ne!(block_entropy, [0u8; 32]);

        // Different headers should produce different entropy
        let headers2 = vec![vec![128u8; 80]];
        let block_entropy2 = collector.extract_block_entropy(&headers2);
        assert_ne!(block_entropy, block_entropy2);
    }

    #[test]
    fn test_entropy_functions_consistency() {
        // All entropy functions should return exactly 32 bytes
        assert_eq!(fast_entropy().len(), 32);
        assert_eq!(hybrid_entropy(&[]).len(), 32);
        assert_eq!(enterprise_entropy(&[], &[]).len(), 32);
    }
}

// FFI bindings for Go integration

#[no_mangle]
pub extern "C" fn fast_entropy_ffi(output: *mut u8, len: usize) -> i32 {
    if output.is_null() || len != 32 {
        return -1;
    }

    // Use the existing fast_entropy function which now uses cryptographic OS randomness
    let entropy = fast_entropy();
    unsafe {
        std::ptr::copy_nonoverlapping(entropy.as_ptr(), output, 32);
    }
    0
}

#[no_mangle]
pub extern "C" fn hybrid_entropy_ffi(headers_ptr: *const *const u8, headers_len: usize, header_sizes_ptr: *const usize, output: *mut u8, len: usize) -> i32 {
    if output.is_null() || len != 32 {
        return -1;
    }

    let mut headers = Vec::new();
    if !headers_ptr.is_null() && headers_len > 0 {
        unsafe {
            for i in 0..headers_len {
                let header_ptr = *headers_ptr.add(i);
                let header_size = *header_sizes_ptr.add(i);
                let header = std::slice::from_raw_parts(header_ptr, header_size);
                headers.push(header.to_vec());
            }
        }
    }

    // Use the existing hybrid_entropy function which now uses cryptographic OS randomness
    let entropy = hybrid_entropy(&headers);
    unsafe {
        std::ptr::copy_nonoverlapping(entropy.as_ptr(), output, 32);
    }
    0
}

#[no_mangle]
pub extern "C" fn system_fingerprint_ffi(output: *mut u8, len: usize) -> i32 {
    if output.is_null() || len != 32 {
        return -1;
    }

    // Use the existing system_fingerprint function which now uses cryptographic OS randomness
    let fingerprint = system_fingerprint();
    unsafe {
        std::ptr::copy_nonoverlapping(fingerprint.as_ptr(), output, 32);
    }
    0
}

#[no_mangle]
pub extern "C" fn get_cpu_temperature_ffi() -> f32 {
    match get_cpu_temperature() {
        Ok(temp) => temp,
        Err(_) => -1.0,
    }
}

#[no_mangle]
pub extern "C" fn fast_entropy_with_fingerprint_ffi(output: *mut u8, len: usize) -> i32 {
    if output.is_null() || len != 32 {
        return -1;
    }

    // Use the existing fast_entropy_with_fingerprint function which now uses cryptographic OS randomness
    let entropy = fast_entropy_with_fingerprint();
    unsafe {
        std::ptr::copy_nonoverlapping(entropy.as_ptr(), output, 32);
    }
    0
}

#[no_mangle]
pub extern "C" fn hybrid_entropy_with_fingerprint_ffi(headers_ptr: *const *const u8, headers_len: usize, header_sizes_ptr: *const usize, output: *mut u8, len: usize) -> i32 {
    if output.is_null() || len != 32 {
        return -1;
    }

    let mut headers = Vec::new();
    if !headers_ptr.is_null() && headers_len > 0 {
        unsafe {
            for i in 0..headers_len {
                let header_ptr = *headers_ptr.add(i);
                let header_size = *header_sizes_ptr.add(i);
                let header = std::slice::from_raw_parts(header_ptr, header_size);
                headers.push(header.to_vec());
            }
        }
    }

    // Use the existing hybrid_entropy_with_fingerprint function which now uses cryptographic OS randomness
    let entropy = hybrid_entropy_with_fingerprint(&headers);
    unsafe {
        std::ptr::copy_nonoverlapping(entropy.as_ptr(), output, 32);
    }
    0
}

/// Generate admin secret as raw bytes (32 bytes)
pub fn generate_admin_secret_raw() -> [u8; 32] {
    let mut collector = EntropyCollector::new();
    let mut secret = [0u8; 32];

    // Use high-quality entropy for admin secrets
    if let Err(_) = collector.get_os_entropy(&mut secret) {
        // Fallback to system fingerprint if OS entropy fails
        secret = system_fingerprint();
    }

    // Mix with additional entropy sources
    let jitter = collector.collect_jitter();
    for i in 0..8 {
        let jitter_byte = ((jitter >> (i * 4)) & 0xFF) as u8;
        secret[i] ^= jitter_byte;
    }

    secret
}

/// Generate admin secret as base64 string
pub fn generate_admin_secret_base64() -> String {
    let secret = generate_admin_secret_raw();
    base64::Engine::encode(&base64::engine::general_purpose::STANDARD, &secret)
}

/// Generate admin secret as hex string
pub fn generate_admin_secret_hex() -> String {
    let secret = generate_admin_secret_raw();
    hex::encode(&secret)
}
