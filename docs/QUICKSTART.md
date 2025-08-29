# Multi-Chain Sprint README - Quick Start Guide

## Overview
Multi-Chain Sprint is a high-performance blockchain infrastructure platform supporting Bitcoin, Ethereum, Solana, Cosmos, Polkadot and other major networks. This guide will help you get the backend up and running quickly.

## Requirements
- Windows/Linux/macOS
- Go 1.19 or higher
- PowerShell (for Windows) or bash (for Linux/macOS)

## Quick Start

### 1. Starting the Backend
Use our backend manager script:

```powershell
# Start the backend with enterprise tier features
.\backend-manager.ps1 start -Tier enterprise -Port 9090 -NoZMQ -Optimized

# Check backend status
.\backend-manager.ps1 status

# Stop the backend
.\backend-manager.ps1 stop

# Restart the backend
.\backend-manager.ps1 restart
```

### 2. API Endpoints
The Bitcoin Sprint backend provides several API endpoints:

- Health Check: `http://localhost:9090/health`
- Status: `http://localhost:9090/status`
- Version: `http://localhost:9090/version`
- Turbo Status: `http://localhost:9090/turbo-status`

Authenticated endpoints (requires API key):
- Latest Block: `http://localhost:9090/latest`
- Mempool Metrics: `http://localhost:9090/metrics`
- Cache Status: `http://localhost:9090/cache-status`

### 3. Running Tests
Use our integration test script to verify the backend is working properly:

```powershell
.\integration-test-backend.ps1 -DurationSeconds 30 -Concurrency 3
```

### 4. Configuring
The Bitcoin Sprint backend can be configured using environment variables:

- `API_PORT` - The port to listen on (default: 8080)
- `API_HOST` - The host to bind to (default: 0.0.0.0)
- `TIER` - License tier: free, pro, enterprise (default: free)
- `API_KEY` - API key for authenticated endpoints (default: changeme)
- `OPTIMIZE_SYSTEM` - Enable system optimizations (default: true)

### 5. Troubleshooting
If the backend isn't responding:

1. Check if the process is running: `Get-Process -Name bitcoin-sprint-backend`
2. Verify the port is listening: `netstat -ano | findstr :9090`
3. Check logs for errors
4. Restart the backend: `.\backend-manager.ps1 restart`

## Architecture

Bitcoin Sprint consists of several components:

1. **Core Backend** (Go) - The main API server and Bitcoin blockchain processor
2. **Web Dashboard** (Next.js) - Optional web interface for monitoring and management
3. **Storage Verifier** (Rust) - Optional component for advanced storage verification

The core backend is the essential component that must be running for Bitcoin Sprint to function.
