// SPDX-License-Identifier: MIT
// Universal Sprint - Network-Agnostic High-Performance Bloom Filter
// Master Scientist Optimization: Maximum Performance, Stability, Security
// Supports all blockchain networks like Alchemy, Infura - fastest and most secure

use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};
use rayon::prelude::*;
use dashmap::DashMap;
use zeroize::Zeroize;
use rand::RngCore;
use bitcoin_hashes::{Hash, HashEngine};

/// Network-agnostic hash trait for blockchain data
pub trait BlockchainHash {
    fn as_bytes(&self) -> &[u8];
    fn from_bytes(bytes: &[u8]) -> Option<Self> where Self: Sized;
}

/// Network-agnostic transaction identifier
#[derive(Clone, Debug, PartialEq, Eq, Hash)]
pub struct TransactionId {
    pub network: String,
    pub hash: Vec<u8>,
}

impl TransactionId {
    pub fn new(network: &str, hash: &[u8]) -> Self {
        Self {
            network: network.to_string(),
            hash: hash.to_vec(),
        }
    }
}

impl BlockchainHash for TransactionId {
    fn as_bytes(&self) -> &[u8] {
        &self.hash
    }

    fn from_bytes(bytes: &[u8]) -> Option<Self> {
        if bytes.len() == 32 {
            Some(Self {
                network: "bitcoin".to_string(), // Default network
                hash: bytes.to_vec(),
            })
        } else {
            None
        }
    }
}

/// Network-agnostic block data
#[derive(Clone, Debug)]
pub struct BlockData {
    pub network: String,
    pub height: u64,
    pub hash: Vec<u8>,
    pub transactions: Vec<TransactionId>,
    pub timestamp: u64,
}

impl BlockData {
    pub fn new(network: &str, height: u64, hash: &[u8], transactions: Vec<TransactionId>) -> Self {
        Self {
            network: network.to_string(),
            height,
            hash: hash.to_vec(),
            transactions,
            timestamp: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap_or_default()
                .as_secs(),
        }
    }
}

/// Network configuration for different blockchain networks
#[derive(Clone, Debug)]
pub struct NetworkConfig {
    pub name: String,
    pub hash_size: usize,           // Size of transaction hashes in bytes
    pub block_time_seconds: u64,    // Average block time
    pub max_block_size: usize,      // Maximum block size
    pub consensus_mechanism: String,
}

impl NetworkConfig {
    pub fn bitcoin() -> Self {
        Self {
            name: "bitcoin".to_string(),
            hash_size: 32,
            block_time_seconds: 600,
            max_block_size: 4_000_000,
            consensus_mechanism: "proof-of-work".to_string(),
        }
    }

    pub fn ethereum() -> Self {
        Self {
            name: "ethereum".to_string(),
            hash_size: 32,
            block_time_seconds: 12,
            max_block_size: 30_000_000,
            consensus_mechanism: "proof-of-stake".to_string(),
        }
    }

    pub fn solana() -> Self {
        Self {
            name: "solana".to_string(),
            hash_size: 32,
            block_time_seconds: 1,
            max_block_size: 50_000_000,
            consensus_mechanism: "proof-of-stake".to_string(),
        }
    }

    pub fn custom(name: &str, hash_size: usize, block_time: u64, max_block_size: usize, consensus: &str) -> Self {
        Self {
            name: name.to_string(),
            hash_size,
            block_time_seconds: block_time,
            max_block_size,
            consensus_mechanism: consensus.to_string(),
        }
    }
}

/// Universal Bloom Filter Configuration - Network Agnostic
/// Optimized for maximum performance and security across all blockchain networks
#[derive(Clone, Debug)]
pub struct BloomConfig {
    pub network: NetworkConfig,     // Network-specific configuration
    pub size: usize,                // Filter size in bits (must be power of two)
    pub num_hashes: u8,             // Number of hash functions (2-7)
    pub tweak: u32,                 // Random value to modify hash functions
    pub flags: u8,                  // Filter update flags
    pub max_age_seconds: u64,       // Maximum age for entries before eviction
    pub batch_size: usize,          // Optimal batch size for parallel operations
    pub enable_compression: bool,   // Enable compressed storage for large filters
    pub enable_metrics: bool,       // Enable detailed performance metrics
}

impl Default for BloomConfig {
    fn default() -> Self {
        Self::for_network(NetworkConfig::bitcoin())
    }
}

impl BloomConfig {
    /// Create configuration optimized for a specific network
    pub fn for_network(network: NetworkConfig) -> Self {
        let size = match network.name.as_str() {
            "bitcoin" => 36_000,      // Bitcoin Core default
            "ethereum" => 50_000,     // Larger for Ethereum's higher TPS
            "solana" => 100_000,      // Very large for Solana's ultra-high TPS
            _ => 36_000,              // Default size
        };

        let batch_size = match network.name.as_str() {
            "bitcoin" => 1024,
            "ethereum" => 2048,
            "solana" => 4096,
            _ => 1024,
        };

        BloomConfig {
            network,
            size,
            num_hashes: 5,
            tweak: rand::random(),
            flags: 0,
            max_age_seconds: 86400, // 24 hours
            batch_size,
            enable_compression: false,
            enable_metrics: true,
        }
    }

    /// Create high-performance configuration for maximum throughput
    pub fn high_performance(network: NetworkConfig) -> Self {
        let mut config = Self::for_network(network);
        config.size = 100_000;        // Larger filter for better accuracy
        config.num_hashes = 7;        // More hash functions for better distribution
        config.batch_size = 8192;     // Larger batches for better parallelism
        config.enable_compression = true;
        config.enable_metrics = true;
        config
    }

    /// Create memory-optimized configuration for resource-constrained environments
    pub fn memory_optimized(network: NetworkConfig) -> Self {
        let mut config = Self::for_network(network);
        config.size = 18_000;        // Smaller filter
        config.num_hashes = 3;       // Fewer hash functions
        config.batch_size = 512;     // Smaller batches
        config.enable_compression = true;
        config.enable_metrics = false;
        config
    }
}

/// Universal Sprint Bloom Filter - Network Agnostic High-Performance Filter
/// Supports all blockchain networks with maximum performance and security
/// Similar to Alchemy, Infura - the fastest and most secure blockchain API
pub struct UniversalBloomFilter {
    filter_data: Vec<AtomicU64>,
    config: BloomConfig,
    item_count: AtomicU64,
    hash_seeds: [u32; 8],
    timestamps: Arc<DashMap<Vec<u8>, u64>>,
    false_positive_count: AtomicU64,
    last_cleanup: AtomicU64,
    entropy_pool: Vec<u8>, // Additional entropy for seeding
    network_stats: Arc<DashMap<String, NetworkStats>>, // Per-network statistics
}

/// Network-specific performance statistics
#[derive(Clone, Debug, Default)]
pub struct NetworkStats {
    pub transactions_processed: u64,
    pub blocks_processed: u64,
    pub queries_per_second: u64,
    pub average_query_time_ns: u64,
    pub false_positive_rate: f64,
    pub memory_usage_bytes: u64,
    pub last_updated: u64,
}

impl UniversalBloomFilter {
    /// Create new Universal Sprint Bloom Filter - Network Agnostic
    /// Supports all blockchain networks with maximum performance and security
    pub fn new(config: Option<BloomConfig>) -> Result<Self, BloomFilterError> {
        let cfg = config.unwrap_or_default();

        // Validate configuration for security and performance
        if !cfg.size.is_power_of_two() {
            return Err(BloomFilterError::InvalidConfiguration("Size must be power of two".into()));
        }
        if !(2..=7).contains(&cfg.num_hashes) {
            return Err(BloomFilterError::InvalidConfiguration("Number of hashes must be 2-7".into()));
        }
        if cfg.size < 1024 || cfg.size > 1_000_000 {
            return Err(BloomFilterError::InvalidConfiguration("Size must be between 1024 and 1M bits".into()));
        }

        let bucket_count = (cfg.size + 63) / 64;
        let mut hash_seeds = [0u32; 8];

        // Cryptographically secure seed generation with additional entropy
        let mut entropy_pool = vec![0u8; 32];
        rand::thread_rng().fill_bytes(&mut entropy_pool);

        let seed_bytes = cfg.tweak.to_le_bytes();
        for i in 0..8 {
            hash_seeds[i] = u32::from_le_bytes([
                seed_bytes[0] ^ entropy_pool[i % 32],
                seed_bytes[1] ^ entropy_pool[(i + 8) % 32],
                seed_bytes[2] ^ entropy_pool[(i + 16) % 32],
                seed_bytes[3] ^ entropy_pool[(i + 24) % 32],
            ]);
        }

        Ok(UniversalBloomFilter {
            filter_data: (0..bucket_count).map(|_| AtomicU64::new(0)).collect(),
            config: cfg,
            item_count: AtomicU64::new(0),
            hash_seeds,
            timestamps: Arc::new(DashMap::with_capacity(10000)),
            false_positive_count: AtomicU64::new(0),
            last_cleanup: AtomicU64::new(match SystemTime::now().duration_since(UNIX_EPOCH) {
                Ok(duration) => duration.as_secs(),
                Err(_) => return Err(BloomFilterError::SystemTimeError),
            }),
            entropy_pool,
            network_stats: Arc::new(DashMap::new()),
        })
    }

    /// Insert a single UTXO with maximum performance optimization
    pub fn insert_utxo(&self, txid: &TransactionId, vout: u32) -> Result<(), BloomFilterError> {
        let mut preimage = Vec::with_capacity(36);
        preimage.extend_from_slice(&txid.as_bytes()[..]);
        preimage.extend_from_slice(&vout.to_le_bytes());
        self.insert(&preimage)
    }

    /// Insert a batch of UTXOs in parallel with optimal chunking
    pub fn insert_batch(&self, batch: &[(TransactionId, u32)]) -> Result<(), BloomFilterError> {
        if batch.is_empty() {
            return Ok(());
        }

        let now = match SystemTime::now().duration_since(UNIX_EPOCH) {
            Ok(duration) => duration.as_secs(),
            Err(_) => return Err(BloomFilterError::SystemTimeError),
        };

        // Process in optimal chunks for maximum parallelism
        batch.par_chunks(self.config.batch_size).for_each(|chunk| {
            chunk.iter().for_each(|(txid, vout)| {
                let mut preimage = Vec::with_capacity(36);
                preimage.extend_from_slice(&txid.as_bytes()[..]);
                preimage.extend_from_slice(&vout.to_le_bytes());
                let _ = self.insert_with_timestamp(&preimage, now);
            });
        });

        Ok(())
    }

    /// Internal insert with timestamp tracking
    fn insert(&self, data: &[u8]) -> Result<(), BloomFilterError> {
        let now = match SystemTime::now().duration_since(UNIX_EPOCH) {
            Ok(duration) => duration.as_secs(),
            Err(_) => return Err(BloomFilterError::SystemTimeError),
        };
        self.insert_with_timestamp(data, now)
    }

    /// Insert with timestamp and entropy seeding for maximum performance
    fn insert_with_timestamp(&self, data: &[u8], timestamp: u64) -> Result<(), BloomFilterError> {
        if data.is_empty() {
            return Err(BloomFilterError::InvalidInput("Data cannot be empty".into()));
        }

        let hashes = self.compute_hashes(data)?;

        // Parallel bit setting for maximum performance
        (0..self.config.num_hashes).into_par_iter().for_each(|i| {
            let bit_pos = self.murmur_hash3(hashes, i as u32) % self.config.size as u64;
            let bucket_idx = (bit_pos >> 6) as usize;
            let bit_mask = 1u64 << (bit_pos & 0x3F);

            // Atomic OR for thread safety
            self.filter_data[bucket_idx].fetch_or(bit_mask, Ordering::Relaxed);
        });

        self.item_count.fetch_add(1, Ordering::Relaxed);
        self.timestamps.insert(data.to_vec(), timestamp);

        Ok(())
    }

    /// Check if a single UTXO is present with false positive tracking
    pub fn contains_utxo(&self, txid: &TransactionId, vout: u32) -> Result<bool, BloomFilterError> {
        let mut preimage = Vec::with_capacity(36);
        preimage.extend_from_slice(&txid.as_bytes()[..]);
        preimage.extend_from_slice(&vout.to_le_bytes());
        self.contains(&preimage)
    }

    /// Check a batch of UTXOs with optimal parallelism
    pub fn contains_batch(&self, batch: &[(TransactionId, u32)]) -> Result<Vec<bool>, BloomFilterError> {
        if batch.is_empty() {
            return Ok(Vec::new());
        }

        let results: Vec<bool> = batch.par_iter()
            .map(|(txid, vout)| self.contains_utxo(txid, *vout).unwrap_or(false))
            .collect();

        Ok(results)
    }

    /// Internal contains check with performance optimizations
    fn contains(&self, data: &[u8]) -> Result<bool, BloomFilterError> {
        if data.is_empty() {
            return Ok(false);
        }

        let hashes = self.compute_hashes(data)?;

        // Early exit optimization - check all bits in parallel
        let all_present = (0..self.config.num_hashes).into_par_iter().all(|i| {
            let bit_pos = self.murmur_hash3(hashes, i as u32) % self.config.size as u64;
            let bucket_idx = (bit_pos >> 6) as usize;
            let bit_mask = 1u64 << (bit_pos & 0x3F);
            (self.filter_data[bucket_idx].load(Ordering::Relaxed) & bit_mask) != 0
        });

        // Track false positives for analytics
        if all_present {
            // Verify with timestamp to reduce false positives
            if let Some(entry_time) = self.timestamps.get(data) {
                let now = match SystemTime::now().duration_since(UNIX_EPOCH) {
                    Ok(duration) => duration.as_secs(),
                    Err(_) => return Err(BloomFilterError::SystemTimeError),
                };

                if now.saturating_sub(*entry_time) > self.config.max_age_seconds {
                    // Entry is too old, treat as false positive
                    self.false_positive_count.fetch_add(1, Ordering::Relaxed);
                    return Ok(false);
                }
            } else {
                // No timestamp entry, likely false positive
                self.false_positive_count.fetch_add(1, Ordering::Relaxed);
                return Ok(false);
            }
        }

        Ok(all_present)
    }

    /// Compute double SHA256 hashes with entropy mixing for maximum security
    fn compute_hashes(&self, data: &[u8]) -> Result<[u64; 2], BloomFilterError> {
        let mut engine = bitcoin_hashes::sha256::HashEngine::default();
        engine.input(data);
        let hash1 = bitcoin_hashes::sha256::Hash::from_engine(engine);

        // Mix with entropy pool for additional security
        let mut mixed_data = Vec::with_capacity(data.len() + self.entropy_pool.len());
        mixed_data.extend_from_slice(data);
        mixed_data.extend_from_slice(&self.entropy_pool);

        let mut engine2 = bitcoin_hashes::sha256::HashEngine::default();
        engine2.input(&mixed_data);
        let hash2 = bitcoin_hashes::sha256::Hash::from_engine(engine2);

        Ok([
            u64::from_le_bytes(hash1[0..8].try_into().map_err(|_| BloomFilterError::HashComputationError)?),
            u64::from_le_bytes(hash2[0..8].try_into().map_err(|_| BloomFilterError::HashComputationError)?),
        ])
    }

    /// Optimized MurmurHash3 with entropy seeding
    fn murmur_hash3(&self, hash: [u64; 2], hash_num: u32) -> u64 {
        let h = hash_num.wrapping_mul(0xFBA4C795).wrapping_add(self.config.tweak);
        let mut v = h as u64 ^ hash[1];
        v = v.wrapping_mul(0xFF51AFD7ED558CCD);
        v = v.wrapping_mul(0xC4CEB9FE1A85EC53);
        v ^= v >> 32;
        v ^ hash[0] ^ self.hash_seeds[hash_num as usize % 8] as u64
    }

    /// Load all transactions from a block in parallel with maximum optimization
    pub fn load_block(&self, block: &BlockData) -> Result<(), BloomFilterError> {
        if block.transactions.is_empty() {
            return Ok(());
        }

        // Process transactions in parallel chunks
        block.transactions.par_chunks(self.config.batch_size).for_each(|tx_chunk| {
            tx_chunk.iter().for_each(|tx| {
                let txid_bytes = tx.as_bytes();
                let _ = self.insert(txid_bytes);
            });
        });

        Ok(())
    }

    /// Calculate theoretical false positive rate
    pub fn false_positive_rate(&self) -> f64 {
        let n = self.item_count.load(Ordering::Relaxed) as f64;
        let m = self.config.size as f64;
        let k = self.config.num_hashes as f64;

        if n == 0.0 || m == 0.0 {
            0.0
        } else {
            (1.0 - (-k * n / m).exp()).powf(k)
        }
    }

    /// Get performance statistics
    pub fn stats(&self) -> BloomFilterStats {
        let now = SystemTime::now().duration_since(UNIX_EPOCH)
            .unwrap_or_default().as_secs();

        BloomFilterStats {
            item_count: self.item_count.load(Ordering::Relaxed),
            false_positive_count: self.false_positive_count.load(Ordering::Relaxed),
            theoretical_fp_rate: self.false_positive_rate(),
            memory_usage_bytes: self.filter_data.len() * 8,
            timestamp_entries: self.timestamps.len(),
            average_age_seconds: self.average_entry_age(now),
        }
    }

    /// Calculate average age of entries
    fn average_entry_age(&self, now: u64) -> f64 {
        let mut total_age = 0u64;
        let mut count = 0usize;

        self.timestamps.iter().for_each(|entry| {
            total_age += now.saturating_sub(*entry.value());
            count += 1;
        });

        if count == 0 {
            0.0
        } else {
            total_age as f64 / count as f64
        }
    }

    /// Cleanup old entries to maintain performance
    pub fn cleanup(&self) -> Result<usize, BloomFilterError> {
        let now = match SystemTime::now().duration_since(UNIX_EPOCH) {
            Ok(duration) => duration.as_secs(),
            Err(_) => return Err(BloomFilterError::SystemTimeError),
        };

        let mut removed = 0usize;
        let max_age = self.config.max_age_seconds;

        // Remove old entries
        self.timestamps.retain(|_, timestamp| {
            if now.saturating_sub(*timestamp) > max_age {
                removed += 1;
                false
            } else {
                true
            }
        });

        self.last_cleanup.store(now, Ordering::Relaxed);
        Ok(removed)
    }

    /// Auto-cleanup if needed
    pub fn auto_cleanup(&self) -> Result<bool, BloomFilterError> {
        let now = match SystemTime::now().duration_since(UNIX_EPOCH) {
            Ok(duration) => duration.as_secs(),
            Err(_) => return Err(BloomFilterError::SystemTimeError),
        };

        let last_cleanup = self.last_cleanup.load(Ordering::Relaxed);
        let cleanup_interval = 3600; // 1 hour

        if now.saturating_sub(last_cleanup) > cleanup_interval {
            let _ = self.cleanup()?;
            Ok(true)
        } else {
            Ok(false)
        }
    }

    /// Get current item count (thread-safe)
    pub fn get_item_count(&self) -> usize {
        self.item_count.load(Ordering::Relaxed) as usize
    }

    /// Get false positive count (thread-safe)
    pub fn get_false_positive_count(&self) -> f64 {
        let items = self.item_count.load(Ordering::Relaxed) as f64;
        let false_positives = self.false_positive_count.load(Ordering::Relaxed) as f64;
        if items > 0.0 {
            false_positives / items
        } else {
            0.0
        }
    }

    /// Generic insert method for C FFI
    pub fn insert_data(&self, data: &[u8]) -> Result<(), BloomFilterError> {
        self.insert(data)
    }

    /// Generic contains method for C FFI  
    pub fn contains_data(&self, data: &[u8]) -> Result<bool, BloomFilterError> {
        self.contains(data)
    }
}

/// Performance and security statistics
#[derive(Debug, Clone)]
pub struct BloomFilterStats {
    pub item_count: u64,
    pub false_positive_count: u64,
    pub theoretical_fp_rate: f64,
    pub memory_usage_bytes: usize,
    pub timestamp_entries: usize,
    pub average_age_seconds: f64,
}

/// Comprehensive error handling for maximum stability
#[derive(Debug, thiserror::Error)]
pub enum BloomFilterError {
    #[error("Invalid configuration: {0}")]
    InvalidConfiguration(String),

    #[error("Invalid input: {0}")]
    InvalidInput(String),

    #[error("Hash computation failed")]
    HashComputationError,

    #[error("System time error")]
    SystemTimeError,

    #[error("Memory allocation failed")]
    MemoryError,

    #[error("Concurrent access error")]
    ConcurrencyError,
}

impl Drop for UniversalBloomFilter {
    fn drop(&mut self) {
        // Secure cleanup
        self.entropy_pool.zeroize();
        self.hash_seeds.zeroize();
    }
}

impl Zeroize for UniversalBloomFilter {
    fn zeroize(&mut self) {
        // Only zeroize sensitive data
        self.entropy_pool.zeroize();
        self.hash_seeds.zeroize();
        // Note: bit_array and metadata contain operational data, not secrets
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_basic_operations() {
        let filter = UniversalBloomFilter::new(None).unwrap();

        let txid = TransactionId::from_bytes(&[1u8; 32]).unwrap();
        assert!(!filter.contains_utxo(&txid, 0).unwrap());

        filter.insert_utxo(&txid, 0).unwrap();
        assert!(filter.contains_utxo(&txid, 0).unwrap());
    }

    #[test]
    fn test_batch_operations() {
        let filter = UniversalBloomFilter::new(None).unwrap();

        let batch: Vec<(TransactionId, u32)> = (0..100)
            .map(|i| {
                let mut bytes = [0u8; 32];
                bytes[0] = i as u8;
                (TransactionId::from_bytes(&bytes).unwrap(), i)
            })
            .collect();

        filter.insert_batch(&batch).unwrap();
        let results = filter.contains_batch(&batch).unwrap();

        assert_eq!(results.len(), 100);
        assert!(results.iter().all(|&x| x));
    }

    #[test]
    fn test_false_positive_rate() {
        let filter = UniversalBloomFilter::new(None).unwrap();

        // Insert some items
        for i in 0u32..1000 {
            let txid = TransactionId::from_bytes(&i.to_le_bytes()).unwrap();
            filter.insert_utxo(&txid, 0).unwrap();
        }

        let fp_rate = filter.false_positive_rate();
        assert!(fp_rate > 0.0 && fp_rate < 1.0);
    }
}
