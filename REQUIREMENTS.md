# "ROAD TO PRODUCTION" REQUIREMENTS

## KrknTargetRequest controller in operator

### Overview
- Krkn Operator must be able to register itself as KrknOperatorTargetProvider via the CRD with name `krkn-operator`.
- Krkn Operator will expose an API to instantiate a new CRD called KrknOperatorTarget
- - `KrknOperatorTarget` must contain the following attributes
- - - `UUID` (String  - UIID)
- - - `ClusterName` (String)
- - - `ClusterAPIURL` (String)
- - - `SecretType` (String can assume "kubeconfig" or "token" or "credentials" values)
- - - `SecretUIID` (String - UUID)
- - - `CABundle` (String)
- krkn operator must expose a CRUD REST API and must adhere to the following rules
- - Creation
- - - A UUID must be created for the `UUID` field 
- - - Two `KrknOperatorTarget` with the same name or the same apiUrl are not allowed
- - - if `SecretType` is token or credentials:
- - - - `ClusterAPIURL` is mandatory
- - - - `ClusterCABundle` is optional, if not set the client will skip TLS verification
- - - if `SecretType` is kubeconfig the `ClusterAPIURL` must be extracted from the kubeconfig
- - - a Secret named with an `UUID` must be created containing a valid kubeconfig created accordingly with the credentials provided
- - Retrival
- - - The kubeconfig will be *never ever passed* back to the client, all the transaction must happen with the UUID and, where needed
      the kubeconfig is resolved accordingly.

- TODO (skip for the moment): krkn operator must have a controller for KrknTargetRequest and must populate it with KrknOperatorTarget as the [krkn-operator-acm](https://github.com/krkn-chaos/krkn-operator-acm/blob/main/internal/controller/krkntargetrequest_controller.go)
  does with the same logic (please read it with a lot of attention) 
## API and operator refactoring to support multiple clusters

The /scenarios/run api method must support multiple targets (so multiple UUIDs) and must instantiate as many krkn jobs as the
uuids passed as a target, it could be implemented simply iterating over the uuids using the same logic. tests must be adapted to this new
requirement.

### Overview
**Status: ✅ IMPLEMENTED**

The `/api/v1/scenarios/run` endpoint has been refactored to support multi-target scenario execution:

#### BREAKING CHANGES
- **Request Format**:
  - Changed from `targetUUID` (singular) to `targetUUIDs` (array, required, minimum 1 element)
  - Removed legacy fields: `targetId` and `clusterName`
  - All requests must now use the new `KrknOperatorTarget` system via UUIDs

- **Response Format**:
  - Changed from flat structure (`jobId`, `status`, `podName`) to array-based structure
  - Returns array of `jobs` with per-target results
  - Includes statistics: `totalTargets`, `successfulJobs`, `failedJobs`
  - Response is ALWAYS an array, even for single target requests

- **Pod Labels**:
  - Removed legacy labels: `krkn-target-id`, `krkn-cluster-name`
  - Added new label: `krkn-target-uuid` (replaces legacy labels)

#### Implementation Details

**Multi-Target Job Creation**:
- For each UUID in `targetUUIDs`, creates an independent krkn scenario job
- Best-effort execution: continues processing remaining targets even if one fails
- Each job gets unique `jobId` and separate Kubernetes resources (Pod, ConfigMaps, Secrets)

**Validation**:
- `targetUUIDs` is required and must contain at least 1 UUID
- No duplicate UUIDs allowed
- No empty strings allowed in the array
- `scenarioImage` and `scenarioName` remain required

**Response Behavior**:
- HTTP 201 (Created) if at least one job was created successfully
- HTTP 500 (Internal Server Error) if ALL jobs failed to create
- Each job in response has:
  - `targetUUID`: The target UUID
  - `jobId`: Unique job identifier (empty if failed)
  - `status`: "Pending" or "Failed"
  - `podName`: Kubernetes pod name (empty if failed)
  - `success`: Boolean indicating if job was created
  - `error`: Error message (only if `success` is false)

**List Endpoint Updates** (`GET /api/v1/scenarios/run`):
- Removed query parameters: `targetId`, `clusterName`
- Added query parameter: `targetUUID` (filters by `krkn-target-uuid` label)
- Response includes `targetUUID` field instead of legacy `targetId`/`clusterName`

#### Testing
Comprehensive test suite added with 11 test cases:
1. Single target success
2. Missing targetUUIDs validation
3. Multiple targets all succeed
4. Multiple targets partial failure (best-effort)
5. Multiple targets all fail
6. Validation tests (empty array, duplicates, empty strings)
7. List all scenario runs
8. Filter scenario runs by targetUUID

All tests passing ✅

## Moving from direct scenario creation to CRD approach

### ✅ Implemented
- Changed from direct Pod creation to CRD-based approach with `KrknScenarioRun`
- Controller reconciles `KrknScenarioRun` and creates jobs for each target cluster
- API endpoints updated to use `scenarioRunName` as primary identifier
- Multi-cluster support with aggregated status

### API Endpoints Structure (Nested Approach)

```
POST   /api/v1/scenarios/run
       → Creates KrknScenarioRun CR
       → Returns: {scenarioRunName, clusterNames, totalTargets}

GET    /api/v1/scenarios/run/{scenarioRunName}
       → Returns aggregated status with list of clusterJobs (each with jobId)

GET    /api/v1/scenarios/run/{scenarioRunName}/jobs/{jobId}
       → Returns status of a single job (TODO)

GET    /api/v1/scenarios/run/{scenarioRunName}/jobs/{jobId}/logs
       → WebSocket stream of logs for specific job (TODO - currently uses clusterName)

DELETE /api/v1/scenarios/run/{scenarioRunName}
       → Deletes entire scenario run (all jobs)

DELETE /api/v1/scenarios/run/{scenarioRunName}/jobs/{jobId}
       → Terminates a single job (TODO)
```

### 🔧 TODO: Pod Recreation and Retry Logic

#### Current Behavior
- Controller creates one pod per cluster when KrknScenarioRun is created
- No automatic retry on pod failure
- No distinction between user-initiated deletion and failure

#### Requirements

**1. Automatic Retry on Failure**
- When a pod fails (phase=Failed), the controller should retry creating a new pod
- Maximum number of retry attempts should be configurable (suggested: 3)
- Retry attempts should be tracked in ClusterJobStatus
- Exponential backoff between retries (suggested: 10s, 30s, 60s)

**2. Manual Cancellation vs Failure**
- User-initiated job deletion (DELETE /jobs/{jobId}) should NOT trigger retry
- Need to distinguish between "pod failed" and "user cancelled"
- Proposed solution: Add a field to ClusterJobStatus to track cancellation intent

**3. Job Lifecycle States**
```
Pending → Running → Succeeded (terminal)
                  → Failed → Retrying → Running → ...
                          → Cancelled (terminal, no retry)
                          → MaxRetriesExceeded (terminal)
```

#### Proposed Solution Options

##### Option A: Cancellation Field in Status (Recommended)
```go
type ClusterJobStatus struct {
    ClusterName string
    JobId       string
    PodName     string
    Phase       string  // Pending, Running, Succeeded, Failed, Cancelled, MaxRetriesExceeded

    // NEW FIELDS
    RetryCount      int       `json:"retryCount,omitempty"`
    MaxRetries      int       `json:"maxRetries,omitempty"`  // Default: 3
    CancelRequested bool      `json:"cancelRequested,omitempty"`
    LastRetryTime   *metav1.Time `json:"lastRetryTime,omitempty"`

    StartTime      *metav1.Time
    CompletionTime *metav1.Time
    Message        string
}
```

**Controller Logic**:
```go
// In updateClusterJobStatuses()
if pod.Status.Phase == corev1.PodFailed {
    if job.CancelRequested {
        job.Phase = "Cancelled"  // Terminal, no retry
    } else if job.RetryCount < job.MaxRetries {
        job.Phase = "Retrying"
        job.RetryCount++
        job.LastRetryTime = now
        // Create new pod with new jobId
        createClusterJob(ctx, scenarioRun, clusterName)
    } else {
        job.Phase = "MaxRetriesExceeded"  // Terminal
    }
}
```

**DELETE /jobs/{jobId} Handler**:
```go
func (h *Handler) DeleteJob(w http.ResponseWriter, r *http.Request) {
    // 1. Find KrknScenarioRun containing this jobId
    // 2. Set job.CancelRequested = true in status
    // 3. Delete the pod
    // 4. Controller sees CancelRequested → does NOT retry
}
```

##### Option B: Finalizers for Cancellation Tracking
- Add finalizer `krkn.krkn-chaos.dev/job-cancellation` to ClusterJobStatus
- When user deletes job, add finalizer before deleting pod
- Controller checks for finalizer → skips retry
- More complex, but leverages K8s patterns

##### Option C: Separate CancellationRequest CR
- Create a new CRD `KrknJobCancellation` to track cancellation intent
- Controller watches both KrknScenarioRun and KrknJobCancellation
- More decoupled, but adds complexity

#### Implementation Plan (TODO)

1. **Phase 1: Add Retry Fields to CRD**
   - Update `ClusterJobStatus` with retry tracking fields
   - Add `maxRetries` to `KrknScenarioRunSpec` (default: 3)
   - Regenerate manifests

2. **Phase 2: Implement Retry Logic in Controller**
   - Detect pod failure vs cancellation
   - Implement retry with exponential backoff
   - Update job phase to reflect retry state
   - Create new pod with new jobId on retry

3. **Phase 3: Add DELETE /jobs/{jobId} Endpoint**
   - Parse scenarioRunName and jobId from path
   - Set CancelRequested flag in CR status
   - Delete pod
   - Return success

4. **Phase 4: Update GET /jobs/{jobId} and Logs Endpoints**
   - Change from `/logs/{clusterName}` to `/logs/{jobId}`
   - Support nested path: `/scenarios/run/{scenarioRunName}/jobs/{jobId}/logs`
   - Add endpoint: `GET /scenarios/run/{scenarioRunName}/jobs/{jobId}` for single job status

#### Configuration

Add to KrknScenarioRunSpec:
```yaml
apiVersion: krkn.krkn-chaos.dev/v1alpha1
kind: KrknScenarioRun
spec:
  # ... existing fields ...

  # Retry configuration
  maxRetries: 3  # Default: 3, set to 0 to disable retry
  retryBackoff: exponential  # exponential or fixed
  retryDelay: 10s  # Initial delay for exponential, fixed delay for fixed
```

### Overview
The CRD-based approach provides better state management, automatic reconciliation, and improved observability compared to direct Pod creation.

# KrknOperatorTargetProviderConfig

I want to create this new CRD, it must behave exactly on the same way of KrknOperatorTargetRequest, each TargetProvider
that can share a configuration scheme will populate the CR status the key of the targetData property must be the name of the operator
the content will be a json with the following structure:

{
config-map: <config map name>
config-schema: <the configuration schema of the operator made in json format>
}

in case the json format could create formatting errors we can consider using base64 format

the status of the CR is completed when all the KrknOperatorTargetProvider registered providers added their entry in the 
targetData (that can be eventually empty). I want that you create a common function in pkg/provider to handle this CR update 
that will be reused by all the operator: it could take the operator name the configmap name and the json schema as parameter.

