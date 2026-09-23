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

### OLM / OperatorHub bundles

Release automation publishes separate bundle images for generic Kubernetes and
OpenShift:

- `quay.io/krkn-chaos/krkn-operator-bundle:<version>` — Kubernetes bundle;
- `quay.io/krkn-chaos/krkn-operator-bundle-ocp:<version>` — OpenShift bundle.

The bundles use the `stable-kubernetes` and `stable-ocp` channels respectively.
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

📖 For configuration, usage, compatibility, and advanced installation options, see the official documentation.📖 For configuration, usage, compatibility, and advanced installation options, see the **[official documentation](https://krkn-chaos.gateway.scarf.sh/krkn-operator/docs?source=github)**.

**Uninstall:**
```bash
helm uninstall krkn-operator -n krkn-operator-system
```

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
| `size`       | int    | no       | Max documents to return. Defaults to `50`; clamped to `500`. Negative values are rejected. |
| `startDate`  | string | no       | Inclusive lower bound on the run timestamp, `yyyy-MM-dd`. Defaults to 30 days ago. |
| `endDate`    | string | no       | Upper bound on the run timestamp, `yyyy-MM-dd`. Includes only the selected calendar day. Defaults to the current instant. |

Results are sorted newest-first by timestamp before the `size` limit is applied.

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
      "status": true
    }
  ],
  "total": 1,
  "stats": {
    "pass": 42,
    "fail": 8,
    "pass_percent": 84.0
  }
}
```

`total` is the number of documents returned. Hits whose stored shape cannot be parsed are skipped rather than failing the request, so `total` may be smaller than the cluster's raw hit count.

`stats` summarizes run-level pass/fail across the entire matched time window (all documents in range), not just the returned `size`-capped page, so `pass` + `fail` may exceed `total`. Fields:

| Field          | Type   | Description |
|----------------|--------|-------------|
| `pass`         | int    | Runs with `job_status` true in the matched window. |
| `fail`         | int    | Runs with `job_status` false in the matched window. |
| `pass_percent` | float  | `pass` / (`pass` + `fail`) as a percentage, `0`-`100`, rounded to 2 decimals; `0` when no runs matched. |

**Errors:**

| Status | Meaning |
|--------|---------|
| `400`  | Invalid request body or validation failure: missing `configName`, negative `size`, a malformed date, `startDate` after `endDate`, or an `endDate` in the future. |
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

Authenticated users can read the image-signature verification setting at
`GET /api/v1/operator/signature-verification`. Administrators can update it
with `PATCH` and a required boolean body, for example
`{"enabled":false}`. Image verification remains observable when enforcement
is disabled; only the enforcement result is ignored.

## License

Copyright 2025 krkn-chaos

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
