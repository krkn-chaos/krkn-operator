#!/bin/bash
# Script to debug cluster permission issues in production
# This helps verify that ClusterAPIURLs are populated and permissions are correct

set -e

NAMESPACE=${NAMESPACE:-krkn-operator-system}

echo "==================================="
echo "Cluster Permission Debug Tool"
echo "==================================="
echo ""

echo "1. Checking KrknScenarioRuns for ClusterAPIURLs..."
echo "---------------------------------------------------"
kubectl get krknscenariorun -n "$NAMESPACE" -o json | jq -r '.items[] | {
  name: .metadata.name,
  phase: .status.phase,
  clusterAPIURLs: .spec.clusterApiUrls,
  hasClusterAPIURLs: (.spec.clusterApiUrls != null and (.spec.clusterApiUrls | length) > 0)
}'
echo ""

echo "2. Runs WITHOUT ClusterAPIURLs (legacy runs - users won't see these):"
echo "----------------------------------------------------------------------"
kubectl get krknscenariorun -n "$NAMESPACE" -o json | jq -r '.items[] | select(.spec.clusterApiUrls == null or (.spec.clusterApiUrls | length) == 0) | .metadata.name'
echo ""

echo "3. Checking KrknUserGroups and their permissions..."
echo "----------------------------------------------------"
kubectl get krknusergroup -n "$NAMESPACE" -o json | jq -r '.items[] | {
  name: .metadata.name,
  description: .spec.description,
  clusterPermissions: .spec.clusterPermissions
}'
echo ""

echo "4. Checking KrknUsers and their group memberships..."
echo "-----------------------------------------------------"
kubectl get krknuser -n "$NAMESPACE" -o json | jq -r '.items[] | {
  name: .metadata.name,
  userID: .spec.userId,
  role: .spec.role,
  groups: [.metadata.labels | to_entries[] | select(.key | startswith("group.krkn.krkn-chaos.dev/")) | .key | split("/")[1]]
}'
echo ""

echo "5. Summary of ClusterAPIURLs across all runs:"
echo "----------------------------------------------"
kubectl get krknscenariorun -n "$NAMESPACE" -o json | jq -r '.items[].spec.clusterApiUrls | to_entries[] | .value' | sort -u
echo ""

echo "==================================="
echo "Debug Information Complete"
echo "==================================="
echo ""
echo "Expected behavior:"
echo "- Regular users should only see runs where they have 'view' permission on at least one cluster in the run"
echo "- Admin users see all runs"
echo "- Runs without ClusterAPIURLs are not visible to regular users (legacy runs)"
echo ""
echo "To test permissions for a specific user:"
echo "1. Get user's groups: kubectl get krknuser <user-name> -n $NAMESPACE -o jsonpath='{.metadata.labels}'"
echo "2. Get group permissions: kubectl get krknusergroup <group-name> -n $NAMESPACE -o jsonpath='{.spec.clusterPermissions}'"
echo "3. Check if user should see a run: verify if user's groups have 'view' permission on any cluster in run's ClusterAPIURLs"
