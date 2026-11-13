#!/usr/bin/env bash

set -euo pipefail

usage() {
  echo "usage: $0 [releases.json]" >&2
  exit 2
}

[[ $# -le 1 ]] || usage

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
generic_selector="$script_dir/select-compatible-release.sh"
releases_file=${1:-}
repository=krkn-chaos/krkn-ai
operator_repository=quay.io/krkn-chaos/krkn-ai-operator
service_repository=quay.io/krkn-chaos/krkn-ai-service
required_contract_major=1
contract_label=io.krkn.operator-contract.major

if [[ -n "$releases_file" ]]; then
  if ! release_tags=$(bash "$generic_selector" "$repository" latest-stable "$releases_file" --all); then
    exit 1
  fi
else
  if ! release_tags=$(bash "$generic_selector" "$repository" latest-stable --all); then
    exit 1
  fi
fi

if [[ -z "$release_tags" ]]; then
  echo "no stable AI releases are available to inspect" >&2
  exit 1
fi

inspect_image() {
  local image_ref=$1
  local manifest_json image_json digest compatibility

  if ! manifest_json=$(docker buildx imagetools inspect --format '{{json .Manifest}}' "$image_ref"); then
    echo "failed to inspect image manifest: $image_ref" >&2
    return 1
  fi
  if ! digest=$(jq -er '
    if type == "object" and (.digest | type == "string") and
      (.digest | test("^sha256:[0-9a-f]{64}$")) then
      .digest
    else
      error("manifest is missing a valid sha256 digest")
    end
  ' <<< "$manifest_json"); then
    echo "invalid manifest metadata for image: $image_ref" >&2
    return 1
  fi

  if ! image_json=$(docker buildx imagetools inspect --format '{{json .Image}}' "${image_ref}@${digest}"); then
    echo "failed to inspect digest-pinned image config: ${image_ref}@${digest}" >&2
    return 1
  fi
  if ! compatibility=$(jq -r --arg label "$contract_label" --arg major "$required_contract_major" '
    def runtime_configs:
      if type != "object" then
        error("image config must be an object")
      elif has("config") then
        [.] # Single-platform config.
      elif length > 0 then
        [to_entries[].value] # Buildx multi-platform config map.
      else
        error("image config map is empty")
      end;

    runtime_configs as $configs |
    if any($configs[]; type != "object" or (.config | type != "object")) then
      error("image config map contains an invalid runtime config")
    elif any($configs[]; (.config | has("Labels")) and (.config.Labels != null) and (.config.Labels | type != "object")) then
      error("runtime config labels are malformed")
    else
      all($configs[]; (.config.Labels[$label] // null) == $major)
    end
  ' <<< "$image_json"); then
    echo "invalid image config metadata for digest-pinned image: ${image_ref}@${digest}" >&2
    return 1
  fi

  IMAGE_DIGEST=$digest
  IMAGE_COMPATIBLE=$compatibility
}

while IFS= read -r release; do
  [[ -n "$release" ]] || continue
  image_tag=${release#v}
  operator_tagged_image="$operator_repository:$image_tag"
  service_tagged_image="$service_repository:$image_tag"

  inspect_image "$operator_tagged_image"
  operator_digest=$IMAGE_DIGEST
  operator_compatible=$IMAGE_COMPATIBLE
  inspect_image "$service_tagged_image"
  service_digest=$IMAGE_DIGEST
  service_compatible=$IMAGE_COMPATIBLE

  if [[ "$operator_compatible" != true ]]; then
    echo "skipping stable AI release $release: operator image has a missing or unsupported $contract_label label" >&2
  fi
  if [[ "$service_compatible" != true ]]; then
    echo "skipping stable AI release $release: service image has a missing or unsupported $contract_label label" >&2
  fi
  if [[ "$operator_compatible" == true && "$service_compatible" == true ]]; then
    jq -cn \
      --arg release "$release" \
      --arg orchestratorImage "$operator_repository@$operator_digest" \
      --arg serviceImage "$service_repository@$service_digest" \
      '{release: $release, orchestratorImage: $orchestratorImage, serviceImage: $serviceImage}'
    exit 0
  fi
done <<< "$release_tags"

echo "no stable AI release has a compatible operator and service image pair" >&2
exit 1
