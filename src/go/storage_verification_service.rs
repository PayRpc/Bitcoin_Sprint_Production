use actix_web::{web, App, HttpServer, Responder, HttpResponse, middleware, Result};
use serde::{Serialize, Deserialize};
use std::sync::Arc;
use tokio::sync::Mutex;
use std::time::{SystemTime, UNIX_EPOCH, Duration, Instant};
use std::collections::HashMap;
use log::{info, error, warn};
use uuid::Uuid;
use reqwest::{Client, ClientBuilder};
use tokio::time::timeout;
use backoff::{ExponentialBackoff, backoff::Backoff};

// --- Connection Management Configuration ---
#[derive(Clone)]
struct ConnectionConfig {
    max_connections: usize,
    connect_timeout: Duration,
    request_timeout: Duration,
    pool_idle_timeout: Duration,
    max_idle_connections: usize,
    keep_alive: Duration,
}

impl Default for ConnectionConfig {
    fn default() -> Self {
        Self {
            max_connections: 100,
            connect_timeout: Duration::from_secs(10),
            request_timeout: Duration::from_secs(30),
            pool_idle_timeout: Duration::from_secs(90),
            max_idle_connections: 10,
            keep_alive: Duration::from_secs(60),
        }
    }
}

// --- Circuit Breaker Implementation ---
#[derive(Clone)]
struct CircuitBreaker {
    failure_count: Arc<Mutex<u32>>,
    last_failure_time: Arc<Mutex<Option<Instant>>>,
    state: Arc<Mutex<CircuitState>>,
    failure_threshold: u32,
    recovery_timeout: Duration,
    success_threshold: u32,
}

#[derive(Clone, PartialEq)]
enum CircuitState {
    Closed,
    Open,
    HalfOpen,
}

impl CircuitBreaker {
    fn new(failure_threshold: u32, recovery_timeout: Duration) -> Self {
        Self {
            failure_count: Arc::new(Mutex::new(0)),
            last_failure_time: Arc::new(Mutex::new(None)),
            state: Arc::new(Mutex::new(CircuitState::Closed)),
            failure_threshold,
            recovery_timeout,
            success_threshold: 3,
        }
    }

    async fn call<F, Fut, T>(&self, f: F) -> Result<T, Box<dyn std::error::Error + Send + Sync>>
    where
        F: FnOnce() -> Fut,
        Fut: std::future::Future<Output = Result<T, Box<dyn std::error::Error + Send + Sync>>>,
    {
        let state = self.state.lock().await.clone();

        match state {
            CircuitState::Open => {
                if let Some(last_failure) = *self.last_failure_time.lock().await {
                    if last_failure.elapsed() > self.recovery_timeout {
                        *self.state.lock().await = CircuitState::HalfOpen;
                        info!("Circuit breaker transitioning to Half-Open");
                    } else {
                        return Err("Circuit breaker is OPEN".into());
                    }
                }
            }
            CircuitState::HalfOpen => {
                // Allow limited requests in half-open state
            }
            CircuitState::Closed => {
                // Normal operation
            }
        }

        match f().await {
            Ok(result) => {
                if state == CircuitState::HalfOpen {
                    *self.state.lock().await = CircuitState::Closed;
                    *self.failure_count.lock().await = 0;
                    info!("Circuit breaker closed after successful call");
                }
                Ok(result)
            }
            Err(e) => {
                let mut failure_count = self.failure_count.lock().await;
                *failure_count += 1;
                *self.last_failure_time.lock().await = Some(Instant::now());

                if *failure_count >= self.failure_threshold {
                    *self.state.lock().await = CircuitState::Open;
                    warn!("Circuit breaker opened after {} failures", *failure_count);
                }
                Err(e)
            }
        }
    }
}

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
    verification_score: f64,
    connection_health: ConnectionHealth,
}

#[derive(Serialize, Deserialize)]
struct ConnectionHealth {
    pool_size: usize,
    active_connections: usize,
    idle_connections: usize,
    circuit_breaker_state: String,
}

#[derive(Serialize, Deserialize)]
struct ErrorResponse {
    error: String,
    code: u16,
    timestamp: u64,
    retry_after: Option<u64>,
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
    verifier: Arc<crate::EntropyStorageVerifier>,
    rate_limiter: Arc<Mutex<RateLimiter>>,
    active_challenges: Arc<Mutex<HashMap<String, Challenge>>>,
    http_client: Client,
    circuit_breaker: CircuitBreaker,
    connection_config: ConnectionConfig,
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

    if !["ipfs", "arweave", "filecoin"].contains(&req.protocol.to_lowercase().as_str()) {
        return Err("unsupported protocol".to_string());
    }

    if req.file_size == 0 || req.file_size > 1024 * 1024 * 1024 { // Max 1GB
        return Err("invalid file size".to_string());
    }

    Ok(())
}

// --- Enhanced API Endpoint with Connection Management ---
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
            retry_after: None,
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
                retry_after: Some(60), // Retry after 60 seconds
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

    // --- Circuit Breaker Protected External Call ---
    let verification_result = state.circuit_breaker.call(|| async {
        let mut backoff = ExponentialBackoff::default();

        loop {
            match timeout(state.connection_config.request_timeout, async {
                // Generate challenge with proper protocol mapping
                let protocol = match req.protocol.to_lowercase().as_str() {
                    "ipfs" => crate::StorageProtocol::IPFS,
                    "arweave" => crate::StorageProtocol::Arweave,
                    "filecoin" => crate::StorageProtocol::Filecoin,
                    _ => crate::StorageProtocol::IPFS,
                };

                let generated_challenge = state.verifier.generate_challenge(
                    &req.file_id,
                    req.file_size,
                    protocol,
                    &req.provider,
                    crate::ChallengeType::RandomSampling
                ).await?;

                // Create proof with connection-aware data generation
                let proof = crate::StorageProof {
                    challenge_id: challenge_id.clone(),
                    file_id: req.file_id.clone(),
                    provider_id: req.provider.clone(),
                    timestamp: now,
                    data_samples: generate_connection_aware_samples(&req.file_id, req.file_size, &state.http_client).await?,
                    merkle_proofs: Some(vec![format!("0x{}", hex::encode(&req.file_id))]),
                    provider_signature: format!("sig_{}_{}", req.provider, challenge_id),
                };

                // Verify proof
                state.verifier.verify_proof(proof).await
            }).await {
                Ok(Ok(result)) => return Ok(result),
                Ok(Err(e)) => {
                    if let Some(backoff_duration) = backoff.next_backoff() {
                        warn!("Request failed, retrying in {:?}: {}", backoff_duration, e);
                        tokio::time::sleep(backoff_duration).await;
                        continue;
                    } else {
                        return Err(e);
                    }
                }
                Err(_) => {
                    if let Some(backoff_duration) = backoff.next_backoff() {
                        warn!("Request timeout, retrying in {:?}", backoff_duration);
                        tokio::time::sleep(backoff_duration).await;
                        continue;
                    } else {
                        return Err("Request timeout after all retries".into());
                    }
                }
            }
        }
    }).await;

    let verification_result = match verification_result {
        Ok(result) => result,
        Err(e) => {
            error!("Verification failed for challenge {}: {}", challenge_id, e);
            return Ok(HttpResponse::InternalServerError().json(ErrorResponse {
                error: "Storage proof verification failed".to_string(),
                code: 500,
                timestamp: now,
                retry_after: Some(30),
            }));
        }
    };

    // --- Calculate Verification Score ---
    let verification_score = calculate_verification_score(
        &verification_result,
        req.file_size,
        &req.protocol
    );

    // --- Generate Signature ---
    let signature = match state.verifier.sign_challenge(
        verification_result.entropy_source.as_bytes(),
        &req.file_id,
        &req.provider
    ) {
        Ok(sig) => sig,
        Err(e) => {
            error!("Signature generation failed: {}", e);
            format!("0x{}", hex::encode("unsigned"))
        }
    };

    // --- Get Connection Health ---
    let connection_health = get_connection_health(&state).await;

    // --- Enhanced Response ---
    let response = VerifyResponse {
        verified: verification_result.verified && verification_score > 0.7,
        timestamp: now,
        signature,
        challenge_id,
        verification_score,
        connection_health,
    };

    info!("Verification completed for {} - Score: {:.3}, Verified: {}",
          req.file_id, verification_score, response.verified);

    Ok(HttpResponse::Ok().json(response))
}

// --- Connection-Aware Sample Generation ---
async fn generate_connection_aware_samples(
    file_id: &str,
    file_size: u64,
    client: &Client
) -> Result<Vec<Vec<u8>>, Box<dyn std::error::Error + Send + Sync>> {
    let sample_count = std::cmp::min(10, file_size / 1024);

    let mut samples = Vec::new();
    for i in 0..sample_count {
        // Simulate connection-aware data fetching with timeout
        let sample_data = timeout(Duration::from_secs(5), async {
            // In real implementation, this would fetch from storage provider
            let mut sample = file_id.as_bytes().to_vec();
            sample.extend_from_slice(&i.to_le_bytes());
            sample.resize(32, 0);
            Ok::<Vec<u8>, Box<dyn std::error::Error + Send + Sync>>(sample)
        }).await??;

        samples.push(sample_data);
    }

    Ok(samples)
}

// --- Connection Health Monitoring ---
async fn get_connection_health(state: &AppState) -> ConnectionHealth {
    // In a real implementation, you'd get actual connection pool stats
    // For now, return mock data
    ConnectionHealth {
        pool_size: state.connection_config.max_connections,
        active_connections: 5, // Mock active connections
        idle_connections: 3,   // Mock idle connections
        circuit_breaker_state: match *state.circuit_breaker.state.lock().await {
            CircuitState::Closed => "CLOSED".to_string(),
            CircuitState::Open => "OPEN".to_string(),
            CircuitState::HalfOpen => "HALF_OPEN".to_string(),
        },
    }
}

// --- Helper Functions ---
fn calculate_verification_score(
    result: &crate::VerificationResult,
    file_size: u64,
    protocol: &str
) -> f64 {
    let mut score = 0.0;

    // Base verification score
    if result.verified {
        score += 0.6;
    }

    // Protocol-specific bonuses
    match protocol.to_lowercase().as_str() {
        "ipfs" => score += 0.2,
        "arweave" => score += 0.25,
        "filecoin" => score += 0.3,
        _ => {}
    }

    // File size factor (larger files get slight bonus)
    let size_factor = (file_size as f64).log10() / 10.0;
    score += size_factor.min(0.15);

    // Ensure score is between 0.0 and 1.0
    score.max(0.0).min(1.0)
}

// --- Health Check Endpoint ---
async fn health(state: web::Data<AppState>) -> impl Responder {
    let connection_health = get_connection_health(&state).await;

    HttpResponse::Ok().json(serde_json::json!({
        "status": "healthy",
        "timestamp": SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
        "service": "entropy-storage-verifier",
        "connection_health": connection_health
    }))
}

// --- Metrics Endpoint ---
async fn metrics(state: web::Data<AppState>) -> impl Responder {
    let active_challenges = {
        let challenges = state.active_challenges.lock().await;
        challenges.len()
    };

    let connection_health = get_connection_health(&state).await;

    HttpResponse::Ok().json(serde_json::json!({
        "active_challenges": active_challenges,
        "timestamp": SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
        "connection_health": connection_health
    }))
}

// --- Enhanced Server with Connection Management ---
#[tokio::main]
async fn main() -> std::io::Result<()> {
    env_logger::init();

    info!("Starting Entropy Storage Verifier Service with Connection Management...");

    // --- Connection Configuration ---
    let connection_config = ConnectionConfig::default();

    // --- HTTP Client with Connection Pooling ---
    let http_client = ClientBuilder::new()
        .pool_max_idle_per_host(connection_config.max_idle_connections)
        .pool_idle_timeout(connection_config.pool_idle_timeout)
        .connect_timeout(connection_config.connect_timeout)
        .timeout(connection_config.request_timeout)
        .tcp_keepalive(connection_config.keep_alive)
        .user_agent("Entropy-Storage-Verifier/1.0")
        .build()
        .expect("Failed to create HTTP client");

    // --- Circuit Breaker ---
    let circuit_breaker = CircuitBreaker::new(5, Duration::from_secs(60));

    // --- Verifier ---
    let verifier = Arc::new(crate::EntropyStorageVerifier::new(
        "https://entropy.example",
        "0xContract"
    ));

    // --- Shared State ---
    let state = web::Data::new(AppState {
        verifier,
        rate_limiter: Arc::new(Mutex::new(RateLimiter::new(10, 60))), // 10 req/min
        active_challenges: Arc::new(Mutex::new(HashMap::new())),
        http_client,
        circuit_breaker,
        connection_config,
    });

    info!("Server configured:");
    info!("  - Rate limit: 10 req/min");
    info!("  - Max connections: {}", connection_config.max_connections);
    info!("  - Request timeout: {:?}", connection_config.request_timeout);
    info!("  - Circuit breaker: 5 failures, 60s recovery");
    info!("  - Binding to 0.0.0.0:8080");

    // --- Graceful Shutdown Handler ---
    let server = HttpServer::new(move || {
        App::new()
            .wrap(middleware::Logger::default())
            .wrap(middleware::DefaultHeaders::new()
                .add(("X-Version", "1.0.0"))
                .add(("X-Service", "entropy-storage-verifier"))
                .add(("X-Connection-Managed", "true")))
            .app_data(state.clone())
            .route("/verify", web::post().to(verify))
            .route("/health", web::get().to(health))
            .route("/metrics", web::get().to(metrics))
    })
    .bind(("0.0.0.0", 8080))?
    .workers(8)
    .shutdown_timeout(30); // 30 second graceful shutdown

    info!("🚀 Server started successfully with connection management");
    server.run().await
}
