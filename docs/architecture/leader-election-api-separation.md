# Leader Election and API Server Separation

## Problem Statement

The krkn-operator was initially designed with a single leader election model where all components (controllers and API server) ran only on the elected leader replica. When multiple replicas were deployed behind a service load balancer, standby replicas would return HTTP 502 errors because the API server was not running on non-leader replicas.

## Architecture Decision

We have separated the leader election requirements between different components:

### Components Requiring Leader Election (Leader-Only)

**Controllers:**
- `KrknScenarioRunReconciler`
- `KrknTargetRequestReconciler`
- `KrknOperatorTargetProviderConfigReconciler`
- `ProviderRegistration`

**Why:** Reconciliation loops MUST run only on the leader to prevent:
- Multiple replicas reconciling the same resource simultaneously
- Infinite reconciliation loops
- Resource conflicts and race conditions
- Unnecessary API server load

**Implementation:** Controllers do NOT implement `NeedLeaderElection()`, so they use the default behavior (leader election required).

### Components Running on All Replicas (Multi-Active)

**API Server:**
- REST API endpoints
- WebSocket log streaming
- Authentication and authorization handlers

**Why:** The API server is completely stateless:
- No shared in-memory state between requests
- All persistence is in Kubernetes resources
- Kubernetes' optimistic concurrency control (resourceVersion) handles concurrent modifications
- Running on all replicas provides high availability and horizontal scalability

**ConfigStore Initializer:**
- Loads JWT secrets and configuration
- Each replica needs its own initialized cache

**Implementation:** These components implement `NeedLeaderElection()` returning `false`.

## Concurrency Safety

### API Server Concurrent Operations

The API server performs Create/Update/Delete operations on Kubernetes resources. This is safe when running on multiple replicas because:

1. **Optimistic Locking:** Kubernetes uses resourceVersion for optimistic concurrency control
   - Each resource has a version that increments on every modification
   - If two replicas modify the same resource, the second update receives a conflict error
   - The client can retry with the updated resource

2. **Idempotent Operations:** Most API operations are designed to be idempotent or fail gracefully
   - Create operations fail if resource already exists
   - Update operations fail if resourceVersion doesn't match
   - Delete operations are idempotent

3. **Stateless Design:** The API server maintains no in-memory state
   - Each HTTP request is independent
   - No session state or caching between requests
   - All state is persisted in Kubernetes CRDs

### Controller Safety with Leader Election

Controllers MUST use leader election because:

1. **Reconciliation Loops:** Only one replica should reconcile a given resource
   - Multiple reconcilers would trigger redundant operations
   - Could cause infinite update loops
   - Wastes cluster resources

2. **State Consistency:** Leader election ensures a single source of truth
   - Status updates reflect the actual state
   - No conflicting status updates from multiple replicas

3. **External Side Effects:** Some reconciliation may trigger external actions
   - Job creation, Pod deletion, etc.
   - These should happen exactly once per reconciliation

## Testing

The `internal/api/server_test.go` file contains tests that verify:

1. `TestServerNeedLeaderElection`: Verifies API server returns `false` for leader election
2. `TestServerImplementsLeaderElectionRunnable`: Verifies the interface is correctly implemented
3. `TestServerStatelessBehavior`: Documents the stateless design
4. `TestServerVsControllerLeaderElection`: Documents the difference between API and controllers

## Deployment Considerations

### High Availability

With this architecture:
- Deploy multiple replicas (e.g., `replicas: 3`)
- Service load balancer distributes API requests across all replicas
- All replicas can serve API traffic
- Only the leader runs reconciliation loops

### Scaling

- **Horizontal Scaling for API:** Increase replicas to handle more API traffic
- **Controller Performance:** Controllers run only on leader (single-threaded per resource)
- **Bottlenecks:** If controller throughput is a bottleneck, consider sharding resources by label/namespace

### Resource Requests

Example deployment configuration:

```yaml
replicas: 3  # Multiple replicas for API HA
resources:
  requests:
    cpu: 100m      # Per-replica CPU (API + leader election participation)
    memory: 128Mi  # Per-replica memory
  limits:
    cpu: 1000m     # Leader needs more CPU for reconciliation
    memory: 512Mi
```

## Migration Notes

### Before (Single Leader)

- API server ran only on leader replica
- Standby replicas returned 502 errors
- Service load balancer would fail ~66% of requests (in 3-replica setup)

### After (Multi-Active API)

- API server runs on all replicas
- All replicas can serve API traffic
- 0% failure rate from leader election
- Better resource utilization

### Rollout

The change is backward compatible:
- No API changes required
- No CRD changes required
- Simply deploy the new version
- Existing clients continue to work

## References

- [Controller Runtime Manager](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/manager)
- [Leader Election](https://pkg.go.dev/k8s.io/client-go/tools/leaderelection)
- [Optimistic Concurrency](https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions)