#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Check if required ports are available for Bitcoin Sprint
.DESCRIPTION
    This script checks if the required ports (8080 for API, 9090 for metrics)
    are available before starting Bitcoin Sprint. It can also kill processes
    using these ports if requested.
.PARAMETER Kill
    Kill any processes using the required ports
.PARAMETER Quiet
    Suppress output except for errors
.EXAMPLE
    .\check-ports.ps1
.EXAMPLE
    .\check-ports.ps1 -Kill
#>

param(
    [switch]$Kill,
    [switch]$Quiet
)

$ports = @(8080, 9090)
$conflicts = @()

function Write-Info {
    param([string]$message)
    if (-not $Quiet) {
        Write-Host $message -ForegroundColor Green
    }
}

function Write-Warning {
    param([string]$message)
    Write-Host $message -ForegroundColor Yellow
}

function Write-Error {
    param([string]$message)
    Write-Host $message -ForegroundColor Red
}

function Test-Port {
    param([int]$port)

    try {
        $tcpClient = New-Object System.Net.Sockets.TcpClient
        $result = $tcpClient.ConnectAsync("127.0.0.1", $port).Wait(1000)
        $tcpClient.Close()

        if ($result) {
            return $true
        }
    }
    catch {
        # Connection failed, port is likely free
    }

    return $false
}

function Get-ProcessUsingPort {
    param([int]$port)

    try {
        $connections = Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue
        if ($connections) {
            foreach ($conn in $connections) {
                $process = Get-Process -Id $conn.OwningProcess -ErrorAction SilentlyContinue
                if ($process) {
                    return @{
                        ProcessId = $process.Id
                        ProcessName = $process.ProcessName
                        Port = $port
                    }
                }
            }
        }
    }
    catch {
        # Fallback method using netstat
        $netstat = netstat -ano | findstr ":$port"
        if ($netstat) {
            $parts = $netstat -split '\s+'
            if ($parts.Length -ge 5) {
                $pid = $parts[-1]
                try {
                    $process = Get-Process -Id $pid -ErrorAction SilentlyContinue
                    if ($process) {
                        return @{
                            ProcessId = $process.Id
                            ProcessName = $process.ProcessName
                            Port = $port
                        }
                    }
                }
                catch {
                    return @{
                        ProcessId = $pid
                        ProcessName = "Unknown"
                        Port = $port
                    }
                }
            }
        }
    }

    return $null
}

Write-Info "Checking port availability for Bitcoin Sprint..."

foreach ($port in $ports) {
    $inUse = Test-Port $port

    if ($inUse) {
        $process = Get-ProcessUsingPort $port
        if ($process) {
            Write-Warning "Port $port is in use by $($process.ProcessName) (PID: $($process.ProcessId))"
            $conflicts += @{
                Port = $port
                Process = $process
            }
        } else {
            Write-Warning "Port $port is in use by unknown process"
            $conflicts += @{
                Port = $port
                Process = $null
            }
        }
    } else {
        Write-Info "Port $port is available"
    }
}

if ($conflicts.Count -gt 0) {
    Write-Warning "`nFound $($conflicts.Count) port conflict(s)"

    if ($Kill) {
        Write-Info "Killing conflicting processes..."
        foreach ($conflict in $conflicts) {
            if ($conflict.Process) {
                try {
                    Stop-Process -Id $conflict.Process.ProcessId -Force
                    Write-Info "Killed $($conflict.Process.ProcessName) (PID: $($conflict.Process.ProcessId))"
                }
                catch {
                    Write-Error "Failed to kill $($conflict.Process.ProcessName) (PID: $($conflict.Process.ProcessId)): $_"
                }
            }
        }

        # Re-check ports after killing
        Start-Sleep -Seconds 2
        Write-Info "`nRe-checking ports..."
        $remainingConflicts = @()

        foreach ($conflict in $conflicts) {
            if (-not (Test-Port $conflict.Port)) {
                Write-Info "Port $($conflict.Port) is now available"
            } else {
                Write-Error "Port $($conflict.Port) is still in use"
                $remainingConflicts += $conflict
            }
        }

        if ($remainingConflicts.Count -eq 0) {
            Write-Info "All ports are now available!"
            exit 0
        } else {
            Write-Error "Some ports are still in use. Manual intervention required."
            exit 1
        }
    } else {
        Write-Error "Use -Kill parameter to automatically free the ports"
        exit 1
    }
} else {
    Write-Info "`nAll required ports are available!"
    exit 0
}
