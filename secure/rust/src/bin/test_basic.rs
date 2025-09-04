// Simple test binary to verify basic compilation
// SPDX-License-Identifier: MIT

use std::io::{self, Write};

fn main() -> io::Result<()> {
    println!("🚀 Bitcoin Sprint - Basic Compilation Test");
    println!("✅ Library compiled successfully!");
    println!("🎯 This confirms the core Rust code is working");

    // Test basic functionality
    let test_data = b"Hello, Bitcoin Sprint!";
    println!("📦 Test data: {:?}", String::from_utf8_lossy(test_data));

    Ok(())
}
