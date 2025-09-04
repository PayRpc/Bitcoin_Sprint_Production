# Quick Tier Performance Test Runner
# Runs the comprehensive tier comparison with optimized settings

param(
    [Parameter(Mandatory=$false)]
    [switch]$Fast,  # Quick test (10 seconds per tier)

    [Parameter(Mandatory=$false)]
    [switch]$Full,  # Full test (60 seconds per tier)

    [Parameter(Mandatory=$false)]
    [switch]$Custom,  # Custom settings

    [Parameter(Mandatory=$false)]
    [int]$Duration = 30,

    [Parameter(Mandatory=$false)]
    [int]$Concurrent = 5
)

$ErrorActionPreference = "Stop"

Write-Host "🚀 Bitcoin Sprint Tier Performance Test Runner" -ForegroundColor Cyan
Write-Host "=" * 50 -ForegroundColor Cyan

if ($Fast) {
    Write-Host "⚡ Running FAST test (10 seconds per tier)" -ForegroundColor Yellow
    & ".\tier-performance-comparison.ps1" -TestDuration 10 -ConcurrentRequests 3
}
elseif ($Full) {
    Write-Host "🏁 Running FULL test (60 seconds per tier)" -ForegroundColor Green
    & ".\tier-performance-comparison.ps1" -TestDuration 60 -ConcurrentRequests 10
}
elseif ($Custom) {
    Write-Host "🔧 Running CUSTOM test" -ForegroundColor Magenta
    Write-Host "Duration: $Duration seconds" -ForegroundColor White
    Write-Host "Concurrent requests: $Concurrent" -ForegroundColor White
    & ".\tier-performance-comparison.ps1" -TestDuration $Duration -ConcurrentRequests $Concurrent
}
else {
    Write-Host "📊 Running STANDARD test (30 seconds per tier)" -ForegroundColor Blue
    & ".\tier-performance-comparison.ps1" -TestDuration 30 -ConcurrentRequests 5
}

Write-Host "`n✅ Test completed! Check results above." -ForegroundColor Green
