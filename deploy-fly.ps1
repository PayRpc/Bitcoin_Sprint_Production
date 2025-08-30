# Bitcoin Sprint Fly.io Deployment Script
# This script handles the complete deployment process for the production demo

param(
    [string]$Action = "deploy",
    [int]$ScaleCount = 1
)

# Configuration
$APP_NAME = "bitcoin-sprint-fastapi"
$PRIMARY_REGION = "iad"
$DOCKERFILE_PATH = "fly/fastapi/Dockerfile"

# Colors for output
$GREEN = "Green"
$RED = "Red"
$YELLOW = "Yellow"
$BLUE = "Cyan"
$NC = "White"

# Logging functions
function Write-Log {
    param([string]$Message, [string]$Color = $GREEN)
    Write-Host "[$((Get-Date).ToString('yyyy-MM-dd HH:mm:ss'))] $Message" -ForegroundColor $Color
}

function Write-Error {
    param([string]$Message)
    Write-Host "[$((Get-Date).ToString('yyyy-MM-dd HH:mm:ss'))] ERROR: $Message" -ForegroundColor $RED
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[$((Get-Date).ToString('yyyy-MM-dd HH:mm:ss'))] WARNING: $Message" -ForegroundColor $YELLOW
}

function Write-Info {
    param([string]$Message)
    Write-Host "[$((Get-Date).ToString('yyyy-MM-dd HH:mm:ss'))] INFO: $Message" -ForegroundColor $BLUE
}

# Function to check if flyctl is installed
function Test-Flyctl {
    try {
        $null = Get-Command flyctl -ErrorAction Stop
        return $true
    }
    catch {
        return $false
    }
}

# Function to check if user is logged in
function Test-FlyLogin {
    try {
        $result = & flyctl auth whoami 2>$null
        return $LASTEXITCODE -eq 0
    }
    catch {
        return $false
    }
}

# Function to validate configuration
function Test-Configuration {
    Write-Log "Validating configuration..."

    # Check if Dockerfile exists
    if (!(Test-Path $DOCKERFILE_PATH)) {
        Write-Error "Dockerfile not found at: $DOCKERFILE_PATH"
        exit 1
    }

    # Check if fly.toml exists
    if (!(Test-Path "fly.toml")) {
        Write-Error "fly.toml not found in current directory"
        exit 1
    }

    # Check if Go binary exists
    $goBinaryExists = (Test-Path "bin/sprintd") -or (Test-Path "bin/sprintd.exe")
    if (!$goBinaryExists) {
        Write-Error "Go binary not found in bin/ directory"
        exit 1
    }

    # Check if FastAPI code exists
    if (!(Test-Path "Bitcoin_Sprint_fastapi/fastapi-gateway")) {
        Write-Error "FastAPI gateway code not found"
        exit 1
    }

    Write-Log "Configuration validation passed"
}

# Function to build and deploy
function Invoke-Deployment {
    Write-Log "Starting deployment process..."

    # Build and deploy
    Write-Info "Building and deploying to Fly.io..."
    try {
        & flyctl deploy --dockerfile $DOCKERFILE_PATH --remote-only --push --verbose
        if ($LASTEXITCODE -eq 0) {
            Write-Log "Deployment completed successfully!"
        } else {
            Write-Error "Deployment failed"
            exit 1
        }
    }
    catch {
        Write-Error "Deployment failed with exception: $($_.Exception.Message)"
        exit 1
    }
}

# Function to check deployment status
function Get-DeploymentStatus {
    Write-Log "Checking deployment status..."

    # Get app status
    Write-Info "App status:"
    try {
        & flyctl status
    }
    catch {
        Write-Warn "Could not retrieve app status"
    }

    # Get app URL
    try {
        $statusJson = & flyctl status --json 2>$null | ConvertFrom-Json
        if ($statusJson -and $statusJson.hostname) {
            Write-Log "Application URL: https://$($statusJson.hostname)"
        }
    }
    catch {
        Write-Warn "Could not retrieve application URL"
    }
}

# Function to setup database
function New-Database {
    $response = Read-Host "Do you want to attach a PostgreSQL database? (y/n)"
    if ($response -match "^[Yy]$") {
        Write-Log "Setting up PostgreSQL database..."

        try {
            & flyctl postgres create --name "$APP_NAME-db" --region $PRIMARY_REGION

            # Attach database to app
            & flyctl postgres attach "$APP_NAME-db" --app $APP_NAME

            Write-Log "Database setup completed"
        }
        catch {
            Write-Error "Database setup failed: $($_.Exception.Message)"
        }
    }
}

# Function to show post-deployment information
function Show-PostDeployInfo {
    Write-Log "Post-deployment information:"
    Write-Host ""
    Write-Host "1. Application URL:"
    try {
        $status = & flyctl status 2>$null
        $hostname = $status | Select-String "Hostname" | ForEach-Object { $_.Line -replace ".*:\s+", "" }
        if ($hostname) {
            Write-Host "   https://$hostname" -ForegroundColor $BLUE
        }
    }
    catch {
        Write-Host "   Check with: flyctl status" -ForegroundColor $YELLOW
    }

    Write-Host ""
    Write-Host "2. Check application health:"
    Write-Host "   curl https://YOUR_APP_URL/health" -ForegroundColor $BLUE
    Write-Host ""
    Write-Host "3. View application logs:"
    Write-Host "   flyctl logs" -ForegroundColor $BLUE
    Write-Host ""
    Write-Host "4. Scale the application:"
    Write-Host "   flyctl scale count 2" -ForegroundColor $BLUE
    Write-Host ""
    Write-Host "5. Monitor the application:"
    Write-Host "   flyctl monitor" -ForegroundColor $BLUE
    Write-Host ""
    Write-Host "6. Access the application:"
    Write-Host "   flyctl ssh console" -ForegroundColor $BLUE
}

# Main deployment function
function Invoke-MainDeployment {
    Write-Log "Bitcoin Sprint Fly.io Deployment Script"
    Write-Log "======================================"

    # Pre-deployment checks
    if (!(Test-Flyctl)) {
        Write-Error "flyctl is not installed. Please install it first:"
        Write-Host "  Download from: https://fly.io/docs/getting-started/installing-flyctl/" -ForegroundColor $YELLOW
        exit 1
    }

    if (!(Test-FlyLogin)) {
        Write-Error "You are not logged in to Fly.io. Please run:"
        Write-Host "  flyctl auth login" -ForegroundColor $YELLOW
        exit 1
    }

    Test-Configuration

    # Setup database if needed
    New-Database

    # Deploy application
    Invoke-Deployment

    # Check deployment
    Get-DeploymentStatus

    # Show post-deployment information
    Show-PostDeployInfo

    Write-Log "Deployment process completed successfully!"
    Write-Log "Your Bitcoin Sprint application is now running on Fly.io"
}

# Handle command line arguments
switch ($Action) {
    "status" {
        Get-DeploymentStatus
    }
    "logs" {
        & flyctl logs
    }
    "ssh" {
        & flyctl ssh console
    }
    "scale" {
        if ($ScaleCount -gt 0) {
            & flyctl scale count $ScaleCount
        } else {
            Write-Host "Usage: .\deploy-fly.ps1 -Action scale -ScaleCount <count>" -ForegroundColor $YELLOW
            exit 1
        }
    }
    default {
        Invoke-MainDeployment
    }
}
