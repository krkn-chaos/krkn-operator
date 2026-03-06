# Deployment Guide

This guide explains how to build, push, and deploy the krkn-operator using the standardized Makefile.

## Architecture

The krkn-operator consists of two components running in a single pod:

1. **Operator (Go)**: Controller managing CRDs and REST API (port 8080)
2. **Data Provider (Python)**: gRPC service using krkn-lib (port 50051)

## Prerequisites

- Docker or Podman for building images
- kubectl configured for your target cluster
- Access to a container registry (for remote deployment)
- make utility

## Makefile Variables

### Registry and Image Configuration

```bash
COMPONENT=krkn-operator                    # Component name (default)
REGISTRY=quay.io/krkn-chaos               # Container registry (default)
NAMESPACE=krkn-operator-system            # Deployment namespace (default)
CONTAINER_TOOL=docker                      # docker or podman (default: docker)
```

### Image Names (Auto-Generated)

```bash
IMG_NAME=$(COMPONENT)                                    # krkn-operator
IMG_DATA_PROVIDER_NAME=$(COMPONENT)-data-provider      # krkn-operator-data-provider

# Full image URLs (default to :latest tag)
IMG=$(REGISTRY)/$(IMG_NAME):latest
IMG_DATA_PROVIDER=$(REGISTRY)/$(IMG_DATA_PROVIDER_NAME):latest
```

### Git Tag Support

The Makefile automatically detects git tags for versioning:

```bash
# Without git tag
make docker-build
# → quay.io/krkn-chaos/krkn-operator:latest

# With git tag v1.0.0
git tag v1.0.0
make docker-build
# → quay.io/krkn-chaos/krkn-operator:latest
# → quay.io/krkn-chaos/krkn-operator:v1.0.0
```

## Building Images

### Build Both Images

```bash
# Default: builds with :latest tag
make docker-build-all

# With git tag: builds :latest and :<git-tag>
git tag v1.0.0
make docker-build-all
# Builds: :latest and :v1.0.0 for both images
```

### Build Individual Images

```bash
# Operator only
make docker-build

# Data provider only
make docker-build-data-provider
```

### Using Podman

```bash
# Option 1: Use podman-specific targets
make podman-build-all

# Option 2: Override CONTAINER_TOOL
make docker-build-all CONTAINER_TOOL=podman
```

### Custom Registry/Images

```bash
# Custom registry (uses default :latest and git tag logic)
make docker-build-all REGISTRY=myregistry.io/myorg

# Custom complete image URLs (skips git tag logic)
make docker-build IMG=custom.io/operator:beta
make docker-build-data-provider IMG_DATA_PROVIDER=custom.io/provider:beta
```

## Pushing Images

### Push Both Images

```bash
# Push :latest (and :tag if git tag exists)
make docker-push-all

# With git tag
git tag v1.0.0
make docker-push-all
# Pushes: :latest and :v1.0.0 for both images
```

### Push Individual Images

```bash
make docker-push
make docker-push-data-provider
```

### Custom Push

```bash
# Custom registry
make docker-push-all REGISTRY=myregistry.io/myorg

# Override specific images
make docker-push IMG=custom.io/op:v1
make docker-push-data-provider IMG_DATA_PROVIDER=custom.io/dp:v1
```

## Deployment

### Quick Deploy

```bash
# Deploy to default namespace (krkn-operator-system)
make deploy

# Deploy creates namespace automatically if missing
```

### Custom Namespace

```bash
# Deploy to custom namespace
make deploy NAMESPACE=my-custom-namespace
```

### Custom Images

```bash
# Deploy with specific image versions
make deploy IMG=myregistry.io/operator:v1.0.0 IMG_DATA_PROVIDER=myregistry.io/provider:v1.0.0
```

### OpenShift Deployment

For OpenShift, use the dedicated target that configures SCC:

```bash
make deploy-openshift NAMESPACE=krkn-operator-system
```

### Install CRDs Only

```bash
# Install only CRDs (without operator deployment)
make install
```

## Complete Workflow Examples

### Development (Local)

```bash
# Build locally
make docker-build-all

# For kind clusters: load images
kind load docker-image quay.io/krkn-chaos/krkn-operator:latest
kind load docker-image quay.io/krkn-chaos/krkn-operator-data-provider:latest

# Deploy with local images
make deploy
```

### Production (With Git Tag)

```bash
# Tag release
git tag v1.0.0

# Build (creates :latest and :v1.0.0)
make docker-build-all

# Push both tags to registry
make docker-push-all

# Deploy using :latest
make deploy

# Or deploy specific version (use full image URL)
make deploy IMG=$(REGISTRY)/$(IMG_NAME):v1.0.0
```

### Custom Registry

```bash
# Build for custom registry
make docker-build-all REGISTRY=harbor.company.com/chaos

# Push to custom registry
make docker-push-all REGISTRY=harbor.company.com/chaos

# Deploy from custom registry
make deploy REGISTRY=harbor.company.com/chaos
```

## Build Installer Bundle

Generate a single YAML file with all resources:

```bash
make build-installer

# Output: dist/install.yaml
kubectl apply -f dist/install.yaml
```

## Verification

```bash
# Check operator pod
kubectl get pods -n krkn-operator-system

# Check both containers
kubectl get pods -n krkn-operator-system -o jsonpath='{.items[0].spec.containers[*].name}'

# Check operator logs
kubectl logs -n krkn-operator-system deployment/krkn-operator-controller-manager -c manager

# Check data-provider logs
kubectl logs -n krkn-operator-system deployment/krkn-operator-controller-manager -c data-provider

# Test REST API (port-forward)
kubectl port-forward -n krkn-operator-system svc/krkn-operator-controller-manager-api-service 8080:8080
curl http://localhost:8080/api/v1/health
```

## Undeployment

```bash
# Remove operator deployment
make undeploy

# Undeploy from custom namespace
make undeploy NAMESPACE=my-custom-namespace

# Remove CRDs (WARNING: deletes all custom resources)
make uninstall
```

## Available Make Targets

### Build Targets
- `docker-build` - Build operator image
- `docker-build-data-provider` - Build data-provider image
- `docker-build-all` - Build both images
- `podman-build` - Build operator with podman
- `podman-build-data-provider` - Build data-provider with podman
- `podman-build-all` - Build both with podman

### Push Targets
- `docker-push` - Push operator image
- `docker-push-data-provider` - Push data-provider image
- `docker-push-all` - Push both images
- `podman-push` - Push operator with podman
- `podman-push-data-provider` - Push data-provider with podman
- `podman-push-all` - Push both with podman

### Deployment Targets
- `install` - Install CRDs only
- `uninstall` - Uninstall CRDs
- `deploy` - Deploy operator to cluster
- `deploy-openshift` - Deploy with OpenShift SCC configuration
- `undeploy` - Remove operator from cluster
- `build-installer` - Generate dist/install.yaml bundle

### Other Targets
- `manifests` - Generate CRD manifests
- `test` - Run unit tests
- `lint` - Run golangci-lint
- `help` - Show all available targets

## Troubleshooting

### Image Pull Errors

Ensure images are pushed and accessible:
```bash
# Verify images exist in registry
docker pull quay.io/krkn-chaos/krkn-operator:latest
docker pull quay.io/krkn-chaos/krkn-operator-data-provider:latest
```

### Pod Not Starting

```bash
# Check pod events
kubectl describe pod -n krkn-operator-system <pod-name>

# Check both container logs
kubectl logs -n krkn-operator-system <pod-name> -c manager
kubectl logs -n krkn-operator-system <pod-name> -c data-provider
```

### gRPC Connection Issues

```bash
# Verify data-provider is listening on port 50051
kubectl exec -n krkn-operator-system <pod-name> -c data-provider -- netstat -ln | grep 50051
```

### Namespace Issues

If namespace doesn't exist, `make deploy` creates it automatically. To manually create:

```bash
kubectl create namespace krkn-operator-system
```

## Multi-Architecture Support

Build for multiple platforms:

```bash
make docker-buildx
# Builds for: linux/arm64, linux/amd64, linux/s390x, linux/ppc64le
```

## Environment Variables

Override Makefile defaults:

```bash
export REGISTRY=harbor.company.com/chaos
export NAMESPACE=production-chaos
export CONTAINER_TOOL=podman

make docker-build-all
make docker-push-all
make deploy
```

## Git Tag Workflow

```bash
# Check current tags
git tag

# Create new tag
git tag v1.0.0

# Build (creates both :latest and :v1.0.0)
make docker-build-all

# Verify tags
docker images | grep krkn-operator

# Push both tags
make docker-push-all

# Delete tag if needed
git tag -d v1.0.0
```
