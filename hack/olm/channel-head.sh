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

yq -r \
  '.entries[]
   | select(.schema == "olm.channel" and .package == strenv(OLM_PACKAGE_NAME) and .name == strenv(OLM_CHANNEL_NAME))
   | .entries[-1].name // ""' \
  "$catalog_template"
