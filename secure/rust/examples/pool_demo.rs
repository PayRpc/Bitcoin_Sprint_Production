use anyhow::Result;
use secure_channel_improved::SecureChannelPool;
use tracing::{info, warn};
use std::time::Duration;
use std::sync::Arc;

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize tracing
    tracing_subscriber::fmt::init();

    // Create pool using builder pattern with custom configuration
    let pool = Arc::new(
        SecureChannelPool::builder("relay.bitcoin-sprint.inc:443")
            .with_namespace("btc_sprint")
            .with_max_connections(50)
            .with_min_idle(5)
            .with_max_latency_ms(300)
            .with_metrics_port(9090)
            .with_metrics_host("0.0.0.0")
            .with_cleanup_interval(Duration::from_secs(180)) // 3 minutes
            .build()?
    );

    info!("SecureChannelPool created with custom configuration");

    // Explicitly spawn background tasks (you choose what to run!)
    let pool_cleanup = pool.clone();
    tokio::spawn(async move {
        info!("Starting cleanup task...");
        pool_cleanup.run_cleanup_task().await;
    });

    let pool_metrics = pool.clone();
    tokio::spawn(async move {
        info!("Starting metrics server...");
        if let Err(e) = pool_metrics.run_metrics_task().await {
            warn!("Metrics server failed: {}", e);
        }
    });

    info!("Background tasks started");
    info!("Metrics available at:");
    info!("  - http://localhost:9090/metrics (Prometheus)");
    info!("  - http://localhost:9090/status/connections (JSON pool status)");
    info!("  - http://localhost:9090/healthz (Health check)");

    // Example worker task using the pool
    let pool_worker = pool.clone();
    let worker_task = tokio::spawn(async move {
        let mut iteration = 0;
        loop {
            iteration += 1;
            info!("Worker iteration #{}", iteration);

            match pool_worker.get_connection().await {
                Ok(mut conn) => {
                    info!("Successfully got connection from pool");
                    
                    // Simulate some work
                    match conn.write_all(b"PING\n").await {
                        Ok(_) => {
                            let mut buf = [0u8; 1024];
                            match conn.read(&mut buf).await {
                                Ok(bytes_read) => {
                                    info!("Read {} bytes from connection", bytes_read);
                                }
                                Err(e) => warn!("Read failed: {}", e),
                            }
                        }
                        Err(e) => warn!("Write failed: {}", e),
                    }

                    // Connection automatically returned to pool when dropped
                }
                Err(e) => {
                    warn!("Failed to get connection: {}", e);
                }
            }

            // Wait before next iteration
            tokio::time::sleep(Duration::from_secs(5)).await;

            // Stop after 20 iterations for demo
            if iteration >= 20 {
                break;
            }
        }
        info!("Worker task completed");
    });

    // You could also run without the worker (just pool + metrics)
    // Or run multiple pools on different ports:
    /*
    let secondary_pool = Arc::new(
        SecureChannelPool::builder("backup.bitcoin-sprint.inc:443")
            .with_namespace("btc_sprint_backup")
            .with_metrics_port(9091)  // Different port!
            .build()?
    );
    */

    // Wait for worker to complete or Ctrl+C
    tokio::select! {
        _ = worker_task => {
            info!("Worker finished, but metrics server continues...");
        }
        _ = tokio::signal::ctrl_c() => {
            info!("Received Ctrl+C, shutting down...");
        }
    }

    info!("Demo completed. Background tasks continue running...");
    
    // In a real application, you might want to:
    // 1. Gracefully shutdown the pool
    // 2. Stop the metrics server
    // 3. Wait for all connections to close
    
    // For demo, keep metrics server alive briefly
    tokio::time::sleep(Duration::from_secs(10)).await;
    info!("Shutdown complete");
    
    Ok(())
}
