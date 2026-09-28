#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
submit_script=(bash "$script_dir/open-kubernetes-operators-pr.sh")
test_dir=$(mktemp -d "${TMPDIR:-/tmp}/kubernetes-operators-pr-test.XXXXXX")
trap 'rm -rf "$test_dir"' EXIT

expect_failure() {
  local expected=$1
  shift
  local output

  if output=$(env -u KUBERNETES_OPERATORS_FORK -u GH_TOKEN "$@" 2>&1); then
    echo "expected command to fail: $*" >&2
    exit 1
  fi
  grep -Fq "$expected" <<<"$output" || {
    echo "expected failure message '$expected', got:" >&2
    echo "$output" >&2
    exit 1
  }
}

expect_failure "invalid release version" "${submit_script[@]}" invalid "$test_dir/missing"

mkdir -p "$test_dir/metadata"
touch "$test_dir/metadata/annotations.yaml"
expect_failure "rendered Kubernetes bundle is incomplete" "${submit_script[@]}" 1.1.0 "$test_dir"

mkdir -p "$test_dir/manifests"
touch "$test_dir/manifests/operator.yaml"
expect_failure "does not contain a ClusterServiceVersion" "${submit_script[@]}" 1.1.0 "$test_dir"

cat > "$test_dir/manifests/operator.clusterserviceversion.yaml" <<'EOF'
kind: ClusterServiceVersion
EOF
expect_failure "KUBERNETES_OPERATORS_FORK must be configured" "${submit_script[@]}" 1.1.0 "$test_dir"

echo "Kubernetes catalog submission validation tests passed"
