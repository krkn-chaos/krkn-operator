#!/usr/bin/env bash

set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
fixture="$script_dir/testdata/catalog-template-basic.yaml"
selector="$script_dir/channel-head.sh"

[[ "$(bash "$selector" "$fixture" krkn-operator stable-ocp)" == krkn-operator.v1.1.0 ]]
[[ -z "$(bash "$selector" "$fixture" krkn-operator stable-missing)" ]]

echo "catalog channel head tests passed"
