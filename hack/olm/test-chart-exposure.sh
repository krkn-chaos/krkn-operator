#!/usr/bin/env bash
set -euo pipefail

command -v helm >/dev/null || { echo "helm is required" >&2; exit 1; }
command -v yq >/dev/null || { echo "yq is required" >&2; exit 1; }

chart_source_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../charts/krkn-operator" && pwd)
repo_root=$(cd "$chart_source_dir/../.." && pwd)
test -f "$chart_source_dir/crds/krkn.krkn-chaos.dev_krknworkspaces.yaml"
work_dir=$(mktemp -d "${TMPDIR:-/tmp}/krkn-chart-exposure.XXXXXX")
trap 'rm -rf "$work_dir"' EXIT
chart_dir="$work_dir/chart"
mkdir -p "$chart_dir"
cp -R "$chart_source_dir/." "$chart_dir/"
while IFS= read -r crd_file; do
  [[ "$(basename "$crd_file")" == "krkn.krkn-chaos.dev_krknairuns.yaml" ]] && continue
  cp "$crd_file" "$chart_dir/crds/"
done < <(find "$repo_root/config/crd/bases" -maxdepth 1 -type f -name '*.yaml' -print)

helm template krkn-operator "$chart_dir" > "$work_dir/base.yaml"
if yq -e 'select(.kind == "Job" and .metadata.name == "krkn-operator-crd-sync")' \
  "$work_dir/base.yaml" >/dev/null; then
  echo "CRD migration hook must only render for upgrades" >&2
  exit 1
fi

helm show crds "$chart_dir" > "$work_dir/crds.yaml"
yq -e 'select(.kind == "CustomResourceDefinition")' \
  "$work_dir/crds.yaml" >/dev/null
if yq -e 'select(.kind == "CustomResourceDefinition" and .metadata.name == "krknairuns.krkn.krkn-chaos.dev")' \
  "$work_dir/crds.yaml" >/dev/null; then
  echo "Krkn-AI CRD must remain opt-in for standard Helm installs" >&2
  exit 1
fi
yq -e 'select(.kind == "CustomResourceDefinition" and .metadata.name == "krkncategories.krkn.krkn-chaos.dev") | .metadata.labels."app.kubernetes.io/name" == "krkn-operator"' \
  "$work_dir/crds.yaml" >/dev/null
yq -e 'select(.kind == "CustomResourceDefinition" and .metadata.name == "krknworkspaces.krkn.krkn-chaos.dev") | .metadata.labels."app.kubernetes.io/name" == "krkn-operator"' \
  "$work_dir/crds.yaml" >/dev/null
if ! yq -e '[select(.kind == "CustomResourceDefinition" and .metadata.labels."app.kubernetes.io/name" != "krkn-operator")] | length == 0' \
  "$work_dir/crds.yaml" >/dev/null; then
  echo "all chart CRDs must have the operator label for uninstall cleanup" >&2
  exit 1
fi

helm template krkn-operator "$chart_dir" --is-upgrade > "$work_dir/upgrade.yaml"
yq -r 'select(.kind == "Job" and .metadata.annotations."helm.sh/hook" == "pre-upgrade") | .spec.template.spec.containers[] | select(.name == "crd-sync") | .args[]' \
  "$work_dir/upgrade.yaml" > "$work_dir/crd-stage-args.txt"
yq -r 'select(.kind == "Job" and .metadata.annotations."helm.sh/hook" == "post-upgrade") | .spec.template.spec.containers[] | select(.name == "crd-sync") | .args[]' \
  "$work_dir/upgrade.yaml" > "$work_dir/crd-sync-args.txt"
grep -Fxq -- '--stage-crd-migration' "$work_dir/crd-stage-args.txt"
grep -Fxq -- '--sync-crds' "$work_dir/crd-sync-args.txt"
grep -Fxq -- '--crd-dir=/crds' "$work_dir/crd-sync-args.txt"
yq -r 'select(.kind == "Role" and .metadata.name == "krkn-operator-crd-sync-runs") | .rules[] | [.apiGroups[0], (.resources | join(",")), (.verbs | join(","))] | join("|")' \
  "$work_dir/upgrade.yaml" > "$work_dir/crd-sync-rules.txt"
grep -Fxq 'krkn.krkn-chaos.dev|krknscenarioruns,krkngraphruns|list,patch' "$work_dir/crd-sync-rules.txt"
grep -Fxq 'krkn.krkn-chaos.dev|krknscenarioruns/status,krkngraphruns/status|patch' "$work_dir/crd-sync-rules.txt"
grep -Fxq 'apps|deployments|get' "$work_dir/crd-sync-rules.txt"
grep -Fxq '|configmaps|get,create,update,delete' "$work_dir/crd-sync-rules.txt"
yq -r 'select(.kind == "Job" and .metadata.annotations."helm.sh/hook" == "post-upgrade") | .spec.template.spec.containers[] | select(.name == "crd-sync") | .env[] | select(.name == "OPERATOR_DEPLOYMENT_NAME") | .value' \
  "$work_dir/upgrade.yaml" | grep -Fxq 'krkn-operator-operator'

yq -e 'select(.kind == "ConfigMap" and .metadata.name == "krkn-operator-console-nginx") | .data["nginx.conf"] | contains("proxy_pass http://krkn-operator-operator:8080;")' \
  "$work_dir/base.yaml" >/dev/null

yq -e 'select(.kind == "Role") | .rules[] | select(.apiGroups[0] == "" and .resources[0] == "events" and .verbs[0] == "create" and .verbs[1] == "patch")' \
  "$work_dir/base.yaml" >/dev/null

if yq -e 'select(.kind == "ConfigMap" and .metadata.name == "krkn-operator-console-nginx") | .data["nginx.conf"] | contains(".svc.cluster.local")' \
  "$work_dir/base.yaml" >/dev/null; then
  echo "console proxy must not hardcode a namespace" >&2
  exit 1
fi

if ! grep -Eq 'scenario-runner|krkn-scenario-runner' "$work_dir/base.yaml"; then
  echo "standard Helm profile must render scenario-runner resources" >&2
  exit 1
fi

if grep -Eq 'krkn-operator-ai-service|krkn-operator-krkn-ai-orchestrator|krknairuns.krkn.krkn-chaos.dev' "$work_dir/base.yaml"; then
  echo "Krkn-AI resources must not render by default" >&2
  exit 1
fi
if ! grep -Fq -- '--enable-krkn-ai=false' "$work_dir/base.yaml"; then
  echo "operator must disable Krkn-AI by default" >&2
  exit 1
fi

helm template krkn-operator "$chart_dir" --set krknAI.enabled=true > "$work_dir/krkn-ai.yaml"
if ! grep -Eq 'krkn-operator-ai-service|krkn-operator-krkn-ai-orchestrator|krknairuns.krkn.krkn-chaos.dev' "$work_dir/krkn-ai.yaml"; then
  echo "enabled Krkn-AI profile must render its resources and CRD" >&2
  exit 1
fi
if ! grep -Fq -- '--enable-krkn-ai=true' "$work_dir/krkn-ai.yaml"; then
  echo "operator must enable Krkn-AI controller when requested" >&2
  exit 1
fi

helm template krkn-operator "$chart_dir" \
  --values "$chart_dir/values-olm-ocp.yaml" \
  --api-versions security.openshift.io/v1/SecurityContextConstraints \
  --set krknAI.enabled=true > "$work_dir/olm-ocp.yaml"

if yq -e 'select(.kind == "Secret" and .metadata.name == "krkn-operator-jwt")' \
  "$work_dir/olm-ocp.yaml" >/dev/null; then
  echo "OLM profile must bootstrap the JWT Secret instead of rendering it" >&2
  exit 1
fi

if grep -Eq 'scenario-runner|krkn-scenario-runner' "$work_dir/olm-ocp.yaml"; then
  echo "OLM profile must not render static scenario-runner resources" >&2
  exit 1
fi

if yq -e 'select((.kind == "Deployment" and .metadata.name == "krkn-operator-ai-service") or
  (.kind == "Service" and .metadata.name == "krkn-operator-ai-service") or
  (.kind == "PersistentVolumeClaim" and .metadata.name == "krkn-operator-ai-service") or
  (.kind == "Secret" and .metadata.name == "krkn-operator-ai-service-token") or
  (.kind == "ServiceAccount" and .metadata.name == "krkn-operator-ai-service") or
  (.kind == "ServiceAccount" and .metadata.name == "krkn-operator-krkn-ai-orchestrator") or
  (.kind == "Role" and .metadata.name == "krkn-operator-krkn-ai-orchestrator") or
  (.kind == "RoleBinding" and .metadata.name == "krkn-operator-krkn-ai-orchestrator") or
  (.kind == "ClusterRoleBinding" and (.metadata.name | test("krkn-operator-ai-orchestrator-anyuid"))) or
  (.kind == "NetworkPolicy" and .metadata.name == "krkn-operator-ai-service"))' \
  "$work_dir/olm-ocp.yaml" >/dev/null; then
  echo "OLM profile must never render runtime-managed Krkn-AI resources" >&2
  exit 1
fi
if yq -e 'select(.kind == "CustomResourceDefinition" and .metadata.name == "krknairuns.krkn.krkn-chaos.dev")' \
  "$work_dir/olm-ocp.yaml" >/dev/null; then
  echo "OLM bundle renderer, not chart templates, owns the optional AI CRD" >&2
  exit 1
fi

if grep -Fq -- '--enable-krkn-ai=' "$work_dir/olm-ocp.yaml"; then
  echo "OLM manager must use KRKNAI_ENABLED instead of a hard-coded argument" >&2
  exit 1
fi

manager_env="$work_dir/olm-manager-env.yaml"
yq -r 'select(.kind == "Deployment" and .metadata.name == "krkn-operator-operator") |
  .spec.template.spec.containers[] | select(.name == "manager") | .env[] |
  select(.name | test("^(KRKN_OLM_ENABLED|KRKN_OLM_OPENSHIFT|KRKNAI_ENABLED|KRKNAI_ORCHESTRATOR_IMAGE|KRKNAI_ORCHESTRATOR_IMAGE_PULL_POLICY|KRKNAI_SERVICE_IMAGE|KRKNAI_SERVICE_IMAGE_PULL_POLICY|KRKNAI_IMAGE_PULL_SECRETS|KRKNAI_STORAGE_EXISTING_CLAIM|KRKNAI_STORAGE_CLASS_NAME|KRKNAI_STORAGE_SIZE|KRKNAI_SERVICE_RESOURCES|KRKNAI_NETWORK_POLICY_ENABLED)$")) |
  [.name, .value] | @tsv' "$work_dir/olm-ocp.yaml" > "$manager_env"
grep -Fq $'KRKN_OLM_ENABLED\ttrue' "$manager_env"
grep -Fq $'KRKN_OLM_OPENSHIFT\ttrue' "$manager_env"
grep -Fq $'KRKNAI_ENABLED\tfalse' "$manager_env"
for env_name in \
  KRKNAI_ORCHESTRATOR_IMAGE \
  KRKNAI_ORCHESTRATOR_IMAGE_PULL_POLICY \
  KRKNAI_SERVICE_IMAGE \
  KRKNAI_SERVICE_IMAGE_PULL_POLICY \
  KRKNAI_IMAGE_PULL_SECRETS \
  KRKNAI_STORAGE_EXISTING_CLAIM \
  KRKNAI_STORAGE_CLASS_NAME \
  KRKNAI_STORAGE_SIZE \
  KRKNAI_SERVICE_RESOURCES \
  KRKNAI_NETWORK_POLICY_ENABLED; do
  grep -Fq "$env_name"$'\t' "$manager_env"
done
if grep -Eq 'KRKNAI_SERVICE_(TOKEN|URL)' "$work_dir/olm-ocp.yaml"; then
  echo "OLM manager must not depend on a runtime token Secret or service URL env" >&2
  exit 1
fi

helm template krkn-operator "$chart_dir" \
  --set console.ingress.enabled=true \
  --set console.ingress.hostname=console.example.test \
  --set console.ingress.tls.enabled=true \
  --set console.ingress.tls.secretName=console-tls > "$work_dir/ingress-map.yaml"
yq -e 'select(.kind == "Ingress") | .spec.tls[0].secretName == "console-tls" and .spec.rules[0].host == "console.example.test" and .metadata.annotations."nginx.ingress.kubernetes.io/proxy-read-timeout" == "360"' \
  "$work_dir/ingress-map.yaml" >/dev/null

helm template krkn-operator "$chart_dir" \
  --set console.ingress.enabled=true \
  --set-json 'console.ingress.annotations={"nginx.ingress.kubernetes.io/proxy-read-timeout":"420"}' > "$work_dir/ingress-override.yaml"

yq -e 'select(.kind == "Ingress") | .metadata.annotations."nginx.ingress.kubernetes.io/proxy-read-timeout" == "420"' \
  "$work_dir/ingress-override.yaml" >/dev/null

helm template krkn-operator "$chart_dir" \
  --set console.ingress.enabled=true \
  --set console.ingress.hostname=console.example.test \
  --set-json 'console.ingress.tls=[{"hosts":["console.example.test"],"secretName":"legacy-console-tls"}]' > "$work_dir/ingress-list.yaml"

yq -e 'select(.kind == "Ingress") | .spec.tls[0].secretName == "legacy-console-tls"' \
  "$work_dir/ingress-list.yaml" >/dev/null

helm template krkn-operator "$chart_dir" \
  --set console.ingress.enabled=true \
  --set-json 'console.ingress.hosts=[{"host":"legacy.example.test","paths":[{"path":"/console","pathType":"Prefix"}]}]' > "$work_dir/ingress-hosts.yaml"

yq -e 'select(.kind == "Ingress") | .spec.rules[0].host == "legacy.example.test" and .spec.rules[0].http.paths[0].path == "/console"' \
  "$work_dir/ingress-hosts.yaml" >/dev/null

helm template krkn-operator "$chart_dir" \
  --api-versions route.openshift.io/v1/Route \
  --set console.route.enabled=true \
  --set console.route.hostname=console.apps.example.test > "$work_dir/route.yaml"

yq -e 'select(.kind == "Route") | .spec.host == "console.apps.example.test" and .spec.to.name == "krkn-operator-console" and .metadata.annotations."haproxy.router.openshift.io/timeout" == "6m"' \
  "$work_dir/route.yaml" >/dev/null

helm template krkn-operator "$chart_dir" \
  --api-versions route.openshift.io/v1/Route \
  --set console.route.enabled=true \
  --set-json 'console.route.annotations={"haproxy.router.openshift.io/timeout":"10m"}' > "$work_dir/route-override.yaml"

yq -e 'select(.kind == "Route") | .metadata.annotations."haproxy.router.openshift.io/timeout" == "10m"' \
  "$work_dir/route-override.yaml" >/dev/null

helm template krkn-operator "$chart_dir" \
  --api-versions route.openshift.io/v1/Route \
  --set console.route.enabled=true > "$work_dir/route-generated.yaml"

if yq -e 'select(.kind == "Route") | has("spec") and (.spec | has("host"))' \
  "$work_dir/route-generated.yaml" >/dev/null; then
  echo "default OpenShift Route must not force a hostname" >&2
  exit 1
fi

helm template krkn-operator "$chart_dir" \
  --api-versions route.openshift.io/v1/Route \
  --set console.route.enabled=true \
  --set console.route.hostname="" \
  --set console.route.host=legacy.apps.example.test > "$work_dir/route-legacy.yaml"

yq -e 'select(.kind == "Route") | .spec.host == "legacy.apps.example.test"' \
  "$work_dir/route-legacy.yaml" >/dev/null

helm template krkn-operator "$chart_dir" \
  --api-versions gateway.networking.k8s.io/v1/HTTPRoute \
  --set console.gateway.enabled=true > "$work_dir/gateway.yaml"

yq -e 'select(.kind == "HTTPRoute") | .spec.rules[0].timeouts.request == "6m" and .spec.rules[0].timeouts.backendRequest == "6m"' \
  "$work_dir/gateway.yaml" >/dev/null

echo "Ingress, Route, and Gateway timeout rendering checks passed"
