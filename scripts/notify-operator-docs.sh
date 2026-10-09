#!/usr/bin/env bash
set -euo pipefail

OPERATOR_TAG="${INPUT_OPERATOR_TAG:-${RELEASE_TAG:-${PUSHED_TAG:-}}}"
: "${OPERATOR_TAG:?OPERATOR_TAG must be set to a published Operator release tag}"

gh api --method POST repos/krkn-chaos/website/dispatches \
  -f event_type=operator-docs-release-ready \
  -F "client_payload[operator_tag]=$OPERATOR_TAG"
