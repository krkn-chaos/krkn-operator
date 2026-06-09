#!/usr/bin/env bash
#
# test-api.sh - Test krkn-operator API endpoints
#
# This script tests the krkn-operator REST API by:
# 1. Setting up port-forward to operator service (optional)
# 2. Checking registration status
# 3. Registering first admin user (if needed)
# 4. Logging in and obtaining JWT token
# 5. Making authenticated API calls
# 6. Testing cluster discovery (placeholder)
#
# Prerequisites:
# - krkn-operator running in Kubernetes cluster
# - kubectl configured with access to cluster
#
# Usage:
#   ./scripts/test-api.sh
#
# Environment Variables:
#   SETUP_PORT_FORWARD - Setup port-forward automatically (default: true)
#   KUBE_CONTEXT       - kubectl context (default: kind-hub)
#   OPERATOR_NAMESPACE - Operator namespace (default: krkn-operator)
#   LOCAL_PORT         - Local port for port-forward (default: 8080)
#   SERVICE_PORT       - Service port (default: 8080)
#   API_URL            - API endpoint (default: http://localhost:${LOCAL_PORT})
#   ADMIN_EMAIL        - Admin email (default: [email protected])
#   ADMIN_PASSWORD     - Admin password (default: AdminPassword123!)
#   ADMIN_NAME         - Admin first name (default: Admin)
#   ADMIN_SURNAME      - Admin last name (default: User)
#   ADMIN_ORG          - Admin organization (default: Krkn Chaos)
#
# Example:
#   # Use existing port-forward or remote API
#   SETUP_PORT_FORWARD=false API_URL=http://krkn-operator.example.com ./scripts/test-api.sh
#
#   # Custom namespace and context
#   KUBE_CONTEXT=my-cluster OPERATOR_NAMESPACE=custom-ns ./scripts/test-api.sh
#
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SETUP_PORT_FORWARD="${SETUP_PORT_FORWARD:-true}"
KUBE_CONTEXT="${KUBE_CONTEXT:-kind-hub}"
OPERATOR_NAMESPACE="${OPERATOR_NAMESPACE:-krkn-operator}"
LOCAL_PORT="${LOCAL_PORT:-8080}"
SERVICE_PORT="${SERVICE_PORT:-8080}"
API_URL="${API_URL:-http://localhost:${LOCAL_PORT}}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@krkn-chaos.local}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-AdminPassword123!}"
ADMIN_NAME="${ADMIN_NAME:-Admin}"
ADMIN_SURNAME="${ADMIN_SURNAME:-User}"
ADMIN_ORG="${ADMIN_ORG:-Krkn Chaos}"

TOKEN_FILE="/tmp/krkn-jwt-token"
PORT_FORWARD_PID=""

# Cleanup function
cleanup() {
    if [ -n "$PORT_FORWARD_PID" ]; then
        log_info "Stopping port-forward (PID: $PORT_FORWARD_PID)..."
        kill $PORT_FORWARD_PID 2>/dev/null || true
        wait $PORT_FORWARD_PID 2>/dev/null || true
        log_info "Port-forward stopped"
    fi
}

# Setup trap for cleanup
trap cleanup EXIT INT TERM

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}krkn-operator API Test Script${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
if [ "$SETUP_PORT_FORWARD" = "true" ]; then
    echo "Port-forward: ENABLED"
    echo "Kubernetes context: $KUBE_CONTEXT"
    echo "Operator namespace: $OPERATOR_NAMESPACE"
    echo "Local port: $LOCAL_PORT"
else
    echo "Port-forward: DISABLED"
fi
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

setup_port_forward() {
    log_step "Setting up port-forward to krkn-operator service..."

    # Check if port is already in use
    if lsof -Pi :${LOCAL_PORT} -sTCP:LISTEN -t >/dev/null 2>&1; then
        log_warn "Port $LOCAL_PORT is already in use"
        read -p "Use existing port-forward? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            log_info "Using existing port-forward on port $LOCAL_PORT"
            return 0
        else
            log_error "Port $LOCAL_PORT is in use. Please free the port or use a different LOCAL_PORT."
            exit 1
        fi
    fi

    # Start port-forward
    log_info "Starting port-forward: $OPERATOR_NAMESPACE/svc/krkn-operator-operator ${LOCAL_PORT}:${SERVICE_PORT}"
    kubectl port-forward -n "$OPERATOR_NAMESPACE" \
        svc/krkn-operator-operator \
        ${LOCAL_PORT}:${SERVICE_PORT} \
        --context "$KUBE_CONTEXT" \
        >/dev/null 2>&1 &

    PORT_FORWARD_PID=$!
    log_info "Port-forward started (PID: $PORT_FORWARD_PID)"

    # Wait for port-forward to be ready
    log_info "Waiting for port-forward to be ready..."
    local timeout=30
    local elapsed=0
    while [ $elapsed -lt $timeout ]; do
        if curl -s http://localhost:${LOCAL_PORT}/api/v1/health >/dev/null 2>&1; then
            log_success "Port-forward is ready"
            return 0
        fi
        sleep 2
        elapsed=$((elapsed + 2))
    done

    log_error "Timeout waiting for port-forward to be ready"
    return 1
}

# Setup port-forward if enabled
if [ "$SETUP_PORT_FORWARD" = "true" ]; then
    setup_port_forward
    echo ""
fi

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

# Test 6: Discover available target clusters via async polling
log_step "Test 6: Discovering available target clusters (async flow)..."

# Step 1: Create target request (POST /api/v1/targets)
log_info "Step 1: Creating target request..."
RESPONSE=$(api_call POST "/api/v1/targets" "" "$TOKEN")
HTTP_CODE=$(extract_http_code "$RESPONSE")
BODY=$(extract_body "$RESPONSE")

if [ "$HTTP_CODE" != "200" ]; then
    log_error "Failed to create target request: HTTP $HTTP_CODE"
    echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"
    exit 1
fi

TARGET_UUID=$(echo "$BODY" | jq -r '.uuid')
if [ -z "$TARGET_UUID" ] || [ "$TARGET_UUID" = "null" ]; then
    log_error "Failed to extract UUID from target request"
    echo "$BODY"
    exit 1
fi

log_success "Target request created with UUID: $TARGET_UUID"
echo ""

# Step 2: Poll target status until ready (GET /api/v1/targets/{uuid})
log_info "Step 2: Polling target status (max 60 seconds)..."
MAX_POLLS=30
POLL_INTERVAL=2
poll_count=0

while [ $poll_count -lt $MAX_POLLS ]; do
    RESPONSE=$(api_call GET "/api/v1/targets/$TARGET_UUID" "" "$TOKEN")
    HTTP_CODE=$(extract_http_code "$RESPONSE")

    if [ "$HTTP_CODE" = "200" ]; then
        log_success "Target request completed (status: 200)"
        break
    elif [ "$HTTP_CODE" = "202" ]; then
        poll_count=$((poll_count + 1))
        log_info "Target pending... (attempt $poll_count/$MAX_POLLS, status: 202)"
        sleep $POLL_INTERVAL
    else
        log_error "Unexpected status code: $HTTP_CODE"
        BODY=$(extract_body "$RESPONSE")
        echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"
        exit 1
    fi
done

if [ $poll_count -ge $MAX_POLLS ]; then
    log_error "Timeout: Target request did not complete after $MAX_POLLS attempts"
    exit 1
fi

echo ""

# Step 3: Get clusters list (GET /api/v1/clusters?id={uuid})
log_info "Step 3: Fetching discovered clusters..."
RESPONSE=$(api_call GET "/api/v1/clusters?id=$TARGET_UUID" "" "$TOKEN")
HTTP_CODE=$(extract_http_code "$RESPONSE")
BODY=$(extract_body "$RESPONSE")

if [ "$HTTP_CODE" != "200" ]; then
    log_error "Failed to get clusters: HTTP $HTTP_CODE"
    echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"
    exit 1
fi

log_success "Clusters retrieved successfully"
log_info "Response:"
echo "$BODY" | jq '.'
echo ""

# Parse and validate clusters from targetData
# Response format: { "status": "Completed", "targetData": { "operator-name": [{ "cluster-name": "...", "cluster-api-url": "..." }] } }
TARGET_DATA=$(echo "$BODY" | jq -r '.targetData')
if [ "$TARGET_DATA" = "null" ] || [ -z "$TARGET_DATA" ]; then
    log_error "No targetData in response"
    exit 1
fi

# Flatten all clusters from all operators
ALL_CLUSTERS=$(echo "$TARGET_DATA" | jq -r '.[].[] | ."cluster-name"' | sort)
CLUSTER_COUNT=$(echo "$ALL_CLUSTERS" | wc -l | tr -d ' ')

log_info "Found $CLUSTER_COUNT clusters across all operators"

if [ "$CLUSTER_COUNT" != "3" ]; then
    log_error "ASSERTION FAILED: Expected 3 clusters, found $CLUSTER_COUNT"
    echo "Clusters found:"
    echo "$ALL_CLUSTERS"
    exit 1
fi

log_success "✓ ASSERTION PASSED: Found exactly 3 clusters"
echo ""

# Verify expected cluster names
log_info "Cluster names (sorted):"
echo "$ALL_CLUSTERS"

EXPECTED_CLUSTERS="cluster1
cluster2
local-cluster"

if [ "$ALL_CLUSTERS" = "$EXPECTED_CLUSTERS" ]; then
    log_success "✓ ASSERTION PASSED: All expected clusters found (local-cluster, cluster1, cluster2)"
else
    log_error "ASSERTION FAILED: Cluster names do not match expected"
    echo "Expected:"
    echo "$EXPECTED_CLUSTERS"
    echo ""
    echo "Found:"
    echo "$ALL_CLUSTERS"
    exit 1
fi

# Show detailed cluster information grouped by operator
echo ""
log_info "Cluster details (grouped by operator):"
echo "$TARGET_DATA" | jq -r 'to_entries[] | "  Operator: \(.key)" as $op | .value[] | "    - \(."cluster-name"): \(."cluster-api-url")"'
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
echo "  - Target clusters verified: 3 clusters (local-cluster, cluster1, cluster2)"
if [ -n "$PORT_FORWARD_PID" ]; then
    echo "  - Port-forward running (PID: $PORT_FORWARD_PID)"
fi
echo ""
echo "Saved token: $TOKEN_FILE"
echo "Token content (first 50 chars): ${TOKEN:0:50}..."
echo ""
if [ -n "$PORT_FORWARD_PID" ]; then
    echo "Port-forward info:"
    echo "  - Local: http://localhost:$LOCAL_PORT"
    echo "  - Service: $OPERATOR_NAMESPACE/krkn-operator-operator:$SERVICE_PORT"
    echo "  - Context: $KUBE_CONTEXT"
    echo "  - PID: $PORT_FORWARD_PID"
    echo ""
    echo "Note: Port-forward will be stopped when this script exits."
    echo "      To keep it running, start it manually:"
    echo "      kubectl port-forward -n $OPERATOR_NAMESPACE svc/krkn-operator-operator $LOCAL_PORT:$SERVICE_PORT --context $KUBE_CONTEXT"
    echo ""
fi
echo "Next steps:"
echo "1. Use the token for further API calls:"
echo "   TOKEN=\$(cat $TOKEN_FILE)"
echo '   curl -H "Authorization: Bearer $TOKEN" '$API_URL'/api/v1/operator/targets | jq'
echo ""
echo "2. Create and run chaos scenarios on discovered clusters"
echo "3. Test scenario execution and monitoring"
echo ""
