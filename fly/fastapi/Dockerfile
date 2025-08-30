# Multi-stage build for Bitcoin Sprint (Go backend + FastAPI gateway)
FROM golang:1.21-alpine AS go-builder

# Set working directory for Go build
WORKDIR /go/src/bitcoin-sprint

# Copy Go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY config/ ./config/

# Build the Go backend
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o sprintd ./cmd/sprintd

# Python stage for FastAPI
# Use Python 3.13.7 for FastAPI (matches local development environment)
FROM python:3.13.7-slim as fastapi-builder

# Install system dependencies (including Rust for pydantic-core compilation)
RUN apt-get update && apt-get install -y \
    gcc \
    curl \
    build-essential \
    && rm -rf /var/lib/apt/lists/* \
    && curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y \
    && . ~/.cargo/env

# Create virtual environment
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Copy Python requirements
COPY Bitcoin_Sprint_fastapi/fastapi-gateway/requirements.txt .
RUN . ~/.cargo/env && pip install --no-cache-dir -r requirements.txt

# Production stage
FROM python:3.13.7-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create app user
RUN useradd --create-home --shell /bin/bash app

# Set working directory
WORKDIR /app

# Copy Go binary from builder
COPY --from=go-builder /go/src/bitcoin-sprint/sprintd /usr/local/bin/sprintd

# Copy Python virtual environment
COPY --from=fastapi-builder /opt/venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Copy FastAPI application code
COPY Bitcoin_Sprint_fastapi/fastapi-gateway/ .

# Copy configuration files
COPY config/ ./config/
COPY .env* ./

# Create necessary directories
RUN mkdir -p /app/logs /app/data /app/config

# Make Go binary executable
RUN chmod +x /usr/local/bin/sprintd

# Copy startup script
COPY fly/fastapi/start.sh /usr/local/bin/start.sh
RUN chmod +x /usr/local/bin/start.sh

# Change ownership
RUN chown -R app:app /app

# Switch to non-root user
USER app

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:$PORT/health || exit 1

# Expose port
EXPOSE 8080

# Default command - use startup script
CMD ["/usr/local/bin/start.sh"]
