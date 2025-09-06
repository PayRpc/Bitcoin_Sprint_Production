# Environment Protection Check
if (-not (.\scripts\powershell\protect-environment.ps1 -Check)) {
    Write-Host "🚨 Environment corruption detected! Restoring..." -ForegroundColor Red
    .\scripts\powershell\protect-environment.ps1 -Restore
}
