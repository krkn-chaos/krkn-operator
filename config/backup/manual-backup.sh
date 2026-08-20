#!/usr/bin/env bash
# Manual backup of krkn-operator resources.
# Exports all configuration resources to JSON, strips cluster-specific
# metadata (resourceVersion, uid, creationTimestamp, managedFields),
# and creates a timestamped .tar.gz archive.
#
# Usage:
#   ./manual-backup.sh <namespace> [output-dir]
#
# Example:
#   ./manual-backup.sh krkn-operator ./backups
#
# Restore:
#   ./restore.sh ./backups/krkn-backup-*.tar.gz <target-namespace>

set -euo pipefail

NAMESPACE="${1:?Usage: $0 <namespace> [output-dir]}"
OUTPUT_DIR="${2:-.}"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
BACKUP_DIR="$(mktemp -d)/krkn-backup"
ARCHIVE="${OUTPUT_DIR}/krkn-backup-${TIMESTAMP}.tar.gz"

cleanup() {
    rm -rf "$(dirname "${BACKUP_DIR}")"
}
trap cleanup EXIT

mkdir -p "${BACKUP_DIR}" "${OUTPUT_DIR}"

strip_metadata='walk(if type == "object" and has("metadata") then .metadata |= (del(.resourceVersion, .uid, .creationTimestamp, .generation, .managedFields, .namespace) | if .annotations then .annotations |= del(.["kubectl.kubernetes.io/last-applied-configuration"]) | if .annotations == {} then del(.annotations) else . end else . end) else . end)'

export_resource() {
    local output
    output=$(kubectl get "$1" -n "${NAMESPACE}" $2 -o json 2>&1) || {
        echo "Error: failed to fetch $1 $2 from namespace ${NAMESPACE}" >&2
        echo "  $output" >&2
        return 1
    }

    local items
    items=$(echo "${output}" | jq -r '.items // [] | length' 2>/dev/null)
    if [ "${items:-0}" -eq 0 ] && [ "$(echo "${output}" | jq -r '.kind' 2>/dev/null)" = "List" ]; then
        return 0
    fi

    echo "${output}" | jq "${strip_metadata}" > "${BACKUP_DIR}/$3" || {
        echo "Error: failed to process $1 with jq" >&2
        return 1
    }
}

echo "Backing up krkn-operator resources from namespace: ${NAMESPACE}"

ERRORS=0

# Custom resources
export_resource "krknusers.krkn.krkn-chaos.dev" "" "users.json" || ((ERRORS++))
export_resource "krknusergroups.krkn.krkn-chaos.dev" "" "usergroups.json" || ((ERRORS++))
export_resource "krknoperatortargets.krkn.krkn-chaos.dev" "" "targets.json" || ((ERRORS++))
export_resource "krknoperatortargetproviders.krkn.krkn-chaos.dev" "" "providers.json" || ((ERRORS++))
export_resource "krknfiletypes.krkn.krkn-chaos.dev" "" "filetypes.json" || ((ERRORS++))

# Secrets
export_resource "secret" "krkn-operator-jwt" "jwt-secret.json" || ((ERRORS++))
export_resource "secrets" "-l app.kubernetes.io/component=authentication" "auth-secrets.json" || ((ERRORS++))
export_resource "secrets" "-l app.kubernetes.io/component=user-auth" "user-auth-secrets.json" || ((ERRORS++))
export_resource "secrets" "-l krkn-target-uuid" "target-secrets.json" || ((ERRORS++))

# Elasticsearch config
export_resource "secrets" "-l app.kubernetes.io/component=elasticsearch-config" "elasticsearch-secrets.json" || ((ERRORS++))

# Registry secrets
export_resource "secrets" "-l app.kubernetes.io/component=registry" "registry-secrets.json" || ((ERRORS++))

# File ConfigMaps
export_resource "configmaps" "-l files.krkn.krkn-chaos.dev/file-id" "file-configmaps.json" || ((ERRORS++))

# Remove empty files (resource types with zero instances)
find "${BACKUP_DIR}" -empty -delete

if [ "$(find "${BACKUP_DIR}" -type f | wc -l)" -eq 0 ]; then
    echo "Error: no resources were backed up" >&2
    exit 1
fi

tar czf "${ARCHIVE}" -C "$(dirname "${BACKUP_DIR}")" "krkn-backup"

echo "Backup saved to: ${ARCHIVE}"
echo "Contents:"
tar tzf "${ARCHIVE}"

if [ "${ERRORS}" -gt 0 ]; then
    echo ""
    echo "Warning: ${ERRORS} resource type(s) failed to export. Backup may be incomplete." >&2
    exit 1
fi
