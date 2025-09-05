//! Example endpoint for exposing PQC validator metrics to Prometheus

use actix_web::{web, App, HttpServer, Responder, HttpResponse};
use prometheus::{Registry, TextEncoder, Encoder};

// Import our TurboValidator
use turbo_validator::TurboValidator;

async fn metrics_handler(validator: web::Data<TurboValidator>) -> impl Responder {
    // Get metrics in Prometheus format
    let metrics = validator.prometheus_metrics();
    HttpResponse::Ok()
        .content_type("text/plain; charset=utf-8")
        .body(metrics)
}

async fn entropy_hybrid_endpoint(
    validator: web::Data<TurboValidator>,
    payload: web::Json<EntropyRequest>,
) -> impl Responder {
    // Generate entropy receipt (demo example)
    let receipt = validator.generate_entropy_hybrid_receipt(
        payload.beacon_round, 
        &payload.attestation, 
        &payload.proof_hash, 
        &payload.verifier_id
    );
    
    // Serialize to JSON
    match TurboValidator::serialize_receipt_json(&receipt) {
        Ok(json) => HttpResponse::Ok()
            .content_type("application/json")
            .body(json),
        Err(_) => HttpResponse::InternalServerError()
            .body("Failed to serialize receipt")
    }
}

// Request model for entropy hybrid endpoint
#[derive(serde::Deserialize)]
struct EntropyRequest {
    beacon_round: u64,
    attestation: String,
    proof_hash: String,
    verifier_id: String,
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    // Create TurboValidator with Prometheus metrics
    let validator = TurboValidator::default();
    
    // Shared app state
    let app_data = web::Data::new(validator);
    
    println!("Starting PQC validator API server on http://127.0.0.1:9093");
    
    // Start web server
    HttpServer::new(move || {
        App::new()
            .app_data(app_data.clone())
            .route("/metrics", web::get().to(metrics_handler))
            .route("/entropy/hybrid", web::post().to(entropy_hybrid_endpoint))
    })
    .bind("127.0.0.1:9093")?
    .run()
    .await
}
