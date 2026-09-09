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

## Ecosystem

- [krkn-operator-console](https://krkn-chaos.gateway.scarf.sh/krkn-operator/console?source=github-main) — Web console and Chaos Studio for Krkn Operator.
- [krkn-operator-acm](https://krkn-chaos.gateway.scarf.sh/krkn-operator/acm?source=github-main) — Open Cluster Management and Red Hat ACM integration for multi-cluster environments.

## Development

Interested in contributing or running Krkn Operator from source? See [CONTRIBUTING.md](CONTRIBUTING.md).

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

Authenticated users can read the image-signature verification setting at
`GET /api/v1/operator/signature-verification`. Administrators can update it
with `PATCH` and a required boolean body, for example
`{"enabled":false}`. Image verification remains observable when enforcement
is disabled; only the enforcement result is ignored.

## Backup and Restore

Admin users can back up and restore operator configuration (users, groups, targets, providers, credentials) for disaster recovery and cross-cluster migration using the web console.

**To download a backup:** Click "Download Backup" in the Backup & Restore card.

**To restore from a backup:** Click "Upload & Restore" in the Backup & Restore card, select your backup archive, and confirm. The console will monitor restore progress in real time.

**Important:** After restore completes, restart the operator pod to apply credential changes:
```bash
kubectl rollout restart deployment krkn-operator -n krkn-operator-system
```

**Limits:**
- Maximum upload size: 100 MB
- Maximum concurrent restores: 1000
- Backup history: Not persisted — status is lost on operator restart

## License

Licensed under the [Apache License 2.0](LICENSE).
