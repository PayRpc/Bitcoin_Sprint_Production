# Verbose Bitcoin Sprint Service Installer with detailed diagnostics
# Run with administrative privileges

# Enable verbose output
$VerbosePreference = "Continue"
$DebugPreference = "Continue"

# Settings
$ServiceName = "BitcoinSprint"
$InstallDir = "C:\Program Files\Bitcoin Sprint"

Write-Host "===== Bitcoin Sprint Service Installer ====="
Write-Host "Time: $(Get-Date)" 
Write-Host "User: $([System.Security.Principal.WindowsIdentity]::GetCurrent().Name)"
Write-Host "Working Directory: $(Get-Location)"

# Check for admin privileges
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")
if (-not $isAdmin) {
    Write-Error "ERROR: This installer must be run as Administrator."
    Write-Host "Press any key to exit..."
    $null = $Host.UI.RawUI.ReadKey('NoEcho,IncludeKeyDown')
    Exit 1
}
Write-Host "Administrator privileges confirmed."

# Find the executable
$binPath = Join-Path (Get-Location) "bitcoin-sprint.exe"
Write-Host "Looking for executable at: $binPath"
if (Test-Path $binPath) {
    $fileInfo = Get-Item $binPath
    Write-Host "Found executable: $($fileInfo.FullName)"
    Write-Host "Size: $($fileInfo.Length) bytes"
    Write-Host "Last Modified: $($fileInfo.LastWriteTime)"
} else {
    Write-Error "ERROR: Executable not found at $binPath"
    Write-Host "Press any key to exit..."
    $null = $Host.UI.RawUI.ReadKey('NoEcho,IncludeKeyDown')
    Exit 1
}

# Create installation directory
Write-Host "`n[1/4] Creating installation directory: $InstallDir"
try {
    if (Test-Path $InstallDir) {
        Write-Host "Directory already exists."
    } else {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        Write-Host "Directory created successfully."
    }
    
    # Check if directory was created/exists
    if (Test-Path $InstallDir) {
        $dirInfo = Get-Item $InstallDir
        Write-Host "Installation directory confirmed: $($dirInfo.FullName)"
    } else {
        throw "Directory does not exist after creation attempt"
    }
} catch {
    Write-Error "ERROR creating directory: $_"
    Write-Host "Detailed error info:"
    $_ | Format-List * -Force
    Write-Host "Press any key to exit..."
    $null = $Host.UI.RawUI.ReadKey('NoEcho,IncludeKeyDown')
    Exit 1
}

# Copy executable
Write-Host "`n[2/4] Copying executable to installation directory..."
$destExe = Join-Path $InstallDir "bitcoin-sprint.exe"
try {
    Copy-Item -Path $binPath -Destination $destExe -Force
    if (Test-Path $destExe) {
        $fileInfo = Get-Item $destExe
        Write-Host "File copied successfully to: $($fileInfo.FullName)"
        Write-Host "Size: $($fileInfo.Length) bytes"
        Write-Host "Last Modified: $($fileInfo.LastWriteTime)"
    } else {
        throw "File not found after copy operation"
    }
} catch {
    Write-Error "ERROR copying executable: $_"
    Write-Host "Detailed error info:"
    $_ | Format-List * -Force
    Write-Host "Press any key to exit..."
    $null = $Host.UI.RawUI.ReadKey('NoEcho,IncludeKeyDown')
    Exit 1
}

# Check for existing service
Write-Host "`n[3/4] Checking for existing service..."
try {
    $existingService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($existingService) {
        Write-Host "Service '$ServiceName' exists. Status: $($existingService.Status)"
        Write-Host "Stopping service..."
        Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
        Start-Sleep -Seconds 3
        
        Write-Host "Removing service..."
        sc.exe delete $ServiceName
        Start-Sleep -Seconds 3
        
        $checkService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
        if ($checkService) {
            Write-Host "Warning: Service still exists after deletion attempt."
        } else {
            Write-Host "Service successfully removed."
        }
    } else {
        Write-Host "No existing service found with name '$ServiceName'."
    }
} catch {
    Write-Host "Warning during service check/removal: $_"
}

# Create and start service
Write-Host "`n[4/4] Creating and starting service..."
try {
    Write-Host "Creating service with sc.exe..."
    $createOutput = sc.exe create $ServiceName binPath= "`"$destExe`"" DisplayName= "Bitcoin Sprint" start= auto
    Write-Host "SC.EXE OUTPUT: $createOutput"
    
    # Verify service was created
    $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($service) {
        Write-Host "Service created successfully."
        Write-Host "Service name: $($service.Name)"
        Write-Host "Display name: $($service.DisplayName)"
        Write-Host "Status: $($service.Status)"
        
        Write-Host "Starting service..."
        $startOutput = sc.exe start $ServiceName
        Write-Host "SC.EXE START OUTPUT: $startOutput"
        
        Start-Sleep -Seconds 3
        
        # Check service status
        $runningService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
        if ($runningService) {
            Write-Host "Final service status: $($runningService.Status)"
            if ($runningService.Status -eq "Running") {
                Write-Host "Service started successfully."
            } else {
                Write-Host "Service is not running. Current status: $($runningService.Status)"
                Write-Host "Check Windows Event Viewer for startup errors."
            }
        } else {
            Write-Host "Warning: Service not found after creation."
        }
    } else {
        throw "Service was not created successfully."
    }
} catch {
    Write-Error "ERROR creating/starting service: $_"
    Write-Host "Detailed error info:"
    $_ | Format-List * -Force
    Write-Host "Press any key to exit..."
    $null = $Host.UI.RawUI.ReadKey('NoEcho,IncludeKeyDown')
    Exit 1
}

Write-Host "`n===== Installation Summary ====="
Write-Host "Service Name: $ServiceName"
Write-Host "Installation Directory: $InstallDir"
Write-Host "Executable Path: $destExe"
Write-Host "Completed at: $(Get-Date)"
Write-Host "Installation completed successfully."
Write-Host "Press any key to exit..."
$null = $Host.UI.RawUI.ReadKey('NoEcho,IncludeKeyDown')
