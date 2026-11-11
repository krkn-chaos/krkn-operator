# Krkn Operator MCP Server

The krkn-operator ships an opt-in [Model Context Protocol](https://modelcontextprotocol.io) (MCP)
server so AI agents can investigate and operate chaos runs on your behalf.

## What it is

- A single **Streamable HTTP** MCP endpoint at **`/api/mcp`**, deployed with the operator and
  **disabled by default**.
- Access is **robot-only**: regular users (JWT) cannot call it, and robots cannot use the REST or
  WebSocket APIs. Robots authenticate with opaque, admin-issued bearer tokens.
- Permissions are the **same group-based cluster permissions** (view / run / cancel) as human
  users, resolved from the robot's group memberships. Nothing a human could not do is possible
  for a robot, and vice versa.
- The transport is **stateless** and returns plain JSON responses, so it works behind the
  multi-replica Service and the console Route/Ingress without session affinity.

### Security model

- **Client-side confirmation for writes.** The server cannot prompt anyone. Tools that start,
  replay, cancel or delete runs are annotated `readOnly=false` / `destructive=true`; configure
  your MCP client to require human approval before calling `krkn_run_*`, `krkn_replay_*`,
  `krkn_cancel_*` and `krkn_delete_*` tools. Every call is still permission-checked and audited
  server-side.
- **Redaction.** Scenario-run and graph configurations are returned with environment variable
  *names* and file *references* only — never values or file contents. Workflows and graph
  definitions are redacted the same way. Cluster lists, scores, counts and node messages are
  filtered to clusters the robot can view; aggregates are recomputed from visible data only.
- **`krkn_get_job_logs`** returns authorized, bounded log excerpts. **Logs may contain sensitive
  output** produced by the scenario itself.
- **Reports** are status-only: the MCP tools tell you whether a report was generated, but HTML
  and PDF downloads are not available through MCP. Report *content* retrieval is out of scope for
  this feature and tracked separately.
- **Workflows** are read- and execute-only. Robots cannot create, edit or delete them.
- **Tokens** are shown once at issuance, stored only as SHA-256 hashes, expire after **90 days**,
  and are rotatable and revocable. Revocation takes effect on the very next request. Tokens are
  additionally bound to the robot's immutable Kubernetes UID: deleting and recreating a robot with
  the same name never transfers old credentials or run ownership to the new incarnation.
- **Submission idempotency.** `krkn_run_scenario` and `krkn_run_workflow` accept an optional
  `idempotencyKey` (1–64 characters of `[a-zA-Z0-9._-]`, starting and ending with an alphanumeric).
  Retrying the same key returns the previously created run instead of starting a duplicate
  experiment; the key is scoped to the submitting robot. Agents that may retry network calls should
  always send one. Without a key, retries can start duplicate experiments. Concurrent submissions
  racing on the same key are not deduplicated — the check covers the sequential-retry case.

## Enabling the endpoint

Helm (recommended):

```bash
helm upgrade krkn-operator charts/krkn-operator --set mcp.enabled=true ...
```

or by environment variable on the operator deployment:

```yaml
env:
  - name: KRKN_MCP_ENABLED
    value: "true"
```

The endpoint is then reachable:

- through the existing console Route/Ingress: `https://<console-host>/api/mcp` (the console nginx
  already proxies `/api/`; no console change is needed), or
- via a port-forward: `kubectl port-forward svc/<operator-service> 8080:8080` →
  `http://localhost:8080/api/mcp`.

## Quickstart (development cluster)

`scripts/mcp-live-demo.sh` deploys everything a dev needs against the cluster
in your current `oc` context and prints a ready-to-use robot token:

```bash
oc login <api-url> -u kubeadmin -p <password>
./scripts/mcp-live-demo.sh setup
./scripts/mcp-live-demo.sh call '{"name":"krkn_whoami","arguments":{}}'
```

It installs CRDs, bootstraps the scenario-runner RBAC, starts the operator
locally with `KRKN_MCP_ENABLED=true`, provisions a robot with view/run/cancel
on the target cluster, registers the target, and mints a token. The script
handles the non-obvious parts: the kubeconfig Secret must be JSON-wrapped as
`{"kubeconfig": "<base64>"}`, `KrknOperatorTarget.status.ready` must be set
via `--subresource=status` (plain apply drops status), and a second
`KrknOperatorTargetProvider` must NOT be created (the operator heartbeats its
own; duplicates block target-request completion). `cleanup` removes
everything. For production use the Helm chart above instead.

## Admin workflow

All robot administration is admin-only REST under `/api/v1/robots` (JWT, like the rest of the
operator API).

1. **Create a robot with group memberships** (groups must already exist):
   ```bash
   curl -X POST https://<host>/api/v1/robots \
     -H "Authorization: Bearer <admin-jwt>" -H "Content-Type: application/json" \
     -d '{"name": "ci-bot", "description": "CI automation", "groups": ["sre"]}'
   ```

2. **Issue a token** (shown exactly once; replaces any previous token):
   ```bash
   curl -X POST https://<host>/api/v1/robots/ci-bot/token \
     -H "Authorization: Bearer <admin-jwt>"
   # → {"token": "krkn_robot_ci-bot.<secret>", "tokenId": "...", "expiresAt": "...",
   #    "message": "Store this token now; it will not be shown again."}
   ```
   The token expires after **90 days**.

3. **Rotate** = issue again (`POST .../token`); the old token stops working immediately.

4. **Revoke** (permanent — the token Secret is deleted and can never be recovered;
   works whether or not the robot is disabled):
   ```bash
   curl -X DELETE https://<host>/api/v1/robots/ci-bot/token -H "Authorization: Bearer <admin-jwt>"
   ```

5. **Disable** (reversible suspension — rejects all tokens while disabled, but
   the token Secret is kept; re-enabling the robot makes the same unexpired
   token valid again immediately. Revoke instead for permanent invalidation):
   ```bash
   curl -X PATCH https://<host>/api/v1/robots/ci-bot \
     -H "Authorization: Bearer <admin-jwt>" -H "Content-Type: application/json" \
     -d '{"disabled": true}'
   ```

6. **Delete**:
   ```bash
   curl -X DELETE https://<host>/api/v1/robots/ci-bot -H "Authorization: Bearer <admin-jwt>"
   ```

`GET /api/v1/robots` lists robots with their token metadata (never token values). Deleting a group
also removes that group label from robots, so re-creating a group with the same name does not
silently restore robot access.

## Codex setup

```bash
export KRKN_ROBOT_TOKEN='<token>'
codex mcp add krkn --url https://<console-host>/api/mcp --bearer-token-env-var KRKN_ROBOT_TOKEN
```

The equivalent `~/.codex/config.toml` stanza (verified against `codex mcp add --help`):

```toml
[mcp_servers.krkn]
url = "https://<console-host>/api/mcp"
bearer_token_env_var = "KRKN_ROBOT_TOKEN"
```

`tool_timeout_sec` can be set per server if you expect long-running tool calls.

**Approval:** start Codex with `-a/--ask-for-approval <policy>` so destructive actions need human
confirmation, and instruct the agent to **require approval for all `krkn_run_*`, `krkn_replay_*`,
`krkn_cancel_*`, `krkn_delete_*` tools**. (Per-tool approval keys were not verifiable in the
installed Codex docs at the time of writing; use the agent instructions plus the global approval
policy.)

## Tool reference

| Tool | Kind | Server-side permission |
| --- | --- | --- |
| `krkn_whoami` | read | authenticated robot |
| `krkn_list_clusters` | read (may create a discovery target request) | lists only clusters with ≥ 1 action |
| `krkn_list_scenario_runs` | read | view on any target; output filtered |
| `krkn_get_scenario_run` | read | view on any target; output filtered |
| `krkn_get_job_logs` | read | view on that job's cluster |
| `krkn_get_scenario_run_config` | read | view on any target; redacted |
| `krkn_get_report_status` | read | view on any target; status only |
| `krkn_list_scenarios` | read | registry access (existing checks) |
| `krkn_get_scenario_detail` | read | registry access (existing checks) |
| `krkn_list_available_resources` | read | existing "available" filters |
| `krkn_list_graph_runs` | read | view on any resolvable target; filtered |
| `krkn_get_graph_run` | read | view on any resolvable target; filtered |
| `krkn_list_workflows` | read | existing file access |
| `krkn_get_workflow` | read | existing file access |
| `krkn_run_scenario` | write | run on all targets + file/registry/cloud-credential access |
| `krkn_replay_scenario_run` | write | as above, for the replayed configuration |
| `krkn_cancel_job` | write | cancel on that job's cluster |
| `krkn_delete_scenario_run` | write | cancel on all jobs |
| `krkn_run_workflow` | write | run on all targets + workflow/file/registry/cloud-credential access |
| `krkn_replay_graph_run` | write | as above, for the replayed graph |
| `krkn_delete_graph_run` | write | robot owner + cancel on every target cluster |
