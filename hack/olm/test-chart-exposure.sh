#!/usr/bin/env bash
set -euo pipefail

command -v helm >/dev/null || { echo "helm is required" >&2; exit 1; }
command -v yq >/dev/null || { echo "yq is required" >&2; exit 1; }

chart_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../charts/krkn-operator" && pwd)
work_dir=$(mktemp -d "${TMPDIR:-/tmp}/krkn-chart-exposure.XXXXXX")
trap 'rm -rf "$work_dir"' EXIT

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
  --api-versions route.openshift.io/v1/Route \
  --set console.route.enabled=true \
  --set console.route.hostname=console.apps.example.test > "$work_dir/route.yaml"

yq -e 'select(.kind == "Route") | .spec.host == "console.apps.example.test" and .spec.to.name == "krkn-operator-console"' \
  "$work_dir/route.yaml" >/dev/null

echo "Ingress and Route rendering checks passed"
