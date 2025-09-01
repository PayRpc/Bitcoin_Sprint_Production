// SPDX-License-Identifier: MIT
// Universal Sprint - Simplified Storage Verification with Optional IPFS
// Enhanced Security, DoS Protection, and Network-Agnostic Design

use std::collections::{HashMap, HashSet};
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH, Duration};
use sha2::{Sha256, Digest};
use rand::{thread_rng, RngCore, Rng};

#[cfg(feature = "ipfs")]
use reqwest::Client;

use thiserror::Error;
use tokio::sync::RwLock;
use log::{info, warn, error, debug};
use hex;

/// Commitment algorithms for file verification
#[derive(Clone, Debug)]
pub enum CommitmentAlg {
    Sha256Chunks,
    MerkleSha256 { root: [u8; 32], chunk_size: u32 }
}

/// Commitment store for file integrity verification
#[derive(Clone, Default)]
pub struct CommitmentStore {
    // (file_id, chunk_index) -> leaf hash (sha256)
    leaves: HashMap<(String, u64), [u8; 32]>,
    meta: HashMap<String, (CommitmentAlg, u32, u64)>, // (alg, chunk_size, total_chunks)
    beacon_timestamps: HashMap<String, u64>, // beacon -> timestamp for cleanup
}

impl CommitmentStore {
    /// Register SHA256 chunks for a file
    pub fn register_sha256_chunks(
        &mut self,
        file_id: &str,
        chunk_size: u32,
        leaf_hashes: Vec<[u8; 32]>
    ) {
        let total = leaf_hashes.len() as u64;
        self.meta.insert(
            file_id.to_string(),
            (CommitmentAlg::Sha256Chunks, chunk_size, total)
        );
        for (i, h) in leaf_hashes.into_iter().enumerate() {
            self.leaves.insert((file_id.to_string(), i as u64), h);
        }
    }

    /// Register Merkle root for a file
    pub fn register_merkle_root(
        &mut self,
        file_id: &str,
        root: [u8; 32],
        chunk_size: u32,
        total_chunks: u64
    ) {
        self.meta.insert(
            file_id.to_string(),
            (CommitmentAlg::MerkleSha256 { root, chunk_size }, chunk_size, total_chunks)
        );
    }

    /// Get chunk metadata for a file
    pub fn get_chunk_meta(&self, file_id: &str) -> Option<(CommitmentAlg, u32, u64)> {
        self.meta.get(file_id).cloned()
    }

    /// Get expected leaf hash for a chunk
    pub fn expected_leaf(&self, file_id: &str, chunk_index: u64) -> Option<[u8; 32]> {
        self.leaves.get(&(file_id.to_string(), chunk_index)).copied()
    }

    /// Store beacon timestamp for cleanup
    pub fn store_beacon_timestamp(&mut self, beacon: &str, timestamp: u64) {
        self.beacon_timestamps.insert(beacon.to_string(), timestamp);
    }

    /// Get beacon timestamp
    pub fn get_beacon_timestamp(&self, beacon: &str) -> Option<u64> {
        self.beacon_timestamps.get(beacon).copied()
    }

    /// Cleanup old beacons
    pub fn cleanup_old_beacons(&mut self, max_age_secs: u64) {
        let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs();
        self.beacon_timestamps.retain(|_, ts| now - *ts < max_age_secs);
    }
}

/// Storage challenge with enhanced cryptographic security
#[derive(Debug, Clone)]
pub struct StorageChallenge {
    pub id: String,
    pub file_id: String,
    pub provider: String,
    pub nonce: u64,
    pub timestamp: u64,
    pub expiry: u64,
    pub beacon: String,
    pub difficulty: u8, // Challenge difficulty level
    pub challenge_data: Vec<u8>, // Specific data to prove possession of
    pub sample_offset: u64, // Offset in file to sample
    pub sample_size: u32, // Size of sample to retrieve
    pub chunk_index: u64, // Which chunk to verify
    pub commitment_alg: String, // "sha256_chunks" or "merkle_sha256"
}

/// Storage proof with cryptographic verification data
#[derive(Debug, Clone)]
pub struct StorageProof {
    pub challenge_id: String,
    pub file_id: String,
    pub provider: String,
    pub timestamp: u64,
    pub proof_data: Vec<u8>, // Actual data sample from storage
    pub merkle_proof: Option<Vec<String>>, // Optional Merkle tree proof
    pub signature: Option<String>, // Optional provider signature
}

/// Verification metrics for monitoring and analytics
#[derive(Debug, Clone, Default)]
pub struct VerificationMetrics {
    pub total_challenges: u64,
    pub successful_proofs: u64,
    pub failed_proofs: u64,
    pub expired_challenges: u64,
    pub rate_limited_requests: u64,
    pub average_response_time_ms: f64,
    pub last_reset: u64,
}

impl VerificationMetrics {
    pub fn success_rate(&self) -> f64 {
        if self.total_challenges == 0 {
            return 0.0;
        }
        self.successful_proofs as f64 / self.total_challenges as f64
    }

    pub fn reset_if_needed(&mut self, now: u64) {
        // Reset metrics daily
        if now - self.last_reset > 86400 {
            *self = Self {
                last_reset: now,
                ..Default::default()
            };
        }
    }
}

/// Enhanced error types for better debugging
#[derive(Debug, thiserror::Error)]
pub enum StorageVerificationError {
    #[error("Rate limit exceeded: {limit} requests per {window}")]
    RateLimitExceeded { limit: u32, window: String },
    
    #[error("Invalid input: {field} - {reason}")]
    InvalidInput { field: String, reason: String },
    
    #[error("Challenge not found: {challenge_id}")]
    ChallengeNotFound { challenge_id: String },
    
    #[error("Cryptographic verification failed: {reason}")]
    CryptographicFailure { reason: String },
    
    #[error("Network error: {source}")]
    NetworkError { 
        #[source]
        source: Box<dyn std::error::Error + Send + Sync> 
    },
    
    #[error("Timeout exceeded: {timeout_ms}ms")]
    TimeoutExceeded { timeout_ms: u64 },
    
    #[error("Provider authentication failed")]
    AuthenticationFailed,
}
/// Rate limiting configuration
#[derive(Debug, Clone)]
pub struct RateLimitConfig {
    pub max_requests_per_minute: u32,
    pub max_requests_per_hour: u32,
    pub cleanup_interval_secs: u64,
}

impl Default for RateLimitConfig {
    fn default() -> Self {
        Self {
            max_requests_per_minute: 60,
            max_requests_per_hour: 1000,
            cleanup_interval_secs: 300, // 5 minutes
        }
    }
}

/// Request tracking for DoS protection
#[derive(Debug, Clone)]
struct RequestTracker {
    minute_requests: Vec<u64>,
    hour_requests: Vec<u64>,
    last_cleanup: u64,
}

impl RequestTracker {
    fn new() -> Self {
        Self {
            minute_requests: Vec::new(),
            hour_requests: Vec::new(),
            last_cleanup: SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
        }
    }

    fn cleanup(&mut self, now: u64) {
        // Remove old requests
        self.minute_requests.retain(|&ts| now - ts < 60);
        self.hour_requests.retain(|&ts| now - ts < 3600);
        self.last_cleanup = now;
    }

    fn can_make_request(&mut self, now: u64, config: &RateLimitConfig) -> bool {
        // Auto-cleanup if needed
        if now - self.last_cleanup > config.cleanup_interval_secs {
            self.cleanup(now);
        }

        self.minute_requests.len() < config.max_requests_per_minute as usize &&
        self.hour_requests.len() < config.max_requests_per_hour as usize
    }

    fn record_request(&mut self, now: u64) {
        self.minute_requests.push(now);
        self.hour_requests.push(now);
    }
}

/// Enhanced storage verifier with cryptographic proofs and monitoring
pub struct StorageVerifier {
    challenges: Arc<tokio::sync::Mutex<HashMap<String, StorageChallenge>>>,
    used_beacons: Arc<tokio::sync::Mutex<HashSet<String>>>,
    request_trackers: Arc<tokio::sync::Mutex<HashMap<String, RequestTracker>>>,
    metrics: Arc<tokio::sync::Mutex<VerificationMetrics>>,
    commitments: Arc<tokio::sync::Mutex<CommitmentStore>>,
    rate_limit_config: RateLimitConfig,
    #[cfg(feature = "ipfs")]
    http_client: Option<Client>,
}

impl StorageVerifier {
    /// Create new verifier with default rate limiting
    pub fn new() -> Self {
        Self::with_config(RateLimitConfig::default())
    }

    /// Create new verifier with custom rate limiting
    pub fn with_config(config: RateLimitConfig) -> Self {
        Self {
            challenges: Arc::new(tokio::sync::Mutex::new(HashMap::new())),
            used_beacons: Arc::new(tokio::sync::Mutex::new(HashSet::new())),
            request_trackers: Arc::new(tokio::sync::Mutex::new(HashMap::new())),
            metrics: Arc::new(tokio::sync::Mutex::new(VerificationMetrics::default())),
            commitments: Arc::new(tokio::sync::Mutex::new(CommitmentStore::default())),
            rate_limit_config: config,
            #[cfg(feature = "ipfs")]
            http_client: Some(Client::builder()
                .timeout(Duration::from_secs(10))
                .user_agent("UniversalSprint/1.0")
                .build()
                .unwrap_or_else(|_| Client::new())
            ),
        }
    }

    /// Generate secure storage challenge with cryptographic requirements
    pub async fn generate_challenge(&self, file_id: &str, provider: &str) -> Result<StorageChallenge, StorageVerificationError> {
        let start_time = SystemTime::now();
        let now = start_time.duration_since(UNIX_EPOCH).unwrap().as_secs();

        // Input validation
        if file_id.is_empty() || provider.is_empty() {
            return Err(StorageVerificationError::InvalidInput {
                field: "file_id or provider".to_string(),
                reason: "Cannot be empty".to_string(),
            });
        }
        if file_id.len() > 256 || provider.len() > 64 {
            return Err(StorageVerificationError::InvalidInput {
                field: "file_id or provider".to_string(),
                reason: "Too long".to_string(),
            });
        }

        // Check if file has commitments registered
        let (alg, chunk_size, total_chunks) = {
            let commitments = self.commitments.lock().await;
            commitments.get_chunk_meta(file_id).ok_or_else(|| StorageVerificationError::InvalidInput {
                field: "file_id".to_string(),
                reason: "No commitment registered for file_id. Register file commitments first.".to_string(),
            })?
        };

        // Rate limiting check
        {
            let mut trackers = self.request_trackers.lock().await;
            let tracker = trackers.entry(provider.to_string()).or_insert_with(RequestTracker::new);

            if !tracker.can_make_request(now, &self.rate_limit_config) {
                let mut metrics = self.metrics.lock().await;
                metrics.rate_limited_requests += 1;
                return Err(StorageVerificationError::RateLimitExceeded {
                    limit: self.rate_limit_config.max_requests_per_minute,
                    window: "minute".to_string(),
                });
            }
            tracker.record_request(now);
        }

        // Generate cryptographic challenge
        let mut rng = thread_rng();
        let random_salt: u64 = rng.gen();
        let chunk_index = rng.gen_range(0..total_chunks);
        let sample_offset = (chunk_index as u64) * (chunk_size as u64);
        let sample_size = chunk_size;

        // Generate challenge data that must be included in proof
        let mut challenge_data = vec![0u8; 32];
        rng.fill_bytes(&mut challenge_data);

        let beacon = self.generate_beacon(file_id, provider, now, random_salt)?;

        // Replay protection
        {
            let mut used = self.used_beacons.lock().await;
            if used.contains(&beacon) {
                return Err(StorageVerificationError::CryptographicFailure {
                    reason: "Beacon collision detected".to_string(),
                });
            }
            used.insert(beacon.clone());

            // Store beacon timestamp for cleanup
            let mut commitments = self.commitments.lock().await;
            commitments.store_beacon_timestamp(&beacon, now);

            // Cleanup old beacons periodically
            if used.len() > 10000 {
                commitments.cleanup_old_beacons(3600); // 1 hour
                used.retain(|b| {
                    if let Some(ts) = commitments.get_beacon_timestamp(b) {
                        now - ts < 3600
                    } else {
                        false
                    }
                });
            }
        }

        let difficulty = self.calculate_difficulty(provider).await;
        let commitment_alg = match alg {
            CommitmentAlg::Sha256Chunks => "sha256_chunks".to_string(),
            CommitmentAlg::MerkleSha256 { .. } => "merkle_sha256".to_string(),
        };

        let challenge = StorageChallenge {
            id: format!("chall_{}_{:x}", &file_id[..std::cmp::min(file_id.len(), 8)], now),
            file_id: file_id.to_string(),
            provider: provider.to_string(),
            nonce: random_salt,
            timestamp: now,
            expiry: now + 1800, // 30 minutes expiry
            beacon,
            difficulty,
            challenge_data,
            sample_offset,
            sample_size,
            chunk_index,
            commitment_alg,
        };

        // Store challenge with automatic cleanup
        {
            let mut challenges = self.challenges.lock().await;
            challenges.insert(challenge.id.clone(), challenge.clone());

            // Cleanup expired challenges
            if challenges.len() > 1000 {
                challenges.retain(|_, c| now < c.expiry);
            }
        }

        // Update metrics
        {
            let mut metrics = self.metrics.lock().await;
            metrics.reset_if_needed(now);
            metrics.total_challenges += 1;
        }

        log::info!("Generated challenge {} for provider {} file {} chunk {}", 
                   challenge.id, provider, file_id, chunk_index);

        Ok(challenge)
    }

    /// Verify storage proof with enhanced cryptographic verification
    pub async fn verify_proof(&self, proof: StorageProof) -> Result<bool, StorageVerificationError> {
        let start_time = SystemTime::now();
        let now = start_time.duration_since(UNIX_EPOCH).unwrap().as_secs();

        // Input validation
        if proof.challenge_id.is_empty() || proof.file_id.is_empty() || proof.provider.is_empty() {
            return Err(StorageVerificationError::InvalidInput {
                field: "proof fields".to_string(),
                reason: "Cannot be empty".to_string(),
            });
        }

        let challenges = self.challenges.lock().await;
        let challenge = challenges.get(&proof.challenge_id)
            .ok_or_else(|| StorageVerificationError::ChallengeNotFound {
                challenge_id: proof.challenge_id.clone(),
            })?;

        // Basic metadata verification
        if proof.file_id != challenge.file_id || proof.provider != challenge.provider {
            let mut metrics = self.metrics.lock().await;
            metrics.failed_proofs += 1;
            return Ok(false);
        }

        // Expiry check
        if now > challenge.expiry {
            let mut metrics = self.metrics.lock().await;
            metrics.expired_challenges += 1;
            return Ok(false);
        }

        // Timestamp validation (allow some clock skew)
        if proof.timestamp < challenge.timestamp || proof.timestamp > now + 300 {
            return Err(StorageVerificationError::CryptographicFailure {
                reason: "Invalid proof timestamp".to_string(),
            });
        }

        // Cryptographic proof verification
        let is_valid = self.verify_cryptographic_proof(&proof, challenge).await?;

        // Update metrics
        {
            let mut metrics = self.metrics.lock().await;
            let elapsed = start_time.elapsed().unwrap_or_default().as_millis() as f64;

            // Use Exponential Moving Average for response time
            let alpha = 0.2; // Smoothing factor
            metrics.average_response_time_ms = if metrics.average_response_time_ms == 0.0 {
                elapsed
            } else {
                alpha * elapsed + (1.0 - alpha) * metrics.average_response_time_ms
            };

            if is_valid {
                metrics.successful_proofs += 1;
                log::info!("Proof verified successfully: {} for provider {}",
                          proof.challenge_id, proof.provider);
            } else {
                metrics.failed_proofs += 1;
                log::warn!("Proof verification failed: {} for provider {}",
                          proof.challenge_id, proof.provider);
            }
        }

        Ok(is_valid)
    }

    /// Perform cryptographic verification of the storage proof
    async fn verify_cryptographic_proof(&self, proof: &StorageProof, challenge: &StorageChallenge) -> Result<bool, StorageVerificationError> {
        // Verify proof data is not empty
        if proof.proof_data.is_empty() {
            return Err(StorageVerificationError::CryptographicFailure {
                reason: "Proof data cannot be empty".to_string(),
            });
        }

        // Verify proof data size matches expected sample size
        if proof.proof_data.len() != challenge.sample_size as usize {
            return Err(StorageVerificationError::CryptographicFailure {
                reason: format!("Proof data size {} does not match expected {}",
                               proof.proof_data.len(), challenge.sample_size),
            });
        }

        // Compute leaf hash of the returned chunk
        let mut hasher = Sha256::new();
        hasher.update(&proof.proof_data);
        let computed_leaf = hasher.finalize();

        // Get expected leaf hash from commitments
        let expected_leaf = {
            let commitments = self.commitments.lock().await;
            commitments.expected_leaf(&challenge.file_id, challenge.chunk_index)
                .ok_or_else(|| StorageVerificationError::CryptographicFailure {
                    reason: format!("Missing chunk commitment for file {} chunk {}",
                                   challenge.file_id, challenge.chunk_index),
                })?
        };

        // Compare computed leaf with expected leaf
        if computed_leaf.as_slice() != expected_leaf {
            log::debug!("Leaf hash mismatch for file {} chunk {}: computed={}, expected={}",
                       challenge.file_id, challenge.chunk_index,
                       hex::encode(computed_leaf), hex::encode(expected_leaf));
            return Ok(false);
        }

        // Optional: Verify Merkle proof if provided and algorithm supports it
        if let Some(ref merkle_proof) = proof.merkle_proof {
            if challenge.commitment_alg == "merkle_sha256" {
                if !self.verify_merkle_proof(merkle_proof, &proof.proof_data, &challenge.file_id).await? {
                    return Ok(false);
                }
            }
        }

        // Optional: Verify provider signature if provided
        if let Some(ref signature) = proof.signature {
            if !self.verify_provider_signature(signature, &proof.proof_data, &proof.provider)? {
                return Ok(false);
            }
        }

        Ok(true)
    }

    /// Register file commitments for verification
    pub async fn register_file_commitments(
        &self,
        file_id: &str,
        chunk_size: u32,
        leaf_hashes: Vec<[u8; 32]>
    ) -> Result<(), StorageVerificationError> {
        if file_id.is_empty() {
            return Err(StorageVerificationError::InvalidInput {
                field: "file_id".to_string(),
                reason: "Cannot be empty".to_string(),
            });
        }
        if leaf_hashes.is_empty() {
            return Err(StorageVerificationError::InvalidInput {
                field: "leaf_hashes".to_string(),
                reason: "Cannot be empty".to_string(),
            });
        }

        let mut commitments = self.commitments.lock().await;
        let leaf_count = leaf_hashes.len();
        commitments.register_sha256_chunks(file_id, chunk_size, leaf_hashes);

        log::info!("Registered {} chunks for file {}", leaf_count, file_id);
        Ok(())
    }

    /// Register Merkle root for a file
    pub async fn register_merkle_root(
        &self,
        file_id: &str,
        root: [u8; 32],
        chunk_size: u32,
        total_chunks: u64
    ) -> Result<(), StorageVerificationError> {
        if file_id.is_empty() {
            return Err(StorageVerificationError::InvalidInput {
                field: "file_id".to_string(),
                reason: "Cannot be empty".to_string(),
            });
        }

        let mut commitments = self.commitments.lock().await;
        commitments.register_merkle_root(file_id, root, chunk_size, total_chunks);

        log::info!("Registered Merkle root for file {} with {} chunks", file_id, total_chunks);
        Ok(())
    }

    /// Verify Merkle proof for file integrity
    async fn verify_merkle_proof(&self, merkle_proof: &[String], proof_data: &[u8], file_id: &str) -> Result<bool, StorageVerificationError> {
        // Get the stored Merkle root for this file
        let commitments = self.commitments.lock().await;
        let (alg, _chunk_size, _total_chunks) = match commitments.get_chunk_meta(file_id) {
            Some(meta) => meta,
            None => {
                log::debug!("No commitment metadata found for file {}", file_id);
                return Ok(false);
            }
        };

        let stored_root = match alg {
            CommitmentAlg::MerkleSha256 { root, .. } => root,
            _ => {
                log::debug!("File {} does not use Merkle tree commitment", file_id);
                return Ok(false);
            }
        };

        // Compute leaf hash from proof data
        let mut hasher = Sha256::new();
        hasher.update(proof_data);
        let mut leaf_hash = [0u8; 32];
        leaf_hash.copy_from_slice(&hasher.finalize());

        // Verify the Merkle proof
        let mut current_hash = leaf_hash;

        for proof_element in merkle_proof {
            // Decode hex proof element
            let proof_bytes = match hex::decode(proof_element.trim_start_matches("0x")) {
                Ok(bytes) => bytes,
                Err(_) => {
                    log::debug!("Invalid hex in Merkle proof element: {}", proof_element);
                    return Ok(false);
                }
            };

            if proof_bytes.len() != 32 {
                log::debug!("Invalid proof element length: {} bytes", proof_bytes.len());
                return Ok(false);
            }

            let mut sibling_hash = [0u8; 32];
            sibling_hash.copy_from_slice(&proof_bytes);

            // Hash current with sibling (order depends on tree structure)
            // For simplicity, we'll assume left-to-right ordering
            let mut combined_hasher = Sha256::new();
            combined_hasher.update(&current_hash);
            combined_hasher.update(&sibling_hash);
            current_hash = combined_hasher.finalize().into();
        }

        // Compare final hash with stored root
        if current_hash != stored_root {
            log::debug!("Merkle proof verification failed for file {}: computed={}, expected={}",
                       file_id, hex::encode(current_hash), hex::encode(stored_root));
            return Ok(false);
        }

        log::debug!("Merkle proof verification successful for file {}", file_id);
        Ok(true)
    }

    /// Verify provider signature for authentication
    fn verify_provider_signature(&self, _signature: &str, _proof_data: &[u8], _provider: &str) -> Result<bool, StorageVerificationError> {
        // Placeholder for digital signature verification
        // In production, this would verify the provider's signature
        log::debug!("Provider signature verification not yet implemented");
        Ok(true)
    }

    /// Get current verification metrics
    pub async fn get_metrics(&self) -> VerificationMetrics {
        let metrics = self.metrics.lock().await;
        metrics.clone()
    }

    /// Reset metrics (useful for testing or periodic resets)
    pub async fn reset_metrics(&self) {
        let mut metrics = self.metrics.lock().await;
        let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs();
        *metrics = VerificationMetrics {
            last_reset: now,
            ..Default::default()
        };
    }
    /// Generate secure beacon with enhanced entropy
    fn generate_beacon(&self, file_id: &str, provider: &str, timestamp: u64, salt: u64) -> Result<String, StorageVerificationError> {
        let mut hasher = Sha256::new();
        hasher.update(file_id.as_bytes());
        hasher.update(provider.as_bytes());
        hasher.update(timestamp.to_le_bytes());
        hasher.update(salt.to_le_bytes());
        hasher.update(b"UniversalSprint"); // Domain separator
        
        Ok(hex::encode(hasher.finalize()))
    }

    /// Calculate challenge difficulty based on provider history
    async fn calculate_difficulty(&self, _provider: &str) -> u8 {
        // Simple static difficulty for now
        // In production, this could be dynamic based on provider reputation
        1
    }

    /// Cleanup expired data
    pub async fn cleanup_expired(&self) {
        let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs();

        // Cleanup challenges
        {
            let mut challenges = self.challenges.lock().await;
            challenges.retain(|_, c| now < c.expiry);
        }

        // Cleanup beacons and beacon timestamps
        {
            let mut beacons = self.used_beacons.lock().await;
            let mut commitments = self.commitments.lock().await;

            if beacons.len() > 5000 {
                commitments.cleanup_old_beacons(3600); // 1 hour
                beacons.retain(|b| {
                    if let Some(ts) = commitments.get_beacon_timestamp(b) {
                        now - ts < 3600
                    } else {
                        false
                    }
                });
            }
        }

        // Cleanup request trackers
        {
            let mut trackers = self.request_trackers.lock().await;
            for tracker in trackers.values_mut() {
                tracker.cleanup(now);
            }
        }
    }
}

// Optional IPFS functionality
#[cfg(feature = "ipfs")]
impl StorageVerifier {
    /// Fetch sample from IPFS with enhanced security
    pub async fn fetch_ipfs_sample(&self, cid: &str, max_size: usize) -> Result<Vec<u8>, StorageVerificationError> {
        // Input validation
        if cid.is_empty() || cid.len() > 128 {
            return Err(StorageVerificationError::InvalidInput {
                field: "cid".to_string(),
                reason: "Invalid CID format".to_string(),
            });
        }
        
        let safe_size = std::cmp::min(max_size, 8192); // Max 8KB sample
        
        let client = self.http_client.as_ref()
            .ok_or_else(|| StorageVerificationError::NetworkError {
                source: "HTTP client not available".to_string().into(),
            })?;

        // Use multiple IPFS gateways for redundancy
        let gateways = [
            "https://ipfs.io/ipfs",
            "https://cloudflare-ipfs.com/ipfs",
            "https://gateway.pinata.cloud/ipfs",
        ];

        for gateway in &gateways {
            let url = format!("{}/{}?format=raw", gateway, cid);
            
            match self.try_fetch_from_gateway(&client, &url, safe_size).await {
                Ok(data) => return Ok(data),
                Err(e) => {
                    log::warn!("Failed to fetch from {}: {:?}", gateway, e);
                    continue;
                }
            }
        }

        Err(StorageVerificationError::NetworkError {
            source: "Failed to fetch from all IPFS gateways".to_string().into(),
        })
    }

    async fn try_fetch_from_gateway(&self, client: &Client, url: &str, size: usize) -> Result<Vec<u8>, StorageVerificationError> {
        let resp = client
            .get(url)
            .header("Range", format!("bytes=0-{}", size - 1))
            .send()
            .await
            .map_err(|e| StorageVerificationError::NetworkError {
                source: format!("HTTP error: {}", e).into()
            })?;        if !resp.status().is_success() {
            return Err(StorageVerificationError::NetworkError {
                source: format!("HTTP {}", resp.status()).into(),
            });
        }

        let bytes = resp
            .bytes()
            .await
            .map_err(|e| StorageVerificationError::NetworkError {
                source: format!("Failed to read response: {}", e).into(),
            })?;

        if bytes.len() > size {
            return Err(StorageVerificationError::InvalidInput {
                field: "response_size".to_string(),
                reason: "Response too large".to_string(),
            });
        }

        Ok(bytes.to_vec())
    }

    /// Verify IPFS content with comprehensive cryptographic checks
    pub async fn verify_ipfs_content(&self, cid: &str, provider: &str, sample_size: Option<usize>) -> Result<bool, StorageVerificationError> {
        let challenge = self.generate_challenge(cid, provider).await?;
        let requested_size = sample_size.unwrap_or(challenge.sample_size as usize);

        // Fetch sample with timeout
        let sample = self.fetch_ipfs_sample(cid, requested_size).await
            .map_err(|e| StorageVerificationError::NetworkError { source: Box::new(e) })?;

        if sample.is_empty() {
            return Ok(false);
        }

        // Verify sample size matches challenge requirements
        if sample.len() != challenge.sample_size as usize {
            return Err(StorageVerificationError::CryptographicFailure {
                reason: format!("Sample size mismatch: got {}, expected {}",
                               sample.len(), challenge.sample_size),
            });
        }

        let proof = StorageProof {
            challenge_id: challenge.id.clone(),
            file_id: cid.to_string(),
            provider: provider.to_string(),
            timestamp: SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
            proof_data: sample,
            merkle_proof: None, // Could be implemented for additional verification
            signature: None,    // Could be implemented for provider authentication
        };

        self.verify_proof(proof).await
    }

    /// Ingest IPFS content and register commitments for future verification
    pub async fn ingest_ipfs_and_register(
        &self,
        cid: &str,
        chunk_size: usize
    ) -> Result<(), StorageVerificationError> {
        // Input validation
        if cid.is_empty() || cid.len() > 128 {
            return Err(StorageVerificationError::InvalidInput {
                field: "cid".to_string(),
                reason: "Invalid CID format".to_string(),
            });
        }

        let client = self.http_client.as_ref()
            .ok_or_else(|| StorageVerificationError::NetworkError {
                source: "HTTP client not available".to_string().into(),
            })?;

        // Fetch the entire file to compute chunk hashes
        let gateways = [
            "https://ipfs.io/ipfs",
            "https://cloudflare-ipfs.com/ipfs",
            "https://gateway.pinata.cloud/ipfs",
        ];

        let mut file_data = None;
        for gateway in &gateways {
            let url = format!("{}/{}", gateway, cid);

            match client
                .get(&url)
                .header("Range", "bytes=0-10485760") // Max 10MB for demo
                .send()
                .await
            {
                Ok(resp) if resp.status().is_success() => {
                    match resp.bytes().await {
                        Ok(bytes) => {
                            file_data = Some(bytes.to_vec());
                            break;
                        }
                        Err(e) => {
                            log::warn!("Failed to read response from {}: {:?}", gateway, e);
                            continue;
                        }
                    }
                }
                Ok(resp) => {
                    log::warn!("HTTP error from {}: {}", gateway, resp.status());
                    continue;
                }
                Err(e) => {
                    log::warn!("Failed to fetch from {}: {:?}", gateway, e);
                    continue;
                }
            }
        }

        let file_data = file_data.ok_or_else(|| StorageVerificationError::NetworkError {
            source: "Failed to fetch file from all IPFS gateways".to_string().into(),
        })?;

        if file_data.is_empty() {
            return Err(StorageVerificationError::InvalidInput {
                field: "file_data".to_string(),
                reason: "File is empty".to_string(),
            });
        }

        // Compute SHA256 hashes for each chunk
        let mut leaf_hashes = Vec::new();
        let mut offset = 0;

        while offset < file_data.len() {
            let end = std::cmp::min(offset + chunk_size, file_data.len());
            let chunk = &file_data[offset..end];

            let mut hasher = Sha256::new();
            hasher.update(chunk);
            let hash = hasher.finalize();
            leaf_hashes.push(hash.into());

            offset = end;
        }

        if leaf_hashes.is_empty() {
            return Err(StorageVerificationError::InvalidInput {
                field: "file_data".to_string(),
                reason: "No chunks generated".to_string(),
            });
        }

        // Register the commitments
        let leaf_count = leaf_hashes.len();
        self.register_file_commitments(cid, chunk_size as u32, leaf_hashes).await?;

        log::info!("Ingested IPFS file {} with {} chunks of size {}", cid, leaf_count, chunk_size);
        Ok(())
    }
}

impl Default for StorageVerifier {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_challenge_generation() {
        let verifier = StorageVerifier::new();

        // Register commitments for the test file first
        let test_data = b"Hello, World!";
        let mut hasher = Sha256::new();
        hasher.update(test_data);
        let leaf_hash = hasher.finalize();

        verifier.register_file_commitments("test_file", test_data.len() as u32, vec![leaf_hash.into()]).await.unwrap();

        let result = verifier.generate_challenge("test_file", "test_provider").await;
        assert!(result.is_ok());

        let challenge = result.unwrap();
        assert_eq!(challenge.file_id, "test_file");
        assert_eq!(challenge.provider, "test_provider");
        assert!(challenge.expiry > challenge.timestamp);
        assert!(!challenge.challenge_data.is_empty());
        assert_eq!(challenge.commitment_alg, "sha256_chunks");
    }

    #[tokio::test]
    async fn test_cryptographic_proof_verification() {
        let verifier = StorageVerifier::new();

        // Register commitments for a test file
        let test_data = b"Hello, World! This is test data for verification.";
        let chunk_size = 16;
        let mut leaf_hashes = Vec::new();

        let mut offset = 0;
        while offset < test_data.len() {
            let end = std::cmp::min(offset + chunk_size, test_data.len());
            let chunk = &test_data[offset..end];

            let mut hasher = Sha256::new();
            hasher.update(chunk);
            let hash = hasher.finalize();
            leaf_hashes.push(hash.into());

            offset = end;
        }

        verifier.register_file_commitments("test_file", chunk_size as u32, leaf_hashes).await.unwrap();

        let challenge = verifier.generate_challenge("test_file", "test_provider").await.unwrap();

        // Create proof with correct data from the committed chunk
        let chunk_index = challenge.chunk_index as usize;
        let chunk_start = chunk_index * chunk_size;
        let chunk_end = std::cmp::min(chunk_start + chunk_size, test_data.len());
        let proof_data = test_data[chunk_start..chunk_end].to_vec();

        let proof = StorageProof {
            challenge_id: challenge.id.clone(),
            file_id: "test_file".to_string(),
            provider: "test_provider".to_string(),
            timestamp: challenge.timestamp + 10,
            proof_data,
            merkle_proof: None,
            signature: None,
        };

        // This should now succeed because we have the correct proof data
        let result = verifier.verify_proof(proof).await;
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), true);
    }

    #[tokio::test]
    async fn test_rate_limiting_with_metrics() {
        let config = RateLimitConfig {
            max_requests_per_minute: 2,
            max_requests_per_hour: 10,
            cleanup_interval_secs: 1,
        };
        let verifier = StorageVerifier::with_config(config);

        // Register commitments for test files
        let test_data = b"test data";
        let chunk_size = 4;
        let mut leaf_hashes = Vec::new();

        let mut offset = 0;
        while offset < test_data.len() {
            let end = std::cmp::min(offset + chunk_size, test_data.len());
            let chunk = &test_data[offset..end];

            let mut hasher = Sha256::new();
            hasher.update(chunk);
            let hash = hasher.finalize();
            leaf_hashes.push(hash.into());

            offset = end;
        }

        verifier.register_file_commitments("file1", chunk_size as u32, leaf_hashes.clone()).await.unwrap();
        verifier.register_file_commitments("file2", chunk_size as u32, leaf_hashes.clone()).await.unwrap();
        verifier.register_file_commitments("file3", chunk_size as u32, leaf_hashes).await.unwrap();

        // First two requests should succeed
        assert!(verifier.generate_challenge("file1", "provider1").await.is_ok());
        assert!(verifier.generate_challenge("file2", "provider1").await.is_ok());

        // Third request should fail due to rate limiting
        let result = verifier.generate_challenge("file3", "provider1").await;
        assert!(result.is_err());

        // Check metrics - only successful challenges should be counted
        let metrics = verifier.get_metrics().await;
        assert_eq!(metrics.total_challenges, 2); // Only successful ones
        assert_eq!(metrics.rate_limited_requests, 1); // Failed one due to rate limiting
    }

    #[tokio::test]
    async fn test_beacon_uniqueness() {
        let verifier = StorageVerifier::new();

        // Register commitments for the test file
        let test_data = b"test data";
        let chunk_size = 4;
        let mut leaf_hashes = Vec::new();

        let mut offset = 0;
        while offset < test_data.len() {
            let end = std::cmp::min(offset + chunk_size, test_data.len());
            let chunk = &test_data[offset..end];

            let mut hasher = Sha256::new();
            hasher.update(chunk);
            let hash = hasher.finalize();
            leaf_hashes.push(hash.into());

            offset = end;
        }

        verifier.register_file_commitments("test_file", chunk_size as u32, leaf_hashes).await.unwrap();

        let challenge1 = verifier.generate_challenge("test_file", "provider1").await.unwrap();
        let challenge2 = verifier.generate_challenge("test_file", "provider1").await.unwrap();

        // Beacons should be different due to randomness
        assert_ne!(challenge1.beacon, challenge2.beacon);
        // Nonces should also be different
        assert_ne!(challenge1.nonce, challenge2.nonce);
    }    #[tokio::test]
    async fn test_metrics_tracking() {
        let verifier = StorageVerifier::new();

        // Register commitments for test files
        let test_data = b"test data";
        let chunk_size = 4;
        let mut leaf_hashes = Vec::new();

        let mut offset = 0;
        while offset < test_data.len() {
            let end = std::cmp::min(offset + chunk_size, test_data.len());
            let chunk = &test_data[offset..end];

            let mut hasher = Sha256::new();
            hasher.update(chunk);
            let hash = hasher.finalize();
            leaf_hashes.push(hash.into());

            offset = end;
        }

        verifier.register_file_commitments("file1", chunk_size as u32, leaf_hashes.clone()).await.unwrap();
        verifier.register_file_commitments("file2", chunk_size as u32, leaf_hashes).await.unwrap();

        // Check initial metrics
        let initial_metrics = verifier.get_metrics().await;
        assert_eq!(initial_metrics.total_challenges, 0);

        // Generate some challenges
        let _challenge1 = verifier.generate_challenge("file1", "provider1").await.unwrap();
        let mid_metrics = verifier.get_metrics().await;
        assert_eq!(mid_metrics.total_challenges, 1);

        let _challenge2 = verifier.generate_challenge("file2", "provider2").await.unwrap();

        let final_metrics = verifier.get_metrics().await;
        assert_eq!(final_metrics.total_challenges, 2);
        assert_eq!(final_metrics.successful_proofs, 0);
        assert_eq!(final_metrics.failed_proofs, 0);

        // Reset metrics
        verifier.reset_metrics().await;
        let metrics_after_reset = verifier.get_metrics().await;
        assert_eq!(metrics_after_reset.total_challenges, 0);
    }
}
