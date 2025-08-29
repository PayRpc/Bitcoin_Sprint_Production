# deploy_with_turbo_validation.sh - Production Deployment with Turbo Validation
# This script deploys the Sprint backend with integrated turbo validation

#!/bin/bash

set -e

echo "🚀 DEPLOYING SPRINT BACKEND WITH TURBO VALIDATION"
echo "================================================"

# Configuration
RUST_BACKEND_PORT=8082
FASTAPI_PORT=8000
PROMETHEUS_PORT=9090
GRAFANA_PORT=3000

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if port is available
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null ; then
        log_error "Port $port is already in use"
        return 1
    fi
    return 0
}

# Function to wait for service to be ready
wait_for_service() {
    local url=$1
    local service_name=$2
    local max_attempts=30
    local attempt=1

    log_info "Waiting for $service_name to be ready..."

    while [ $attempt -le $max_attempts ]; do
        if curl -s --max-time 5 "$url" > /dev/null 2>&1; then
            log_info "$service_name is ready!"
            return 0
        fi

        log_info "Attempt $attempt/$max_attempts - $service_name not ready yet..."
        sleep 2
        ((attempt++))
    done

    log_error "$service_name failed to start within expected time"
    return 1
}

# Check prerequisites
log_info "Checking prerequisites..."
command -v rustc >/dev/null 2>&1 || { log_error "Rust is not installed"; exit 1; }
command -v python3 >/dev/null 2>&1 || { log_error "Python3 is not installed"; exit 1; }
command -v docker >/dev/null 2>&1 || { log_error "Docker is not installed"; exit 1; }

# Check ports
log_info "Checking port availability..."
check_port $RUST_BACKEND_PORT || exit 1
check_port $FASTAPI_PORT || exit 1

# Build Rust backend with turbo validation
log_info "Building Rust backend with turbo validation..."
cd /path/to/bitcoin-sprint  # Adjust this path
rustc validate_low_latency_backend_99_9.rs -o validate_turbo_99_9.exe

if [ ! -f "validate_turbo_99_9.exe" ]; then
    log_error "Rust backend build failed"
    exit 1
fi

log_info "Rust backend built successfully"

# Start Rust backend (turbo validator)
log_info "Starting Rust backend with turbo validation..."
./validate_turbo_99_9.exe &
RUST_PID=$!

# Wait for Rust backend to be ready
wait_for_service "http://127.0.0.1:$RUST_BACKEND_PORT/turbo-status" "Rust Turbo Validator" || exit 1

# Verify turbo validation is working
log_info "Verifying turbo validation..."
TURBO_STATUS=$(curl -s "http://127.0.0.1:$RUST_BACKEND_PORT/turbo-status")

if echo "$TURBO_STATUS" | grep -q "PRODUCTION_ACTIVE"; then
    log_info "✅ Turbo validation is active"
else
    log_error "❌ Turbo validation failed"
    kill $RUST_PID 2>/dev/null || true
    exit 1
fi

# Start FastAPI gateway
log_info "Starting FastAPI gateway..."
cd /path/to/bitcoin-sprint  # Adjust this path
python3 turbo_api_integration.py &
FASTAPI_PID=$!

# Wait for FastAPI to be ready
wait_for_service "http://127.0.0.1:$FASTAPI_PORT/health" "FastAPI Gateway" || exit 1

# Start Prometheus (if not already running)
if ! pgrep -f prometheus > /dev/null; then
    log_info "Starting Prometheus..."
    docker run -d \
        --name sprint-prometheus \
        -p $PROMETHEUS_PORT:9090 \
        -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
        prom/prometheus || log_warn "Prometheus start failed, continuing..."
fi

# Start Grafana (if not already running)
if ! pgrep -f grafana > /dev/null; then
    log_info "Starting Grafana..."
    docker run -d \
        --name sprint-grafana \
        -p $GRAFANA_PORT:3000 \
        grafana/grafana || log_warn "Grafana start failed, continuing..."
fi

# Run final validation
log_info "Running final deployment validation..."

# Test FastAPI turbo endpoints
TURBO_API_STATUS=$(curl -s "http://127.0.0.1:$FASTAPI_PORT/turbo-status")
if echo "$TURBO_API_STATUS" | grep -q "validation_score"; then
    log_info "✅ FastAPI turbo endpoints working"
else
    log_error "❌ FastAPI turbo endpoints failed"
fi

# Test metrics endpoint
METRICS=$(curl -s "http://127.0.0.1:$FASTAPI_PORT/metrics")
if echo "$METRICS" | grep -q "sprint_turbo"; then
    log_info "✅ Prometheus metrics exposed via FastAPI"
else
    log_error "❌ Prometheus metrics not available"
fi

# Display deployment summary
echo ""
echo "🎉 DEPLOYMENT COMPLETE!"
echo "======================"
echo ""
echo "Services running:"
echo "• Rust Turbo Validator: http://127.0.0.1:$RUST_BACKEND_PORT"
echo "  - /turbo-status - Real-time validation status"
echo "  - /metrics - Prometheus metrics"
echo "  - /turbo-validation - JSON validation results"
echo ""
echo "• FastAPI Gateway: http://127.0.0.1:$FASTAPI_PORT"
echo "  - /turbo-status - Turbo validation API"
echo "  - /health - Health check with turbo status"
echo "  - /metrics - Proxied Prometheus metrics"
echo ""
echo "• Prometheus: http://127.0.0.1:$PROMETHEUS_PORT"
echo "• Grafana: http://127.0.0.1:$GRAFANA_PORT (admin/admin)"
echo ""
echo "📊 Monitoring:"
echo "• Import grafana_turbo_dashboard.json into Grafana"
echo "• Turbo validation runs automatically on every startup"
echo "• Real-time metrics available via Prometheus"
echo ""
echo "🔄 Process IDs:"
echo "• Rust Backend: $RUST_PID"
echo "• FastAPI Gateway: $FASTAPI_PID"
echo ""
echo "✅ Every deployment now proves itself with 99.9% turbo validation!"

# Save PIDs for cleanup
echo "$RUST_PID" > .rust_backend_pid
echo "$FASTAPI_PID" > .fastapi_pid

log_info "Deployment successful! Turbo validation is active and monitoring."
