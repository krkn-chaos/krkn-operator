#!/usr/bin/env bash

set -euo pipefail

usage() {
  echo "usage: $0 <github-owner/repository> <operator-tag|latest-stable> [releases.json] [--all]" >&2
  exit 2
}

[[ $# -ge 2 && $# -le 4 ]] || usage

repository=$1
operator_tag=$2
releases_file=
all=false

case $# in
  3)
    if [[ "$3" == "--all" ]]; then
      all=true
    else
      releases_file=$3
    fi
    ;;
  4)
    releases_file=$3
    [[ "$4" == "--all" ]] || usage
    all=true
    ;;
esac

if [[ "$operator_tag" == "latest-stable" ]]; then
  # Independently versioned components do not share the operator's release line.
  channel=stable
else
  if [[ "$operator_tag" =~ ^v?([0-9]+)\.([0-9]+)\.([0-9]+)(-([0-9A-Za-z.-]+))?$ ]]; then
    major=${BASH_REMATCH[1]}
    minor=${BASH_REMATCH[2]}
    operator_prerelease=${BASH_REMATCH[5]:-}
  else
    echo "invalid operator version: $operator_tag" >&2
    exit 1
  fi

  if [[ -z "$operator_prerelease" ]]; then
    channel=stable
  elif [[ "$operator_prerelease" =~ ^beta([.-]?[0-9]+)$ ]]; then
    channel=beta
  elif [[ "$operator_prerelease" =~ ^rc([.-]?[0-9]+)$ ]]; then
    channel=rc
  elif [[ "$operator_prerelease" =~ ^alpha([.-]?[0-9]+)$ ]]; then
    channel=alpha
  else
    echo "unsupported prerelease channel in operator tag: $operator_tag" >&2
    exit 1
  fi
fi

release_filter='.[] | select(.draft != true)'
if [[ "$operator_tag" == "latest-stable" ]]; then
  release_filter+=' | select(.prerelease != true)'
fi
release_filter+=' | .tag_name'

if [[ -n "$releases_file" ]]; then
  release_tags=$(jq -r "$release_filter" "$releases_file")
else
  command -v gh >/dev/null || { echo "gh is required" >&2; exit 1; }
  release_tags=$(gh api --paginate "repos/$repository/releases?per_page=100" \
    --jq "$release_filter")
fi

compatible=()
while IFS= read -r candidate; do
  [[ -n "$candidate" ]] || continue
  if [[ "$candidate" =~ ^v?([0-9]+)\.([0-9]+)\.([0-9]+)(-([0-9A-Za-z.-]+))?$ ]]; then
    if [[ "$operator_tag" != "latest-stable" ]]; then
      [[ "${BASH_REMATCH[1]}" == "$major" && "${BASH_REMATCH[2]}" == "$minor" ]] || continue
    fi
    candidate_prerelease=${BASH_REMATCH[5]:-}
  else
    continue
  fi

  case "$channel" in
    stable) [[ -z "$candidate_prerelease" ]] || continue ;;
    beta) [[ "$candidate_prerelease" =~ ^beta([.-]?[0-9]+)$ ]] || continue ;;
    rc) [[ "$candidate_prerelease" =~ ^rc([.-]?[0-9]+)$ ]] || continue ;;
    alpha) [[ "$candidate_prerelease" =~ ^alpha([.-]?[0-9]+)$ ]] || continue ;;
  esac
  compatible+=("$candidate")
done <<< "$release_tags"

if [[ ${#compatible[@]} -eq 0 ]]; then
  if [[ "$operator_tag" == "latest-stable" ]]; then
    echo "no stable release found in $repository" >&2
  else
    echo "no $channel release found for $major.$minor in $repository" >&2
  fi
  exit 1
fi

# Compare versions without the optional v prefix, but return the original tags.
sorted=$(for candidate in "${compatible[@]}"; do
  printf '%s\t%s\n' "${candidate#v}" "$candidate"
done | sort -k1,1Vr)
if [[ "$all" == true ]]; then
  cut -f2 <<< "$sorted"
else
  first_line=${sorted%%$'\n'*}
  printf '%s\n' "${first_line#*$'\t'}"
fi
