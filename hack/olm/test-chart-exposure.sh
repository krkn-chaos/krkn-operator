#!/usr/bin/env bash
set -euo pipefail

command -v helm >/dev/null || { echo "helm is required" >&2; exit 1; }
command -v yq >/dev/null || { echo "yq is required" >&2; exit 1; }

chart_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../charts/krkn-operator" && pwd)
work_dir=$(mktemp -d "${TMPDIR:-/tmp}/krkn-chart-exposure.XXXXXX")
trap 'rm -rf "$work_dir"' EXIT

helm template krkn-operator "$chart_dir" > "$work_dir/base.yaml"

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
  --set console.route.enabled=true \
  --set console.route.hostname="" \
  --set console.route.host=legacy.apps.example.test > "$work_dir/route-legacy.yaml"

yq -e 'select(.kind == "Route") | .spec.host == "legacy.apps.example.test"' \
  "$work_dir/route-legacy.yaml" >/dev/null

echo "Ingress and Route rendering checks passed"
