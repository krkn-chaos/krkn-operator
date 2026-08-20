# Backup and Restore

Back up krkn-operator state and restore it to the same or a different cluster using the provided backup and restore scripts.

## Prerequisites

- `kubectl` configured with access to the cluster
- `jq` installed locally
- krkn-operator Helm chart installed (CRDs must exist)

## What Gets Backed Up

### Configuration (always backed up)

| Resource | Description |
|----------|-------------|
| `KrknUser` CRs | User accounts, roles, organizations |
| `KrknUserGroup` CRs | RBAC groups with per-cluster permissions |
| `KrknOperatorTarget` CRs | Registered target clusters |
| `KrknOperatorTargetProvider` CRs | Provider registrations |
| `KrknFileType` CRs | File category metadata |
| `Secret` `krkn-operator-jwt` | JWT signing key — loss invalidates all active tokens |
| User password `Secret`s | Referenced by KrknUser CRs |
| Target credential `Secret`s | Kubeconfig secrets referenced by KrknOperatorTarget CRs |
| Registry `Secret`s | Container registry credentials and configuration |
| Elasticsearch `Secret`s | Elasticsearch connection credentials and configuration |
| `ConfigMap`s with label `files.krkn.krkn-chaos.dev/file-id` | User-uploaded scenario config files |

### Excluded (ephemeral per-job resources)

These are created and deleted per scenario execution and should **not** be backed up:

- `ConfigMap`s with label `krkn-job-id` — per-job kubeconfig and file mounts
- `Secret`s with label `krkn-job-id` — per-job image pull secrets
- `Pod`s — scenario execution pods
- `KrknTargetRequest` CRs — transient discovery requests
- `KrknOperatorTargetProviderConfig` CRs — transient config-schema requests

## Security Warning

Backups contain sensitive material:

- **Kubeconfig files** for target clusters
- **Password hashes** for krkn-operator users
- **JWT signing key** used for authentication tokens
- **Registry credentials** for private container registries
- **Elasticsearch credentials**

Treat backup archives as sensitive. Store them in a secure location with restricted access.

## One-Time Backup

```bash
# Run from the repo root:
./config/backup/manual-backup.sh <OPERATOR_NAMESPACE> ./backups

# Creates: ./backups/krkn-backup-20260818-140000.tar.gz
```

The script exports all configuration CRs, Secrets, and file ConfigMaps to JSON, strips cluster-specific metadata (`resourceVersion`, `uid`, etc.) so the files are portable, and archives them.

The script will report errors if it cannot fetch resources (e.g., due to authentication or connectivity issues) and exit with a non-zero status if the backup is incomplete.

## Restoring

### Prerequisites

Before restoring, the target cluster must have:

1. **krkn-operator Helm chart installed** — CRDs must exist so `kubectl apply` can create custom resources
2. **Operator scaled to zero** — to avoid reconciliation conflicts during restore

### Restore to the same cluster

```bash
# 1. Scale operator to zero:
kubectl scale deployment -n <NAMESPACE> -l app.kubernetes.io/name=krkn-operator --replicas=0

# 2. Delete the auto-generated JWT secret so the backup's JWT secret takes its place:
kubectl delete secret krkn-operator-jwt -n <NAMESPACE>

# 3. Restore from the backup:
./config/backup/restore.sh ./backups/krkn-backup-*.tar.gz <NAMESPACE>

# 4. Scale operator back up:
kubectl scale deployment -n <NAMESPACE> -l app.kubernetes.io/name=krkn-operator --replicas=1
```

### Restore to a new cluster

```bash
# 1. Install the Helm chart on the new cluster first:
helm install krkn-operator charts/krkn-operator -n <NEW_NAMESPACE> --create-namespace

# 2. Scale operator down:
kubectl scale deployment -n <NEW_NAMESPACE> -l app.kubernetes.io/name=krkn-operator --replicas=0

# 3. Delete the auto-generated JWT secret:
kubectl delete secret krkn-operator-jwt -n <NEW_NAMESPACE>

# 4. Restore from the backup:
./config/backup/restore.sh ./backups/krkn-backup-*.tar.gz <NEW_NAMESPACE>

# 5. Scale operator back up:
kubectl scale deployment -n <NEW_NAMESPACE> -l app.kubernetes.io/name=krkn-operator --replicas=1
```

### Conflict handling

`kubectl apply` will update resources that already exist and create ones that don't. If you want to avoid overwriting existing resources, use `kubectl create` instead (it will skip existing resources with an error).

## Verifying a Restore

After restoring, verify that the system is functional:

```bash
# 1. Check that CRs were restored:
kubectl get krknusers -n <NAMESPACE>
kubectl get krknusergroups -n <NAMESPACE>
kubectl get krknoperatortargets -n <NAMESPACE>
kubectl get krknoperatortargetproviders -n <NAMESPACE>
kubectl get krknfiletypes -n <NAMESPACE>

# 2. Check that secrets exist:
kubectl get secret krkn-operator-jwt -n <NAMESPACE>
kubectl get secrets -n <NAMESPACE> -l "app.kubernetes.io/component=authentication"
kubectl get secrets -n <NAMESPACE> -l "krkn-target-uuid"
kubectl get secrets -n <NAMESPACE> -l "app.kubernetes.io/component=registry"
kubectl get secrets -n <NAMESPACE> -l "app.kubernetes.io/component=elasticsearch-config"

# 3. Check that file ConfigMaps exist:
kubectl get configmaps -n <NAMESPACE> -l "files.krkn.krkn-chaos.dev/file-id"

# 4. Verify a user can log in (confirms JWT secret + password secrets restored):
curl -X POST https://<OPERATOR_URL>/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username": "<USER>", "password": "<PASSWORD>"}'

# 5. Verify target connectivity (confirms kubeconfig secrets restored):
kubectl logs -n <NAMESPACE> -l app.kubernetes.io/name=krkn-operator --tail=50
```

## File Reference

| File | Description |
|------|-------------|
| `config/backup/manual-backup.sh` | Shell script for one-time backup to `.tar.gz` |
| `config/backup/restore.sh` | Shell script to restore from a backup archive |
