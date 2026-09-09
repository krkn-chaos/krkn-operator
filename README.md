# krkn-operator

![test](https://github.com/krkn-chaos/krkn-operator/actions/workflows/test.yml/badge.svg)
![pr-checks](https://github.com/krkn-chaos/krkn-operator/actions/workflows/pr-checks.yml/badge.svg)
![coverage](https://krkn-chaos.github.io/krkn-lib-docs/coverage_badge_krkn-operator.svg)


**Centralized, multi-cluster chaos engineering for Kubernetes and OpenShift.**

Krkn Operator is a Kubernetes-native platform built on the [Krkn](https://github.com/krkn-chaos/krkn) framework to centrally orchestrate and manage chaos experiments across multiple clusters.

* **Multi-cluster orchestration** — Run chaos experiments across Kubernetes and OpenShift clusters from a single control plane.
* **Chaos Studio** — Visually compose and execute reusable chaos workflows.
* **Access control** — Manage users, groups, cluster access, and permissions.
* **OCM/ACM integration** — Discover and run experiments on clusters managed by Open Cluster Management or Red Hat Advanced Cluster Management.

<img width="2872" height="1848" alt="image" src="https://github.com/user-attachments/assets/19391c24-760e-495e-83ef-1f944bd196be" />



## Quick Start

Install Krkn Operator using Helm:

```bash id="krkn-install"
helm install krkn-operator oci://quay.io/krkn-chaos/charts/krkn-operator --version <version> \
  -n krkn-operator-system --create-namespace
```

📖 For configuration, usage, compatibility, and advanced installation options, see the official documentation.📖 For configuration, usage, compatibility, and advanced installation options, see the **[official documentation](https://krkn-chaos.gateway.scarf.sh/krkn-operator/docs?source=github)**.

## Telemetry Query API

Query chaos-run telemetry stored in Elasticsearch/OpenSearch through the operator's REST API. Credentials are never sent by the client — the operator resolves them server-side from a previously saved Elasticsearch config, so the request only references that config by name.

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
  "total": 1
}
```

`total` is the number of documents returned. Hits whose stored shape cannot be parsed are skipped rather than failing the request, so `total` may be smaller than the cluster's raw hit count.

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

- [krkn-operator-console](https://krkn-chaos.gateway.scarf.sh/krkn-operator/console?source=github-main) — Web console and Chaos Studio for Krkn Operator.
- [krkn-operator-acm](https://krkn-chaos.gateway.scarf.sh/krkn-operator/acm?source=github-main) — Open Cluster Management and Red Hat ACM integration for multi-cluster environments.

## Development

Interested in contributing or running Krkn Operator from source? See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Licensed under the [Apache License 2.0](LICENSE).
