# Multi-Chain Sprint — Enterprise Blockchain Infrastructure

[![CI](https://github.com/PayRpc/BItcoin_Sprint/actions/workflows/ci.yml/badge.svg)](https://github.com/PayRpc/BItcoin_Sprint/actions/workflows/ci.yml)
[![Windows CGO](https://github.com/PayRpc/BItcoin_Sprint/actions/workflows/windows-cgo.yml/badge.svg)](https://github.com/PayRpc/BItcoin_Sprint/actions/workflows/windows-cgo.yml)

## 🎯 QUICK START - Streamlined Python 3.13 Setup

**Too big? Not connected?** Here's the simplified version:

### One-Click Startup (Windows)
```cmd
# Double-click to start everything:
start-all.bat
```

### Services Overview
| Service | Port | Status |
|---------|------|--------|
| **FastAPI Gateway** | 8000 | ✅ Python 3.13 |
| **Next.js Frontend** | 3002 | ✅ Running |
| **Go Backend** | 8080 | ✅ Connected |
| **Grafana** | 3000 | ✅ Monitoring |

### Python 3.13 Everywhere
- ✅ FastAPI Gateway: Python 3.13 venv
- ✅ Fly Deployment: Python 3.13 Docker
- ✅ System: Python 3.13 available
- ✅ All dependencies: Compatible with Python 3.13

---

**Enterprise-grade multi-chain blockchain infrastructure** that transforms how businesses interact with Bitcoin, Ethereum, Solana, Cosmos, Polkadot and other major blockchains. We provide real-time block monitoring, secure API access, and enterprise-level integrations that scale from individual developers to large financial institutions.

## 🚀 Production Readiness Status

### ✅ CURRENT STATUS: PARTIAL PRODUCTION READY

**Ready for Bitcoin + Ethereum Production Deployment**

- ✅ **Bitcoin P2P**: Fully operational with active peer connections
- ✅ **Ethereum P2P**: Fully operational with active peer connections
- ⚠️ **Solana P2P**: Requires gossip protocol implementation
- ✅ **API Endpoints**: All core endpoints functional
- ✅ **Security**: Enterprise-grade with Rust memory protection
- ✅ **Monitoring**: Comprehensive health checks and diagnostics

### 📊 Production Readiness Score: 70/100

**What's Working:**
- Bitcoin and Ethereum P2P connectivity established
- Professional API infrastructure with rate limiting
- Enterprise security with Rust-powered memory protection
- Comprehensive monitoring and health checks
- Docker containerization ready

**What Needs Work:**
- Solana gossip protocol implementation (requires specialized P2P client)
- Full multi-chain SLA capabilities

### 🚀 Deployment Recommendation

**IMMEDIATE NEXT STEP: Deploy Bitcoin + Ethereum System**

The system is **production-ready for Bitcoin and Ethereum** with enterprise-grade reliability. Deploy now and serve customers needing Bitcoin/Ethereum connectivity while Solana integration is completed.

```bash
# Quick production deployment
docker-compose up -d bitcoin-core geth
./start-system.ps1
```

**Access Points:**
- **Status Dashboard**: http://localhost:8080/status
- **Production Readiness**: http://localhost:8080/readiness
- **API Documentation**: http://localhost:8080/docs

## 📖 Documentationi-Chain Sprint — Enterprise Blockchain Infrastructure

[![CI](https://github.com/PayRpc/BItcoin_Sprint/actions/workflows/ci.yml/badge.svg)](https://github.com/PayRpc/BItcoin_Sprint/actions/workflows/ci.yml)
[![Windows CGO](https://github.com/PayRpc/BItcoin_Sprint/actions/workflows/windows-cgo.yml/badge.svg)](https://github.com/PayRpc/BItcoin_Sprint/actions/workflows/windows-cgo.yml)

**Enterprise-grade multi-chain blockchain infrastructure** that transforms how businesses interact with Bitcoin, Ethereum, Solana, Cosmos, Polkadot and other major blockchains. We provide real-time block monitoring, secure API access, and enterprise-level integrations that scale from individual developers to large financial institutions.

## 🚀 What We Offer

### Service Tiers
- **🆓 FREE**: Basic API access, rate-limited endpoints, community support
- **💼 PRO**: 5x higher rate limits, priority authentication, enhanced monitoring, email support
- **🏆 ENTERPRISE**: Unlimited requests, 99.9% uptime SLA, 24/7 support, custom integrations

### Key Features
- ⚡ **Real-time block detection** with sub-second latency
- 🔒 **Enterprise-grade security** with Rust-powered memory protection  
- 📡 **Scalable API infrastructure** from prototype to production
- 🛠 **Professional Bitcoin Core integration** without the complexity

**[📋 Complete Service Overview →](WHAT_WE_OFFER.md)**

## 📖 Documentation

- **[📚 Complete API Documentation](API.md)** - REST endpoints, authentication, rate limits
- **[🏗 Architecture Guide](ARCHITECTURE.md)** - Hub-and-spoke security model
- **[🌐 Web Documentation](web/pages/docs/index.tsx)** - Interactive docs with examples
- **[📋 What We Offer](WHAT_WE_OFFER.md)** - Comprehensive service overview

## Project Structure

```text
Bitcoin Sprint/
├── cmd/sprint/          # Application entry point
│   └── main.go         # Complete Bitcoin Sprint application  
├── internal/           # Internal Go packages
│   ├── sprint/         # Core Sprint functionality
│   ├── rpc/            # RPC client helpers
│   ├── peer/           # P2P networking helpers
│   └── secure/         # Go FFI wrapper for Rust SecureBuffer
│       ├── securebuffer.go     # CGO integration with error handling
│       ├── securebuffer_test.go # Integration tests
│       └── example.go          # Usage examples
├── secure/             # Rust secure memory implementation
│   ├── rust/           # Rust SecureBuffer crate
│   │   ├── src/
│   │   │   ├── lib.rs          # FFI exports
│   │   │   └── securebuffer.rs # Core implementation
│   │   ├── include/
│   │   │   └── securebuffer.h  # C header for FFI
│   │   ├── Cargo.toml          # Rust configuration
│   │   └── target/release/     # Compiled artifacts
│   └── SETUP.md        # SecureBuffer setup guide
├── build.ps1           # Automated build script
├── check-setup.ps1     # Development environment checker
├── test_secure.go      # SecureBuffer integration test
├── go.mod              # Root workspace module
└── INTEGRATION.md      # Complete FFI setup guide
```

## Quick Start

### Prerequisites

Bitcoin Sprint uses CGO to link with Rust libraries. You need a C compiler:

**Windows**: MSYS2/MinGW, Visual Studio Build Tools, or TDM-GCC  
**Linux**: `sudo apt install build-essential`  
**macOS**: `xcode-select --install`

On Windows, the easiest path is to run the helper:

```powershell
.\tools\\dev-win.ps1          # Auto-detects MSVC or MinGW, configures env, builds & tests
```

### Automated Build

```powershell
# Check your development environment
.\check-setup.ps1

# Build everything (Rust + Go)
.\build.ps1

# Build with tests
.\build.ps1 -Test

# Clean rebuild
.\build.ps1 -Clean
```

### Manual Build

```powershell
# 1. Build Rust SecureBuffer
cd secure\rust
cargo build --release

# 2. Build Go application (auto-links Rust)
cd ..\..
go build -o bitcoin-sprint.exe ./cmd/sprint
```

### Run Application

```powershell
.\bitcoin-sprint.exe
```

### Windows CGO Setup (manual)

Developer PowerShell (MSVC/clang-cl recommended):

```powershell
$env:CGO_ENABLED = "1"
$env:CC = "clang-cl"
go test ./internal/secure -v
```

MSYS2 MinGW alternative:

```powershell
$env:Path = "C:\\msys64\\mingw64\\bin;" + $env:Path
$env:CGO_ENABLED = "1"
go test ./internal/secure -v
```

## Configuration

Bitcoin Sprint supports multiple configuration formats with automatic precedence resolution:

### Configuration Sources (Priority Order)

1. **Environment Variables** (highest priority)
2. **`.env.local`** - Local environment file (gitignored) 
3. **`.env`** - Environment file template
4. **`config.json`** - JSON configuration with ${VAR} placeholders
5. **Built-in Defaults** - Fallback values

### Environment Variables

- `LICENSE_KEY` - Your Bitcoin Sprint license key
- `RPC_USER` / `RPC_PASS` - Bitcoin Core RPC credentials  
- `RPC_NODES` - Bitcoin Core RPC endpoints (comma-separated)
- `PEER_SECRET` - Secure peer connection secret
- `TURBO_MODE` - Enable turbo mode (true/false)
- `LOG_LEVEL` - Logging level (debug, info, warn, error)

### .env File Configuration

For development, copy `cmd/sprint/.env.example` to `.env.local`:

```bash
LICENSE_KEY=your_license_key_here
RPC_USER=your_rpc_username
RPC_PASS=your_rpc_password
PEER_SECRET=your_peer_secret
RPC_NODES=https://node1.com:8332,https://node2.com:8332
TURBO_MODE=false
```

### JSON Configuration with Placeholders

Use `config.json` with environment variable placeholders for shared configurations:

```json
{
  "license_key": "${LICENSE_KEY}",
  "rpc_user": "${RPC_USER}",
  "rpc_pass": "${RPC_PASS}",
  "peer_secret": "${PEER_SECRET}",
  "tier": "pro",
  "turbo_mode": true
}
```

### Configuration Logging

On startup, Bitcoin Sprint logs the configuration source and a safe summary:

```
INFO Configuration loaded source=.env.local
INFO Config summary tier=pro turbo_mode=true license_key=abcd****wxyz
```

Sensitive values are automatically masked in logs for security.

## Security Features

### Rust FFI SecureBuffer

Bitcoin Sprint uses a Rust-based secure memory system via CGO for handling sensitive data:

- **Memory locking**: Uses `mlock()` (Unix) / `VirtualLock()` (Windows) to prevent swapping
- **Automatic zeroization**: Memory is zeroed before deallocation
- **Cross-platform**: Windows (.dll), Linux (.so), macOS (.dylib) support
- **Error handling**: Comprehensive error checking and memory management

### Usage Example

```go
import "github.com/PayRpc/Bitcoin-Sprint/internal/secure"

// Create secure buffer for sensitive data (32 bytes)
buffer := secure.NewSecureBuffer(32)
if buffer == nil {
    log.Fatal("Failed to create secure buffer")
}
defer buffer.Free() // Automatically zeros memory

// Store sensitive data (private keys, license keys)
if !buffer.Copy([]byte("sensitive-data-here")) {
    log.Fatal("Failed to copy data")
}

// Access protected data
data := buffer.Data()
// Use data for Bitcoin operations...
```

### FFI Integration

The Go-Rust integration happens automatically via CGO directives:

```go
/*
#cgo windows LDFLAGS: -L${SRCDIR}/../../secure/rust/target/release -lsecurebuffer
#cgo linux   LDFLAGS: -L${SRCDIR}/../../secure/rust/target/release -lsecurebuffer
#cgo darwin  LDFLAGS: -L${SRCDIR}/../../secure/rust/target/release -lsecurebuffer
*/
```

When you run `go build`, it automatically finds and links the Rust library.

## Network Architecture

### RPC Polling

- Multiple Bitcoin Core RPC endpoints with failover
- Configurable polling intervals with adaptive timing
- Connection pooling and timeout management
- Exponential backoff on failed nodes

### Peer-to-Peer Networking

- Configurable peer listen port (default: 8335)
- Gossip protocol for block relay
- Concurrent peer connections with timeout handling
- HMAC authentication for secure peer communication

### Dashboard

- Real-time web interface on configurable port
- Block detection metrics and peer status
- Performance analytics and latency tracking
- RESTful API endpoints for monitoring

## API Endpoints

- `GET /status` - Application status and metrics
- `GET /peers` - Connected peer information
- `GET /latest` - Latest block information
- `GET /metrics` - Performance metrics
- `GET /config` - Current configuration

## Development

### Dependencies

**Go:**

- `go.uber.org/zap` - Structured logging
- `golang.org/x/time` - Rate limiting

**Rust:**

- `zeroize` - Secure memory clearing
- `thiserror` - Error handling
- `libc` (Unix) / `winapi` (Windows) - System calls for memory locking

### Build Artifacts

The Rust SecureBuffer produces platform-specific libraries:

- **Windows**: `target/release/securebuffer.dll`
- **Linux**: `target/release/libsecurebuffer.so`
- **macOS**: `target/release/libsecurebuffer.dylib`

### Testing

Test the SecureBuffer integration:

```powershell
go run test_secure.go
```

### Integration Testing

Bitcoin Sprint includes a comprehensive integration test suite that validates proper connectivity with Bitcoin Core. These tests:

- Connect to a running Bitcoin Core node (regtest mode)
- Generate test blocks using the Bitcoin Core RPC
- Verify that Bitcoin Sprint detects blocks via ZMQ
- Validate API endpoint responses

To run the integration tests locally:

```powershell
# Automatically starts Bitcoin Core in Docker and runs the tests
.\run-integration-tests.ps1

# If you have Bitcoin Core already running locally
.\run-integration-tests.ps1 -UseBitcoinCoreDocker:$false
```

The integration tests are also automatically run in CI to ensure Bitcoin Sprint maintains compatibility with Bitcoin Core.

Run comprehensive tests:
```powershell
.\build.ps1 -Test
```

Run individual test suites:
```powershell
# Rust tests
cd secure\rust
cargo test

# Go tests  
go test ./internal/secure
```

## 🛠️ Build Scripts

### build.ps1

Automated build script with options:
```powershell
.\build.ps1           # Basic build
.\build.ps1 -Test     # Build with testing
.\build.ps1 -Clean    # Clean rebuild
.\build.ps1 -Release  # Optimized release build
```

### check-setup.ps1

Development environment verification:
```powershell
.\check-setup.ps1    # Check Go, Rust, C compiler, artifacts
```

## Deployment

### Single Binary

The build process produces a single `bitcoin-sprint.exe` with everything included:
- Go application code
- Linked Rust SecureBuffer library
- No external dependencies required

### Cross-Platform

The same build process works on:
- **Windows** with MSYS2/MinGW, Visual Studio, or TDM-GCC
- **Linux** with build-essential package
- **macOS** with Xcode command line tools

## 🏢 Enterprise Features

Bitcoin Sprint provides production-ready Bitcoin infrastructure:

- **Single binary deployment** - No external dependencies
- **Secure memory management** - Rust FFI for sensitive data
- **Cross-platform support** - Windows, Linux, macOS
- **Automated build system** - One-command builds with testing
- **Professional logging** - Structured output with configurable levels
- **Real-time monitoring** - Dashboard and API endpoints

## � Performance

The system is optimized for:

- **Low-latency block detection** through efficient RPC polling
- **Memory-locked sensitive data** handling via Rust SecureBuffer
- **Automatic linking** - CGO integrates Rust libraries seamlessly
- **Connection pooling** and persistent connections
- **Smart failover** strategies for node failures

## 📝 Documentation

- `INTEGRATION.md` - Complete FFI setup and troubleshooting guide
- `secure/SETUP.md` - Quick SecureBuffer setup instructions
- `FFI-COMPLETE.md` - Project status and architecture summary

## �📝 License

SPDX-License-Identifier: MIT  
Copyright (c) 2025 BitcoinCab.inc

---

**Built for Enterprise**: Professional Bitcoin infrastructure with Rust FFI security and automated deployment.
