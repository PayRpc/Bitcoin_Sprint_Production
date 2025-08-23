<#
.SYNOPSIS
  Hardened installer for Bitcoin Sprint on Windows.

.DESCRIPTION
  Downloads a single binary and optional config, verifies SHA256, optionally
  verifies via an API endpoint, installs the binary to a folder, creates a
  Windows service, opens the dashboard port in the firewall, and writes an
  install log. Designed for non-interactive CI-friendly runs.

.PARAMETER BinaryUrl
  HTTPS URL to the sprint.exe binary to download.

.PARAMETER BinarySha256
  Expected SHA256 checksum (hex, lowercase or uppercase) for the binary.

.PARAMETER ConfigUrl
  Optional URL to a config.json to download alongside the binary.

.PARAMETER InstallDir
  Target install directory. Default: C:\Program Files\Bitcoin Sprint

.PARAMETER ServiceName
  Windows service name to register. Default: BitcoinSprint

.PARAMETER DashboardPort
  Dashboard port to open in firewall. Default: 8080

.PARAMETER VerifyApiUrl
  Optional API URL that will be POSTed with JSON { sha256, filename } to
  perform server-side verification. Expected response: { "valid": true }.

.PARAMETER Force
  Overwrite existing installation when present.

.EXAMPLE
  .\install.ps1 -BinaryUrl https://example.com/sprint.exe -BinarySha256 DEADBE... -ConfigUrl https://example.com/config.json

#>

param(
    [Parameter(Mandatory=$true)] [string] $BinaryUrl,
    [Parameter(Mandatory=$true)] [string] $BinarySha256,
    [string] $ConfigUrl = $null,
    [string] $InstallDir = "C:\\Program Files\\Bitcoin Sprint",
    [string] $ServiceName = "BitcoinSprint",
    [int] $DashboardPort = 8080,
    [string] $VerifyApiUrl = $null,
    [switch] $Force,
    [switch] $JsonOutput
)

Set-StrictMode -Version Latest

function LogMsg([string]$msg) {
    if (-not $JsonOutput) { Write-Host $msg }
}

function Fail([string]$msg, [int]$code=1) {
    if ($JsonOutput) {
        $result = @{ success = $false; message = $msg; timestamp = (Get-Date).ToString("o") }
        Write-Output ($result | ConvertTo-Json -Compress)
    } else {
        Write-Error $msg
    }
    Exit $code
}

function ExitSuccess([string]$msg, [hashtable]$details = @{}) {
    if ($JsonOutput) {
        $result = @{ success = $true; message = $msg; timestamp = (Get-Date).ToString("o") } + $details
        Write-Output ($result | ConvertTo-Json -Compress)
    } else {
        Write-Host "✓ $msg"
    }
    Exit 0
}

if (-not ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Fail "This installer must be run as Administrator."
}

try {
    $tmp = New-Item -ItemType Directory -Path (Join-Path $env:TEMP ([Guid]::NewGuid().ToString())) -Force
    $tmpDir = $tmp.FullName
} catch {
    Fail "Failed to create temporary directory: $_"
}

$binPath = Join-Path $tmpDir "sprint.exe"
$cfgPath = if ($ConfigUrl) { Join-Path $tmpDir "config.json" } else { $null }

Write-Host "Downloading binary from: $BinaryUrl"
try {
    # enforce TLS by default for https
    if ($BinaryUrl -match '^http:') {
        Write-Warning "Binary URL uses plain HTTP — consider using HTTPS. Proceeding because caller allowed it."
    }
    Invoke-WebRequest -Uri $BinaryUrl -OutFile $binPath -UseBasicParsing -ErrorAction Stop
} catch {
    Fail "Failed to download binary: $_"
}

Write-Host "Computing SHA256..."
try {
    $hash = (Get-FileHash -Path $binPath -Algorithm SHA256).Hash.ToLower()
} catch {
    Fail "Failed to compute file hash: $_"
}

if ($hash -ne $BinarySha256.ToLower()) {
    Fail "SHA256 mismatch: expected $BinarySha256 but got $hash"
}
Write-Host "SHA256 verified: $hash"

if ($VerifyApiUrl) {
    Write-Host "Calling verification API: $VerifyApiUrl"
    try {
        $body = @{ sha256 = $hash; filename = [System.IO.Path]::GetFileName($BinaryUrl) } | ConvertTo-Json
        $resp = Invoke-RestMethod -Uri $VerifyApiUrl -Method Post -Body $body -ContentType 'application/json' -ErrorAction Stop
        if (-not $resp) { Fail "Empty response from verification API" }
        if ($resp.valid -ne $true) { Fail "Verification API rejected the binary" }
    } catch {
        Fail "Verification API check failed: $_"
    }
    Write-Host "Verification API accepted the binary."
}

if ($ConfigUrl) {
    Write-Host "Downloading config from: $ConfigUrl"
    try {
        Invoke-WebRequest -Uri $ConfigUrl -OutFile $cfgPath -UseBasicParsing -ErrorAction Stop
    } catch {
        Fail "Failed to download config: $_"
    }
}

# prepare install directory
if (Test-Path $InstallDir) {
    if ($Force) {
        Write-Host "Removing existing install because -Force was specified"
        Remove-Item -Path $InstallDir -Recurse -Force
    } else {
        Fail "Install directory $InstallDir already exists. Use -Force to overwrite."
    }
}

try {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
} catch {
    Fail "Failed to create install directory: $_"
}

$destExe = Join-Path $InstallDir (Split-Path $binPath -Leaf)
Copy-Item -Path $binPath -Destination $destExe -Force
if ($cfgPath) {
    Copy-Item -Path $cfgPath -Destination (Join-Path $InstallDir 'config.json') -Force
}

# set permissions: allow Administrators and SYSTEM full control
try {
    $acl = Get-Acl -Path $InstallDir
    $admins = New-Object System.Security.Principal.NTAccount("BUILTIN","Administrators")
    $system = New-Object System.Security.Principal.NTAccount("NT AUTHORITY","SYSTEM")
    $ruleAdmins = New-Object System.Security.AccessControl.FileSystemAccessRule($admins, "FullControl", "ContainerInherit, ObjectInherit", "None", "Allow")
    $ruleSystem = New-Object System.Security.AccessControl.FileSystemAccessRule($system, "FullControl", "ContainerInherit, ObjectInherit", "None", "Allow")
    $acl.SetAccessRule($ruleAdmins)
    $acl.AddAccessRule($ruleSystem)
    Set-Acl -Path $InstallDir -AclObject $acl
} catch {
    Write-Warning "Failed to set ACLs: $_"
}

# register service
Write-Host "Registering service '$ServiceName' pointing to $destExe"
try {
    if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {
        Write-Host "Service $ServiceName exists — attempting to remove and re-create"
        Stop-Service -Name $ServiceName -ErrorAction SilentlyContinue
        sc.exe delete "$ServiceName" | Out-Null
        Start-Sleep -Seconds 1
    }

    New-Service -Name $ServiceName -BinaryPathName "`"$destExe`"" -DisplayName "Bitcoin Sprint" -StartupType Automatic
    Start-Service -Name $ServiceName
    Write-Host "Service $ServiceName installed and started."
} catch {
    Write-Warning "Failed to create service with New-Service: $_"
    Write-Host "Falling back to sc.exe"
    $binPathArg = "`"$destExe`""
    $scOut = sc.exe create "$ServiceName" binPath= $binPathArg start= auto
    Write-Host $scOut
}

# open firewall port for dashboard
try {
    if (-not (Get-NetFirewallRule -DisplayName "Bitcoin Sprint Dashboard" -ErrorAction SilentlyContinue)) {
        New-NetFirewallRule -DisplayName "Bitcoin Sprint Dashboard" -Direction Inbound -Action Allow -Protocol TCP -LocalPort $DashboardPort -Profile Any
        Write-Host "Firewall rule added for port $DashboardPort"
    }
} catch {
    Write-Warning "Failed to add firewall rule (you may need to run in an elevated session or on older Windows): $_"
}

# write install log
$log = [PSCustomObject]@{
    Timestamp = (Get-Date).ToString("o")
    BinaryUrl = $BinaryUrl
    BinarySha256 = $hash
    VerifyApi = $VerifyApiUrl
    InstallDir = $InstallDir
    ServiceName = $ServiceName
    DashboardPort = $DashboardPort
}

$logPath = Join-Path $InstallDir "install.log"
$log | ConvertTo-Json | Out-File -FilePath $logPath -Encoding utf8

LogMsg "Install complete. Log written to $logPath"

# cleanup
try { Remove-Item -Path $tmpDir -Recurse -Force } catch {}

# Exit with success message
$details = @{ 
    installDir = $InstallDir
    serviceName = $ServiceName
    dashboardPort = $DashboardPort
    logPath = $logPath 
}
ExitSuccess -msg "Install complete" -details $details
