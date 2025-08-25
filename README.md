# Bitcoin Sprint — Enterprise Bitcoin Block Detection

[![CI](https://github.com/PayRpc/BItcoin_Sprint/actions/workflows/ci.yml/badge.svg)](https://github.com/PayRpc/BItcoin_Sprint/actions/workflows/ci.yml)
[![Windows CGO](https://github.com/PayRpc/BItcoin_Sprint/actions/workflows/windows-cgo.yml/badge.svg)](https://github.com/PayRpc/BItcoin_Sprint/actions/workflows/windows-cgo.yml)

Professional Bitcoin block monitoring system with Rust FFI secure memory integration and automated build system.

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
