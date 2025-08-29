# Solana Load Testing and Multi-Validator Management Script
# Run comprehensive load tests and manage multiple Solana validators

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet("start-validators", "stop-validators", "load-test", "stress-test", "monitor", "cleanup")]
    [string]$Action = "load-test",

    [Parameter(Mandatory=$false)]
    [int]$Duration = 60,

    [Parameter(Mandatory=$false)]
    [int]$TxCount = 1000,

    [Parameter(Mandatory=$false)]
    [int]$Threads = 4
)

function Write-Header {
    param([string]$Text)
    Write-Host "`n=========================================" -ForegroundColor Cyan
    Write-Host " $Text" -ForegroundColor Yellow
    Write-Host "=========================================" -ForegroundColor Cyan
}

function Start-MultiValidators {
    Write-Header "STARTING MULTI-VALIDATOR SETUP"

    Write-Host "Starting additional Solana validators..." -ForegroundColor Green

    # Start validator 2
    Write-Host "Starting solana-validator-2..." -ForegroundColor Yellow
    docker-compose --profile multi-validator up -d solana-validator-2

    # Start validator 3
    Write-Host "Starting solana-validator-3..." -ForegroundColor Yellow
    docker-compose --profile multi-validator up -d solana-validator-3

    # Wait for validators to be healthy
    Write-Host "Waiting for validators to become healthy..." -ForegroundColor Yellow
    Start-Sleep -Seconds 30

    # Check validator status
    Write-Host "`nValidator Status:" -ForegroundColor Green
    docker ps --format "table {{.Names}}\t{{.Status}}" | findstr solana-validator

    Write-Host "`nMulti-validator setup complete!" -ForegroundColor Green
    Write-Host "Validators running on:" -ForegroundColor Cyan
    Write-Host "  - solana-validator:8899 (Primary)"
    Write-Host "  - solana-validator-2:8901 (Secondary)"
    Write-Host "  - solana-validator-3:8903 (Tertiary)"
}

function Stop-MultiValidators {
    Write-Header "STOPPING MULTI-VALIDATOR SETUP"

    Write-Host "Stopping additional validators..." -ForegroundColor Yellow
    docker-compose --profile multi-validator down

    Write-Host "Multi-validator cleanup complete!" -ForegroundColor Green
}

function Start-LoadTesting {
    Write-Header "STARTING SOLANA LOAD TESTING TOOL"

    Write-Host "Starting solana-bench-tps container..." -ForegroundColor Green
    docker-compose --profile load-testing up -d solana-bench-tps

    Write-Host "`nLoad testing tool is ready!" -ForegroundColor Green
    Write-Host "Container: solana-bench-tps" -ForegroundColor Cyan
}

function Run-LoadTest {
    param([int]$Duration, [int]$TxCount, [int]$Threads)

    Write-Header "RUNNING SOLANA LOAD TEST"
    Write-Host "Test Parameters:" -ForegroundColor Green
    Write-Host "  Duration: $Duration seconds" -ForegroundColor Cyan
    Write-Host "  Transactions: $TxCount" -ForegroundColor Cyan
    Write-Host "  Threads: $Threads" -ForegroundColor Cyan
    Write-Host "  Target: http://solana-validator:8899" -ForegroundColor Cyan

    Write-Host "`nStarting load test..." -ForegroundColor Yellow

    # Run the load test
    docker exec solana-bench-tps solana-bench-tps `
        --entrypoint http://solana-validator:8899 `
        --duration $Duration `
        --tx-count $TxCount `
        --thread-batch-sleep-ms 10 `
        --threads $Threads `
        --log

    Write-Host "`nLoad test completed!" -ForegroundColor Green
}

function Run-StressTest {
    Write-Header "RUNNING SOLANA STRESS TEST"

    Write-Host "Running comprehensive stress test..." -ForegroundColor Yellow

    # Test 1: Basic load test
    Write-Host "`n=== TEST 1: Basic Load (1000 tx, 30s) ===" -ForegroundColor Cyan
    Run-LoadTest -Duration 30 -TxCount 1000 -Threads 4

    # Test 2: Medium load test
    Write-Host "`n=== TEST 2: Medium Load (5000 tx, 60s) ===" -ForegroundColor Cyan
    Run-LoadTest -Duration 60 -TxCount 5000 -Threads 8

    # Test 3: High load test
    Write-Host "`n=== TEST 3: High Load (10000 tx, 120s) ===" -ForegroundColor Cyan
    Run-LoadTest -Duration 120 -TxCount 10000 -Threads 16

    Write-Host "`nStress test completed!" -ForegroundColor Green
}

function Monitor-Performance {
    Write-Header "MONITORING SOLANA PERFORMANCE"

    Write-Host "Current Metrics:" -ForegroundColor Green

    # Get current slot height
    $slotHeight = curl -s "http://localhost:9091/api/v1/query?query=solana_slot_height" | findstr "value"
    Write-Host "Slot Height: $slotHeight" -ForegroundColor Cyan

    # Get current TPS
    $tps = curl -s "http://localhost:9091/api/v1/query?query=solana_tps" | findstr "value"
    Write-Host "TPS: $tps" -ForegroundColor Cyan

    # Get validator count
    $validators = curl -s "http://localhost:9091/api/v1/query?query=solana_validator_count" | findstr "value"
    Write-Host "Validators: $validators" -ForegroundColor Cyan

    Write-Host "`nGrafana Dashboard: http://localhost:3000" -ForegroundColor Yellow
}

function Cleanup-LoadTesting {
    Write-Header "CLEANUP LOAD TESTING"

    Write-Host "Stopping load testing containers..." -ForegroundColor Yellow
    docker-compose --profile load-testing down

    Write-Host "Cleanup complete!" -ForegroundColor Green
}

# Main execution logic
switch ($Action) {
    "start-validators" {
        Start-MultiValidators
    }
    "stop-validators" {
        Stop-MultiValidators
    }
    "load-test" {
        Start-LoadTesting
        Run-LoadTest -Duration $Duration -TxCount $TxCount -Threads $Threads
    }
    "stress-test" {
        Start-LoadTesting
        Run-StressTest
    }
    "monitor" {
        Monitor-Performance
    }
    "cleanup" {
        Cleanup-LoadTesting
    }
    default {
        Write-Host "Usage: .\solana-load-test.ps1 -Action <action> [parameters]" -ForegroundColor Yellow
        Write-Host "`nActions:" -ForegroundColor Green
        Write-Host "  start-validators  - Start additional Solana validators"
        Write-Host "  stop-validators   - Stop additional validators"
        Write-Host "  load-test         - Run basic load test (default)"
        Write-Host "  stress-test       - Run comprehensive stress test"
        Write-Host "  monitor          - Show current performance metrics"
        Write-Host "  cleanup          - Clean up load testing containers"
        Write-Host "`nParameters:" -ForegroundColor Green
        Write-Host "  -Duration <sec>   - Test duration (default: 60)"
        Write-Host "  -TxCount <num>    - Number of transactions (default: 1000)"
        Write-Host "  -Threads <num>    - Number of threads (default: 4)"
    }
}

Write-Host "`nScript completed!" -ForegroundColor Green
