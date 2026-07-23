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
## Running Locally

The operator requires three components running concurrently: the Python gRPC data provider, the Go operator, and (optionally) the React console.

### Prerequisites

- Go 1.21+
- Python 3.11+
- `kubectl` connected to a Kubernetes cluster
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

Alternatively, use the helper script which also installs CRDs and builds the binary:

```bash
./start_operator.sh
```

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

## License

Copyright 2025 krkn-chaos

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
