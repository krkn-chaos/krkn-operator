# krkn-operator Test Scripts

This directory contains test scripts for the krkn-operator REST API.

## API Workflow Test

### Overview

`api_workflow_test.sh` is an end-to-end test script that validates the complete workflow of the krkn-operator REST API.

### Prerequisites

- `curl` - for making HTTP requests
- `jq` - for JSON parsing (install from https://stedolan.github.io/jq/)
- Access to a running krkn-operator instance

### Workflow Steps

The script executes the following workflow:

1. **POST /targets** - Creates a new KrknTargetRequest and captures the UUID
2. **GET /targets/{UUID}** - Polls the request status until completed (200 OK)
   - Retries every 5 seconds
   - Max retries: 60 (5 minutes timeout)
   - Expects 100 (Continue) while pending, 200 (OK) when completed
3. **GET /clusters?id={UUID}** - Retrieves the list of available clusters
4. **GET /nodes?id={UUID}&cluster-name={name}** - Fetches nodes from the first cluster

### Usage

```bash
./test/api_workflow_test.sh <host>
```

**Examples:**

```bash
# Local development
./test/api_workflow_test.sh http://localhost:8080

# Remote cluster
./test/api_workflow_test.sh http://krkn-operator-service.namespace.svc.cluster.local:8080

# OpenShift route
./test/api_workflow_test.sh https://krkn-operator-route.apps.cluster.example.com
```

### Output

The script provides colored output for easy monitoring:

- **Green** - Successful steps
- **Yellow** - Pending/waiting states
- **Red** - Errors

Example output:

```
=== krkn-operator API Workflow Test ===
Host: http://localhost:8080

Step 1: Creating new target request (POST /targets)
✓ Target request created successfully
UUID: 123e4567-e89b-12d3-a456-426614174000

Step 2: Polling target request status (GET /targets/123e4567-e89b-12d3-a456-426614174000)
Waiting for request to complete (max 60 retries, 5s interval)...
⏳ Request still pending (status 100) - retry 1/60
⏳ Request still pending (status 100) - retry 2/60
✓ Target request completed (status 200)

Step 3: Fetching cluster list (GET /clusters?id=123e4567-e89b-12d3-a456-426614174000)
✓ Cluster list retrieved successfully
Response:
{
  "targetData": {
    "krkn-operator-acm": [
      {
        "cluster-name": "cluster-1",
        "cluster-api-url": "https://api.cluster1.example.com"
      }
    ]
  },
  "status": "Completed"
}
First cluster: cluster-1 (from operator: krkn-operator-acm)

Step 4: Fetching nodes for cluster 'cluster-1' (GET /nodes?id=123e4567-e89b-12d3-a456-426614174000&cluster-name=cluster-1)
✓ Node list retrieved successfully
Response:
{
  "nodes": [
    "node-1",
    "node-2",
    "node-3"
  ]
}
Total nodes: 3

=== Workflow completed successfully! ===
Summary:
  - UUID: 123e4567-e89b-12d3-a456-426614174000
  - Cluster: cluster-1
  - Nodes: 3
```

### Error Handling

The script will exit with code 1 and display error messages if:

- Host parameter is missing
- jq is not installed
- Any HTTP request fails with unexpected status code
- Target request is not found (404)
- Timeout waiting for request completion (5 minutes)
- No clusters found in response
- Any other API error occurs

### Configuration

You can modify these variables at the top of the script:

- `MAX_RETRIES` - Maximum polling attempts (default: 60)
- `RETRY_INTERVAL` - Seconds between retries (default: 5)

### Testing Against Production

When testing against a production OpenShift cluster:

1. Forward the service port locally:
   ```bash
   kubectl port-forward -n krkn-operator-system svc/krkn-operator-controller-manager-api-service 8080:8080
   ```

2. Run the test:
   ```bash
   ./test/api_workflow_test.sh http://localhost:8080
   ```

Or use the service URL directly if you have network access:

```bash
./test/api_workflow_test.sh http://krkn-operator-controller-manager-api-service.krkn-operator-system.svc.cluster.local:8080
```
