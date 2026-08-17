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

## Development

Interested in contributing or running Krkn Operator from source? See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Licensed under the [Apache License 2.0](LICENSE).
