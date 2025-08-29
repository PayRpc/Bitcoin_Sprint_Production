#!/usr/bin/env pwsh
# tools/dev-win.ps1 — Bitcoin Sprint Enterprise Development Script for Windows
# Comprehensive Windows developer helper for CGO + Rust builds with enterprise security features
# Detects MSVC (clang-cl) or MinGW-w64, configures environment, and runs builds with comprehensive testing

param(
	[switch]$NoTests,
	[switch]$Production,
	[switch]$Enterprise,
	[switch]$Benchmark,
	[switch]$SecurityTest,
	[switch]$FastBuild,
	[string]$Config = "config.json",
	[string]$OutputName = "bitcoin-sprint.exe",
	[switch]$Verbose
)

$ErrorActionPreference = 'Stop'

# Enhanced logging functions with timestamps and enterprise formatting
function Write-Info($msg) { 
	$timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
	Write-Host "[$timestamp] [INFO] $msg" -ForegroundColor Cyan 
}
function Write-Ok($msg) { 
	$timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
	Write-Host "[$timestamp] [SUCCESS] $msg" -ForegroundColor Green 
}
function Write-Warn($msg) { 
	$timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
	Write-Host "[$timestamp] [WARNING] $msg" -ForegroundColor Yellow 
}
function Write-Err($msg) { 
	$timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
	Write-Host "[$timestamp] [ERROR] $msg" -ForegroundColor Red 
}
function Write-Debug($msg) {
	if ($Verbose) {
		$timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
		Write-Host "[$timestamp] [DEBUG] $msg" -ForegroundColor DarkGray
	}
}

# Enterprise build configuration detection
function Get-BuildMode {
	if ($Production) { return "production" }
	if ($Enterprise) { return "enterprise" }
	if ($Benchmark) { return "benchmark" }
	if ($SecurityTest) { return "security" }
	return "development"
}

$BuildMode = Get-BuildMode
Write-Info "Bitcoin Sprint Enterprise Development Script v2.0"
Write-Info "Build Mode: $BuildMode"
Write-Info "Output: $OutputName"
Write-Info "Config: $Config"

Write-Info "Detecting Windows CGO toolchain for enterprise builds..."

# Enhanced MSVC/Visual Studio detection with enterprise validation
$vsWhere = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vswhere.exe"
if (Test-Path $vsWhere) {
	$vsPath = & $vsWhere -latest -property installationPath 2>$null
	$vsVersion = & $vsWhere -latest -property catalog_productDisplayVersion 2>$null
	Write-Debug "Visual Studio path: $vsPath"
	Write-Debug "Visual Studio version: $vsVersion"
	
	if ($vsPath -and !(Get-Command "clang-cl.exe" -ErrorAction SilentlyContinue)) {
		Write-Warn "Enterprise builds recommend 'Developer PowerShell' from VS Tools for optimal clang-cl CGO support"
		Write-Info "Alternative: Use MinGW-w64 for consistent cross-platform builds"
	}
}

# Enhanced toolchain selection with enterprise optimizations
$usingMSVC = $false
$toolchainInfo = ""

# Prefer MSVC clang-cl when available (Developer PowerShell) for enterprise builds
if (Get-Command "clang-cl.exe" -ErrorAction SilentlyContinue) {
	$env:CC = "clang-cl"
	$env:CXX = "clang-cl"
	$env:CGO_ENABLED = "1"
	$usingMSVC = $true
	$clangVersion = & clang-cl --version 2>$null | Select-Object -First 1
	$toolchainInfo = "MSVC clang-cl: $clangVersion"
	Write-Ok "Using MSVC (clang-cl) - Enterprise optimized"
}
elseif (Get-Command "cl.exe" -ErrorAction SilentlyContinue) {
	# Fallback: plain MSVC cl for enterprise compatibility
	$env:CC = "cl"
	$env:CXX = "cl"
	$env:CGO_ENABLED = "1"
	$usingMSVC = $true
	$clVersion = & cl 2>&1 | Select-Object -First 1
	$toolchainInfo = "MSVC cl: $clVersion"
	Write-Ok "Using MSVC (cl) - Enterprise compatible"
}

# Enhanced MinGW-w64 detection with enterprise configuration
if (-not $usingMSVC) {
	$mingwPaths = @(
		"C:\msys64\mingw64\bin",
		"C:\mingw64\bin",
		"C:\TDM-GCC-64\bin"
	)
	
	$mingwFound = $false
	foreach ($mingwPath in $mingwPaths) {
		$gccPath = Join-Path $mingwPath "gcc.exe"
		if (Test-Path $gccPath -PathType Leaf) {
			if (-not ($env:Path -split ';' | Where-Object { $_ -eq $mingwPath })) {
				$env:Path = "$mingwPath;" + $env:Path
				Write-Debug "Added $mingwPath to PATH"
			}
			Remove-Item Env:CC -ErrorAction SilentlyContinue
			Remove-Item Env:CXX -ErrorAction SilentlyContinue
			$env:CC = "gcc"
			$env:CXX = "g++"
			$env:CGO_ENABLED = "1"
			$mingwFound = $true
			$gccVersion = & gcc --version 2>$null | Select-Object -First 1
			$toolchainInfo = "MinGW-w64 GCC: $gccVersion"
			Write-Ok "Using MinGW-w64 GCC ($mingwPath) - Enterprise compatible"
			break
		}
	}
	
	# Check for GCC in PATH as fallback
	if (-not $mingwFound) {
		$mingwInPath = Get-Command gcc.exe -ErrorAction SilentlyContinue
		if ($mingwInPath) {
			Remove-Item Env:CC -ErrorAction SilentlyContinue
			Remove-Item Env:CXX -ErrorAction SilentlyContinue
			$env:CC = "gcc"
			$env:CXX = "g++"
			$env:CGO_ENABLED = "1"
			$mingwFound = $true
			$gccVersion = & gcc --version 2>$null | Select-Object -First 1
			$toolchainInfo = "MinGW-w64 GCC (PATH): $gccVersion"
			Write-Ok "Using MinGW-w64 GCC from PATH - Enterprise compatible"
		}
	}
	
	if (-not $mingwFound) {
		$usingMSVC = $false
	}
}

# Enhanced toolchain validation with enterprise requirements
$hasCompiler = $false
$compilerValidation = ""

try {
	if ($env:CC) {
		$compilerOutput = & $env:CC --version 2>&1
		$hasCompiler = $LASTEXITCODE -eq 0
		$compilerValidation = "CC validation: $($compilerOutput | Select-Object -First 1)"
		Write-Debug $compilerValidation
	}
 else {
		$gcc = Get-Command gcc -ErrorAction SilentlyContinue
		if ($gcc) { 
			$hasCompiler = $true
			$compilerValidation = "GCC found in PATH: $($gcc.Source)"
			Write-Debug $compilerValidation
		}
	}
}
catch {
	$compilerValidation = "Compiler validation failed: $_"
	Write-Debug $compilerValidation
}

if (-not $hasCompiler) {
	Write-Err @"
Bitcoin Sprint Enterprise Build requires a valid CGO toolchain.
Install one of the following enterprise-compatible options:

RECOMMENDED FOR ENTERPRISE:
  - Visual Studio Build Tools 2022 (C++ workload)
  - Use 'Developer PowerShell for VS 2022' (clang-cl preferred)

ALTERNATIVE:
  - MSYS2 MinGW-w64: https://www.msys2.org/
  - Ensure C:\msys64\mingw64\bin is in PATH
  - Run: pacman -S mingw-w64-x86_64-gcc mingw-w64-x86_64-cmake

Current environment:
  CGO_ENABLED: $($env:CGO_ENABLED)
  CC: $($env:CC)
  PATH: $($env:Path -split ';' | Select-Object -First 5 | Join-String -Separator ';')...
"@
	exit 1
}

Write-Info "Toolchain validated: $toolchainInfo"

# Move to repo root (this script lives under tools/)
$root = Split-Path -Path $PSScriptRoot -Parent
Set-Location $root
Write-Debug "Working directory: $(Get-Location)"

# Enterprise environment configuration
Write-Info "Configuring enterprise build environment..."

# Set enterprise-specific environment variables
switch ($BuildMode) {
	"production" {
		$env:BITCOIN_SPRINT_MODE = "production"
		$env:RUST_LOG = "warn"
		$env:GO_LDFLAGS = "-s -w -extldflags=-static"
		Write-Info "Production mode: Optimized binaries, minimal logging"
	}
	"enterprise" {
		$env:BITCOIN_SPRINT_MODE = "enterprise"
		$env:RUST_LOG = "info"
		$env:GO_LDFLAGS = "-s -w"
		$env:ENTERPRISE_FEATURES = "1"
		Write-Info "Enterprise mode: Full feature set, compliance logging"
	}
	"benchmark" {
		$env:BITCOIN_SPRINT_MODE = "benchmark"
		$env:RUST_LOG = "warn"
		$env:GO_LDFLAGS = "-s -w"
		$env:BENCHMARK_MODE = "1"
		Write-Info "Benchmark mode: Performance optimizations enabled"
	}
	"security" {
		$env:BITCOIN_SPRINT_MODE = "security"
		$env:RUST_LOG = "debug"
		$env:SECURITY_AUDIT = "1"
		Write-Info "Security test mode: Full audit logging enabled"
	}
	default {
		$env:BITCOIN_SPRINT_MODE = "development"
		$env:RUST_LOG = "debug"
		$env:DEVELOPMENT_MODE = "1"
		Write-Info "Development mode: Full logging, debug symbols"
	}
}

# Rust environment configuration
$rustTarget = "x86_64-pc-windows-gnu"
if ($usingMSVC) {
	$rustTarget = "x86_64-pc-windows-msvc"
}
$env:RUST_TARGET = $rustTarget
Write-Debug "Rust target: $rustTarget"

# Enhanced environment summary for enterprise builds
Write-Info "Enterprise Build Environment Summary:"
Write-Info "  CGO_ENABLED: $($env:CGO_ENABLED)"
Write-Info "  CC: $($env:CC)"
Write-Info "  CXX: $($env:CXX)"
Write-Info "  RUST_TARGET: $($env:RUST_TARGET)"
Write-Info "  BUILD_MODE: $BuildMode"
Write-Info "  CONFIG_FILE: $Config"

# Enterprise pre-build validation
Write-Info "Running enterprise pre-build validation..."

# Check configuration file exists
if (-not (Test-Path $Config)) {
	Write-Warn "Configuration file '$Config' not found, using default config.json"
	$Config = "config.json"
	if (-not (Test-Path $Config)) {
		Write-Err "No configuration file found. Enterprise builds require valid configuration."
		exit 1
	}
}

# Validate Rust toolchain for secure components
try {
	$rustVersion = & rustc --version 2>$null
	Write-Debug "Rust toolchain: $rustVersion"
	
	$cargoVersion = & cargo --version 2>$null
	Write-Debug "Cargo version: $cargoVersion"
	
	# Check for required Rust targets
	$rustTargets = & rustup target list --installed 2>$null
	if ($rustTargets -notcontains $rustTarget) {
		Write-Info "Installing Rust target: $rustTarget"
		& rustup target add $rustTarget
	}
}
catch {
	Write-Warn "Rust toolchain validation failed: $_"
	Write-Info "Some enterprise security features may not be available"
}

# Run enhanced setup check
try {
	Write-Info "Running enhanced setup validation..."
	if (Test-Path ".\check-setup.ps1") {
		& .\check-setup.ps1
		if ($LASTEXITCODE -ne 0) {
			Write-Warn "Setup validation reported issues (exit code: $LASTEXITCODE)"
		}
	}
 else {
		Write-Debug "check-setup.ps1 not found, skipping setup validation"
	}
}
catch {
	Write-Warn "Setup validation encountered issues: $_"
	Write-Info "Proceeding with build to show detailed errors..."
}

# Enterprise Rust security component build
if ((Test-Path ".\secure\rust\Cargo.toml") -and -not $FastBuild) {
	Write-Info "Building enterprise Rust security components..."
	try {
		Push-Location ".\secure\rust"
		
		$cargoArgs = @("build", "--release", "--target", $rustTarget)
		if ($Verbose) {
			$cargoArgs += "--verbose"
		}
		
		Write-Debug "Cargo command: cargo $($cargoArgs -join ' ')"
		& cargo @cargoArgs
		
		if ($LASTEXITCODE -ne 0) {
			throw "Rust security component build failed with exit code $LASTEXITCODE"
		}
		Write-Ok "Enterprise Rust security components built successfully"
		
		Pop-Location
	}
 catch {
		Pop-Location
		Write-Err "Failed to build Rust security components: $_"
		if (-not $Production) {
			Write-Info "Continuing with Go-only build for development..."
		}
		else {
			exit 1
		}
	}
}

# Enterprise Go build execution
Write-Info "Starting enterprise Go build process..."

try {
	$buildStart = Get-Date
	
	# Determine build command based on mode and requirements
	$buildArgs = @()
	$buildTags = @()
	
	# Configure build based on mode
	switch ($BuildMode) {
		"production" {
			$buildArgs += @("-ldflags", "-s -w -extldflags=-static", "-trimpath")
			$buildTags += "production"
		}
		"enterprise" {
			$buildArgs += @("-ldflags", "-s -w")
			$buildTags += @("enterprise", "cgo")
		}
		"benchmark" {
			$buildArgs += @("-ldflags", "-s -w")
			$buildTags += @("benchmark", "cgo")
		}
		"security" {
			$buildArgs += @("-race") # Enable race detection for security testing
			$buildTags += @("security", "cgo")
		}
		default {
			$buildArgs += @("-race") # Enable race detection for development
			$buildTags += @("development", "cgo")
		}
	}
	
	# Add CGO tags if enabled
	if ($env:CGO_ENABLED -eq "1") {
		$buildTags += "cgo"
	}
	
	# Build tags configuration
	if ($buildTags.Count -gt 0) {
		$buildArgs += @("-tags", ($buildTags -join ","))
	}
	
	# Output configuration
	$buildArgs += @("-o", $OutputName)
	$buildArgs += ".\cmd\sprint"
	
	Write-Info "Go build command: go build $($buildArgs -join ' ')"
	Write-Debug "Build tags: $($buildTags -join ', ')"
	
	# Execute the build
	& go build @buildArgs
	
	if ($LASTEXITCODE -ne 0) {
		throw "Go build failed with exit code $LASTEXITCODE"
	}
	
	$buildEnd = Get-Date
	$buildTime = ($buildEnd - $buildStart).TotalSeconds
	Write-Ok "Go build completed successfully in $([math]::Round($buildTime, 2)) seconds"
	
	# Verify output file
	if (Test-Path $OutputName) {
		$fileInfo = Get-Item $OutputName
		Write-Info "Output file: $OutputName ($([math]::Round($fileInfo.Length / 1MB, 2)) MB)"
	}
 else {
		throw "Output file '$OutputName' was not created"
	}
	
}
catch {
	Write-Err "Enterprise Go build failed: $_"
	exit 1
}

# Enterprise testing suite
if (-not $NoTests -and -not $FastBuild) {
	Write-Info "Running enterprise test suite..."
	
	try {
		$testStart = Get-Date
		
		# Standard Go tests
		Write-Info "Running Go unit tests..."
		& go test -v ./...
		if ($LASTEXITCODE -ne 0) {
			Write-Warn "Some Go tests failed (exit code: $LASTEXITCODE)"
		}
		
		# Security-specific tests
		if ($SecurityTest -or $BuildMode -eq "security") {
			Write-Info "Running security validation tests..."
			
			# Run security-specific test patterns
			& go test -v -run ".*Security.*" ./...
			& go test -v -run ".*Secure.*" ./...
			
			if (Test-Path ".\cmd\selfcheck") {
				Write-Info "Running security self-check..."
				Push-Location ".\cmd\selfcheck"
				& go build -o "..\..\selfcheck-test.exe"
				Pop-Location
				
				if (Test-Path ".\selfcheck-test.exe") {
					& .\selfcheck-test.exe
					if ($LASTEXITCODE -eq 0) {
						Write-Ok "Security self-check passed"
					}
					else {
						Write-Warn "Security self-check reported issues"
					}
					Remove-Item ".\selfcheck-test.exe" -ErrorAction SilentlyContinue
				}
			}
		}
		
		# Benchmark tests
		if ($Benchmark -or $BuildMode -eq "benchmark") {
			Write-Info "Running performance benchmarks..."
			& go test -v -bench=. -benchmem ./...
		}
		
		$testEnd = Get-Date
		$testTime = ($testEnd - $testStart).TotalSeconds
		Write-Ok "Enterprise test suite completed in $([math]::Round($testTime, 2)) seconds"
		
	}
 catch {
		Write-Warn "Test suite encountered issues: $_"
		Write-Info "Build succeeded, but some tests may have failed"
	}
}

# Enterprise build summary
Write-Info "=== Bitcoin Sprint Enterprise Build Summary ==="
Write-Ok "Build Mode: $BuildMode"
Write-Ok "Output: $OutputName"
Write-Ok "Toolchain: $toolchainInfo"
Write-Ok "Configuration: $Config"

if (Test-Path $OutputName) {
	$finalFile = Get-Item $OutputName
	Write-Ok "Binary Size: $([math]::Round($finalFile.Length / 1MB, 2)) MB"
	Write-Ok "Created: $($finalFile.LastWriteTime)"
}

Write-Ok "✅ Enterprise build completed successfully!"
Write-Info "Ready for deployment to Bitcoin Sprint enterprise infrastructure"

exit 0
