// Example main.rs showing different SecureBuffer usage patterns
use anyhow::Result;
use securebuffer::SecureBuffer;
use tracing::info;
use std::time::Duration;

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize structured logging
    tracing_subscriber::fmt()
        .init();

    info!("Starting Bitcoin Sprint SecureBuffer demonstration");

    // Pattern 1: Basic SecureBuffer usage
    run_basic_example().await?;

    // Pattern 2: Multiple secure buffers
    run_multi_buffer_example().await?;

    // Pattern 3: SecureBuffer with operations
    run_operations_example().await?;

    Ok(())
}

/// Basic SecureBuffer demonstration
async fn run_basic_example() -> Result<()> {
    info!("=== Basic SecureBuffer Example ===");

    // Create a secure buffer for storing sensitive data (like private keys)
    let mut buffer = SecureBuffer::new(32)?; // 32 bytes for Bitcoin private key
    
    // Example Bitcoin private key data (for demonstration only)
    let sample_data = b"demo_private_key_32_bytes_here!!";
    buffer.copy_from_slice(sample_data)?;
    
    info!("SecureBuffer created and populated with sample data");
    info!("Buffer size: {} bytes", buffer.len());
    
    // Demonstrate secure access
    if let Some(data) = buffer.as_slice() {
        info!("Successfully accessed secure data (length: {})", data.len());
        // In real usage, you'd use this data for cryptographic operations
    }
    
    // Buffer will be automatically zeroized when dropped
    drop(buffer);
    info!("SecureBuffer safely disposed (memory zeroized)");
    
    Ok(())
}

/// Multiple SecureBuffer demonstration  
async fn run_multi_buffer_example() -> Result<()> {
    info!("=== Multi-Buffer Example ===");

    // Create multiple secure buffers for different purposes
    let mut key_buffer = SecureBuffer::new(32)?;  // Private key
    let mut sig_buffer = SecureBuffer::new(64)?;  // Signature
    let mut hash_buffer = SecureBuffer::new(32)?; // Hash
    
    // Populate with sample data
    let sample_key = b"bitcoin_private_key_32_bytes_pad";
    let sample_sig = b"bitcoin_signature_64_bytes_padding_here_for_demonstration_use";
    let sample_hash = b"bitcoin_hash_32_bytes_padding!!";
    
    key_buffer.copy_from_slice(sample_key)?;
    sig_buffer.copy_from_slice(sample_sig)?;
    hash_buffer.copy_from_slice(sample_hash)?;
    
    info!("Created 3 SecureBuffers:");
    info!("  - Key buffer: {} bytes", key_buffer.len());
    info!("  - Signature buffer: {} bytes", sig_buffer.len());  
    info!("  - Hash buffer: {} bytes", hash_buffer.len());
    
    // Simulate some work
    tokio::time::sleep(Duration::from_millis(100)).await;
    
    info!("Multi-buffer operations completed");
    
    Ok(())
}

/// SecureBuffer operations demonstration
async fn run_operations_example() -> Result<()> {
    info!("=== Operations Example ===");

    let mut buffer = SecureBuffer::new(64)?;
    
    // Example: Simulate Bitcoin Core integration workflow
    info!("Simulating Bitcoin Core integration workflow...");
    
    // Step 1: Generate/receive data
    let bitcoin_data = b"bitcoin_core_rpc_response_data_for_secure_storage_example_here";
    buffer.copy_from_slice(bitcoin_data)?;
    info!("Step 1: Stored Bitcoin Core RPC data securely");
    
    // Step 2: Process data (simulated)
    tokio::time::sleep(Duration::from_millis(50)).await;
    info!("Step 2: Processing secure data...");
    
    // Step 3: Use data (access without copying)
    if let Some(secure_data) = buffer.as_slice() {
        info!("Step 3: Accessed secure data for Bitcoin Core integration (length: {})", secure_data.len());
        // In real usage: send to Bitcoin Core RPC, sign transactions, etc.
    }
    
    info!("Bitcoin Core integration workflow completed");
    info!("SecureBuffer will be zeroized on drop for security");
    
    Ok(())
}

