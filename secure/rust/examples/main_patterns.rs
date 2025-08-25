// Example main.rs showing different SecureChannelPool usage patterns
use anyhow::Result;
use secure_channel_improved::SecureChannelPool;
use tracing::{info, warn, error};
use std::time::Duration;
use std::sync::Arc;

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize structured logging
    tracing_subscriber::fmt()
        .with_env_filter("info,secure_channel_improved=debug")
        .init();

    info!("Starting Bitcoin Sprint with SecureChannel pool");

    // Pattern 1: Single pool with all features
    run_full_featured_example().await?;

    // Pattern 2: Multiple pools for different endpoints
    // run_multi_pool_example().await?;

    // Pattern 3: Pool without metrics (for embedded/testing)
    // run_minimal_example().await?;

    Ok(())
}

/// Full-featured pool with cleanup and metrics
async fn run_full_featured_example() -> Result<()> {
    info!("=== Full Featured Pool Example ===");

    // Create pool with custom configuration
    let pool = Arc::new(
        SecureChannelPool::builder("relay.bitcoin-sprint.inc:443")
            .with_namespace("btc_sprint")
            .with_max_connections(100)
            .with_min_idle(10)
            .with_max_latency_ms(500)
            .with_cleanup_interval(Duration::from_secs(300)) // 5 minutes
            .with_metrics_port(9090)
            .with_metrics_host("0.0.0.0")
            .build()?
    );

    // Explicitly spawn background tasks
    let pool_cleanup = pool.clone();
    let cleanup_handle = tokio::spawn(async move {
        info!("Background cleanup task started");
        pool_cleanup.run_cleanup_task().await;
    });

    let pool_metrics = pool.clone();
    let metrics_handle = tokio::spawn(async move {
        info!("Metrics server starting on http://0.0.0.0:9090");
        if let Err(e) = pool_metrics.run_metrics_task().await {
            error!("Metrics server failed: {}", e);
        }
    });

    // Business logic using the pool
    let pool_worker = pool.clone();
    let worker_handle = tokio::spawn(async move {
        loop {
            match pool_worker.get_connection().await {
                Ok(mut conn) => {
                    // Simulate Bitcoin RPC call
                    if let Err(e) = conn.write_all(b"getblockchaininfo\n").await {
                        warn!("Failed to send RPC command: {}", e);
                        continue;
                    }

                    let mut response = vec![0u8; 4096];
                    match conn.read(&mut response).await {
                        Ok(bytes_read) => {
                            info!("Received {} bytes from Bitcoin node", bytes_read);
                        }
                        Err(e) => warn!("Failed to read RPC response: {}", e),
                    }
                }
                Err(e) => {
                    warn!("Failed to get connection: {}", e);
                    tokio::time::sleep(Duration::from_secs(1)).await;
                }
            }

            tokio::time::sleep(Duration::from_secs(10)).await;
        }
    });

    info!("All tasks started. Available endpoints:");
    info!("  - http://localhost:9090/metrics (Prometheus)");
    info!("  - http://localhost:9090/status/connections (JSON)");
    info!("  - http://localhost:9090/healthz (Health)");

    // Wait for shutdown signal
    tokio::signal::ctrl_c().await?;
    info!("Received shutdown signal");

    // Graceful shutdown (abort background tasks)
    cleanup_handle.abort();
    metrics_handle.abort();
    worker_handle.abort();

    info!("Shutdown complete");
    Ok(())
}

/// Multiple pools for different Bitcoin networks or endpoints
async fn run_multi_pool_example() -> Result<()> {
    info!("=== Multi-Pool Example ===");

    // Primary Bitcoin node pool
    let primary_pool = Arc::new(
        SecureChannelPool::builder("primary.bitcoin-sprint.inc:443")
            .with_namespace("btc_primary")
            .with_max_connections(50)
            .with_metrics_port(9090)
            .build()?
    );

    // Backup Bitcoin node pool
    let backup_pool = Arc::new(
        SecureChannelPool::builder("backup.bitcoin-sprint.inc:443")
            .with_namespace("btc_backup")
            .with_max_connections(25)
            .with_metrics_port(9091) // Different port!
            .build()?
    );

    // Start cleanup for both pools
    let primary_cleanup = primary_pool.clone();
    tokio::spawn(async move {
        primary_cleanup.run_cleanup_task().await;
    });

    let backup_cleanup = backup_pool.clone();
    tokio::spawn(async move {
        backup_cleanup.run_cleanup_task().await;
    });

    // Start metrics servers on different ports
    let primary_metrics = primary_pool.clone();
    tokio::spawn(async move {
        if let Err(e) = primary_metrics.run_metrics_task().await {
            error!("Primary metrics server failed: {}", e);
        }
    });

    let backup_metrics = backup_pool.clone();
    tokio::spawn(async move {
        if let Err(e) = backup_metrics.run_metrics_task().await {
            error!("Backup metrics server failed: {}", e);
        }
    });

    // Business logic with failover
    tokio::spawn(async move {
        loop {
            // Try primary first
            match primary_pool.get_connection().await {
                Ok(mut conn) => {
                    info!("Using primary connection");
                    if let Err(e) = conn.write_all(b"ping\n").await {
                        warn!("Primary connection failed: {}", e);
                    }
                }
                Err(_) => {
                    // Fallback to backup
                    match backup_pool.get_connection().await {
                        Ok(mut conn) => {
                            info!("Using backup connection");
                            if let Err(e) = conn.write_all(b"ping\n").await {
                                warn!("Backup connection failed: {}", e);
                            }
                        }
                        Err(e) => warn!("Both pools failed: {}", e),
                    }
                }
            }

            tokio::time::sleep(Duration::from_secs(5)).await;
        }
    });

    info!("Multi-pool setup complete:");
    info!("  - Primary metrics: http://localhost:9090");
    info!("  - Backup metrics:  http://localhost:9091");

    tokio::signal::ctrl_c().await?;
    Ok(())
}

/// Minimal pool without metrics (useful for testing or embedded environments)
async fn run_minimal_example() -> Result<()> {
    info!("=== Minimal Pool Example (No Metrics) ===");

    let pool = Arc::new(
        SecureChannelPool::builder("test.bitcoin-sprint.inc:443")
            .with_namespace("btc_test")
            .with_max_connections(5)
            .with_min_idle(1)
            .build()?
    );

    // Only run cleanup, no metrics server
    let pool_cleanup = pool.clone();
    tokio::spawn(async move {
        pool_cleanup.run_cleanup_task().await;
    });

    // Simple connection test
    match pool.get_connection().await {
        Ok(mut conn) => {
            info!("Connection test successful");
            let _ = conn.write_all(b"test\n").await;
        }
        Err(e) => warn!("Connection test failed: {}", e),
    }

    info!("Minimal pool setup complete (no metrics server)");
    tokio::time::sleep(Duration::from_secs(5)).await;
    Ok(())
}

/// Example configuration for different environments
fn get_pool_config_for_env(env: &str) -> SecureChannelPool {
    match env {
        "production" => {
            SecureChannelPool::builder("prod.bitcoin-sprint.inc:443")
                .with_namespace("btc_prod")
                .with_max_connections(200)
                .with_min_idle(20)
                .with_max_latency_ms(300)
                .with_cleanup_interval(Duration::from_secs(180))
                .with_metrics_port(9090)
                .build()
                .expect("Failed to build production pool")
        }
        "staging" => {
            SecureChannelPool::builder("staging.bitcoin-sprint.inc:443")
                .with_namespace("btc_staging")
                .with_max_connections(50)
                .with_min_idle(5)
                .with_max_latency_ms(500)
                .with_metrics_port(9091)
                .build()
                .expect("Failed to build staging pool")
        }
        "development" => {
            SecureChannelPool::builder("localhost:8443")
                .with_namespace("btc_dev")
                .with_max_connections(10)
                .with_min_idle(2)
                .with_max_latency_ms(1000)
                .with_metrics_port(9092)
                .build()
                .expect("Failed to build development pool")
        }
        _ => panic!("Unknown environment: {}", env),
    }
}
