#!/bin/bash

# Bitcoin Sprint Fly.io Deployment Script
# This script handles the complete deployment process for the production demo

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
APP_NAME="bitcoin-sprint"
PRIMARY_REGION="iad"
DOCKERFILE_PATH="Dockerfile"

# Logging functions
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}" >&2
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

info() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $1${NC}"
}

# Function to check if flyctl is installed
check_flyctl() {
    if ! command -v flyctl &> /dev/null; then
        error "flyctl is not installed. Please install it first:"
        echo "  curl -L https://fly.io/install.sh | sh"
        exit 1
    fi
}

# Function to check if user is logged in
check_fly_login() {
    if ! flyctl auth whoami &> /dev/null; then
        error "You are not logged in to Fly.io. Please run:"
        echo "  flyctl auth login"
        exit 1
    fi
}

# Function to validate configuration
validate_config() {
    log "Validating configuration..."

    # Check if Dockerfile exists
    if [ ! -f "$DOCKERFILE_PATH" ]; then
        error "Dockerfile not found at: $DOCKERFILE_PATH"
        exit 1
    fi

    # Check if fly.toml exists
    if [ ! -f "fly.toml" ]; then
        error "fly.toml not found in current directory"
        exit 1
    fi

    # Check if Go modules exist
    if [ ! -f "go.mod" ]; then
        error "go.mod not found. This doesn't appear to be a Go project."
        exit 1
    fi

    # Check if Go source exists
    if [ ! -d "cmd/sprintd" ]; then
        error "Go source code not found in cmd/sprintd/"
        exit 1
    fi

    # Check if main.go exists
    if [ ! -f "cmd/sprintd/main.go" ]; then
        error "main.go not found in cmd/sprintd/"
        exit 1
    fi

    # Validate Go module
    if ! go mod verify &> /dev/null; then
        warn "Go modules may be corrupted. Consider running 'go mod tidy'"
    fi

    log "Configuration validation passed"
}

# Function to build and deploy
deploy_app() {
    log "Starting deployment process..."

    # Build and deploy
    info "Building and deploying Bitcoin Sprint Go application to Fly.io..."
    if flyctl deploy \
        --dockerfile "$DOCKERFILE_PATH" \
        --remote-only \
        --push \
        --verbose; then

        log "Deployment completed successfully!"
        return 0
    else
        error "Deployment failed"
        exit 1
    fi
}

# Function to check deployment status
check_deployment() {
    log "Checking deployment status..."

    # Get app status
    info "App status:"
    flyctl status

    # Get app URL
    APP_URL=$(flyctl status --json | grep -o '"hostname":"[^"]*"' | cut -d'"' -f4)
    if [ -n "$APP_URL" ]; then
        log "Application URL: https://$APP_URL"
    fi
}

# Function to setup secrets (if needed)
setup_secrets() {
    log "Setting up application secrets..."

    # Check if .env file exists
    if [ -f ".env" ]; then
        info "Found .env file. Setting up secrets..."
        # Note: In production, you should set secrets individually
        # flyctl secrets set KEY1=value1 KEY2=value2
        warn "Please set sensitive secrets manually using:"
        echo "  flyctl secrets set SECRET_KEY=your_secret_value"
    else
        warn "No .env file found. Make sure to set required secrets."
    fi
}

# Function to setup database (if using managed PostgreSQL)
setup_database() {
    read -p "Do you want to attach a PostgreSQL database? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log "Setting up PostgreSQL database..."

        # Create PostgreSQL database
        flyctl postgres create --name "${APP_NAME}-db" --region "$PRIMARY_REGION"

        # Attach database to app
        flyctl postgres attach "${APP_NAME}-db" --app "$APP_NAME"

        # Get database connection string
        DB_CONNECTION_STRING=$(flyctl postgres status --app "${APP_NAME}-db" | grep 'Connection string:' | sed 's/.*Connection string: //')

        if [ -n "$DB_CONNECTION_STRING" ]; then
            # Set database environment variables
            flyctl secrets set DATABASE_TYPE=postgres
            flyctl secrets set DATABASE_URL="$DB_CONNECTION_STRING"
            log "Database setup completed successfully"
        else
            error "Failed to get database connection string"
            exit 1
        fi
    fi
}

# Function to show post-deployment information
show_post_deploy_info() {
    log "Post-deployment information:"
    echo ""
    echo "1. Application URL:"
    flyctl status | grep "Hostname" | awk '{print "   https://" $2}'
    echo ""
    echo "2. Check application health:"
    echo "   curl https://YOUR_APP_URL/health"
    echo ""
    echo "3. Check API version:"
    echo "   curl https://YOUR_APP_URL/version"
    echo ""
    echo "4. View application logs:"
    echo "   flyctl logs"
    echo ""
    echo "4. Scale the application:"
    echo "   flyctl scale count 2"
    echo ""
    echo "5. Monitor the application:"
    echo "   flyctl monitor"
    echo ""
    echo "6. Access the application:"
    echo "   flyctl ssh console"
}

# Main deployment function
main() {
    log "Bitcoin Sprint Go Application Fly.io Deployment Script"
    log "===================================================="

    # Pre-deployment checks
    check_flyctl
    check_fly_login
    validate_config

    # Setup database if needed
    setup_database

    # Setup secrets
    setup_secrets

    # Deploy application
    deploy_app

    # Check deployment
    check_deployment

    # Show post-deployment information
    show_post_deploy_info

    log "Deployment process completed successfully!"
    log "Your Bitcoin Sprint application is now running on Fly.io"
}

# Handle command line arguments
case "${1:-}" in
    "status")
        check_deployment
        ;;
    "logs")
        flyctl logs
        ;;
    "ssh")
        flyctl ssh console
        ;;
    "scale")
        if [ -n "$2" ]; then
            flyctl scale count "$2"
        else
            echo "Usage: $0 scale <count>"
            exit 1
        fi
        ;;
    *)
        main
        ;;
esac
