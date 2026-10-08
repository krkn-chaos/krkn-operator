#!/usr/bin/env bash
set -euo pipefail

command -v helm >/dev/null || { echo "helm is required" >&2; exit 1; }
command -v yq >/dev/null || { echo "yq is required" >&2; exit 1; }

chart_source_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../charts/krkn-operator" && pwd)
repo_root=$(cd "$chart_source_dir/../.." && pwd)
work_dir=$(mktemp -d "${TMPDIR:-/tmp}/krkn-chart-exposure.XXXXXX")
trap 'rm -rf "$work_dir"' EXIT
chart_dir="$work_dir/chart"
mkdir -p "$chart_dir"
cp -R "$chart_source_dir/." "$chart_dir/"
cp "$repo_root"/config/crd/bases/*.yaml "$chart_dir/crds/"

helm template krkn-operator "$chart_dir" > "$work_dir/base.yaml"
if yq -e 'select(.kind == "Job" and .metadata.name == "krkn-operator-crd-sync")' \
  "$work_dir/base.yaml" >/dev/null; then
  echo "CRD migration hook must only render for upgrades" >&2
  exit 1
fi

helm show crds "$chart_dir" > "$work_dir/crds.yaml"
yq -e 'select(.kind == "CustomResourceDefinition")' \
  "$work_dir/crds.yaml" >/dev/null
yq -e 'select(.kind == "CustomResourceDefinition" and .metadata.name == "krkncategories.krkn.krkn-chaos.dev") | .metadata.labels."app.kubernetes.io/name" == "krkn-operator"' \
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
yq -r 'select(.kind == "Job") | .spec.template.spec.containers[] | select(.name == "crd-sync") | .env[] | select(.name == "POD_NAMESPACE") | .value' \
  "$work_dir/upgrade.yaml" > "$work_dir/pod-namespaces.txt"
grep -Fxq 'default' "$work_dir/pod-namespaces.txt"
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
  --api-versions security.openshift.io/v1/SecurityContextConstraints > "$work_dir/olm-ocp.yaml"

if yq -e 'select(.kind == "Secret" and .metadata.name == "krkn-operator-jwt")' \
  "$work_dir/olm-ocp.yaml" >/dev/null; then
  echo "OLM profile must bootstrap the JWT Secret instead of rendering it" >&2
  exit 1
fi

if grep -Eq 'scenario-runner|krkn-scenario-runner' "$work_dir/olm-ocp.yaml"; then
  echo "OLM profile must not render static scenario-runner resources" >&2
  exit 1
fi

if grep -Eq 'krkn-operator-ai-service|krkn-operator-krkn-ai-orchestrator|krknairuns.krkn.krkn-chaos.dev' "$work_dir/olm-ocp.yaml"; then
  echo "OLM profile must not expose the Helm-only Krkn-AI beta" >&2
  exit 1
fi

helm template krkn-operator "$chart_dir" \
  --set console.ingress.enabled=true \
  --set console.ingress.hostname=console.example.test \
  --set console.ingress.tls.enabled=true \
  --set console.ingress.tls.secretName=console-tls > "$work_dir/ingress-map.yaml"

yq -e 'select(.kind == "Ingress") | .spec.tls[0].secretName == "console-tls" and .spec.rules[0].host == "console.example.test"' \
  "$work_dir/ingress-map.yaml" >/dev/null

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

yq -e 'select(.kind == "Route") | .spec.host == "console.apps.example.test" and .spec.to.name == "krkn-operator-console"' \
  "$work_dir/route.yaml" >/dev/null

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

echo "Ingress and Route rendering checks passed"
