#!/usr/bin/env bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
HUB_CONTEXT="${HUB_CONTEXT:-kind-hub}"
CLUSTER1_CONTEXT="${CLUSTER1_CONTEXT:-kind-cluster1}"
CLUSTER2_CONTEXT="${CLUSTER2_CONTEXT:-kind-cluster2}"
OPERATOR_NAMESPACE="${OPERATOR_NAMESPACE:-krkn-operator}"
ADDON_NAMESPACE="open-cluster-management-addon"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}OCM & krkn-operator Setup Script${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Hub context: $HUB_CONTEXT"
echo "Managed clusters: $CLUSTER1_CONTEXT, $CLUSTER2_CONTEXT"
echo "Operator namespace: $OPERATOR_NAMESPACE"
echo ""

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

wait_for_condition() {
    local description="$1"
    local timeout="$2"
    local check_command="$3"

    log_info "Waiting for: $description (timeout: ${timeout}s)..."
    local elapsed=0
    while [ $elapsed -lt $timeout ]; do
        if eval "$check_command" >/dev/null 2>&1; then
            log_info "✓ $description"
            return 0
        fi
        sleep 5
        elapsed=$((elapsed + 5))
        echo -n "."
    done
    echo ""
    log_error "Timeout waiting for: $description"
    return 1
}

# Step 1: Install ManagedServiceAccount addon
log_info "Step 1: Installing ManagedServiceAccount addon..."

log_info "Adding OCM Helm repository..."
helm repo add ocm https://open-cluster-management.io/helm-charts >/dev/null 2>&1 || true
helm repo update >/dev/null

log_info "Installing ManagedServiceAccount addon..."
helm install -n $ADDON_NAMESPACE --create-namespace \
    managed-serviceaccount ocm/managed-serviceaccount \
    --kube-context $HUB_CONTEXT \
    --wait --timeout 3m

wait_for_condition "ManagedServiceAccount addon pod to be ready" 120 \
    "kubectl get pods -n $ADDON_NAMESPACE --context $HUB_CONTEXT 2>/dev/null | grep -q managed-serviceaccount"

kubectl wait --for=condition=ready pod \
    -n $ADDON_NAMESPACE \
    --all \
    --timeout=120s \
    --context $HUB_CONTEXT

log_info "✓ ManagedServiceAccount addon installed"
echo ""

# Step 2: Create ManagedServiceAccounts
log_info "Step 2: Creating ManagedServiceAccounts..."

log_info "Creating ManagedServiceAccount for cluster1..."
kubectl apply --context $HUB_CONTEXT -f - <<EOF
apiVersion: authentication.open-cluster-management.io/v1beta1
kind: ManagedServiceAccount
metadata:
  name: application-manager
  namespace: cluster1
spec:
  rotation:
    enabled: true
    validity: 8640h
EOF

log_info "Creating ManagedServiceAccount for cluster2..."
kubectl apply --context $HUB_CONTEXT -f - <<EOF
apiVersion: authentication.open-cluster-management.io/v1beta1
kind: ManagedServiceAccount
metadata:
  name: application-manager
  namespace: cluster2
spec:
  rotation:
    enabled: true
    validity: 8640h
EOF

log_info "Waiting for tokens to be reported..."
sleep 10

kubectl wait --for=condition=TokenReported \
    managedserviceaccount/application-manager \
    -n cluster1 \
    --timeout=60s \
    --context $HUB_CONTEXT

kubectl wait --for=condition=TokenReported \
    managedserviceaccount/application-manager \
    -n cluster2 \
    --timeout=60s \
    --context $HUB_CONTEXT

log_info "✓ ManagedServiceAccounts created and tokens reported"
echo ""

# Step 3: Grant cluster-admin permissions
log_info "Step 3: Creating ManifestWork for RBAC permissions..."

log_info "Creating ManifestWork for cluster1 RBAC..."
kubectl apply --context $HUB_CONTEXT -f - <<EOF
apiVersion: work.open-cluster-management.io/v1
kind: ManifestWork
metadata:
  name: application-manager-rbac
  namespace: cluster1
spec:
  workload:
    manifests:
    - apiVersion: rbac.authorization.k8s.io/v1
      kind: ClusterRoleBinding
      metadata:
        name: application-manager-admin
      roleRef:
        apiGroup: rbac.authorization.k8s.io
        kind: ClusterRole
        name: cluster-admin
      subjects:
      - kind: ServiceAccount
        name: application-manager
        namespace: open-cluster-management-agent-addon
EOF

log_info "Creating ManifestWork for cluster2 RBAC..."
kubectl apply --context $HUB_CONTEXT -f - <<EOF
apiVersion: work.open-cluster-management.io/v1
kind: ManifestWork
metadata:
  name: application-manager-rbac
  namespace: cluster2
spec:
  workload:
    manifests:
    - apiVersion: rbac.authorization.k8s.io/v1
      kind: ClusterRoleBinding
      metadata:
        name: application-manager-admin
      roleRef:
        apiGroup: rbac.authorization.k8s.io
        kind: ClusterRole
        name: cluster-admin
      subjects:
      - kind: ServiceAccount
        name: application-manager
        namespace: open-cluster-management-agent-addon
EOF

log_info "Waiting for ManifestWorks to be applied..."
sleep 10

kubectl wait --for=condition=Applied \
    manifestwork/application-manager-rbac \
    -n cluster1 \
    --timeout=60s \
    --context $HUB_CONTEXT

kubectl wait --for=condition=Applied \
    manifestwork/application-manager-rbac \
    -n cluster2 \
    --timeout=60s \
    --context $HUB_CONTEXT

log_info "✓ RBAC permissions granted"
echo ""

# Step 4: Verify ManagedServiceAccounts
log_info "Step 4: Verifying ManagedServiceAccounts setup..."

log_info "ManagedServiceAccounts:"
kubectl get managedserviceaccount -A --context $HUB_CONTEXT

log_info "Secrets on hub:"
kubectl get secret -n cluster1 application-manager --context $HUB_CONTEXT -o name
kubectl get secret -n cluster2 application-manager --context $HUB_CONTEXT -o name

log_info "ServiceAccounts on managed clusters:"
kubectl get sa -n open-cluster-management-agent-addon --context $CLUSTER1_CONTEXT | grep application-manager || log_warn "ServiceAccount not found on cluster1"
kubectl get sa -n open-cluster-management-agent-addon --context $CLUSTER2_CONTEXT | grep application-manager || log_warn "ServiceAccount not found on cluster2"

log_info "✓ ManagedServiceAccounts verified"
echo ""

# Step 5: Install krkn-operator
log_info "Step 5: Installing krkn-operator via Helm..."

# Change to repo root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

log_info "Installing from chart: $REPO_ROOT/charts/krkn-operator"

helm install krkn-operator "$REPO_ROOT/charts/krkn-operator" \
    --namespace $OPERATOR_NAMESPACE \
    --create-namespace \
    --set images.operator.image=quay.io/krkn-chaos/krkn-operator:latest \
    --kube-context $HUB_CONTEXT \
    --wait \
    --timeout 5m

log_info "✓ krkn-operator Helm chart installed"
echo ""

# Step 6: Wait for operator to be ready
log_info "Step 6: Waiting for krkn-operator to be ready..."

kubectl wait --for=condition=available deployment/krkn-operator-operator \
    -n $OPERATOR_NAMESPACE \
    --timeout=300s \
    --context $HUB_CONTEXT

kubectl wait --for=condition=ready pod \
    -n $OPERATOR_NAMESPACE \
    -l app.kubernetes.io/component=operator \
    --timeout=120s \
    --context $HUB_CONTEXT

log_info "✓ krkn-operator is ready"
echo ""

# Step 7: Show operator status
log_info "Step 7: Operator status..."
kubectl get pods -n $OPERATOR_NAMESPACE --context $HUB_CONTEXT
echo ""

# Final summary
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✓ Setup Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Next steps:"
echo "1. Port-forward the operator API:"
echo "   kubectl port-forward -n $OPERATOR_NAMESPACE svc/krkn-operator-operator 8080:8080 --context $HUB_CONTEXT"
echo ""
echo "2. Check if registration is open:"
echo "   curl -s http://localhost:8080/api/v1/auth/is-registered | jq"
echo ""
echo "3. Register first admin user:"
echo '   curl -X POST http://localhost:8080/api/v1/auth/register \'
echo '     -H "Content-Type: application/json" \'
echo '     -d '"'"'{'
echo '       "userId": "admin@krkn-chaos.local",'
echo '       "password": "AdminPassword123!",'
echo '       "name": "Admin",'
echo '       "surname": "User",'
echo '       "organization": "Krkn Chaos",'
echo '       "role": "admin"'
echo '     }'"'"' | jq'
echo ""
echo "4. Login:"
echo '   curl -X POST http://localhost:8080/api/v1/auth/login \'
echo '     -H "Content-Type: application/json" \'
echo '     -d '"'"'{'
echo '       "userId": "admin@krkn-chaos.local",'
echo '       "password": "AdminPassword123!"'
echo '     }'"'"' | jq'
echo ""