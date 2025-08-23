<#
.SYNOPSIS
  Uninstaller for Bitcoin Sprint on Windows.

.DESCRIPTION
  Stops and removes the Bitcoin Sprint service, removes firewall rules,
  and optionally removes the install directory. Fast and reliable cleanup.

.PARAMETER ServiceName
  Windows service name to remove. Default: BitcoinSprint

.PARAMETER InstallDir
  Install directory to remove. Default: C:\Program Files\Bitcoin Sprint

.PARAMETER KeepConfig
  Keep config files when removing (useful for upgrades).

.PARAMETER JsonOutput
  Output machine-readable JSON result for CI/automation.

.EXAMPLE
  .\uninstall.ps1 -JsonOutput

#>

param(
    [string] $ServiceName = "BitcoinSprint",
    [string] $InstallDir = "C:\\Program Files\\Bitcoin Sprint",
    [switch] $KeepConfig,
    [switch] $JsonOutput
)

Set-StrictMode -Version Latest

function LogMsg([string]$msg) {
    if (-not $JsonOutput) { Write-Host $msg }
}

function ExitWithResult([bool]$success, [string]$msg, [hashtable]$details = @{}) {
    if ($JsonOutput) {
        $result = @{
            success = $success
            message = $msg
            timestamp = (Get-Date).ToString("o")
        } + $details
        Write-Output ($result | ConvertTo-Json -Compress)
    } else {
        if ($success) { Write-Host "✓ $msg" } else { Write-Error $msg }
    }
    Exit $(if ($success) { 0 } else { 1 })
}

if (-not ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    ExitWithResult $false "Must run as Administrator"
}

$removed = @()

# stop and remove service
if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {
    LogMsg "Stopping service $ServiceName..."
    try {
        Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
        sc.exe delete "$ServiceName" | Out-Null
        $removed += "service"
        LogMsg "Service removed"
    } catch {
        ExitWithResult $false "Failed to remove service: $_"
    }
} else {
    LogMsg "Service $ServiceName not found"
}

# remove firewall rule
try {
    $rule = Get-NetFirewallRule -DisplayName "Bitcoin Sprint Dashboard" -ErrorAction SilentlyContinue
    if ($rule) {
        Remove-NetFirewallRule -DisplayName "Bitcoin Sprint Dashboard"
        $removed += "firewall"
        LogMsg "Firewall rule removed"
    }
} catch {
    LogMsg "Warning: could not remove firewall rule: $_"
}

# remove install directory
if (Test-Path $InstallDir) {
    if ($KeepConfig) {
        LogMsg "Keeping config files in $InstallDir"
        try {
            Remove-Item -Path (Join-Path $InstallDir "*.exe") -Force -ErrorAction SilentlyContinue
            Remove-Item -Path (Join-Path $InstallDir "*.log") -Force -ErrorAction SilentlyContinue
            $removed += "binaries"
        } catch {
            ExitWithResult $false "Failed to remove binaries: $_"
        }
    } else {
        LogMsg "Removing $InstallDir..."
        try {
            Remove-Item -Path $InstallDir -Recurse -Force
            $removed += "directory"
            LogMsg "Install directory removed"
        } catch {
            ExitWithResult $false "Failed to remove directory: $_"
        }
    }
} else {
    LogMsg "Install directory not found"
}

ExitWithResult $true "Uninstall complete" @{ removed = $removed }
