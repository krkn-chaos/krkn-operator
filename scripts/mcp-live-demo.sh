#!/usr/bin/env bash
# Live manual-test setup for the krkn-operator MCP server (issue #171 / PR #175).
#
# Usage (run from the repo root):
#   oc login <api-url> -u kubeadmin -p <password>   # first
#   ./scripts/mcp-live-demo.sh setup                             # deploy + bootstrap + robot
#   ./scripts/mcp-live-demo.sh token                             # print the robot token
#   ./scripts/mcp-live-demo.sh call '<json>'                     # raw MCP tool call
#   ./scripts/mcp-live-demo.sh cleanup                           # remove everything
#
# Requires: oc, kubectl, go, python3. Run from the krkn-operator-mcp worktree.
set -euo pipefail

NS="${KRKN_LIVE_NS:-krkn-live-test}"
API_PORT="${KRKN_LIVE_PORT:-18080}"
LOG="${TMPDIR:-/tmp}/krkn-live-operator.log"
TOKEN_FILE="${TMPDIR:-/tmp}/krkn-live-robot-token.txt"
ROBOT="${KRKN_LIVE_ROBOT:-manual-bot}"
BIN="${TMPDIR:-/tmp}/krkn-operator-live"
CLUSTER_API=$(oc whoami --show-server 2>/dev/null) || { echo "Run 'oc login' first"; exit 1; }

build() { go build -o "$BIN" ./cmd/main.go; }

setup() {
  build
  kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f - >/dev/null

  echo "==> Installing CRDs"
  make install >/dev/null

  echo "==> Bootstrapping SA/RBAC/SCC"
  POD_NAMESPACE="$NS" KRKN_OLM_OPENSHIFT=true "$BIN" --bootstrap-resources \
    --metrics-bind-address=0 --leader-elect=false >/dev/null 2>&1

  echo "==> Starting operator (log: $LOG)"
  pkill -f "krkn-operator-live -api-port=$API_PORT" 2>/dev/null || true
  KRKN_MCP_ENABLED=true KRKN_NAMESPACE="$NS" POD_NAMESPACE="$NS" \
    nohup "$BIN" -api-port=$API_PORT -health-probe-bind-address=:$((API_PORT+1)) \
    > "$LOG" 2>&1 &
  echo $! > "${TMPDIR:-/tmp}/krkn-live-operator.pid"

  echo "==> Creating group + robot + target"
  kubectl apply -f - >/dev/null <<EOF
apiVersion: krkn.krkn-chaos.dev/v1alpha1
kind: KrknUserGroup
metadata: {name: live-sre, namespace: $NS}
spec:
  name: live-sre
  clusterPermissions:
    "$CLUSTER_API": {actions: ["view", "run", "cancel"]}
---
apiVersion: krkn.krkn-chaos.dev/v1alpha1
kind: KrknRobot
metadata:
  name: $ROBOT
  namespace: $NS
  labels: {group.krkn.krkn-chaos.dev/live-sre: "true"}
spec: {description: manual test robot}
EOF

  # Kubeconfig secret: the controller expects JSON {"kubeconfig": "<base64>"}.
  python3 - "$NS" "$ROBOT" "$TOKEN_FILE" <<'PYEOF'
import base64, hashlib, json, os, secrets, subprocess, sys, datetime
ns, robot, token_file = sys.argv[1], sys.argv[2], sys.argv[3]
uid = subprocess.check_output(["kubectl","get","krknrobot",robot,"-n",ns,
    "-o","jsonpath={.metadata.uid}"]).decode().strip()
token = f"krkn_robot_{robot}." + base64.urlsafe_b64encode(secrets.token_bytes(32)).rstrip(b"=").decode()
expires = (datetime.datetime.now(datetime.timezone.utc)+datetime.timedelta(days=90)).strftime("%Y-%m-%dT%H:%M:%SZ")
kube_raw = open(os.path.expanduser("~/.kube/config"), "rb").read()
kube = json.dumps({"kubeconfig": base64.b64encode(kube_raw).decode()})
subprocess.run(["kubectl","create","secret","generic","live-target-kubeconfig","-n",ns,
    "--from-file=kubeconfig=/dev/stdin"], input=kube.encode(), check=True,
    stdout=subprocess.DEVNULL)
secret = f"""apiVersion: v1
kind: Secret
metadata:
  name: krknrobot-{robot}-token
  namespace: {ns}
  labels:
    krkn.krkn-chaos.dev/robot-token: "true"
    krkn.krkn-chaos.dev/robot: {robot}
type: Opaque
stringData:
  tokenHash: {hashlib.sha256(token.encode()).hexdigest()}
  tokenId: {secrets.token_hex(4)}
  expiresAt: "{expires}"
  robotUID: {uid}
"""
subprocess.run(["kubectl","apply","-f","-"], input=secret.encode(), check=True,
    stdout=subprocess.DEVNULL)
open(token_file,"w").write(token)
PYEOF

  # Target: status.ready must be set via the status subresource (apply drops it).
  kubectl apply -f - >/dev/null <<EOF
apiVersion: krkn.krkn-chaos.dev/v1alpha1
kind: KrknOperatorTarget
metadata: {name: live-cluster, namespace: $NS}
spec:
  uuid: live-cluster
  clusterName: live-cluster
  clusterAPIURL: $CLUSTER_API
  secretType: kubeconfig
  secretUUID: live-target-kubeconfig
EOF
  kubectl patch krknoperatortarget live-cluster -n "$NS" --subresource=status \
    --type=merge -p '{"status":{"ready":true}}' >/dev/null
  # Note: the operator heartbeats its own KrknOperatorTargetProvider; do NOT
  # create a second one or target requests never complete.

  echo
  echo "==> Waiting for API server"
  until curl -s -o /dev/null -m 2 "http://localhost:$API_PORT/api/mcp" -X POST; do sleep 2; done
  echo "Ready. MCP endpoint: http://localhost:$API_PORT/api/mcp"
  echo "Robot token (shown once): $(cat "$TOKEN_FILE")"
  echo
  echo "Quick test:"
  echo "  ./scripts/mcp-live-demo.sh call '{\"name\":\"krkn_whoami\",\"arguments\":{}}'"
}

call() {
  local body
  body=$(python3 -c "
import json,sys
print(json.dumps({'jsonrpc':'2.0','id':1,'method':'tools/call','params':json.loads(sys.argv[1])}))" "$1")
  curl -s -m 120 -X POST "http://localhost:$API_PORT/api/mcp" \
    -H "Authorization: Bearer $(cat "$TOKEN_FILE")" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json, text/event-stream" \
    -d "$body" | python3 -c "
import json,sys
r=json.load(sys.stdin)
res=r.get('result',{})
print('isError:', res.get('isError', False))
print(json.dumps(res.get('structuredContent', res.get('content','')), indent=2))"
}

cleanup() {
  pkill -f "krkn-operator-live -api-port=$API_PORT" 2>/dev/null || true
  kubectl delete namespace "$NS" --wait=false 2>/dev/null || true
  make uninstall >/dev/null 2>&1 || true
  rm -f "$TOKEN_FILE"
  echo "Cleaned up (namespace deletion continues in background)."
}

case "${1:-}" in
  setup) setup ;;
  token) cat "$TOKEN_FILE" ;;
  call) shift; call "$1" ;;
  cleanup) cleanup ;;
  *) echo "Usage (run from the repo root): $0 {setup|token|call '<json>'|cleanup}"; exit 1 ;;
esac
