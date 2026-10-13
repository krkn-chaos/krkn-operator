#!/usr/bin/env bash
set -euo pipefail

if [[ "${GITHUB_EVENT_NAME:-}" == "workflow_run" && -n "${SOURCE_RELEASE_CHART:-}" && -f "${SOURCE_RELEASE_CHART}" ]] && \
  grep -Eq '^[[:space:]]*notify-website:[[:space:]]*$' "$SOURCE_RELEASE_CHART" && \
  grep -Eq '^[[:space:]]*uses:[[:space:]]+\./\.github/workflows/notify-operator-docs\.yml[[:space:]]*$' "$SOURCE_RELEASE_CHART"; then
  echo "The tag's chart workflow already called the notifier directly; skipping fallback dispatch." >&2
  exit 0
fi

OPERATOR_TAG="${INPUT_OPERATOR_TAG:-}"

if [[ "${GITHUB_EVENT_NAME:-}" == "workflow_run" ]]; then
  head_sha="${WORKFLOW_RUN_HEAD_SHA:-}"
  repo_root="${OPERATOR_REPO_ROOT:-$PWD}"
  : "${head_sha:?WORKFLOW_RUN_HEAD_SHA must be set for the legacy chart-workflow fallback}"

  tags_at_head=()
  tag_count=0
  while IFS= read -r candidate; do
    if [[ -n "$candidate" ]]; then
      tags_at_head[$tag_count]="$candidate"
      tag_count=$((tag_count + 1))
    fi
  done < <(git -C "$repo_root" tag --points-at "$head_sha" | sort)
  if [[ -n "${PUSHED_TAG:-}" ]]; then
    for ((index = 0; index < tag_count; index++)); do
      candidate="${tags_at_head[$index]}"
      if [[ "$candidate" == "$PUSHED_TAG" ]]; then
        OPERATOR_TAG="$candidate"
        break
      fi
    done
  fi

  if [[ -z "$OPERATOR_TAG" ]]; then
    if [[ $tag_count -eq 1 ]]; then
      OPERATOR_TAG="${tags_at_head[0]}"
    elif [[ $tag_count -gt 1 ]]; then
      echo "Multiple tags point to workflow run head SHA ${head_sha}; refusing to guess from head_branch '${PUSHED_TAG:-}'." >&2
      exit 1
    else
      echo "No tag points to workflow run head SHA ${head_sha}; refusing to treat head_branch '${PUSHED_TAG:-}' as a release tag." >&2
      exit 1
    fi
  fi
else
  OPERATOR_TAG="${OPERATOR_TAG:-${PUSHED_TAG:-}}"
fi

: "${OPERATOR_TAG:?OPERATOR_TAG must be set from a workflow input or verified release tag}"

if [[ ! "$OPERATOR_TAG" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  echo "Skipping non-stable Operator release tag: ${OPERATOR_TAG}." >&2
  exit 0
fi

gh api --method POST repos/krkn-chaos/website/dispatches \
  -f event_type=operator-docs-release-ready \
  -F "client_payload[operator_tag]=$OPERATOR_TAG"
