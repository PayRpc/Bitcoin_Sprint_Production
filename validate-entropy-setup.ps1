# 🔍 Entropy Monitoring Validation Script

param(
    [switch]$QuickTest,
    [switch]$FullValidation,
    [string]$ApiKey = $env:API_KEY,
    [string]$ApiUrl = "http://127.0.0.1:8080"
)

function Write-Step {
    param([string]$Message)
    Write-Host "🔍 $Message" -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host "✅ $Message" -ForegroundColor Green
}

function Write-Error {
    param([string]$Message)
    Write-Host "❌ $Message" -ForegroundColor Red
}

function Test-ApiConnectivity {
    Write-Step "Testing API connectivity..."

    try {
        $headers = if ($ApiKey) { @{ "X-API-Key" = $ApiKey } } else { @{} }
        $response = Invoke-WebRequest -Uri "$ApiUrl/health" -Headers $headers -TimeoutSec 10
        if ($response.StatusCode -eq 200) {
            Write-Success "API is accessible"
            return $true
        }
    }
    catch {
        Write-Error "API connection failed: $($_.Exception.Message)"
        return $false
    }
}

function Test-EntropyMetrics {
    Write-Step "Testing entropy metrics availability..."

    try {
        $headers = if ($ApiKey) { @{ "X-API-Key" = $ApiKey } } else { @{} }
        $response = Invoke-WebRequest -Uri "$ApiUrl/metrics" -Headers $headers -TimeoutSec 10

        $metrics = $response.Content
        $entropyMetrics = $metrics | Select-String -Pattern "entropy|relay_cpu_temperature|fingerprint"

        if ($entropyMetrics.Count -gt 0) {
            Write-Success "Found $($entropyMetrics.Count) entropy-related metrics"
            if ($FullValidation) {
                Write-Host "📊 Available entropy metrics:" -ForegroundColor Yellow
                $entropyMetrics | ForEach-Object { Write-Host "   $($_.Line)" }
            }
            return $true
        } else {
            Write-Error "No entropy metrics found in /metrics endpoint"
            return $false
        }
    }
    catch {
        Write-Error "Failed to retrieve metrics: $($_.Exception.Message)"
        return $false
    }
}

function Test-EntropyEndpoints {
    Write-Step "Testing entropy-specific endpoints..."

    $endpoints = @(
        "/api/v1/entropy/fingerprint",
        "/api/v1/entropy/temperature",
        "/api/v1/entropy/status"
    )

    $successCount = 0
    foreach ($endpoint in $endpoints) {
        try {
            $headers = if ($ApiKey) { @{ "X-API-Key" = $ApiKey } } else { @{} }
            $response = Invoke-WebRequest -Uri "$ApiUrl$endpoint" -Headers $headers -TimeoutSec 10

            if ($response.StatusCode -eq 200) {
                Write-Success "Endpoint $endpoint is accessible"
                $successCount++
            }
        }
        catch {
            Write-Error "Endpoint $endpoint failed: $($_.Exception.Message)"
        }
    }

    return $successCount -gt 0
}

function Test-EnvironmentConfiguration {
    Write-Step "Validating environment configuration..."

    $requiredVars = @(
        "ENABLE_ENTROPY_MONITORING",
        "CPU_FINGERPRINT_ENABLED",
        "TEMPERATURE_MONITORING_ENABLED"
    )

    $configuredCount = 0
    foreach ($var in $requiredVars) {
        if (Test-Path env:$var) {
            Write-Success "Environment variable $var is set to: $(Get-Content env:$var)"
            $configuredCount++
        } else {
            Write-Error "Environment variable $var is not set"
        }
    }

    return $configuredCount -eq $requiredVars.Count
}

function Test-ServiceStatus {
    Write-Step "Checking Bitcoin Sprint service status..."

    $processes = Get-Process | Where-Object { $_.Name -like "*bitcoin*" -or $_.Name -like "*sprint*" }

    if ($processes.Count -gt 0) {
        Write-Success "Found $($processes.Count) Bitcoin Sprint related processes"
        if ($FullValidation) {
            $processes | ForEach-Object {
                Write-Host "   Process: $($_.Name) (ID: $($_.Id), CPU: $($_.CPU), Memory: $([math]::Round($_.WorkingSet64 / 1MB, 2)) MB)" -ForegroundColor Gray
            }
        }
        return $true
    } else {
        Write-Error "No Bitcoin Sprint processes found"
        return $false
    }
}

function Show-ValidationSummary {
    param([array]$Results)

    Write-Host "`n📋 Validation Summary:" -ForegroundColor Yellow
    Write-Host "═══════════════════════════════════════" -ForegroundColor Yellow

    $totalTests = $Results.Count
    $passedTests = ($Results | Where-Object { $_ }).Count
    $failedTests = $totalTests - $passedTests

    Write-Host "Total Tests: $totalTests" -ForegroundColor White
    Write-Host "Passed: $passedTests" -ForegroundColor Green
    Write-Host "Failed: $failedTests" -ForegroundColor Red

    if ($failedTests -eq 0) {
        Write-Host "`n🎉 All validation tests passed! Entropy monitoring is working correctly." -ForegroundColor Green
    } else {
        Write-Host "`n⚠️  Some tests failed. Check the output above for details." -ForegroundColor Yellow
    }
}

# Main validation logic
Write-Host "🚀 Starting Entropy Monitoring Validation" -ForegroundColor Magenta
Write-Host "═══════════════════════════════════════" -ForegroundColor Magenta

$testResults = @()

# Quick test mode
if ($QuickTest) {
    Write-Host "Running quick validation tests..." -ForegroundColor Yellow

    $testResults += Test-ApiConnectivity
    $testResults += Test-EntropyMetrics
    $testResults += Test-EnvironmentConfiguration

    Show-ValidationSummary -Results $testResults
    exit
}

# Full validation mode
if ($FullValidation) {
    Write-Host "Running comprehensive validation tests..." -ForegroundColor Yellow

    $testResults += Test-ServiceStatus
    $testResults += Test-EnvironmentConfiguration
    $testResults += Test-ApiConnectivity
    $testResults += Test-EntropyMetrics
    $testResults += Test-EntropyEndpoints

    Show-ValidationSummary -Results $testResults
    exit
}

# Default: Run all tests
Write-Host "Running complete validation suite..." -ForegroundColor Yellow

$testResults += Test-ServiceStatus
$testResults += Test-EnvironmentConfiguration
$testResults += Test-ApiConnectivity
$testResults += Test-EntropyMetrics
$testResults += Test-EntropyEndpoints

Show-ValidationSummary -Results $testResults

# Provide next steps
Write-Host "`n💡 Next Steps:" -ForegroundColor Cyan
Write-Host "1. If all tests passed, entropy monitoring is working correctly!" -ForegroundColor White
Write-Host "2. Run continuous monitoring: .\monitor-entropy.ps1 -Continuous" -ForegroundColor White
Write-Host "3. Check metrics in Prometheus format: curl http://127.0.0.1:8080/metrics" -ForegroundColor White
Write-Host "4. For troubleshooting, check application logs in the logs\ directory" -ForegroundColor White
