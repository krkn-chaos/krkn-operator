# krkn-operator Development Progress (Post-MVP)

## Current Status

**Last updated:** 2026-01-15
**Current phase:** Planning post-MVP features

**MVP Documentation**: See [agent-specs/PROGRESS-MVP.md](../agent-specs/PROGRESS-MVP.md) for MVP development history.

## MVP Completion Summary

The MVP phase has been successfully completed with the following achievements:

### Core Infrastructure
- ✅ Kubernetes Operator scaffolding with Kubebuilder
- ✅ CRD definitions (KrknTargetRequest, KrknOperatorTargetProvider)
- ✅ REST API server with standard net/http
- ✅ gRPC data-provider sidecar integration
- ✅ OpenShift deployment with namespace-scoped RBAC

### API Endpoints Completed
- ✅ GET /health - Health check
- ✅ GET /clusters - List target clusters
- ✅ GET /nodes - List nodes from target cluster (via gRPC)
- ✅ POST /targets - Create target request
- ✅ GET /targets/{UUID} - Check target request status
- ✅ POST /scenarios - List available scenarios
- ✅ POST /scenarios/detail/{scenario_name} - Get scenario details
- ✅ POST /scenarios/globals/{scenario_name} - Get global environment fields
- ✅ POST /scenarios/run - Create and start scenario job
- ✅ GET /scenarios/run - List all jobs
- ✅ GET /scenarios/run/{jobId} - Get job status
- ✅ GET /scenarios/run/{jobId}/logs - Stream logs via WebSocket
- ✅ DELETE /scenarios/run/{jobId} - Stop and delete job

### Key Technical Achievements
- ✅ WebSocket log streaming with proper hijacker support
- ✅ ServiceAccount-based pod execution (UID 1001)
- ✅ Private registry support with ImagePullSecrets
- ✅ File mounting via ConfigMaps
- ✅ Comprehensive structured logging
- ✅ Helper function refactoring (getKubeconfigFromTarget)

---

## Post-MVP Development

### Phase 1: KrknOperatorTarget CRD and CRUD API
**Status:** COMPLETED
**Completed:** 2026-01-19

**Objectives:**
1. Create KrknOperatorTarget CRD for direct target management
2. Implement CRUD REST API for target management
3. Implement kubeconfig generation from token/credentials
4. Add comprehensive test coverage

**Implementation:**
- [x] Create KrknOperatorTarget CRD with kubebuilder
- [x] Generate CRD manifests and deepcopy code
- [x] Create internal/kubeconfig package with helper functions
- [x] Implement POST /api/v1/targets (Create)
- [x] Implement GET /api/v1/targets (List)
- [x] Implement GET /api/v1/targets/{uuid} (Get)
- [x] Implement PUT /api/v1/targets/{uuid} (Update)
- [x] Implement DELETE /api/v1/targets/{uuid} (Delete)
- [x] Add unit tests for kubeconfig helpers (100% pass)
- [x] Code organization - separated CRUD handlers in targets_crud.go

**Files Created:**
- api/v1alpha1/krknoperatortarget_types.go
- internal/kubeconfig/generator.go
- internal/kubeconfig/generator_test.go
- internal/api/targets_crud.go
- config/crd/bases/krkn.krkn-chaos.dev_krknoperatortargets.yaml

**Files Modified:**
- internal/api/server.go - Added /api/v1/targets routes
- internal/api/types.go - Added CreateTargetRequest, TargetResponse, etc.

**Testing:**
- [x] Unit tests for kubeconfig package (6 test functions, all passing)
- [ ] Unit tests for CRUD handlers (pending)
- [ ] Integration tests (pending)
- [ ] Manual testing (pending)

**Key Features:**
- Automatic kubeconfig generation from token or credentials
- Secret management with JSON-formatted kubeconfig storage
- Validation at API layer (clusterName and apiURL uniqueness)
- Support for 3 authentication methods: kubeconfig, token, credentials
- CA bundle support with optional InsecureSkipTLSVerify
- Automatic API URL extraction from kubeconfig

**Notes:**
- Secret always contains a valid kubeconfig (normalized format)
- Credentials/tokens are converted to kubeconfig before storage
- Status field is immediately "Ready" (validation happens at API layer)
- GET /clusters endpoint kept for backward compatibility with frontend

### Pending

*No pending phases*

### In Progress

*None*

### Completed

- Phase 1: KrknOperatorTarget CRD and CRUD API (2026-01-19)

---

## Development Phases Template

Use the following template when documenting new development phases:

```markdown
### Phase N: [Phase Name]
**Status:** PENDING/IN_PROGRESS/COMPLETED

**Objectives:**
1. Objective 1
2. Objective 2

**Implementation:**
- [ ] Task 1
- [ ] Task 2
- [ ] Task 3

**Files Modified:**
- path/to/file.go
- path/to/another/file.go

**Testing:**
- [ ] Unit tests
- [ ] Integration tests
- [ ] Manual testing

**Notes:**
- Important note 1
- Important note 2
```

---

## Next Steps

1. Define post-MVP requirements in REQUIREMENTS.md
2. Prioritize features for next iteration
3. Plan implementation phases
4. Update this document as work progresses

## Future Considerations

- Controller implementation for KrknTargetRequest
- Additional gRPC methods (GetPods, GetNamespaces, etc.)
- API authentication/authorization
- Metrics and observability improvements
- Helm chart for easier deployment
- CI/CD pipeline
- OpenAPI/Swagger documentation
