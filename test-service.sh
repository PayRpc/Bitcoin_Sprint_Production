#!/bin/bash

# Bitcoin Sprint - Storage Verification Service Test Script
# This script provides examples of how to test the service endpoints

set -e

SERVICE_URL="${SERVICE_URL:-http://localhost:8080}"
echo "🧪 Testing Bitcoin Sprint Storage Verification Service"
echo "======================================================"
echo "Service URL: $SERVICE_URL"
echo ""

# Function to test health endpoint
test_health() {
    echo "🏥 Testing Health Endpoint..."
    echo "GET $SERVICE_URL/health"

    response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" "$SERVICE_URL/health")

    http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)
    body=$(echo "$response" | sed '/HTTP_STATUS:/d')

    if [ "$http_status" = "200" ]; then
        echo "✅ Health check passed"
        echo "Response: $body"
    else
        echo "❌ Health check failed (HTTP $http_status)"
        echo "Response: $body"
    fi
    echo ""
}

# Function to test metrics endpoint
test_metrics() {
    echo "📊 Testing Metrics Endpoint..."
    echo "GET $SERVICE_URL/metrics"

    response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" "$SERVICE_URL/metrics")

    http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)
    body=$(echo "$response" | sed '/HTTP_STATUS:/d')

    if [ "$http_status" = "200" ]; then
        echo "✅ Metrics check passed"
        echo "Response: $body"
    else
        echo "❌ Metrics check failed (HTTP $http_status)"
        echo "Response: $body"
    fi
    echo ""
}

# Function to test verification endpoint
test_verification() {
    echo "🔍 Testing Verification Endpoint..."
    echo "POST $SERVICE_URL/verify"

    # Create test payload
    payload='{
        "file_id": "bitcoin_block_800000.dat",
        "provider": "test_provider",
        "protocol": "ipfs",
        "file_size": 1048576
    }'

    echo "Request payload:"
    echo "$payload" | jq . 2>/dev/null || echo "$payload"

    response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
        -X POST \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "$SERVICE_URL/verify")

    http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)
    body=$(echo "$response" | sed '/HTTP_STATUS:/d')

    if [ "$http_status" = "200" ]; then
        echo "✅ Verification request successful"
        echo "Response: $body" | jq . 2>/dev/null || echo "$body"
    else
        echo "❌ Verification request failed (HTTP $http_status)"
        echo "Response: $body"
    fi
    echo ""
}

# Function to test rate limiting
test_rate_limiting() {
    echo "🚦 Testing Rate Limiting..."
    echo "Making multiple rapid requests to test rate limiting"

    for i in {1..15}; do
        echo -n "Request $i: "

        response=$(curl -s -w "%{http_code}" \
            -X POST \
            -H "Content-Type: application/json" \
            -d '{
                "file_id": "test_file_'$i'.dat",
                "provider": "test_provider",
                "protocol": "ipfs",
                "file_size": 1024
            }' \
            "$SERVICE_URL/verify")

        if [ "$response" = "429" ]; then
            echo "✅ Rate limited (HTTP 429) - Working correctly"
            break
        elif [ "$response" = "200" ]; then
            echo "✅ OK (HTTP 200)"
        else
            echo "❌ Unexpected response (HTTP $response)"
        fi

        # Small delay between requests
        sleep 0.1
    done
    echo ""
}

# Function to test invalid requests
test_invalid_requests() {
    echo "❌ Testing Invalid Requests..."

    # Test empty file_id
    echo "Testing empty file_id:"
    response=$(curl -s -w "%{http_code}" \
        -X POST \
        -H "Content-Type: application/json" \
        -d '{
            "file_id": "",
            "provider": "test_provider",
            "protocol": "ipfs",
            "file_size": 1024
        }' \
        "$SERVICE_URL/verify")

    if [ "$response" = "400" ]; then
        echo "✅ Correctly rejected empty file_id (HTTP 400)"
    else
        echo "❌ Should have rejected empty file_id (HTTP $response)"
    fi

    # Test invalid protocol
    echo "Testing invalid protocol:"
    response=$(curl -s -w "%{http_code}" \
        -X POST \
        -H "Content-Type: application/json" \
        -d '{
            "file_id": "test_file.dat",
            "provider": "test_provider",
            "protocol": "invalid_protocol",
            "file_size": 1024
        }' \
        "$SERVICE_URL/verify")

    if [ "$response" = "400" ]; then
        echo "✅ Correctly rejected invalid protocol (HTTP 400)"
    else
        echo "❌ Should have rejected invalid protocol (HTTP $response)"
    fi

    # Test oversized file
    echo "Testing oversized file:"
    response=$(curl -s -w "%{http_code}" \
        -X POST \
        -H "Content-Type: application/json" \
        -d '{
            "file_id": "test_file.dat",
            "provider": "test_provider",
            "protocol": "ipfs",
            "file_size": 2147483648
        }' \
        "$SERVICE_URL/verify")

    if [ "$response" = "400" ]; then
        echo "✅ Correctly rejected oversized file (HTTP 400)"
    else
        echo "❌ Should have rejected oversized file (HTTP $response)"
    fi

    echo ""
}

# Function to test connection health
test_connection_health() {
    echo "🔗 Testing Connection Health Monitoring..."

    # Get health status
    response=$(curl -s "$SERVICE_URL/health")

    # Check if connection_health is present in response
    if echo "$response" | grep -q "connection_health"; then
        echo "✅ Connection health monitoring is active"
        echo "Health data:"
        echo "$response" | jq '.connection_health' 2>/dev/null || echo "$response"
    else
        echo "⚠️  Connection health monitoring may not be enabled"
        echo "Response: $response"
    fi
    echo ""
}

# Function to show help
show_help() {
    echo "📖 Usage: $0 [test_type]"
    echo ""
    echo "Test Types:"
    echo "  all       Run all tests (default)"
    echo "  health    Test health endpoint only"
    echo "  metrics   Test metrics endpoint only"
    echo "  verify    Test verification endpoint only"
    echo "  rate      Test rate limiting"
    echo "  invalid   Test invalid request handling"
    echo "  conn      Test connection health monitoring"
    echo "  help      Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  SERVICE_URL  Service URL (default: http://localhost:8080)"
    echo ""
    echo "Examples:"
    echo "  $0 all                    # Run all tests"
    echo "  $0 health                 # Test health endpoint"
    echo "  SERVICE_URL=http://prod:8080 $0 verify  # Test production service"
    echo ""
}

# Check if service is running
check_service() {
    echo "🔍 Checking if service is running..."

    if curl -s --max-time 5 "$SERVICE_URL/health" > /dev/null 2>&1; then
        echo "✅ Service is running and responding"
        return 0
    else
        echo "❌ Service is not responding at $SERVICE_URL"
        echo "   Make sure the service is running first:"
        echo "   ./build-service.sh run"
        return 1
    fi
}

# Main script logic
if ! check_service; then
    exit 1
fi

case "${1:-all}" in
    "all")
        test_health
        test_metrics
        test_verification
        test_rate_limiting
        test_invalid_requests
        test_connection_health
        ;;
    "health")
        test_health
        ;;
    "metrics")
        test_metrics
        ;;
    "verify")
        test_verification
        ;;
    "rate")
        test_rate_limiting
        ;;
    "invalid")
        test_invalid_requests
        ;;
    "conn")
        test_connection_health
        ;;
    "help"|*)
        show_help
        ;;
esac

echo "🎉 Testing completed!"
echo ""
echo "📊 Test Summary:"
echo "   - Health endpoint: $SERVICE_URL/health"
echo "   - Metrics endpoint: $SERVICE_URL/metrics"
echo "   - Verification endpoint: $SERVICE_URL/verify"
echo "   - Rate limiting: Configured for 10 requests per minute"
echo "   - Connection management: HTTP client with pooling and circuit breaker"
echo ""
