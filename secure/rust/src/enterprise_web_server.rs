// SPDX-License-Identifier: MIT
// Bitcoin Sprint - Enterprise Storage Validation API with Paid Service Support
// Enhanced with subscription tiers, advanced analytics, and multi-protocol support

#[cfg(feature = "web-server")]
mod web_server {
    use actix_web::{web, App, HttpServer, Responder, HttpResponse, Result, HttpRequest, middleware};
    use actix_web::http::header::{HeaderName, HeaderValue};
    use serde::{Serialize, Deserialize};
    use std::sync::Arc;
    use std::sync::Mutex;
    use tokio::sync::Mutex as AsyncMutex;
    use std::time::{SystemTime, UNIX_EPOCH, Duration, Instant};
    use std::collections::HashMap;
    use log::{info, error, warn};
    use uuid::Uuid;

    // Re-export our storage verifier
    use crate::storage_verifier::{
        StorageVerifier, RateLimitConfig, StorageChallenge, StorageProof,
        StorageVerificationError
    };

    // --- Enhanced Request/Response Types for Paid Service ---
    #[derive(Serialize, Deserialize)]
    pub struct ValidateStorageRequest {
        pub file_id: String,
        pub protocol: String,
        #[serde(default)]
        pub provider: Option<String>,
        #[serde(default)]
        pub file_size: Option<u64>,
        #[serde(default)]
        pub tier: String,
        #[serde(default)]
        pub webhook_url: Option<String>,
        #[serde(default)]
        pub merkle_proof: Option<MerkleProofData>,
    }

    #[derive(Serialize, Deserialize)]
    pub struct MerkleProofData {
        pub root: String,
        pub proof: Vec<MerkleProofElement>,
        pub leaf: String,
        pub index: u64,
        pub total_chunks: u64,
        pub chunk_size: u64,
        #[serde(default)]
        pub leaf_hashes: Vec<String>,
    }

    #[derive(Serialize, Deserialize)]
    pub struct MerkleProofElement {
        pub hash: String,
        pub position: String,
    }

    #[derive(Serialize, Deserialize)]
    pub struct ValidateStorageResponse {
        pub status: String,
        pub verified: bool,
        pub verification_score: f64,
        pub response_time_ms: u64,
        pub challenge_id: String,
        pub protocol: String,
        pub provider: String,
        pub tier_used: String,
        pub credits_used: u32,
        pub credits_remaining: u32,
        #[serde(default)]
        pub merkle_root: Option<String>,
        #[serde(default)]
        pub merkle_proof_valid: bool,
        pub timestamp: u64,
        #[serde(default)]
        pub webhook_sent: bool,
    }

    #[derive(Serialize, Deserialize)]
    pub struct SubscriptionInfo {
        pub tier: String,
        pub credits_remaining: u32,
        pub monthly_limit: u32,
        pub reset_date: u64,
        pub features: Vec<String>,
    }

    #[derive(Serialize, Deserialize)]
    pub struct AnalyticsResponse {
        pub total_verifications: u64,
        pub success_rate: f64,
        pub average_response_time: f64,
        pub protocol_usage: HashMap<String, u64>,
        pub daily_stats: Vec<DailyStat>,
        pub top_providers: Vec<ProviderStat>,
    }

    #[derive(Serialize, Deserialize)]
    pub struct DailyStat {
        pub date: String,
        pub verifications: u64,
        pub success_rate: f64,
    }

    #[derive(Serialize, Deserialize)]
    pub struct ProviderStat {
        pub provider: String,
        pub verifications: u64,
        pub success_rate: f64,
    }

    // --- Subscription Tiers ---
    #[derive(Clone, Debug)]
    pub struct SubscriptionTier {
        pub name: String,
        pub monthly_credits: u32,
        pub max_concurrent_requests: u32,
        pub features: Vec<String>,
        pub priority: u8,
    }

    impl SubscriptionTier {
        pub fn new(name: &str, credits: u32, max_concurrent: u32, priority: u8) -> Self {
            let features = match name {
                "free" => vec![
                    "Basic verification".to_string(),
                    "IPFS support".to_string(),
                    "Email support".to_string(),
                ],
                "developer" => vec![
                    "Advanced verification".to_string(),
                    "All protocols".to_string(),
                    "Priority support".to_string(),
                    "Basic analytics".to_string(),
                ],
                "professional" => vec![
                    "Enterprise verification".to_string(),
                    "Custom protocols".to_string(),
                    "Advanced analytics".to_string(),
                    "Webhook notifications".to_string(),
                    "SLA monitoring".to_string(),
                ],
                "enterprise" => vec![
                    "Unlimited verification".to_string(),
                    "White-label solution".to_string(),
                    "Dedicated support".to_string(),
                    "Custom SLAs".to_string(),
                    "On-premise deployment".to_string(),
                ],
                _ => vec![],
            };

            Self {
                name: name.to_string(),
                monthly_credits: credits,
                max_concurrent_requests: max_concurrent,
                features,
                priority,
            }
        }
    }

    // --- Enhanced Web Server with Paid Service Support ---
    #[derive(Clone)]
    pub struct EnterpriseWebServer {
        verifier: Arc<StorageVerifier>,
        subscriptions: Arc<AsyncMutex<HashMap<String, SubscriptionTier>>>,
        usage_stats: Arc<AsyncMutex<HashMap<String, UserStats>>>,
        active_requests: Arc<AsyncMutex<HashMap<String, Vec<Instant>>>>,
    }

    #[derive(Clone)]
    struct UserStats {
        total_verifications: u64,
        successful_verifications: u64,
        total_response_time: u64,
        protocol_usage: HashMap<String, u64>,
        last_reset: u64,
        credits_used: u32,
    }

    impl EnterpriseWebServer {
        pub fn new(verifier: StorageVerifier) -> Self {
            let mut subscriptions = HashMap::new();

            // Initialize subscription tiers
            subscriptions.insert("free".to_string(), SubscriptionTier::new("free", 100, 1, 0));
            subscriptions.insert("developer".to_string(), SubscriptionTier::new("developer", 1000, 5, 1));
            subscriptions.insert("professional".to_string(), SubscriptionTier::new("professional", 50000, 20, 2));
            subscriptions.insert("enterprise".to_string(), SubscriptionTier::new("enterprise", u32::MAX, 100, 3));

            Self {
                verifier: Arc::new(verifier),
                subscriptions: Arc::new(AsyncMutex::new(subscriptions)),
                usage_stats: Arc::new(AsyncMutex::new(HashMap::new())),
                active_requests: Arc::new(AsyncMutex::new(HashMap::new())),
            }
        }

        fn get_api_key_from_request(req: &HttpRequest) -> Option<String> {
            req.headers()
                .get("authorization")
                .and_then(|h| h.to_str().ok())
                .and_then(|s| s.strip_prefix("Bearer "))
                .map(|s| s.to_string())
        }

        async fn authenticate_and_get_tier(&self, api_key: &str) -> Result<SubscriptionTier, HttpResponse> {
            // In production, this would validate against a database
            // For demo purposes, we'll use a simple mapping
            let tier_name = match api_key {
                "free_trial_key" => "free",
                "dev_key_123" => "developer",
                "pro_key_456" => "professional",
                "ent_key_789" => "enterprise",
                _ => return Err(HttpResponse::Unauthorized().json(serde_json::json!({
                    "error": "Invalid API key",
                    "code": 401
                }))),
            };

            let subscriptions = self.subscriptions.lock().await;
            subscriptions.get(tier_name)
                .cloned()
                .ok_or_else(|| HttpResponse::InternalServerError().json(serde_json::json!({
                    "error": "Subscription tier not found",
                    "code": 500
                })))
        }

        async fn check_rate_limits(&self, api_key: &str, tier: &SubscriptionTier) -> Result<(), HttpResponse> {
            let mut active_requests = self.active_requests.lock().await;

            let user_requests = active_requests.entry(api_key.to_string()).or_insert_with(Vec::new);

            // Remove expired requests (older than 1 minute)
            let now = Instant::now();
            user_requests.retain(|&time| now.duration_since(time) < Duration::from_secs(60));

            if user_requests.len() >= tier.max_concurrent_requests as usize {
                return Err(HttpResponse::TooManyRequests().json(serde_json::json!({
                    "error": "Rate limit exceeded",
                    "code": 429,
                    "retry_after": 60
                })));
            }

            user_requests.push(now);
            Ok(())
        }

        async fn check_credits(&self, api_key: &str, tier: &SubscriptionTier) -> Result<u32, HttpResponse> {
            let mut usage_stats = self.usage_stats.lock().await;
            let stats = usage_stats.entry(api_key.to_string()).or_insert_with(|| UserStats {
                total_verifications: 0,
                successful_verifications: 0,
                total_response_time: 0,
                protocol_usage: HashMap::new(),
                last_reset: SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
                credits_used: 0,
            });

            // Reset credits if monthly limit reached or new month
            let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs();
            if now - stats.last_reset > 30 * 24 * 3600 { // 30 days
                stats.credits_used = 0;
                stats.last_reset = now;
            }

            let credits_remaining = if tier.monthly_credits == u32::MAX {
                u32::MAX // Unlimited for enterprise
            } else {
                tier.monthly_credits.saturating_sub(stats.credits_used)
            };

            if credits_remaining == 0 {
                return Err(HttpResponse::PaymentRequired().json(serde_json::json!({
                    "error": "Monthly credit limit exceeded",
                    "code": 402,
                    "upgrade_url": "/pricing"
                })));
            }

            Ok(credits_remaining)
        }

        async fn update_stats(&self, api_key: &str, protocol: &str, success: bool, response_time: u64) {
            let mut usage_stats = self.usage_stats.lock().await;
            let stats = usage_stats.entry(api_key.to_string()).or_insert_with(|| UserStats {
                total_verifications: 0,
                successful_verifications: 0,
                total_response_time: 0,
                protocol_usage: HashMap::new(),
                last_reset: SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
                credits_used: 0,
            });

            stats.total_verifications += 1;
            stats.total_response_time += response_time;
            *stats.protocol_usage.entry(protocol.to_string()).or_insert(0) += 1;

            if success {
                stats.successful_verifications += 1;
            }

            stats.credits_used += 1;
        }
    }

    // --- API Handlers ---
    impl EnterpriseWebServer {
        pub async fn validate_storage(
            &self,
            req: web::Json<ValidateStorageRequest>,
            http_req: HttpRequest,
        ) -> Result<HttpResponse> {
            let start_time = Instant::now();

            // Extract and validate API key
            let api_key = match Self::get_api_key_from_request(&http_req) {
                Some(key) => key,
                None => return Ok(HttpResponse::Unauthorized().json(serde_json::json!({
                    "error": "Missing API key",
                    "code": 401
                }))),
            };

            // Authenticate and get subscription tier
            let tier = match self.authenticate_and_get_tier(&api_key).await {
                Ok(tier) => tier,
                Err(resp) => return Ok(resp),
            };

            // Check rate limits
            if let Err(resp) = self.check_rate_limits(&api_key, &tier).await {
                return Ok(resp);
            }

            // Check credits
            let credits_remaining = match self.check_credits(&api_key, &tier).await {
                Ok(remaining) => remaining,
                Err(resp) => return Ok(resp),
            };

            // Perform validation
            let challenge = StorageChallenge {
                id: Uuid::new_v4().to_string(),
                file_id: req.file_id.clone(),
                provider: req.provider.clone().unwrap_or_else(|| "auto".to_string()),
                nonce: rand::random(),
                timestamp: SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
                expiry: SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs() + 3600, // 1 hour
                beacon: format!("beacon-{}", rand::random::<u64>()),
                difficulty: 8,
                challenge_data: vec![],
                sample_offset: 0,
                sample_size: 1024,
                chunk_index: 0,
                commitment_alg: "sha256_chunks".to_string(),
            };

            // Generate proof for the challenge
            let mut proof = StorageProof {
                challenge_id: challenge.id.clone(),
                file_id: challenge.file_id.clone(),
                provider: challenge.provider.clone(),
                timestamp: SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
                proof_data: vec![1, 2, 3, 4], // Mock proof data
                merkle_proof: None,
                signature: None,
            };

            // Handle Merkle proof if provided
            let mut merkle_root = None;
            let mut merkle_proof_valid = false;

            if let Some(merkle_data) = &req.merkle_proof {
                // Convert web Merkle proof format to internal format
                let mut merkle_proof_strings = Vec::new();
                for element in &merkle_data.proof {
                    merkle_proof_strings.push(element.hash.clone());
                }

                proof.merkle_proof = Some(merkle_proof_strings);

                // Store Merkle root for verification
                if let Ok(root_bytes) = hex::decode(&merkle_data.root.trim_start_matches("0x")) {
                    if root_bytes.len() == 32 {
                        let mut root_array = [0u8; 32];
                        root_array.copy_from_slice(&root_bytes);
                        merkle_root = Some(merkle_data.root.clone());

                        // Register Merkle root with the verifier
                        if let Err(e) = self.verifier.register_merkle_root(
                            &req.file_id,
                            root_array,
                            merkle_data.chunk_size as u32,
                            merkle_data.total_chunks
                        ).await {
                            warn!("Failed to register Merkle root: {:?}", e);
                        }
                    }
                }
            }

            let verification_result = self.verifier.verify_proof(proof.clone()).await;
            let response_time = start_time.elapsed().as_millis() as u64;

            let (verified, verification_score) = match verification_result {
                Ok(_) => {
                    merkle_proof_valid = req.merkle_proof.is_some();
                    (true, 0.95)
                },
                Err(_) => (false, 0.0),
            };

            // Update statistics
            self.update_stats(&api_key, &req.protocol, verified, response_time).await;

            // Send webhook if provided
            let webhook_sent = if let Some(webhook_url) = &req.webhook_url {
                self.send_webhook(webhook_url, &challenge, verified, verification_score).await
            } else {
                false
            };

            let response = ValidateStorageResponse {
                status: if verified { "verified" } else { "failed" }.to_string(),
                verified,
                verification_score,
                response_time_ms: response_time,
                challenge_id: challenge.id,
                protocol: req.protocol.clone(),
                provider: challenge.provider,
                tier_used: tier.name,
                credits_used: 1,
                credits_remaining: credits_remaining.saturating_sub(1),
                merkle_root,
                merkle_proof_valid,
                timestamp: SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
                webhook_sent,
            };

            Ok(HttpResponse::Ok().json(response))
        }

        pub async fn get_subscription_info(
            &self,
            http_req: HttpRequest,
        ) -> Result<HttpResponse> {
            let api_key = match Self::get_api_key_from_request(&http_req) {
                Some(key) => key,
                None => return Ok(HttpResponse::Unauthorized().json(serde_json::json!({
                    "error": "Missing API key",
                    "code": 401
                }))),
            };

            let tier = match self.authenticate_and_get_tier(&api_key).await {
                Ok(tier) => tier,
                Err(resp) => return Ok(resp),
            };

            let credits_remaining = match self.check_credits(&api_key, &tier).await {
                Ok(remaining) => remaining,
                Err(_) => 0, // If error, assume no credits
            };

            let info = SubscriptionInfo {
                tier: tier.name,
                credits_remaining,
                monthly_limit: tier.monthly_credits,
                reset_date: SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs() + 30 * 24 * 3600,
                features: tier.features,
            };

            Ok(HttpResponse::Ok().json(info))
        }

        pub async fn get_analytics(
            &self,
            http_req: HttpRequest,
        ) -> Result<HttpResponse> {
            let api_key = match Self::get_api_key_from_request(&http_req) {
                Some(key) => key,
                None => return Ok(HttpResponse::Unauthorized().json(serde_json::json!({
                    "error": "Missing API key",
                    "code": 401
                }))),
            };

            let tier = match self.authenticate_and_get_tier(&api_key).await {
                Ok(tier) => tier,
                Err(resp) => return Ok(resp),
            };

            // Only professional and enterprise tiers get analytics
            if !matches!(tier.name.as_str(), "professional" | "enterprise") {
                return Ok(HttpResponse::Forbidden().json(serde_json::json!({
                    "error": "Analytics requires Professional or Enterprise tier",
                    "code": 403
                })));
            }

            let usage_stats = self.usage_stats.lock().await;
            let default_stats = UserStats {
                total_verifications: 0,
                successful_verifications: 0,
                total_response_time: 0,
                protocol_usage: HashMap::new(),
                last_reset: SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
                credits_used: 0,
            };
            let stats = usage_stats.get(&api_key).unwrap_or(&default_stats);

            let success_rate = if stats.total_verifications > 0 {
                stats.successful_verifications as f64 / stats.total_verifications as f64
            } else {
                0.0
            };

            let average_response_time = if stats.total_verifications > 0 {
                stats.total_response_time as f64 / stats.total_verifications as f64
            } else {
                0.0
            };

            let analytics = AnalyticsResponse {
                total_verifications: stats.total_verifications,
                success_rate,
                average_response_time,
                protocol_usage: stats.protocol_usage.clone(),
                daily_stats: vec![], // Would be populated from database
                top_providers: vec![], // Would be populated from database
            };

            Ok(HttpResponse::Ok().json(analytics))
        }

        async fn send_webhook(&self, webhook_url: &str, challenge: &StorageChallenge, verified: bool, score: f64) -> bool {
            let payload = serde_json::json!({
                "event": "storage_verification_complete",
                "challenge_id": challenge.id,
                "file_id": challenge.file_id,
                "verified": verified,
                "verification_score": score,
                "protocol": challenge.provider,
                "provider": challenge.provider,
                "timestamp": SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
            });

            // In production, this would use a proper HTTP client with retries
            // For demo purposes, we'll just log it
            info!("Webhook would be sent to {}: {}", webhook_url, payload);
            true
        }
    }

    // --- Server Setup ---
    pub async fn run_enterprise_server(verifier: StorageVerifier, port: u16) -> std::io::Result<()> {
        let server = EnterpriseWebServer::new(verifier);

        info!("🚀 Starting Bitcoin Sprint Enterprise Storage Validation Server on port {}", port);

        HttpServer::new(move || {
            App::new()
                .app_data(web::Data::new(server.clone()))
                .wrap(middleware::Logger::default())
                .route("/api/validate-storage", web::post().to(
                    |req: web::Json<ValidateStorageRequest>, http_req: HttpRequest, server: web::Data<EnterpriseWebServer>| async move {
                        server.validate_storage(req, http_req).await
                    }
                ))
                .route("/api/subscription", web::get().to(
                    |http_req: HttpRequest, server: web::Data<EnterpriseWebServer>| async move {
                        server.get_subscription_info(http_req).await
                    }
                ))
                .route("/api/analytics", web::get().to(
                    |http_req: HttpRequest, server: web::Data<EnterpriseWebServer>| async move {
                        server.get_analytics(http_req).await
                    }
                ))
                .route("/health", web::get().to(|| async {
                    HttpResponse::Ok().json(serde_json::json!({
                        "status": "healthy",
                        "timestamp": SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs()
                    }))
                }))
        })
        .bind(("0.0.0.0", port))?
        .run()
        .await
    }
}

// Re-export the public function when the feature is enabled
#[cfg(feature = "web-server")]
pub use web_server::run_enterprise_server;

// Re-export the request/response types
#[cfg(feature = "web-server")]
pub use web_server::{ValidateStorageRequest, ValidateStorageResponse, MerkleProofData, MerkleProofElement};