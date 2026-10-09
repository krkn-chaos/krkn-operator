#!/usr/bin/env bash
set -euo pipefail

OPERATOR_TAG="${INPUT_OPERATOR_TAG:-${RELEASE_TAG:-${PUSHED_TAG:-}}}"
: "${OPERATOR_TAG:?OPERATOR_TAG must be set from a workflow input, published release, or pushed tag}"

if [[ "${RELEASE_IS_PRERELEASE:-false}" == "true" ]]; then
  echo "Skipping non-stable Operator release ${OPERATOR_TAG}: GitHub marks it as a prerelease." >&2
  exit 0
fi

if [[ ! "$OPERATOR_TAG" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  echo "Skipping non-stable Operator release tag: ${OPERATOR_TAG}." >&2
  exit 0
fi

gh api --method POST repos/krkn-chaos/website/dispatches \
  -f event_type=operator-docs-release-ready \
  -F "client_payload[operator_tag]=$OPERATOR_TAG"
