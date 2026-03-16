# krkn-operator Helm Chart

Helm chart for deploying the krkn-operator chaos engineering ecosystem on Kubernetes/OpenShift.

## Installation

The Helm chart is published as an OCI artifact to Quay.io.

### Install with defaults (Kind/Minikube)

```bash
helm install my-krkn oci://quay.io/krkn-chaos/charts/krkn-operator --version 0.1.0
```

### Install with OpenShift Route

```bash
helm install my-krkn oci://quay.io/krkn-chaos/charts/krkn-operator --version 0.1.0 \
  --set console.route.enabled=true \
  --set console.route.hostname=krkn.apps.cluster.com
```

### Install with Kubernetes Ingress (legacy)

```bash
helm install my-krkn oci://quay.io/krkn-chaos/charts/krkn-operator --version 0.1.0 \
  --set console.ingress.enabled=true \
  --set console.ingress.hostname=krkn.example.com
```

### Install with Gateway API (recommended for Kubernetes)

Gateway API is the modern successor to Ingress. Requires Gateway API CRDs and an existing Gateway resource.

```bash
helm install my-krkn oci://quay.io/krkn-chaos/charts/krkn-operator --version 0.1.0 \
  --set console.gateway.enabled=true \
  --set console.gateway.gatewayName=krkn-gateway \
  --set console.gateway.hostname=krkn.example.com
```

### Install with ACM integration

```bash
helm install my-krkn oci://quay.io/krkn-chaos/charts/krkn-operator --version 0.1.0 \
  --set acm.enabled=true \
  --set console.route.enabled=true
```

## Components

- **krkn-operator** (core): Main operator with REST API and gRPC data provider sidecar
- **krkn-operator-console** (UI): React-based web console
- **krkn-operator-acm** (optional): ACM (Advanced Cluster Management) integration

## Configuration

See [values.yaml](values.yaml) for all available configuration options.

### Key Configuration Options

| Parameter | Description | Default |
|-----------|-------------|---------|
| `console.enabled` | Enable web console | `true` |
| `console.gateway.enabled` | Enable Gateway API (recommended) | `false` |
| `console.ingress.enabled` | Enable Kubernetes Ingress (legacy) | `false` |
| `console.route.enabled` | Enable OpenShift Route | `false` |
| `acm.enabled` | Enable ACM integration | `false` |
| `monitoring.enabled` | Enable Prometheus ServiceMonitor | `false` |

## Console nginx Configuration

The console uses nginx as a reverse proxy to the operator API. The Helm chart automatically configures the nginx proxy with the correct operator service URL based on the release name and namespace.

**Compatibility:**
- ✅ `kubectl apply` - Uses embedded nginx.conf from Docker image
- ✅ `make deploy` (Kustomize) - Uses embedded nginx.conf from Docker image
- ✅ `helm install` - Uses ConfigMap with dynamic service URL

## Multi-Architecture Support

The chart uses multi-architecture images (linux/amd64, linux/arm64) automatically published to Quay.io. The container runtime automatically selects the correct architecture.

**Note**: Default `imagePullPolicy: Always` for `:latest` tag ensures the correct architecture is pulled on Kind/Minikube clusters. For production with specific versions, you can override:

```bash
helm install krkn-operator oci://quay.io/krkn-chaos/charts/krkn-operator --version 0.1.0 \
  --set images.operator.image=quay.io/krkn-chaos/krkn-operator:v1.0.0 \
  --set images.operator.pullPolicy=IfNotPresent
```

## Requirements

- Kubernetes 1.19+ or OpenShift 4.x
- Helm 3.0+

## Uninstalling

```bash
helm uninstall my-krkn
```

**Note:** CRDs are kept by default to prevent data loss. To remove CRDs:

```bash
kubectl delete crd krknscenari oruns.krkn.krkn-chaos.dev
kubectl delete crd krkntargetrequests.krkn.krkn-chaos.dev
# ... etc
```
