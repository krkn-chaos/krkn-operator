#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 <ocp|kubernetes> <version> <output-dir>" >&2
  exit 2
}

[[ $# -eq 3 ]] || usage
[[ -n "$3" ]] || { echo "output-dir must not be empty" >&2; exit 2; }
[[ "$3" != "/" && "$3" != "." && "$3" != ".." ]] || {
  echo "output-dir is a protected path: $3" >&2
  exit 2
}
output_basename=$(basename "$3")
[[ "$output_basename" != "." && "$output_basename" != ".." ]] || {
  echo "output-dir has a protected basename: $3" >&2
  exit 2
}

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
source "$repo_root/hack/olm/release-channel.sh"
profile=$1
version=$2
output_parent=$(dirname "$3")
mkdir -p "$output_parent"
output_dir="$(cd "$output_parent" && pwd)/$(basename "$3")"
[[ "$output_dir" != "/" && "$output_dir" != "$repo_root" ]] || {
  echo "refusing to remove protected output directory: $output_dir" >&2
  exit 2
}

case "$profile" in
  ocp)
    values_file="$repo_root/charts/krkn-operator/values-olm-ocp.yaml"
    channel=stable-ocp
    api_versions=(--api-versions route.openshift.io/v1/Route
      --api-versions security.openshift.io/v1/SecurityContextConstraints)
    ;;
  kubernetes)
    values_file="$repo_root/charts/krkn-operator/values-olm-kubernetes.yaml"
    channel=stable-kubernetes
    api_versions=()
    ;;
  *)
    echo "unsupported profile: $profile" >&2
    usage
    ;;
esac

channel=$(release_channel_for_version "$channel" "$version")

command -v helm >/dev/null || { echo "helm is required" >&2; exit 1; }
command -v operator-sdk >/dev/null || { echo "operator-sdk is required" >&2; exit 1; }
command -v yq >/dev/null || { echo "yq is required" >&2; exit 1; }
command -v python3 >/dev/null || { echo "python3 is required" >&2; exit 1; }

operator_image=${OPERATOR_IMAGE:-krkn-chaos.docker.scarf.sh/krkn-chaos/krkn-operator:${version}}
data_provider_image=${DATA_PROVIDER_IMAGE:-krkn-chaos.docker.scarf.sh/krkn-chaos/krkn-operator-data-provider:${version}}
console_image=${CONSOLE_IMAGE:-krkn-chaos.docker.scarf.sh/krkn-chaos/krkn-operator-console:latest}
min_kube_version=${MIN_KUBE_VERSION:-1.19.0}
openshift_versions=${OPENSHIFT_VERSIONS:-v4.19-v4.22}
ai_orchestrator_image=${AI_ORCHESTRATOR_IMAGE:-quay.io/krkn-chaos/krkn-ai-operator:latest}
ai_service_image=${AI_SERVICE_IMAGE:-quay.io/krkn-chaos/krkn-ai-service:latest}
if [[ "${USE_IMAGE_DIGESTS:-false}" == "true" ]]; then
  resolve_runtime_digest() {
    local image=$1
    local digest
    if [[ "$image" == *@sha256:* ]]; then
      [[ "$image" =~ @sha256:[[:xdigit:]]{64}$ ]] || {
        echo "invalid sha256 runtime image digest: $image" >&2
        exit 1
      }
      printf '%s\n' "$image"
      return
    fi
    command -v docker >/dev/null || {
      echo "docker is required to resolve runtime image digest: $image" >&2
      exit 1
    }
    digest=$(docker buildx imagetools inspect "$image" | awk '$1 == "Digest:" { print $2; exit }')
    [[ "$digest" =~ ^sha256:[[:xdigit:]]{64}$ ]] || {
      echo "could not resolve a sha256 digest for runtime image: $image" >&2
      exit 1
    }
    printf '%s@%s\n' "$image" "$digest"
  }
  ai_orchestrator_image=$(resolve_runtime_digest "$ai_orchestrator_image")
  ai_service_image=$(resolve_runtime_digest "$ai_service_image")
fi
export AI_ORCHESTRATOR_IMAGE="$ai_orchestrator_image"
export AI_SERVICE_IMAGE="$ai_service_image"
export OPERATOR_IMAGE="$operator_image"
export DATA_PROVIDER_IMAGE="$data_provider_image"
export CONSOLE_IMAGE="$console_image"
export EXAMPLES_FILE="$repo_root/config/olm/examples.yaml"
export MIN_KUBE_VERSION="$min_kube_version"

icon_file=${ICON_FILE:-$repo_root/config/olm/assets/krkn.png}
if [[ -f "$icon_file" ]]; then
  case "${icon_file##*.}" in
    svg) icon_mediatype=image/svg+xml ;;
    png) icon_mediatype=image/png ;;
    jpg|jpeg) icon_mediatype=image/jpeg ;;
    *) echo "unsupported icon format: $icon_file" >&2; exit 1 ;;
  esac
  # Keep the base64 value valid while avoiding a single very long YAML line.
  export ICON_BASE64="$(base64 < "$icon_file" | tr -d '\n' | fold -w 76)"
  export ICON_MEDIATYPE="$icon_mediatype"
else
  export ICON_BASE64=""
  export ICON_MEDIATYPE=""
fi
export OPENSHIFT_VERSIONS="$openshift_versions"

work_dir=$(mktemp -d "${TMPDIR:-/tmp}/krkn-olm.XXXXXX")
trap 'rm -rf "$work_dir"' EXIT
render_dir="$work_dir/render"

helm template krkn-operator "$repo_root/charts/krkn-operator" \
  --namespace krkn-operator-system \
  --values "$values_file" \
  ${api_versions[@]+"${api_versions[@]}"} \
  --set-string "images.operator.image=${operator_image}" \
  --set-string "images.dataProvider.image=${data_provider_image}" \
  --set-string "images.console.image=${console_image}" \
  --set-string "images.aiOrchestrator.image=${ai_orchestrator_image}" \
  --set-string "images.aiService.image=${ai_service_image}" \
  --output-dir "$render_dir" >/dev/null

# Bundles always own the optional Krkn-AI API, while runtime resources remain
# managed by the operator after installation.
mkdir -p "$render_dir/krkn-operator/crds"
cp "$repo_root"/charts/krkn-operator/crds/*.yaml "$render_dir/krkn-operator/crds/"
cp "$repo_root/config/crd/bases/krkn.krkn-chaos.dev_krknairuns.yaml" \
  "$render_dir/krkn-operator/crds/"

# Feed examples to Operator SDK before bundle generation as well as applying
# them to the final CRDs below. This keeps generation quiet and ensures the
# generated CSV sees the same examples that OperatorHub displays.
while IFS= read -r crd_file; do
  [[ "$(yq -r '.kind // ""' "$crd_file")" == "CustomResourceDefinition" ]] || continue
  crd_kind=$(yq -r '.spec.names.kind // ""' "$crd_file")
  [[ -n "$crd_kind" ]] || continue
  export CRD_KIND="$crd_kind"
  crd_example_json=$(yq -o=json -I=2 '[.[] | select(.kind == strenv(CRD_KIND))]' "$EXAMPLES_FILE")
  export CRD_EXAMPLE_JSON="$crd_example_json"
  yq -i \
    '.metadata.annotations = (.metadata.annotations // {}) |
     .metadata.annotations."alm-examples" = strenv(CRD_EXAMPLE_JSON)' \
    "$crd_file"
done < <(find "$render_dir/krkn-operator/crds" -type f -name '*.yaml' -print)

rm -rf "$output_dir"
mkdir -p "$output_dir"

bundle_args=(
  --input-dir "$render_dir/krkn-operator"
  --kustomize-dir "$repo_root/config/olm"
  --output-dir "$output_dir"
  --package krkn-operator
  --version "$version"
  --channels "$channel"
  --default-channel "$channel"
  --overwrite
  --manifests
  --metadata
)

if [[ "${USE_IMAGE_DIGESTS:-false}" == "true" ]]; then
  bundle_args+=(--use-image-digests)
fi

(
  cd "$work_dir"
  operator-sdk generate bundle "${bundle_args[@]}" < /dev/null
)

# Operator SDK writes its generated Dockerfile relative to the working
# directory. Create a portable Dockerfile in the bundle itself so the output
# can be built by a later release job without absolute temp paths.
cat > "$output_dir/bundle.Dockerfile" <<EOF
FROM scratch

LABEL operators.operatorframework.io.bundle.mediatype.v1="registry+v1"
LABEL operators.operatorframework.io.bundle.manifests.v1="manifests/"
LABEL operators.operatorframework.io.bundle.metadata.v1="metadata/"
LABEL operators.operatorframework.io.bundle.package.v1="krkn-operator"
LABEL operators.operatorframework.io.bundle.channels.v1="$channel"
LABEL operators.operatorframework.io.bundle.channel.default.v1="$channel"
LABEL operators.operatorframework.io.metrics.builder="operator-sdk-v1.41.1"
LABEL operators.operatorframework.io.metrics.mediatype.v1="metrics+v1"

COPY manifests/ manifests/
COPY metadata/ metadata/
EOF

if [[ "$profile" == "ocp" ]]; then
  yq -i '.annotations."com.redhat.openshift.versions" = strenv(OPENSHIFT_VERSIONS)' \
    "$output_dir/metadata/annotations.yaml"
  printf 'LABEL com.redhat.openshift.versions="%s"\n' "$openshift_versions" \
    >> "$output_dir/bundle.Dockerfile"
fi

csv_file="$output_dir/manifests/krkn-operator.clusterserviceversion.yaml"
resolved_operator_image="$OPERATOR_IMAGE"
resolved_data_provider_image="$DATA_PROVIDER_IMAGE"
resolved_console_image="$CONSOLE_IMAGE"
resolved_ai_orchestrator_image="$AI_ORCHESTRATOR_IMAGE"
resolved_ai_service_image="$AI_SERVICE_IMAGE"
if [[ "${USE_IMAGE_DIGESTS:-false}" == "true" ]]; then
  resolved_operator_image=$(yq -r '.spec.install.spec.deployments[] | select(.name == "krkn-operator-operator") | .spec.template.spec.containers[] | select(.name == "manager") | .image' "$csv_file")
  resolved_data_provider_image=$(yq -r '.spec.install.spec.deployments[] | select(.name == "krkn-operator-operator") | .spec.template.spec.containers[] | select(.name == "data-provider") | .image' "$csv_file")
  resolved_console_image=$(yq -r '.spec.install.spec.deployments[] | select(.name == "krkn-operator-console") | .spec.template.spec.containers[] | select(.name == "console") | .image' "$csv_file")
  resolved_ai_orchestrator_image=$(yq -r '.spec.install.spec.deployments[] | select(.name == "krkn-operator-operator") | .spec.template.spec.containers[] | select(.name == "manager") | .env[] | select(.name == "KRKNAI_ORCHESTRATOR_IMAGE") | .value' "$csv_file")
  resolved_ai_service_image=$(yq -r '.spec.install.spec.deployments[] | select(.name == "krkn-operator-operator") | .spec.template.spec.containers[] | select(.name == "manager") | .env[] | select(.name == "KRKNAI_SERVICE_IMAGE") | .value' "$csv_file")
  for image in "$resolved_operator_image" "$resolved_data_provider_image" "$resolved_console_image" "$resolved_ai_orchestrator_image" "$resolved_ai_service_image"; do
    [[ "$image" =~ @sha256:[[:xdigit:]]{64}$ ]] || { echo "digest mode produced a mutable image: $image" >&2; exit 1; }
  done
fi
export RESOLVED_OPERATOR_IMAGE="$resolved_operator_image"
export RESOLVED_DATA_PROVIDER_IMAGE="$resolved_data_provider_image"
export RESOLVED_CONSOLE_IMAGE="$resolved_console_image"
export RESOLVED_AI_ORCHESTRATOR_IMAGE="$resolved_ai_orchestrator_image"
export RESOLVED_AI_SERVICE_IMAGE="$resolved_ai_service_image"
yq -i \
  '(.spec.install.spec.deployments[] | select(.name == "krkn-operator-operator") |
    .spec.template.spec.containers[] | select(.name == "manager") |
    .env[] | select(.name == "KRKN_ROLLBACK_INSPECTOR_IMAGE") | .value) = strenv(RESOLVED_OPERATOR_IMAGE)' \
  "$csv_file"
yq -i \
  '.metadata.annotations.containerImage = strenv(RESOLVED_OPERATOR_IMAGE) |
   .metadata.annotations."alm-examples" = (load(strenv(EXAMPLES_FILE)) | to_json) |
   .spec.minKubeVersion = strenv(MIN_KUBE_VERSION) |
   .spec.relatedImages = [
     {"name": "krkn-operator", "image": strenv(RESOLVED_OPERATOR_IMAGE)},
     {"name": "krkn-operator-data-provider", "image": strenv(RESOLVED_DATA_PROVIDER_IMAGE)},
     {"name": "krkn-operator-console", "image": strenv(RESOLVED_CONSOLE_IMAGE)},
     {"name": "krkn-ai-orchestrator", "image": strenv(RESOLVED_AI_ORCHESTRATOR_IMAGE)},
     {"name": "krkn-ai-service", "image": strenv(RESOLVED_AI_SERVICE_IMAGE)}
   ] |
   .spec.install.spec.permissions += [{
     "serviceAccountName": "krkn-operator",
     "rules": [{"apiGroups": [""], "resources": ["serviceaccounts"], "verbs": ["create", "get", "list", "watch"]}]
   }] |
   .spec.install.spec.clusterPermissions += [{
     "serviceAccountName": "krkn-operator",
     "rules": [{"apiGroups": ["rbac.authorization.k8s.io"], "resources": ["clusterroles"], "verbs": ["create", "get", "list", "watch"]}]
   }, {
     "serviceAccountName": "krkn-operator",
     "rules": [{"apiGroups": ["rbac.authorization.k8s.io"], "resources": ["clusterrolebindings"], "verbs": ["create", "get", "list", "watch"]}]
   }, {
     "serviceAccountName": "krkn-operator",
     "rules": [{"apiGroups": ["rbac.authorization.k8s.io"], "resources": ["clusterroles"], "resourceNames": ["krkn-operator-scenario-runner"], "verbs": ["bind", "escalate"]}]
   }, {
     "serviceAccountName": "krkn-operator",
     "rules": [
       {"apiGroups": [""], "resources": ["nodes"], "verbs": ["get", "list", "watch", "patch", "update"]},
       {"apiGroups": [""], "resources": ["pods"], "verbs": ["create", "delete", "get", "list", "patch", "update", "watch"]},
       {"apiGroups": [""], "resources": ["pods/log", "pods/exec"], "verbs": ["create", "delete", "get", "list", "patch", "update", "watch"]},
       {"apiGroups": [""], "resources": ["services"], "verbs": ["get", "list", "watch"]}
     ]
   }]' \
  "$csv_file"

# OLM expects each owned API to have a human-readable description and an
# example annotation on the corresponding CRD. The CRD schemas already carry
# the authoritative descriptions; copy them into the generated CSV and add a
# per-kind example to the rendered CRD instead of maintaining a second list.
for crd_file in "$output_dir"/manifests/*.yaml; do
  [[ -f "$crd_file" ]] || continue
  [[ "$(yq -r '.kind // ""' "$crd_file")" == "CustomResourceDefinition" ]] || continue

  crd_name=$(yq -r '.metadata.name // ""' "$crd_file")
  crd_kind=$(yq -r '.spec.names.kind // ""' "$crd_file")
  crd_description=$(yq -r '.spec.versions[0].schema.openAPIV3Schema.description // ""' "$crd_file")
  [[ -n "$crd_name" && -n "$crd_kind" ]] || continue

  case "$crd_kind" in
    KrknCategory) crd_display_name="Krkn Category" ;;
    KrknComponent) crd_display_name="Krkn Component" ;;
    KrknGraphRun) crd_display_name="Krkn Graph Run" ;;
    KrknOperatorTargetProviderConfig) crd_display_name="Krkn Operator Target Provider Config" ;;
    KrknOperatorTargetProvider) crd_display_name="Krkn Operator Target Provider" ;;
    KrknOperatorTarget) crd_display_name="Krkn Operator Target" ;;
    KrknScenarioRun) crd_display_name="Krkn Scenario Run" ;;
    KrknWorkspace) crd_display_name="Krkn Workspace" ;;
    KrknTargetRequest) crd_display_name="Krkn Target Request" ;;
    KrknUserGroup) crd_display_name="Krkn User Group" ;;
    KrknUser) crd_display_name="Krkn User" ;;
    KrknAIRun) crd_display_name="Krkn AI Run" ;;
    *) crd_display_name="$crd_kind" ;;
  esac

  export CRD_NAME="$crd_name"
  export CRD_KIND="$crd_kind"
  export CRD_DESCRIPTION="$crd_description"
  export CRD_DISPLAY_NAME="$crd_display_name"
  export CRD_EXAMPLE_JSON="$(yq -o=json -I=2 '[.[] | select(.kind == strenv(CRD_KIND))]' "$EXAMPLES_FILE")"

  yq -i \
    '.metadata.annotations = (.metadata.annotations // {}) |
     .metadata.annotations."alm-examples" = strenv(CRD_EXAMPLE_JSON)' \
    "$crd_file"

  yq -i \
    '(.spec.customresourcedefinitions.owned[] |
      select(.name == strenv(CRD_NAME)) | .displayName) = strenv(CRD_DISPLAY_NAME) |
     (.spec.customresourcedefinitions.owned[] |
      select(.name == strenv(CRD_NAME)) | .description) = strenv(CRD_DESCRIPTION)' \
    "$csv_file"
done

if [[ "$profile" == "ocp" ]]; then
  yq -i '.spec.install.spec.clusterPermissions += [{
    "serviceAccountName": "krkn-operator",
    "rules": [{"apiGroups": ["rbac.authorization.k8s.io"], "resources": ["clusterroles"], "resourceNames": ["system:openshift:scc:anyuid"], "verbs": ["bind"]}]
  }, {
    "serviceAccountName": "krkn-operator",
    "rules": [{"apiGroups": ["rbac.authorization.k8s.io"], "resources": ["clusterrolebindings"], "verbs": ["update"]}]
  }]' "$csv_file"
fi

if [[ -n "$ICON_BASE64" ]]; then
  yq -i '.spec.icon = [{"base64data": strenv(ICON_BASE64), "mediatype": strenv(ICON_MEDIATYPE)}]' \
    "$csv_file"
fi

# Keep generated YAML compatible with the catalog linter without changing scalar
# values. Literal blocks retain existing newlines; long single-line values use
# double-quoted YAML continuations, including deeply indented CEL expressions.
# Explicit document markers make each file a standalone YAML document.
while IFS= read -r yaml_file; do
  yq -i '
    (... | select(tag == "!!str" and length > 80 and contains("\n") == false)) style="double" |
    (... | select(tag == "!!str" and length > 180 and contains("\n"))) style="literal"
  ' "$yaml_file"
  python3 - "$yaml_file" <<'PY'
import re
import sys
from pathlib import Path


def wrap_quoted_line(line, width=180):
    if len(line) <= width:
        return line
    match = re.fullmatch(
        r'(\s*(?:-\s+)?(?:[^"\n]*:\s+)?)("(?:[^"\\]|\\.)*")(\s*(?:#.*)?)',
        line,
    )
    if not match:
        return line
    prefix, quoted, suffix = match.groups()
    indent = re.match(r"\s*", prefix).group() + "  "
    tokens = re.findall(
        r"\\(?:x[0-9a-fA-F]{2}|u[0-9a-fA-F]{4}|U[0-9a-fA-F]{8}|.)|[^\\]",
        quoted[1:-1],
    )
    lines = []
    current = prefix + '"'
    for token in tokens:
        if len(current) + len(token) > width - 1:
            lines.append(current + "\\")
            current = indent
        # YAML discards indentation after an escaped newline.
        if current == indent and token == " ":
            token = r"\x20"
        current += token
    lines.append(current + '"' + suffix)
    return "\n".join(lines)


path = Path(sys.argv[1])
lines = path.read_text(encoding="utf-8").splitlines()
path.write_text("\n".join(wrap_quoted_line(line) for line in lines) + "\n", encoding="utf-8")
PY
  if [[ "$(head -n 1 "$yaml_file")" != "---" ]]; then
    normalized_file="$yaml_file.normalized"
    {
      printf '%s\n' '---'
      cat "$yaml_file"
    } > "$normalized_file"
    mv "$normalized_file" "$yaml_file"
  fi
done < <(find "$output_dir" -type f -name '*.yaml' -print)

operator-sdk bundle validate "$output_dir"
