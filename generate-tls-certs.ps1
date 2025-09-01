# Generate TLS certificates for Rust Web Server
# This script creates self-signed certificates for development

param(
    [string]$CertDir = "config\tls",
    [string]$CertFile = "cert.pem",
    [string]$KeyFile = "key.pem",
    [string]$Subject = "/C=US/ST=State/L=City/O=Bitcoin Sprint/CN=localhost"
)

$ErrorActionPreference = "Stop"

# Create certificate directory if it doesn't exist
if (!(Test-Path $CertDir)) {
    New-Item -ItemType Directory -Path $CertDir -Force | Out-Null
    Write-Host "Created directory: $CertDir" -ForegroundColor Green
}

$CertPath = Join-Path $CertDir $CertFile
$KeyPath = Join-Path $CertDir $KeyFile

# Check if certificates already exist
if ((Test-Path $CertPath) -and (Test-Path $KeyPath)) {
    Write-Host "TLS certificates already exist:" -ForegroundColor Yellow
    Write-Host "  Certificate: $CertPath" -ForegroundColor White
    Write-Host "  Private Key: $KeyPath" -ForegroundColor White
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
prompt = no

[req_distinguished_name]
C = US
ST = State
L = City
O = Bitcoin Sprint
CN = localhost

[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = bitcoin-sprint-rust
IP.1 = 127.0.0.1
IP.2 = 0.0.0.0
"@

    $configPath = Join-Path $CertDir "openssl.cnf"
    $opensslConfig | Out-File -FilePath $configPath -Encoding ASCII

    # Generate certificate
    & openssl req -x509 -newkey rsa:4096 -keyout $KeyPath -out $CertPath -days 365 -nodes -config $configPath

    if ($LASTEXITCODE -eq 0) {
        Write-Host "TLS certificates generated successfully!" -ForegroundColor Green
        Write-Host "Certificate details:" -ForegroundColor Cyan
        & openssl x509 -in $CertPath -text -noout | Select-String -Pattern "Subject:|Issuer:|Not Before:|Not After:"
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
