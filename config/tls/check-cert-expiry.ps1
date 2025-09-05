# Check TLS certificate expiry and alert if renewal is needed
param(
    [string]$CertPath = "config\tls\cert.pem",
    [int]$WarnDays = 30
)

$ErrorActionPreference = "Stop"

if (!(Test-Path $CertPath)) {
    Write-Host "Certificate not found: $CertPath" -ForegroundColor Red
    exit 1
}

try {
    $expiry = & openssl x509 -enddate -noout -in $CertPath
    $expiryDate = ($expiry -replace 'notAfter=', '').Trim()
    $expiryDateObj = [datetime]::Parse($expiryDate)
    $daysLeft = ($expiryDateObj - (Get-Date)).Days
    Write-Host "Certificate expires on: $expiryDateObj ($daysLeft days left)" -ForegroundColor Cyan
    if ($daysLeft -le $WarnDays) {
        Write-Host "WARNING: Certificate expires in $daysLeft days! Renew soon." -ForegroundColor Yellow
        exit 2
    }
} catch {
    Write-Host "Error checking certificate expiry: $_" -ForegroundColor Red
    exit 1
}
