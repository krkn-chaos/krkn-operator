#!/bin/bash
# test-terminal-api.sh
# Test script for Terminal API local development

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

API_URL="${API_URL:-http://localhost:8080}"
CLUSTER_ID="${CLUSTER_ID:-local-cluster}"
UUID="${UUID:-}"

echo "🧪 Terminal API Test Script"
echo "============================"
echo ""

# Function to print colored output
print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

# Check if gRPC server is running
check_grpc() {
    print_info "Checking if gRPC server is running on localhost:50051..."
    if lsof -i :50051 &> /dev/null; then
        print_success "gRPC server is running"
    else
        print_error "gRPC server is NOT running"
        echo "   Start it with: cd krkn-operator-data-provider && python server.py"
        exit 1
    fi
}

# Check if operator API is running
check_operator() {
    print_info "Checking if operator API is running on $API_URL..."
    if curl -s -f "$API_URL/api/v1/health" > /dev/null 2>&1; then
        print_success "Operator API is running"
    else
        print_error "Operator API is NOT running"
        echo "   Start it with: make run"
        exit 1
    fi
}

# Login and get JWT token
login() {
    print_info "Logging in as admin..."

    # Try to login
    RESPONSE=$(curl -s -X POST "$API_URL/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin"}' 2>&1)

    # Extract token
    TOKEN=$(echo "$RESPONSE" | grep -o '"token":"[^"]*' | cut -d'"' -f4)

    if [ -z "$TOKEN" ]; then
        print_error "Login failed"
        echo "Response: $RESPONSE"
        echo ""
        echo "Make sure admin user exists. If not, register first:"
        echo "  curl -X POST $API_URL/api/v1/auth/register \\"
        echo "    -H 'Content-Type: application/json' \\"
        echo "    -d '{\"username\":\"admin\",\"password\":\"admin\"}'"
        exit 1
    fi

    print_success "Login successful, got JWT token"
    echo "   Token: ${TOKEN:0:50}..."
}

# Get or prompt for UUID
get_uuid() {
    if [ -z "$UUID" ]; then
        print_info "No UUID provided. Listing available KrknTargetRequests..."
        echo ""
        echo "Run this command to get available UUIDs:"
        echo "  kubectl get krkntargetrequests -n krkn-operator -o jsonpath='{range .items[*]}{.metadata.name}{\"\\t\"}{.spec.uuid}{\"\\n\"}{end}'"
        echo ""
        read -p "Enter UUID: " UUID

        if [ -z "$UUID" ]; then
            print_error "UUID is required"
            exit 1
        fi
    fi

    print_info "Using UUID: $UUID"
}

# Test: kubectl get pods
test_get_pods() {
    print_info "Test 1: kubectl get pods -n default"

    RESPONSE=$(curl -s -X POST "$API_URL/api/v1/terminal" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d "{
            \"cluster_id\": \"$CLUSTER_ID\",
            \"uuid\": \"$UUID\",
            \"command\": \"kubectl get pods -n default\"
        }")

    EXIT_CODE=$(echo "$RESPONSE" | grep -o '"exit_code":[0-9]*' | cut -d':' -f2)
    ERROR=$(echo "$RESPONSE" | grep -o '"error":"[^"]*' | cut -d'"' -f4)

    if [ "$EXIT_CODE" = "0" ] && [ -z "$ERROR" ]; then
        print_success "Command executed successfully"

        # Decode and show output
        STDOUT_BASE64=$(echo "$RESPONSE" | grep -o '"stdout_base64":"[^"]*' | cut -d'"' -f4)
        if [ -n "$STDOUT_BASE64" ]; then
            echo ""
            echo "Output:"
            echo "-------"
            echo "$STDOUT_BASE64" | base64 -d
            echo "-------"
        fi
    else
        print_error "Command failed"
        echo "Exit Code: $EXIT_CODE"
        echo "Error: $ERROR"
        echo "Full Response: $RESPONSE"
    fi
}

# Test: kubectl get nodes
test_get_nodes() {
    print_info "Test 2: kubectl get nodes"

    RESPONSE=$(curl -s -X POST "$API_URL/api/v1/terminal" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d "{
            \"cluster_id\": \"$CLUSTER_ID\",
            \"uuid\": \"$UUID\",
            \"command\": \"kubectl get nodes\"
        }")

    EXIT_CODE=$(echo "$RESPONSE" | grep -o '"exit_code":[0-9]*' | cut -d':' -f2)

    if [ "$EXIT_CODE" = "0" ]; then
        print_success "Command executed successfully"
        STDOUT_BASE64=$(echo "$RESPONSE" | grep -o '"stdout_base64":"[^"]*' | cut -d'"' -f4)
        echo "$STDOUT_BASE64" | base64 -d | head -3
    else
        print_error "Command failed (exit code: $EXIT_CODE)"
    fi
}

# Test: blocked command (should fail)
test_blocked_command() {
    print_info "Test 3: kubectl get pods --watch (should be blocked)"

    RESPONSE=$(curl -s -X POST "$API_URL/api/v1/terminal" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d "{
            \"cluster_id\": \"$CLUSTER_ID\",
            \"uuid\": \"$UUID\",
            \"command\": \"kubectl get pods --watch\"
        }")

    ERROR=$(echo "$RESPONSE" | grep -o '"error":"[^"]*' | cut -d'"' -f4)

    if [ "$ERROR" = "not_permitted" ]; then
        print_success "Command correctly blocked"
        MESSAGE=$(echo "$RESPONSE" | grep -o '"message":"[^"]*' | cut -d'"' -f4)
        echo "   Message: $MESSAGE"
    else
        print_error "Command should have been blocked but wasn't!"
        echo "Response: $RESPONSE"
    fi
}

# Main
main() {
    check_grpc
    echo ""
    check_operator
    echo ""
    login
    echo ""
    get_uuid
    echo ""
    echo "Running tests..."
    echo "================"
    echo ""
    test_get_pods
    echo ""
    test_get_nodes
    echo ""
    test_blocked_command
    echo ""
    print_success "All tests completed!"
}

# Run main
main
