#!/usr/bin/env bash
# Restore krkn-operator resources from a backup archive.
#
# Applies all resources, then patches status subresources for CRs
# that use the status subresource (kubectl apply ignores status).
#
# Usage:
#   ./restore.sh <backup-archive> <target-namespace>
#
# Example:
#   ./restore.sh ./backups/krkn-backup-20260818-142444.tar.gz krkn-operator

set -euo pipefail

ARCHIVE="${1:?Usage: $0 <backup-archive> <target-namespace>}"
NAMESPACE="${2:?Usage: $0 <backup-archive> <target-namespace>}"
EXTRACT_DIR="$(mktemp -d)"

cleanup() {
    rm -rf "${EXTRACT_DIR}"
}
trap cleanup EXIT

echo "Extracting ${ARCHIVE}..."
tar xzf "${ARCHIVE}" -C "${EXTRACT_DIR}"

BACKUP_DIR="${EXTRACT_DIR}/krkn-backup"

if [ ! -d "${BACKUP_DIR}" ]; then
    echo "Error: backup directory not found in archive" >&2
    exit 1
fi

echo "Restoring resources to namespace: ${NAMESPACE}"
kubectl apply -f "${BACKUP_DIR}/" -n "${NAMESPACE}"

echo "Restoring status subresources..."

STATUS_ERRORS=0

for file in "${BACKUP_DIR}"/*.json; do
    [ -f "$file" ] || continue

    ITEMS=$(jq -r '.items // [] | length' "$file" 2>/dev/null || echo "0")
    if [ "${ITEMS}" -eq 0 ]; then
        continue
    fi

    for i in $(seq 0 $((ITEMS - 1))); do
        KIND=$(jq -r ".items[$i].kind" "$file")
        NAME=$(jq -r ".items[$i].metadata.name" "$file")
        STATUS=$(jq -c ".items[$i].status // empty" "$file")
        APIVERSION=$(jq -r ".items[$i].apiVersion" "$file")

        if [ -z "${STATUS}" ] || [ "${STATUS}" = "null" ]; then
            continue
        fi

        # Only patch status for krkn CRDs
        if [[ "${APIVERSION}" != *"krkn-chaos.dev"* ]]; then
            continue
        fi

        RESOURCE=$(echo "${KIND}" | tr '[:upper:]' '[:lower:]')s
        echo "  Patching status: ${RESOURCE}/${NAME}"
        if ! kubectl patch "${RESOURCE}.krkn.krkn-chaos.dev" "${NAME}" \
            -n "${NAMESPACE}" \
            --type=merge \
            --subresource=status \
            -p "{\"status\": ${STATUS}}" 2>/dev/null; then
            echo "  Error: failed to patch status for ${RESOURCE}/${NAME}" >&2
            ((STATUS_ERRORS++))
        fi
    done
done

echo ""
if [ "${STATUS_ERRORS}" -gt 0 ]; then
    echo "Restore completed with ${STATUS_ERRORS} status patch failure(s)." >&2
    echo "Some resources may not be fully functional (e.g., users may appear disabled)." >&2
    echo "Check and patch manually:" >&2
    echo "  kubectl patch <resource> <name> -n ${NAMESPACE} --type=merge --subresource=status -p '{\"status\":{\"active\":true}}'" >&2
    exit 1
fi

echo "Restore complete. Verify with:"
echo "  kubectl get krknusers -n ${NAMESPACE}"
echo "  kubectl get krknoperatortargets -n ${NAMESPACE}"
echo "  kubectl get secret krkn-operator-jwt -n ${NAMESPACE}"
