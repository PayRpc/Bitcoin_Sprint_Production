#!/bin/bash
# Bitcoin Sprint Integration Test Script
# Tests Rust SecureBuffer integration with Go main application

echo "🧪 Bitcoin Sprint Integration Test"
echo "=================================="

# Stop any running instances
echo "🔄 Stopping existing processes..."
taskkill /f /im bitcoin-sprint*.exe 2>/dev/null || echo "No existing processes"

# Set test environment
export LICENSE_KEY="DEMO_LICENSE_BYPASS"
export PEER_SECRET="demo_peer_secret_123"

echo ""
echo "🏗️  Building with Rust SecureBuffer integration..."

# Build with CGO and Rust
cd "C:\Projects\Bitcoin_Sprint_final_1\BItcoin_Sprint"
export PATH="C:\msys64\mingw64\bin:$PATH"
export CGO_ENABLED=1
export CC=gcc
export CXX=g++

if go build -tags cgo -o bitcoin-sprint-test.exe ./cmd/sprint; then
    echo "✅ Build successful"
else
    echo "❌ Build failed"
    exit 1
fi

echo ""
echo "🔍 Checking Rust library..."
if [ -f "secure/rust/target/release/securebuffer.dll" ]; then
    echo "✅ Rust SecureBuffer library found"
    ls -la secure/rust/target/release/securebuffer.*
else
    echo "❌ Rust library missing"
    exit 1
fi

echo ""
echo "🚀 Testing application startup..."

# Start in background
./bitcoin-sprint-test.exe &
APP_PID=$!
sleep 3

# Test health endpoint
echo "🌐 Testing health endpoint..."
if curl -s http://localhost:8080/api/v1/status | grep -q "memory_protection"; then
    echo "✅ Health endpoint responding"
    echo "🔒 Memory protection status:"
    curl -s http://localhost:8080/api/v1/status | jq '.memory_protection' 2>/dev/null || echo "Raw response received"
else
    echo "❌ Health endpoint not responding"
fi

# Cleanup
echo ""
echo "🧹 Cleaning up..."
kill $APP_PID 2>/dev/null
sleep 1

echo "✅ Integration test complete!"
