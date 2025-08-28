#!/bin/bash

# Bitcoin Sprint - Storage Verification Service Build Script
# This script helps build and run the enhanced storage verification service

set -e

echo "🚀 Bitcoin Sprint - Enhanced Storage Verification Service"
echo "========================================================"

# Check if Rust is installed
if ! command -v cargo &> /dev/null; then
    echo "❌ Rust/Cargo is not installed. Please install Rust first:"
    echo "   curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh"
    exit 1
fi

# Check Rust version
RUST_VERSION=$(cargo --version | cut -d' ' -f2)
echo "✅ Rust version: $RUST_VERSION"

# Function to build the service
build_service() {
    echo ""
    echo "🔨 Building Storage Verification Service..."

    if [ "$1" = "release" ]; then
        echo "   Building in release mode (optimized)..."
        cargo build --release
        echo "✅ Release build completed: target/release/bitcoin-sprint-storage-verifier"
    else
        echo "   Building in debug mode..."
        cargo build
        echo "✅ Debug build completed: target/debug/bitcoin-sprint-storage-verifier"
    fi
}

# Function to run the service
run_service() {
    echo ""
    echo "🏃 Running Storage Verification Service..."

    if [ "$1" = "release" ]; then
        echo "   Starting in release mode..."
        cargo run --release
    else
        echo "   Starting in debug mode..."
        cargo run
    fi
}

# Function to run tests
run_tests() {
    echo ""
    echo "🧪 Running tests..."

    if [ "$1" = "integration" ]; then
        echo "   Running integration tests..."
        cargo test --test integration
    else
        echo "   Running unit tests..."
        cargo test
    fi
}

# Function to check dependencies
check_dependencies() {
    echo ""
    echo "🔍 Checking dependencies..."

    # Check for required dependencies
    local deps=("reqwest" "tokio" "serde" "actix-web" "uuid" "backoff")
    local missing_deps=()

    for dep in "${deps[@]}"; do
        if ! cargo tree | grep -q "$dep"; then
            missing_deps+=("$dep")
        fi
    done

    if [ ${#missing_deps[@]} -ne 0 ]; then
        echo "⚠️  Missing dependencies: ${missing_deps[*]}"
        echo "   Run 'cargo build' to install them"
    else
        echo "✅ All dependencies are available"
    fi
}

# Function to show help
show_help() {
    echo ""
    echo "📖 Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  build     Build the service (default: debug)"
    echo "  release   Build the service in release mode"
    echo "  run       Run the service (default: debug)"
    echo "  run-rel   Run the service in release mode"
    echo "  test      Run unit tests"
    echo "  test-int  Run integration tests"
    echo "  check     Check dependencies"
    echo "  clean     Clean build artifacts"
    echo "  help      Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 build          # Build in debug mode"
    echo "  $0 release        # Build in release mode"
    echo "  $0 run            # Run in debug mode"
    echo "  $0 run-rel        # Run in release mode"
    echo "  $0 test           # Run unit tests"
    echo "  $0 check          # Check dependencies"
    echo ""
}

# Function to clean build artifacts
clean_build() {
    echo ""
    echo "🧹 Cleaning build artifacts..."
    cargo clean
    echo "✅ Build artifacts cleaned"
}

# Main script logic
case "${1:-help}" in
    "build")
        build_service
        ;;
    "release")
        build_service release
        ;;
    "run")
        run_service
        ;;
    "run-rel")
        run_service release
        ;;
    "test")
        run_tests
        ;;
    "test-int")
        run_tests integration
        ;;
    "check")
        check_dependencies
        ;;
    "clean")
        clean_build
        ;;
    "help"|*)
        show_help
        ;;
esac

echo ""
echo "🎉 Operation completed successfully!"
echo ""
echo "📊 Service Information:"
echo "   - Main file: storage_verification_service.rs"
echo "   - Config file: Cargo.toml"
echo "   - Documentation: STORAGE_VERIFICATION_SERVICE_README.md"
echo "   - Default port: 8080"
echo "   - Health check: http://localhost:8080/health"
echo "   - Metrics: http://localhost:8080/metrics"
echo ""
echo "🔧 Connection Management Features:"
echo "   - HTTP connection pooling with reqwest"
echo "   - Circuit breaker pattern implementation"
echo "   - Exponential backoff retry logic"
echo "   - Configurable timeouts and limits"
echo "   - Real-time health monitoring"
echo ""
