<#
Launcher: start-core-and-sprint.ps1
- Detects bitcoin-core binary (bitcoind or bitcoin-qt)
- Optionally runs with -reindex-chainstate
- Waits for RPC (127.0.0.1:8332) to be available
- Optionally starts bitcoin-sprint with sensible env vars
#>
param(
	[switch]$Reindex,
	[int]$WaitTimeoutSeconds = 900,
	[switch]$StartSprint,
	[string]$SprintPath = ".\bitcoin-sprint.exe",
	[string]$ConfPath = "$env:LOCALAPPDATA\Bitcoin\bitcoin.conf"
)

function Write-Log { param($m, $c = 'White') Write-Host $m -ForegroundColor $c }

Write-Log "Launcher started. Conf: $ConfPath" Cyan

# Locate binaries
$bitcoindPaths = @(
	'C:\Program Files\Bitcoin\daemon\bitcoind.exe',
	'C:\Program Files\Bitcoin\bitcoind.exe'
)
$bitcoinQtPaths = @(
	'C:\Program Files\Bitcoin\bitcoin-qt.exe',
	'C:\Program Files (x86)\Bitcoin\bitcoin-qt.exe'
)

$bitcoind = $null
foreach ($p in $bitcoindPaths) { if (Test-Path $p) { $bitcoind = $p; break } }
$bitcoinQt = $null
foreach ($p in $bitcoinQtPaths) { if (Test-Path $p) { $bitcoinQt = $p; break } }

if (-not (Test-Path $ConfPath)) {
	Write-Log "Config file not found at $ConfPath" Yellow
}
else {
	Write-Log "Found config: $ConfPath" Green
}

if ($bitcoind) {
	Write-Log "Found bitcoind: $bitcoind" Green
}
elseif ($bitcoinQt) {
	Write-Log "Found bitcoin-qt (GUI only): $bitcoinQt" Yellow
}
else {
	Write-Log "No bitcoin-core binaries found. Please install bitcoind or use the GUI bitcoin-qt." Red
	exit 2
}

# Build argument list
$confArg = "-conf=\"$ConfPath\""

# If Reindex requested, append reindex arg
$reindexArg = $null
if ($Reindex) { $reindexArg = '-reindex-chainstate' }

# Start node
if ($bitcoind) {
	$args = @($confArg)
	if ($reindexArg) { $args += $reindexArg }
	Write-Log "Starting bitcoind: $bitcoind $($args -join ' ')" Cyan
	$proc = Start-Process -FilePath $bitcoind -ArgumentList $args -PassThru -NoNewWindow -WindowStyle Hidden
	Write-Log "bitcoind started (PID $($proc.Id))" Green
}
else {
	# Use bitcoin-qt GUI
	$args = @($confArg)
	if ($reindexArg) { $args += $reindexArg }
	Write-Log "Starting bitcoin-qt (GUI): $bitcoinQt $($args -join ' ')" Cyan
	$proc = Start-Process -FilePath $bitcoinQt -ArgumentList $args -PassThru
	Write-Log "bitcoin-qt launched (PID $($proc.Id))" Green
}

# Wait for RPC on 127.0.0.1:8332
$deadline = (Get-Date).AddSeconds($WaitTimeoutSeconds)
Write-Log "Waiting for RPC (127.0.0.1:8332) to become available (timeout ${WaitTimeoutSeconds}s)..." Cyan
while ((Get-Date) -lt $deadline) {
	try {
		$ok = Test-NetConnection -ComputerName 127.0.0.1 -Port 8332 -WarningAction SilentlyContinue
		if ($ok.TcpTestSucceeded) { Write-Log "RPC port 8332 is listening" Green; break }
	}
 catch {}
	Start-Sleep -Seconds 2
}

if (-not $ok -or -not $ok.TcpTestSucceeded) {
	Write-Log "Timeout waiting for RPC port 8332. Check Core logs (debug.log) and retry." Red
	exit 3
}

# Verify RPC responds to a simple call
try {
	$conf = Get-Content $ConfPath | ConvertFrom-StringData -ErrorAction SilentlyContinue
	$rpcuser = $null; $rpcpass = $null
	if ($conf) { $rpcuser = $conf['rpcuser']; $rpcpass = $conf['rpcpassword'] }
	if (-not $rpcuser -or -not $rpcpass) {
		Write-Log "rpcuser/rpcpassword not found in plain conf; trying cookie auth (may fail for GUI runs)" Yellow
	}
	if ($rpcuser -and $rpcpass) {
		$b64 = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes("$rpcuser`:$rpcpass"))
		$body = '{"jsonrpc":"1.0","id":"1","method":"getblockcount","params":[] }'
		$r = Invoke-RestMethod -Uri 'http://127.0.0.1:8332/' -Method Post -Body $body -ContentType 'application/json' -Headers @{ Authorization = "Basic $b64" } -TimeoutSec 5
		Write-Log "RPC responded: getblockcount=$($r.result)" Green
	}
 else {
		Write-Log "Skipping RPC auth test because rpcuser/rpcpassword not available; assume cookie auth or GUI login" Yellow
	}
}
catch {
	Write-Log "RPC call failed: $($_.Exception.Message)" Yellow
}

# Start Sprint if requested
if ($StartSprint) {
	if (-not (Test-Path $SprintPath)) { Write-Log "Sprint binary not found at $SprintPath" Red; exit 4 }
	Write-Log "Starting Bitcoin Sprint in foreground..." Cyan
	$env:RPC_NODES = "http://127.0.0.1:8332"
	if ($rpcuser) { $env:RPC_USER = $rpcuser }
	if ($rpcpass) { $env:RPC_PASS = $rpcpass }
	$env:API_PORT = "8080"
	Write-Log "Environment: RPC_NODES=$env:RPC_NODES RPC_USER=$env:RPC_USER API_PORT=$env:API_PORT" Gray
	& $SprintPath
}

Write-Log "Launcher finished." Cyan
