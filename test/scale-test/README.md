# Scale Test - User Generation

This directory contains utilities for generating test users for scale testing the krkn-operator authentication and authorization system.

## Overview

The `generate-users.sh` script creates 15 KrknUser resources with associated password secrets for testing purposes.

### Generated Users

- **Total Users**: 15
- **Admin Users**: 2 (test-user-1 and test-user-2) - no group assignment
- **QA Team**: 5 users (test-user-3 through test-user-7)
- **Developers**: 5 users (test-user-8 through test-user-12)
- **SRE Team**: 3 users (test-user-13 through test-user-15)
- **Default Password**: `TempPass123!` (same for all users)
- **Email Format**: `test-user-{N}@krkn-chaos.dev`

The script automatically creates 3 user groups and assigns users to them during generation.

## Prerequisites

### Required Tools

- `kubectl` - Kubernetes CLI
- `htpasswd` - For bcrypt password hashing

#### Installing htpasswd

**macOS**:
```bash
brew install httpd
```

**Ubuntu/Debian**:
```bash
sudo apt-get install apache2-utils
```

**RHEL/Fedora**:
```bash
sudo dnf install httpd-tools
```

## Usage

### Generate Manifests Only

Generate YAML manifests without applying them to the cluster:

```bash
./generate-users.sh generate
```

This creates individual YAML files in `manifests/` directory and a combined `manifests/all-users.yaml` file.

### Generate and Apply to Cluster

Generate manifests (including groups and user assignments) and apply them to the cluster in one step:

```bash
./generate-users.sh apply
```

This will:
1. Create 3 KrknUserGroup resources (qa-team, developers, sre-team)
2. Create 15 KrknUser resources
3. Create 15 password secrets
4. Automatically assign users to groups via labels
5. Enable all users (set status.active=true)

**Important**: Make sure your krkn-operator is running and watching the same namespace:
```bash
KRKN_NAMESPACE=krkn-operator-system ./start_operator.sh -k /path/to/kubeconfig
```

By default, everything is created in the `krkn-operator-system` namespace. To use a different namespace:

```bash
NAMESPACE=my-namespace ./generate-users.sh apply
```

To use a specific kubeconfig file:

```bash
KUBECONFIG_PATH=~/.kube/my-cluster ./generate-users.sh apply
```

Or combine both:

```bash
NAMESPACE=my-namespace KUBECONFIG_PATH=~/.kube/my-cluster ./generate-users.sh apply
```

### Delete Test Users

Remove all generated users from the cluster:

```bash
./generate-users.sh delete
```

### Show Credentials

Display the credentials for all test users:

```bash
./generate-users.sh show
```

## User Details

### Admin Users (2)

1. `test-user-1@krkn-chaos.dev` - Admin role, no group
2. `test-user-2@krkn-chaos.dev` - Admin role, no group

### QA Team (5)

3. `test-user-3@krkn-chaos.dev` - User role, qa-team group
4. `test-user-4@krkn-chaos.dev` - User role, qa-team group
5. `test-user-5@krkn-chaos.dev` - User role, qa-team group
6. `test-user-6@krkn-chaos.dev` - User role, qa-team group
7. `test-user-7@krkn-chaos.dev` - User role, qa-team group

### Developers (5)

8. `test-user-8@krkn-chaos.dev` - User role, developers group
9. `test-user-9@krkn-chaos.dev` - User role, developers group
10. `test-user-10@krkn-chaos.dev` - User role, developers group
11. `test-user-11@krkn-chaos.dev` - User role, developers group
12. `test-user-12@krkn-chaos.dev` - User role, developers group

### SRE Team (3)

13. `test-user-13@krkn-chaos.dev` - User role, sre-team group
14. `test-user-14@krkn-chaos.dev` - User role, sre-team group
15. `test-user-15@krkn-chaos.dev` - User role, sre-team group

### Password

All users share the same temporary password: **`TempPass123!`**

⚠️ **WARNING**: These are temporary passwords for testing only. Do NOT use these users or passwords in production environments.

## Testing Scenarios

### Authentication Testing

Test user authentication with the REST API:

```bash
# Login as admin user
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "test-user-1@krkn-chaos.dev",
    "password": "TempPass123!"
  }'

# Login as regular user
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "test-user-5@krkn-chaos.dev",
    "password": "TempPass123!"
  }'
```

### Group Permissions Testing

The script automatically creates these groups with different permission levels:

- **qa-team**: View + run permissions on dev clusters, full permissions on dev-cluster-2
- **developers**: View-only permissions on dev-cluster-1
- **sre-team**: Full permissions (view, run, cancel) on all clusters including production

Test group-based permissions by logging in as users from different groups.

### Load Testing

Use these users for concurrent authentication and API load testing:

```bash
# Example: Test concurrent logins
for i in {1..15}; do
  curl -X POST http://localhost:8080/api/v1/login \
    -H "Content-Type: application/json" \
    -d "{\"userId\":\"test-user-${i}@krkn-chaos.dev\",\"password\":\"TempPass123!\"}" &
done
wait
```

## File Structure

```
test/scale-test/
├── README.md              # This file
├── generate-users.sh      # Main generation script
├── user-template.yaml     # Configurable user YAML template
├── sample-groups.yaml     # Example group configurations
└── manifests/             # Generated YAML files (gitignored)
    ├── user-1.yaml
    ├── user-2.yaml
    ├── ...
    ├── user-15.yaml
    └── all-users.yaml     # Combined manifest
```

## Verification

Check that users were created successfully:

```bash
# List all KrknUser resources
kubectl get krknusers -n krkn-operator-system

# Get details for a specific user
kubectl get krknuser krkn-user-1 -n krkn-operator-system -o yaml

# Verify password secrets
kubectl get secrets -n krkn-operator-system -l krkn.krkn-chaos.dev/user-account=true
```

## Cleanup

To remove all test users and their secrets:

```bash
./generate-users.sh delete
```

This will delete:
- All 15 KrknUser resources
- All associated password secrets

## Customization

### Customize User Structure

Edit [user-template.yaml](user-template.yaml) to change the user resource structure:

```yaml
# Available template variables:
# {{USER_NUM}}, {{USER_ID}}, {{NAME}}, {{SURNAME}}, 
# {{ORGANIZATION}}, {{ROLE}}, {{SECRET_NAME}},
# {{PASSWORD_HASH}}, {{NAMESPACE}}, {{GROUP_LABELS}}
```

Example: Add custom labels or annotations:
```yaml
metadata:
  name: krkn-user-{{USER_NUM}}
  namespace: {{NAMESPACE}}
  labels:
    krkn.krkn-chaos.dev/user-account: "true"
    krkn.krkn-chaos.dev/role: "{{ROLE}}"
    my-custom-label: "my-value"  # Add your labels here
  annotations:
    description: "Scale test user {{USER_NUM}}"
{{GROUP_LABELS}}
```

### Customize Number of Users or Password

Edit [generate-users.sh](generate-users.sh):
```bash
NUM_USERS=15              # Change number of users
PASSWORD="TempPass123!"   # Change default password
```

Then regenerate:
```bash
./generate-users.sh generate
```
