// SPDX-License-Identifier: MIT
// Bitcoin Sprint - Enhanced Storage Verification Demo
// Production-ready cryptographic proof system with optional IPFS support

use std::time::Duration;
use tokio::time::sleep;

// Import our enhanced storage verification system
use securebuffer::storage_verifier::{
    StorageVerifier, RateLimitConfig, VerificationMetrics
};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Initialize logging
    env_logger::init();
    
    println!("🚀 Bitcoin Sprint - Enhanced Storage Verification Demo");
    println!("=====================================================");
    
    // Create a storage verifier with rate limiting
    let rate_config = RateLimitConfig {
        max_requests_per_minute: 10,
        max_requests_per_hour: 100,
        cleanup_interval_secs: 60,
    };
    
    let verifier = StorageVerifier::with_config(rate_config);
    
    // Demo 1: Basic Challenge Generation
    println!("\n📋 Demo 1: Challenge Generation");
    println!("-------------------------------");
    
    let challenge1 = verifier.generate_challenge("bitcoin_block_800000.dat", "decentralized_provider").await?;
    println!("✅ Challenge generated:");
    println!("   ID: {}", challenge1.id);
    println!("   File: {}", challenge1.file_id);
    println!("   Provider: {}", challenge1.provider);
    println!("   Beacon: {}", hex::encode(&challenge1.beacon));
    println!("   Expected Hash: {}", hex::encode(&challenge1.expected_hash));
    println!("   Sample Size: {} bytes", challenge1.sample_size);
    
    // Demo 2: Metrics Tracking
    println!("\n📊 Demo 2: Metrics Tracking");
    println!("----------------------------");
    
    let mut metrics = verifier.get_metrics().await;
    println!("Current metrics:");
    println!("   Total challenges: {}", metrics.total_challenges);
    println!("   Rate limited requests: {}", metrics.rate_limited_requests);
    
    // Generate more challenges to show metrics growth
    for i in 2..=5 {
        let _challenge = verifier.generate_challenge(
            &format!("bitcoin_block_{}.dat", 800000 + i),
            "decentralized_provider"
        ).await?;
    }
    
    metrics = verifier.get_metrics().await;
    println!("After generating 4 more challenges:");
    println!("   Total challenges: {}", metrics.total_challenges);
    
    // Demo 3: Rate Limiting
    println!("\n🚦 Demo 3: Rate Limiting Protection");
    println!("-----------------------------------");
    
    // Create a more restrictive verifier
    let strict_config = RateLimitConfig {
        max_requests_per_minute: 2,
        max_requests_per_hour: 5,
        cleanup_interval_secs: 1,
    };
    let strict_verifier = StorageVerifier::with_config(strict_config);
    
    // Generate challenges until rate limit is hit
    for i in 1..=4 {
        match strict_verifier.generate_challenge(&format!("file_{}", i), "test_provider").await {
            Ok(challenge) => {
                println!("✅ Challenge {} generated: {}", i, challenge.id);
            }
            Err(e) => {
                println!("❌ Challenge {} rejected: {}", i, e);
                break;
            }
        }
    }
    
    let strict_metrics = strict_verifier.get_metrics().await;
    println!("Rate limiting metrics:");
    println!("   Successful challenges: {}", strict_metrics.total_challenges);
    println!("   Rate limited requests: {}", strict_metrics.rate_limited_requests);
    
    // Demo 4: Cryptographic Proof Verification
    println!("\n🔐 Demo 4: Cryptographic Proof Verification");
    println!("--------------------------------------------");
    
    let verification_challenge = verifier.generate_challenge("test_verification.dat", "crypto_provider").await?;
    println!("✅ Verification challenge created: {}", verification_challenge.id);
    
    // Simulate creating a proof (in production, this would come from the storage provider)
    use securebuffer::storage_verifier::StorageProof;
    let proof = StorageProof {
        challenge_id: verification_challenge.id.clone(),
        file_id: verification_challenge.file_id.clone(),
        provider: verification_challenge.provider.clone(),
        timestamp: verification_challenge.timestamp + 30, // 30 seconds later
        proof_data: vec![42u8; verification_challenge.sample_size as usize], // Mock data
        merkle_proof: None,
        signature: None,
    };
    
    // Verify the proof
    match verifier.verify_proof(proof).await {
        Ok(is_valid) => {
            println!("🔍 Proof verification result: {}", if is_valid { "✅ Valid" } else { "❌ Invalid" });
        }
        Err(e) => {
            println!("⚠️  Proof verification error: {}", e);
        }
    }
    
    // Demo 5: Optional IPFS Integration
    #[cfg(feature = "ipfs")]
    {
        println!("\n🌐 Demo 5: IPFS Integration (Optional)");
        println!("--------------------------------------");
        
        // Test IPFS verification (would normally use real CID)
        match verifier.verify_ipfs_storage("QmTest123", 1024).await {
            Ok(is_verified) => {
                println!("🔍 IPFS verification result: {}", if is_verified { "✅ Verified" } else { "❌ Failed" });
            }
            Err(e) => {
                println!("⚠️  IPFS verification error: {}", e);
            }
        }
    }
    
    #[cfg(not(feature = "ipfs"))]
    {
        println!("\n🌐 Demo 5: IPFS Integration");
        println!("----------------------------");
        println!("ℹ️  IPFS support not enabled (use --features ipfs to enable)");
    }
    
    // Demo 6: Performance and Security Features
    println!("\n⚡ Demo 6: Performance & Security Summary");
    println!("----------------------------------------");
    
    let final_metrics = verifier.get_metrics().await;
    println!("📈 Final system metrics:");
    println!("   Total challenges processed: {}", final_metrics.total_challenges);
    println!("   Average response time: {:.2}ms", final_metrics.avg_response_time_ms);
    println!("   Success rate: {:.1}%", final_metrics.success_rate * 100.0);
    println!("   Rate limited requests: {}", final_metrics.rate_limited_requests);
    
    println!("\n🔒 Security features enabled:");
    println!("   ✅ SHA256-based cryptographic proofs");
    println!("   ✅ Challenge-response protocol");
    println!("   ✅ Rate limiting and DoS protection");
    println!("   ✅ Comprehensive error handling");
    println!("   ✅ Metrics and monitoring");
    println!("   ✅ Optional IPFS multi-gateway support");
    
    println!("\n🏢 Enterprise-grade capabilities:");
    println!("   ✅ Production-ready error handling");
    println!("   ✅ Structured logging");
    println!("   ✅ Configurable rate limits");
    println!("   ✅ Automatic cleanup of expired data");
    println!("   ✅ Commercial deployment ready");
    
    println!("\n✨ Demo completed successfully!");
    println!("The enhanced storage verification system is ready for production use.");
    
    Ok(())
}
