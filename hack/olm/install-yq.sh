#!/usr/bin/env bash
set -euo pipefail

yq_version=v4.45.1
yq_sha256=654d2943ca1d3be2024089eb4f270f4070f491a0610481d128509b2834870049

curl --fail --location --retry 3 \
  --output /tmp/yq \
  "https://github.com/mikefarah/yq/releases/download/${yq_version}/yq_linux_amd64"
printf '%s  %s\n' "$yq_sha256" /tmp/yq | sha256sum --check --status -
sudo install -m 0755 /tmp/yq /usr/local/bin/yq
