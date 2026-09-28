#!/usr/bin/env bash

ensure_catalog_channel() {
  local catalog_template=$1
  local package_name=$2
  local channel_name=$3
  local channel_count

  export OLM_PACKAGE_NAME="$package_name" OLM_CHANNEL_NAME="$channel_name"
  channel_count=$(yq -r '[.entries[] | select(.schema == "olm.channel" and .package == strenv(OLM_PACKAGE_NAME) and .name == strenv(OLM_CHANNEL_NAME))] | length' "$catalog_template")
  if [[ "$channel_count" == 0 ]]; then
    yq -i '.entries += [{"entries": [], "name": strenv(OLM_CHANNEL_NAME), "package": strenv(OLM_PACKAGE_NAME), "schema": "olm.channel"}]' "$catalog_template"
  elif [[ "$channel_count" != 1 ]]; then
    echo "expected at most one $package_name/$channel_name channel entry, found $channel_count" >&2
    return 1
  fi
}

channel_has_entries() {
  local catalog_template=$1
  local package_name=$2
  local channel_name=$3
  export OLM_PACKAGE_NAME="$package_name" OLM_CHANNEL_NAME="$channel_name"
  [[ "$(yq -r '[.entries[] | select(.schema == "olm.channel" and .package == strenv(OLM_PACKAGE_NAME) and .name == strenv(OLM_CHANNEL_NAME) and (.entries | length > 0))] | length' "$catalog_template")" == 1 ]]
}

update_default_channel() {
  local catalog_template=$1
  local package_name=$2
  local channel_name=$3
  export OLM_PACKAGE_NAME="$package_name" OLM_CHANNEL_NAME="$channel_name"
  yq -i '(.entries[] | select(.schema == "olm.package" and .name == strenv(OLM_PACKAGE_NAME)) | .defaultChannel) = strenv(OLM_CHANNEL_NAME)' "$catalog_template"
}

write_release_config() {
  local version_dir=$1
  local channel_name=$2
  local previous_bundle=$3

  cat > "$version_dir/release-config.yaml" <<EOF
---
catalog_templates:
  - template_name: basic.yaml
    channels:
      - $channel_name
EOF
  if [[ -n "$previous_bundle" ]]; then
    printf '    replaces: %s\n' "$previous_bundle" >>"$version_dir/release-config.yaml"
  fi
}
