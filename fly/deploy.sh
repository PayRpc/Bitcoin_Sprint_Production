# Fly.io Deployment Script for Bitcoin Sprint
# This script deploys FastAPI, Grafana, and connects to managed PostgreSQL

#!/bin/bash

set -e

echo "🛫 Starting Bitcoin Sprint Fly.io Deployment..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if flyctl is installed
if ! command -v flyctl &> /dev/null; then
    print_error "flyctl is not installed. Please install it first:"
    echo "curl -L https://fly.io/install.sh | sh"
    exit 1
fi

# Check if user is logged in
if ! flyctl auth whoami &> /dev/null; then
    print_error "Please login to Fly.io first:"
    echo "flyctl auth login"
    exit 1
fi

print_status "Fly.io authentication confirmed"

# Deploy FastAPI service
print_status "Deploying FastAPI service..."
cd fly/fastapi
if [ -f "fly.toml" ]; then
    flyctl deploy --config fly.toml
    print_success "FastAPI service deployed"
else
    print_error "FastAPI fly.toml not found"
    exit 1
fi
cd ../..

# Deploy Grafana service
print_status "Deploying Grafana service..."
cd fly/grafana
if [ -f "fly.toml" ]; then
    flyctl deploy --config fly.toml
    print_success "Grafana service deployed"
else
    print_error "Grafana fly.toml not found"
    exit 1
fi
cd ../..

# Get service URLs
print_status "Getting service URLs..."
FASTAPI_URL=$(flyctl apps list | grep bitcoin-sprint-fastapi | awk '{print $2}')
GRAFANA_URL=$(flyctl apps list | grep bitcoin-sprint-grafana | awk '{print $2}')

print_success "Deployment completed!"
echo ""
echo "📊 Service URLs:"
echo "  FastAPI: https://$FASTAPI_URL"
echo "  Grafana: https://$GRAFANA_URL"
echo ""
print_warning "Remember to:"
echo "  1. Set up your PostgreSQL database connection strings"
echo "  2. Configure Grafana data sources to point to FastAPI"
echo "  3. Set up proper CORS policies"
echo "  4. Configure monitoring and alerting"

print_success "Bitcoin Sprint is now live on Fly.io! 🚀"
