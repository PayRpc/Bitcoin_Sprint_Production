use axum::{extract::{Path, Query}, middleware, middleware::from_fn, http::{Request, StatusCode}, response::{IntoResponse, Response}, routing::{get, post}, Router, Json};
use chrono::{DateTime, Utc};
use dotenvy::dotenv;
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use sha2::{Digest, Sha256};
use std::collections::HashMap;
use std::env;
use std::net::SocketAddr;
use std::sync::Arc;
use tokio::sync::Mutex;
use std::time::{Duration, Instant};
use tokio::net::TcpStream;
use tokio::sync::mpsc;
use tracing::{debug, error, info, warn};
use axum::http::header::CONTENT_TYPE;
use prometheus::{Encoder, TextEncoder, register_counter_vec, CounterVec, register_gauge_vec, GaugeVec, register_histogram_vec, HistogramVec};
use uuid::Uuid;
use rand;
use hex;

// Version information
const VERSION: &str = env!("CARGO_PKG_VERSION");
const COMMIT: &str = "unknown";

// Protocol types
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
enum ProtocolType {
    Bitcoin,
    Ethereum,
    Solana,
}

impl std::fmt::Display for ProtocolType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ProtocolType::Bitcoin => write!(f, "bitcoin"),
            ProtocolType::Ethereum => write!(f, "ethereum"),
            ProtocolType::Solana => write!(f, "solana"),
        }
    }
}

// Config struct (expanded to match Go more closely)
#[derive(Debug, Serialize, Deserialize, Clone)]
struct Config {
    tier: String,
    api_host: String,
    api_port: u16,
    max_connections: u32,
    message_queue_size: u32,
    circuit_breaker_threshold: u32,
    circuit_breaker_timeout: u32,
    circuit_breaker_half_open_max: u32,
    enable_encryption: bool,
    pipeline_workers: u32,
    write_deadline: Duration,
    optimize_system: bool,
    buffer_size: u32,
    worker_count: u32,
    simulate_blocks: bool,
    tcp_keep_alive: Duration,
    read_buffer_size: u32,
    write_buffer_size: u32,
    connection_timeout: Duration,
    idle_timeout: Duration,
    max_cpu: u32,
    gc_percent: u32,
    prealloc_buffers: bool,
    lock_os_thread: bool,
    license_key: String,
    zmq_endpoint: String,
    bloom_filter_enabled: bool,
    enterprise_security_enabled: bool,
    audit_log_path: String,
    max_retries: u32,
    retry_backoff: Duration,
    cache_size: u32,
    cache_ttl: Duration,
    websocket_max_connections: u32,
    websocket_max_per_ip: u32,
    websocket_max_per_chain: u32,
    database_type: String,
    database_url: String,
    database_max_conns: u32,
    database_min_conns: u32,
    rust_web_server_enabled: bool,
    rust_web_server_host: String,
    rust_web_server_port: u16,
    rust_admin_server_port: u16,
    rust_metrics_port: u16,
    rust_tls_cert_path: String,
    rust_tls_key_path: String,
    rust_redis_url: String,
    // Protocol toggles
    enable_bitcoin: bool,
    enable_ethereum: bool,
    enable_solana: bool,
}

impl Config {
    fn load() -> Self {
        dotenv().ok();

        // Expanded parsing to match Go's getEnv* functions
        let parse_duration_secs = |key: &str, default: u64| -> Duration {
            let val = env::var(key).unwrap_or_else(|_| format!("{}s", default));
            let secs: u64 = val.trim_end_matches('s').parse().unwrap_or(default);
            Duration::from_secs(secs)
        };

        let parse_duration_ms = |key: &str, default: u64| -> Duration {
            let val = env::var(key).unwrap_or_else(|_| format!("{}ms", default));
            let ms: u64 = val.trim_end_matches("ms").parse().unwrap_or(default);
            Duration::from_millis(ms)
        };

        Config {
            tier: env::var("RELAY_TIER").unwrap_or("Enterprise".to_string()),
            api_host: env::var("API_HOST").unwrap_or("0.0.0.0".to_string()),
            api_port: env::var("API_PORT").ok().and_then(|s| s.parse().ok()).unwrap_or(8443),
            max_connections: env::var("MAX_CONNECTIONS").ok().and_then(|s| s.parse().ok()).unwrap_or(20),
            message_queue_size: env::var("MESSAGE_QUEUE_SIZE").ok().and_then(|s| s.parse().ok()).unwrap_or(1000),
            circuit_breaker_threshold: env::var("CIRCUIT_BREAKER_THRESHOLD").ok().and_then(|s| s.parse().ok()).unwrap_or(3),
            circuit_breaker_timeout: env::var("CIRCUIT_BREAKER_TIMEOUT").ok().and_then(|s| s.parse().ok()).unwrap_or(30),
            circuit_breaker_half_open_max: env::var("CIRCUIT_BREAKER_HALF_OPEN_MAX").ok().and_then(|s| s.parse().ok()).unwrap_or(2),
            enable_encryption: env::var("ENABLE_ENCRYPTION").map(|s| s == "true").unwrap_or(true),
            pipeline_workers: env::var("PIPELINE_WORKERS").ok().and_then(|s| s.parse().ok()).unwrap_or(10),
            write_deadline: parse_duration_ms("WRITE_DEADLINE", 100),
            optimize_system: env::var("OPTIMIZE_SYSTEM").map(|s| s == "true").unwrap_or(true),
            buffer_size: env::var("BUFFER_SIZE").ok().and_then(|s| s.parse().ok()).unwrap_or(1000),
            worker_count: env::var("WORKER_COUNT").ok().and_then(|s| s.parse().ok()).unwrap_or(num_cpus::get() as u32),
            simulate_blocks: env::var("SIMULATE_BLOCKS").map(|s| s == "true").unwrap_or(false),
            tcp_keep_alive: parse_duration_secs("TCP_KEEP_ALIVE", 15),
            read_buffer_size: env::var("READ_BUFFER_SIZE").ok().and_then(|s| s.parse().ok()).unwrap_or(16 * 1024),
            write_buffer_size: env::var("WRITE_BUFFER_SIZE").ok().and_then(|s| s.parse().ok()).unwrap_or(16 * 1024),
            connection_timeout: parse_duration_secs("CONNECTION_TIMEOUT", 5),
            idle_timeout: parse_duration_secs("IDLE_TIMEOUT", 120),
            max_cpu: env::var("MAX_CPU").ok().and_then(|s| s.parse().ok()).unwrap_or(num_cpus::get() as u32),
            gc_percent: env::var("GC_PERCENT").ok().and_then(|s| s.parse().ok()).unwrap_or(100),
            prealloc_buffers: env::var("PREALLOC_BUFFERS").map(|s| s == "true").unwrap_or(true),
            lock_os_thread: env::var("LOCK_OS_THREAD").map(|s| s == "true").unwrap_or(true),
            license_key: env::var("LICENSE_KEY").unwrap_or_default(),
            zmq_endpoint: env::var("ZMQ_ENDPOINT").unwrap_or("tcp://127.0.0.1:28332".to_string()),
            bloom_filter_enabled: env::var("BLOOM_FILTER_ENABLED").map(|s| s == "true").unwrap_or(true),
            enterprise_security_enabled: env::var("ENTERPRISE_SECURITY_ENABLED").map(|s| s == "true").unwrap_or(true),
            audit_log_path: env::var("AUDIT_LOG_PATH").unwrap_or("/var/log/sprint/audit.log".to_string()),
            max_retries: env::var("MAX_RETRIES").ok().and_then(|s| s.parse().ok()).unwrap_or(3),
            retry_backoff: parse_duration_ms("RETRY_BACKOFF", 100),
            cache_size: env::var("CACHE_SIZE").ok().and_then(|s| s.parse().ok()).unwrap_or(10000),
            cache_ttl: parse_duration_secs("CACHE_TTL", 5 * 60),
            websocket_max_connections: env::var("WEBSOCKET_MAX_CONNECTIONS").ok().and_then(|s| s.parse().ok()).unwrap_or(1000),
            websocket_max_per_ip: env::var("WEBSOCKET_MAX_PER_IP").ok().and_then(|s| s.parse().ok()).unwrap_or(100),
            websocket_max_per_chain: env::var("WEBSOCKET_MAX_PER_CHAIN").ok().and_then(|s| s.parse().ok()).unwrap_or(200),
            database_type: env::var("DATABASE_TYPE").unwrap_or("sqlite".to_string()),
            database_url: env::var("DATABASE_URL").unwrap_or("./sprint.db".to_string()),
            database_max_conns: env::var("DATABASE_MAX_CONNS").ok().and_then(|s| s.parse().ok()).unwrap_or(10),
            database_min_conns: env::var("DATABASE_MIN_CONNS").ok().and_then(|s| s.parse().ok()).unwrap_or(2),
            rust_web_server_enabled: env::var("RUST_WEB_SERVER_ENABLED").map(|s| s == "true").unwrap_or(true),
            rust_web_server_host: env::var("RUST_WEB_SERVER_HOST").unwrap_or("127.0.0.1".to_string()),
            rust_web_server_port: env::var("RUST_WEB_SERVER_PORT").ok().and_then(|s| s.parse().ok()).unwrap_or(8443),
            rust_admin_server_port: env::var("RUST_ADMIN_SERVER_PORT").ok().and_then(|s| s.parse().ok()).unwrap_or(8444),
            rust_metrics_port: env::var("RUST_METRICS_PORT").ok().and_then(|s| s.parse().ok()).unwrap_or(9092),
            rust_tls_cert_path: env::var("RUST_TLS_CERT_PATH").unwrap_or("/app/config/tls/cert.pem".to_string()),
            rust_tls_key_path: env::var("RUST_TLS_KEY_PATH").unwrap_or("/app/config/tls/key.pem".to_string()),
            rust_redis_url: env::var("RUST_REDIS_URL").unwrap_or("redis://redis:6379".to_string()),
            // Protocol toggles (default: only Bitcoin enabled)
            enable_bitcoin: env::var("ENABLE_BITCOIN").map(|s| s == "true").unwrap_or(true),
            enable_ethereum: env::var("ENABLE_ETHEREUM").map(|s| s == "true").unwrap_or(false),
            enable_solana: env::var("ENABLE_SOLANA").map(|s| s == "true").unwrap_or(false),
        }
    }
}

// Simplified Cache (matching Go's Cache)
#[derive(Clone)]
struct Cache {
    items: Arc<Mutex<HashMap<String, CacheItem>>>,
    max_size: usize,
}

#[derive(Clone)]
struct CacheItem {
    value: Value,
    expires_at: DateTime<Utc>,
}

impl Cache {
    fn new(max_size: usize) -> Self {
        Cache {
            items: Arc::new(Mutex::new(HashMap::new())),
            max_size,
        }
    }

    async fn set(&self, key: String, value: Value, ttl: Duration) {
        let mut items = self.items.lock().await;
        if items.len() >= self.max_size {
            // Simple eviction: remove oldest (not LRU, but approx)
            let oldest_key = items.keys().next().cloned().unwrap_or_default();
            items.remove(&oldest_key);
        }
        items.insert(
            key,
            CacheItem {
                value,
                expires_at: Utc::now() + chrono::Duration::from_std(ttl).unwrap(),
            },
        );
    }

    async fn get(&self, key: &str) -> Option<Value> {
        let mut items = self.items.lock().await;
        if let Some(item) = items.get(key) {
            if Utc::now() > item.expires_at {
                items.remove(key);
                return None;
            }
            Some(item.value.clone())
        } else {
            None
        }
    }
}

// Simplified LatencyOptimizer
#[derive(Clone)]
struct LatencyOptimizer {
    target_p99: Duration,
    chain_latencies: Arc<Mutex<HashMap<String, Vec<Duration>>>>,
}

impl LatencyOptimizer {
    fn new(target_p99: Duration) -> Self {
        LatencyOptimizer {
            target_p99,
            chain_latencies: Arc::new(Mutex::new(HashMap::new())),
        }
    }

    async fn track_request(&self, chain: &str, duration: Duration) {
        let mut latencies = self.chain_latencies.lock().await;
        let chain_vec = latencies.entry(chain.to_string()).or_insert(Vec::new());
        chain_vec.push(duration);
        if chain_vec.len() > 1000 {
            chain_vec.remove(0);
        }
        // Simplified P99 calculation
        if chain_vec.len() >= 10 {
            let mut sorted = chain_vec.clone();
            sorted.sort();
            let p99_index = ((0.99f64 * sorted.len() as f64).ceil() as usize).saturating_sub(1);
            let current_p99 = sorted[p99_index];
            if current_p99 > self.target_p99 {
                warn!("P99 exceeded for chain {}: {:?} > {:?}", chain, current_p99, self.target_p99);
            }
        }
    }
}

// Tier Management System (ported from Go)
#[derive(Debug, Clone, Serialize, Deserialize)]
struct TierConfig {
    name: String,
    requests_per_second: u32,
    requests_per_month: u64,
    max_concurrent: u32,
    cache_priority: u32,
    latency_target_ms: u64,
    features: Vec<String>,
    price_per_request: f64,
}

#[derive(Debug, Clone)]
struct TierManager {
    tiers: HashMap<String, TierConfig>,
    user_tiers: Arc<Mutex<HashMap<String, String>>>,
    rate_limiters: Arc<Mutex<HashMap<String, RateLimiter>>>,
    monetization: MonetizationEngine,
}

impl TierManager {
    fn new() -> Self {
        let mut tiers = HashMap::new();

        // Free tier
        tiers.insert("free".to_string(), TierConfig {
            name: "Free".to_string(),
            requests_per_second: 10,
            requests_per_month: 100_000,
            max_concurrent: 5,
            cache_priority: 1,
            latency_target_ms: 500,
            features: vec!["basic_api".to_string()],
            price_per_request: 0.0,
        });

        // Pro tier
        tiers.insert("pro".to_string(), TierConfig {
            name: "Pro".to_string(),
            requests_per_second: 100,
            requests_per_month: 10_000_000,
            max_concurrent: 50,
            cache_priority: 2,
            latency_target_ms: 100,
            features: vec!["basic_api".to_string(), "websockets".to_string(), "historical_data".to_string()],
            price_per_request: 0.0001,
        });

        // Enterprise tier
        tiers.insert("enterprise".to_string(), TierConfig {
            name: "Enterprise".to_string(),
            requests_per_second: 1000,
            requests_per_month: 1_000_000_000,
            max_concurrent: 500,
            cache_priority: 3,
            latency_target_ms: 50,
            features: vec!["all".to_string(), "custom_endpoints".to_string(), "dedicated_support".to_string(), "sla".to_string()],
            price_per_request: 0.00005,
        });

        TierManager {
            tiers,
            user_tiers: Arc::new(Mutex::new(HashMap::new())),
            rate_limiters: Arc::new(Mutex::new(HashMap::new())),
            monetization: MonetizationEngine::new(),
        }
    }

    async fn get_tier_config(&self, tier: &str) -> Option<&TierConfig> {
        self.tiers.get(tier)
    }

    async fn assign_user_tier(&self, user_id: &str, tier: &str) {
        let mut user_tiers = self.user_tiers.lock().await;
        user_tiers.insert(user_id.to_string(), tier.to_string());
    }

    async fn get_user_tier(&self, user_id: &str) -> String {
        let user_tiers = self.user_tiers.lock().await;
        user_tiers.get(user_id).cloned().unwrap_or_else(|| "free".to_string())
    }

    async fn check_rate_limit(&self, user_id: &str) -> bool {
        let user_tier = self.get_user_tier(user_id).await;
        let tier_config = match self.get_tier_config(&user_tier).await {
            Some(config) => config,
            None => return false,
        };

        let mut rate_limiters = self.rate_limiters.lock().await;
        let limiter = rate_limiters.entry(user_id.to_string()).or_insert_with(|| {
            RateLimiter::new(tier_config.requests_per_second as u64, Duration::from_secs(60))
        });

        limiter.allow()
    }
}

// Rate Limiter (ported from Go)
#[derive(Debug, Clone)]
struct RateLimiter {
    tokens: Arc<Mutex<f64>>,
    max_tokens: f64,
    refill_rate: f64, // tokens per second
    last_refill: Arc<Mutex<DateTime<Utc>>>,
}

impl RateLimiter {
    fn new(requests_per_minute: u64, window: Duration) -> Self {
        let max_tokens = requests_per_minute as f64;
        let refill_rate = max_tokens / window.as_secs_f64();

        RateLimiter {
            tokens: Arc::new(Mutex::new(max_tokens)),
            max_tokens,
            refill_rate,
            last_refill: Arc::new(Mutex::new(Utc::now())),
        }
    }

    fn allow(&self) -> bool {
        // For simplicity, we'll use a synchronous approach here
        // In a real implementation, this would need to be async
        let mut tokens = self.tokens.try_lock().unwrap();
        let mut last_refill = self.last_refill.try_lock().unwrap();

        let now = Utc::now();
        let elapsed = now.signed_duration_since(*last_refill).num_milliseconds() as f64 / 1000.0;
        *tokens = (*tokens + elapsed * self.refill_rate).min(self.max_tokens);
        *last_refill = now;

        if *tokens >= 1.0 {
            *tokens -= 1.0;
            true
        } else {
            false
        }
    }
}

// Key Manager (ported from Go)
#[derive(Debug, Clone)]
struct KeyManager {
    keys: Arc<Mutex<HashMap<String, KeyDetails>>>,
}

impl KeyManager {
    fn new() -> Self {
        KeyManager {
            keys: Arc::new(Mutex::new(HashMap::new())),
        }
    }

    async fn generate_key(&self, tier: &str, _client_ip: &str) -> Result<String, String> {
        use rand::Rng;
        let mut rng = rand::thread_rng();
        let key_bytes: [u8; 16] = rng.gen();
        let key = format!("key_{}", hex::encode(key_bytes));

        let details = KeyDetails {
            hash: hex::encode(Sha256::digest(key.as_bytes())),
            tier: tier.to_string(),
            created_at: Utc::now(),
            expires_at: Utc::now() + chrono::Duration::days(30),
            request_count: 0,
            rate_limit_remaining: self.get_rate_limit_for_tier(tier),
        };

        let mut keys = self.keys.lock().await;
        keys.insert(key.clone(), details);

        Ok(key)
    }

    async fn validate_key(&self, key: &str) -> Option<KeyDetails> {
        let keys = self.keys.lock().await;
        keys.get(key).cloned()
    }

    fn get_rate_limit_for_tier(&self, tier: &str) -> u32 {
        match tier {
            "free" => 1000,
            "pro" => 10_000,
            "enterprise" => 100_000,
            _ => 1000,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct KeyDetails {
    hash: String,
    tier: String,
    created_at: DateTime<Utc>,
    expires_at: DateTime<Utc>,
    request_count: u64,
    rate_limit_remaining: u32,
}

// Monetization Engine (ported from Go)
#[derive(Debug, Clone)]
struct MonetizationEngine {}

impl MonetizationEngine {
    fn new() -> Self {
        MonetizationEngine {}
    }

    async fn calculate_cost(&self, tier: &str, request_count: u64) -> f64 {
        match tier {
            "free" => 0.0,
            "pro" => request_count as f64 * 0.0001,
            "enterprise" => request_count as f64 * 0.00005,
            _ => 0.0,
        }
    }
}

// Predictive Cache (ported from Go)
#[derive(Clone)]
struct PredictiveCache {
    cache: Arc<Mutex<HashMap<String, CacheEntry>>>,
    predictions: Arc<Mutex<PredictionEngine>>,
    max_size: usize,
    current_size: Arc<Mutex<usize>>,
}

#[derive(Clone)]
struct CacheEntry {
    key: String,
    value: Value,
    created: DateTime<Utc>,
    last_access: DateTime<Utc>,
    access_count: u64,
    prediction: f64,
    ttl: Duration,
}

#[derive(Clone)]
struct PredictionEngine {
    patterns: Arc<Mutex<HashMap<String, AccessPattern>>>,
    ml_model: SimpleMLModel,
    prediction_ttl: Duration,
}

#[derive(Clone)]
struct AccessPattern {
    frequency: HashMap<String, u32>,
    last_accesses: Vec<DateTime<Utc>>,
    trend_score: f64,
}

#[derive(Clone)]
struct SimpleMLModel {}

impl PredictiveCache {
    fn new(max_size: usize) -> Self {
        PredictiveCache {
            cache: Arc::new(Mutex::new(HashMap::new())),
            predictions: Arc::new(Mutex::new(PredictionEngine {
                patterns: Arc::new(Mutex::new(HashMap::new())),
                ml_model: SimpleMLModel {},
                prediction_ttl: Duration::from_secs(300),
            })),
            max_size,
            current_size: Arc::new(Mutex::new(0)),
        }
    }

    async fn get(&self, key: &str) -> Option<Value> {
        let mut cache = self.cache.lock().await;
        if let Some(entry) = cache.get_mut(key) {
            if Utc::now() > entry.created + chrono::Duration::from_std(entry.ttl).unwrap() {
                cache.remove(key);
                return None;
            }
            entry.last_access = Utc::now();
            entry.access_count += 1;
            Some(entry.value.clone())
        } else {
            None
        }
    }

    async fn set(&self, key: String, value: Value) {
        let mut cache = self.cache.lock().await;
        let mut current_size = self.current_size.lock().await;

        if *current_size >= self.max_size {
            self.evict_least_predicted(&mut cache).await;
        }

        let predicted_ttl = self.predictions.lock().await.predict_optimal_ttl(&key).await;
        let entry = CacheEntry {
            key: key.clone(),
            value,
            created: Utc::now(),
            last_access: Utc::now(),
            access_count: 0,
            prediction: 0.0,
            ttl: predicted_ttl,
        };

        cache.insert(key, entry);
        *current_size += 1;
    }

    async fn evict_least_predicted(&self, cache: &mut HashMap<String, CacheEntry>) {
        let mut min_prediction = f64::INFINITY;
        let mut key_to_remove = None;

        for (key, entry) in cache.iter() {
            if entry.prediction < min_prediction {
                min_prediction = entry.prediction;
                key_to_remove = Some(key.clone());
            }
        }

        if let Some(key) = key_to_remove {
            cache.remove(&key);
            let mut current_size = self.current_size.lock().await;
            *current_size -= 1;
        }
    }
}

impl PredictionEngine {
    async fn predict_optimal_ttl(&self, _key: &str) -> Duration {
        // Simple prediction: return 5 minutes for now
        // In production, this would use ML to predict based on access patterns
        Duration::from_secs(300)
    }
}

// Metrics Tracker with labeled Prometheus metrics
#[derive(Clone)]
struct MetricsTracker {
    requests_total: CounterVec,
    request_duration: HistogramVec,
    cache_hits: CounterVec,
    cache_misses: CounterVec,
    active_connections: GaugeVec,
}

impl MetricsTracker {
    fn new() -> Self {
        let requests_total = register_counter_vec!(
            "sprint_requests_total",
            "Total number of requests",
            &["chain", "method", "status"]
        ).unwrap();

        let request_duration = register_histogram_vec!(
            "sprint_request_duration_seconds",
            "Request duration in seconds",
            &["chain", "method"]
        ).unwrap();

        let cache_hits = register_counter_vec!(
            "sprint_cache_hits_total",
            "Total number of cache hits",
            &["chain", "method"]
        ).unwrap();

        let cache_misses = register_counter_vec!(
            "sprint_cache_misses_total",
            "Total number of cache misses",
            &["chain", "method"]
        ).unwrap();

        let active_connections = register_gauge_vec!(
            "sprint_active_connections",
            "Number of active connections",
            &["chain"]
        ).unwrap();

        MetricsTracker {
            requests_total,
            request_duration,
            cache_hits,
            cache_misses,
            active_connections,
        }
    }

    fn increment_requests(&self, chain: &str, method: &str, status: &str) {
        self.requests_total.with_label_values(&[chain, method, status]).inc();
    }

    fn observe_duration(&self, chain: &str, method: &str, duration: f64) {
        self.request_duration.with_label_values(&[chain, method]).observe(duration);
    }

    fn increment_cache_hit(&self, chain: &str, method: &str) {
        self.cache_hits.with_label_values(&[chain, method]).inc();
    }

    fn increment_cache_miss(&self, chain: &str, method: &str) {
        self.cache_misses.with_label_values(&[chain, method]).inc();
    }

    fn set_active_connections(&self, chain: &str, count: f64) {
        self.active_connections.with_label_values(&[chain]).set(count);
    }
}

// Middleware for API key authentication
async fn auth_middleware(req: axum::http::Request<axum::body::Body>, next: axum::middleware::Next) -> Result<axum::response::Response, axum::http::StatusCode> {
    // Simple API key check (in production, use HMAC or JWT)
    let api_key = req.headers().get("x-api-key").and_then(|v| v.to_str().ok());
    if api_key != Some("sprint-api-key") { // Replace with env var in production
        return Err(axum::http::StatusCode::UNAUTHORIZED);
    }
    Ok(next.run(req).await)
}

// UniversalClient (expanded to match more Go methods)
#[derive(Clone)]
struct UniversalClient {
    cfg: Config,
    protocol: ProtocolType,
    peers: Arc<Mutex<HashMap<String, TcpStream>>>,
}

impl UniversalClient {
    async fn new(cfg: Config, protocol: ProtocolType) -> Result<Self, String> {
        Ok(UniversalClient {
            cfg,
            protocol,
            peers: Arc::new(Mutex::new(HashMap::new())),
        })
    }

    async fn connect_to_network(&self) -> Result<(), String> {
        let seeds = self.get_default_seeds();
        let mut success = 0;
        for addr in seeds {
            match tokio::time::timeout(self.cfg.connection_timeout, TcpStream::connect(&addr)).await {
                Ok(Ok(conn)) => {
                    conn.set_nodelay(true).ok();
                    let peer_id = self.generate_peer_id(&addr);
                    self.peers.lock().await.insert(peer_id, conn);
                    info!("Connected to peer: {}", addr);
                    success += 1;
                }
                Ok(Err(e)) => error!("Failed to connect to {}: {}", addr, e),
                Err(_) => warn!("Timeout connecting to {}", addr),
            }
        }
        if success == 0 {
            Err("Failed to connect to any peers".to_string())
        } else {
            Ok(())
        }
    }

    fn get_default_seeds(&self) -> Vec<String> {
        // Allow overrides via env vars: BITCOIN_SEEDS/ETHEREUM_SEEDS/SOLANA_SEEDS (comma-separated host:port)
        let override_key = match self.protocol {
            ProtocolType::Bitcoin => "BITCOIN_SEEDS",
            ProtocolType::Ethereum => "ETHEREUM_SEEDS",
            ProtocolType::Solana => "SOLANA_SEEDS",
        };
        if let Ok(v) = env::var(override_key) {
            let list: Vec<String> = v
                .split(',')
                .map(|s| s.trim().to_string())
                .filter(|s| !s.is_empty())
                .collect();
            if !list.is_empty() {
                return list;
            }
        }
        match self.protocol {
            ProtocolType::Bitcoin => vec![
                "seed.bitcoin.sipa.be:8333".to_string(),
                "dnsseed.bluematt.me:8333".to_string(),
                "dnsseed.bitcoin.dashjr.org:8333".to_string(),
                "seed.bitcoinstats.com:8333".to_string(),
                "seed.bitnodes.io:8333".to_string(),
                "dnsseed.emzy.de:8333".to_string(),
                "seed.bitcoin.jonasschnelli.ch:8333".to_string(),
            ],
            ProtocolType::Ethereum => vec![
                "18.138.108.67:30303".to_string(),
                "3.209.45.79:30303".to_string(),
                "34.255.23.113:30303".to_string(),
                "35.158.244.151:30303".to_string(),
                "52.74.57.123:30303".to_string(),
            ],
            ProtocolType::Solana => vec![
                // Use raw host:port for TCP probing; JSON-RPC is HTTP but initial connectivity
                // checks should avoid URL schemes here.
                // Defaults intentionally empty to avoid noisy connection attempts unless enabled
                // Provide SOLANA_SEEDS env (comma-separated) to set your own list.
            ],
        }
    }

    fn generate_peer_id(&self, address: &str) -> String {
        let mut hasher = Sha256::new();
        hasher.update(address.as_bytes());
        hasher.update(self.protocol.to_string().as_bytes());
        let result = hasher.finalize();
        format!("peer_{:x}", u64::from_be_bytes(result[0..8].try_into().unwrap()))
    }

    async fn get_peer_count(&self) -> usize {
        self.peers.lock().await.len()
    }

    // Potential shutdown hook: currently peers are ephemeral, clear when needed
    async fn shutdown(&self) {
        let mut peers = self.peers.lock().await;
        peers.clear();
    }
}

// Server (expanded with more handlers and components)
#[derive(Clone)]
struct Server {
    cfg: Arc<Config>,
    cache: Cache,
    latency_optimizer: LatencyOptimizer,
    p2p_clients: Arc<Mutex<HashMap<ProtocolType, UniversalClient>>>,
    tier_manager: Arc<TierManager>,
    key_manager: Arc<KeyManager>,
    predictive_cache: Arc<PredictiveCache>,
    metrics: Arc<MetricsTracker>,
}

impl Server {
    async fn new(cfg: Config) -> Self {
        let cfg_arc = Arc::new(cfg.clone());
        let mut p2p_clients = HashMap::new();
    // Build enabled protocols list
    let mut protocols: Vec<ProtocolType> = Vec::new();
    if cfg.enable_bitcoin { protocols.push(ProtocolType::Bitcoin); }
    if cfg.enable_ethereum { protocols.push(ProtocolType::Ethereum); }
    if cfg.enable_solana { protocols.push(ProtocolType::Solana); }

    for protocol in protocols {
            match UniversalClient::new(cfg.clone(), protocol.clone()).await {
                Ok(client) => {
                    p2p_clients.insert(protocol, client);
                }
                Err(e) => error!("Failed to create P2P client for {:?}: {}", protocol, e),
            }
        }

        Server {
            cfg: cfg_arc,
            cache: Cache::new(cfg.cache_size as usize),
            latency_optimizer: LatencyOptimizer::new(Duration::from_millis(100)),
            p2p_clients: Arc::new(Mutex::new(p2p_clients)),
            tier_manager: Arc::new(TierManager::new()),
            key_manager: Arc::new(KeyManager::new()),
            predictive_cache: Arc::new(PredictiveCache::new(cfg.cache_size as usize)),
            metrics: Arc::new(MetricsTracker::new()),
        }
    }

    fn register_routes(&self) -> Router<Server> {
        let protected_routes = Router::new()
            .route("/api/v1/universal/:chain/:method", post(universal_handler))
            .route("/api/v1/latency", get(latency_stats_handler))
            .route("/api/v1/cache", get(cache_stats_handler))
            .layer(middleware::from_fn(auth_middleware));

        let enterprise_routes = Router::new()
            .route("/api/v1/enterprise/entropy/*path", get(enterprise_entropy_handler))
            .route("/system/fingerprint", get(system_fingerprint_handler))
            .route("/system/temperature", get(system_temperature_handler))
            .layer(middleware::from_fn(auth_middleware));

        Router::new()
            .merge(protected_routes)
            .merge(enterprise_routes)
            .route("/health", get(health_handler))
            .route("/metrics", get(metrics_handler))
            .route("/version", get(version_handler))
            .route("/status", get(status_handler))
            .route("/mempool", get(mempool_handler))
            .route("/chains", get(chains_handler))
            .route("/ready", get(ready_handler))
            .route("/generate-key", post(|| async { "Not implemented yet" }))
            .route("/license", get(license_handler))
    }

    async fn start(&self) -> Result<(), Box<dyn std::error::Error>> {
        let app = self.register_routes().with_state(self.clone());

        let addr: SocketAddr = format!("{}:{}", self.cfg.api_host, self.cfg.api_port).parse().unwrap();
        info!("Starting Sprint API server on {}", addr);

        // Connect P2P clients in background
        let p2p_clients_clone = self.p2p_clients.clone();
        tokio::task::spawn(async move {
            let mut clients = p2p_clients_clone.lock().await;
            for (protocol, client) in clients.iter_mut() {
                if let Err(e) = client.connect_to_network().await {
                    match protocol {
                        ProtocolType::Solana => debug!("P2P connect (Solana) not ready: {}", e),
                        _ => error!("P2P connect failed for {:?}: {}", protocol, e),
                    }
                } else {
                    info!("P2P connected for {:?}", protocol);
                }
            }
        });

        // Simplified database init (assuming sqlx or similar; here mock)
        if self.cfg.database_type == "postgres" {
            info!("Database enabled: {}", self.cfg.database_type);
            // In real: connect to DB
        }

        // Rust web server integration (mock exec)
        if self.cfg.rust_web_server_enabled {
            info!("Rust web server enabled");
            // In real: spawn process with Command
        }

        let listener = tokio::net::TcpListener::bind(&addr).await?;
        let shutdown = async {
            // Graceful shutdown on Ctrl+C
            if tokio::signal::ctrl_c().await.is_ok() {
                info!("Shutdown signal received");
            }
        };
        axum::serve(listener, app)
            .with_graceful_shutdown(shutdown)
            .await?;
        Ok(())
    }
}

// Handlers (matching Go's HTTP handlers)
async fn universal_handler(
    state: axum::extract::State<Server>,
    Path((chain, method)): Path<(String, String)>,
    body: Json<Value>,
) -> impl IntoResponse {
    let start = Instant::now();

    // Check predictive cache first
    let cache_key = format!("{}_{}_{}", chain, method, body.to_string());
    if let Some(cached_response) = state.predictive_cache.get(&cache_key).await {
        state.metrics.increment_cache_hit(&chain, &method);
        state.metrics.increment_requests(&chain, &method, "200");
        let duration = start.elapsed().as_secs_f64();
        state.metrics.observe_duration(&chain, &method, duration);
        return (StatusCode::OK, Json(cached_response));
    }

    state.metrics.increment_cache_miss(&chain, &method);

    // Simplified logic
    let response = json!({
        "chain": chain,
        "method": method,
        "data": *body,
        "timestamp": Utc::now().to_rfc3339(),
        "sprint_advantages": {
            "unified_api": "Single endpoint for all chains",
            "predictive_cache": "ML-powered caching",
            "enterprise_security": "Advanced security features",
        }
    });

    // Cache the response
    state.predictive_cache.set(cache_key, response.clone()).await;

    let duration = start.elapsed();
    state.latency_optimizer.track_request(&chain, duration).await;

    if duration > Duration::from_millis(100) {
        warn!("P99 exceeded for {}: {:?}", chain, duration);
    }

    state.metrics.increment_requests(&chain, &method, "200");
    state.metrics.observe_duration(&chain, &method, duration.as_secs_f64());

    (StatusCode::OK, Json(response))
}

async fn latency_stats_handler(
    _state: axum::extract::State<Server>,
) -> impl IntoResponse {
    // Mock stats
    let stats = json!({
        "target_p99": "100ms",
        "current_p99": "85ms",
    });
    (StatusCode::OK, Json(stats))
}

async fn cache_stats_handler(
    state: axum::extract::State<Server>,
) -> impl IntoResponse {
    let items = state.cache.items.lock().await;
    let stats = json!({
        "size": items.len(),
        "max_size": state.cache.max_size,
    });
    (StatusCode::OK, Json(stats))
}

async fn metrics_handler(
    _state: axum::extract::State<Server>,
) -> impl IntoResponse {
    let encoder = TextEncoder::new();
    let metric_families = prometheus::gather();
    let mut buf = Vec::new();
    if let Err(e) = encoder.encode(&metric_families, &mut buf) {
        return (StatusCode::INTERNAL_SERVER_ERROR, Json(json!({"error": e.to_string()}))).into_response();
    }
    let body = String::from_utf8(buf).unwrap_or_default();
    (
        StatusCode::OK,
        [(CONTENT_TYPE, encoder.format_type())],
        body,
    )
        .into_response()
}

async fn health_handler(
    _state: axum::extract::State<Server>,
) -> impl IntoResponse {
    let resp = json!({
        "status": "healthy",
        "timestamp": Utc::now().to_rfc3339(),
        "version": VERSION,
        "service": "sprint-api",
    });
    (StatusCode::OK, Json(resp))
}

async fn version_handler(
    state: axum::extract::State<Server>,
) -> impl IntoResponse {
    let resp = json!({
        "version": VERSION,
        "build": "enterprise",
        "build_time": COMMIT,
        "tier": state.cfg.tier,
        "turbo_mode": state.cfg.tier == "Enterprise",
        "timestamp": Utc::now().to_rfc3339(),
    });
    (StatusCode::OK, Json(resp))
}

async fn status_handler(
    state: axum::extract::State<Server>,
) -> impl IntoResponse {
    let p2p_clients = state.p2p_clients.lock().await;
    let mut connections = 0;
    for client in p2p_clients.values() {
        connections += client.get_peer_count().await;
    }
    let status = json!({
        "server": {
            "uptime": "1h", // Mock
            "version": "2.5.0",
            "tier": state.cfg.tier,
            "status": "running",
        },
        "p2p": {
            "connections": connections,
            "protocols": ["bitcoin", "ethereum", "solana"],
        },
        "cache": {
            "entries": true,
            "size": "dynamic",
        },
    });
    (StatusCode::OK, Json(status))
}

async fn mempool_handler(
    _state: axum::extract::State<Server>,
) -> impl IntoResponse {
    let resp = json!({
        "mempool_size": 100,
        "transactions": ["tx1", "tx2", "tx3"],
        "timestamp": Utc::now().to_rfc3339(),
    });
    (StatusCode::OK, Json(resp))
}

async fn chains_handler(
    _state: axum::extract::State<Server>,
) -> impl IntoResponse {
    let chains = vec!["bitcoin", "ethereum", "solana"];
    let resp = json!({
        "chains": chains,
        "total_chains": chains.len(),
        "unified_api": true,
        "latency_target": "100ms P99",
    });
    (StatusCode::OK, Json(resp))
}

async fn ready_handler(
    state: axum::extract::State<Server>,
) -> impl IntoResponse {
    let p2p_clients = state.p2p_clients.lock().await;
    let mut ready = true;
    for client in p2p_clients.values() {
        if client.get_peer_count().await == 0 {
            ready = false;
            break;
        }
    }
    let status = if ready { "ready" } else { "not ready" };
    let resp = json!({
        "status": status,
        "timestamp": Utc::now().to_rfc3339(),
        "version": VERSION,
        "service": "sprint-api",
    });
    (StatusCode::OK, Json(resp))
}

async fn generate_key_handler(
    state: axum::extract::State<Server>,
) -> impl IntoResponse {
    let tier = "free".to_string(); // Default to free tier
    let client_ip = "127.0.0.1".to_string(); // In production, extract from request

    match state.key_manager.generate_key(&tier, &client_ip).await {
        Ok(key) => {
            let resp = json!({
                "key": key,
                "tier": tier,
                "generated": Utc::now().to_rfc3339(),
                "expires": (Utc::now() + chrono::Duration::days(30)).to_rfc3339(),
            });
            (StatusCode::OK, Json(resp))
        }
        Err(e) => {
            let resp = json!({
                "error": e,
            });
            (StatusCode::INTERNAL_SERVER_ERROR, Json(resp))
        }
    }
}

async fn license_handler(
    _state: axum::extract::State<Server>,
) -> impl IntoResponse {
    let resp = json!({
        "license": {
            "type": "enterprise",
            "valid_until": (Utc::now() + chrono::Duration::days(365)).to_rfc3339(),
            "features": ["unlimited_requests", "enterprise_security", "turbo_mode", "predictive_cache"],
        },
        "compliance": {
            "gdpr_compliant": true,
            "audit_trail": true,
            "data_encryption": true,
        },
    });
    (StatusCode::OK, Json(resp))
}

async fn enterprise_entropy_handler(
    _state: axum::extract::State<Server>,
    Path(path): Path<String>,
) -> impl IntoResponse {
    // Enterprise entropy monitoring endpoint
    let resp = json!({
        "entropy": {
            "level": 0.85,
            "quality": "high",
            "source": "system",
            "timestamp": Utc::now().to_rfc3339(),
        },
        "path": path,
    });
    (StatusCode::OK, Json(resp))
}

async fn system_fingerprint_handler(
    _state: axum::extract::State<Server>,
) -> impl IntoResponse {
    // System fingerprint for enterprise security
    let resp = json!({
        "fingerprint": {
            "system_id": "sprint-enterprise-001",
            "security_level": "enterprise",
            "encryption_enabled": true,
            "audit_enabled": true,
            "timestamp": Utc::now().to_rfc3339(),
        },
    });
    (StatusCode::OK, Json(resp))
}

async fn system_temperature_handler(
    _state: axum::extract::State<Server>,
) -> impl IntoResponse {
    // System temperature monitoring
    let resp = json!({
        "temperature": {
            "cpu": 65.5,
            "memory": 72.3,
            "disk": 45.2,
            "network": 55.8,
            "timestamp": Utc::now().to_rfc3339(),
        },
    });
    (StatusCode::OK, Json(resp))
}

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt::init();
    let cfg = Config::load();
    info!("Starting Sprint API server, tier: {}", cfg.tier);
    info!("Config - Host: {}, Port: {}", cfg.api_host, cfg.api_port);

    let server = Server::new(cfg).await;
    if let Err(e) = server.start().await {
        error!("Server failed to start or crashed: {}", e);
        std::process::exit(1);
    }
}
