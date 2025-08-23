# Simple version of the Bitcoin Sprint Installer
# Only installs the service from a local binary

param(
    [string] $InstallDir = "C:\Program Files\Bitcoin Sprint",
    [string] $ServiceName = "BitcoinSprint"
)

# Check if running as administrator
if (-not ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Error "This installer must be run as Administrator."
    Exit 1
}

Write-Host "Installing Bitcoin Sprint..."

# Get the executable path
$binPath = Join-Path (Get-Location) "bitcoin-sprint.exe"
if (-not (Test-Path $binPath)) {
    Write-Error "Executable not found at $binPath"
    Exit 1
}

# Create installation directory
if (-not (Test-Path $InstallDir)) {
    try {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        Write-Host "Created directory: $InstallDir"
    } catch {
        Write-Error "Failed to create install directory: $_"
        Exit 1
    }
}

# Copy executable
$destExe = Join-Path $InstallDir "bitcoin-sprint.exe"
try {
    Copy-Item -Path $binPath -Destination $destExe -Force
    Write-Host "Copied executable to $destExe"
} catch {
    Write-Error "Failed to copy executable: $_"
    Exit 1
}

# Register service
Write-Host "Installing service '$ServiceName'..."
try {
    # Remove existing service if it exists
    if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {
        Write-Host "Service $ServiceName exists - removing it first"
        Stop-Service -Name $ServiceName -ErrorAction SilentlyContinue
        Start-Sleep -Seconds 2
        sc.exe delete $ServiceName | Out-Null
        Start-Sleep -Seconds 2
    }

    # Create the service
    Write-Host "Creating service..."
    New-Service -Name $ServiceName -BinaryPathName "`"$destExe`"" -DisplayName "Bitcoin Sprint" -StartupType Automatic | Out-Null
    
    # Start the service
    Write-Host "Starting service..."
    Start-Service -Name $ServiceName
    
    Write-Host "Service $ServiceName installed and started successfully."
} catch {
    Write-Error "Failed to install service: $_"
    # Try fallback method with sc.exe
    Write-Host "Trying alternate method with sc.exe..."
    $binPathArg = "`"$destExe`""
    sc.exe create $ServiceName binPath= $binPathArg DisplayName= "Bitcoin Sprint" start= auto
    sc.exe start $ServiceName
}

Write-Host "Installation completed successfully."
