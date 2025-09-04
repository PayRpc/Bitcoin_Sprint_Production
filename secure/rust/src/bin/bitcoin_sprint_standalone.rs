// SPDX-License-Identifier: MIT
// Bitcoin Sprint - Standalone Axum API Server
// Completely independent of the securebuffer library

use axum::{http::StatusCode, response::IntoResponse, routing::get, Router, Json};
use chrono::{DateTime, Utc};
use serde::Serialize;
use std::env;
use std::net::SocketAddr;
use tracing::{info};

#[derive(Serialize)]
struct HealthResponse {
    status: String,
    timestamp: String,
    version: String,
}

#[derive(Serialize)]
struct BitcoinStatusResponse {
    network: String,
    block_height: u64,
    difficulty: u64,
    hash_rate: String,
    mempool_size: u32,
    timestamp: String,
}

#[derive(Serialize)]
struct NetworkInfoResponse {
    peers_connected: u32,
    peers_total: u32,
    networks: Vec<String>,
    timestamp: String,
}

async fn health_handler() -> impl IntoResponse {
    let response = HealthResponse {
        status: "healthy".to_string(),
        timestamp: Utc::now().to_rfc3339(),
        version: "1.0.0".to_string(),
    };
    (StatusCode::OK, Json(response))
}

async fn bitcoin_status_handler() -> impl IntoResponse {
    let response = BitcoinStatusResponse {
        network: "mainnet".to_string(),
        block_height: 850000,
        difficulty: 1000000000000,
        hash_rate: "500 EH/s".to_string(),
        mempool_size: 25000,
        timestamp: Utc::now().to_rfc3339(),
    };
    (StatusCode::OK, Json(response))
}

async fn network_info_handler() -> impl IntoResponse {
    let response = NetworkInfoResponse {
        peers_connected: 8,
        peers_total: 1250,
        networks: vec!["bitcoin".to_string(), "ethereum".to_string(), "solana".to_string()],
        timestamp: Utc::now().to_rfc3339(),
    };
    (StatusCode::OK, Json(response))
}

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt::init();

    let port = env::var("PORT")
        .unwrap_or_else(|_| "8080".to_string())
        .parse::<u16>()
        .unwrap_or(8080);

    let host = env::var("HOST")
        .unwrap_or_else(|_| "127.0.0.1".to_string());

    info!("🚀 Starting Bitcoin Sprint Standalone API Server");
    info!("📡 Server will be available at: http://{}:{}", host, port);
    info!("🌐 Health check: http://{}:{}/health", host, port);
    info!("₿ Bitcoin status: http://{}:{}/bitcoin/status", host, port);
    info!("🌐 Network info: http://{}:{}/network/info", host, port);

    let app = Router::new()
        .route("/health", get(health_handler))
        .route("/bitcoin/status", get(bitcoin_status_handler))
        .route("/network/info", get(network_info_handler));

    let addr = format!("{}:{}", host, port).parse::<SocketAddr>().unwrap();
    info!("Server starting on {}", addr);

    let listener = tokio::net::TcpListener::bind(addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}
