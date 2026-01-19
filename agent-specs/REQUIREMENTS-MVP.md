# Overview
This is an operator that will manage the orchestration of the chaos scenarios exposing
a REST API to interact with a frontend where the chaos will be orchestrated. The Operator will
also comunicate via gRPC with a python container which service will be located in the
krkn-operator-data-provider folder, this container will be deployed together with the operator 
container in the same pod.

# API Requirements

- The operator must expose a REST API

## GET methods
### /clusters
- this method will return the list of available target clusters
- this method will take a CR id as parameter
- the available clusters will be fetched from the cluster CR KrknTarget request
  that will be named after the id passed as a parameter.
  if no CR with is id is present a 404 status will be returned.
  you can find the definition in the api/v1alpha1 folder.




#### refactoring
- select only KrknTargetRequests *only* if are completed
- replace the gin gonic framework with plain net/http for API serving


### /nodes
- this method will return the nodes of a specified cluster
- this method will take two parameters, CR id and cluster-name
- this method will get a secret from the configured namespace (the same logic as /clusters)
- from the secret content structure will retrieve the kubeconfig contained inside the secret structure under the `cluster-name` object
- for the moment the method will return the kubeconfig (this will be refactored)

### /targets/{UUID} ✅ COMPLETED
- this method will return 404 if the UUID is not found
- this method will return 100 if the UUID is found but the status is not Completed
- this method will return 200 if the UUID is found and Completed

## POST methods

### /targets ✅ COMPLETED

- this method will create a new KrknTargetRequest CR in the same operator namespace with:
  - metadata.name = generated UUID
  - spec.uuid = generated UUID
- this method will return a 102 status and the UUID of the KrknTargetRequest

### /scenarios ✅ COMPLETED
- this method must be built using the already available golang package made for krknctl to retrieve
  the available krkn scenarios either from quay.io or from a private registry
- the package is on the following repo https://github.com/krkn-chaos/krknctl/tree/main/pkg/provider
- the method must instantiate the factory of the scenario providers based on the user parameters
- - if the payload contains private registry infos `RegistryV2` provider must be instantiated
- - if no payload is passed it must default on quay.io
- the purpose is to return the list of the available krkn scenarios

### /scenarios/detail/{scenario_name} ✅ COMPLETED
- this method must be built using the already available golang package made for krknctl to retrieve
  the available krkn scenarios either from quay.io or from a private registry
- the package is on the following repo https://github.com/krkn-chaos/krknctl/tree/main/pkg/provider
- the method must instantiate the factory of the scenario providers based on the user parameters
- - if the payload contains private registry infos `RegistryV2` provider must be instantiated
- - if no payload is passed it must default on quay.io
- the purpose is to return the detail of the scenario in json format using the `GetScenarioDetail` method or 404 if not found

### /scenarios/globals/{scenario_name} ✅ COMPLETED
- this method must be built using the already available golang package made for krknctl to retrieve
  the available krkn scenarios either from quay.io or from a private registry
- the package is on the following repo https://github.com/krkn-chaos/krknctl/tree/main/pkg/provider
- the method must instantiate the factory of the scenario providers based on the user parameters
- - if the payload contains private registry infos `RegistryV2` provider must be instantiated
- - if no payload is passed it must default on quay.io
- the purpose is to return the global fields of the scenario using the `GetGlobalEnvironment` method of the provider
- the method takes scenario_name as path parameter (same pattern as /scenarios/detail/{scenario_name})
- the method returns 404 if global environment not found

### /scenarios/run/{UUID}
- this endpoint will instantiate a krknfai 


#### refactoring
- type should be returned with the string value of the enum and not with the numeric value ✅ COMPLETED
- since the retrival of the KrknTargetRequest object and the respective secret to return the kubeconfig of one of the target cluster
  is performed in several parts of the code, I want to have a common function to do that and I want it used in every place where this
  operation is made. ✅ COMPLETED

# Scenario Job Management

The operator must be able to schedule scenario pods in its own namespace, track their execution, stream their logs, and stop them on demand.

## POST methods

### /scenarios/run
- this method creates and starts a new scenario job as a Kubernetes pod
- the method accepts a JSON payload with scenario configuration
- the payload must include:
  - `targetId`: the UUID of the KrknTargetRequest CR (required)
  - `clusterName`: the name of the target cluster (required)
  - `scenarioImage`: the container image to run (required)
  - `scenarioName`: the name of the scenario being executed (required)
  - `environment`: map of environment variables to pass to the container (optional)
  - `files`: array of file objects to mount in the container (optional)
  - `kubeconfigPath`: the path where kubeconfig should be mounted in the container (optional, default: `/home/krkn/.kube/config`)
- the payload must support private registry configuration (same as other scenario endpoints):
  - `registryUrl`: private registry URL (optional)
  - `scenarioRepository`: scenario repository name (optional)
  - `username`: registry username (optional)
  - `password`: registry password (optional)
  - `token`: registry token, alternative to username/password (optional)
  - `skipTls`: skip TLS verification (optional, default: false)
  - `insecure`: allow insecure connections (optional, default: false)
- file objects must contain:
  - `name`: the file name
  - `content`: the file content encoded in base64
  - `mountPath`: the absolute path where the file should be mounted in the container
- the method must:
  - generate a unique job ID (UUID)
  - fetch the secret named `targetId` from the operator namespace
  - extract kubeconfig from secret structure: `secret.Data["managed-clusters"]` → JSON → `["krkn-operator-acm"][clusterName]["kubeconfig"]`
  - the kubeconfig in the secret is already base64 encoded
  - create a ConfigMap containing the decoded kubeconfig
  - create a Kubernetes pod in the operator namespace with appropriate labels for tracking
  - mount the kubeconfig ConfigMap as a volume in the pod at the path specified by `kubeconfigPath` (default: `/home/krkn/.kube/config`)
  - support private registry authentication via imagePullSecrets if registry config provided
  - create ConfigMaps for file mounts if files are provided
  - mount file ConfigMaps as volumes in the pod at specified mountPaths
  - return 201 status with the job ID and initial status
  - return 404 if targetId secret not found
  - return 404 if clusterName not found in secret structure
- pod labels must include:
  - `app: krkn-scenario`
  - `krkn-job-id: <uuid>`
  - `krkn-scenario-name: <scenarioName>`
  - `krkn-target-id: <targetId>`
  - `krkn-cluster-name: <clusterName>`
- the method must return the job ID and initial status (Pending)

### Request example:
```json
{
  "targetId": "550e8400-e29b-41d4-a716-446655440000",
  "clusterName": "my-cluster-1",
  "scenarioImage": "quay.io/krkn-chaos/krkn-hub:pod-scenarios",
  "scenarioName": "pod-scenarios",
  "kubeconfigPath": "/home/krkn/.kube/config",
  "environment": {
    "NAMESPACE": "default",
    "LABEL_SELECTOR": "app=myapp",
    "POD_COUNT": "1"
  },
  "files": [
    {
      "name": "config.yaml",
      "content": "base64-encoded-content-here",
      "mountPath": "/config/scenario.yaml"
    }
  ],
  "registryUrl": "registry.example.com",
  "scenarioRepository": "org/krkn-scenarios",
  "username": "user",
  "password": "pass"
}
```

### Response example:
```json
{
  "jobId": "550e8400-e29b-41d4-a716-446655440000",
  "status": "Pending",
  "podName": "krkn-job-550e8400-e29b-41d4-a716-446655440000"
}
```

## GET methods

### /scenarios/run/{jobId}
- this method returns the current status of a specific job
- the method accepts the job ID as a path parameter
- the method must:
  - find the pod associated with the job ID using labels
  - return 404 if job not found
  - return job status information including:
    - `jobId`: the unique job identifier
    - `targetId`: the KrknTargetRequest UUID
    - `clusterName`: the target cluster name
    - `scenarioName`: the scenario name
    - `status`: current status (Pending, Running, Succeeded, Failed, Stopped)
    - `podName`: the Kubernetes pod name
    - `startTime`: when the job started (optional)
    - `completionTime`: when the job completed (optional)
    - `message`: additional status message or error details (optional)

### Response example:
```json
{
  "jobId": "550e8400-e29b-41d4-a716-446655440000",
  "targetId": "550e8400-e29b-41d4-a716-446655440000",
  "clusterName": "my-cluster-1",
  "scenarioName": "pod-scenarios",
  "status": "Running",
  "podName": "krkn-job-550e8400-e29b-41d4-a716-446655440000",
  "startTime": "2026-01-13T16:00:00Z"
}
```

### /scenarios/run/{jobId}/logs
- this method streams the stdout/stderr logs of a running or completed job
- the method accepts the job ID as a path parameter
- the method must:
  - find the pod associated with the job ID
  - return 404 if job not found
  - stream pod logs in real-time using chunked transfer encoding
  - support the `follow` query parameter (default: false)
    - if `follow=true`: stream logs continuously until pod terminates
    - if `follow=false`: return current logs and close connection
  - support the `tailLines` query parameter to limit output to last N lines (optional)
  - support the `timestamps` query parameter to include timestamps (default: false)
  - set appropriate HTTP headers for streaming:
    - `Content-Type: text/plain`
    - `Transfer-Encoding: chunked`
    - `Cache-Control: no-cache`

### Request example:
```bash
GET /scenarios/run/550e8400-e29b-41d4-a716-446655440000/logs?follow=true&timestamps=true&tailLines=100
```

### Response:
Streamed plain text output directly from pod stdout/stderr

### /scenarios/run
- this method returns a list of all scenario jobs
- the method must:
  - list all pods with label `app=krkn-scenario`
  - extract job information from pod labels and status
  - return array of job status objects
  - support optional query parameters for filtering:
    - `status`: filter by job status (Pending, Running, Succeeded, Failed, Stopped)
    - `scenarioName`: filter by scenario name
    - `targetId`: filter by target ID
    - `clusterName`: filter by cluster name
- the method returns an array of job objects with same structure as /scenarios/run/{jobId}

### Response example:
```json
{
  "jobs": [
    {
      "jobId": "550e8400-e29b-41d4-a716-446655440000",
      "targetId": "550e8400-e29b-41d4-a716-446655440000",
      "clusterName": "my-cluster-1",
      "scenarioName": "pod-scenarios",
      "status": "Running",
      "podName": "krkn-job-550e8400-e29b-41d4-a716-446655440000",
      "startTime": "2026-01-13T16:00:00Z"
    },
    {
      "jobId": "660e8400-e29b-41d4-a716-446655440001",
      "targetId": "660e8400-e29b-41d4-a716-446655440001",
      "clusterName": "my-cluster-2",
      "scenarioName": "node-scenarios",
      "status": "Succeeded",
      "podName": "krkn-job-660e8400-e29b-41d4-a716-446655440001",
      "startTime": "2026-01-13T15:00:00Z",
      "completionTime": "2026-01-13T15:30:00Z"
    }
  ]
}
```

## DELETE methods

### /scenarios/run/{jobId}
- this method stops and deletes a running job
- the method accepts the job ID as a path parameter
- the method must:
  - find the pod associated with the job ID
  - return 404 if job not found
  - delete the Kubernetes pod with graceful termination (5 second grace period)
  - delete associated ConfigMaps (kubeconfig + any user-provided files)
  - return 200 status on successful deletion
  - return the final job status with status: "Stopped"

### Response example:
```json
{
  "jobId": "550e8400-e29b-41d4-a716-446655440000",
  "status": "Stopped",
  "message": "Job stopped and deleted successfully"
}
```

## Implementation details

- All job management must be done in the Go operator (not data-provider)
- Pod tracking via Kubernetes labels (no CRD required)
- Kubeconfig retrieval uses same logic as GET /nodes endpoint:
  - Fetch secret named `targetId` from operator namespace
  - Parse `secret.Data["managed-clusters"]` as JSON
  - Extract kubeconfig from path: `["krkn-operator-acm"][clusterName]["kubeconfig"]`
  - The kubeconfig is already base64 encoded in the secret, must be decoded before mounting
- Kubeconfig ConfigMap naming convention: `krkn-job-<jobId>-kubeconfig`
- User files ConfigMap naming convention: `krkn-job-<jobId>-file-<sanitized-filename>`
- All ConfigMaps must have label `krkn-job-id: <jobId>` for cleanup
- Private registry authentication via ImagePullSecrets (if credentials provided)
- File mounts via ConfigMaps created dynamically
- Pod restartPolicy must be "Never" (one-time execution)
- Logs streaming must use Kubernetes pod logs API with io.Copy for efficiency
- Default kubeconfig mount path: `/home/krkn/.kube/config` (overridable via `kubeconfigPath` in request)
- RBAC must include permissions for:
  - pods: get, list, watch, create, delete
  - pods/log: get
  - configmaps: get, create, delete
  - secrets: get (to read kubeconfig from KrknTargetRequest secret)
  - secrets: create (for imagePullSecrets if private registry used)

## Status mapping from Pod Phase

- Pod phase "Pending" → Job status "Pending"
- Pod phase "Running" → Job status "Running"
- Pod phase "Succeeded" → Job status "Succeeded"
- Pod phase "Failed" → Job status "Failed"
- Pod deleted manually → Job status "Stopped" (or not found)

# Grpc python service requirement

## get nodes
- in the krkn-operator-data-provider I want to create a grpc python service that must interact with the go api
- the service must have `krkn-lib` pypi module along with the others (for the moment reference the git branch https://github.com/krkn-chaos/krkn-lib.git@init_from_string when a new version will be released we'll switch to the pypi version) 
- the get nodes will receive as a grpc parameter the kubeconfig in base64 format
- the get nodes will initialize the KrknKubernetes python object with the `kubeconfig_string` named parameter containing the decoded kubeconfig
- the get nodes will call the list_nodes() method on the KrknKubernetes object and that will return a list of strings
- the list of strings will be returned to the Go API
- the list of strings will be returned to the client
- remove the kubeconfig return parameter from the go api


## get pods
- in the krkn-operator-data-provider I want to add a method to list all the pods and 