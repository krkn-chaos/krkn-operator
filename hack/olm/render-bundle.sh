#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 <ocp|kubernetes> <version> <output-dir>" >&2
  exit 2
}

[[ $# -eq 3 ]] || usage

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
profile=$1
version=$2
output_parent=$(dirname "$3")
mkdir -p "$output_parent"
output_dir="$(cd "$output_parent" && pwd)/$(basename "$3")"

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

command -v helm >/dev/null || { echo "helm is required" >&2; exit 1; }
command -v operator-sdk >/dev/null || { echo "operator-sdk is required" >&2; exit 1; }
command -v yq >/dev/null || { echo "yq is required" >&2; exit 1; }

operator_image=${OPERATOR_IMAGE:-krkn-chaos.docker.scarf.sh/krkn-chaos/krkn-operator:${version}}
data_provider_image=${DATA_PROVIDER_IMAGE:-krkn-chaos.docker.scarf.sh/krkn-chaos/krkn-operator-data-provider:${version}}
console_image=${CONSOLE_IMAGE:-krkn-chaos.docker.scarf.sh/krkn-chaos/krkn-operator-console:latest}
min_kube_version=${MIN_KUBE_VERSION:-1.36.0}
export OPERATOR_IMAGE="$operator_image"
export DATA_PROVIDER_IMAGE="$data_provider_image"
export CONSOLE_IMAGE="$console_image"
export EXAMPLES_FILE="$repo_root/config/olm/examples.yaml"
export MIN_KUBE_VERSION="$min_kube_version"

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
  --output-dir "$render_dir" >/dev/null

# Helm does not render the chart's CRD directory through templates. Bundle
# generation still needs those CRDs alongside the rendered workload resources.
mkdir -p "$render_dir/krkn-operator/crds"
cp "$repo_root"/charts/krkn-operator/crds/*.yaml "$render_dir/krkn-operator/crds/"

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
  operator-sdk generate bundle "${bundle_args[@]}"
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

csv_file="$output_dir/manifests/krkn-operator.clusterserviceversion.yaml"
yq -i \
  '.metadata.annotations.containerImage = strenv(OPERATOR_IMAGE) |
   .metadata.annotations."alm-examples" = (load(strenv(EXAMPLES_FILE)) | to_json) |
   .spec.minKubeVersion = strenv(MIN_KUBE_VERSION) |
   .spec.relatedImages = [
     {"name": "krkn-operator", "image": strenv(OPERATOR_IMAGE)},
     {"name": "krkn-operator-data-provider", "image": strenv(DATA_PROVIDER_IMAGE)},
     {"name": "krkn-operator-console", "image": strenv(CONSOLE_IMAGE)}
   ]' \
  "$csv_file"

operator-sdk bundle validate "$output_dir"
