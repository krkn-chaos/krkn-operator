#!/usr/bin/env bash

set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
fixture="$script_dir/testdata/releases.json"
ai_fixture="$script_dir/testdata/ai-releases.json"
selector="$script_dir/select-compatible-release.sh"

[[ "$(bash "$selector" example/component v1.1.0-beta.1 "$fixture")" == v1.1.0-beta.3 ]]
[[ "$(bash "$selector" example/component v1.1.0-rc1 "$fixture")" == v1.1.0-rc2 ]]
[[ "$(bash "$selector" example/component v1.1.0 "$fixture")" == v1.1.2 ]]
[[ "$(bash "$selector" example/component v1.1.0-beta.1 "$fixture" --all)" == $'v1.1.0-beta.3\nv1.1.0-beta.1' ]]
[[ "$(bash "$selector" example/component v1.1.0-rc1 "$fixture" --all)" == $'v1.1.0-rc2\nv1.1.0-rc1' ]]
[[ "$(bash "$selector" example/component v1.1.0 "$fixture" --all)" == $'v1.1.2\nv1.1.0' ]]
if bash "$selector" example/component v2.0.0 "$fixture" >/dev/null 2>&1; then
  echo "missing compatible release must fail" >&2
  exit 1
fi

[[ "$(bash "$selector" krkn-chaos/krkn-ai latest-stable "$ai_fixture")" == 0.2.0 ]]
[[ "$(bash "$selector" krkn-chaos/krkn-ai latest-stable "$ai_fixture" --all)" == $'0.2.0\nv0.1.10\nv0.1.9\nv0.1.2' ]]
[[ "$(bash "$selector" krkn-chaos/krkn-ai latest-stable \
  <(printf '%s\n' '[{"tag_name":"v0.1.9"},{"tag_name":"v0.1.10"}]'))" == v0.1.10 ]]
if bash "$selector" krkn-chaos/krkn-ai latest-stable \
  <(printf '%s\n' '[{"tag_name":"v1.0.0-rc1","prerelease":true},{"tag_name":"v0.1.2","draft":true}]') \
  >/dev/null 2>&1; then
  echo "missing stable AI release must fail rather than selecting a draft or prerelease" >&2
  exit 1
fi

echo "compatible release selection tests passed"
