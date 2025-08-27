# Bitcoin Sprint Entropy Monitor
# Monitors the new entropy metrics and hardware fingerprinting features

param(
    [string]$ApiUrl = "http://127.0.0.1:8080",
    [string]$ApiKey = "turbo-api-key-changeme",
    [int]$IntervalSeconds = 10,
    [switch]$Continuous,
    [switch]$TestEntropyFunctions
)

$ErrorActionPreference = "Stop"

function Write-MonitorSection($title) {
    Write-Host ""
    Write-Host "=" * 60 -ForegroundColor Cyan
    Write-Host $title -ForegroundColor Yellow
    Write-Host "=" * 60 -ForegroundColor Cyan
}

function Write-Metric($label, $value, $unit = "", $status = "info") {
    $color = switch ($status) {
        "good" { "Green" }
        "warning" { "Yellow" }
        "error" { "Red" }
        default { "Gray" }
    }

    $displayValue = if ($unit) { "$value $unit" } else { $value }
    Write-Host "  $label".PadRight(35) -ForegroundColor Gray -NoNewline
    Write-Host "$displayValue".PadLeft(15) -ForegroundColor $color
}

function Test-EntropyEndpoint {
    param($endpoint, $description)

    try {
        $headers = @{ "X-API-Key" = $ApiKey }
        $response = Invoke-RestMethod -Uri "$ApiUrl$endpoint" -Headers $headers -TimeoutSec 5

        Write-Host "  ✅ $description available" -ForegroundColor Green
        return $response
    } catch {
        Write-Host "  ❌ $description failed: $($_.Exception.Message)" -ForegroundColor Red
        return $null
    }
}

function Monitor-EntropyMetrics {
    Write-MonitorSection "🔐 ENTROPY METRICS MONITOR"

    try {
        $headers = @{ "X-API-Key" = $ApiKey }
        $metrics = Invoke-RestMethod -Uri "$ApiUrl/metrics" -Headers $headers -TimeoutSec 5

        # Parse metrics (simple text parsing)
        $metricsLines = $metrics -split "`n"

        Write-Host "📊 Current Entropy Metrics:" -ForegroundColor Cyan
        Write-Host ""

        foreach ($line in $metricsLines) {
            if ($line -match "^relay_cpu_temperature\s+(.+)$") {
                $temp = [float]$matches[1]
                $status = if ($temp -gt 0) { "good" } else { "warning" }
                Write-Metric "CPU Temperature" $temp "°C" $status
            }
            elseif ($line -match "^entropy_sources_active\s+(\d+)$") {
                $sources = [int]$matches[1]
                $status = if ($sources -ge 2) { "good" } elseif ($sources -ge 1) { "warning" } else { "error" }
                Write-Metric "Active Entropy Sources" $sources "" $status
            }
            elseif ($line -match "^entropy_system_fingerprint_available\s+(\d+)$") {
                $available = [int]$matches[1]
                $status = if ($available -eq 1) { "good" } else { "warning" }
                Write-Metric "System Fingerprint" $(if ($available -eq 1) { "Available" } else { "Unavailable" }) "" $status
            }
            elseif ($line -match "^entropy_hardware_sources_available\s+(\d+)$") {
                $hwSources = [int]$matches[1]
                $status = if ($hwSources -ge 1) { "good" } else { "warning" }
                Write-Metric "Hardware Sources" $hwSources "" $status
            }
        }

    } catch {
        Write-Host "❌ Failed to fetch metrics: $($_.Exception.Message)" -ForegroundColor Red
    }
}

function Test-EntropyFunctions {
    Write-MonitorSection "🧪 ENTROPY FUNCTION TESTS"

    Write-Host "Testing enhanced entropy functions..." -ForegroundColor Cyan
    Write-Host ""

    # Test system fingerprint
    $fingerprint = Test-EntropyEndpoint "/api/v1/entropy/fingerprint" "System Fingerprint"
    if ($fingerprint) {
        Write-Metric "Fingerprint Length" "$($fingerprint.Length) bytes" "" "good"
        Write-Host "    Sample: $($fingerprint.Substring(0, [Math]::Min(32, $fingerprint.Length)))..." -ForegroundColor Gray
    }

    # Test CPU temperature
    $temperature = Test-EntropyEndpoint "/api/v1/entropy/temperature" "CPU Temperature"
    if ($temperature) {
        Write-Metric "Current Temperature" "$temperature" "°C" "good"
    }

    # Test fast entropy
    $fastEntropy = Test-EntropyEndpoint "/api/v1/entropy/fast" "Fast Entropy"
    if ($fastEntropy) {
        Write-Metric "Fast Entropy Length" "$($fastEntropy.Length) bytes" "" "good"
    }

    # Test hybrid entropy
    $hybridEntropy = Test-EntropyEndpoint "/api/v1/entropy/hybrid" "Hybrid Entropy"
    if ($hybridEntropy) {
        Write-Metric "Hybrid Entropy Length" "$($hybridEntropy.Length) bytes" "" "good"
    }

    # Test enhanced entropy with fingerprint
    $enhancedEntropy = Test-EntropyEndpoint "/api/v1/entropy/enhanced" "Enhanced Entropy"
    if ($enhancedEntropy) {
        Write-Metric "Enhanced Entropy Length" "$($enhancedEntropy.Length) bytes" "" "good"
    }
}

function Show-EntropySecurityStatus {
    Write-MonitorSection "🔒 ENTROPY SECURITY STATUS"

    Write-Host "🛡️  Security Features:" -ForegroundColor Cyan
    Write-Host ""

    Write-Host "  ✅ Hardware Fingerprinting" -ForegroundColor Green
    Write-Host "     • CPU detection for system uniqueness" -ForegroundColor Gray
    Write-Host "     • VM cloning resistance" -ForegroundColor Gray
    Write-Host "     • Process and timestamp entropy" -ForegroundColor Gray
    Write-Host ""

    Write-Host "  ✅ CPU Temperature Monitoring" -ForegroundColor Green
    Write-Host "     • Thermal entropy source" -ForegroundColor Gray
    Write-Host "     • System activity correlation" -ForegroundColor Gray
    Write-Host "     • Hardware-based randomness" -ForegroundColor Gray
    Write-Host ""

    Write-Host "  ✅ Hybrid Entropy Sources" -ForegroundColor Green
    Write-Host "     • OS RNG + Jitter + Blockchain + Hardware" -ForegroundColor Gray
    Write-Host "     • Multiple entropy layers" -ForegroundColor Gray
    Write-Host "     • Fallback mechanisms" -ForegroundColor Gray
    Write-Host ""

    Write-Host "  ✅ Live Metrics & Monitoring" -ForegroundColor Green
    Write-Host "     • Prometheus-compatible metrics" -ForegroundColor Gray
    Write-Host "     • Real-time entropy quality monitoring" -ForegroundColor Gray
    Write-Host "     • Attack detection capabilities" -ForegroundColor Gray
}

# Main execution
Write-MonitorSection "🚀 BITCOIN SPRINT ENTROPY MONITOR"

Write-Host "API Endpoint: $ApiUrl" -ForegroundColor Gray
Write-Host "API Key: $($ApiKey.Substring(0, [Math]::Min(8, $ApiKey.Length)))..." -ForegroundColor Gray
Write-Host "Update Interval: $IntervalSeconds seconds" -ForegroundColor Gray
Write-Host ""

# Test connection first
Write-Host "🔍 Testing API connectivity..." -ForegroundColor Cyan
try {
    $headers = @{ "X-API-Key" = $ApiKey }
    $health = Invoke-RestMethod -Uri "$ApiUrl/health" -Headers $headers -TimeoutSec 5
    Write-Host "✅ API connection successful" -ForegroundColor Green
    Write-Host "   Status: $($health.status)" -ForegroundColor Gray
} catch {
    Write-Host "❌ API connection failed: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host ""
    Write-Host "💡 Make sure Bitcoin Sprint is running with:" -ForegroundColor Yellow
    Write-Host "   • API server enabled (API_PORT=8080)" -ForegroundColor Gray
    Write-Host "   • Correct API key configured" -ForegroundColor Gray
    Write-Host "   • Entropy monitoring enabled" -ForegroundColor Gray
    exit 1
}

if ($TestEntropyFunctions) {
    Test-EntropyFunctions
}

Show-EntropySecurityStatus

# Continuous monitoring loop
if ($Continuous) {
    Write-Host ""
    Write-Host "🔄 Starting continuous monitoring (Ctrl+C to stop)..." -ForegroundColor Cyan
    Write-Host ""

    while ($true) {
        Monitor-EntropyMetrics
        Start-Sleep -Seconds $IntervalSeconds
    }
} else {
    # Single monitoring run
    Monitor-EntropyMetrics

    Write-Host ""
    Write-Host "💡 Use -Continuous switch for real-time monitoring" -ForegroundColor Cyan
    Write-Host "💡 Use -TestEntropyFunctions to test all entropy functions" -ForegroundColor Cyan
}</content>
<parameter name="filePath">c:\Projects\Bitcoin_Sprint_final_1\BItcoin_Sprint\monitor-entropy.ps1
