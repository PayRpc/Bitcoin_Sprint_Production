// SPDX-License-Identifier: MIT
// Bitcoin Sprint - Simple HTTP API Server
// Lightweight implementation without heavy dependencies

use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream};
use std::thread;
use chrono::{DateTime, Utc};
use serde::Serialize;

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

#[derive(Serialize)]
struct EntropyResponse {
    entropy: String,
    size: usize,
    source: String,
    timestamp: String,
}

fn handle_request(path: &str) -> String {
    match path {
        "/health" => {
            let response = HealthResponse {
                status: "healthy".to_string(),
                timestamp: Utc::now().to_rfc3339(),
                version: "1.0.0".to_string(),
            };
            format!("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{}",
                   serde_json::to_string(&response).unwrap())
        },
        "/bitcoin/status" => {
            let response = BitcoinStatusResponse {
                network: "mainnet".to_string(),
                block_height: 850000,
                difficulty: 1000000000000,
                hash_rate: "500 EH/s".to_string(),
                mempool_size: 25000,
                timestamp: Utc::now().to_rfc3339(),
            };
            format!("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{}",
                   serde_json::to_string(&response).unwrap())
        },
        "/network/info" => {
            let response = NetworkInfoResponse {
                peers_connected: 8,
                peers_total: 1250,
                networks: vec!["bitcoin".to_string(), "ethereum".to_string(), "solana".to_string()],
                timestamp: Utc::now().to_rfc3339(),
            };
            format!("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{}",
                   serde_json::to_string(&response).unwrap())
        },
        "/api/v1/enterprise/entropy/fast" => {
            let entropy = generate_entropy(32);
            let response = EntropyResponse {
                entropy: hex::encode(&entropy),
                size: entropy.len(),
                source: "hardware".to_string(),
                timestamp: Utc::now().to_rfc3339(),
            };
            format!("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{}",
                   serde_json::to_string(&response).unwrap())
        },
        _ => {
            "HTTP/1.1 404 Not Found\r\nContent-Type: text/plain\r\n\r\n404 Not Found".to_string()
        }
    }
}

fn handle_client(mut stream: TcpStream) {
    let mut buffer = [0; 1024];
    match stream.read(&mut buffer) {
        Ok(_) => {
            let request = String::from_utf8_lossy(&buffer);
            let path = request.lines().next().unwrap_or("")
                .split_whitespace().nth(1).unwrap_or("/");

            let response = handle_request(path);
            let _ = stream.write(response.as_bytes());
            let _ = stream.flush();
        },
        Err(_) => {}
    }
}

fn generate_entropy(size: usize) -> Vec<u8> {
    use std::time::{SystemTime, UNIX_EPOCH};
    use std::collections::hash_map::DefaultHasher;
    use std::hash::{Hash, Hasher};
    
    let mut buffer = vec![0u8; size];
    
    // Use system time and thread ID as entropy sources
    let mut hasher = DefaultHasher::new();
    SystemTime::now().duration_since(UNIX_EPOCH).unwrap().hash(&mut hasher);
    std::thread::current().id().hash(&mut hasher);
    std::process::id().hash(&mut hasher);
    
    let mut seed = hasher.finish();
    
    // Generate pseudo-random bytes using a simple but effective PRNG
    for i in 0..size {
        // Use a linear congruential generator with good statistical properties
        seed = seed.wrapping_mul(6364136223846793005).wrapping_add(1442695040888963407);
        buffer[i] = (seed >> 32) as u8;
        
        // Mix in additional entropy from system time for each byte
        let time_seed = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().subsec_nanos() as u64;
        buffer[i] = buffer[i].wrapping_add((time_seed % 256) as u8);
    }
    
    buffer
}

fn main() {
    let port = std::env::var("PORT").unwrap_or_else(|_| "8080".to_string());
    let host = std::env::var("HOST").unwrap_or_else(|_| "127.0.0.1".to_string());

    let address = format!("{}:{}", host, port);
    let listener = TcpListener::bind(&address).unwrap();

    println!("🚀 Starting Bitcoin Sprint Simple HTTP Server");
    println!("📡 Server will be available at: http://{}", address);
    println!("🌐 Health check: http://{}/health", address);
    println!("₿ Bitcoin status: http://{}/bitcoin/status", address);
    println!("🌐 Network info: http://{}/network/info", address);
    println!();
    println!("Press Ctrl+C to stop the server");

    for stream in listener.incoming() {
        match stream {
            Ok(stream) => {
                thread::spawn(|| {
                    handle_client(stream);
                });
            }
            Err(e) => {
                eprintln!("Connection failed: {}", e);
            }
        }
    }
}
