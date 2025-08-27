param (
    [string]$User = "sprint",
    [string]$Password = "MyStrongPassw0rd!"
)

Write-Host "🔐 Bitcoin Core RPC Auth Generator (PowerShell)" -ForegroundColor Cyan
Write-Host "=============================================" -ForegroundColor Cyan

# Generate random salt
$Salt = -join ((65..90 + 97..122 + 48..57) | Get-Random -Count 16 | % {[char]$_})

# Compute HMAC-SHA256 hash of password+salt
$HMAC = New-Object System.Security.Cryptography.HMACSHA256
$HMAC.Key = [Text.Encoding]::UTF8.GetBytes($Salt)
$HashBytes = $HMAC.ComputeHash([Text.Encoding]::UTF8.GetBytes($Password))
$HashHex = -join ($HashBytes | ForEach-Object { $_.ToString("x2") })

# Output rpcauth line
$RpcAuthLine = "rpcauth=$User`:$Salt`$$HashHex"

Write-Host ""
Write-Host "Generated RPC Auth Configuration:" -ForegroundColor Green
Write-Host "=================================" -ForegroundColor Green
Write-Host $RpcAuthLine -ForegroundColor Yellow
Write-Host ""
Write-Host "📋 Copy this line to your bitcoin.conf file" -ForegroundColor White
Write-Host "📋 Use these credentials in your .env file:" -ForegroundColor White
Write-Host "   BTC_RPC_USER=$User" -ForegroundColor Gray
Write-Host "   BTC_RPC_PASS=$Password" -ForegroundColor Gray
Write-Host ""
Write-Host "✅ Setup complete! Your RPC credentials are now secure." -ForegroundColor Green
