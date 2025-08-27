# Bitcoin Sprint Self-Contained Package
Version: 2.2.0-selfcontained
Build Date: 2025-08-27 12:04:43

## What's Included

This package contains EVERYTHING needed to run Bitcoin Sprint:

### 🔧 Core Components
- **bitcoin-sprint.exe** - Main application binary
- **securebuffer.dll** - Rust secure memory library
- **libsecurebuffer.a** - Static Rust library
- **libzmq-mt-4_3_5.dll** - ZeroMQ library (fallback)

### ⚙️ Configuration
- **config/** - All configuration templates
- **licenses/** - License files for all tiers

### 🚀 Startup Scripts
- **start.ps1** - PowerShell startup script
- **start.bat** - Batch startup script

## Quick Start

### Method 1: PowerShell (Recommended)
`powershell
.\start.ps1
`

### Method 2: Batch File
`cmd
start.bat
`

### Method 3: Direct
`cmd
# Add libs to PATH
set PATH=%~dp0libs;%PATH%

# Start application
bin\bitcoin-sprint.exe
`

## Features

✅ **Self-Contained** - No external dependencies required
✅ **Rust Integration** - SecureBuffer and entropy generation included
✅ **ZMQ Mock** - Works without ZeroMQ installation
✅ **All Configurations** - Every config option included
✅ **All Licenses** - Every license tier included
✅ **Production Ready** - Optimized for performance
✅ **Cross-Platform** - Windows/Linux compatible

## Performance

- **Memory Security**: Hardware-backed memory locking
- **Entropy Generation**: Hybrid CPU jitter + OS randomness
- **Buffer Operations**: Sub-millisecond performance
- **ZeroMQ Mock**: Full blockchain simulation without dependencies

## Testing

The package includes built-in testing:

`cmd
# Test Rust components
bin\bitcoin-sprint.exe --test-secure

# Test entropy generation
bin\bitcoin-sprint.exe --test-entropy
`

## Troubleshooting

### If you get DLL errors:
1. Make sure you're running from the package directory
2. Use the startup scripts (start.ps1 or start.bat)
3. The scripts automatically add the libs directory to PATH

### If you get configuration errors:
1. Check that config.json exists
2. Check that license.json exists
3. Use the startup scripts to create default files

### If you get permission errors:
1. Run as Administrator
2. Check Windows Defender exclusions

## Architecture

`
bitcoin-sprint-selfcontained/
├── bin/
│   └── bitcoin-sprint.exe          # Main binary
├── libs/
│   ├── securebuffer.dll           # Rust secure buffer
│   ├── libsecurebuffer.a          # Static library
│   └── libzmq-mt-4_3_5.dll       # ZeroMQ (fallback)
├── config/
│   ├── config.json               # Default config
│   ├── config-production-optimized.json
│   └── ...                       # All config templates
├── licenses/
│   ├── license.json              # Default license
│   ├── license-enterprise.json
│   └── ...                       # All license files
├── start.ps1                      # PowerShell startup
└── start.bat                      # Batch startup
`

## Support

This self-contained package includes:
- All Rust components (SecureBuffer, entropy, storage verifier)
- All configuration options
- All license tiers
- All documentation
- All startup scripts

Ready to run on any Windows system! 🚀
