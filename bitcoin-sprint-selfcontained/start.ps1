# Bitcoin Sprint Self-Contained Startup Script
# This script ensures all dependencies are available and starts the application

param(
    [string]$ConfigFile = "config/config-production-optimized.json",
    [string]$LicenseFile = "licenses/license-enterprise.json",
    [switch]$TurboMode = $true
)

Write-Host "🚀 Bitcoin Sprint Self-Contained Startup" -ForegroundColor Green
Write-Host "=" * 50

# Ensure we're in the right directory
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ScriptDir

# Add libs directory to PATH so DLLs can be found
$env:PATH = "$ScriptDir\libs;" + $env:PATH

# Copy config files if they don't exist
if (!(Test-Path "config.json") -and (Test-Path $ConfigFile)) {
    Copy-Item $ConfigFile "config.json"
    Write-Host "✅ Config file ready" -ForegroundColor Green
}

if (!(Test-Path "license.json") -and (Test-Path $LicenseFile)) {
    Copy-Item $LicenseFile "license.json"
    Write-Host "✅ License file ready" -ForegroundColor Green
}

# Set turbo mode if requested
if ($TurboMode) {
    $env:TIER = "turbo"
    Write-Host "⚡ Turbo mode enabled" -ForegroundColor Yellow
}

# Start Bitcoin Sprint
Write-Host "🏃 Starting Bitcoin Sprint..." -ForegroundColor Cyan
& ".\bin\bitcoin-sprint.exe"
