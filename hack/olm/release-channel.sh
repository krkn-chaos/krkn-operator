#!/usr/bin/env bash

# Sourced by the OLM renderer and catalog submitter to derive the channel for a
# release line. The 1.0.x line keeps its existing channel; later lines use a
# major.minor suffix, for example: release_channel_for_version stable-ocp 1.1.0.
release_channel_for_version() {
  local base_channel=$1
  local version=$2
  [[ "$version" =~ ^([0-9]+)\.([0-9]+)\.[0-9]+([-.+][0-9A-Za-z.-]+)?$ ]] || return 2
  if [[ "${BASH_REMATCH[1]}.${BASH_REMATCH[2]}" == "1.0" ]]; then
    printf '%s\n' "$base_channel"
  else
    printf '%s-%s.%s\n' "$base_channel" "${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}"
  fi
}
