# Wrapper to launch the installer with admin privileges
Write-Host "Bitcoin Sprint Service Installer"
Write-Host "----------------------------"

$binPath = Join-Path (Get-Location) "bitcoin-sprint.exe"
if (-not (Test-Path $binPath)) {
    Write-Error "Bitcoin Sprint executable not found at $binPath"
    Exit 1
}

# Check if running as admin already
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")

if ($isAdmin) {
    Write-Host "Already running as administrator, proceeding with installation..."
    # Run the installation directly
    & .\quick-install.ps1
} else {
    Write-Host "Requesting administrator privileges..."
    
    # Get paths
    $scriptPath = Join-Path (Get-Location) "quick-install.ps1"
    
    # Start a new elevated PowerShell process
    try {
        Start-Process powershell -ArgumentList "-ExecutionPolicy Bypass -File `"$scriptPath`"" -Verb RunAs -Wait
        Write-Host "Installation process completed."
    } catch {
        Write-Error "Failed to launch elevated process: $_"
        Exit 1
    }
}
