#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
workflow="$script_dir/../../.github/workflows/release.yml"

python3 - "$workflow" <<'PY'
import re
import subprocess
import sys

workflow = sys.argv[1]
cases = {
    "v1.1.1": True,
    "v1.1.2": True,
    "v1.2.0": True,
    "v2.0.0": True,
    "v1.1.0-rc.1": False,
    "v1.2.0-rc.1": False,
    "v1.2.0-beta.2": False,
    "v2.0.0-rc.1": False,
}
for job in ("submit-community-catalog", "submit-operatorhub-catalog"):
    condition = subprocess.check_output(
        ["yq", "-r", f'.jobs["{job}"].if // ""', workflow], text=True
    ).strip()
    match = re.fullmatch(
        r"\$\{\{\s*!contains\(\s*github\.ref_name\s*,\s*'([^']+)'\s*\)\s*\}\}",
        condition,
    )
    if not match:
        raise SystemExit(f"{job} has an unsupported release gate: {condition}")
    marker = match.group(1)
    if marker != "-":
        raise SystemExit(f"{job} must detect prereleases on every version; got marker {marker}")
    for tag, expected in cases.items():
        actual = marker not in tag
        if actual != expected:
            raise SystemExit(f"{job} gate gives {actual} for {tag}; expected {expected}")

print("Public catalog release gates are valid")
PY
