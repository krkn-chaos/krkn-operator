# krkn-operator Development Progress

## Current Status

**Last updated:** 2026-01-14
**Current phase:** Scenario Job Management - COMPLETED

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
- **POST /scenarios endpoint completed:**
  - ✅ Integrated krknctl provider package (v0.10.15-beta)
  - ✅ Implemented factory pattern for Quay and RegistryV2 providers
  - ✅ Default mode: quay.io registry (no body required)
  - ✅ Private registry support with authentication (username/password/token)
  - ✅ Returns list of available krkn scenario tags with metadata
  - ✅ Request/response types defined in internal/api/types.go
  - ✅ Handler registered at POST /scenarios
- **POST /scenarios/detail/{scenario_name} endpoint completed:**
  - ✅ Extracts scenario_name from URL path
  - ✅ Reuses krknctl models.ScenarioDetail (no custom DTOs)
  - ✅ Same registry configuration pattern as /scenarios
  - ✅ Calls GetScenarioDetail(scenario_name, registry)
  - ✅ Returns detailed scenario information (title, description, input fields)
  - ✅ Returns 404 if scenario not found
  - ✅ Handler registered at POST /scenarios/detail/{scenario_name}
- **POST /scenarios/globals/{scenario_name} endpoint completed:**
  - ✅ Extracts scenario_name from URL path
  - ✅ Same registry configuration pattern as /scenarios and /scenarios/detail
  - ✅ Body is optional (registry config only)
  - ✅ Defaults to quay.io when no body provided
  - ✅ Calls GetGlobalEnvironment(registry, scenario_name)
  - ✅ Returns global environment details for single scenario
  - ✅ Returns 404 if global environment not found
  - ✅ Type fields converted to strings (not enum integers)
  - ✅ Handler registered at POST /scenarios/globals/{scenario_name}
- **Scenario Job Management endpoints completed:**
  - ✅ POST /scenarios/run - Create and start scenario job
  - ✅ GET /scenarios/run/{jobId} - Get job status
  - ✅ GET /scenarios/run/{jobId}/logs - Stream logs (stub, requires clientset)
  - ✅ GET /scenarios/run - List all jobs with filtering
  - ✅ DELETE /scenarios/run/{jobId} - Stop and delete job
  - ✅ Router handler for endpoint dispatching
  - ✅ Complete kubeconfig retrieval and mounting
  - ✅ File mounts via ConfigMaps (base64 encoded)
  - ✅ Private registry support with ImagePullSecrets
  - ✅ Pod lifecycle management with cleanup
  - ✅ RBAC permissions updated
- **Code refactoring completed:**
  - ✅ Extracted getKubeconfigFromTarget() helper function
  - ✅ Refactored GET /nodes to use helper
  - ✅ Refactored POST /scenarios/run to use helper
  - ✅ Eliminated ~130 lines of duplicated code

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

#### POST /scenarios
**Purpose:** Retrieve list of available krkn chaos scenarios from registry

**Request Body (optional):**
```json
{
  "registryUrl": "registry.example.com",
  "scenarioRepository": "org/krkn-scenarios",
  "username": "user",
  "password": "pass",
  "token": "alternative-to-user-pass",
  "skipTls": false,
  "insecure": false
}
```

**Behavior:**
1. Parse optional request body
2. If body contains registry configuration → use RegistryV2 provider (Private mode)
3. If no body or empty body → use Quay provider (default to quay.io)
4. Load krknctl configuration (embedded config.json)
5. Create provider factory and instantiate appropriate provider
6. Call `GetRegistryImages(registry)` to fetch scenario list
7. Convert krknctl models to API response types
8. Return list of scenarios with metadata

**Status:** COMPLETED

**Response Format:**
```json
{
  "scenarios": [
    {
      "name": "pod-scenarios",
      "digest": "sha256:abc123...",
      "size": 123456789,
      "lastModified": "2025-01-12T10:30:00Z"
    },
    {
      "name": "node-scenarios",
      "digest": "sha256:def456...",
      "size": 987654321,
      "lastModified": "2025-01-11T15:20:00Z"
    }
  ]
}
```

**Error Responses:**

400 Bad Request (invalid body):
```json
{
  "error": "bad_request",
  "message": "Invalid request body: ..."
}
```

400 Bad Request (partial registry info):
```json
{
  "error": "bad_request",
  "message": "Both registryUrl and scenarioRepository are required for private registry"
}
```

500 Internal Server Error:
```json
{
  "error": "internal_error",
  "message": "Failed to get scenarios from registry: ..."
}
```

**Implementation Details:**
- Uses `github.com/krkn-chaos/krknctl/pkg/provider` package
- Factory pattern: `factory.NewProviderFactory(config).NewInstance(mode)`
- Two modes:
  - `provider.Quay`: Default quay.io registry (no authentication needed)
  - `provider.Private`: Custom registry with RegistryV2 configuration
- Supports multiple authentication methods: username/password, token
- Optional TLS skip and insecure connection for private registries

#### POST /scenarios/detail/{scenario_name}
**Purpose:** Retrieve detailed information about a specific chaos scenario

**Path Parameter:**
- `scenario_name` (required): The name/tag of the scenario to retrieve

**Request Body (optional):**
Same as POST /scenarios - registry configuration for private registry
```json
{
  "registryUrl": "registry.example.com",
  "scenarioRepository": "org/krkn-scenarios",
  "username": "user",
  "password": "pass",
  "token": "alternative-to-user-pass",
  "skipTls": false,
  "insecure": false
}
```

**Behavior:**
1. Extract scenario_name from URL path
2. Parse optional request body for registry configuration
3. If body contains registry configuration → use RegistryV2 provider (Private mode)
4. If no body or empty body → use Quay provider (default to quay.io)
5. Load krknctl configuration (embedded config.json)
6. Create provider factory and instantiate appropriate provider
7. Call `GetScenarioDetail(scenario_name, registry)` to fetch scenario details
8. Return scenario detail with input fields metadata
9. Return 404 if scenario not found

**Status:** COMPLETED

**Response Format (200 OK):**
```json
{
  "name": "pod-scenarios",
  "digest": "sha256:abc123...",
  "size": 123456789,
  "lastModified": "2025-01-13T10:30:00Z",
  "title": "Pod Scenarios",
  "description": "Chaos engineering scenarios for Kubernetes pods",
  "fields": [
    {
      "name": "namespace",
      "variable": "NAMESPACE",
      "type": "string",
      "description": "Target namespace for pod scenarios",
      "required": true,
      "default": "default"
    },
    {
      "name": "label_selector",
      "variable": "LABEL_SELECTOR",
      "type": "string",
      "description": "Label selector to filter pods",
      "required": false
    },
    {
      "name": "config_file",
      "variable": "CONFIG_FILE",
      "type": "file",
      "description": "Configuration file for scenario",
      "required": false,
      "mount_path": "/config/scenario.yaml"
    }
  ]
}
```

**Error Responses:**

400 Bad Request (missing scenario_name):
```json
{
  "error": "bad_request",
  "message": "scenario_name parameter is required in path"
}
```

400 Bad Request (invalid body):
```json
{
  "error": "bad_request",
  "message": "Invalid request body: ..."
}
```

400 Bad Request (partial registry info):
```json
{
  "error": "bad_request",
  "message": "Both registryUrl and scenarioRepository are required for private registry"
}
```

404 Not Found (scenario not found):
```json
{
  "error": "not_found",
  "message": "Scenario 'invalid-scenario' not found"
}
```

500 Internal Server Error:
```json
{
  "error": "internal_error",
  "message": "Failed to get scenario detail: ..."
}
```

**Implementation Details:**
- Uses `github.com/krkn-chaos/krknctl/pkg/provider` package
- Reuses krknctl `models.ScenarioDetail` structure (no custom DTOs)
- Factory pattern: `factory.NewProviderFactory(config).NewInstance(mode)`
- Same registry configuration pattern as POST /scenarios
- Two modes:
  - `provider.Quay`: Default quay.io registry
  - `provider.Private`: Custom registry with RegistryV2 configuration
- Returns complete scenario metadata including:
  - Basic info: name, digest, size, lastModified
  - Descriptive info: title, description
  - Input fields: detailed field configurations with types, validation, defaults

**Input Field Types:**
- `string`: Text input with optional regex validation
- `number`: Numeric input
- `boolean`: Boolean flag (true/false)
- `enum`: Enumerated values with allowed_values list
- `file`: File mount (requires mount_path)
- `file_base64`: Base64-encoded file content

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

### Phase 9: POST /scenarios Endpoint ✅
**Status:** COMPLETED

1. ✅ Dependency Integration
   - Added krknctl package (github.com/krkn-chaos/krknctl v0.10.15-beta)
   - Imported provider, factory, models, and config packages
   - Executed `go mod tidy` to resolve transitive dependencies

2. ✅ Type Definitions (internal/api/types.go)
   - Created `ScenariosRequest` struct for optional registry configuration
   - Created `ScenarioTag` struct matching krknctl models.ScenarioTag
   - Created `ScenariosResponse` with list of scenarios
   - All fields properly documented with JSON tags

3. ✅ Handler Implementation (internal/api/handlers.go)
   - Implemented `PostScenarios(w, r)` handler
   - Request body parsing with ContentLength check
   - Validation logic: both registryUrl and scenarioRepository required for private registry
   - Mode selection: provider.Quay (default) vs provider.Private
   - Factory pattern: `factory.NewProviderFactory(&cfg).NewInstance(mode)`
   - Call to `GetRegistryImages(registry)` for scenario list retrieval
   - Model conversion from krknctl types to API types
   - Comprehensive error handling (400, 500)

4. ✅ Route Registration (internal/api/server.go)
   - Registered POST /scenarios route
   - Handler accessible at http://operator:8080/scenarios

5. ✅ Build Verification
   - Successful compilation with no errors
   - All dependencies resolved
   - Binary built and ready for testing

**Implementation Highlights:**
- **No body**: Defaults to quay.io (public scenarios)
- **With body**: Private registry with RegistryV2 configuration
- **Authentication**: Supports username/password or token
- **Flexibility**: SkipTLS and Insecure options for development environments

### Phase 10: POST /scenarios/detail/{scenario_name} Endpoint ✅
**Status:** COMPLETED

1. ✅ Architecture Decision
   - Decided to reuse krknctl `models.ScenarioDetail` structure
   - No custom DTOs created - direct use of upstream models
   - Maintains consistency with krknctl ecosystem

2. ✅ Handler Implementation (internal/api/handlers.go)
   - Implemented `PostScenarioDetail(w, r)` handler
   - Path parameter extraction for scenario_name
   - Same registry configuration pattern as POST /scenarios
   - Request body parsing with ContentLength check
   - Validation logic: both registryUrl and scenarioRepository required for private registry
   - Mode selection: provider.Quay (default) vs provider.Private
   - Factory pattern: `factory.NewProviderFactory(&cfg).NewInstance(mode)`
   - Call to `GetScenarioDetail(scenario_name, registry)` for detailed scenario retrieval
   - 404 response when scenario not found
   - Direct JSON marshaling of krknctl models.ScenarioDetail
   - Comprehensive error handling (400, 404, 500)

3. ✅ Route Registration (internal/api/server.go)
   - Registered POST /scenarios/detail/{scenario_name} route
   - Handler accessible at http://operator:8080/scenarios/detail/{scenario_name}

4. ✅ Build Verification
   - Successful compilation with no errors
   - All dependencies resolved
   - Binary built and ready for testing

5. ✅ Documentation
   - Updated REQUIREMENTS.md with ✅ COMPLETED marker
   - Added complete endpoint documentation to PROGRESS.md
   - Documented all request/response formats
   - Documented all input field types (string, number, boolean, enum, file, file_base64)

**Implementation Highlights:**
- **Reuses upstream models**: No DTOs, direct krknctl models.ScenarioDetail
- **Consistent pattern**: Same registry config as POST /scenarios
- **Rich metadata**: Returns title, description, and complete field configurations
- **Field metadata includes**:
  - Field type and validation rules
  - Required/optional flags
  - Default values
  - Mount paths for file types
  - Dependencies and mutual exclusions
  - Secret field marking

### Refactoring: Type Field as String ✅
**Status:** COMPLETED

**Issue:** The `Type` field in input fields was being serialized as integer (enum value) instead of string.

**Solution:**
1. Created `InputFieldResponse` and `ScenarioDetailResponse` wrapper types
2. Convert `typing.Type` enum to string using `Type.String()` method
3. Map all fields from krknctl models to response models

**Before (incorrect):**
```json
{
  "type": 0  // Integer enum value
}
```

**After (correct):**
```json
{
  "type": "string"  // Human-readable string
}
```

**Possible type values:**
- `"string"`, `"number"`, `"boolean"`, `"enum"`, `"file"`, `"file_base64"`

### Phase 11: POST /scenarios/globals/{scenario_name} Endpoint ✅
**Status:** COMPLETED

1. ✅ Handler Implementation (internal/api/handlers.go)
   - Implemented `PostScenarioGlobals(w, r)` handler
   - Extracts scenario_name from URL path
   - Request body is optional (same as /scenarios/detail)
   - Same registry configuration pattern as other scenario endpoints
   - Mode selection: provider.Quay (default) vs provider.Private
   - Factory pattern: `factory.NewProviderFactory(&cfg).NewInstance(mode)`
   - Calls `GetGlobalEnvironment(registry, scenarioName)` for single scenario
   - Returns 404 if global environment not found
   - Converts Type fields to strings using `field.Type.String()`
   - Returns `ScenarioDetailResponse` directly (not a map)
   - Comprehensive error handling (400, 404, 500)

2. ✅ Route Registration (internal/api/server.go)
   - Registered POST /scenarios/globals/{scenario_name} route
   - Handler accessible at http://operator:8080/scenarios/globals/{scenario_name}

3. ✅ Build Verification
   - Successful compilation with no errors
   - All dependencies resolved
   - Binary built and ready for testing

**Implementation Highlights:**
- **Path parameter**: scenario_name in URL path (same as /scenarios/detail)
- **Body optional**: Only for private registry configuration
- **Default mode**: Uses quay.io when no body provided
- **Response format**: Single ScenarioDetailResponse (not a map)
- **Consistency**: Identical pattern to /scenarios/detail/{scenario_name}
- **Error handling**: 404 when global environment not found

**Usage Examples:**

Default (quay.io):
```bash
curl -X POST http://localhost:8080/scenarios/globals/pod-scenarios | jq
```

With private registry:
```bash
curl -X POST http://localhost:8080/scenarios/globals/pod-scenarios \
  -H "Content-Type: application/json" \
  -d '{
    "registryUrl": "registry.example.com",
    "scenarioRepository": "org/krkn-scenarios"
  }' | jq
```

**Response Example:**
```json
{
  "name": "krkn",
  "title": "Global Environment",
  "description": "Global environment variables for krkn",
  "fields": [
    {
      "name": "kubeconfig",
      "variable": "KUBECONFIG",
      "type": "file",
      "required": true,
      "mount_path": "/root/.kube/config"
    }
  ]
}
```

### Phase 12: Scenario Job Management - /scenarios/run Endpoints ✅
**Status:** COMPLETED

1. ✅ Type Definitions (internal/api/types.go)
   - Created `FileMount` struct for user file uploads
   - Created `ScenarioRunRequest` with embedded `ScenariosRequest`
   - Created `ScenarioRunResponse` for job creation
   - Created `JobStatusResponse` with targetId, clusterName metadata
   - Created `JobsListResponse` for listing jobs

2. ✅ POST /scenarios/run Implementation (internal/api/handlers.go)
   - Validates required fields: targetId, clusterName, scenarioImage, scenarioName
   - Uses `getKubeconfigFromTarget()` helper to retrieve kubeconfig
   - Creates ConfigMap for kubeconfig (decoded from base64)
   - Creates ConfigMaps for user-provided files (base64 decoded)
   - Supports private registry with ImagePullSecret creation
   - Creates pod with:
     - RestartPolicy: Never (one-time execution)
     - Labels: app, krkn-job-id, krkn-scenario-name, krkn-target-id, krkn-cluster-name
     - Environment variables from request
     - Volume mounts for kubeconfig and user files
     - ImagePullSecrets if registry credentials provided
   - Returns 201 Created with jobId, status "Pending", podName
   - Complete cleanup on error (deletes created ConfigMaps/Secrets)

3. ✅ GET /scenarios/run/{jobId} Implementation
   - Finds pod by label `krkn-job-id`
   - Maps pod phase to job status
   - Extracts metadata from pod labels
   - Returns timestamps (startTime, completionTime)
   - Returns error messages for failed jobs

4. ✅ GET /scenarios/run/{jobId}/logs Implementation (STUB)
   - Endpoint registered and routed correctly
   - Returns 501 Not Implemented
   - TODO: Requires Kubernetes clientset integration for actual log streaming
   - Query parameter parsing ready (follow, timestamps, tailLines)

5. ✅ GET /scenarios/run Implementation
   - Lists all pods with label `app=krkn-scenario`
   - Supports filtering by: status, scenarioName, targetId, clusterName
   - Returns array of JobStatusResponse
   - Includes timestamps and metadata for each job

6. ✅ DELETE /scenarios/run/{jobId} Implementation
   - Finds and deletes pod with 5 second grace period
   - Cleanup associated ConfigMaps (by label `krkn-job-id`)
   - Cleanup associated Secrets (ImagePullSecrets)
   - Returns status "Stopped" with success message

7. ✅ Router Handler (ScenariosRunRouter)
   - Dispatches requests based on path and method
   - POST /scenarios/run → PostScenarioRun
   - GET /scenarios/run → ListScenarioRuns
   - GET /scenarios/run/{jobId} → GetScenarioRunStatus
   - GET /scenarios/run/{jobId}/logs → GetScenarioRunLogs
   - DELETE /scenarios/run/{jobId} → DeleteScenarioRun
   - Returns 405 Method Not Allowed for invalid methods

8. ✅ RBAC Updates (config/rbac/role.yaml)
   - pods: added create, delete verbs
   - pods/log: added get verb
   - configmaps: added get, list, create, delete verbs
   - secrets: added create, delete verbs

9. ✅ Build Verification
   - Successful compilation with no errors
   - All handlers properly integrated
   - Routes registered correctly

**Implementation Highlights:**
- **Pod naming**: `krkn-job-{jobId}`
- **ConfigMap naming**: `krkn-job-{jobId}-kubeconfig`, `krkn-job-{jobId}-file-{sanitized-name}`
- **Default kubeconfig path**: `/home/krkn/.kube/config` (configurable)
- **Error handling**: Comprehensive cleanup on failures
- **Label-based tracking**: No CRD required, uses pod labels
- **Private registry**: Full support with ImagePullSecret creation

**Request Example:**
```json
{
  "targetId": "550e8400-e29b-41d4-a716-446655440000",
  "clusterName": "my-cluster-1",
  "scenarioImage": "quay.io/krkn-chaos/krkn-hub:pod-scenarios",
  "scenarioName": "pod-scenarios",
  "kubeconfigPath": "/home/krkn/.kube/config",
  "environment": {
    "NAMESPACE": "default",
    "LABEL_SELECTOR": "app=myapp"
  },
  "files": [
    {
      "name": "config.yaml",
      "content": "base64-encoded-content",
      "mountPath": "/config/scenario.yaml"
    }
  ]
}
```

**Response Example:**
```json
{
  "jobId": "generated-uuid",
  "status": "Pending",
  "podName": "krkn-job-generated-uuid"
}
```

### Code Refactoring: getKubeconfigFromTarget() Helper ✅
**Status:** COMPLETED

**Problem:** Kubeconfig retrieval logic was duplicated in GET /nodes and POST /scenarios/run (~130 lines total)

**Solution:** Extracted common helper function `getKubeconfigFromTarget(ctx, targetId, clusterName)`

**Location:** internal/api/handlers.go:275-318

**Functionality:**
1. Fetches secret named `targetId` from operator namespace
2. Parses `managed-clusters` JSON from secret data
3. Navigates structure: `["krkn-operator-acm"][clusterName]["kubeconfig"]`
4. Returns base64-encoded kubeconfig string
5. Returns descriptive errors for all failure cases

**Refactored Endpoints:**
1. GET /nodes (lines ~137-156): Replaced ~65 lines with helper call
2. POST /scenarios/run (lines ~820-838): Replaced ~70 lines with helper call

**Benefits:**
- **DRY Principle**: Single source of truth for kubeconfig retrieval
- **Maintainability**: Changes only needed in one place
- **Consistency**: Identical error handling across endpoints
- **Testability**: Helper can be unit tested independently
- **Code reduction**: ~130 lines eliminated

**Before:**
```go
// Duplicated in GET /nodes and POST /scenarios/run
var secret corev1.Secret
err := h.client.Get(ctx, ...)
// ... 60+ lines of identical code ...
```

**After:**
```go
// Both endpoints now use:
kubeconfigBase64, err := h.getKubeconfigFromTarget(ctx, targetId, clusterName)
if err != nil {
    // Error handling
}
```

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
- `internal/api/server.go`: API server with net/http.ServeMux and lifecycle management, routes for all endpoints
- `internal/api/handlers.go`: HTTP handlers using standard http.ResponseWriter/Request, gRPC client calls, krknctl provider integration
- `internal/api/types.go`: Request/response type definitions (Clusters, Nodes, Scenarios, Targets, Errors)
- `internal/api/handlers_test.go`: Comprehensive test suite (10 tests, all passing)
- `cmd/main.go`: Updated to integrate API server with namespace and gRPC configuration
- `start_operator.sh`: Launch script with support for KRKN_NAMESPACE env var
- `go.mod`: Dependencies including krknctl v0.10.15-beta for scenario provider

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
9. ✅ Implement POST /scenarios endpoint with krknctl integration
10. ✅ Implement POST /scenarios/detail/{scenario_name} endpoint
11. ✅ Implement POST /scenarios/globals endpoint
12. ✅ Implement Scenario Job Management (/scenarios/run endpoints)
13. ✅ Code refactoring - extract getKubeconfigFromTarget() helper
14. Complete GET /scenarios/run/{jobId}/logs endpoint (requires Kubernetes clientset)
15. Create API documentation (OpenAPI/Swagger spec)
16. Add additional endpoints as requirements evolve
17. Implement controller logic for KrknTargetRequest

## Future Work (Not in Current Scope)

- Complete log streaming implementation for GET /scenarios/run/{jobId}/logs
  - Requires Kubernetes clientset integration (rest.Config)
  - Need to add clientset to Handler struct
  - Implement actual streaming with io.Copy from pod logs API
- Controller implementation for KrknTargetRequest
- Controller implementation for KrknOperatorTargetProvider
- Migration from environment variable to ConfigMap for namespace configuration
- Additional REST API endpoints
- Additional gRPC methods in data-provider (e.g., GetPods, GetNamespaces, etc.)
- Helm chart for easier deployment
- CI/CD pipeline for automated builds and deployments