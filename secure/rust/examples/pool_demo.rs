use anyhow::Result;
use securebuffer::SecureBuffer;
use tracing::{info, warn};
use std::time::Duration;

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize tracing
    tracing_subscriber::fmt::init();

    info!("Starting Bitcoin Sprint SecureBuffer Pool Demo");

    // Create multiple secure buffers (simulating a pool)
    let mut buffers = Vec::new();
    
    // Create a pool of 5 secure buffers for Bitcoin operations
    for i in 0..5 {
        let buffer = SecureBuffer::new(64)?; // 64 bytes for various Bitcoin data
        buffers.push(buffer);
        info!("Created SecureBuffer #{} (64 bytes)", i + 1);
    }

    info!("SecureBuffer pool created with {} buffers", buffers.len());
    info!("Pool ready for Bitcoin Core integration operations");

    // Simulate Bitcoin Core integration usage
    demonstrate_bitcoin_core_integration(&mut buffers).await?;

    // Demonstrate concurrent access simulation
    demonstrate_concurrent_operations(&mut buffers).await?;

    info!("Demo completed. All SecureBuffers will be securely zeroized.");
    
    Ok(())
}

/// Demonstrate Bitcoin Core integration patterns
async fn demonstrate_bitcoin_core_integration(buffers: &mut Vec<SecureBuffer>) -> Result<()> {
    info!("=== Bitcoin Core Integration Demo ===");
    
    // Simulate different types of Bitcoin data
    let bitcoin_data_types = [
        ("Private Key", b"bitcoin_private_key_32_bytes_secure_storage_demo_padding_here!" as &[u8]),
        ("Transaction Hash", b"bitcoin_transaction_hash_32_bytes_secure_demo_padding_here!!" as &[u8]),
        ("Signature", b"bitcoin_signature_data_64_bytes_secure_storage_demonstration!" as &[u8]),
        ("RPC Response", b"bitcoin_core_rpc_response_secure_temporary_storage_demo_data!" as &[u8]),
        ("Wallet Seed", b"bitcoin_wallet_seed_phrase_secure_storage_demo_padding_here!!" as &[u8]),
    ];
    
    for (i, (data_type, sample_data)) in bitcoin_data_types.iter().enumerate() {
        if let Some(buffer) = buffers.get_mut(i) {
            buffer.copy_from_slice(*sample_data)?;
            info!("Buffer #{}: Stored {} securely", i + 1, data_type);
            
            // Simulate processing delay
            tokio::time::sleep(Duration::from_millis(100)).await;
            
            // Demonstrate secure access
            if let Some(_secure_data) = buffer.as_slice() {
                info!("Buffer #{}: {} ready for Bitcoin Core operations", i + 1, data_type);
            }
        }
    }
    
    info!("Bitcoin Core integration demo completed");
    Ok(())
}

/// Demonstrate concurrent operations simulation
async fn demonstrate_concurrent_operations(buffers: &mut Vec<SecureBuffer>) -> Result<()> {
    info!("=== Concurrent Operations Demo ===");
    
    // Simulate concurrent Bitcoin operations
    let tasks = (0..buffers.len()).map(|i| {
        tokio::spawn(async move {
            info!("Worker #{}: Processing Bitcoin operation", i + 1);
            
            // Simulate work
            tokio::time::sleep(Duration::from_millis(200 + i as u64 * 50)).await;
            
            info!("Worker #{}: Bitcoin operation completed", i + 1);
        })
    }).collect::<Vec<_>>();
    
    // Wait for all workers to complete
    for (i, task) in tasks.into_iter().enumerate() {
        match task.await {
            Ok(_) => info!("Worker #{}: Task completed successfully", i + 1),
            Err(e) => warn!("Worker #{}: Task failed: {}", i + 1, e),
        }
    }
    
    info!("All concurrent operations completed");
    Ok(())
}

