#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
workflow="$script_dir/../../.github/workflows/release.yml"
stable_gate="\${{ !contains(github.ref_name, '-') }}"

for job in submit-community-catalog submit-operatorhub-catalog; do
  condition=$(JOB_NAME="$job" yq -r '.jobs[strenv(JOB_NAME)].if // ""' "$workflow")
  [[ "$condition" == "$stable_gate" ]] || {
    echo "$job must use the stable release gate; got: $condition" >&2
    exit 1
  }

  for tag in v1.1.1 v1.2.0; do
    [[ "$tag" != *-* ]] || {
      echo "$job must run for stable tag $tag" >&2
      exit 1
    }
  done
  for tag in v1.2.0-rc.1 v1.2.0-beta.2; do
    [[ "$tag" == *-* ]] || {
      echo "$job must skip prerelease tag $tag" >&2
      exit 1
    }
  done
done

echo "Public catalog release gates are valid"
