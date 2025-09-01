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
    param($endpoint, $description, $Method = "GET", $Body = $null)

    try {
        $headers = @{ "X-API-Key" = $ApiKey }
        $params = @{
            Uri = "$ApiUrl$endpoint"
            Headers = $headers
            TimeoutSec = 5
            Method = $Method
        }

        if ($Body) {
            $params["Body"] = $Body
            $params["ContentType"] = "application/json"
        }

        $response = Invoke-RestMethod @params

        Write-Host "  ✅ $description available" -ForegroundColor Green
        return $response
    } catch {
        Write-Host "  ❌ $description failed: $($_.Exception.Message)" -ForegroundColor Red
        return $null
    }
}

function Test-EntropyFunctions {
    Write-MonitorSection "ENTROPY FUNCTION TESTS"

    Write-Host "Testing enhanced entropy functions..." -ForegroundColor Cyan
    Write-Host ""

    # Test system fingerprint
    $fingerprint = Test-EntropyEndpoint "/api/v1/enterprise/system/fingerprint" "System Fingerprint"
    if ($fingerprint) {
        Write-Metric "Fingerprint Length" "$($fingerprint.fingerprint.Length/2) bytes" "" "good"
        Write-Host "    Sample: $($fingerprint.fingerprint.Substring(0, [Math]::Min(32, $fingerprint.fingerprint.Length)))..." -ForegroundColor Gray
    }

    # Test CPU temperature
    $temperature = Test-EntropyEndpoint "/api/v1/enterprise/system/temperature" "CPU Temperature"
    if ($temperature) {
        Write-Metric "Current Temperature" "$($temperature.temperature)" "°C" "good"
    }

    # Test fast entropy
    $fastEntropy = Test-EntropyEndpoint "/api/v1/enterprise/entropy/fast" "Fast Entropy" -Method "POST" -Body '{"size": 32}'
    if ($fastEntropy) {
        Write-Metric "Fast Entropy Length" "$($fastEntropy.size) bytes" "" "good"
        Write-Host "    Sample: $($fastEntropy.entropy.Substring(0, [Math]::Min(32, $fastEntropy.entropy.Length)))..." -ForegroundColor Gray
    }

    # Test hybrid entropy
    $hybridEntropy = Test-EntropyEndpoint "/api/v1/enterprise/entropy/hybrid" "Hybrid Entropy" -Method "POST" -Body '{"size": 32, "headers": []}'
    if ($hybridEntropy) {
        Write-Metric "Hybrid Entropy Length" "$($hybridEntropy.size) bytes" "" "good"
        Write-Host "    Sample: $($hybridEntropy.entropy.Substring(0, [Math]::Min(32, $hybridEntropy.entropy.Length)))..." -ForegroundColor Gray
    }
}

# Main execution
Write-MonitorSection "BITCOIN SPRINT ENTROPY MONITOR"

Write-Host "API Endpoint: $ApiUrl" -ForegroundColor Gray
Write-Host "API Key: $($ApiKey.Substring(0, [Math]::Min(8, $ApiKey.Length)))..." -ForegroundColor Gray
Write-Host "Update Interval: $IntervalSeconds seconds" -ForegroundColor Gray
Write-Host ""

# Test connection first
Write-Host "Testing API connectivity..." -ForegroundColor Cyan
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

    Write-Host ""
    Write-Host "Use -Continuous switch for real-time monitoring" -ForegroundColor Cyan
    Write-Host "Use -TestEntropyFunctions to test all entropy functions" -ForegroundColor Cyan