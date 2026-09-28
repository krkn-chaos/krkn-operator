#!/usr/bin/env bash

set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
source "$script_dir/release-channel.sh"
source "$script_dir/catalog-channel.sh"

[[ "$(release_channel_for_version stable-ocp 1.0.8)" == stable-ocp ]]
[[ "$(release_channel_for_version stable-ocp 1.1.0-rc.2)" == stable-ocp-1.1 ]]
[[ "$(release_channel_for_version stable-kubernetes 2.0.0)" == stable-kubernetes-2.0 ]]
if release_channel_for_version stable-ocp 1.1. >/dev/null 2>&1 || release_channel_for_version stable-ocp bad-version >/dev/null 2>&1; then
  echo "invalid release version was accepted" >&2
  exit 1
fi

test_dir=$(mktemp -d "${TMPDIR:-/tmp}/test-catalog-channel.XXXXXX")
trap 'rm -rf "$test_dir"' EXIT
catalog_template="$test_dir/basic.yaml"
cp "$script_dir/testdata/catalog-template-basic.yaml" "$catalog_template"
ensure_catalog_channel "$catalog_template" krkn-operator stable-ocp-1.1
[[ "$(yq -r '.entries[] | select(.schema == "olm.channel" and .name == "stable-ocp-1.1") | .entries | length' "$catalog_template")" == 0 ]]
update_default_channel "$catalog_template" krkn-operator stable-ocp-1.1
[[ "$(yq -r '.entries[] | select(.schema == "olm.package" and .name == "krkn-operator") | .defaultChannel' "$catalog_template")" == stable-ocp-1.1 ]]
write_release_config "$test_dir" stable-ocp-1.1 ""
! grep -q '^    replaces:' "$test_dir/release-config.yaml"
write_release_config "$test_dir" stable-ocp-1.1 krkn-operator.v1.1.0
grep -Fxq '    replaces: krkn-operator.v1.1.0' "$test_dir/release-config.yaml"

echo "release channel tests passed"
