#!/bin/bash

# Bitcoin Sprint Production Startup Script
# This script manages both the Go backend (sprintd) and FastAPI gateway

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}" >&2
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

# Function to check if a port is in use
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null ; then
        return 0
    else
        return 1
    fi
}

# Function to wait for service to be ready
wait_for_service() {
    local url=$1
    local timeout=${2:-30}
    local interval=${3:-2}

    log "Waiting for service at $url (timeout: ${timeout}s)"

    local count=0
    while [ $count -lt $timeout ]; do
        if curl -f -s "$url" >/dev/null 2>&1; then
            log "Service is ready at $url"
            return 0
        fi
        sleep $interval
        count=$((count + interval))
    done

    error "Service at $url failed to start within ${timeout} seconds"
    return 1
}

# Function to start Go backend
start_backend() {
    log "Starting Go backend (sprintd)..."

    # Check if backend is already running
    if check_port 9090; then
        warn "Backend appears to already be running on port 9090"
        return 0
    fi

    # Start backend in background
    /usr/local/bin/sprintd \
        --port 9090 \
        --log-level info \
        --config /app/config \
        --data-dir /app/data \
        > /app/logs/backend.log 2>&1 &

    local backend_pid=$!
    echo $backend_pid > /tmp/backend.pid

    log "Backend started with PID: $backend_pid"

    # Wait for backend to be ready
    if ! wait_for_service "http://localhost:9090/health" 30; then
        error "Backend failed to start properly"
        return 1
    fi

    return 0
}

# Function to start FastAPI gateway
start_fastapi() {
    log "Starting FastAPI gateway..."

    # Check if FastAPI is already running
    if check_port 8080; then
        warn "FastAPI appears to already be running on port 8080"
        return 0
    fi

    # Set environment variables
    export PYTHONPATH=/app
    export BACKEND_URL=http://localhost:9090
    export ENVIRONMENT=production
    export LOG_LEVEL=info

    # Start FastAPI in background
    uvicorn main:app \
        --host 0.0.0.0 \
        --port 8080 \
        --workers 1 \
        --log-level info \
        --access-log \
        > /app/logs/fastapi.log 2>&1 &

    local fastapi_pid=$!
    echo $fastapi_pid > /tmp/fastapi.pid

    log "FastAPI started with PID: $fastapi_pid"

    # Wait for FastAPI to be ready
    if ! wait_for_service "http://localhost:8080/health" 30; then
        error "FastAPI failed to start properly"
        return 1
    fi

    return 0
}

# Function to stop services
stop_services() {
    log "Stopping services..."

    # Stop FastAPI
    if [ -f /tmp/fastapi.pid ]; then
        local fastapi_pid=$(cat /tmp/fastapi.pid)
        if kill -0 $fastapi_pid 2>/dev/null; then
            log "Stopping FastAPI (PID: $fastapi_pid)"
            kill $fastapi_pid
            wait $fastapi_pid 2>/dev/null || true
        fi
        rm -f /tmp/fastapi.pid
    fi

    # Stop backend
    if [ -f /tmp/backend.pid ]; then
        local backend_pid=$(cat /tmp/backend.pid)
        if kill -0 $backend_pid 2>/dev/null; then
            log "Stopping backend (PID: $backend_pid)"
            kill $backend_pid
            wait $backend_pid 2>/dev/null || true
        fi
        rm -f /tmp/backend.pid
    fi

    log "All services stopped"
}

# Function to check service health
check_health() {
    local service=$1
    local url=$2

    if curl -f -s "$url" >/dev/null 2>&1; then
        log "$service is healthy"
        return 0
    else
        warn "$service is not responding"
        return 1
    fi
}

# Main startup sequence
main() {
    log "Bitcoin Sprint Production Startup"
    log "================================="

    # Create log directory if it doesn't exist
    mkdir -p /app/logs

    # Trap signals for graceful shutdown
    trap 'error "Received signal, shutting down..."; stop_services; exit 1' INT TERM

    # Start services
    if start_backend && start_fastapi; then
        log "All services started successfully"
        log "Go Backend: http://localhost:9090"
        log "FastAPI Gateway: http://localhost:8080"
        log "Health Check: http://localhost:8080/health"

        # Health monitoring loop
        while true; do
            sleep 60

            # Check backend health
            check_health "Backend" "http://localhost:9090/health"

            # Check FastAPI health
            check_health "FastAPI" "http://localhost:8080/health"
        done
    else
        error "Failed to start one or more services"
        stop_services
        exit 1
    fi
}

# Run main function
main "$@"
