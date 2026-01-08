#!/bin/bash

# API Workflow Test Script
# This script tests the complete workflow of the krkn-operator REST API
#
# Usage: ./api_workflow_test.sh <host>
# Example: ./api_workflow_test.sh http://localhost:8080

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if host parameter is provided
if [ -z "$1" ]; then
    echo -e "${RED}Error: Host parameter is required${NC}"
    echo "Usage: $0 <host>"
    echo "Example: $0 http://localhost:8080"
    exit 1
fi

HOST="$1"
MAX_RETRIES=60  # Maximum number of retries for polling (60 * 5s = 5 minutes)
RETRY_INTERVAL=5  # Seconds between retries

echo -e "${GREEN}=== krkn-operator API Workflow Test ===${NC}"
echo -e "Host: ${YELLOW}${HOST}${NC}"
echo ""

# Function to check if jq is available
check_jq() {
    if ! command -v jq &> /dev/null; then
        echo -e "${RED}Error: jq is not installed${NC}"
        echo "Please install jq: https://stedolan.github.io/jq/"
        exit 1
    fi
}

# Function to make HTTP request and check status
http_request() {
    local method="$1"
    local endpoint="$2"
    local expected_status="$3"

    response=$(curl -s -w "\n%{http_code}" -X "${method}" "${HOST}${endpoint}")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    echo "$body"
    return $http_code
}

check_jq

# Step 1: POST /targets - Create a new target request
echo -e "${GREEN}Step 1: Creating new target request (POST /targets)${NC}"
response=$(curl -s -w "\n%{http_code}" -X POST "${HOST}/targets")
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" != "102" ]; then
    echo -e "${RED}Error: Expected status 102, got ${http_code}${NC}"
    echo "Response: $body"
    exit 1
fi

UUID=$(echo "$body" | jq -r '.uuid')
if [ -z "$UUID" ] || [ "$UUID" == "null" ]; then
    echo -e "${RED}Error: Failed to extract UUID from response${NC}"
    echo "Response: $body"
    exit 1
fi

echo -e "${GREEN}✓ Target request created successfully${NC}"
echo -e "UUID: ${YELLOW}${UUID}${NC}"
echo ""

# Step 2: Poll GET /targets/{UUID} until status is 200 (Completed)
echo -e "${GREEN}Step 2: Polling target request status (GET /targets/${UUID})${NC}"
echo -e "Waiting for request to complete (max ${MAX_RETRIES} retries, ${RETRY_INTERVAL}s interval)..."

retry_count=0
while [ $retry_count -lt $MAX_RETRIES ]; do
    http_code=$(curl -s -o /dev/null -w "%{http_code}" "${HOST}/targets/${UUID}")

    if [ "$http_code" == "200" ]; then
        echo -e "${GREEN}✓ Target request completed (status 200)${NC}"
        break
    elif [ "$http_code" == "100" ]; then
        echo -e "${YELLOW}⏳ Request still pending (status 100) - retry $((retry_count + 1))/${MAX_RETRIES}${NC}"
        sleep $RETRY_INTERVAL
        retry_count=$((retry_count + 1))
    elif [ "$http_code" == "404" ]; then
        echo -e "${RED}Error: Target request not found (status 404)${NC}"
        exit 1
    else
        echo -e "${RED}Error: Unexpected status code ${http_code}${NC}"
        exit 1
    fi
done

if [ $retry_count -ge $MAX_RETRIES ]; then
    echo -e "${RED}Error: Timeout waiting for target request to complete${NC}"
    exit 1
fi
echo ""

# Step 3: GET /clusters?id={UUID} - Get cluster list
echo -e "${GREEN}Step 3: Fetching cluster list (GET /clusters?id=${UUID})${NC}"
response=$(curl -s -w "\n%{http_code}" "${HOST}/clusters?id=${UUID}")
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" != "200" ]; then
    echo -e "${RED}Error: Expected status 200, got ${http_code}${NC}"
    echo "Response: $body"
    exit 1
fi

echo -e "${GREEN}✓ Cluster list retrieved successfully${NC}"
echo "Response:"
echo "$body" | jq '.'

# Extract the first cluster from the response
# Structure: { "targetData": { "operator-name": [{ "cluster-name": "...", "cluster-api-url": "..." }] } }
FIRST_OPERATOR=$(echo "$body" | jq -r '.targetData | keys[0]')
FIRST_CLUSTER_NAME=$(echo "$body" | jq -r ".targetData[\"${FIRST_OPERATOR}\"][0][\"cluster-name\"]")

if [ -z "$FIRST_CLUSTER_NAME" ] || [ "$FIRST_CLUSTER_NAME" == "null" ]; then
    echo -e "${RED}Error: No clusters found in response${NC}"
    exit 1
fi

echo -e "First cluster: ${YELLOW}${FIRST_CLUSTER_NAME}${NC} (from operator: ${FIRST_OPERATOR})"
echo ""

# Step 4: GET /nodes?id={UUID}&cluster-name={cluster-name} - Get nodes for the first cluster
echo -e "${GREEN}Step 4: Fetching nodes for cluster '${FIRST_CLUSTER_NAME}' (GET /nodes?id=${UUID}&cluster-name=${FIRST_CLUSTER_NAME})${NC}"
response=$(curl -s -w "\n%{http_code}" "${HOST}/nodes?id=${UUID}&cluster-name=${FIRST_CLUSTER_NAME}")
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" != "200" ]; then
    echo -e "${RED}Error: Expected status 200, got ${http_code}${NC}"
    echo "Response: $body"
    exit 1
fi

echo -e "${GREEN}✓ Node list retrieved successfully${NC}"
echo "Response:"
echo "$body" | jq '.'

# Extract and display node count
NODE_COUNT=$(echo "$body" | jq '.nodes | length')
echo -e "Total nodes: ${YELLOW}${NODE_COUNT}${NC}"
echo ""

# Summary
echo -e "${GREEN}=== Workflow completed successfully! ===${NC}"
echo -e "Summary:"
echo -e "  - UUID: ${YELLOW}${UUID}${NC}"
echo -e "  - Cluster: ${YELLOW}${FIRST_CLUSTER_NAME}${NC}"
echo -e "  - Nodes: ${YELLOW}${NODE_COUNT}${NC}"
