# SecureBuffer FFI Setup Guide

This guide explains how to set up the development environment for Bitcoin Sprint's Rust-Go FFI integration.

## Prerequisites

The Bitcoin Sprint project uses CGO to link with Rust libraries. You need a C compiler installed:

### Linux (Debian/Ubuntu)
```bash
sudo apt update
sudo apt install build-essential pkg-config
```

### macOS
```bash
xcode-select --install
```

### Windows Options

#### Option 1: MSYS2/MinGW (Recommended)
```bash
# Install MSYS2 from https://www.msys2.org/
# Then in MSYS2 terminal:
pacman -S mingw-w64-x86_64-gcc
```

#### Option 2: Visual Studio Build Tools
- Install Visual Studio Build Tools or Visual Studio Community
- Ensure "C++ build tools" workload is selected

#### Option 3: TDM-GCC
- Download and install TDM-GCC from https://jmeubank.github.io/tdm-gcc/

## Build Process

The project is configured for automatic building:

1. **Build Rust components** (one-time or when Rust code changes):
   ```bash
   cd secure/rust
   cargo build --release
   ```

2. **Build Go application** (automatically links Rust):
   ```bash
   go build -o bitcoin-sprint.exe .
   ```

## How It Works

- The `internal/secure/securebuffer.go` file contains CGO directives that automatically link the Rust library
- Cross-platform library paths are configured via `#cgo` directives
- The build process finds the Rust artifacts in `secure/rust/target/release/`
- No manual file copying or complex build scripts required

## Troubleshooting

### "gcc not found" Error
- Ensure your C compiler is in PATH
- On Windows with MSYS2, use the MSYS2 terminal or add MinGW64 to PATH

### "cannot find -lsecurebuffer" Error
- Run `cargo build --release` in the `secure/rust` directory first
- Verify that `libsecurebuffer.rlib` exists in `secure/rust/target/release/`

### Permission Issues on Windows
- Run terminal as Administrator if needed
- Ensure Windows Defender isn't blocking the build process

## Architecture

```
Bitcoin Sprint
├── cmd/sprint/          # Go application entry point
├── internal/secure/     # Go FFI wrapper
└── secure/rust/         # Rust SecureBuffer implementation
    ├── src/            # Rust source code
    ├── include/        # C header for FFI
    └── target/release/ # Compiled Rust artifacts
```

The FFI integration provides memory-locked secure buffers for sensitive data handling in Bitcoin operations.
