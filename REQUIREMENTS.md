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

- I want to change the current implementation of the KRKN job creation, I want to have a CRD called `KrknChaosJob` that must have 
  all the details currently defined in the `createScenarioJob` method in internal/api/handlers.go to instantiate a job
- I'd like that the reconcile loop keeps track of the Job status and updates the `KrknChaosJob` accordinglyall
- I want to have a controller able to reconcile the `KrknChaosJob` and instantiate the the chaos job as it does the createScenarioJob
- I want that the current /scenarios/run methods creates the new CR `KrknChaosJob` and returns the job uuid
- I want that the `GetScenarioRunStatus` is eventually adapted to this new behaviour

### Overview
TODO

