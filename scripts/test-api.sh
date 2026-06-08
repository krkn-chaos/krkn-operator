#!/usr/bin/env bash
#
# test-api.sh - Test krkn-operator API endpoints
#
# This script tests the krkn-operator REST API by:
# 1. Checking registration status
# 2. Registering first admin user (if needed)
# 3. Logging in and obtaining JWT token
# 4. Making authenticated API calls
# 5. Testing cluster discovery (placeholder)
#
# Prerequisites:
# - krkn-operator running and accessible
# - Port-forward or direct access to API (default: http://localhost:8080)
#
# Usage:
#   ./scripts/test-api.sh
#
# Environment Variables:
#   API_URL         - API endpoint (default: http://localhost:8080)
#   ADMIN_EMAIL     - Admin email (default: [email protected])
#   ADMIN_PASSWORD  - Admin password (default: AdminPassword123!)
#   ADMIN_NAME      - Admin first name (default: Admin)
#   ADMIN_SURNAME   - Admin last name (default: User)
#   ADMIN_ORG       - Admin organization (default: Krkn Chaos)
#
# Example:
#   API_URL=http://krkn-operator.example.com \
#   ADMIN_EMAIL=[email protected] \
#   ./scripts/test-api.sh
#
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
API_URL="${API_URL:-http://localhost:8080}"
ADMIN_EMAIL="${ADMIN_EMAIL:-[email protected]}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-AdminPassword123!}"
ADMIN_NAME="${ADMIN_NAME:-Admin}"
ADMIN_SURNAME="${ADMIN_SURNAME:-User}"
ADMIN_ORG="${ADMIN_ORG:-Krkn Chaos}"

TOKEN_FILE="/tmp/krkn-jwt-token"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}krkn-operator API Test Script${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "API URL: $API_URL"
echo "Admin email: $ADMIN_EMAIL"
echo ""

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

api_call() {
    local method="$1"
    local endpoint="$2"
    local data="$3"
    local auth_token="$4"

    local curl_args=(-s -w "\n%{http_code}")

    if [ -n "$auth_token" ]; then
        curl_args+=(-H "Authorization: Bearer $auth_token")
    fi

    if [ -n "$data" ]; then
        curl_args+=(-H "Content-Type: application/json" -d "$data")
    fi

    curl "${curl_args[@]}" -X "$method" "${API_URL}${endpoint}"
}

extract_http_code() {
    echo "$1" | tail -n1
}

extract_body() {
    echo "$1" | sed '$d'
}

# Test 1: Check API health (unauthenticated endpoint)
log_step "Test 1: Checking API health..."
RESPONSE=$(api_call GET "/api/v1/health")
HTTP_CODE=$(extract_http_code "$RESPONSE")

if [ "$HTTP_CODE" = "401" ] || [ "$HTTP_CODE" = "200" ]; then
    log_success "API is responding (HTTP $HTTP_CODE)"
else
    log_error "Unexpected response from health endpoint: HTTP $HTTP_CODE"
    exit 1
fi
echo ""

# Test 2: Check registration status
log_step "Test 2: Checking registration status..."
RESPONSE=$(api_call GET "/api/v1/auth/is-registered")
HTTP_CODE=$(extract_http_code "$RESPONSE")
BODY=$(extract_body "$RESPONSE")

if [ "$HTTP_CODE" != "200" ]; then
    log_error "Failed to check registration status: HTTP $HTTP_CODE"
    echo "$BODY"
    exit 1
fi

REGISTERED=$(echo "$BODY" | jq -r '.registered')
log_info "Registration status: $BODY"

if [ "$REGISTERED" = "true" ]; then
    log_warn "Admin already registered, skipping registration"
    SKIP_REGISTRATION=true
else
    log_info "No admin registered yet"
    SKIP_REGISTRATION=false
fi
echo ""

# Test 3: Register admin user
if [ "$SKIP_REGISTRATION" = "false" ]; then
    log_step "Test 3: Registering admin user..."

    REGISTER_DATA=$(cat <<EOF
{
  "userId": "$ADMIN_EMAIL",
  "password": "$ADMIN_PASSWORD",
  "name": "$ADMIN_NAME",
  "surname": "$ADMIN_SURNAME",
  "organization": "$ADMIN_ORG",
  "role": "admin"
}
EOF
)

    RESPONSE=$(api_call POST "/api/v1/auth/register" "$REGISTER_DATA")
    HTTP_CODE=$(extract_http_code "$RESPONSE")
    BODY=$(extract_body "$RESPONSE")

    if [ "$HTTP_CODE" != "201" ]; then
        log_error "Failed to register admin user: HTTP $HTTP_CODE"
        echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"
        exit 1
    fi

    log_success "Admin user registered successfully"
    echo "$BODY" | jq '.'
else
    log_info "Skipping registration (admin already exists)"
fi
echo ""

# Test 4: Login and get JWT token
log_step "Test 4: Logging in as admin..."

LOGIN_DATA=$(cat <<EOF
{
  "userId": "$ADMIN_EMAIL",
  "password": "$ADMIN_PASSWORD"
}
EOF
)

RESPONSE=$(api_call POST "/api/v1/auth/login" "$LOGIN_DATA")
HTTP_CODE=$(extract_http_code "$RESPONSE")
BODY=$(extract_body "$RESPONSE")

if [ "$HTTP_CODE" != "200" ]; then
    log_error "Failed to login: HTTP $HTTP_CODE"
    echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"
    exit 1
fi

TOKEN=$(echo "$BODY" | jq -r '.token')
USER_ID=$(echo "$BODY" | jq -r '.userId')
ROLE=$(echo "$BODY" | jq -r '.role')
EXPIRES_AT=$(echo "$BODY" | jq -r '.expiresAt')

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
    log_error "Failed to extract JWT token"
    echo "$BODY"
    exit 1
fi

log_success "Login successful"
echo "  User: $USER_ID"
echo "  Role: $ROLE"
echo "  Token: ${TOKEN:0:30}..."
echo "  Expires: $EXPIRES_AT"

# Save token for later use
echo "$TOKEN" > "$TOKEN_FILE"
log_info "Token saved to $TOKEN_FILE"
echo ""

# Test 5: Authenticated health check
log_step "Test 5: Testing authenticated health check..."
RESPONSE=$(api_call GET "/api/v1/health" "" "$TOKEN")
HTTP_CODE=$(extract_http_code "$RESPONSE")
BODY=$(extract_body "$RESPONSE")

if [ "$HTTP_CODE" != "200" ]; then
    log_error "Authenticated health check failed: HTTP $HTTP_CODE"
    echo "$BODY"
    exit 1
fi

HEALTH_STATUS=$(echo "$BODY" | jq -r '.status')
if [ "$HEALTH_STATUS" = "healthy" ]; then
    log_success "Health check passed: $HEALTH_STATUS"
else
    log_warn "Unexpected health status: $HEALTH_STATUS"
fi
echo ""

# Test 6: Create target request to discover clusters
log_step "Test 6: Creating target request to discover OCM clusters..."

# First, we need to create a target request
# The operator should auto-discover OCM ManagedClusters
REQUEST_ID="test-discovery-$(date +%s)"

# Note: The actual endpoint for creating target requests might differ
# This is a placeholder - adjust based on actual API
log_info "Request ID: $REQUEST_ID"

# Give the operator some time to discover OCM clusters if it hasn't already
log_info "Waiting 5 seconds for operator to initialize providers..."
sleep 5

# For now, we'll just verify we can make authenticated requests
# The actual cluster discovery API needs to be implemented
log_warn "Cluster discovery API endpoint needs to be verified"
log_info "Expected: GET /api/v1/clusters should return hub, cluster1, cluster2"
echo ""

# Test 7: List available providers (if endpoint exists)
log_step "Test 7: Testing provider endpoints..."
RESPONSE=$(api_call GET "/api/v1/providers" "" "$TOKEN")
HTTP_CODE=$(extract_http_code "$RESPONSE")
BODY=$(extract_body "$RESPONSE")

if [ "$HTTP_CODE" = "200" ]; then
    log_success "Providers endpoint accessible"
    echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"
elif [ "$HTTP_CODE" = "404" ]; then
    log_warn "Providers endpoint not found (might not be implemented yet)"
else
    log_info "Providers endpoint returned HTTP $HTTP_CODE"
fi
echo ""

# Final summary
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✓ API Tests Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Summary:"
echo "  - API is accessible and responding"
echo "  - Admin user registered: $USER_ID"
echo "  - Authentication working"
echo "  - JWT token obtained and saved"
echo ""
echo "Saved token: $TOKEN_FILE"
echo "Token content (first 50 chars): ${TOKEN:0:50}..."
echo ""
echo "Next steps:"
echo "1. Use the token for further API calls:"
echo "   TOKEN=\$(cat $TOKEN_FILE)"
echo '   curl -H "Authorization: Bearer $TOKEN" '$API_URL'/api/v1/health | jq'
echo ""
echo "2. Implement and test cluster discovery endpoints"
echo "3. Test scenario execution"
echo ""
