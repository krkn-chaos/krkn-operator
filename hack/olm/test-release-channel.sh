#!/usr/bin/env bash

set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
source "$script_dir/release-channel.sh"

[[ "$(release_channel_for_version stable-ocp 1.0.8)" == stable-ocp ]]
[[ "$(release_channel_for_version stable-ocp 1.1.0-rc.2)" == stable-ocp-1.1 ]]
[[ "$(release_channel_for_version stable-kubernetes 2.0.0)" == stable-kubernetes-2.0 ]]
if release_channel_for_version stable-ocp bad-version >/dev/null 2>&1; then
  echo "invalid release version was accepted" >&2
  exit 1
fi

echo "release channel tests passed"
