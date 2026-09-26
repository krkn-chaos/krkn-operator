#!/usr/bin/env bash

set -euo pipefail

[[ $# -eq 3 ]] || {
  echo "usage: $0 <catalog-template> <package> <channel>" >&2
  exit 2
}

catalog_template=$1
package_name=$2
channel_name=$3

export OLM_PACKAGE_NAME="$package_name"
export OLM_CHANNEL_NAME="$channel_name"

channel_entries=()
while IFS= read -r entry; do
  channel_entries+=("$entry")
done < <(yq -r \
  '.entries[]
   | select(.schema == "olm.channel" and .package == strenv(OLM_PACKAGE_NAME) and .name == strenv(OLM_CHANNEL_NAME))
   | .entries[]
   | .name' \
  "$catalog_template")

replaced_entries=()
while IFS= read -r replaced; do
  replaced_entries+=("$replaced")
done < <(yq -r \
  '.entries[]
   | select(.schema == "olm.channel" and .package == strenv(OLM_PACKAGE_NAME) and .name == strenv(OLM_CHANNEL_NAME))
   | .entries[]
   | .replaces // ""' \
  "$catalog_template")

heads=()
for entry in "${channel_entries[@]}"; do
  referenced=false
  for replaced in ${replaced_entries[*]-}; do
    if [[ "$entry" == "$replaced" ]]; then
      referenced=true
      break
    fi
  done
  [[ "$referenced" == true ]] || heads+=("$entry")
done

if [[ "${#heads[@]}" -ne 1 ]]; then
  echo "expected exactly one head for $package_name/$channel_name, found ${#heads[@]}" >&2
  exit 1
fi

printf '%s\n' "${heads[0]}"
