#!/usr/bin/env bash

set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
selector="$script_dir/select-ai-release.sh"
releases_fixture="$script_dir/testdata/ai-releases.json"
single_release_fixture="$script_dir/testdata/ai-single-release.json"
metadata_fixture="$script_dir/testdata/ai-image-metadata.json"
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT
mkdir -p "$tmp_dir/bin"

cat > "$tmp_dir/bin/gh" <<'GH_SHIM'
#!/usr/bin/env bash
set -euo pipefail
[[ "${1:-}" == api ]] || { echo "unexpected gh command" >&2; exit 2; }
shift
filter=
while [[ $# -gt 0 ]]; do
  if [[ "$1" == --jq ]]; then
    filter=$2
    shift 2
  else
    shift
  fi
done
[[ -n "$filter" && -n "${GH_RELEASES_FIXTURE:-}" ]] || { echo "missing gh fixture" >&2; exit 2; }
jq -r "$filter" "$GH_RELEASES_FIXTURE"
GH_SHIM

cat > "$tmp_dir/bin/docker" <<'DOCKER_SHIM'
#!/usr/bin/env bash
set -euo pipefail
[[ $# -eq 6 && "$1 $2 $3" == "buildx imagetools inspect" && "$4" == --format ]] || {
  echo "unexpected docker command" >&2
  exit 2
}
format=$5
image_ref=$6
case "$image_ref" in
  quay.io/krkn-chaos/krkn-ai-operator:*)
    side=operator
    tag=${image_ref#quay.io/krkn-chaos/krkn-ai-operator:}
    ;;
  quay.io/krkn-chaos/krkn-ai-service:*)
    side=service
    tag=${image_ref#quay.io/krkn-chaos/krkn-ai-service:}
    ;;
  *) echo "unexpected image ref: $image_ref" >&2; exit 2 ;;
esac
tag=${tag%%@*}
record=$(jq -ce --arg scenario "$AI_IMAGE_CASE" --arg side "$side" --arg tag "$tag" \
  '.[$scenario][$side][$tag] // .["compatible-pair"][$side][$tag] // error("image metadata fixture not found")' "$AI_IMAGE_METADATA")
if jq -e 'has("error")' <<< "$record" >/dev/null; then
  jq -r '.error' <<< "$record" >&2
  exit 1
fi
case "$format" in
  '{{json .Manifest}}')
    jq -ce '.manifest // error("manifest fixture missing")' <<< "$record"
    ;;
  '{{json .Image}}')
    [[ "$image_ref" == *@* ]] || { echo "config inspection was not digest-pinned" >&2; exit 2; }
    requested_digest=${image_ref##*@}
    fixture_digest=$(jq -r '.manifest.digest // empty' <<< "$record")
    [[ "$requested_digest" == "$fixture_digest" ]] || {
      echo "config inspection used a different digest" >&2
      exit 2
    }
    jq -ce '.image // error("image config fixture missing")' <<< "$record"
    ;;
  *) echo "unexpected buildx format: $format" >&2; exit 2 ;;
esac
DOCKER_SHIM
chmod +x "$tmp_dir/bin/gh" "$tmp_dir/bin/docker"
PATH="$tmp_dir/bin:$PATH"
export PATH
AI_IMAGE_METADATA=$metadata_fixture
export AI_IMAGE_METADATA

selection=$(AI_IMAGE_CASE=compatible-pair bash "$selector" "$releases_fixture")
[[ $(jq -r '.release' <<< "$selection") == v0.1.10 ]]
[[ $(jq -r '.orchestratorImage' <<< "$selection") == \
  quay.io/krkn-chaos/krkn-ai-operator@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b852 ]]
[[ $(jq -r '.serviceImage' <<< "$selection") == \
  quay.io/krkn-chaos/krkn-ai-service@sha256:7007b387ccd52bd42a050f2e8020e56e64622c9269bf7bbe257b326fe99daf12 ]]

# The no-fixture path must fetch releases through gh and still ask the generic selector for all candidates.
selection=$(GH_RELEASES_FIXTURE="$releases_fixture" AI_IMAGE_CASE=compatible-pair bash "$selector")
[[ $(jq -r '.release' <<< "$selection") == v0.1.10 ]]

expect_failure() {
  local scenario=$1 fixture=$2 output
  if output=$(AI_IMAGE_CASE="$scenario" bash "$selector" "$fixture" 2>"$tmp_dir/stderr"); then
    echo "AI selector unexpectedly succeeded for $scenario" >&2
    exit 1
  fi
  [[ -z "$output" ]] || { echo "AI selector returned a selection on failure" >&2; exit 1; }
}

for scenario in service-missing-label service-null-label operator-wrong-major; do
  selection=$(AI_IMAGE_CASE="$scenario" bash "$selector" "$releases_fixture")
  [[ $(jq -r '.release' <<< "$selection") == v0.1.10 ]]
done

expect_failure no-compatible-pair "$single_release_fixture"
# An older compatible pair exists: malformed metadata and registry failures must
# abort publication rather than silently selecting that older pair.
expect_failure malformed-labels "$releases_fixture"
expect_failure registry-failure "$releases_fixture"

echo "AI release selection tests passed"
