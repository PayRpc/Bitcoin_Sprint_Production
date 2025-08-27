#!/usr/bin/env powershell

# Bitcoin Sprint Turbo Mode Demonstration Script
# This script shows how to activate different performance tiers

Write-Host "🚀 Bitcoin Sprint Turbo Mode Setup" -ForegroundColor Green
Write-Host "====================================" -ForegroundColor Green
Write-Host ""

# Function to set environment variables for a specific tier
function Set-TierConfiguration {
    param(
        [string]$Tier
    )
    
    Write-Host "🔧 Configuring Bitcoin Sprint for $Tier tier..." -ForegroundColor Yellow
    
    # Base configuration
    [Environment]::SetEnvironmentVariable("TIER", $Tier, "Process")
    [Environment]::SetEnvironmentVariable("API_HOST", "0.0.0.0", "Process")
    [Environment]::SetEnvironmentVariable("API_PORT", "8080", "Process")
    [Environment]::SetEnvironmentVariable("BITCOIN_NODE", "127.0.0.1:8333", "Process")
    [Environment]::SetEnvironmentVariable("ZMQ_NODE", "127.0.0.1:28332", "Process")
    [Environment]::SetEnvironmentVariable("PEER_LISTEN_PORT", "8335", "Process")
    [Environment]::SetEnvironmentVariable("SKIP_LICENSE_VALIDATION", "true", "Process")
    
    switch ($Tier) {
        "turbo" {
            Write-Host "⚡ Enabling Turbo optimizations:" -ForegroundColor Cyan
            Write-Host "  - Write deadline: 500µs" -ForegroundColor White
            Write-Host "  - Shared memory: Enabled" -ForegroundColor White
            Write-Host "  - Buffer size: 2048" -ForegroundColor White
            Write-Host "  - Direct P2P: Enabled" -ForegroundColor White
            Write-Host "  - Memory channel: Enabled" -ForegroundColor White
            
            [Environment]::SetEnvironmentVariable("USE_SHARED_MEMORY", "true", "Process")
            [Environment]::SetEnvironmentVariable("USE_DIRECT_P2P", "true", "Process")
            [Environment]::SetEnvironmentVariable("USE_MEMORY_CHANNEL", "true", "Process")
            [Environment]::SetEnvironmentVariable("OPTIMIZE_SYSTEM", "true", "Process")
            [Environment]::SetEnvironmentVariable("APP_MODE", "turbo", "Process")
        }
        
        "enterprise" {
            Write-Host "🏆 Enabling Enterprise optimizations:" -ForegroundColor Cyan
            Write-Host "  - Write deadline: 200µs" -ForegroundColor White
            Write-Host "  - Shared memory: Enabled" -ForegroundColor White
            Write-Host "  - Buffer size: 4096" -ForegroundColor White
            Write-Host "  - Direct P2P: Enabled" -ForegroundColor White
            Write-Host "  - Memory channel: Enabled" -ForegroundColor White
            Write-Host "  - System optimizations: Enabled" -ForegroundColor White
            
            [Environment]::SetEnvironmentVariable("USE_SHARED_MEMORY", "true", "Process")
            [Environment]::SetEnvironmentVariable("USE_DIRECT_P2P", "true", "Process")
            [Environment]::SetEnvironmentVariable("USE_MEMORY_CHANNEL", "true", "Process")
            [Environment]::SetEnvironmentVariable("OPTIMIZE_SYSTEM", "true", "Process")
            [Environment]::SetEnvironmentVariable("ENABLE_KERNEL_BYPASS", "false", "Process")
            [Environment]::SetEnvironmentVariable("APP_MODE", "enterprise", "Process")
        }
        
        "business" {
            Write-Host "💼 Enabling Business optimizations:" -ForegroundColor Cyan
            Write-Host "  - Write deadline: 1s" -ForegroundColor White
            Write-Host "  - Buffer size: 1536" -ForegroundColor White
            Write-Host "  - Standard relay mechanisms" -ForegroundColor White
            
            [Environment]::SetEnvironmentVariable("USE_SHARED_MEMORY", "false", "Process")
            [Environment]::SetEnvironmentVariable("USE_DIRECT_P2P", "false", "Process")
            [Environment]::SetEnvironmentVariable("USE_MEMORY_CHANNEL", "false", "Process")
            [Environment]::SetEnvironmentVariable("APP_MODE", "business", "Process")
        }
        
        "pro" {
            Write-Host "🔧 Enabling Pro optimizations:" -ForegroundColor Cyan
            Write-Host "  - Write deadline: 1.5s" -ForegroundColor White
            Write-Host "  - Buffer size: 1280" -ForegroundColor White
            Write-Host "  - Enhanced relay mechanisms" -ForegroundColor White
            
            [Environment]::SetEnvironmentVariable("USE_SHARED_MEMORY", "false", "Process")
            [Environment]::SetEnvironmentVariable("USE_DIRECT_P2P", "false", "Process")
            [Environment]::SetEnvironmentVariable("USE_MEMORY_CHANNEL", "false", "Process")
            [Environment]::SetEnvironmentVariable("APP_MODE", "pro", "Process")
        }
        
        default {
            Write-Host "🆓 Using Free tier configuration:" -ForegroundColor Cyan
            Write-Host "  - Write deadline: 2s" -ForegroundColor White
            Write-Host "  - Buffer size: 512" -ForegroundColor White
            Write-Host "  - Basic relay mechanisms" -ForegroundColor White
            
            [Environment]::SetEnvironmentVariable("USE_SHARED_MEMORY", "false", "Process")
            [Environment]::SetEnvironmentVariable("USE_DIRECT_P2P", "false", "Process")
            [Environment]::SetEnvironmentVariable("USE_MEMORY_CHANNEL", "false", "Process")
            [Environment]::SetEnvironmentVariable("APP_MODE", "free", "Process")
        }
    }
    
    Write-Host ""
}

# Function to show current configuration
function Show-Configuration {
    Write-Host "📊 Current Configuration:" -ForegroundColor Green
    Write-Host "========================" -ForegroundColor Green
    Write-Host "TIER: $($env:TIER)" -ForegroundColor White
    Write-Host "APP_MODE: $($env:APP_MODE)" -ForegroundColor White
    Write-Host "USE_SHARED_MEMORY: $($env:USE_SHARED_MEMORY)" -ForegroundColor White
    Write-Host "USE_DIRECT_P2P: $($env:USE_DIRECT_P2P)" -ForegroundColor White
    Write-Host "USE_MEMORY_CHANNEL: $($env:USE_MEMORY_CHANNEL)" -ForegroundColor White
    Write-Host "OPTIMIZE_SYSTEM: $($env:OPTIMIZE_SYSTEM)" -ForegroundColor White
    Write-Host ""
}

# Function to build and run Bitcoin Sprint
function Start-BitcoinSprint {
    param(
        [string]$Mode = "run"
    )
    
    Write-Host "🔨 Building Bitcoin Sprint..." -ForegroundColor Yellow
    
    if ($Mode -eq "build-only") {
        # Just build the optimized version
        & go build -ldflags="-s -w -extldflags=-static" -trimpath -o "bitcoin-sprint-$($env:TIER).exe" .
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✅ Build successful: bitcoin-sprint-$($env:TIER).exe" -ForegroundColor Green
        } else {
            Write-Host "❌ Build failed" -ForegroundColor Red
        }
    } else {
        # Build and run
        & go run cmd/sprintd/main.go
    }
}

# Function to test the API
function Test-TurboStatus {
    Write-Host "🧪 Testing Turbo Status API..." -ForegroundColor Yellow
    
    Start-Sleep -Seconds 2
    
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:8080/api/turbo-status" -Method GET -ContentType "application/json"
        
        Write-Host "✅ API Response:" -ForegroundColor Green
        Write-Host "Tier: $($response.tier)" -ForegroundColor White
        Write-Host "Turbo Enabled: $($response.turboModeEnabled)" -ForegroundColor White
        Write-Host "Write Deadline: $($response.writeDeadline)" -ForegroundColor White
        Write-Host "Features: $($response.features -join ', ')" -ForegroundColor White
        Write-Host "Block Relay Target: $($response.performanceTargets.blockRelayLatency)" -ForegroundColor White
    } catch {
        Write-Host "❌ API test failed: $_" -ForegroundColor Red
    }
}

# Main script logic
Write-Host "Choose a performance tier:" -ForegroundColor Yellow
Write-Host "1. Free (Basic performance)" -ForegroundColor White
Write-Host "2. Pro (Enhanced performance)" -ForegroundColor White
Write-Host "3. Business (High performance)" -ForegroundColor White
Write-Host "4. Turbo (Ultra-low latency)" -ForegroundColor White
Write-Host "5. Enterprise (Maximum performance)" -ForegroundColor White
Write-Host ""

$choice = Read-Host "Enter your choice (1-5)"

switch ($choice) {
    "1" { Set-TierConfiguration -Tier "free" }
    "2" { Set-TierConfiguration -Tier "pro" }
    "3" { Set-TierConfiguration -Tier "business" }
    "4" { Set-TierConfiguration -Tier "turbo" }
    "5" { Set-TierConfiguration -Tier "enterprise" }
    default { 
        Write-Host "❌ Invalid choice, using Free tier" -ForegroundColor Red
        Set-TierConfiguration -Tier "free"
    }
}

Show-Configuration

Write-Host "Choose action:" -ForegroundColor Yellow
Write-Host "1. Build only" -ForegroundColor White
Write-Host "2. Build and run" -ForegroundColor White
Write-Host "3. Just show config" -ForegroundColor White

$action = Read-Host "Enter your choice (1-3)"

switch ($action) {
    "1" { Start-BitcoinSprint -Mode "build-only" }
    "2" { 
        Write-Host "Starting Bitcoin Sprint in $($env:TIER) mode..." -ForegroundColor Green
        Write-Host "Press Ctrl+C to stop" -ForegroundColor Yellow
        Write-Host ""
        Start-BitcoinSprint -Mode "run"
    }
    "3" { 
        Write-Host "Configuration displayed above. You can now manually run:" -ForegroundColor Green
        Write-Host "go run cmd/sprintd/main.go" -ForegroundColor Cyan
    }
    default { 
        Write-Host "No action taken." -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "💡 Tips:" -ForegroundColor Green
Write-Host "- Use TIER=turbo for ultra-low latency (<10ms block relay)" -ForegroundColor White
Write-Host "- Use TIER=enterprise for maximum performance (<5ms block relay)" -ForegroundColor White
Write-Host "- Monitor performance with: curl http://localhost:8080/api/turbo-status" -ForegroundColor White
Write-Host "- Check health with: curl http://localhost:8080/api/health" -ForegroundColor White
