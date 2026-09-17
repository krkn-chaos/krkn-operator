#!/usr/bin/env bash

set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
fixture="$script_dir/testdata/releases.json"
selector="$script_dir/select-compatible-release.sh"

[[ "$(bash "$selector" example/component v1.1.0-beta.1 "$fixture")" == v1.1.0-beta.3 ]]
[[ "$(bash "$selector" example/component v1.1.0-rc1 "$fixture")" == v1.1.0-rc2 ]]
[[ "$(bash "$selector" example/component v1.1.0 "$fixture")" == v1.1.2 ]]
[[ "$(bash "$selector" example/component v2.0.0 "$fixture" 2>/dev/null || true)" == "" ]]

echo "compatible release selection tests passed"
