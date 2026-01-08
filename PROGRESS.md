# krkn-operator Development Progress

## Current Status

**Last updated:** 2026-01-08
**Current phase:** REST API Completion - COMPLETED

### Completed
- Project scaffolding with Kubebuilder
- CRD definitions:
  - `KrknTargetRequest`: manages target cluster requests with UUID-based identification (namespaced)
  - `KrknOperatorTargetProvider`: registers operator target providers
- Requirements analysis for REST API
- REST API server implementation in `internal/api/` using standard net/http
- GET /clusters endpoint implementation
  - Filters only "Completed" status requests
  - Namespace-aware (configurable via KRKN_NAMESPACE env var)
- GET /health endpoint implementation
- Integration with operator manager in main.go
- Comprehensive test suite for API endpoints
- Build verification successful
- **Refactoring completed:**
  - ✅ Replaced gin-gonic/gin with standard net/http library
  - ✅ Added namespace configuration via KRKN_NAMESPACE environment variable (default: "default")
  - ✅ Added status filter for "Completed" requests only
  - ✅ Updated health probe port to 8083 (from 8081)
  - ✅ Custom logging middleware for net/http
  - ✅ Response writer wrapper to capture status codes
- GET /nodes endpoint implementation (with gRPC integration)
  - Retrieves kubeconfig from secret based on cluster-name
  - Calls Python data-provider via gRPC to get actual node list
  - Returns list of node names from target cluster
- **gRPC Integration completed:**
  - ✅ Created protobuf definition (proto/dataprovider.proto)
  - ✅ Generated Go gRPC client code
  - ✅ Generated Python gRPC server code
  - ✅ Implemented Python data-provider service using krkn-lib
  - ✅ Integrated gRPC client in Go operator
  - ✅ End-to-end tested: Go API → gRPC → Python → krkn-lib → Kubernetes
- **Deployment Configuration completed:**
  - ✅ Created Dockerfile for Python data-provider (simplified, single-stage)
  - ✅ Created Kustomize patch for sidecar container
  - ✅ Updated Makefile with data-provider build targets (Docker + Podman)
  - ✅ Configured multi-container deployment
  - ✅ Created comprehensive deployment documentation (DEPLOYMENT.md)
- **Production Deployment completed:**
  - ✅ Added Podman support to Makefile (podman-build, podman-push, podman-build-all, podman-push-all)
  - ✅ Fixed Dockerfiles for podman compatibility (full image names: docker.io/library/...)
  - ✅ Added proto/ directory to operator Dockerfile
  - ✅ Configured namespace-scoped RBAC (Role instead of ClusterRole)
  - ✅ Added POD_NAMESPACE environment variable via Downward API
  - ✅ Configured Manager cache to watch only operator namespace
  - ✅ Added imagePullPolicy: Always for both containers
  - ✅ Created Kubernetes Service for REST API (api_service.yaml)
  - ✅ Updated deployment documentation with Service access patterns
  - ✅ Successfully deployed to production OpenShift cluster

### In Progress
- None

### Pending
- Controller implementation for KrknTargetRequest
- API documentation (OpenAPI/Swagger spec)
- Additional gRPC methods (GetPods, GetNamespaces, etc.)

## Requirements Overview

### REST API Endpoints

#### GET /clusters
**Purpose:** Return list of available target clusters for a specific completed request

**Parameters:**
- `id` (required): The UUID identifier of the KrknTargetRequest CR

**Behavior:**
1. Fetch KrknTargetRequest CR named with the provided id from configured namespace
2. Check if status is "Completed" (only completed requests are returned)
3. Extract and return the list of available clusters from CR status
4. Return 404 if CR with specified id does not exist or is not completed

**Response Format:**
```json
{
  "targetData": {
    "operator-name": [
      {
        "cluster-name": "cluster-1",
        "cluster-api-url": "https://api.cluster1.example.com"
      }
    ]
  },
  "status": "Completed"
}
```

**Error Responses:**

400 Bad Request (missing id):
```json
{
  "error": "bad_request",
  "message": "id parameter is required"
}
```

404 Not Found (CR not found or not completed):
```json
{
  "error": "not_found",
  "message": "KrknTargetRequest with id 'xyz' not found"
}
```

#### GET /nodes
**Purpose:** Return list of nodes from a specified cluster using gRPC data-provider

**Parameters:**
- `id` (required): The UUID identifier of the KrknTargetRequest CR
- `cluster-name` (required): The name of the cluster

**Behavior:**
1. Get secret from configured namespace (same logic as /clusters)
2. Retrieve kubeconfig (base64 encoded) from secret structure under the `cluster-name` object
3. Call gRPC data-provider service with kubeconfig
4. Data-provider uses krkn-lib to connect to cluster and fetch nodes
5. Return list of node names

**Status:** COMPLETED (Full Implementation with gRPC)

**Response Format:**
```json
{
  "nodes": ["node-1", "node-2", "node-3"]
}
```

**Error Responses:**

400 Bad Request (missing id):
```json
{
  "error": "bad_request",
  "message": "id parameter is required"
}
```

400 Bad Request (missing cluster-name):
```json
{
  "error": "bad_request",
  "message": "cluster-name parameter is required"
}
```

404 Not Found (secret not found):
```json
{
  "error": "not_found",
  "message": "Secret with id 'xyz' not found"
}
```

404 Not Found (cluster not in secret):
```json
{
  "error": "not_found",
  "message": "Kubeconfig for cluster 'cluster-name' not found in secret"
}
```

#### GET /targets/{UUID}
**Purpose:** Check the status of a KrknTargetRequest CR by UUID

**Parameters:**
- `uuid` (required): The UUID identifier in the path

**Behavior:**
1. Extract UUID from path `/targets/{uuid}`
2. Fetch KrknTargetRequest CR named with the UUID from configured namespace
3. Return 404 if CR not found
4. Return 100 (Continue) if CR found but status != "Completed"
5. Return 200 (OK) if CR found and status == "Completed"

**Status:** COMPLETED

**Response Codes:**
- `404 Not Found`: KrknTargetRequest with UUID not found
- `100 Continue`: KrknTargetRequest found but not completed
- `200 OK`: KrknTargetRequest found and completed
- `500 Internal Server Error`: Other errors

#### POST /targets
**Purpose:** Create a new KrknTargetRequest CR with a generated UUID

**Behavior:**
1. Generate a new UUID
2. Create a new KrknTargetRequest CR with:
   - metadata.name = UUID
   - spec.uuid = UUID
3. Create in the operator namespace
4. Return 102 (Processing) with the UUID

**Status:** COMPLETED

**Response Format:**
```json
{
  "uuid": "generated-uuid-here"
}
```

**Response Codes:**
- `102 Processing`: KrknTargetRequest created successfully
- `500 Internal Server Error`: Failed to create CR

## Architecture Decisions

### REST API Framework
**Selected:** Standard library net/http
**Rationale:**
- No external dependencies needed for simple HTTP serving
- Better control over middleware and request handling
- Lighter weight and more maintainable
- Previous: gin-gonic/gin (replaced during refactoring)

### Namespace Configuration
**Approach:** Environment variable with future ConfigMap migration path
**Implementation:**
- Current: `KRKN_NAMESPACE` environment variable (default: "default")
- Future: ConfigMap-based configuration (easy migration - only change in main.go)

### Data Provider Architecture
**Selected:** Sidecar container pattern with gRPC communication
**Rationale:**
- Separates concerns: Go operator handles CRD management, Python handles cluster operations
- Leverages krkn-lib (Python) without introducing Python dependencies to Go operator
- gRPC provides efficient, type-safe communication between containers
- Localhost communication within the pod is secure and fast
- Sidecar pattern allows independent scaling and updates

**Implementation:**
- Data-provider runs as sidecar container in same pod as operator
- Communication via gRPC on localhost:50051
- Protobuf defines service contract
- Python service uses krkn-lib for Kubernetes operations on target clusters

### Integration Points
1. REST API server runs alongside operator manager as a runnable
2. Uses same Kubernetes client from manager
3. Shares access to CRD resources (KrknTargetRequest) and Secrets
4. Namespace-aware operations via KRKN_NAMESPACE env var
5. gRPC client calls to data-provider sidecar for cluster operations

## Implementation Details

### Phase 1: REST API Setup ✅
1. ✅ Initial implementation with gin-gonic/gin framework (v1.11.0)
2. ✅ **REFACTORED:** Migrated to standard library net/http
3. ✅ Created API server structure in `internal/api/` package
   - `server.go`: API server with net/http.ServeMux and lifecycle management
   - `handlers.go`: HTTP handlers using http.ResponseWriter and *http.Request
   - `types.go`: Request/response type definitions
4. ✅ Setup logging middleware with custom responseWriter wrapper
5. ✅ Configured graceful shutdown with context

### Phase 2: GET /clusters Endpoint ✅
1. ✅ Implemented endpoint handler in `handlers.go`
2. ✅ Added Kubernetes client integration to fetch KrknTargetRequest CRs
3. ✅ **ENHANCED:** Added namespace awareness via KRKN_NAMESPACE
4. ✅ **ENHANCED:** Added status filter for "Completed" requests only
5. ✅ Implemented JSON response serialization
6. ✅ Added comprehensive error handling:
   - 400 Bad Request (missing id parameter)
   - 404 Not Found (CR not found or not completed)
   - 500 Internal Server Error (other errors)

### Phase 3: Integration ✅
1. ✅ Started API server from main.go as manager runnable
2. ✅ Configured server port via `--api-port` flag (default: 8080)
3. ✅ **ENHANCED:** Added namespace configuration via KRKN_NAMESPACE env var
4. ✅ Added `/health` endpoint
5. ✅ Ensured graceful shutdown via manager context
6. ✅ Updated health probe port to 8083 (from 8081)

### Phase 4: Testing ✅
1. ✅ Unit tests for all endpoint handlers (migrated to net/http)
   - TestGetClusters_Success
   - TestGetClusters_NotFound
   - TestGetClusters_NotCompleted (new test for status filter)
   - TestGetClusters_MissingID
   - TestHealthCheck
   - TestGetTargetByUUID_Success
   - TestGetTargetByUUID_NotFound
   - TestGetTargetByUUID_NotCompleted
   - TestGetTargetByUUID_MissingUUID
   - TestPostTarget_Success
2. ✅ Integration tests with fake Kubernetes client
3. ✅ All tests updated for namespace parameter
4. ✅ Tests updated for "Completed" status (capitalized)
5. ✅ API workflow test script (test/api_workflow_test.sh)
   - Automated end-to-end workflow testing
   - POST /targets → poll /targets/{UUID} → GET /clusters → GET /nodes
   - Configurable host parameter
   - Colored output and error handling

### Phase 5: GET /nodes Endpoint ✅
**Status:** COMPLETED (Full gRPC Integration)

1. ✅ Implemented endpoint handler in `handlers.go`
2. ✅ Added Secret fetching from configured namespace
3. ✅ Parameter validation for id and cluster-name
4. ✅ Kubeconfig retrieval from secret data (base64 encoded)
5. ✅ gRPC client integration to call data-provider
6. ✅ JSON response with node list from target cluster
7. ✅ Comprehensive error handling:
   - 400 Bad Request (missing id or cluster-name parameter)
   - 404 Not Found (secret not found or cluster not found in secret)
   - 500 Internal Server Error (gRPC errors, other errors)
8. ✅ Route registered in server.go
9. ✅ End-to-end tested with Python data-provider

**Implementation Details:**
- Retrieves kubeconfig from secret (base64 encoded under krkn-operator-acm → cluster-name → kubeconfig)
- Calls gRPC data-provider service on localhost:50051 (configurable via --grpc-server-address)
- Data-provider uses krkn-lib to connect to target cluster via kubeconfig string
- Returns list of node names from target cluster

### Phase 6: gRPC Data Provider ✅
**Status:** COMPLETED

1. ✅ Created protobuf service definition (proto/dataprovider.proto)
2. ✅ Generated Go client code (generate_proto_go.sh)
3. ✅ Generated Python server code (generate_proto.sh)
4. ✅ Implemented Python gRPC server (krkn-operator-data-provider/server.py)
5. ✅ Integrated krkn-lib for Kubernetes operations
6. ✅ Fixed Python imports for generated protobuf code
7. ✅ End-to-end tested: Go API → gRPC → Python → krkn-lib → Kubernetes

**Python Data Provider:**
- Location: `krkn-operator-data-provider/`
- Dependencies: grpcio, grpcio-tools, krkn-lib (from git branch init_from_string)
- Port: 50051 (gRPC)
- Implementation: Uses KrknKubernetes with kubeconfig_string parameter

### Phase 7: Deployment Configuration ✅
**Status:** COMPLETED

1. ✅ Created Dockerfile for Python data-provider
   - Multi-stage build (builder + runtime)
   - Python 3.11-slim base image
   - Optimized for size and security
2. ✅ Created Kustomize patch for sidecar deployment
   - config/default/data_provider_patch.yaml
   - Adds data-provider container to operator deployment
   - Configured resource limits and health probes
3. ✅ Updated config/default/kustomization.yaml
   - Added data_provider_patch.yaml to patches
4. ✅ Updated Makefile with new targets:
   - docker-build-data-provider: Build data-provider image
   - docker-push-data-provider: Push data-provider image
   - docker-build-all: Build both images
   - docker-push-all: Push both images
   - Updated deploy and build-installer to set both images
5. ✅ Created comprehensive deployment documentation
   - DEPLOYMENT.md with complete deployment guide
   - Build, push, and deploy instructions
   - Troubleshooting section
   - Multi-architecture build support

### Phase 8: Production Deployment ✅
**Status:** COMPLETED

1. ✅ Podman Support
   - Added podman-build, podman-push targets for operator
   - Added podman-build-data-provider, podman-push-data-provider targets
   - Added podman-build-all, podman-push-all for building both images
   - Updated DEPLOYMENT.md with podman usage examples

2. ✅ Dockerfile Fixes
   - Fixed operator Dockerfile: added COPY proto/ proto/
   - Fixed image names for podman: docker.io/library/golang:1.24
   - Fixed data-provider Dockerfile: docker.io/library/python:3.11-slim
   - Simplified data-provider to single-stage build (no venv)

3. ✅ RBAC Configuration
   - Converted ClusterRole to namespace-scoped Role
   - Converted ClusterRoleBinding to namespace-scoped RoleBinding
   - Added permissions for secrets, KrknTargetRequest, KrknOperatorTargetProvider
   - Namespace parametrized via kustomize (namespace: system placeholder)

4. ✅ Manager Configuration
   - Added POD_NAMESPACE env var via Kubernetes Downward API
   - Configured Manager cache to watch only operator namespace (Cache.DefaultNamespaces)
   - Separated operator namespace (POD_NAMESPACE) from CR namespace (KRKN_NAMESPACE)
   - Added cache import: sigs.k8s.io/controller-runtime/pkg/cache

5. ✅ Kubernetes Service
   - Created config/default/api_service.yaml for REST API
   - Service name: krkn-operator-controller-manager-api-service
   - Port: 8080, Type: ClusterIP
   - Updated kustomization.yaml to include api_service.yaml

6. ✅ Image Pull Policy
   - Added imagePullPolicy: Always to manager container (config/manager/manager.yaml)
   - Added imagePullPolicy: Always to data-provider container (config/default/data_provider_patch.yaml)
   - Ensures fresh images are pulled even with same tag

7. ✅ Production Testing
   - Successfully deployed to OpenShift cluster
   - Resolved RBAC permission errors
   - Verified namespace-scoped permissions work correctly
   - Tested REST API via Service
   - Verified gRPC communication between containers

## Technical Notes

### KrknTargetRequest Structure
```go
type ClusterTarget struct {
    ClusterName   string `json:"cluster-name"`
    ClusterAPIURL string `json:"cluster-api-url"`
}

type KrknTargetRequestStatus struct {
    Status     string                       `json:"status,omitempty"`
    TargetData map[string][]ClusterTarget  `json:"targetData,omitempty"`
    Created    *metav1.Time                 `json:"created,omitempty"`
    Completed  *metav1.Time                 `json:"completed,omitempty"`
}
```

**Status Values:**
- `"Pending"` - Request created but not yet processed
- `"Completed"` - Request processing finished, data available

### Implementation Files

**Go Operator:**
- `internal/api/server.go`: API server with net/http.ServeMux and lifecycle management
- `internal/api/handlers.go`: HTTP handlers using standard http.ResponseWriter/Request, gRPC client calls
- `internal/api/types.go`: Request/response type definitions
- `internal/api/handlers_test.go`: Comprehensive test suite (10 tests, all passing)
- `cmd/main.go`: Updated to integrate API server with namespace and gRPC configuration
- `start_operator.sh`: Launch script with support for KRKN_NAMESPACE env var

**Testing:**
- `test/api_workflow_test.sh`: End-to-end API workflow test script
  - Tests complete workflow: POST /targets → poll /targets/{UUID} → GET /clusters → GET /nodes
  - Usage: `./test/api_workflow_test.sh <host>`
  - Requires: curl, jq

**gRPC Protocol:**
- `proto/dataprovider.proto`: Protobuf service definition for data provider
- `proto/dataprovider/`: Generated Go client code
- `generate_proto_go.sh`: Script to generate Go gRPC code

**Python Data Provider:**
- `krkn-operator-data-provider/server.py`: Python gRPC server implementation
- `krkn-operator-data-provider/generated/`: Generated Python protobuf code
- `krkn-operator-data-provider/requirements.txt`: Python dependencies
- `krkn-operator-data-provider/generate_proto.sh`: Script to generate Python gRPC code
- `krkn-operator-data-provider/Dockerfile`: Multi-stage Docker build
- `krkn-operator-data-provider/.dockerignore`: Docker ignore rules

**Deployment Configuration:**
- `config/default/data_provider_patch.yaml`: Kustomize patch for sidecar container (with imagePullPolicy)
- `config/default/api_service.yaml`: Kubernetes Service for REST API
- `config/default/kustomization.yaml`: Updated with patches and api_service
- `config/manager/manager.yaml`: Updated with POD_NAMESPACE env var and imagePullPolicy
- `config/rbac/role.yaml`: Namespace-scoped Role (not ClusterRole)
- `config/rbac/role_binding.yaml`: Namespace-scoped RoleBinding
- `Makefile`: Updated with podman and data-provider build targets
- `Dockerfile`: Fixed with proto/ copy and full image names
- `krkn-operator-data-provider/Dockerfile`: Fixed with full image names, simplified build
- `DEPLOYMENT.md`: Comprehensive deployment documentation

### Server Configuration
- Default API port: 8080 (configurable via `--api-port` flag)
- Default health probe port: 8083 (configurable via `--health-probe-bind-address` flag)
- Default namespace: "default" (configurable via `KRKN_NAMESPACE` env var)
- Logging: Custom middleware with responseWriter wrapper for status code capture
- Graceful shutdown: Managed by controller-runtime manager
- Health check: Available at `/health`

### Environment Variables
- `POD_NAMESPACE`: Namespace where the operator pod is running (set via Downward API, fallback: "krkn-operator-system")
  - Used for Manager cache configuration (what namespace to watch)
- `KRKN_NAMESPACE`: Namespace for KrknTargetRequest CRs (default: same as POD_NAMESPACE)
  - Used by REST API to find CRs and Secrets
- `API_PORT`: REST API port (default: 8080)
- `HEALTH_PROBE_ADDR`: Health probe address (default: :8083)
- `METRICS_ADDR`: Metrics endpoint address (default: :8443)
- `--grpc-server-address`: gRPC data provider address (default: "localhost:50051")

## Next Steps

1. ✅ Implement namespace configuration via environment variable
2. ✅ Refactor from gin-gonic to net/http
3. ✅ Implement GET /nodes endpoint with gRPC integration
4. ✅ Implement gRPC communication with data provider
5. ✅ Create multi-container deployment configuration
6. ✅ Production deployment with namespace-scoped RBAC
7. ✅ Podman support and container tooling
8. ✅ Kubernetes Service for REST API access
9. Create API documentation (OpenAPI/Swagger spec)
10. Add additional endpoints as requirements evolve
11. Implement controller logic for KrknTargetRequest

## Future Work (Not in Current Scope)

- Controller implementation for KrknTargetRequest
- Controller implementation for KrknOperatorTargetProvider
- Migration from environment variable to ConfigMap for namespace configuration
- Additional REST API endpoints
- Additional gRPC methods in data-provider (e.g., GetPods, GetNamespaces, etc.)
- Helm chart for easier deployment
- CI/CD pipeline for automated builds and deployments