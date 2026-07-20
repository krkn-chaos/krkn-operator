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

**Uninstall:**
```bash
helm uninstall krkn-operator -n krkn-operator-system
```

See [DEPLOYMENT.md](DEPLOYMENT.md) for full installation options and configuration.

## License

Copyright 2025 krkn-chaos

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
