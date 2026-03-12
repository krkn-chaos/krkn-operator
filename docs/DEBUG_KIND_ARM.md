# Debugging krkn-operator on Kind (macOS ARM)

## Issue
Operator fails to start or crashes on Kind running on Apple Silicon (ARM64/M1/M2).

## Diagnostic Steps

### 1. Get Full Logs

```bash
# Operator logs
kubectl logs -n krkn-operator-system deployment/krkn-operator-controller-manager -c manager --previous

# Data provider logs
kubectl logs -n krkn-operator-system deployment/krkn-operator-controller-manager -c data-provider --previous

# Pod events
kubectl describe pod -n krkn-operator-system -l control-plane=controller-manager
```

### 2. Verify Image Architecture

```bash
# Check which image is being used
kubectl get deployment -n krkn-operator-system krkn-operator-controller-manager \
  -o jsonpath='{.spec.template.spec.containers[*].image}'

# Inspect image architecture (if pulled locally)
docker inspect quay.io/krkn-chaos/krkn-operator:latest | grep Architecture
docker inspect quay.io/krkn-chaos/krkn-operator-data-provider:latest | grep Architecture
```

### 3. Check Resource Constraints

```bash
# View resource requests/limits
kubectl get pod -n krkn-operator-system -o yaml | grep -A 10 resources:

# Check node resources
kubectl top nodes
kubectl describe node <kind-node-name>
```

### 4. Test Binary Directly

```bash
# Get the manager binary from image
docker run --rm --entrypoint /bin/sh quay.io/krkn-operator:latest -c "file /manager"

# Should output: /manager: ELF 64-bit LSB executable, ARM aarch64...
```

## Common Issues & Solutions

### Issue 1: Wrong Architecture Image

**Symptom**: Pod crashes immediately, exec format error in logs

**Solution**: Ensure you're using multi-arch image or ARM-specific tag
```bash
# Pull and verify architecture
docker pull --platform linux/arm64 quay.io/krkn-chaos/krkn-operator:latest
docker inspect quay.io/krkn-chaos/krkn-operator:latest | grep -i arch
```

### Issue 2: Insufficient Resources

**Symptom**: Pod stuck in Pending or OOMKilled

**Solution**: Increase Kind cluster resources
```yaml
# kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraMounts:
  - containerPath: /var/lib/kubelet/config.json
    hostPath: $HOME/.docker/config.json
```

### Issue 3: Python Dependencies on ARM

**Symptom**: data-provider container fails to start

**Solution**: Check Python package compatibility
```bash
# Test data-provider container
docker run --rm --platform linux/arm64 \
  quay.io/krkn-chaos/krkn-operator-data-provider:latest \
  python -c "import grpc; print('OK')"
```

### Issue 4: Goroutine/GC Issues

**Symptom**: Operator appears to hang, goroutine dumps show only GC workers

**Possible causes**:
- Deadlock in initialization code
- Waiting for unavailable resource (e.g., CRD not ready)
- Network connectivity issue

**Debug**:
```bash
# Enable debug logging
kubectl set env deployment/krkn-operator-controller-manager -n krkn-operator-system LOG_LEVEL=debug

# Check if CRDs are installed
kubectl get crd | grep krkn

# Check RBAC permissions
kubectl auth can-i --list --as=system:serviceaccount:krkn-operator-system:krkn-operator-controller-manager
```

## Recommended Kind Configuration for ARM

```yaml
# kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
        system-reserved: memory=2Gi
```

Create cluster:
```bash
kind create cluster --config kind-config.yaml
```

## Quick Test

```bash
# Minimal test deployment (operator only, no console/ACM)
helm install krkn-operator oci://quay.io/krkn-chaos/charts/krkn-operator --version 0.1.0 \
  --set console.enabled=false \
  --set acm.enabled=false \
  --set operator.resources.limits.memory=256Mi \
  --set operator.resources.requests.memory=128Mi

# Wait and check
kubectl wait --for=condition=ready pod -l control-plane=controller-manager -n krkn-operator-system --timeout=5m
kubectl logs -n krkn-operator-system -l control-plane=controller-manager -c manager -f
```

## Report Issue

If problem persists, collect diagnostics:

```bash
# Create diagnostics bundle
kubectl logs -n krkn-operator-system deployment/krkn-operator-controller-manager -c manager > operator.log
kubectl logs -n krkn-operator-system deployment/krkn-operator-controller-manager -c data-provider > data-provider.log
kubectl describe pod -n krkn-operator-system -l control-plane=controller-manager > pod-describe.txt
kubectl get events -n krkn-operator-system --sort-by='.lastTimestamp' > events.txt

# Attach to GitHub issue with:
# - macOS version
# - Kind version
# - Helm chart version
# - All log files above
```