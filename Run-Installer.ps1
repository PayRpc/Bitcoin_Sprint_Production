# Self-elevating wrapper for install.ps1
# Automatically requests admin rights if needed

# Check if running as administrator
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")

if (-not $isAdmin) {
    Write-Host "Requesting administrative privileges..." -ForegroundColor Yellow
    
    # Get the full path to the script we're running
    $scriptPath = Join-Path (Get-Location) "install.ps1"
    $binPath = Join-Path (Get-Location) "bitcoin-sprint.exe"
    
    # Calculate SHA256 of the executable
    $sha256 = (Get-FileHash -Path $binPath -Algorithm SHA256).Hash
    
    # Prepare the command to run with elevation
    $installCommand = "-ExecutionPolicy Bypass -Command `"cd '$((Get-Location))'; .\install.ps1 -BinaryUrl 'file:///$($binPath.Replace('\', '/'))' -BinarySha256 '$sha256' -Force`""
    
    # Start a new PowerShell process with admin rights
    Start-Process powershell -ArgumentList $installCommand -Verb RunAs
    
    Write-Host "Installation process started with elevation. Check the new window." -ForegroundColor Green
    Exit
}
else {
    # Already running as admin, so just run the installer directly
    Write-Host "Already running as administrator. Running installer directly..." -ForegroundColor Green
    
    $binPath = Join-Path (Get-Location) "bitcoin-sprint.exe"
    $sha256 = (Get-FileHash -Path $binPath -Algorithm SHA256).Hash
    
    # Run the installer
    & .\install.ps1 -BinaryUrl "file:///$($binPath.Replace('\', '/'))" -BinarySha256 $sha256 -Force
}
