// SPDX-License-Identifier: MIT
// Bitcoin Sprint - Storage Verifier Web Server Binary
// Entry point for the REST API server

#[cfg(feature = "web-server")]
use securebuffer::web_server;

#[cfg(feature = "web-server")]
#[actix_web::main]
async fn main() -> std::io::Result<()> {
    env_logger::builder()
        .filter_level(log::LevelFilter::Info)
        .try_init()
        .unwrap_or_else(|_| eprintln!("Logger already initialized"));

    log::info!("Starting Bitcoin Sprint Storage Verifier Web Server...");
    log::info!("Server will be available at http://0.0.0.0:8080");

    web_server::run_server().await
}

#[cfg(not(feature = "web-server"))]
fn main() {
    eprintln!("This binary requires the 'web-server' feature to be enabled.");
    eprintln!("Build with: cargo build --bin storage_verifier_server --features web-server");
    std::process::exit(1);
}
