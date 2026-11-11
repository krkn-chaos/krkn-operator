# Scenario workspaces on a shared PVC

The environment SRE supplies a PVC in the operator namespace. It must be
`Bound`, use `Filesystem`, request `ReadWriteMany`, and allow scenario pods
running as UID/GID `1001:1001` to read and write in newly created directories
directly under the PVC root. The operator does not create,
delete, or replace the claim. Storage provisioning and backend configuration
belong to the environment; Krkn does not require a particular storage vendor,
NFS implementation, or cluster. An administrator publishes the PVC as a
workspace, and scenario runs select it through the `workspace` field.

## Publish and select a workspace

An authenticated administrator first lists eligible PVCs with
`GET /api/v1/workspaces/pvcs/available`. This admin-only endpoint returns PVCs
in the operator namespace that are `Bound`, `Filesystem`, and request
`ReadWriteMany`. The console presents these claims in a selector; it does not
ask the administrator to type a claim name. Selecting a claim, assigning a
workspace name and visibility, and publishing it creates the `KrknWorkspace`
CR. The API revalidates the selected claim when the admin calls
`PUT /api/v1/workspaces/{name}`. For example:

```json
{"claimName": "krkn-shared-file", "availableToAll": true}
```

For group access, use:

```json
{"claimName": "krkn-shared-file", "groups": ["team-a"]}
```

With no visibility field, only admins can select the workspace. The API creates
a `KrknWorkspace` CR in the operator namespace and pins the PVC UID. It does not
label or alter the PVC. The workspace may have at most one group and cannot be
both public and group-scoped. `DELETE` on the same endpoint deletes the wrapper
CR without deleting the PVC or its data. Administrators see all wrappers and
their backing PVC names with `GET /api/v1/workspaces`; other authenticated users
see only available workspaces they may select with
`GET /api/v1/workspaces/available`. The user-facing list omits PVC names.
The workspace name is the Kubernetes resource name and is unique within the
operator namespace. An existing workspace can change its visibility; binding it to a
different PVC requires deleting and recreating the wrapper, which changes its
UID and invalidates old run bindings.
After upgrading from a version with `spec.displayName`, clients should use the
resource name shown in the catalog. The old display value is ignored, and the
publication API rejects a `displayName` request field.

The workspace reconciler watches the wrapper and its referenced PVC, and
rechecks the claim at least every five minutes. Its
`Available` condition reports whether that exact PVC is present, `Bound`,
`Filesystem`, and RWX. This condition does not prove runtime write permissions.
It may lag a PVC change; the API and scenario controller verify the PVC again
before creating a run or Pod.

A standalone scenario request may include:

```json
{"workspace": {"name": "shared-rollback-files"}}
```

Graph scenario nodes accept the same `workspace` field. The server checks
visibility and saves the wrapper UID plus PVC name and UID on the run. Scenario
pods mount their assigned
subdirectory at `/var/lib/krkn/workspace`. The subdirectory is one path
component directly under the PVC root, based on the ScenarioRun UID, a hash of
the target provider and cluster, and the execution attempt ID. Each
retry therefore receives a different directory. The scenario container mounts
only that directory through `subPath`; it does not mount the PVC root. The
directory split prevents accidental path collisions; it is not a permission
boundary. Kubernetes creates a missing `subPath` directory during Pod startup;
the Pod already specifies `runAsUser`, `runAsGroup`, and `fsGroup` as `1001`.
The operator does not start a preparatory container or change storage
ownership. The environment must ensure that the created directory is writable
by the scenario container.

## Storage validation

Writing to the PVC root is only a preliminary check. On each worker that may
host scenario pods, verify that a container running as `1001:1001` can write
inside a newly created one-level `subPath`. Kubernetes may create this directory
before the container starts, and its owner or inherited permissions can differ
from the PVC root. On NFS storage where kubelet creates directories as a
different UID, making the PVC root owned by group `1001` with mode `2775`
can pass group `1001` to the new directory. Verify this on the actual backend;
the PVC root's own write test does not prove inheritance. When rollback is
enabled, the scenario Pod tests writing in its
assigned workspace before starting Krkn and fails with
`rollback workspace is not writable` if that test fails. The storage
administrator must arrange permissions so new directories are
writable by `1001:1001` on every eligible worker.

In the IBM OpenShift test environment, `krkn-operator-system/krkn-shared-file`
is a 10 GB IBM File Storage for VPC share exposed by a static NFS PV with
`nfsvers=4.1,sec=sys`, because the available StorageClasses were block-only.
The share initially reported `initial_owner=0:0`; mounting succeeded but
writing as `1001:1001` failed. After the SRE changed the share root ownership
to `1001:1001` on the worker's NFS mount, a Pod running as `1001:1001` printed
`RWX_OK` when writing to the PVC root. IBM still reported `initial_owner=0:0`.
The later scenario test exposed a second storage requirement: kubelet-created
`runs/...` directories on this share were owned by `99:99` with mode `0775`.
The scenario container running as `1001:1001` could not write there. Changing
the PVC root owner alone did not make nested `subPath` directories writable.
An NFS test with a setgid parent (`1001:1001`, mode `2775`) produced one
kubelet-created child with `99:1001` and mode `0775`; a Pod running as `1001`
wrote successfully. A second nested level lost the inherited group, so the
operator now uses one directory level under the root. The SRE set the IBM PVC
root to mode `2775`; new scenario runs still need an end-to-end write test on
every eligible worker. The operator does not create a preparatory container.

## Replay and unavailable storage

Replay payloads retain the original workspace and PVC identities, but a replay
creates a new run and a new subdirectory. If the wrapper or claim is missing,
unavailable, or has been recreated with the same name and a different UID, the
request is rejected before a new run is created. The controller repeats the
identity and PVC checks before creating each scenario pod. An administrator
must explicitly select a new workspace to change the replay's storage dependency.
Replay and configuration APIs omit the cluster PVC name from their responses.
For nonadmins, they include the pinned UIDs only when the caller may still
select that workspace; otherwise they omit the workspace reference. Admins
retain the pinned UIDs even if the claim is missing. The server resolves the
PVC name again from the pinned workspace identity.

The workspace directory persists independently of the run resource. Storage
permissions, retention, backup, and cleanup remain the administrator's
responsibility. Publication controls API selection.

## Reuse from another operator

`github.com/krkn-chaos/krkn-operator/pkg/workspace` is the public Go API for
operators that create Pods in the operator namespace, alongside published
`KrknWorkspace` objects and their backing PVCs. A consumer such as krknAI can use the same
workspace alias and reconciler status without implementing its own PVC lookup
or mount construction. It must import a released version of the operator Go
module; a neighboring source checkout is not used automatically. The CRD and
workspace reconciler must be installed in the cluster. The library does not
install them or create PVCs.

The consumer passes **its Pod namespace** to `Resolve`. In the current deployment
this is the operator namespace shared by the consumer. The function reads the
`KrknWorkspace` and PVC in that namespace and returns a `Binding` containing
both pinned UIDs. The caller is responsible for passing the same namespace
when it creates the Pod. The `Binding.Reference` can be stored in the
consumer's workload spec or another durable record and reconstructed on retry.
`Validate` checks the stored reference against the live wrapper and PVC before
each Pod creation. If the PVC disappears, becomes unsuitable, or is recreated
under the same name with a different UID, the consumer should stop scheduling
that workload and surface the error. It must not silently switch to another
claim. A fresh selection may use `Resolve` again.

```go
binding, err := workspace.Resolve(ctx, kubeClient, podNamespace, workspaceName)
if err != nil {
    return err
}
// Persist binding.Reference with the workload before scheduling it.

// On each attempt, reconstruct Binding from the persisted reference.
if err := workspace.Validate(ctx, kubeClient, binding); err != nil {
    return err
}
directory, err := workspace.SubPathFor(workspace.WorkloadKey{
    Consumer: "krkn-ai", WorkloadUID: workloadUID,
    Target: clusterIdentity, Attempt: attemptID,
})
if err != nil {
    return err
}
volume, mount, err := workspace.Mount(binding, workspace.MountOptions{
    VolumeName: "workspace", MountPath: "/var/lib/krkn-ai/workspace",
    SubPath: directory,
})
if err != nil {
    return err
}
pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, mount)
```

The example assumes `ctx`, `kubeClient`, `podNamespace`, `workspaceName`,
`workloadUID`, `clusterIdentity`, `attemptID`, and `pod` are supplied by the
consumer. `SubPathFor` makes one stable directory directly under the PVC root
from the consumer name, workload UID, target, and attempt. Reusing the same key
finds the same directory; changing any field selects a separate directory.
The path contains a hash instead of raw caller values. Consumers must persist
or reconstruct the **same key** when reading files later. They may deliberately
share a directory by using the same key or mount a known directory read-only;
directory separation itself provides no permission isolation. The existing
Krkn scenario and rollback flows keep using `SubPath`, whose directory naming
is unchanged so previous artifacts remain discoverable.

`Mount` returns only the PVC volume and its matching container mount. The
consumer adds them to its Pod, chooses the mount path and volume name, and
sets its Pod security context. `ReadOnly: true` applies to both the volume and
mount. The storage administrator must ensure that the consumer's UID/GID can
access kubelet-created directories on every eligible worker; the library does
not change ownership or prepare the filesystem.

`Available` reads the wrapper's reconciled status for display or initial
selection. It is an observation that can lag a PVC change and cannot establish
filesystem write access. `Resolve` requires a current `Available` condition
and rechecks the PVC. `Validate` checks the live identities and PVC even when
the reconciler has not yet updated status. Both return errors comparable with
`errors.Is(err, workspace.ErrUnavailable)` or
`errors.Is(err, workspace.ErrReplaced)`; API read failures retain their original
cause. `Mount` rejects incomplete bindings and invalid mount options, but does
not query Kubernetes, so call `Validate` first.

Public and group visibility govern **platform API selection**. The operator's
REST API checks the requesting user's access before calling `Resolve`. The Go
functions use the caller's Kubernetes credentials and do not authenticate an
end user. A consumer exposing workspace selection to its own users must apply
its own authorization policy before resolution. Its service account needs
read access to the wrapper and PVC in the supplied namespace, plus the normal
permissions to create its Pods there.

## Rollback artifact persistence

Scenario requests and graph nodes may set `rollbackEnabled: true`. This requires
a workspace; selecting a workspace without enabling rollback remains valid.
Rollback-enabled runs use zero orchestrator retries. The REST API persists
`maxRetries: 0` even when the request supplies a positive value, and the
controller enforces the same limit for runs created directly as CRs. For a
graph, only nodes with rollback enabled use zero retries; other nodes retain
the graph's configured retry count. Previously created rollback-enabled runs
also stop retrying when the updated controller reconciles them.
The operator mounts the per-attempt workspace at `/var/lib/krkn/workspace` and
sets `ROLLBACK_VERSIONS_DIRECTORY` to that exact path in the Krkn container,
overriding any caller-provided value when rollback is enabled. Krkn Hub images
must render this environment variable into `kraken.rollback_versions_directory`
in their generated config. For public images, the operator checks the image's
`config.yaml.template` before starting Krkn and fails the Pod if that mapping
is absent. An image with the field missing would otherwise make Krkn fall back
to container-local `/home/krkn/.krkn/rollback` even though the PVC is mounted.
The published CPU Hog digest observed in the IBM test lacked this field and
needs to be rebuilt and published from the updated Krkn Hub source. Private
image authors remain responsible for honoring the environment variable.
That image's base `config/config.yaml` contains an empty
`rollback_versions_directory` field, but CPU Hog's `run.sh` executes Krkn with
the generated `config/cpu_config.yaml`; the latter omitted the field in the
published digest even when the Pod environment variable was populated.
The operator also sets `UUID` to the attempt JobID, overriding a caller value,
so Krkn's rollback context can be identified later without guessing.
When rollback is disabled, the operator does not override the image's default
rollback directory. The flag persists in the ScenarioRun spec and replay payload.
It controls durable rollback artifacts, not automatic rollback execution.

Krkn writes rollback files inside the workspace assigned to the specific
ScenarioRun, target cluster, and execution attempt. A successful scenario removes
its rollback files. A scenario that stops or fails may leave them; some scenarios
do not implement rollback and never create them. Therefore the presence of
rollback files in that exact workspace is the source of truth for whether the
platform can offer rollback. Run status alone is insufficient.

`GET /api/v1/scenarios/run/{runName}/rollback/{jobID}` checks one attempt.
It validates the pinned workspace and PVC, then starts a short-lived inspection
Pod using the original scenario image with a read-only mount of that attempt's
`subPath`. The Pod checks for unexecuted Krkn rollback version files; it does
not execute them. It uses the scenario runner ServiceAccount for the same
UID/GID `1001:1001` storage access as scenario Pods, without mounting a service
account token. The API returns `checking`, `available`, `absent`, `unavailable`,
`error`, or a rollback execution state. Successful inspection results expire
after 30 seconds and are checked again. If the inspection Pod cannot start,
fails, or remains pending for more than two minutes, the API returns
`unavailable` with `canExecute: false` instead of recreating it on every poll.
This reports that rollback cannot currently be offered; it does not claim the
files are absent. Viewing requires cluster view permission;
the inspection Pod is created only for callers with cluster run permission.
`canExecute` is true only with cluster run permission and the original
kubeconfig still available. An active scenario retry returns `unavailable`
without starting an inspection Pod. A succeeded rollback returns `succeeded`
with the existing rollback run name, even when its files have been removed.
For private scenarios, the caller must also have access to the original
registry before inspection can create a Pod or read retained files.
The inspection Pod runs the installed operator image with the
`/manager inspect-rollback` command, not the original scenario image. It only
checks regular file names in the read-only attempt directory. This also covers
older attempts whose saved `scenarioImage` contains a mutable tag. The operator
image must be supplied as `KRKN_ROLLBACK_INSPECTOR_IMAGE` with a SHA-256 digest;
Helm and OLM release manifests pass the installed operator image to this
variable. Helm also passes `images.pullSecrets` as
`KRKN_ROLLBACK_INSPECTOR_PULL_SECRETS` for private operator registries. Local
installations using a tag must configure a digest-pinned operator image to
enable rollback inspection. Without a valid digest, GET reports `unavailable`
and POST returns a conflict without creating a probe.

`POST` to the same path requires cluster run permission and a recent `available`
result. It creates a separate `KrknScenarioRun` for the Krkn `workload-rollback`
image in the same registry as the original scenario. The controller resolves its
digest and verifies its signature through the normal image path. That run uses the
original attempt's kubeconfig ConfigMap and cluster API identity, mounts the
original attempt's directory,
and receives the attempt JobID as `UUID`. Repeated requests while a rollback is
running or succeeded return the existing run. A failed rollback can be retried
while files remain; each retry has its own run record. The UI offers the action
only when the inspection finds files and asks for confirmation before starting.
The Jobs WebSocket snapshot and run-detail response include `rollbackEnabled`
from the `KrknScenarioRun` spec and the earlier attempt IDs, so the console can
inspect rollback availability for every finished attempt.
When files are present, GET also checks that `workload-rollback` exists in the
original scenario's registry and has a trusted signature when operator-wide
signature verification is enabled. A missing image, untrusted signature,
unavailable registry, or insufficient registry access disables the action and
returns an explanatory message. Successful and definitive negative image checks
are cached for 30 seconds to limit registry traffic. POST always checks again
before creating the rollback run; the controller independently resolves and
verifies the image before creating its Pod.
Replay and run-configuration endpoints return an error if a stored workspace
reference cannot be checked, or a conflict if the caller can no longer select
it. They do not silently omit the volume from a replay payload.
If the API cannot load rollback run state, it returns HTTP 500 with
`internal_error`. If POST cannot inspect files or check the rollback image, it
returns HTTP 503 with `rollback_inspection_failed` or
`rollback_image_check_failed`. These failures do not establish that rollback
files are absent; callers can retry after the dependency recovers.

The original run status retains earlier attempt IDs and images when retrying,
so each attempt's files can be inspected independently. If the workspace or PVC
is missing or replaced, inspection reports `unavailable` and execution is
rejected. If the original attempt's kubeconfig ConfigMap is missing or has a
different owner, the files may still be detected but `canExecute` is false and
POST is rejected. Storage retention and cleanup remain administrator
responsibilities.

### Rollback history in Jobs

`GET /api/v2/jobs?groupRollbacks=true` and a Jobs WebSocket subscription with
`groupRollbacks: true` return rollback executions in the source scenario item's
`rollbackRuns` array. The default Jobs view remains flat for existing clients.
Grouping happens before pagination and statistics, so a child does not consume
a top-level page slot or count as a separate job in the grouped view. Each child
contains its source attempt JobID, rollback run name, iteration, phase, actor,
creation time, and current rollback job/pod IDs. The source name and immutable
UID must both match; deleted sources, graph-node sources, and children whose
parent is not visible to the caller remain top-level history.

The source scenario phase remains unchanged. The console derives `Rolling back`,
`Rolled back`, `Partially rolled back`, or `Rollback failed` from the latest
rollback execution for each source attempt. This is execution history, separate
from the retained-file inspection that controls whether another rollback can be
started. The indicator covers attempts with a rollback run; it does not assert
that every failed target was restored. Group members see pending rollback runs
using the pinned source cluster
identity before a rollback job appears; `view` permits inspection and `run` on
that cluster is required to launch another rollback.
