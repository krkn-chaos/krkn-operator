# krkn-operator

![test](https://github.com/krkn-chaos/krkn-operator/actions/workflows/test.yml/badge.svg)
![pr-checks](https://github.com/krkn-chaos/krkn-operator/actions/workflows/pr-checks.yml/badge.svg)
![coverage](https://krkn-chaos.github.io/krkn-lib-docs/coverage_badge_krkn-operator.svg)

Kubernetes operator for chaos engineering built on the [krkn](https://github.com/krkn-chaos/krkn) framework. Orchestrates chaos scenarios across Kubernetes clusters through custom resource definitions (CRDs) and provides a REST API for programmatic access.

## Documentation

📖 **[Official Documentation](https://krkn-chaos.dev/docs/krkn-operator)**

## Quick Start

**Install:**
```bash
helm install krkn-operator oci://quay.io/krkn-chaos/charts/krkn-operator --version <version> \
  -n krkn-operator-system --create-namespace
```

**Upgrade:**
```bash
helm upgrade krkn-operator oci://quay.io/krkn-chaos/charts/krkn-operator --version <version> \
  -n krkn-operator-system
```

Helm installs CRDs from the chart on a fresh install but does not upgrade files
under `crds/`. During `helm upgrade`, the chart runs a pre-upgrade job that adds
missing CRDs and updates changed schemas from the operator image.

### OLM / OperatorHub bundles

Release automation publishes separate bundle images for generic Kubernetes and
OpenShift:

- `quay.io/krkn-chaos/krkn-operator-bundle:<version>` — Kubernetes bundle;
- `quay.io/krkn-chaos/krkn-operator-bundle-ocp:<version>` — OpenShift bundle.

The `1.0.x` bundles use the `stable-kubernetes` and `stable-ocp` channels
respectively. Later release lines use versioned channels, such as
`stable-kubernetes-1.1` and `stable-ocp-1.1`, so maintenance releases on one
line do not replace bundles from another line. A prerelease does not change the
package default channel; the first stable release of a newer line promotes its
versioned channel as the default, while maintenance releases on older lines do
not change it back.
The OLM bundle declares Kubernetes `1.19.0` as its minimum version, matching the
published compatibility matrix.
For a disposable cluster with OLM installed, a published bundle can be tested
with:

```bash
operator-sdk run bundle \
  quay.io/krkn-chaos/krkn-operator-bundle:<version>
```

On OpenShift, use the `-ocp` repository. Route, Ingress, and Gateway resources
are intentionally not created by the bundle; expose the console using the
cluster administrator's preferred TLS and networking configuration.

When a release tag is pushed, the release workflow also prepares the OpenShift
Community Operators catalog submission and opens the catalog pull request. It
requires the `COMMUNITY_OPERATORS_FORK` repository variable (for example,
`<team-or-user>/community-operators-prod`) and a
`COMMUNITY_OPERATORS_TOKEN` secret with permission to push to that fork and
open pull requests against `redhat-openshift-ecosystem/community-operators-prod`.
OperatorHub remains responsible for validating and merging the catalog pull
request.

The same release workflow also prepares a Kubernetes bundle and opens a PR
against `k8s-operatorhub/community-operators` when the
`KUBERNETES_OPERATORS_FORK` repository variable and
`KUBERNETES_OPERATORS_TOKEN` secret are configured.

📖 For configuration, usage, compatibility, and advanced installation options, see the official documentation.📖 For configuration, usage, compatibility, and advanced installation options, see the **[official documentation](https://krkn-chaos.gateway.scarf.sh/krkn-operator/docs?source=github)**.

**Uninstall:**
```bash
helm uninstall krkn-operator -n krkn-operator-system
```

CRDs remain after uninstall to protect custom resources. After removing any
custom resources you still need, remove the operator CRDs with:

```bash
kubectl delete crds -l app.kubernetes.io/name=krkn-operator
```

Upgrading an existing release applies this label to its CRDs so the selector
also matches CRDs created by earlier releases.

## Resiliency History API

The history API returns calculated resiliency scores for runs labeled with the
selected categories. Both endpoints require a bearer token. Users must be able
to view each category and each scored cluster; inaccessible cluster scores are
omitted.

### Query several categories and clusters

`POST /api/v2/resiliency-history` accepts non-empty `categories` and `clusters`
arrays. If cluster names are shared by multiple providers, optionally pass
`clusterProviders` to select the providers for each name. Omitting that field
includes every provider for each selected cluster name.

```bash
curl -sS -X POST https://<operator-host>/api/v2/resiliency-history \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
        "categories": ["resilience"],
        "clusters": ["cluster-a"],
        "clusterProviders": {"cluster-a": ["krkn-operator"]}
      }'
```

The response nests points under `clusters[clusterName][categoryName]` and
configuration groups under `configurationGroups[categoryName][groupId]`.
Each point includes `providerName`, `runId`, `runType`, `date`, `score`, and
`configurationGroupId`; the group entry describes the equivalent run
configuration. `baseline` is present when configured.

```json
{
  "clusters": {
    "cluster-a": {
      "resilience": [
        {
          "date": "2026-09-29T10:00:00Z",
          "providerName": "krkn-operator",
          "runId": "scenario-run-123",
          "runType": "scenario-runs",
          "score": 91.5,
          "configurationGroupId": "scenario-runs/scenario-run-123"
        }
      ]
    }
  },
  "configurationGroups": {
    "resilience": {
      "scenario-runs/scenario-run-123": {
        "runType": "scenario-runs",
        "representativeRunId": "scenario-run-123",
        "parameterProfileFingerprint": "…",
        "parameterProfileName": "default"
      }
    }
  }
}
```

### Query one category

`GET /api/v2/categories/{category}/resiliency-history` returns the same score
points for one category, grouped as `clusters[clusterName]`; each point's
`providerName` distinguishes same-named clusters owned by different providers.

Invalid filters return `400`, missing authentication returns `401`, category
access failures return `403`, a missing category returns `404`, and unexpected
server errors return `500`. See the
[official API documentation](https://krkn-chaos.dev/docs/krkn-operator) for
the full operator API reference.

## Telemetry Query API

Query chaos-run telemetry stored in Elasticsearch/OpenSearch through the operator's REST API. The request supplies the connection in one of two mutually exclusive ways:

- **Saved config (recommended):** the request references a previously saved Elasticsearch config by name (`configName`), and the operator resolves the credentials server-side from the backing Kubernetes Secret. No credentials are sent by the client.
- **Inline connection:** the request supplies the connection details, including `username` and `password`, directly in the request body (`inline`). These credentials are used only to service that single request and are **never persisted** — no Secret is created or updated. Inline destinations are subject to the destination policy (loopback, private, and metadata addresses are rejected) to guard against server-side request forgery, and always use default TLS verification against the system trust store (custom CA certificates and insecure-skip-TLS remain admin-only, saved-config settings).

In both modes credentials never leave the backend beyond the connection to the target cluster, and they are never returned in any API response.

**Endpoint:** `POST /api/v1/elasticsearch-query`

**Authentication:** Required. Send a valid bearer token: `Authorization: Bearer <token>`. Available to any authenticated user (no admin role required). Unauthenticated requests receive `401`.

**Prerequisite:** A saved Elasticsearch config must already exist (created via `POST /api/v1/elasticsearch-configs`, admin only) and must have a telemetry index configured. The query targets that config's telemetry index.

**Request body:**

| Field        | Type   | Required | Description |
|--------------|--------|----------|-------------|
| `configName` | string | yes      | Name of the saved Elasticsearch config to query. |
| `size`       | int    | no       | Max documents per page. Defaults to `50`; clamped to `500`. Negative values are rejected. |
| `page`       | int    | no       | 1-based page number. Values below `1` default to `1`. The offset is `(page-1)*size`; a page whose offset plus size exceeds the `10000` result window is rejected. |
| `startDate`  | string | no       | Inclusive lower bound on the run timestamp, `yyyy-MM-dd`. Defaults to 30 days ago. |
| `endDate`    | string | no       | Upper bound on the run timestamp, `yyyy-MM-dd`. Includes only the selected calendar day. Defaults to the current instant. |
| `filters`    | object | no       | Map of facet category to selected values. Keys must be known facet categories (`scenario_type`, `job_status`, `cloud_infrastructure`, `cloud_type`, `major_version`, `network_plugins`); unknown keys are rejected. Values within a category are OR-ed; categories are AND-ed. `job_status` values must be `"true"` or `"false"`. |

Results are sorted newest-first by timestamp, then the `page`/`size` window is applied.

**Response `200` shape:**

```json
{
  "documents": [
    {
      "run_uuid": "…",
      "scenario_type": "pod_disruption_scenarios",
      "start_timestamp": 1735689600,
      "end_timestamp": 1735689900,
      "namespace": "openshift-kube-apiserver",
      "status": true,
      "metadata": {
        "kubernetes_objects_count": {"Pod": 701, "ConfigMap": 1064},
        "network_plugins": ["OVNKubernetes"],
        "total_node_count": 9,
        "cloud_infrastructure": "AWS",
        "cloud_type": "self-managed",
        "cluster_version": "4.19.0",
        "major_version": "4.19",
        "build_url": "https://example/1",
        "fips_enabled": false,
        "tag": "cr",
        "etcd_encryption_enabled": false,
        "ipsec_enabled": false,
        "node_summary_infos": [
          {
            "count": 3,
            "nodes_type": "master",
            "architecture": "amd64",
            "instance_type": "m5.2xlarge",
            "kernel_version": "5.14.0-570.51.1.el9_6.x86_64",
            "kubelet_version": "v1.33.5",
            "os_version": "Red Hat Enterprise Linux CoreOS 9.6.20250930-0 (Plow)"
          }
        ]
      },
      "scenarios": [
        {
          "scenario_type": "pod_disruption_scenarios",
          "start_timestamp": 1735689600,
          "end_timestamp": 1735689900,
          "exit_status": 0,
          "parameters": {"kill_count": 1},
          "affected_pods": [
            {
              "pod_name": "kube-apiserver-master-0",
              "namespace": "openshift-kube-apiserver",
              "total_recovery_time": 45.2,
              "pod_readiness_time": 42.1,
              "pod_rescheduling_time": 3.1
            }
          ]
        }
      ]
    }
  ],
  "total": 50,
  "stats": {
    "pass": 42,
    "fail": 8,
    "pass_percent": 84.0
  },
  "facets": {
    "job_status": [
      {"value": "true", "count": 42},
      {"value": "false", "count": 8}
    ],
    "cloud_infrastructure": [
      {"value": "AWS", "count": 30},
      {"value": "GCP", "count": 20}
    ]
  }
}
```

Each document includes run-level cluster/infrastructure metadata (`metadata`) and complete scenario details (`scenarios`). Optional fields are omitted when absent in source data.

**Document fields:**

| Field | Type | Description |
|-------|------|-------------|
| `run_uuid` | string | Unique identifier for this telemetry run. |
| `scenario_type` | string | Scenario type from run's first scenario (for table view compatibility). |
| `start_timestamp` | int64 | Start time from run's first scenario (Unix seconds, UTC). |
| `end_timestamp` | int64 | End time from run's first scenario (Unix seconds, UTC). |
| `namespace` | string | Namespace from run's first scenario. |
| `status` | bool | Run-level pass/fail status. |
| `metadata` | object | Run-level cluster/infrastructure details: object counts, network plugins, node summaries, cloud type, versions, security settings. Omitted when source document had no metadata. |
| `scenarios` | array | All scenarios executed in run. Each includes type, timestamps, exit status, raw parameters, and optional affected pod recovery timings. |

`total` is the count of documents matching the query across the whole matched window (all documents in range), not just the returned `page`. The client uses it to compute the page count; it equals `stats.pass` + `stats.fail` only when every matching document has `job_status` set (documents missing it count toward `total` but neither `pass` nor `fail`).

`stats` summarizes run-level pass/fail across the entire matched time window (all documents in range), not just the returned `size`-capped page. Fields:

| Field          | Type   | Description |
|----------------|--------|-------------|
| `pass`         | int    | Runs with `job_status` true in the matched window. |
| `fail`         | int    | Runs with `job_status` false in the matched window. |
| `pass_percent` | float  | `pass` / (`pass` + `fail`) as a percentage, `0`-`100`, rounded to 2 decimals; `0` when no runs matched. |

`facets` maps each filter category to the available values in the matched window (from a per-category terms aggregation), each with its document count. The UI populates the value multi-select from it. Because filters are applied in the query, facet counts narrow as filters are selected.

| Field   | Type   | Description |
|---------|--------|-------------|
| `value` | string | A selectable facet value. |
| `count` | int    | Documents in the matched window carrying that value. |

Each string category returns up to 100 values; `job_status` returns its two. When a category has more distinct values than were returned, its `facets` list is a prefix and the category is flagged in `facetsTruncated` (a map of category to `true`); the UI should fetch the remaining values rather than treat the dropdown as complete. `facetsTruncated` is omitted when every category's values fit.

**Errors:**

| Status | Meaning |
|--------|---------|
| `400`  | Invalid request body or validation failure: missing `configName`, negative `size`, a `page` whose offset exceeds the result window, an unknown `filters` category, an invalid `job_status` value, a malformed date, `startDate` after `endDate`, or an `endDate` in the future. |
| `401`  | Missing or invalid authentication token. |
| `404`  | The named Elasticsearch config does not exist. |
| `502`  | The upstream Elasticsearch cluster could not be reached or returned an error. The response carries a stable, sanitized message; detailed upstream diagnostics are logged server-side only. |

**Example request:**

```bash
curl -sS -X POST https://<operator-host>/api/v1/elasticsearch-query \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
        "configName": "prod-es",
        "size": 25,
        "startDate": "2026-08-01",
        "endDate": "2026-08-26"
      }'
```

## Alerts Query API

Query alert documents from the `alertsIndex` configured on a saved Elasticsearch config. The operator resolves the config's credentials and alerts index server-side; alert sources are redacted before they are returned.

**Endpoint:** `POST /api/v1/elasticsearch-alerts-query`

**Authentication:** Required. Send `Authorization: Bearer <token>`. The named config must be public, accessible through the user's group, or queried by an administrator.

**Request body:**

```json
{
  "configName": "prod-es",
  "size": 50,
  "startDate": "2026-08-01",
  "endDate": "2026-08-26"
}
```

`size` defaults to `50` and is capped at `500`. Date bounds use `yyyy-MM-dd` and filter the alert index's `created_at` field. Results are sorted newest-first by `created_at`.

If the selected saved config does not define an `alertsIndex`, the endpoint returns `400` with `error: "configuration_error"`; no Elasticsearch request is made. Elasticsearch connectivity or query failures return `502` with `error: "upstream_error"`.

**Example:**

```bash
curl -sS -X POST https://<operator-host>/api/v1/elasticsearch-alerts-query \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"configName":"prod-es","size":50,"startDate":"2026-08-01","endDate":"2026-08-26"}'
```

The complete request and response schema is available through the generated Swagger documentation at `/api/swagger/index.html`.

## Ecosystem

See [DEPLOYMENT.md](DEPLOYMENT.md) for full installation options and configuration.
## Running Locally

The operator requires three components running concurrently: the Python gRPC data provider, the Go operator, and (optionally) the React console.

### Prerequisites

- Go 1.21+
- Python 3.11+
- `kubectl` connected to a Kubernetes cluster
- `helm` (used by `start_operator.sh` to render RBAC resources from the Helm chart)
- Node.js 18+ (only if running the console)

### 1. Start the gRPC data provider

```bash
cd krkn-operator-data-provider

# First-time setup
python3.11 -m venv venv-dev
source venv-dev/bin/activate
pip install --upgrade pip
pip install grpcio>=1.60.0 grpcio-tools>=1.60.0
pip install git+https://github.com/krkn-chaos/krkn-lib.git@init_from_string

# Start the server (listens on :50051)
python server.py
```

Keep this terminal open. See [krkn-operator-data-provider/RUN_LOCALLY.md](krkn-operator-data-provider/RUN_LOCALLY.md) for more detail.

### 2. Run the operator

In a new terminal:

```bash
cd krkn-operator

export GRPC_SERVER_ADDR=localhost:50051
make run
```

The REST API is available at `http://localhost:8080`.


Alternatively, use the helper script which also installs CRDs, service account and builds the binary:

```bash
./start_operator.sh
```

This script automatically:
- Checks cluster connectivity
- Creates or uses the target namespace (defaults to `krkn-operator-system`, override with `KRKN_NAMESPACE=my-ns`)
- Installs CRDs
- Provisions a `ServiceAccount` and least-privilege `ClusterRole` for scenario execution (pod chaos, node operations, discovery)
- On OpenShift, grants the `anyuid` Security Context Constraint (required for UID 1001)
- Builds and starts the operator

**Requirements:** Your kubeconfig user must have permission to create RBAC bindings and CRDs.

### 3. Create an admin user and log in

```bash
# Register the first user (must use role "admin")
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"userId":"admin@local.dev","password":"Admin1234!","name":"Admin","surname":"User","role":"admin"}'

# Log in
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"userId":"admin@local.dev","password":"Admin1234!"}'

export TOKEN="<paste-token-here>"
```

### 4. (Optional) Run the web console

See the [krkn-operator-console README](https://github.com/krkn-chaos/krkn-operator-console#running-locally) for setup. The console proxies `/api` to `http://localhost:8080` automatically.

### Architecture overview

```
krkn-operator-data-provider  (Python, :50051)
  ↑ gRPC
krkn-operator                (Go,     :8080 REST API)
  ↑ HTTP
krkn-operator-console        (React,  :3000)
```

For terminal API details and troubleshooting see [QUICKSTART_TERMINAL_API.md](QUICKSTART_TERMINAL_API.md).

## API compatibility notes

### Retry limits

Scenario and graph creation requests accept an optional `maxRetries` integer. It
is the number of retries after the initial attempt and defaults to `3` when the
field is omitted. Set it to `0` to disable retries. Negative values are invalid
and return HTTP `400` with an error code of `bad_request`.

For scenario runs, `POST /api/v1/scenarios/run` returns the applied limit as
`maxRetries`. For graph runs, `POST /api/v1/graphruns` returns it in the run
specification as `spec.maxRetries`; job status responses also expose the limit
as `maxRetries` alongside `retryCount`.

The replay endpoints preserve the configured retry value, including `0`:

- `GET /api/v1/scenarios/run/{scenarioRunName}/config`
- `GET /api/v1/graphruns/{graphRunName}/config`

Example scenario request:

```json
{
  "targetRequestId": "target-request-id",
  "targetClusters": {"provider": ["cluster"]},
  "scenario": {"name": "pod-delete", "private": false},
  "maxRetries": 0
}
```

Scenario run requests identify the scenario rather than supplying an executable
image. The operator resolves the image from the scenario name and selected
registry:

```json
{
  "targetRequestId": "target-request-id",
  "targetClusters": {"provider": ["cluster"]},
  "scenario": {
    "name": "pod-delete",
    "private": false
  }
}
```

For a saved private registry, set `private` to `true` and include its
`registryName`. Direct image references are not accepted.

### Graph node resiliency weights

Graph runs accept an optional `resiliencyWeight` on each node in the `graph`
object. It is a positive multiplier for that scenario's contribution to the
cluster resiliency score; omitted or legacy nodes use `1`.

```json
{
  "graph": {
    "pod-delete": {
      "scenario": {"name": "pod-delete", "private": false},
      "resiliencyWeight": 2.0
    },
    "pod-disruption": {
      "scenario": {"name": "pod-disruption", "private": false},
      "depends_on": "pod-delete"
    }
  }
}
```

When resiliency scoring is enabled, the final score for a cluster is the
weighted average of the completed node scores:
`sum(node score * resiliencyWeight) / sum(resiliencyWeight)`.

### Log streaming

The WebSocket log endpoint
(`/api/v1/scenarios/run/{name}/jobs/{jobID}/logs`) filters out base64-encoded
report payloads (HTML and PDF) that are embedded in pod output between
`===KRKN_REPORT_*_START===` / `===KRKN_REPORT_*_END===` markers. Clients
receive only human-readable log lines. The operator controller still reads
the full unfiltered logs to extract and store reports.

Authenticated users can read the image-signature verification setting at
`GET /api/v1/operator/signature-verification`. Administrators can update it
with `PATCH` and a required boolean body, for example
`{"enabled":false}`. Image verification remains observable when enforcement
is disabled; only the enforcement result is ignored.

## Backup and Restore

Admin users can back up and restore operator configuration (users, groups, targets, providers, credentials) for disaster recovery and cross-cluster migration using the web console.

**To download a backup:** Click "Download Backup" in the Backup & Restore card.

**To restore from a backup:** Click "Upload & Restore" in the Backup & Restore card, select your backup archive, and confirm. The console will monitor restore progress in real time.

**Important:** Restore runs asynchronously. Poll the restore job until it completes, then refresh the page if the restored settings are not visible.

**Limits:**
- Maximum upload size: 100 MB
- Maximum active restore: 1
- Maximum tracked job records: 1000
- Backup history: Not persisted — status is lost on operator restart

## License

Copyright 2025 krkn-chaos

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
