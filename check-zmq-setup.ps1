#!/usr/bin/env pwsh
# ZeroMQ Setup Checker and Installer for Bitcoin Sprint
# Checks for libzmq.dll and guides installation if missing
# Date: August 26, 2025

param(
    [switch]$Install,
    [switch]$TestBuild,
    [string]$VcpkgPath = ""
)

Write-Host "⚡ ZeroMQ Setup for Bitcoin Sprint" -ForegroundColor Cyan
Write-Host "=================================" -ForegroundColor Cyan

$ErrorCount = 0
$WarningCount = 0

# Function to find files in common locations
function Find-ZmqFile {
    param($FileName)
    
    $searchPaths = @(
        "C:\vcpkg\installed\x64-windows\bin",
        "C:\vcpkg\installed\x64-windows\lib", 
        "C:\vcpkg\installed\x64-windows\include",
        "C:\Program Files\ZeroMQ",
        "C:\msys64\mingw64\bin",
        "C:\msys64\mingw64\lib",
        "C:\msys64\mingw64\include",
        "$env:USERPROFILE\vcpkg\installed\x64-windows\bin",
        "$env:USERPROFILE\vcpkg\installed\x64-windows\lib",
        "$env:USERPROFILE\vcpkg\installed\x64-windows\include",
        "$PWD\vcpkg\installed\x64-windows\bin",
        "$PWD\vcpkg\installed\x64-windows\lib",
        "$PWD\vcpkg\installed\x64-windows\include",
        "$PWD\vendor\zmq\lib",
        "$PWD\deps\zmq\lib"
    )

    # If vcpkg path provided, add it to search
    if ($VcpkgPath) {
        $searchPaths = @(
            "$VcpkgPath\installed\x64-windows\bin", 
            "$VcpkgPath\installed\x64-windows\lib",
            "$VcpkgPath\installed\x64-windows\include"
        ) + $searchPaths
    }
    
    foreach ($path in $searchPaths) {
        if (Test-Path $path) {
            $fullPath = Join-Path $path $FileName
            if (Test-Path $fullPath) {
                return $fullPath
            }
            # Also search recursively
            $found = Get-ChildItem -Path $path -Recurse -Name $FileName -ErrorAction SilentlyContinue | Select-Object -First 1
            if ($found) {
                return Join-Path $path $found
            }
        }
    }
    return $null
}

# 1. Check ZeroMQ Runtime Library
Write-Host "`n📦 Checking ZeroMQ Runtime..." -ForegroundColor Yellow

$libzmqDll = Find-ZmqFile "libzmq.dll"
if (-not $libzmqDll) {
    # Try versioned naming pattern (vcpkg uses libzmq-mt-x_x_x.dll)
    foreach ($path in $searchPaths) {
        if (Test-Path $path) {
            $found = Get-ChildItem -Path $path -Filter "libzmq-mt-*.dll" -ErrorAction SilentlyContinue | Select-Object -First 1
            if ($found) {
                $libzmqDll = $found.FullName
                break
            }
        }
    }
}
if ($libzmqDll) {
    Write-Host "   ✅ Found libzmq.dll: $libzmqDll" -ForegroundColor Green
    
    try {
        $dllInfo = Get-Item $libzmqDll
        $version = $dllInfo.VersionInfo.FileVersion
        $size = [math]::Round($dllInfo.Length / 1KB, 1)
        Write-Host "      Version: $version" -ForegroundColor Gray
        Write-Host "      Size: ${size} KB" -ForegroundColor Gray
        
        # Test if DLL can be loaded
        try {
            Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;
public class ZmqTest {
    [DllImport("$libzmqDll")]
    public static extern IntPtr zmq_version(out int major, out int minor, out int patch);
}
"@
            Write-Host "      ✅ DLL is loadable" -ForegroundColor Green
        } catch {
            Write-Host "      ⚠️  DLL might have dependency issues" -ForegroundColor Yellow
            $WarningCount++
        }
    } catch {
        Write-Host "      ⚠️  Cannot read DLL info" -ForegroundColor Yellow
        $WarningCount++
    }
} else {
    Write-Host "   ❌ libzmq.dll NOT FOUND" -ForegroundColor Red
    Write-Host "      This will cause ZeroMQ to run in MOCK MODE" -ForegroundColor Yellow
    $ErrorCount++
}

# 2. Check ZeroMQ Development Files
Write-Host "`n🔧 Checking ZeroMQ Development Files..." -ForegroundColor Yellow

$libzmqLib = Find-ZmqFile "libzmq.lib"
if (-not $libzmqLib) {
    # Try versioned naming pattern (vcpkg uses libzmq-mt-x_x_x.lib)
    foreach ($path in $searchPaths) {
        if (Test-Path $path) {
            $found = Get-ChildItem -Path $path -Filter "libzmq-mt-*.lib" -ErrorAction SilentlyContinue | Select-Object -First 1
            if ($found) {
                $libzmqLib = $found.FullName
                break
            }
        }
    }
}
$zmqH = Find-ZmqFile "zmq.h"

if ($libzmqLib) {
    Write-Host "   ✅ Found libzmq.lib: $libzmqLib" -ForegroundColor Green
} else {
    Write-Host "   ⚠️  libzmq.lib not found (needed for CGO linking)" -ForegroundColor Yellow
    $WarningCount++
}

if ($zmqH) {
    Write-Host "   ✅ Found zmq.h: $zmqH" -ForegroundColor Green
    
    # Check zmq.h version
    try {
        $headerContent = Get-Content $zmqH -ErrorAction SilentlyContinue
        $versionMajor = ($headerContent | Select-String "#define ZMQ_VERSION_MAJOR" | Select-Object -First 1) -replace ".*?(\d+).*", '$1'
        $versionMinor = ($headerContent | Select-String "#define ZMQ_VERSION_MINOR" | Select-Object -First 1) -replace ".*?(\d+).*", '$1'
        if ($versionMajor -and $versionMinor) {
            Write-Host "      Header version: $versionMajor.$versionMinor" -ForegroundColor Gray
        }
    } catch {
        Write-Host "      ⚠️  Cannot read header version" -ForegroundColor Yellow
    }
} else {
    Write-Host "   ⚠️  zmq.h not found (needed for CGO compilation)" -ForegroundColor Yellow
    $WarningCount++
}

# 3. Check vcpkg Installation
Write-Host "`n📦 Checking vcpkg..." -ForegroundColor Yellow

$vcpkgLocations = @(
    "C:\vcpkg\vcpkg.exe",
    "$env:USERPROFILE\vcpkg\vcpkg.exe",
    "$PWD\vcpkg\vcpkg.exe"
)

if ($VcpkgPath) {
    $vcpkgLocations = @("$VcpkgPath\vcpkg.exe") + $vcpkgLocations
}

$vcpkgExe = $null
foreach ($location in $vcpkgLocations) {
    if (Test-Path $location) {
        $vcpkgExe = $location
        Write-Host "   ✅ Found vcpkg: $location" -ForegroundColor Green
        
        # Check if zeromq is installed
        try {
            $installed = & $location list zeromq 2>$null
            if ($installed -match "zeromq") {
                Write-Host "   ✅ ZeroMQ package: $($installed -split '\n' | Select-Object -First 1)" -ForegroundColor Green
            } else {
                Write-Host "   ⚠️  ZeroMQ not installed via vcpkg" -ForegroundColor Yellow
                $WarningCount++
            }
        } catch {
            Write-Host "   ⚠️  Cannot check vcpkg packages" -ForegroundColor Yellow
        }
        break
    }
}

if (-not $vcpkgExe) {
    Write-Host "   ⚠️  vcpkg not found (recommended for easy ZeroMQ install)" -ForegroundColor Yellow
    $WarningCount++
}

# 4. Check CGO Environment
Write-Host "`n🔧 Checking CGO Environment..." -ForegroundColor Yellow

$cgoEnabled = $env:CGO_ENABLED
if ($cgoEnabled -eq "0") {
    Write-Host "   ❌ CGO_ENABLED is disabled" -ForegroundColor Red
    Write-Host "      Set: `$env:CGO_ENABLED = '1'" -ForegroundColor Yellow
    $ErrorCount++
} else {
    Write-Host "   ✅ CGO_ENABLED: $($cgoEnabled ?? 'default (1)')" -ForegroundColor Green
}

# Check CGO flags
$cgoFlags = $env:CGO_CFLAGS
$cgoLdFlags = $env:CGO_LDFLAGS

if ($cgoFlags) {
    Write-Host "   📋 CGO_CFLAGS: $cgoFlags" -ForegroundColor Gray
} else {
    Write-Host "   ℹ️  CGO_CFLAGS not set (will auto-configure if ZMQ found)" -ForegroundColor Gray
}

if ($cgoLdFlags) {
    Write-Host "   📋 CGO_LDFLAGS: $cgoLdFlags" -ForegroundColor Gray
} else {
    Write-Host "   ℹ️  CGO_LDFLAGS not set (will auto-configure if ZMQ found)" -ForegroundColor Gray
}

# Summary
Write-Host "`n📊 ZeroMQ Status Summary" -ForegroundColor Cyan
Write-Host "========================" -ForegroundColor Cyan

$zmqReady = ($libzmqDll -and $zmqH)

if ($ErrorCount -eq 0 -and $WarningCount -eq 0 -and $zmqReady) {
    Write-Host "✅ ZeroMQ is fully configured and ready!" -ForegroundColor Green
    Write-Host "   Bitcoin Sprint will use REAL ZMQ (not mock mode)" -ForegroundColor Green
} elseif ($zmqReady) {
    Write-Host "⚠️  ZeroMQ found but with $WarningCount warnings" -ForegroundColor Yellow
    Write-Host "   Bitcoin Sprint should work (check logs for mock mode)" -ForegroundColor Yellow
} else {
    Write-Host "❌ ZeroMQ not properly configured" -ForegroundColor Red
    Write-Host "   Bitcoin Sprint will run in MOCK MODE" -ForegroundColor Yellow
    Write-Host "   Install ZeroMQ for production use" -ForegroundColor Yellow
}

# Installation Guide
if (-not $zmqReady -or $Install) {
    Write-Host "`n🛠️  ZeroMQ Installation Guide" -ForegroundColor Cyan
    Write-Host "=============================" -ForegroundColor Cyan
    
    if ($vcpkgExe) {
        Write-Host "`n📦 Method 1: Install via existing vcpkg (RECOMMENDED)" -ForegroundColor Green
        Write-Host "Commands to run:" -ForegroundColor White
        Write-Host "   $vcpkgExe install zeromq:x64-windows" -ForegroundColor Gray
        Write-Host "   $vcpkgExe integrate install" -ForegroundColor Gray
        
        if ($Install) {
            Write-Host "`n🚀 Auto-installing ZeroMQ via vcpkg..." -ForegroundColor Green
            try {
                Write-Host "Running: $vcpkgExe install zeromq:x64-windows" -ForegroundColor Gray
                & $vcpkgExe install zeromq:x64-windows
                
                if ($LASTEXITCODE -eq 0) {
                    Write-Host "✅ ZeroMQ installation completed!" -ForegroundColor Green
                    
                    # Re-check
                    $libzmqDll = Find-ZmqFile "libzmq.dll"
                    if ($libzmqDll) {
                        Write-Host "✅ libzmq.dll now found: $libzmqDll" -ForegroundColor Green
                    }
                } else {
                    Write-Host "❌ Installation failed (exit code $LASTEXITCODE)" -ForegroundColor Red
                }
            } catch {
                Write-Host "❌ Installation error: $_" -ForegroundColor Red
            }
        }
    } else {
        Write-Host "`n📦 Method 1: Install vcpkg first, then ZeroMQ" -ForegroundColor Yellow
        Write-Host "Commands to run:" -ForegroundColor White
        Write-Host "   git clone https://github.com/microsoft/vcpkg" -ForegroundColor Gray
        Write-Host "   cd vcpkg" -ForegroundColor Gray
        Write-Host "   .\bootstrap-vcpkg.bat" -ForegroundColor Gray
        Write-Host "   .\vcpkg.exe install zeromq:x64-windows" -ForegroundColor Gray
    }
    
    Write-Host "`n📥 Method 2: Download prebuilt binaries" -ForegroundColor Yellow
    Write-Host "1. Download from: https://github.com/zeromq/libzmq/releases" -ForegroundColor Gray
    Write-Host "2. Extract to C:\Program Files\ZeroMQ\" -ForegroundColor Gray
    Write-Host "3. Add bin folder to PATH" -ForegroundColor Gray
    
    Write-Host "`n🔧 Method 3: Build from source" -ForegroundColor Yellow
    Write-Host "1. Install CMake and Visual Studio" -ForegroundColor Gray
    Write-Host "2. git clone https://github.com/zeromq/libzmq" -ForegroundColor Gray
    Write-Host "3. Follow build instructions in README" -ForegroundColor Gray
}

# Environment Setup
if ($libzmqDll -and (-not $cgoFlags -or -not $cgoLdFlags)) {
    Write-Host "`n🔧 Environment Configuration" -ForegroundColor Cyan
    Write-Host "============================" -ForegroundColor Cyan
    
    $zmqBinDir = Split-Path $libzmqDll
    $zmqIncludeDir = $zmqBinDir -replace "\\bin$", "\include"
    $zmqLibDir = $zmqBinDir -replace "\\bin$", "\lib"
    
    Write-Host "`nAdd these to your PowerShell session:" -ForegroundColor Yellow
    Write-Host "`$env:CGO_CFLAGS = `"-I$zmqIncludeDir`"" -ForegroundColor Gray
    Write-Host "`$env:CGO_LDFLAGS = `"-L$zmqLibDir -lzmq`"" -ForegroundColor Gray
    Write-Host "`$env:PATH = `"$zmqBinDir;`$env:PATH`"" -ForegroundColor Gray
    
    if ($TestBuild) {
        Write-Host "`n🏗️  Auto-configuring environment and testing build..." -ForegroundColor Green
        
        $env:CGO_CFLAGS = "-I$zmqIncludeDir"
        $env:CGO_LDFLAGS = "-L$zmqLibDir -lzmq"
        $env:PATH = "$zmqBinDir;$env:PATH"
        
        Write-Host "Building Bitcoin Sprint with ZeroMQ..." -ForegroundColor Yellow
        go build -ldflags="-s -w" -o bitcoin-sprint-zmq-test.exe ./cmd/sprintd
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✅ Build successful! ZeroMQ integration working." -ForegroundColor Green
            
            # Quick test run
            Write-Host "`nTesting ZeroMQ connection..." -ForegroundColor Yellow
            $env:TIER = "turbo"
            $env:SKIP_LICENSE_VALIDATION = "true"
            
            $testLog = "zmq-test-$(Get-Date -Format 'yyyyMMdd-HHmmss').log"
            $process = Start-Process -FilePath ".\bitcoin-sprint-zmq-test.exe" -PassThru -RedirectStandardOutput $testLog -WindowStyle Hidden
            Start-Sleep -Seconds 3
            Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
            
            if (Test-Path $testLog) {
                $logs = Get-Content $testLog -Raw
                if ($logs -match 'Starting ZMQ client.*tcp://') {
                    Write-Host "✅ ZeroMQ client started successfully (NOT mock mode)!" -ForegroundColor Green
                    Write-Host "   Check $testLog for full logs" -ForegroundColor Gray
                } elseif ($logs -match 'mock mode') {
                    Write-Host "⚠️  Still running in mock mode - check ZMQ setup" -ForegroundColor Yellow
                    Write-Host "   Check $testLog for details" -ForegroundColor Gray
                } else {
                    Write-Host "ℹ️  Build works, check $testLog for ZMQ status" -ForegroundColor Gray
                }
            }
            
            # Cleanup
            Remove-Item "bitcoin-sprint-zmq-test.exe" -ErrorAction SilentlyContinue
        } else {
            Write-Host "❌ Build failed. Check CGO and ZeroMQ configuration." -ForegroundColor Red
        }
    }
}

Write-Host "`n🎯 Next Steps:" -ForegroundColor Cyan
if ($zmqReady) {
    Write-Host "1. Build: go build -o bitcoin-sprint.exe ./cmd/sprintd" -ForegroundColor White
    Write-Host "2. Run: .\bitcoin-sprint.exe" -ForegroundColor White
    Write-Host "3. Look for log: 'Starting ZMQ client' (not 'mock mode')" -ForegroundColor White
} else {
    Write-Host "1. Install ZeroMQ (see guide above)" -ForegroundColor White
    Write-Host "2. Re-run: .\check-zmq-setup.ps1 -TestBuild" -ForegroundColor White
    Write-Host "3. Build when ZeroMQ is properly configured" -ForegroundColor White
}

exit $ErrorCount
