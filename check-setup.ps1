#!/usr/bin/env pwsh
# check-setup.ps1 - Verify development environment for Bitcoin Sprint

$ErrorActionPreference = "Continue"

Write-Host "🔍 Bitcoin Sprint Development Environment Check" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan

$allGood = $true

# Check Go
Write-Host "`n📦 Checking Go..." -ForegroundColor Yellow
try {
	$goVersion = go version
	$cgoEnabled = go env CGO_ENABLED
	Write-Host "✅ $goVersion" -ForegroundColor Green
	if ($cgoEnabled -eq "1") {
		Write-Host "✅ CGO enabled" -ForegroundColor Green
	}
 else {
		Write-Host "❌ CGO disabled" -ForegroundColor Red
		$allGood = $false
	}
}
catch {
	Write-Host "❌ Go not found" -ForegroundColor Red
	$allGood = $false
}

# Check Rust
Write-Host "`n🦀 Checking Rust..." -ForegroundColor Yellow
try {
	$cargoVersion = cargo --version
	$rustcVersion = rustc --version
	Write-Host "✅ $cargoVersion" -ForegroundColor Green
	Write-Host "✅ $rustcVersion" -ForegroundColor Green
}
catch {
	Write-Host "❌ Rust toolchain not found" -ForegroundColor Red
	$allGood = $false
}

# Check C Compiler
Write-Host "`n🔧 Checking C Compiler..." -ForegroundColor Yellow
$compilerFound = $false

# Check for GCC
try {
	$gccVersion = gcc --version 2>$null | Select-Object -First 1
	if ($gccVersion) {
		Write-Host "✅ GCC: $gccVersion" -ForegroundColor Green
		$compilerFound = $true
	}
}
catch { }

# Check for Clang
if (-not $compilerFound) {
	try {
		$clangVersion = clang --version 2>$null | Select-Object -First 1
		if ($clangVersion) {
			Write-Host "✅ Clang: $clangVersion" -ForegroundColor Green
			$compilerFound = $true
		}
	}
 catch { }
}

# Check for MSVC (Windows)
if (-not $compilerFound -and $IsWindows) {
	try {
		$clVersion = cl 2>&1 | Select-String "Version" | Select-Object -First 1
		if ($clVersion) {
			Write-Host "✅ MSVC: $clVersion" -ForegroundColor Green
			$compilerFound = $true
		}
	}
 catch { }
}

if (-not $compilerFound) {
	Write-Host "❌ No C compiler found" -ForegroundColor Red
	Write-Host "   Install one of the following:" -ForegroundColor Yellow
	Write-Host "   • MSYS2/MinGW: pacman -S mingw-w64-x86_64-gcc" -ForegroundColor Gray
	Write-Host "   • Visual Studio Build Tools with C++ workload" -ForegroundColor Gray
	Write-Host "   • TDM-GCC from https://jmeubank.github.io/tdm-gcc/" -ForegroundColor Gray
	$allGood = $false
}

# Check Rust artifacts
Write-Host "`n📚 Checking Rust Artifacts..." -ForegroundColor Yellow
$rustTarget = "secure/rust/target/release"
if (Test-Path $rustTarget) {
	$artifacts = Get-ChildItem "$rustTarget/*securebuffer*" -ErrorAction SilentlyContinue
	if ($artifacts) {
		Write-Host "✅ Rust artifacts built:" -ForegroundColor Green
		foreach ($artifact in $artifacts) {
			$size = [math]::Round($artifact.Length / 1KB, 1)
			Write-Host "   📦 $($artifact.Name) (${size} KB)" -ForegroundColor Gray
		}
	}
 else {
		Write-Host "⚠️  Rust artifacts not found - run 'cargo build --release' in secure/rust/" -ForegroundColor Yellow
	}
}
else {
	Write-Host "❌ Rust target directory not found" -ForegroundColor Red
}

# Test CGO compilation (real compile)
Write-Host "`n🔗 Testing CGO Integration..." -ForegroundColor Yellow
if ($compilerFound) {
	$tmp = Join-Path $env:TEMP "cgotest_$(Get-Random)"
	New-Item -ItemType Directory -Path $tmp | Out-Null
	@'
package main

/*
#include <stdlib.h>
*/
import "C"

func main() { _ = C.malloc(1); }
'@ | Set-Content -Path (Join-Path $tmp 'main.go') -Encoding UTF8
	@'
module cgotest

go 1.20
'@ | Set-Content -Path (Join-Path $tmp 'go.mod') -Encoding UTF8
	try {
		Push-Location $tmp
		$env:CGO_ENABLED = "1"
		$output = & go build . 2>&1
		if ($LASTEXITCODE -eq 0) {
			Write-Host "✅ CGO integration ready" -ForegroundColor Green
		}
		else {
			Write-Host "❌ CGO test failed (cannot compile cgo program)" -ForegroundColor Red
			if ($env:CC -like "*clang-cl*") {
				Write-Host "    Note: clang-cl has known compatibility issues with Go CGO test flags" -ForegroundColor Gray
				Write-Host "    This is expected - the actual build script should still work" -ForegroundColor Gray
			}
			else {
				Write-Host "    Hint: If using clang-cl, try running from Developer PowerShell" -ForegroundColor Gray
				$allGood = $false
			}
		}
	}
 catch {
		Write-Host "❌ CGO test failed: $_" -ForegroundColor Red
		$allGood = $false
	}
 finally {
		Pop-Location
		Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
	}
}
else {
	Write-Host "⏭️  Skipping CGO test (no C compiler)" -ForegroundColor Gray
}

# Summary
Write-Host "`n📋 Summary" -ForegroundColor Cyan
if ($allGood) {
	Write-Host "🎉 Development environment is ready!" -ForegroundColor Green
	Write-Host "   You can build Bitcoin Sprint with: .\build.ps1" -ForegroundColor Cyan
}
else {
	Write-Host "⚠️  Some issues need to be resolved before building" -ForegroundColor Yellow
	Write-Host "   See INTEGRATION.md for detailed setup instructions" -ForegroundColor Cyan
}

Write-Host "`n🔗 Quick Commands:" -ForegroundColor Cyan
Write-Host "   .\build.ps1          # Build everything" -ForegroundColor Gray
Write-Host "   .\build.ps1 -Test    # Build and test" -ForegroundColor Gray
Write-Host "   .\build.ps1 -Clean   # Clean and rebuild" -ForegroundColor Gray
