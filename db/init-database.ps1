# Initialize PostgreSQL database for Bitcoin Sprint
# Runs the production SQL script: init-db.sql

param(
    [string]$DbName = "bitcoin_sprint",
    [string]$DbUser = "postgres",
    [string]$DbPassword = $null,
    [string]$DbHost = "localhost",
    [string]$DbPort = "5432",
    [string]$SqlFile = "init-db.sql",
    [switch]$CreateDb = $false,
    [switch]$Force = $false
)

$ErrorActionPreference = "Stop"

# Ensure SQL file exists
if (-not (Test-Path $SqlFile)) {
    Write-Host "ERROR: SQL initialization file not found: $SqlFile" -ForegroundColor Red
    exit 1
}

# Check if psql is available
$psql = Get-Command psql -ErrorAction SilentlyContinue
if (-not $psql) {
    Write-Host "ERROR: PostgreSQL client (psql) not found in PATH" -ForegroundColor Red
    Write-Host "Please install PostgreSQL and ensure psql is in your PATH" -ForegroundColor Yellow
    exit 1
}

# Build connection string
$PsqlEnv = @{}
if ($DbPassword) {
    $PsqlEnv["PGPASSWORD"] = $DbPassword
}

$ConnStr = "-h $DbHost -p $DbPort -U $DbUser"

# Create database if requested
if ($CreateDb) {
    Write-Host "Creating database: $DbName" -ForegroundColor Cyan
    try {
        $output = psql $ConnStr -c "SELECT 1 FROM pg_database WHERE datname='$DbName'" | Select-String -Pattern "1 row"
        if ($output) {
            if ($Force) {
                Write-Host "Database exists, dropping and recreating due to -Force" -ForegroundColor Yellow
                psql $ConnStr -c "DROP DATABASE IF EXISTS $DbName" | Out-Null
                psql $ConnStr -c "CREATE DATABASE $DbName" | Out-Null
            } else {
                Write-Host "Database already exists. Use -Force to drop and recreate." -ForegroundColor Yellow
            }
        } else {
            psql $ConnStr -c "CREATE DATABASE $DbName" | Out-Null
            Write-Host "Database created successfully" -ForegroundColor Green
        }
    } catch {
        Write-Host "Failed to create database: $_" -ForegroundColor Red
        exit 1
    }
}

# Run initialization SQL
Write-Host "Initializing database schema from $SqlFile..." -ForegroundColor Cyan
try {
    $result = psql $ConnStr -d $DbName -f $SqlFile 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Error executing SQL script:" -ForegroundColor Red
        Write-Host $result -ForegroundColor Red
        exit 1
    }
    Write-Host "Database schema initialized successfully!" -ForegroundColor Green
    
    # Verify key tables exist
    $tableCheck = psql $ConnStr -d $DbName -c "\dt" | Select-String -Pattern "blocks|transactions"
    if ($tableCheck) {
        Write-Host "Tables verified:" -ForegroundColor Green
        $tableCheck | ForEach-Object { Write-Host "  - $_" -ForegroundColor White }
    } else {
        Write-Host "WARNING: Could not verify tables were created" -ForegroundColor Yellow
    }
} catch {
    Write-Host "Failed to initialize database: $_" -ForegroundColor Red
    exit 1
}

Write-Host "✅ Database setup complete!" -ForegroundColor Green
