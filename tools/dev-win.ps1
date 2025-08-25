#!/usr/bin/env pwsh
# tools/dev-win.ps1 — Windows developer helper for CGO + Rust builds
# Detects MSVC (clang-cl) or MinGW-w64, configures env, and runs the build with tests.

param(
	[switch]$NoTests
)

$ErrorActionPreference = 'Stop'

function Write-Info($msg) { Write-Host $msg -ForegroundColor Cyan }
function Write-Ok($msg) { Write-Host $msg -ForegroundColor Green }
function Write-Warn($msg) { Write-Host $msg -ForegroundColor Yellow }
function Write-Err($msg) { Write-Host $msg -ForegroundColor Red }

Write-Info "Detecting Windows CGO toolchain..."

# For clang-cl reliability, detect if we're in Developer PowerShell
$vsWhere = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vswhere.exe"
if (Test-Path $vsWhere) {
	$vsPath = & $vsWhere -latest -property installationPath 2>$null
	if ($vsPath -and !(Get-Command "clang-cl.exe" -ErrorAction SilentlyContinue)) {
		Write-Warn "Note: For best clang-cl CGO support, use 'Developer PowerShell' from VS Tools"
	}
}

# Prefer MSVC clang-cl when available (Developer PowerShell)
$usingMSVC = $false
if (Get-Command "clang-cl.exe" -ErrorAction SilentlyContinue) {
	$env:CC = "clang-cl"
	$env:CGO_ENABLED = "1"
	$usingMSVC = $true
	Write-Ok "Using MSVC (clang-cl)"
}
elseif (Get-Command "cl.exe" -ErrorAction SilentlyContinue) {
	# Fallback: plain MSVC cl. Go can use cl via CC on Windows, but clang-cl is preferred.
	$env:CC = "cl"
	$env:CGO_ENABLED = "1"
	$usingMSVC = $true
	Write-Ok "Using MSVC (cl)"
}

# Detect MinGW-w64 as an alternative
if (-not $usingMSVC) {
	$mingwDefault = "C:\msys64\mingw64\bin\gcc.exe"
	$mingwInPath = Get-Command gcc.exe -ErrorAction SilentlyContinue
	if (Test-Path $mingwDefault -PathType Leaf) {
		if (-not ($env:Path -split ';' | Where-Object { $_ -eq "C:\\msys64\\mingw64\\bin" })) {
			$env:Path = "C:\msys64\mingw64\bin;" + $env:Path
		}
		Remove-Item Env:CC -ErrorAction SilentlyContinue
		$env:CGO_ENABLED = "1"
		Write-Ok "Using MinGW-w64 GCC (C:\msys64\mingw64\bin)"
	}
 elseif ($mingwInPath) {
		Remove-Item Env:CC -ErrorAction SilentlyContinue
		$env:CGO_ENABLED = "1"
		Write-Ok "Using MinGW-w64 GCC from PATH"
	}
}

# Final validation
$hasCompiler = $false
try {
	if ($env:CC) {
		$null = & $env:CC --version 2>$null
		$hasCompiler = $true
	}
 else {
		$gcc = Get-Command gcc -ErrorAction SilentlyContinue
		if ($gcc) { $hasCompiler = $true }
	}
}
catch { }

if (-not $hasCompiler) {
	Write-Err @"
No valid CGO toolchain found. Install one of:
	- Visual Studio Build Tools (C++ workload), then use Developer PowerShell (clang-cl preferred)
	- OR MSYS2 MinGW-w64 and ensure C:\msys64\mingw64\bin is in PATH
"@
	exit 1
}

# Move to repo root (this script lives under tools/)
$root = Split-Path -Path $PSScriptRoot -Parent
Set-Location $root

# Optional: print quick environment summary
Write-Info "CGO_ENABLED=$($env:CGO_ENABLED) CC=$($env:CC)"

# Run setup check first for fast feedback
try {
	Write-Info "Running check-setup.ps1..."
	.\check-setup.ps1
}
catch {
	Write-Warn "check-setup.ps1 reported issues. Proceeding to build to show full errors..."
}

# Run build
try {
	if ($NoTests) {
		Write-Info "Running build.ps1 (no tests)..."
		.\build.ps1
	}
 else {
		Write-Info "Running build.ps1 -Test..."
		.\build.ps1 -Test
	}
	if ($LASTEXITCODE -ne 0) { throw "Build failed with exit code $LASTEXITCODE" }
	Write-Ok "Build finished successfully"
	exit 0
}
catch {
	Write-Err "Build failed: $_"
	exit 1
}
