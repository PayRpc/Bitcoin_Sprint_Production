# Bitcoin Sprint Multi-Chain Relay Platform
# Multi-stage build for optimized production container

# ===== BUILDER STAGE =====
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    gcc \
    musl-dev \
    pkgconfig \
    git \
    make \
    curl \
    unzip

# Set working directory
WORKDIR /app

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=$(git describe --tags --always --dirty) -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -a -installsuffix cgo \
    -o sprintd \
    ./cmd/sprintd

# Build enterprise demo
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -a -installsuffix cgo \
    -o enterprise-demo \
    ./examples/enterprise_api_demo.go

# ===== RUNTIME STAGE =====
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    jq \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1001 -S sprint && \
    adduser -u 1001 -S sprint -G sprint

# Create necessary directories
RUN mkdir -p /app/data /app/logs /app/config && \
    chown -R sprint:sprint /app

# Copy binaries from builder
COPY --from=builder --chown=sprint:sprint /app/sprintd /app/sprintd
COPY --from=builder --chown=sprint:sprint /app/enterprise-demo /app/enterprise-demo

# Copy configuration files
COPY --chown=sprint:sprint bitcoin.conf /app/config/
COPY --chown=sprint:sprint bitcoin-testnet.conf /app/config/

# Copy web assets if they exist
COPY --chown=sprint:sprint dashboard.html entropy-monitor.html provider-selection.html /app/web/

# Set working directory
WORKDIR /app

# Switch to non-root user
USER sprint

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Expose ports
EXPOSE 8080 8081 9090 6060

# Environment variables with defaults
ENV SPRINT_TIER=enterprise \
    SPRINT_API_HOST=0.0.0.0 \
    SPRINT_API_PORT=8080 \
    SPRINT_ADMIN_PORT=8081 \
    SPRINT_METRICS_PORT=9090 \
    SPRINT_PPROF_PORT=6060 \
    SPRINT_LICENSE_KEY="" \
    SPRINT_TURBO_MODE=true \
    SPRINT_ENTERPRISE_FEATURES=true \
    SPRINT_LOG_LEVEL=info \
    SPRINT_DATA_DIR=/app/data \
    SPRINT_CONFIG_DIR=/app/config

# Default command
CMD ["./sprintd"]

# Metadata
LABEL maintainer="Bitcoin Sprint Team" \
      description="Bitcoin Sprint Multi-Chain Relay Platform" \
      version="1.0.0" \
      org.opencontainers.image.title="Bitcoin Sprint" \
      org.opencontainers.image.description="Enterprise-grade multi-chain blockchain relay platform" \
      org.opencontainers.image.vendor="Bitcoin Sprint" \
      org.opencontainers.image.licenses="Enterprise"
