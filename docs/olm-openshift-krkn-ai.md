# Optional Krkn-AI with OLM on OpenShift

Krkn-AI is **disabled by default**. The bundle includes the `KrknAIRun` CRD;
setting `KRKNAI_ENABLED=true` in the Subscription provisions the AI service.
No separate Krkn-AI operator or Helm release is needed.

## Prerequisites

- A disposable OpenShift cluster, `oc` logged in with CRD/RBAC installation
  permissions, and a fresh namespace—not an existing Helm installation.
  OpenShift already provides OLM.
- Helm, `jq`, `yq` v4, Podman, and Operator SDK v1.41.1 (`make operator-sdk`).
- Cluster-pullable images and a default StorageClass or writable existing PVC.
  The examples assume public images; private images need pull credentials.
- For chaos runs: a designated disposable target, target credentials, and
  reachable Prometheus.

## 1. Build a development bundle

To test uncommitted changes, build/push a new operator and bundle. Replace the
registry and image placeholders with compatible versions; AI versions need not
match the operator version.

```bash
export REGISTRY=quay.io/YOUR_ACCOUNT
export VERSION=1.1.0-olm-ai.1
export OPERATOR_IMAGE="$REGISTRY/krkn-operator:$VERSION"
export BUNDLE_IMAGE="$REGISTRY/krkn-operator-bundle-ocp:$VERSION"
export CONSOLE_IMAGE="$REGISTRY/krkn-operator-console:$VERSION"
export AI_ORCHESTRATOR_IMAGE="$REGISTRY/krkn-ai-operator:$VERSION"
export AI_SERVICE_IMAGE="$REGISTRY/krkn-ai-service:$VERSION"
export DATA_PROVIDER_IMAGE="quay.io/krkn-chaos/krkn-operator-data-provider"

GOTOOLCHAIN=auto make docker-build CONTAINER_TOOL=podman IMG="$OPERATOR_IMAGE"
podman push "$OPERATOR_IMAGE"
make operator-sdk
export PATH="$PWD/bin:$PATH"

bash hack/olm/render-bundle.sh ocp "$VERSION" dist/bundle-ocp
podman build -f dist/bundle-ocp/bundle.Dockerfile \
  -t "$BUNDLE_IMAGE" dist/bundle-ocp
podman push "$BUNDLE_IMAGE"
```

Prefer immutable image references, or set `USE_IMAGE_DIGESTS=true` when rendering
(requires Docker Buildx). Automated releases select the highest stable Krkn-AI
release for both images, strip its leading `v`, and pin OLM digests or Helm tags.
No AI image repository variables are required; test compatibility before promotion.

## 2. Install with AI disabled

```bash
export NS=krkn-olm-ai-test
oc create namespace "$NS"
operator-sdk run bundle "$BUNDLE_IMAGE" \
  --namespace "$NS" --install-mode OwnNamespace \
  --security-context-config restricted --timeout 10m

oc rollout status deployment/krkn-operator-operator -n "$NS" --timeout=5m
oc get csv,subscription -n "$NS"
oc get crd krknairuns.krkn.krkn-chaos.dev
oc get deployment krkn-operator-ai-service -n "$NS"
```

The final command should return `NotFound`: AI workloads are not installed yet.
For catalog installations, use a Subscription to an OpenShift bundle containing
this implementation, then continue below.

## 3. Enable AI

Find the Subscription:

```bash
SUB=$(oc get subscriptions -n "$NS" -o jsonpath='{.items[?(@.spec.name=="krkn-operator")].metadata.name}')
oc edit subscription "$SUB" -n "$NS"
```

Add or update `KRKNAI_ENABLED` at **`spec.config.env`**, preserving existing
environment entries and other `spec` fields. The nesting must be:

```yaml
spec:
  config:
    env:
      - name: KRKNAI_ENABLED
        value: "true"
```

Choose storage settings in the same `env` list **before first enabling**:

| Optional variable | Setting |
|---|---|
| `KRKNAI_STORAGE_CLASS_NAME` | StorageClass; empty uses the cluster default. |
| `KRKNAI_STORAGE_SIZE` | Managed claim size; default `1Gi` total for retained runs and staging. |
| `KRKNAI_STORAGE_EXISTING_CLAIM` | Existing claim in `$NS`; never created, modified, or deleted by the operator. |

Claims must be writable by the service's SCC-admitted UID/groups. Changing a claim
or StorageClass does not migrate artifacts; back up and migrate explicitly.
For private AI images, set `KRKNAI_IMAGE_PULL_SECRETS` to a JSON list such as
`[{"name":"registry-pull-secret"}]`.

Alternatively, after choosing storage settings, enable AI without manually
editing YAML. This preserves other environment entries and Subscription fields:

```bash
oc get subscription "$SUB" -n "$NS" -o json |
  jq '.spec.config.env = (
    [(.spec.config.env // [])[] | select(.name != "KRKNAI_ENABLED")] +
    [{"name": "KRKNAI_ENABLED", "value": "true"}]
  )' |
  oc replace -f -
```

The fetched resource version protects against concurrent edits; on a conflict,
rerun the whole pipeline. Confirm the saved Subscription returns `true`:

```bash
oc get subscription "$SUB" -n "$NS" \
  -o jsonpath='{.spec.config.env[?(@.name=="KRKNAI_ENABLED")].value}{"\n"}'
```

Wait for OLM to apply the setting and roll out both Deployments:

```bash
oc wait deployment/krkn-operator-operator -n "$NS" \
  --for='jsonpath={.spec.template.spec.containers[?(@.name=="manager")].env[?(@.name=="KRKNAI_ENABLED")].value}=true' \
  --timeout=3m
oc rollout status deployment/krkn-operator-operator -n "$NS" --timeout=5m
oc rollout status deployment/krkn-operator-ai-service -n "$NS" --timeout=5m
oc get pvc,pods -n "$NS"
```

Configure the Subscription, not the Deployment or CSV; OLM owns those objects.

## 4. Verify a run

```bash
oc port-forward -n "$NS" service/krkn-operator-console 18081:8080
```

Open `http://127.0.0.1:18081` and sign in. No Route is created automatically.
Select only the disposable target, discover it, save a configuration, and start
a small run. Check child scenarios, both Pod containers, and committed results:

```bash
oc get krknairuns,krknscenarioruns,pods -n "$NS"
oc logs -n "$NS" POD_NAME -c orchestrator
oc logs -n "$NS" POD_NAME -c result-uploader
```

The [AI API reference](../README.md#krkn-ai-operator-api) covers REST endpoints.
Its status endpoint reports configured enablement, not service/storage readiness.

## Disable, upgrade, and remove

- **Disable:** set `KRKNAI_ENABLED=false` in the Subscription. New operations are
  blocked; active runs, cleanup, artifact access, service, token, and PVC remain.
- **Upgrade:** build a newer bundle and run
  `operator-sdk run bundle-upgrade "$NEW_BUNDLE_IMAGE" --namespace "$NS" --timeout 10m`.
  Verify existing results and storage remain accessible. The singleton RWO service
  uses `Recreate`, so expect a brief outage; avoid untested upgrades during runs.
- **Remove:** export results and finish/delete unwanted runs while their finalizers
  can still execute, then run `operator-sdk cleanup krkn-operator --namespace "$NS"`.
  OLM leaves CRDs and operator-created AI resources behind. Clean up only this
  installation's AI resources and namespace-specific SCC binding; retain the token
  and PVC until deliberately discarding data. Never delete a user-supplied claim
  incidentally. Deleting the namespace can also destroy its retained artifacts.

## Troubleshooting

```bash
oc logs deployment/krkn-operator-operator -n "$NS" -c manager
oc logs deployment/krkn-operator-operator -n "$NS" -c olm-bootstrap-resources
oc logs deployment/krkn-operator-ai-service -n "$NS"
oc describe pvc krkn-operator-ai-service -n "$NS"
oc get events -n "$NS" --sort-by=.lastTimestamp
```

Check image pulls, storage/volume permissions, SCC admission, and NetworkPolicy
connectivity. The orchestrator's dedicated ServiceAccount gets `anyuid`; the
service stays non-root with SCC-assigned volume groups. Do not share Secret
contents or kubeconfigs in diagnostics.

### Reset a disposable test installation

Use only a dedicated test namespace: this deletes its Secrets/PVCs and may
permanently destroy stored artifacts. Export anything you need first.

Delete runs while the operator and AI service are still running, then uninstall:

```bash
: "${NS:?Set NS to your disposable test namespace}" &&
oc delete krknairuns,krkngraphruns,krknscenarioruns --all -n "$NS" --timeout=5m &&
operator-sdk cleanup krkn-operator --namespace "$NS" --timeout 5m &&
oc delete namespace "$NS" --timeout=5m
```

If the run CRDs were never installed, skip the run-deletion command. If deletion
hangs, fix the operator/service first rather than force-removing finalizers.
Shared CRDs and cluster-scoped RBAC remain; do not delete resources used by other
installations. After the namespace is gone, rerun **step 2** to create it and
install the bundle again, then enable AI.

