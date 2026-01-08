# Deployment Guide

This guide explains how to build and deploy the krkn-operator with its data-provider sidecar to a Kubernetes cluster.

## Architecture

The krkn-operator consists of two main components:

1. **Operator (Go)**: The main controller that manages KrknTargetRequest CRDs and exposes a REST API
2. **Data Provider (Python)**: A gRPC service that uses krkn-lib to interact with Kubernetes clusters

These components run as a multi-container pod using the sidecar pattern, communicating via gRPC on localhost:50051.

## Prerequisites

- Docker or Podman for building container images
- Access to a container registry (for pushing images)
- kubectl configured to access your target Kubernetes cluster
- make utility

## Building Images

### Build Both Images

```bash
# Build operator and data-provider images
make docker-build-all

# With custom image tags
make docker-build-all IMG=myregistry/krkn-operator:v1.0.0 DATA_PROVIDER_IMG=myregistry/data-provider:v1.0.0
```

### Build Individual Images

```bash
# Build operator image only
make docker-build

# Build data-provider image only
make docker-build-data-provider
```

### Using Podman Instead of Docker

If you prefer to use Podman instead of Docker, you can use the dedicated podman targets:

```bash
# Build both images with podman
make podman-build-all IMG=myregistry/krkn-operator:v1.0.0 DATA_PROVIDER_IMG=myregistry/data-provider:v1.0.0

# Build individual images with podman
make podman-build IMG=myregistry/krkn-operator:v1.0.0
make podman-build-data-provider DATA_PROVIDER_IMG=myregistry/data-provider:v1.0.0

# Push images with podman
make podman-push-all IMG=myregistry/krkn-operator:v1.0.0 DATA_PROVIDER_IMG=myregistry/data-provider:v1.0.0
```

Alternatively, you can set the `CONTAINER_TOOL` variable:

```bash
# Use podman for all container operations
make docker-build-all CONTAINER_TOOL=podman IMG=myregistry/krkn-operator:v1.0.0 DATA_PROVIDER_IMG=myregistry/data-provider:v1.0.0
```

## Pushing Images to Registry

Before deploying to a cluster, push the images to a container registry:

```bash
# Push both images
make docker-push-all IMG=myregistry/krkn-operator:v1.0.0 DATA_PROVIDER_IMG=myregistry/data-provider:v1.0.0

# Or push individually
make docker-push IMG=myregistry/krkn-operator:v1.0.0
make docker-push-data-provider DATA_PROVIDER_IMG=myregistry/data-provider:v1.0.0
```

## Deployment

### Option 1: Direct Deployment with kubectl

```bash
# Install CRDs
make install

# Deploy the operator with both containers
make deploy IMG=myregistry/krkn-operator:v1.0.0 DATA_PROVIDER_IMG=myregistry/data-provider:v1.0.0
```

### Option 2: Generate Installation YAML

Generate a consolidated YAML file containing all resources:

```bash
make build-installer IMG=myregistry/krkn-operator:v1.0.0 DATA_PROVIDER_IMG=myregistry/data-provider:v1.0.0
```

This creates `dist/install.yaml` which you can apply directly:

```bash
kubectl apply -f dist/install.yaml
```

## Configuration

### Environment Variables

The operator supports the following environment variables:

- `KRKN_NAMESPACE`: Namespace to watch for KrknTargetRequest resources (future: will support ConfigMap)

### Image Configuration

Images are configured via Makefile variables:

- `IMG`: Operator container image (default: `controller:latest`)
- `DATA_PROVIDER_IMG`: Data provider container image (default: `data-provider:latest`)
- `CONTAINER_TOOL`: Container tool to use (default: `docker`, can use `podman`)

## Verification

After deployment, verify the operator is running:

```bash
# Check operator pod
kubectl get pods -n krkn-operator-system

# Check both containers in the pod
kubectl get pods -n krkn-operator-system -o jsonpath='{.items[0].spec.containers[*].name}'

# Check logs for operator
kubectl logs -n krkn-operator-system deployment/krkn-operator-controller-manager -c manager

# Check logs for data-provider
kubectl logs -n krkn-operator-system deployment/krkn-operator-controller-manager -c data-provider
```

## REST API Access

The operator exposes a REST API on port 8080 through a Kubernetes Service.

### Access from within the cluster

From any pod in the same namespace:

```bash
curl http://krkn-operator-controller-manager-api-service:8080/api/v1/targets
```

From a pod in a different namespace:

```bash
curl http://krkn-operator-controller-manager-api-service.krkn-operator-system.svc.cluster.local:8080/api/v1/targets
```

### Access from outside the cluster

#### Option 1: Port Forward

```bash
# Port forward to the service
kubectl port-forward -n krkn-operator-system service/krkn-operator-controller-manager-api-service 8080:8080

# In another terminal, test the API
curl http://localhost:8080/api/v1/targets

# Get nodes for a specific cluster
curl http://localhost:8080/api/v1/nodes/<secret-id>?cluster-name=<cluster-name>
```

#### Option 2: NodePort (for development/testing)

Edit the service to use NodePort:

```bash
kubectl patch service -n krkn-operator-system krkn-operator-controller-manager-api-service -p '{"spec":{"type":"NodePort"}}'

# Get the NodePort
kubectl get service -n krkn-operator-system krkn-operator-controller-manager-api-service
```

#### Option 3: LoadBalancer (for production with cloud provider)

```bash
kubectl patch service -n krkn-operator-system krkn-operator-controller-manager-api-service -p '{"spec":{"type":"LoadBalancer"}}'

# Get the external IP
kubectl get service -n krkn-operator-system krkn-operator-controller-manager-api-service
```

#### Option 4: Ingress (recommended for production)

Create an Ingress resource to expose the API with hostname-based routing and TLS.

## Undeployment

To remove the operator from your cluster:

```bash
# Remove the operator deployment
make undeploy

# Remove the CRDs (warning: this will delete all custom resources)
make uninstall
```

## Troubleshooting

### Pod Not Starting

Check pod events and logs:

```bash
kubectl describe pod -n krkn-operator-system <pod-name>
kubectl logs -n krkn-operator-system <pod-name> -c manager
kubectl logs -n krkn-operator-system <pod-name> -c data-provider
```

### gRPC Connection Issues

If the operator cannot connect to the data-provider:

1. Check that both containers are running in the same pod
2. Verify the data-provider is listening on port 50051:
   ```bash
   kubectl exec -n krkn-operator-system <pod-name> -c data-provider -- netstat -ln | grep 50051
   ```
3. Check data-provider logs for startup errors

### Image Pull Errors

Ensure:
- Images are pushed to the registry
- The cluster has credentials to pull from your registry
- Image names and tags are correct in the deployment

## Development Workflow

For local development:

1. Build images locally:
   ```bash
   make docker-build-all
   ```

2. For kind clusters, load images directly:
   ```bash
   kind load docker-image controller:latest --name <cluster-name>
   kind load docker-image data-provider:latest --name <cluster-name>
   ```

3. Deploy with local images:
   ```bash
   make deploy
   ```

## Multi-Architecture Builds

To build for multiple architectures:

```bash
make docker-buildx IMG=myregistry/krkn-operator:v1.0.0
```

This builds and pushes images for: linux/arm64, linux/amd64, linux/s390x, linux/ppc64le

## Component Details

### Operator Container

- **Port 8080**: REST API endpoint
- **Port 8083**: Health probe endpoint
- **Resources**:
  - Requests: 10m CPU, 64Mi memory
  - Limits: 500m CPU, 128Mi memory

### Data Provider Container

- **Port 50051**: gRPC endpoint
- **Resources**:
  - Requests: 50m CPU, 128Mi memory
  - Limits: 200m CPU, 256Mi memory
- **Health Checks**: TCP socket probes on gRPC port
