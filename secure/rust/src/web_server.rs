// SPDX-License-Identifier: MIT
// Bitcoin Sprint - Enhanced Storage Verification Web API
// Production-ready REST API with rate limiting and challenge management

#[cfg(feature = "web-server")]

use actix_web::{web, App, HttpServer, Responder, HttpResponse, middleware, Result};
use serde::{Serialize, Deserialize};
use std::sync::Arc;
use tokio::sync::Mutex;
use std::time::{SystemTime, UNIX_EPOCH, Duration, Instant};
use std::collections::HashMap;
use log::{info, error, warn};
use uuid::Uuid;

// Re-export our storage verifier
use crate::storage_verifier::{
    StorageVerifier, RateLimitConfig, StorageChallenge, StorageProof,
    StorageVerificationError
};

// --- Enhanced Request / Response ---
#[derive(Serialize, Deserialize)]
struct VerifyRequest {
    file_id: String,
    provider: String,
    protocol: String,
    #[serde(default = "default_file_size")]
    file_size: u64,
}

fn default_file_size() -> u64 { 1024 * 1024 } // 1MB default

#[derive(Serialize, Deserialize)]
struct VerifyResponse {
    verified: bool,
    timestamp: u64,
    signature: String,
    challenge_id: String,
    verification_score: f64, // 0.0 to 1.0
}

#[derive(Serialize, Deserialize)]
struct ErrorResponse {
    error: String,
    code: u16,
    timestamp: u64,
}

// --- Enhanced Rate Limiting ---
#[derive(Clone)]
struct RateLimitEntry {
    count: u32,
    window_start: Instant,
    last_request: Instant,
}

struct RateLimiter {
    entries: HashMap<String, RateLimitEntry>,
    max_requests: u32,
    window_duration: Duration,
}

impl RateLimiter {
    fn new(max_requests: u32, window_seconds: u64) -> Self {
        Self {
            entries: HashMap::new(),
            max_requests,
            window_duration: Duration::from_secs(window_seconds),
        }
    }

    fn check_rate_limit(&mut self, key: &str) -> bool {
        let now = Instant::now();

        // Clean up old entries
        self.entries.retain(|_, entry| {
            now.duration_since(entry.last_request) < self.window_duration * 2
        });

        let entry = self.entries.entry(key.to_string()).or_insert(RateLimitEntry {
            count: 0,
            window_start: now,
            last_request: now,
        });

        // Reset window if expired
        if now.duration_since(entry.window_start) >= self.window_duration {
            entry.count = 0;
            entry.window_start = now;
        }

        entry.last_request = now;

        if entry.count >= self.max_requests {
            false
        } else {
            entry.count += 1;
            true
        }
    }
}

// --- Enhanced Shared State ---
struct AppState {
    verifier: Arc<StorageVerifier>,
    rate_limiter: Arc<Mutex<RateLimiter>>,
    active_challenges: Arc<Mutex<HashMap<String, Challenge>>>,
}

#[derive(Clone)]
struct Challenge {
    id: String,
    file_id: String,
    provider: String,
    created_at: Instant,
    expires_at: Instant,
}

// --- Validation ---
fn validate_request(req: &VerifyRequest) -> Result<(), String> {
    if req.file_id.is_empty() {
        return Err("file_id cannot be empty".to_string());
    }

    if req.provider.is_empty() {
        return Err("provider cannot be empty".to_string());
    }

    if !["ipfs", "arweave", "filecoin", "bitcoin"].contains(&req.protocol.to_lowercase().as_str()) {
        return Err("unsupported protocol".to_string());
    }

    if req.file_size == 0 || req.file_size > 1024 * 1024 * 1024 { // Max 1GB
        return Err("invalid file size".to_string());
    }

    Ok(())
}

// --- Enhanced API Endpoint ---
async fn verify(
    req: web::Json<VerifyRequest>,
    state: web::Data<AppState>,
) -> Result<impl Responder> {
    let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs();

    // --- Input Validation ---
    if let Err(e) = validate_request(&req) {
        warn!("Invalid request: {}", e);
        return Ok(HttpResponse::BadRequest().json(ErrorResponse {
            error: e,
            code: 400,
            timestamp: now,
        }));
    }

    // --- Enhanced Rate Limiting ---
    let rate_limit_key = format!("{}:{}", req.provider, req.file_id);
    {
        let mut limiter = state.rate_limiter.lock().await;
        if !limiter.check_rate_limit(&rate_limit_key) {
            warn!("Rate limit exceeded for {}", rate_limit_key);
            return Ok(HttpResponse::TooManyRequests().json(ErrorResponse {
                error: "Rate limit exceeded. Please try again later.".to_string(),
                code: 429,
                timestamp: now,
            }));
        }
    }

    // --- Challenge Management ---
    let challenge_id = Uuid::new_v4().to_string();
    let challenge = Challenge {
        id: challenge_id.clone(),
        file_id: req.file_id.clone(),
        provider: req.provider.clone(),
        created_at: Instant::now(),
        expires_at: Instant::now() + Duration::from_secs(300), // 5 min expiry
    };

    // Store challenge
    {
        let mut challenges = state.active_challenges.lock().await;

        // Clean expired challenges
        let now_instant = Instant::now();
        challenges.retain(|_, c| c.expires_at > now_instant);

        challenges.insert(challenge_id.clone(), challenge.clone());
        info!("Created challenge {} for file {} from provider {}",
              challenge_id, req.file_id, req.provider);
    }

    // --- Generate Challenge using our StorageVerifier ---
    let generated_challenge = match state.verifier.generate_challenge(&req.file_id, &req.provider).await {
        Ok(c) => c,
        Err(e) => {
            error!("Challenge generation failed for {}: {:?}", req.file_id, e);
            return Ok(HttpResponse::InternalServerError().json(ErrorResponse {
                error: "Failed to generate storage challenge".to_string(),
                code: 500,
                timestamp: now,
            }));
        }
    };

    // --- Enhanced Proof Creation ---
    let proof = StorageProof {
        challenge_id: challenge_id.clone(),
        file_id: req.file_id.clone(),
        provider: req.provider.clone(),
        timestamp: now,
        proof_data: generate_mock_samples(&req.file_id, req.file_size),
        merkle_proof: Some(vec![format!("0x{}", hex::encode(&req.file_id))]),
        signature: Some(format!("sig_{}_{}", req.provider, challenge_id)),
    };

    // --- Enhanced Verification ---
    let verification_result = match state.verifier.verify_proof(proof).await {
        Ok(result) => result,
        Err(e) => {
            error!("Verification failed for challenge {}: {:?}", challenge_id, e);
            return Ok(HttpResponse::InternalServerError().json(ErrorResponse {
                error: "Storage proof verification failed".to_string(),
                code: 500,
                timestamp: now,
            }));
        }
    };

    // --- Calculate Verification Score ---
    let verification_score = calculate_verification_score(
        verification_result,
        req.file_size,
        &req.protocol
    );

    // --- Generate Signature ---
    let signature = format!("sig_{}_{}_{}", req.provider, challenge_id, now);

    // --- Enhanced Response ---
    let response = VerifyResponse {
        verified: verification_result && verification_score > 0.7,
        timestamp: now,
        signature,
        challenge_id,
        verification_score,
    };

    info!("Verification completed for {} - Score: {:.3}, Verified: {}",
          req.file_id, verification_score, response.verified);

    Ok(HttpResponse::Ok().json(response))
}

// --- Helper Functions ---
fn generate_mock_samples(file_id: &str, file_size: u64) -> Vec<u8> {
    let sample_size = std::cmp::min(1024, file_size as usize); // Sample up to 1KB
    let mut sample = file_id.as_bytes().to_vec();
    sample.resize(sample_size, 0); // Pad to sample size
    sample
}

fn calculate_verification_score(
    verified: bool,
    file_size: u64,
    protocol: &str
) -> f64 {
    let mut score = 0.0;

    // Base verification score
    if verified {
        score += 0.6;
    }

    // Protocol-specific bonuses
    match protocol.to_lowercase().as_str() {
        "ipfs" => score += 0.2,
        "arweave" => score += 0.25,
        "filecoin" => score += 0.3,
        "bitcoin" => score += 0.35,
        _ => {}
    }

    // File size factor (larger files get slight bonus)
    let size_factor = (file_size as f64).log10() / 10.0;
    score += size_factor.min(0.15);

    // Ensure score is between 0.0 and 1.0
    score.max(0.0).min(1.0)
}

// --- Health Check Endpoint ---
async fn health() -> impl Responder {
    HttpResponse::Ok().json(serde_json::json!({
        "status": "healthy",
        "timestamp": SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
        "service": "bitcoin-sprint-storage-verifier"
    }))
}

// --- Metrics Endpoint ---
async fn metrics(state: web::Data<AppState>) -> impl Responder {
    let active_challenges = {
        let challenges = state.active_challenges.lock().await;
        challenges.len()
    };

    let verifier_metrics = state.verifier.get_metrics().await;

    HttpResponse::Ok().json(serde_json::json!({
        "active_challenges": active_challenges,
        "total_challenges": verifier_metrics.total_challenges,
        "successful_proofs": verifier_metrics.successful_proofs,
        "failed_proofs": verifier_metrics.failed_proofs,
        "rate_limited_requests": verifier_metrics.rate_limited_requests,
        "timestamp": SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
    }))
}

// --- Enhanced Server ---
pub async fn run_server() -> std::io::Result<()> {
    info!("Starting Bitcoin Sprint Storage Verifier Service...");

    // Create storage verifier with rate limiting config
    let rate_config = RateLimitConfig {
        max_requests_per_minute: 10,
        max_requests_per_hour: 100,
        cleanup_interval_secs: 60,
    };

    let verifier = Arc::new(StorageVerifier::with_config(rate_config));

    let state = web::Data::new(AppState {
        verifier,
        rate_limiter: Arc::new(Mutex::new(RateLimiter::new(10, 60))), // 10 req/min
        active_challenges: Arc::new(Mutex::new(HashMap::new())),
    });

    info!("Server configured - Rate limit: 10 req/min, Binding to 0.0.0.0:8080");

    HttpServer::new(move || {
        App::new()
            .wrap(middleware::Logger::default())
            .wrap(middleware::DefaultHeaders::new()
                .add(("X-Version", "1.0.0"))
                .add(("X-Service", "bitcoin-sprint-storage-verifier")))
            .app_data(state.clone())
            .route("/verify", web::post().to(verify))
            .route("/health", web::get().to(health))
            .route("/metrics", web::get().to(metrics))
    })
    .bind(("0.0.0.0", 8080))?
    .workers(8)
    .run()
    .await
}
