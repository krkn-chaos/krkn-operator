# Scale Test Quick Start

## One-Command Setup

```bash
cd test/scale-test
KUBECONFIG_PATH=/path/to/kubeconfig ./generate-users.sh apply
```

This single command:
✅ Creates 3 user groups (qa-team, developers, sre-team)  
✅ Creates 15 users with bcrypt-hashed passwords  
✅ Automatically assigns users to groups  
✅ Applies everything to your cluster  
✅ Enables all users automatically

**Important**: Start operator with matching namespace:
```bash
cd /Users/prubenda/Github/krkn-operator
KRKN_NAMESPACE=krkn-operator-system ./start_operator.sh -k /path/to/kubeconfig --skip-build
```  

## What Gets Created

### Groups (3)
- **qa-team** - View + run on dev clusters
- **developers** - View-only on dev clusters  
- **sre-team** - Full permissions on all clusters

### Users (15)

| Range | Count | Role | Group | Description |
|-------|-------|------|-------|-------------|
| 1-2 | 2 | admin | _(none)_ | Admin users, bypass all group checks |
| 3-7 | 5 | user | qa-team | QA testers with run permissions |
| 8-12 | 5 | user | developers | Developers with view-only access |
| 13-15 | 3 | user | sre-team | SRE with full cluster access |

**Password (all users):** `TempPass123!`  
**Email format:** `test-user-{N}@krkn-chaos.dev`

## Test Authentication

```bash
# Admin user
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"userId":"test-user-1@krkn-chaos.dev","password":"TempPass123!"}'

# QA team member
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"userId":"test-user-5@krkn-chaos.dev","password":"TempPass123!"}'

# Developer
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"userId":"test-user-10@krkn-chaos.dev","password":"TempPass123!"}'

# SRE
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"userId":"test-user-15@krkn-chaos.dev","password":"TempPass123!"}'
```

## Verify Setup

```bash
# Check groups
kubectl get krknusergroups -n krkn-operator-system

# Check users
kubectl get krknusers -n krkn-operator-system

# Verify group assignments
kubectl get krknuser krkn-user-5 -n krkn-operator-system -o jsonpath='{.metadata.labels}'
```

## Cleanup

```bash
./generate-users.sh delete
```

## Customize

**User Structure:** Edit [user-template.yaml](user-template.yaml) to customize the YAML structure.

**Script Variables:** Edit these in [generate-users.sh](generate-users.sh):
- `NUM_USERS` - Number of users to create (default: 15)
- `PASSWORD` - Temporary password (default: TempPass123!)
- `NAMESPACE` - Target namespace (default: krkn-operator-system)

Or override via environment:
```bash
NAMESPACE=my-test PASSWORD=MyPass123 ./generate-users.sh apply
```

## Use Different Cluster

If you have multiple clusters, specify the kubeconfig:

```bash
# Use specific kubeconfig file
KUBECONFIG_PATH=~/.kube/my-cluster ./generate-users.sh apply

# Or set KUBECONFIG directly
export KUBECONFIG=~/.kube/my-cluster
./generate-users.sh apply
```
