# Performance test for entropy generation
Write-Host "=== ENTROPY PERFORMANCE TEST ===" -ForegroundColor Cyan
Write-Host ""

$times = @()
$samples = @()

1..5 | ForEach-Object {
    Write-Host "Test $_`: " -NoNewline

    $start = Get-Date
    try {
        $response = Invoke-RestMethod -Uri 'http://127.0.0.1:8080/api/v1/enterprise/entropy/fast' -Method POST -Body '{"size": 32}' -ContentType 'application/json' -Headers @{'X-API-Key'='turbo-api-key-2024'}
        $end = Get-Date
        $duration = ($end - $start).TotalMilliseconds

        $times += $duration
        $samples += $response.entropy

        Write-Host "$duration ms" -ForegroundColor Green
        Write-Host "  Sample: $($response.entropy.Substring(0, [Math]::Min(20, $response.entropy.Length)))..." -ForegroundColor Yellow
    } catch {
        Write-Host "ERROR: $($_.Exception.Message)" -ForegroundColor Red
    }
    Write-Host ""
}

# Calculate statistics
if ($times.Count -gt 0) {
    $avg = ($times | Measure-Object -Average).Average
    $min = ($times | Measure-Object -Minimum).Minimum
    $max = ($times | Measure-Object -Maximum).Maximum

    Write-Host "=== PERFORMANCE SUMMARY ===" -ForegroundColor Cyan
    Write-Host "Average Time: $([math]::Round($avg, 2)) ms" -ForegroundColor White
    Write-Host "Fastest: $min ms" -ForegroundColor Green
    Write-Host "Slowest: $max ms" -ForegroundColor Red
    Write-Host "Requests/second: $([math]::Round(1000/$avg, 1))" -ForegroundColor White
    Write-Host ""
    Write-Host "Sample entropy values:" -ForegroundColor Cyan
    $samples | ForEach-Object { 
        $sample = $_
        if ($sample.Length -gt 32) { $sample = $sample.Substring(0,32) }
        Write-Host "  $sample..." -ForegroundColor Yellow 
    }
}
