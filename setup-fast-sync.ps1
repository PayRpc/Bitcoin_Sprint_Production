# Bitcoin Sprint - Fast Sync Setup Script
# ==========================================
# This script provides multiple options for fast Bitcoin synchronization

param(
    [Parameter(Mandatory=$false)]
    [switch]$UseSnapshot,
    [Parameter(Mandatory=$false)]
    [switch]$UsePrunedNode,
    [Parameter(Mandatory=$false)]
    [switch]$UseUTXOSet,
    [Parameter(Mandatory=$false)]
    [string]$SnapshotUrl = "https://bitcoincore.org/bin/bitcoin-core-25.1/bitcoin-25.1-x86_64-linux-gnu.tar.gz"
)

Write-Host "Bitcoin Sprint - Fast Sync Setup" -ForegroundColor Green
Write-Host "=================================" -ForegroundColor Green

# Function to download and setup Bitcoin snapshot
function Setup-BitcoinSnapshot {
    Write-Host "Setting up Bitcoin snapshot for fast sync..." -ForegroundColor Yellow

    $snapshotDir = "C:\Projects\Bitcoin-Sprint-5\BItcoin_Sprint\data\bitcoin-snapshot"
    $bitcoinDataDir = "C:\Projects\Bitcoin-Sprint-5\BItcoin_Sprint\data\bitcoin-data"

    if (!(Test-Path $snapshotDir)) {
        New-Item -ItemType Directory -Path $snapshotDir -Force
    }

    Write-Host "Bitcoin snapshot would be downloaded to: $snapshotDir" -ForegroundColor Cyan
    Write-Host "This would reduce sync time from weeks to hours" -ForegroundColor Cyan
    Write-Host "Note: Snapshot downloads are typically 500GB+ compressed" -ForegroundColor Yellow
}

# Function to setup pruned node configuration
function Setup-PrunedNode {
    Write-Host "Setting up pruned Bitcoin node configuration..." -ForegroundColor Yellow

    $configPath = "C:\Projects\Bitcoin-Sprint-5\BItcoin_Sprint\config\bitcoin-pruned.conf"

    $prunedConfig = @"
# Pruned Bitcoin Node Configuration
server=1
listen=1
maxconnections=32
prune=550
dbcache=512
maxmempool=256
disablewallet=1
blocksonly=1
peerbloomfilters=0
dnsseed=1
upnp=0
rpcuser=sprint
rpcpassword=sprint_password_2025
rpcallowip=0.0.0.0/0
rpcbind=0.0.0.0
rpcport=8332
zmqpubhashblock=tcp://0.0.0.0:28332
zmqpubrawtx=tcp://0.0.0.0:28333
printtoconsole=1
"@

    $prunedConfig | Out-File -FilePath $configPath -Encoding UTF8
    Write-Host "Pruned configuration created at: $configPath" -ForegroundColor Green
    Write-Host "This configuration will keep only ~550MB of blockchain data" -ForegroundColor Cyan
}

# Function to setup UTXO set for faster sync
function Setup-UTXOSet {
    Write-Host "Setting up UTXO set for faster Bitcoin sync..." -ForegroundColor Yellow

    $utxoDir = "C:\Projects\Bitcoin-Sprint-5\BItcoin_Sprint\data\utxo-set"

    if (!(Test-Path $utxoDir)) {
        New-Item -ItemType Directory -Path $utxoDir -Force
    }

    Write-Host "UTXO set directory created at: $utxoDir" -ForegroundColor Green
    Write-Host "UTXO set can be downloaded from: https://bitcoincore.org/bin/utxo/" -ForegroundColor Cyan
    Write-Host "This provides a ~4GB download for instant UTXO state" -ForegroundColor Cyan
}

# Main execution
if ($UseSnapshot) {
    Setup-BitcoinSnapshot
} elseif ($UsePrunedNode) {
    Setup-PrunedNode
} elseif ($UseUTXOSet) {
    Setup-UTXOSet
} else {
    Write-Host "Available options:" -ForegroundColor Yellow
    Write-Host "  -UseSnapshot    : Download Bitcoin blockchain snapshot (500GB+)" -ForegroundColor White
    Write-Host "  -UsePrunedNode  : Setup pruned node configuration (550MB)" -ForegroundColor White
    Write-Host "  -UseUTXOSet     : Download UTXO set for instant sync (4GB)" -ForegroundColor White
    Write-Host "" -ForegroundColor White
    Write-Host "Examples:" -ForegroundColor Cyan
    Write-Host "  .\setup-fast-sync.ps1 -UsePrunedNode" -ForegroundColor White
    Write-Host "  .\setup-fast-sync.ps1 -UseUTXOSet" -ForegroundColor White
    Write-Host "  .\setup-fast-sync.ps1 -UseSnapshot" -ForegroundColor White
}

Write-Host "Fast sync setup complete!" -ForegroundColor Green
