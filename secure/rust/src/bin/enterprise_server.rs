use securebuffer::enterprise_web_server::{run_enterprise_server, ValidateStorageRequest, ValidateStorageResponse};
use securebuffer::storage_verifier::StorageVerifier;
use std::env;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    env_logger::init();

    // Get port from environment or use default
    let port = env::var("PORT")
        .unwrap_or_else(|_| "8443".to_string())
        .parse::<u16>()
        .unwrap_or(8443);

    println!("🚀 Starting Bitcoin Sprint Enterprise Storage Validation Service");
    println!("📡 Server will be available at: https://localhost:{}", port);
    println!("🌐 Web interface: https://localhost:{}/web/enterprise-storage-validation.html", port);
    println!("📊 API endpoints:");
    println!("  POST /api/validate-storage - Validate storage");
    println!("  GET  /api/subscription - Get subscription info");
    println!("  GET  /api/analytics - Get analytics (Professional+)");
    println!("  GET  /health - Health check");
    println!();

    // Create storage verifier
    let verifier = StorageVerifier::new();

    // Run the enterprise server
    run_enterprise_server(verifier, port).await?;

    Ok(())
}
