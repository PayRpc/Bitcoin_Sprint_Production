#!/usr/bin/env pwsh
# build.ps1 - Automated build script for Bitcoin Sprint with Rust FFI

param(
	[switch]$Clean,
	[switch]$Test,
	[switch]$Release,
	[string]$Output = "bitcoin-sprint.exe"
)

$ErrorActionPreference = "Stop"

Write-Host "🚀 Bitcoin Sprint Build Script" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan

# Check for required tools
Write-Host "Checking prerequisites..." -ForegroundColor Yellow

# Check for Rust
try {
	$rustVersion = cargo --version
	Write-Host "✅ Rust: $rustVersion" -ForegroundColor Green
}
catch {
	Write-Error "❌ Rust not found. Please install Rust from https://rustup.rs/"
}

# Check for Go
try {
	$goVersion = go version
	Write-Host "✅ Go: $goVersion" -ForegroundColor Green
}
catch {
	Write-Error "❌ Go not found. Please install Go from https://golang.org/"
}

# Check for C compiler (CGO requirement)
try {
	if (Get-Command gcc -ErrorAction SilentlyContinue) {
		$gccVersion = gcc --version | Select-Object -First 1
		Write-Host "✅ GCC: $gccVersion" -ForegroundColor Green
		$env:CC = "gcc"
	}
 elseif (Get-Command clang -ErrorAction SilentlyContinue) {
		$clangVersion = clang --version | Select-Object -First 1
		Write-Host "✅ Clang: $clangVersion" -ForegroundColor Green
		$env:CC = "clang"
	}
 else {
		Write-Warning "⚠️  No C compiler found. CGO requires gcc, clang, or MSVC."
		Write-Host "Install options:" -ForegroundColor Yellow
		Write-Host "  • MSYS2/MinGW: pacman -S mingw-w64-x86_64-gcc" -ForegroundColor Gray
		Write-Host "  • Visual Studio Build Tools with C++ workload" -ForegroundColor Gray
		Write-Host "  • TDM-GCC from https://jmeubank.github.io/tdm-gcc/" -ForegroundColor Gray
	}
}
catch {
	Write-Warning "⚠️  Could not verify C compiler availability"
}

# Clean if requested
if ($Clean) {
	Write-Host "`nCleaning build artifacts..." -ForegroundColor Yellow
	if (Test-Path "secure/rust/target") {
		Remove-Item "secure/rust/target" -Recurse -Force
		Write-Host "✅ Cleaned Rust artifacts" -ForegroundColor Green
	}
	if (Test-Path $Output) {
		Remove-Item $Output -Force
		Write-Host "✅ Cleaned Go binary" -ForegroundColor Green
	}
}

# Build Rust components
Write-Host "`nBuilding Rust SecureBuffer..." -ForegroundColor Yellow
Push-Location "secure/rust"
try {
	if ($Release) {
		cargo build --release
	}
 else {
		cargo build --release  # Always use release for FFI
	}
	Write-Host "✅ Rust build completed" -ForegroundColor Green
    
	# Verify artifacts
	$artifacts = Get-ChildItem "target/release/*securebuffer*" -ErrorAction SilentlyContinue
	if ($artifacts) {
		Write-Host "📦 Rust artifacts:" -ForegroundColor Cyan
		foreach ($artifact in $artifacts) {
			$size = [math]::Round($artifact.Length / 1KB, 1)
			Write-Host "   $($artifact.Name) (${size} KB)" -ForegroundColor Gray
		}
	}
}
catch {
	Write-Error "❌ Rust build failed: $_"
}
finally {
	Pop-Location
}

# Test Rust if requested
if ($Test) {
	Write-Host "`nTesting Rust components..." -ForegroundColor Yellow
	Push-Location "secure/rust"
	try {
		cargo test
		Write-Host "✅ Rust tests passed" -ForegroundColor Green
	}
 catch {
		Write-Warning "⚠️  Rust tests failed: $_"
	}
 finally {
		Pop-Location
	}
}

# Build Go application
Write-Host "`nBuilding Go application..." -ForegroundColor Yellow
Push-Location "cmd/sprint"
try {
	$env:CGO_ENABLED = "1"
    
	if ($Release) {
		$ldflags = "-s -w -X main.Version=1.0.0 -X main.BuildTime=$(Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ')"
		& go build -ldflags $ldflags -o "../../$Output" .
	}
 else {
		& go build -o "../../$Output" .
	}

	if ($LASTEXITCODE -ne 0 -or -not (Test-Path "../../$Output")) {
		throw "Go build failed with exit code $LASTEXITCODE"
	}

	Write-Host "✅ Go build completed" -ForegroundColor Green
    
	# Verify binary
	if (Test-Path "../../$Output") {
		$size = [math]::Round((Get-Item "../../$Output").Length / 1MB, 1)
		Write-Host "📦 Binary: $Output (${size} MB)" -ForegroundColor Cyan
	}
}
catch {
	Write-Error "❌ Go build failed: $_"
	exit 1
}
finally {
	Pop-Location
}

# Test Go if requested
if ($Test) {
	Write-Host "`nTesting Go components..." -ForegroundColor Yellow
	& go test ./internal/... -v
	if ($LASTEXITCODE -ne 0) {
		Write-Error "❌ Go tests failed with exit code $LASTEXITCODE"
		exit $LASTEXITCODE
	}
	Write-Host "✅ Go tests passed" -ForegroundColor Green
}

Write-Host "`n🎉 Build completed successfully!" -ForegroundColor Green
Write-Host "Run with: ./$Output" -ForegroundColor Cyan
