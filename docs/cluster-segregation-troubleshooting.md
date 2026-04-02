# Cluster Segregation Troubleshooting

## Overview

The krkn-operator implements group-based access control to segregate scenario runs based on cluster permissions. Users should only see runs from clusters where their groups have at least `view` permission.

## How It Works

### Permission Model

1. **Users** belong to **Groups** (via labels on KrknUser CR)
2. **Groups** have **Permissions** on clusters (defined by cluster API URLs)
3. **Scenario Runs** store cluster API URLs (in `spec.clusterApiUrls`)
4. **Filtering**: Users see runs where they have `view` permission on ≥1 cluster

### Permission Types

- `view`: Can see runs and logs
- `run`: Can create runs
- `cancel`: Can cancel/delete runs

### Access Rules

- **Admin users**: See all runs (bypass all checks)
- **Regular users**: See only runs where they have `view` permission on at least one cluster
- **Legacy runs** (without `clusterApiUrls`): Excluded for regular users, visible to admins

## Common Issues

### Issue: User sees ALL runs (not properly segregated)

**Possible causes:**

1. **User is admin**
   - Check: `kubectl get krknuser <user> -n krkn-operator-system -o jsonpath='{.spec.role}'`
   - If role is `admin`, they will see all runs by design

2. **Runs are legacy (no ClusterAPIURLs)**
   - Check: Run `scripts/debug-cluster-permissions.sh`
   - Look for runs in "Runs WITHOUT ClusterAPIURLs" section
   - **Solution**: These are old runs created before the feature was added
     - They will be excluded for regular users
     - Only admins can see them
     - New runs created via API will have ClusterAPIURLs populated

3. **User has permissions on all clusters**
   - Check user's groups and their permissions
   - If user's groups cover all clusters in all runs, they will see everything
   - This is correct behavior

4. **ClusterAPIURLs not being populated**
   - Check recent runs: `kubectl get krknscenariorun <run-name> -n krkn-operator-system -o jsonpath='{.spec.clusterApiUrls}'`
   - Should return a map like: `{"cluster1":"https://...","cluster2":"https://..."}`
   - If empty, check that KrknTargetRequest has `status.targetData` populated

### Issue: User sees NO runs (too restrictive)

**Possible causes:**

1. **User has no group memberships**
   - Check: `kubectl get krknuser <user> -n krkn-operator-system -o jsonpath='{.metadata.labels}'`
   - Should see labels like `group.krkn.krkn-chaos.dev/<group-name>: "true"`
   - **Solution**: Add user to appropriate groups

2. **User's groups have no cluster permissions**
   - Check group permissions: `kubectl get krknusergroup <group> -n krkn-operator-system -o jsonpath='{.spec.clusterPermissions}'`
   - **Solution**: Add cluster permissions to groups

3. **Permission type mismatch**
   - User needs `view` permission to see runs
   - Check that groups have `"view"` in their actions array

## Debugging Steps

### Step 1: Verify User Setup

```bash
# Get user details
kubectl get krknuser -n krkn-operator-system

# Check specific user
kubectl get krknuser krknuser-<sanitized-email> -n krkn-operator-system -o yaml

# Verify role (admin vs user)
kubectl get krknuser krknuser-<sanitized-email> -n krkn-operator-system -o jsonpath='{.spec.role}'

# Check group memberships
kubectl get krknuser krknuser-<sanitized-email> -n krkn-operator-system -o jsonpath='{.metadata.labels}'
```

### Step 2: Verify Group Permissions

```bash
# List all groups
kubectl get krknusergroup -n krkn-operator-system

# Check specific group
kubectl get krknusergroup <group-name> -n krkn-operator-system -o yaml

# View cluster permissions
kubectl get krknusergroup <group-name> -n krkn-operator-system -o jsonpath='{.spec.clusterPermissions}'
```

### Step 3: Verify Scenario Runs

```bash
# List all runs
kubectl get krknscenariorun -n krkn-operator-system

# Check ClusterAPIURLs for specific run
kubectl get krknscenariorun <run-name> -n krkn-operator-system -o jsonpath='{.spec.clusterApiUrls}'

# Find runs without ClusterAPIURLs (legacy runs)
kubectl get krknscenariorun -n krkn-operator-system -o json | \
  jq -r '.items[] | select(.spec.clusterApiUrls == null or (.spec.clusterApiUrls | length) == 0) | .metadata.name'
```

### Step 4: Run Debug Script

```bash
# Run comprehensive debug script
./scripts/debug-cluster-permissions.sh
```

This will show:
- All runs and their ClusterAPIURLs
- Legacy runs without ClusterAPIURLs
- All groups and their permissions
- All users and their group memberships
- Summary of cluster API URLs

### Step 5: Test API Endpoint

```bash
# Get JWT token for user
TOKEN=$(curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}' | jq -r '.token')

# List runs as that user
curl -X GET http://localhost:8080/api/v1/scenarios/run \
  -H "Authorization: Bearer $TOKEN" | jq '.runs[] | .scenarioRunName'
```

## Expected Behavior Examples

### Example 1: Proper Segregation

Setup:
- User A belongs to Group A
- Group A has `view` permission on `https://cluster1.example.com:6443`
- Run1 targets cluster1
- Run2 targets cluster2
- Run3 targets both cluster1 and cluster2

Expected:
- User A sees: Run1, Run3 (has permission on cluster1)
- User A does NOT see: Run2 (no permission on cluster2)

### Example 2: Multiple Groups

Setup:
- User B belongs to Group A and Group B
- Group A has `view` on cluster1
- Group B has `view` on cluster2
- Run1 targets cluster1
- Run2 targets cluster2

Expected:
- User B sees: Run1, Run2 (union of permissions from both groups)

### Example 3: Admin User

Setup:
- User C has role `admin`
- Runs exist on various clusters

Expected:
- User C sees: ALL runs (admins bypass permission checks)

## Code References

- Authorization logic: `internal/api/authorization.go`
  - `filterScenarioRunsByGroupPermission()` - Filters runs for users
  - `checkScenarioRunGroupAccess()` - Checks if user has permission on run

- Run creation: `internal/api/handlers.go`
  - `PostScenarioRun()` - Populates `ClusterAPIURLs` on creation

- Tests: `internal/api/scenario_run_cluster_segregation_test.go`
  - Comprehensive tests covering multiple scenarios

## Migration from Legacy Runs

If you have runs created before this feature:

1. **They will not be visible to regular users** (only admins)
2. To make them visible, you would need to manually patch them:
   ```bash
   kubectl patch krknscenariorun <run-name> -n krkn-operator-system \
     --type=merge -p '{"spec":{"clusterApiUrls":{"cluster1":"https://cluster1.example.com:6443"}}}'
   ```
3. **Recommended**: Delete old runs and create new ones via the API

## Verification Checklist

- [ ] User exists in krkn-operator-system namespace
- [ ] User has group membership labels
- [ ] Groups exist and have cluster permissions
- [ ] Groups include "view" in their actions
- [ ] Scenario runs have ClusterAPIURLs populated
- [ ] Cluster API URLs in runs match those in group permissions
- [ ] User role is correct (admin vs user)
- [ ] API endpoint returns expected filtered results
