# Bitcoin Sprint with Entropy Monitoring Startup Script
# Loads turbo configuration and starts entropy monitoring

param(
    [switch]$MonitorOnly,
    [switch]$Background,
    [int]$MonitorInterval = 30
)

$ErrorActionPreference = "Stop"

function Write-StartupSection($title) {
    Write-Host ""
    Write-Host "=" * 70 -ForegroundColor Cyan
    Write-Host $title -ForegroundColor Yellow
    Write-Host "=" * 70 -ForegroundColor Cyan
}

function Start-EntropyMonitoring {
    Write-Host "🔐 Starting entropy monitoring..." -ForegroundColor Cyan

    $monitorJob = Start-Job -ScriptBlock {
        param($interval)
        while ($true) {
            try {
                # Call the entropy monitor script
                & "$PSScriptRoot\monitor-entropy.ps1" -Continuous -IntervalSeconds $interval
            } catch {
                Write-Host "Monitor error: $($_.Exception.Message)" -ForegroundColor Red
                Start-Sleep -Seconds 5
            }
        }
    } -ArgumentList $MonitorInterval

    Write-Host "✅ Entropy monitoring started (Job ID: $($monitorJob.Id))" -ForegroundColor Green
    return $monitorJob
}

# Main startup sequence
Write-StartupSection "🚀 BITCOIN SPRINT ENTROPY MONITORING STARTUP"

Write-Host "Loading turbo configuration with entropy monitoring..." -ForegroundColor Cyan
Write-Host ""

# Load environment variables from .env.turbo
if (Test-Path ".env.turbo") {
    Write-Host "📄 Loading .env.turbo configuration..." -ForegroundColor Gray
    $envContent = Get-Content ".env.turbo" -Raw

    # Parse and set environment variables (simple implementation)
    $envContent -split "`n" | ForEach-Object {
        $line = $_.Trim()
        if ($line -and -not $line.StartsWith("#")) {
            $parts = $line -split "=", 2
            if ($parts.Count -eq 2) {
                $key = $parts[0].Trim()
                $value = $parts[1].Trim()
                [Environment]::SetEnvironmentVariable($key, $value)
                Write-Host "  ✅ $key = $value" -ForegroundColor Gray
            }
        }
    }
} else {
    Write-Host "⚠️  .env.turbo not found, using default configuration" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "🔧 Entropy Configuration Applied:" -ForegroundColor Cyan
Write-Host "  • Tier: $($env:TIER)" -ForegroundColor Gray
Write-Host "  • Entropy Monitoring: $($env:ENABLE_ENTROPY_MONITORING)" -ForegroundColor Gray
Write-Host "  • Hardware Fingerprinting: $($env:CPU_FINGERPRINT_ENABLED)" -ForegroundColor Gray
Write-Host "  • Temperature Monitoring: $($env:TEMPERATURE_MONITORING_ENABLED)" -ForegroundColor Gray
Write-Host "  • Security Level: $($env:ENTROPY_SECURITY_LEVEL)" -ForegroundColor Gray
Write-Host ""

if (-not $MonitorOnly) {
    Write-StartupSection "🌟 STARTING BITCOIN SPRINT SERVICE"

    # Start Bitcoin Sprint in background
    Write-Host "🚀 Launching Bitcoin Sprint with entropy enhancements..." -ForegroundColor Green

    if ($Background) {
        $sprintJob = Start-Job -ScriptBlock {
            try {
                # Build and start the service
                & go build -o bitcoin-sprint-entropy.exe ./cmd/sprintd
                if ($LASTEXITCODE -eq 0) {
                    Write-Host "Bitcoin Sprint built successfully" -ForegroundColor Green
                    & ./bitcoin-sprint-entropy.exe
                } else {
                    Write-Host "Build failed" -ForegroundColor Red
                }
            } catch {
                Write-Host "Service error: $($_.Exception.Message)" -ForegroundColor Red
            }
        }

        Write-Host "✅ Bitcoin Sprint started in background (Job ID: $($sprintJob.Id))" -ForegroundColor Green
        Start-Sleep -Seconds 5
    } else {
        # Start in foreground
        Write-Host "Starting Bitcoin Sprint (press Ctrl+C to stop)..." -ForegroundColor Yellow
        & go build -o bitcoin-sprint-entropy.exe ./cmd/sprintd
        if ($LASTEXITCODE -eq 0) {
            & ./bitcoin-sprint-entropy.exe
        }
        return
    }
}

# Start entropy monitoring
$monitorJob = Start-EntropyMonitoring

Write-Host ""
Write-StartupSection "📊 MONITORING DASHBOARD"

Write-Host "🔗 Service Endpoints:" -ForegroundColor Cyan
Write-Host "  • API Server: http://127.0.0.1:8080" -ForegroundColor Gray
Write-Host "  • Metrics: http://127.0.0.1:8080/metrics" -ForegroundColor Gray
Write-Host "  • Health Check: http://127.0.0.1:8080/health" -ForegroundColor Gray
Write-Host ""

Write-Host "📈 Entropy Metrics Available:" -ForegroundColor Cyan
Write-Host "  • relay_cpu_temperature - Current CPU temperature" -ForegroundColor Gray
Write-Host "  • entropy_sources_active - Number of active entropy sources" -ForegroundColor Gray
Write-Host "  • entropy_system_fingerprint_available - System fingerprint status" -ForegroundColor Gray
Write-Host "  • entropy_hardware_sources_available - Hardware entropy sources" -ForegroundColor Gray
Write-Host ""

Write-Host "🛡️  Security Features Active:" -ForegroundColor Green
Write-Host "  • Hardware fingerprinting for VM cloning resistance" -ForegroundColor Gray
Write-Host "  • CPU temperature entropy mixing" -ForegroundColor Gray
Write-Host "  • Hybrid entropy combining OS RNG + blockchain + hardware" -ForegroundColor Gray
Write-Host "  • Real-time entropy quality monitoring" -ForegroundColor Gray
Write-Host ""

if ($Background) {
    Write-Host "🎯 Next Steps:" -ForegroundColor Yellow
    Write-Host "  1. Wait 10-15 seconds for services to fully start" -ForegroundColor Gray
    Write-Host "  2. Check metrics: curl http://127.0.0.1:8080/metrics" -ForegroundColor Gray
    Write-Host "  3. Monitor entropy: .\monitor-entropy.ps1 -Continuous" -ForegroundColor Gray
    Write-Host "  4. View logs: Get-Job | Receive-Job" -ForegroundColor Gray
    Write-Host ""

    Write-Host "🛑 To stop all services:" -ForegroundColor Red
    Write-Host "   Get-Job | Stop-Job; Get-Job | Remove-Job" -ForegroundColor Gray
}

Write-Host ""
Write-Host "✅ Entropy monitoring is now active!" -ForegroundColor Green</content>
<parameter name="filePath">c:\Projects\Bitcoin_Sprint_final_1\BItcoin_Sprint\start-entropy-monitoring.ps1
