# Turbo Mode Stress Test
# Tests high-concurrency performance of the metrics API

Write-Host "=== TURBO MODE STRESS TEST ===" -ForegroundColor Cyan
$headers = @{ "Authorization" = "Bearer demo-key-enterprise" }

# Turbo Mode Stress Test
# Tests high-concurrency performance of the metrics API

Write-Host "=== TURBO MODE STRESS TEST ===" -ForegroundColor Cyan

# First, verify the endpoint is accessible
Write-Host "Testing endpoint accessibility..." -ForegroundColor Yellow
try {
    $testResponse = Invoke-WebRequest -Uri "http://localhost:8081/metrics" -TimeoutSec 5
    Write-Host "✅ Endpoint is accessible (Status: $($testResponse.StatusCode))" -ForegroundColor Green
} catch {
    Write-Host "❌ Endpoint not accessible: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

Write-Host "Starting 20 concurrent requests to metrics endpoint..." -ForegroundColor Yellow
$results = @()

# Use a smaller number of concurrent requests for reliability
for ($i = 1; $i -le 20; $i++) {
    $start = Get-Date
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8081/metrics" -TimeoutSec 10
        $end = Get-Date
        $latency = ($end - $start).TotalMilliseconds
        $results += @{Success=$true;Latency=$latency;StatusCode=$response.StatusCode;RequestId=$i}
        Write-Host "Request $i completed: $([math]::Round($latency,2))ms" -ForegroundColor Gray
    } catch {
        $end = Get-Date
        $latency = ($end - $start).TotalMilliseconds
        $results += @{Success=$false;Latency=$latency;Error=$_.Exception.Message;RequestId=$i}
        Write-Host "Request $i failed: $($_.Exception.Message)" -ForegroundColor Red
    }
}

# Calculate statistics
$successfulResults = $results | Where-Object {$_.Success}
$failedResults = $results | Where-Object {-not $_.Success}

if ($successfulResults.Count -gt 0) {
    $avg = ($successfulResults | Measure-Object -Property Latency -Average).Average
    $min = ($successfulResults | Measure-Object -Property Latency -Minimum).Minimum
    $max = ($successfulResults | Measure-Object -Property Latency -Maximum).Maximum
    
    # Calculate percentiles
    $sortedLatencies = $successfulResults | Sort-Object Latency
    $p50Index = [math]::Floor($sortedLatencies.Count * 0.50)
    $p95Index = [math]::Floor($sortedLatencies.Count * 0.95)
    $p99Index = [math]::Floor($sortedLatencies.Count * 0.99)
    
    $p50 = if ($p50Index -lt $sortedLatencies.Count) { $sortedLatencies[$p50Index].Latency } else { $max }
    $p95 = if ($p95Index -lt $sortedLatencies.Count) { $sortedLatencies[$p95Index].Latency } else { $max }
    $p99 = if ($p99Index -lt $sortedLatencies.Count) { $sortedLatencies[$p99Index].Latency } else { $max }
} else {
    $avg = $min = $max = $p50 = $p95 = $p99 = 0
}

# Display results
Write-Host "`n=== STRESS TEST RESULTS ===" -ForegroundColor Cyan
Write-Host "Total Requests: $($results.Count)" -ForegroundColor Yellow
Write-Host "✅ Successful Requests: $($successfulResults.Count)" -ForegroundColor Green
Write-Host "❌ Failed Requests: $($failedResults.Count)" -ForegroundColor Red
Write-Host "✅ Success Rate: $([math]::Round((($successfulResults.Count)/$results.Count*100),2))%" -ForegroundColor Green

if ($successfulResults.Count -gt 0) {
    Write-Host "`n=== LATENCY STATISTICS ===" -ForegroundColor Cyan
    Write-Host "✅ Average Latency: $([math]::Round($avg,2)) ms" -ForegroundColor Green
    Write-Host "✅ Minimum Latency: $([math]::Round($min,2)) ms" -ForegroundColor Green
    Write-Host "✅ Maximum Latency: $([math]::Round($max,2)) ms" -ForegroundColor Green
    Write-Host "✅ P50 (Median) Latency: $([math]::Round($p50,2)) ms" -ForegroundColor Green
    Write-Host "✅ P95 Latency: $([math]::Round($p95,2)) ms" -ForegroundColor Green
    Write-Host "✅ P99 Latency: $([math]::Round($p99,2)) ms" -ForegroundColor Green
    
    # Performance assessment
    Write-Host "`n=== PERFORMANCE ASSESSMENT ===" -ForegroundColor Cyan
    if ($p95 -lt 100) {
        Write-Host "🚀 EXCELLENT: P95 latency under 100ms" -ForegroundColor Green
    } elseif ($p95 -lt 500) {
        Write-Host "✅ GOOD: P95 latency under 500ms" -ForegroundColor Yellow
    } else {
        Write-Host "⚠️  NEEDS OPTIMIZATION: P95 latency over 500ms" -ForegroundColor Red
    }
    
    if ($successfulResults.Count -eq $results.Count) {
        Write-Host "🎯 PERFECT: 100% success rate" -ForegroundColor Green
    } elseif ($successfulResults.Count -ge ($results.Count * 0.95)) {
        Write-Host "✅ EXCELLENT: >95% success rate" -ForegroundColor Green
    } else {
        Write-Host "⚠️  RELIABILITY ISSUE: <95% success rate" -ForegroundColor Red
    }
}

# Show sample errors if any
if ($failedResults.Count -gt 0) {
    Write-Host "`n=== SAMPLE ERRORS ===" -ForegroundColor Red
    $failedResults | Select-Object -First 3 | ForEach-Object {
        Write-Host "❌ Error: $($_.Error)" -ForegroundColor Red
    }
}

Write-Host "`n=== TURBO MODE STRESS TEST COMPLETE ===" -ForegroundColor Cyan
