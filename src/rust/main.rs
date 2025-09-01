// main.rs
// Example usage of the netkit networking library
// Demonstrates Bitcoin Sprint networking capabilities

use bitcoin_sprint_storage_verifier::netkit;
use std::time::Duration;
use tokio;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("🚀 Bitcoin Sprint Netkit Demo");
    println!("==============================");

    // Example 1: Happy-Eyeballs TCP connection
    println!("\n1️⃣ Testing Happy-Eyeballs TCP connection...");
    match netkit::connect_happy("google.com:80", Duration::from_secs(5)).await {
        Ok(tcp) => {
            println!("✅ Successfully connected to google.com:80");
            drop(tcp); // Clean up
        }
        Err(e) => println!("❌ TCP connection failed: {}", e),
    }

    // Example 2: TLS connection with proper certificates
    println!("\n2️⃣ Testing TLS connection...");
    match netkit::connect_tls("httpbin.org", 443, Duration::from_secs(10)).await {
        Ok(mut tls) => {
            println!("✅ Successfully connected to httpbin.org:443 with TLS");

            // Example HTTP request
            let request = b"GET /get HTTP/1.1\r\nHost: httpbin.org\r\nConnection: close\r\n\r\n";
            match netkit::write_all_deadline(&mut tls, request, Duration::from_secs(3)).await {
                Ok(_) => println!("✅ Sent HTTP request successfully"),
                Err(e) => println!("❌ Failed to send request: {}", e),
            }

            // Try to read response
            let mut buf = [0u8; 1024];
            match netkit::read_exact_deadline(&mut tls, &mut buf[..100], Duration::from_secs(3)).await {
                Ok(_) => println!("✅ Read response successfully"),
                Err(e) => println!("❌ Failed to read response: {}", e),
            }
        }
        Err(e) => println!("❌ TLS connection failed: {}", e),
    }

    // Example 3: Frame padding for consistent packet sizes
    println!("\n3️⃣ Testing frame padding...");
    let original = vec![1, 2, 3, 4, 5];
    let padded = netkit::pad_frame(original.clone(), 8);
    println!("Original: {:?} (len: {})", original, original.len());
    println!("Padded:   {:?} (len: {})", padded, padded.len());

    println!("\n🎉 Netkit demo complete!");
    println!("💡 Ready to use in your Bitcoin Sprint P2P networking");

    Ok(())
}
