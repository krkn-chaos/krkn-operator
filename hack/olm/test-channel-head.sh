#!/usr/bin/env bash

set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
fixture="$script_dir/testdata/catalog-template-basic.yaml"
ambiguous_fixture="$script_dir/testdata/catalog-template-ambiguous.yaml"
selector="$script_dir/channel-head.sh"

actual_head=$(bash "$selector" "$fixture" krkn-operator stable-ocp)
[[ "$actual_head" == krkn-operator.v1.1.0 ]]
if bash "$selector" "$fixture" krkn-operator stable-missing >/dev/null 2>&1; then
  echo "missing channel was accepted" >&2
  exit 1
fi
if bash "$selector" "$ambiguous_fixture" krkn-operator stable-ocp >/dev/null 2>&1; then
  echo "ambiguous channel was accepted" >&2
  exit 1
fi

echo "catalog channel head tests passed"
