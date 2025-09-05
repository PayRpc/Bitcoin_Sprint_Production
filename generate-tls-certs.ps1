# Generate TLS certificates for Rust Web Server
# This script creates self-signed certificates for development

param(
    [string]$CertDir = "config\tls",
    [string]$CertFile = "cert.pem",
    [string]$KeyFile = "key.pem",
    [string]$Subject = "/C=US/ST=State/L=City/O=Bitcoin Sprint/CN=localhost",
    [string[]]$SANs = @("localhost", "bitcoin-sprint-rust"),
    [ValidateSet("ECC", "RSA")][string]$KeyType = "ECC",
    [switch]$Force
)


$ErrorActionPreference = "Stop"

# Check OpenSSL availability
if (-not (Get-Command openssl -ErrorAction SilentlyContinue)) {
    Write-Host "OpenSSL is not installed or not in PATH. Please install OpenSSL to proceed." -ForegroundColor Red
    exit 1
}

# Create certificate directory if it doesn't exist
if (!(Test-Path $CertDir)) {
    New-Item -ItemType Directory -Path $CertDir -Force | Out-Null
    Write-Host "Created directory: $CertDir" -ForegroundColor Green
}

$CertPath = Join-Path $CertDir $CertFile
$KeyPath = Join-Path $CertDir $KeyFile

# Check if certificates already exist
if ((Test-Path $CertPath) -and (Test-Path $KeyPath) -and -not $Force) {
    Write-Host "TLS certificates already exist:" -ForegroundColor Yellow
    Write-Host "  Certificate: $CertPath" -ForegroundColor White
    Write-Host "  Private Key: $KeyPath" -ForegroundColor White
    Write-Host "Use -Force to regenerate certificates." -ForegroundColor Cyan
    exit 0
}

Write-Host "Generating TLS certificates for Rust Web Server..." -ForegroundColor Cyan
Write-Host "Subject: $Subject" -ForegroundColor White
Write-Host "Certificate: $CertPath" -ForegroundColor White
Write-Host "Private Key: $KeyPath" -ForegroundColor White

try {
    # Generate private key and certificate using OpenSSL
    $opensslConfig = @"
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
    $sanEntries = @()
    $dnsIndex = 1
    $ipIndex = 1
    foreach ($san in $SANs) {
        if ($san -match "^\d+\.\d+\.\d+\.\d+$") {
            $sanEntries += "IP.$ipIndex = $san"
            $ipIndex++
        } else {
            $sanEntries += "DNS.$dnsIndex = $san"
            $dnsIndex++
        }
    }
    $sanBlock = $sanEntries -join "`n"
    $opensslConfig = @"
[req]

    # Generate certificate
    if ($KeyType -eq "ECC") {
        & openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:secp384r1 -keyout $KeyPath -out $CertPath -days 365 -nodes -config $configPath -subj $Subject
    } else {
        & openssl req -x509 -newkey rsa:4096 -keyout $KeyPath -out $CertPath -days 365 -nodes -config $configPath -subj $Subject
    }


    if ($LASTEXITCODE -eq 0) {
        Write-Host "TLS certificates generated successfully!" -ForegroundColor Green
        Write-Host "Certificate details:" -ForegroundColor Cyan
        & openssl x509 -in $CertPath -text -noout | Select-String -Pattern "Subject:|Issuer:|Not Before:|Not After:|Public-Key:|Curve:"
        Write-Host "Key Type: $KeyType" -ForegroundColor Magenta
        Write-Host "SANs: $($SANs -join ', ')" -ForegroundColor Magenta
        Write-Host "Security Note: ECC (secp384r1) is recommended for modern deployments. Use RSA only if required for legacy compatibility." -ForegroundColor Yellow
    } else {
        throw "OpenSSL command failed"
    }

    # Clean up config file
    Remove-Item $configPath -ErrorAction SilentlyContinue

} catch {
    Write-Host "Error generating TLS certificates: $_" -ForegroundColor Red
    Write-Host "Make sure OpenSSL is installed and available in PATH" -ForegroundColor Yellow
    exit 1
}

Write-Host "`nTLS certificates are ready for the Rust Web Server!" -ForegroundColor Green
