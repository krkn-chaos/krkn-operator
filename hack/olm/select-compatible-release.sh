#!/usr/bin/env bash

set -euo pipefail

usage() {
  echo "usage: $0 <github-owner/repository> <operator-tag> [releases.json]" >&2
  exit 2
}

[[ $# -ge 2 && $# -le 3 ]] || usage

repository=$1
operator_tag=$2
releases_file=${3:-}

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

if [[ -n "$releases_file" ]]; then
  release_tags=$(jq -r '.[] | select(.draft != true) | .tag_name' "$releases_file")
else
  command -v gh >/dev/null || { echo "gh is required" >&2; exit 1; }
  release_tags=$(gh api --paginate "repos/$repository/releases?per_page=100" \
    --jq '.[] | select(.draft != true) | .tag_name')
fi

compatible=()
while IFS= read -r candidate; do
  [[ -n "$candidate" ]] || continue
  if [[ "$candidate" =~ ^v?([0-9]+)\.([0-9]+)\.([0-9]+)(-([0-9A-Za-z.-]+))?$ ]]; then
    [[ "${BASH_REMATCH[1]}" == "$major" && "${BASH_REMATCH[2]}" == "$minor" ]] || continue
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
  echo "no $channel release found for $major.$minor in $repository" >&2
  exit 1
fi

printf '%s\n' "${compatible[@]}" | sort -V | tail -n 1
