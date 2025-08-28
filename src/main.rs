use actix_web::{web, App, HttpServer, Responder, HttpResponse, middleware, Result};
use serde::{Serialize, Deserialize};
use std::sync::Arc;
use tokio::sync::Mutex;
use std::time::{SystemTime, UNIX_EPOCH, Duration, Instant};
use std::collections::HashMap;
use log::info;
use uuid::Uuid;
use reqwest::ClientBuilder;
use prometheus::{Encoder, TextEncoder, register_counter, register_histogram, Counter, Histogram};

// --- Metrics ---
lazy_static::lazy_static! {
    static ref HTTP_REQUESTS_TOTAL: Counter = register_counter!(
        "sprint_api_requests_total",
        "Total number of API requests"
    ).expect("Can't create metrics");

    static ref HTTP_REQUEST_DURATION: Histogram = register_histogram!(
        "sprint_api_request_duration_seconds",
        "Request duration in seconds"
    ).expect("Can't create metrics");
}

// --- Data Structures ---
#[derive(Serialize, Deserialize, Clone)]
struct HealthStatus {
    status: String,
    timestamp: u64,
    uptime_seconds: u64,
    version: String,
}

#[derive(Serialize, Deserialize, Clone)]
struct APIResponse<T> {
    success: bool,
    data: Option<T>,
    error: Option<String>,
    timestamp: u64,
}

#[derive(Clone)]
struct AppState {
    start_time: Instant,
    request_count: Arc<Mutex<u64>>,
}

// --- API Handlers ---
async fn health_check(data: web::Data<AppState>) -> Result<impl Responder> {
    let uptime = data.start_time.elapsed().as_secs();
    let timestamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs();

    HTTP_REQUESTS_TOTAL.inc();

    let health = HealthStatus {
        status: "healthy".to_string(),
        timestamp,
        uptime_seconds: uptime,
        version: env!("CARGO_PKG_VERSION").to_string(),
    };

    let response = APIResponse {
        success: true,
        data: Some(health),
        error: None,
        timestamp,
    };

    Ok(HttpResponse::Ok().json(response))
}

async fn api_status(data: web::Data<AppState>) -> Result<impl Responder> {
    let timer = HTTP_REQUEST_DURATION.start_timer();
    let count = *data.request_count.lock().await;
    let timestamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs();

    HTTP_REQUESTS_TOTAL.inc();

    let status = serde_json::json!({
        "service": "Bitcoin Sprint API",
        "version": env!("CARGO_PKG_VERSION"),
        "status": "running",
        "requests_served": count,
        "uptime_seconds": data.start_time.elapsed().as_secs(),
        "timestamp": timestamp
    });

    let response = APIResponse {
        success: true,
        data: Some(status),
        error: None,
        timestamp,
    };

    timer.observe_duration();
    Ok(HttpResponse::Ok().json(response))
}

async fn metrics() -> Result<impl Responder> {
    let encoder = TextEncoder::new();
    let metric_families = prometheus::gather();
    let mut buffer = Vec::new();

    encoder.encode(&metric_families, &mut buffer).unwrap();
    Ok(HttpResponse::Ok()
        .content_type("text/plain; charset=utf-8")
        .body(String::from_utf8(buffer).unwrap()))
}

async fn storage_verification(
    query: web::Query<HashMap<String, String>>,
    _data: web::Data<AppState>
) -> Result<impl Responder> {
    let timer = HTTP_REQUEST_DURATION.start_timer();
    let timestamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs();

    HTTP_REQUESTS_TOTAL.inc();

    // Simulate storage verification logic
    let provider = query.get("provider").unwrap_or(&"ipfs".to_string()).clone();
    let file_id = query.get("file_id").unwrap_or(&"demo-file".to_string()).clone();

    info!("Storage verification request: provider={}, file_id={}", provider, file_id);

    let result = serde_json::json!({
        "provider": provider,
        "file_id": file_id,
        "verified": true,
        "timestamp": timestamp,
        "proof": format!("proof-{}-{}", provider, Uuid::new_v4())
    });

    let response = APIResponse {
        success: true,
        data: Some(result),
        error: None,
        timestamp,
    };

    timer.observe_duration();
    Ok(HttpResponse::Ok().json(response))
}

// --- Main Function ---
#[actix_web::main]
async fn main() -> std::io::Result<()> {
    // Initialize logging
    env_logger::init();

    info!("🚀 Starting Bitcoin Sprint API Server v{}", env!("CARGO_PKG_VERSION"));

    // Initialize application state
    let app_state = web::Data::new(AppState {
        start_time: Instant::now(),
        request_count: Arc::new(Mutex::new(0)),
    });

    // Configure HTTP client with connection pooling
    let _client = ClientBuilder::new()
        .pool_max_idle_per_host(10)
        .pool_idle_timeout(Duration::from_secs(90))
        .timeout(Duration::from_secs(30))
        .build()
        .expect("Failed to build HTTP client");

    info!("📊 Metrics server starting on http://0.0.0.0:9090/metrics");
    info!("🌐 API server starting on http://0.0.0.0:8080");

    // For now, just run the main API server
    // Metrics can be accessed through the main server
    HttpServer::new(move || {
        App::new()
            .app_data(app_state.clone())
            .wrap(middleware::Logger::default())
            .route("/health", web::get().to(health_check))
            .route("/api/v1/status", web::get().to(api_status))
            .route("/api/v1/storage/verify", web::get().to(storage_verification))
            .route("/metrics", web::get().to(metrics))
            .route("/", web::get().to(|| async {
                HttpResponse::Ok().json(serde_json::json!({
                    "service": "Bitcoin Sprint API",
                    "version": env!("CARGO_PKG_VERSION"),
                    "endpoints": {
                        "health": "/health",
                        "status": "/api/v1/status",
                        "storage_verify": "/api/v1/storage/verify",
                        "metrics": "/metrics"
                    }
                }))
            }))
    })
    .bind("0.0.0.0:8080")?
    .run()
    .await
}
