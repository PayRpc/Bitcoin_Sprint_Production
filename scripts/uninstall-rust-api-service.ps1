$ErrorActionPreference = 'Stop'

param(
  [string]$ServiceName = 'BitcoinSprintRust',
  [string]$NssmPath = 'nssm.exe'
)

try { & $NssmPath stop $ServiceName | Out-Null } catch {}
try { & $NssmPath remove $ServiceName confirm | Out-Null } catch {}
Write-Host "Service '$ServiceName' removed." -ForegroundColor Green
