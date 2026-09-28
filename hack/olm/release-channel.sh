#!/usr/bin/env bash

release_channel_for_version() {
  local base_channel=$1
  local version=$2
  [[ "$version" =~ ^([0-9]+)\.([0-9]+)\. ]] || return 2
  if [[ "${BASH_REMATCH[1]}.${BASH_REMATCH[2]}" == "1.0" ]]; then
    printf '%s\n' "$base_channel"
  else
    printf '%s-%s.%s\n' "$base_channel" "${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}"
  fi
}
