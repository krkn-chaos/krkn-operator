# Krkn-Operator Roadmap

**Updated:** 2026-09-30
**Cadence:** Release 1 on 2026-08-18, then 9-week cycles

---

## Release 1 — 2026-08-18
**Theme: Elasticsearch, Rerun, Workflow Templates**

| Issue | Summary | GitHub |
|-------|---------|--------|
| Save Elasticsearch details in admin view | Store ES connection config in the admin panel | [krkn-operator-console#45](https://github.com/krkn-chaos/krkn-operator-console/issues/45) |
| Replay scenario (rerunning) | Re-trigger a completed or failed scenario run | [krkn-operator-console#46](https://github.com/krkn-chaos/krkn-operator-console/issues/46) |
| Create savable workflow templates | Save a workflow configuration as a reusable template | [krkn-operator-console#47](https://github.com/krkn-chaos/krkn-operator-console/issues/47) |
| ✅ Allow custom name/tag on Scenario Runs | Let users label runs for easier identification | [krkn-operator-console#7](https://github.com/krkn-chaos/krkn-operator-console/issues/7) |
| ✅ Download logs as HTML/PDF/JSON | Export run logs in multiple formats | [krkn-operator-console#48](https://github.com/krkn-chaos/krkn-operator-console/issues/48) |
| Cluster proxy support in ACM/OCM | Support HTTP/HTTPS proxy for ACM-managed clusters | [krkn-operator#34](https://github.com/krkn-chaos/krkn-operator/issues/34) |
| Per-node resiliency score in graph scenario summary | Break down resiliency scores at the node level | [krkn-operator-console#49](https://github.com/krkn-chaos/krkn-operator-console/issues/49) |
| **Bug:** Fix cluster terminal error | Terminal crashes on certain cluster states | [krkn-operator-console#50](https://github.com/krkn-chaos/krkn-operator-console/issues/50) |
| **Bug:** Scenario runs page flickering every 3-4 seconds | Page refreshes cause disruptive UI flicker | [krkn-operator-console#10](https://github.com/krkn-chaos/krkn-operator-console/issues/10) |
| **Bug:** Scenario logs not persisted for failed runs | Logs disappear when a scenario run fails | [krkn-operator-console#11](https://github.com/krkn-chaos/krkn-operator-console/issues/11) |
| ✅ Add summary view to top of job list | Show aggregate status counts above the runs table | [krkn-operator-console#54](https://github.com/krkn-chaos/krkn-operator-console/issues/54) |
| ✅ Harden email & password validation in auth layer | Stricter validation on login and registration forms | [krkn-operator-console#29](https://github.com/krkn-chaos/krkn-operator-console/issues/29) |
| ✅ **Bug:** Terminal does not recognize `oc` command | `oc` is missing from PATH in the operator terminal | [krkn-operator-console#57](https://github.com/krkn-chaos/krkn-operator-console/issues/57) |

---

## Release 2 — 2026-10-20
**Theme: UX Hardening, Observability, Docs**

| Issue | Summary | GitHub |
|-------|---------|--------|
| Elastic runs summary chart view | Visualize run history from Elasticsearch as a chart | [krkn-operator-console#51](https://github.com/krkn-chaos/krkn-operator-console/issues/51) |
| ES runs and key metrics tab | Dedicated tab for Elasticsearch-sourced run data and metrics | [krkn-operator-console#52](https://github.com/krkn-chaos/krkn-operator-console/issues/52) |
| Add alerts view into new tab | Surface Prometheus/alerting data in a dedicated tab | [krkn-operator-console#53](https://github.com/krkn-chaos/krkn-operator-console/issues/53) |
| Duplicate a failed run for re-run with edits | Clone a run's config and allow edits before re-submitting | [krkn-operator-console#9](https://github.com/krkn-chaos/krkn-operator-console/issues/9) |
| Import/export Scenario Runs with optional logs | Portability for run configs and results across environments | [krkn-operator-console#6](https://github.com/krkn-chaos/krkn-operator-console/issues/6) |
| ✅ Weighted per-node resilience score in Chaos Studio | Assign weights to workflow nodes and calculate weighted overall resilience score | [krkn-operator#80](https://github.com/krkn-chaos/krkn-operator/issues/80) |
| Install krkn-visualize and save details in dashboard | Integrate krkn-visualize setup and config into the operator | [krkn-operator-console#60](https://github.com/krkn-chaos/krkn-operator-console/issues/60) |
| Metrics and alert capturing/visualization on scenario details | Embed Prometheus metrics and alerts into scenario detail view | [krkn-operator-console#59](https://github.com/krkn-chaos/krkn-operator-console/issues/59) |
| Implement cluster explorer via gRPC server | Browse cluster resources through a gRPC-backed explorer | [krkn-operator#1](https://github.com/krkn-chaos/krkn-operator/issues/1) |
| ✅ Toggle switches for enable/disable fields | Replace checkboxes with toggles; collapse disabled subsections | [krkn-operator-console#4](https://github.com/krkn-chaos/krkn-operator-console/issues/4) |
| Save/modify/delete scenario templates | Full CRUD for scenario templates | [krkn-operator#35](https://github.com/krkn-chaos/krkn-operator/issues/35) |
| Ability to rollback failed scenario | Rollback cluster state after a scenario fails | [krkn-operator#36](https://github.com/krkn-chaos/krkn-operator/issues/36) |
| ✅ Add resilience score per single run | Show a resiliency score for each individual scenario run | [krkn-operator-console#55](https://github.com/krkn-chaos/krkn-operator-console/issues/55) |
| ✅ Upload cloud provider credentials | Store AWS/GCP/Azure credentials for cloud-targeted scenarios | [krkn-operator#43](https://github.com/krkn-chaos/krkn-operator/issues/43) |
| Ability to save a backup of runs and restore in new cluster | Save a backup of current configurations and runs for migration to a new cluster | [krkn-operator#63](https://github.com/krkn-chaos/krkn-operator/issues/63) |
| Improve developer READMEs | Better onboarding docs for contributors | [krkn-operator#38](https://github.com/krkn-chaos/krkn-operator/issues/38) |
| REST API documentation via Swagger | Auto-generate and expose API docs at `/swagger` | [krkn-operator#37](https://github.com/krkn-chaos/krkn-operator/issues/37) |
| Improve run scenario UX by displaying all execution options | Make Run Scenario and Chaos Studio clearly visible as separate execution options | [krkn-operator#81](https://github.com/krkn-chaos/krkn-operator/issues/81) |
| ✅ Add paging to users and groups view | Paginate users and groups tables for better performance with many entries | [krkn-operator-console#89](https://github.com/krkn-chaos/krkn-operator-console/issues/89) |
| **Bug:** Fix 403 error in deploy preview workflow | Permissions error blocks deploy preview runs | [krkn-operator#39](https://github.com/krkn-chaos/krkn-operator/issues/39) |
| **Bug:** Duplicate logs at end of job completion | Logs repeat in UI after job completes and changes status | [krkn-operator-console#88](https://github.com/krkn-chaos/krkn-operator-console/issues/88) |
| Add workflow run history and resilience score trends | View past runs of saved workflows with resilience scores to identify regressions | [krkn-operator#82](https://github.com/krkn-chaos/krkn-operator/issues/82) |
| ✅ Test matrix for OCM releases | Coverage tracking across OCM versions | |
| ✅ Make `maxRetries` more visible | Clarify retry behavior while a job is running | [krkn-operator#126](https://github.com/krkn-chaos/krkn-operator/issues/126) |
| ✅ Replay saved workflows | Allow saved workflow runs to be replayed | [krkn-operator#117](https://github.com/krkn-chaos/krkn-operator/issues/117) |
| ✅ Move report actions into the run overflow menu | Make report actions easier to discover in run details | [krkn-operator-console#133](https://github.com/krkn-chaos/krkn-operator-console/issues/133) |
| ✅ Add access groups to Elasticsearch configurations | Restrict access to saved Elasticsearch connections | [krkn-operator-console#122](https://github.com/krkn-chaos/krkn-operator-console/issues/122) |
| ✅ Fix Elasticsearch view column widths | Keep telemetry columns readable across the data view | [krkn-operator-console#121](https://github.com/krkn-chaos/krkn-operator-console/issues/121) |
| ✅ Reset global parameters when switching scenarios | Prevent stale Studio configuration from carrying across scenarios | [krkn-operator-console#102](https://github.com/krkn-chaos/krkn-operator-console/issues/102) |
| ✅ Extract shared global-parameter handling | Centralize reusable global parameter behavior | [krkn-operator-console#103](https://github.com/krkn-chaos/krkn-operator-console/issues/103) |
| ✅ Fix service-account setup in the install script | Ensure installation creates the required service account | [krkn-operator#99](https://github.com/krkn-chaos/krkn-operator/issues/99) |
| ✅ Preserve `customRunName` in Helm chart CRDs | Prevent chart installs from pruning the custom run name | [krkn-operator#85](https://github.com/krkn-chaos/krkn-operator/issues/85) |
| Upload workflow files | Load workflow definitions into Chaos Studio | [krkn-operator#118](https://github.com/krkn-chaos/krkn-operator/issues/118) |


---

## Release 3 — 2026-12-22
**Theme: Multi-cluster, Advanced Features, Test Coverage**

| Issue | Summary | GitHub |
|-------|---------|--------|
| Krkn-AI integration in the operator and console | Integrate Krkn-AI runs and results into the operator and web console | [krkn-operator#96](https://github.com/krkn-chaos/krkn-operator/pull/96), [console#144](https://github.com/krkn-chaos/krkn-operator-console/pull/144) |
| Opt-in Krkn-AI installation | Make Krkn-AI installation configurable through the operator | [krkn-operator#115](https://github.com/krkn-chaos/krkn-operator/issues/115) |
| Test matrix for ACM releases | Coverage tracking across ACM versions | |
| Auto-populate pod/node names and labels from cluster | Pull live cluster data into scenario config fields | [krkn-operator-console#2](https://github.com/krkn-chaos/krkn-operator-console/issues/2) |
| Add cluster liveness check to frontend | Surface cluster health status in the UI | [krkn-operator-console#58](https://github.com/krkn-chaos/krkn-operator-console/issues/58) |
| Allowed scenarios per group configuration | Restrict which scenarios each user group can run | [krkn-operator#41](https://github.com/krkn-chaos/krkn-operator/issues/41) |
| Marketplace for Chaos Studio workflows | Share and discover workflow templates based on real-world outages | [krkn-operator-console#87](https://github.com/krkn-chaos/krkn-operator-console/issues/87) |
| Increase test coverage | Expand unit and integration test suite | [krkn-operator#42](https://github.com/krkn-chaos/krkn-operator/issues/42) |
| Schedule jobs | Run scenarios at a defined time or cadence | [krkn-operator-console#101](https://github.com/krkn-chaos/krkn-operator-console/issues/101) |
| Event triggers in Chaos Studio | Start workflows from event-driven conditions | [krkn-operator-console#94](https://github.com/krkn-chaos/krkn-operator-console/issues/94) |
| Webhook triggers for external tools | Allow external systems to start chaos experiments | [krkn-operator#114](https://github.com/krkn-chaos/krkn-operator/issues/114) |
| Cache scenario images | Reduce repeated image resolution and pull overhead | [krkn-operator#141](https://github.com/krkn-chaos/krkn-operator/issues/141) |
| Automatically create workflow file type | Generate the workflow file type for saved workflows | [krkn-operator#110](https://github.com/krkn-chaos/krkn-operator/issues/110) |
| Generate and maintain API reference from CRDs | Keep operator API documentation synchronized with generated CRDs | [krkn-operator#91](https://github.com/krkn-chaos/krkn-operator/issues/91) |
| Enforce scenario image signature verification | Verify scenario images before execution | [krkn-operator#97](https://github.com/krkn-chaos/krkn-operator/issues/97) |
| Support bearer challenges for private image digests | Resolve signed images from registries that require bearer authentication | [krkn-operator#139](https://github.com/krkn-chaos/krkn-operator/issues/139) |
| Harden terminal API command allowlisting | Prevent command impersonation through name-only allowlists | [krkn-operator#93](https://github.com/krkn-chaos/krkn-operator/issues/93) |
| Improve cluster-name collision and identity error handling | Detect collisions proactively and return safer validation errors | [krkn-operator#95](https://github.com/krkn-chaos/krkn-operator/issues/95) |
| Fix stale generated-manifest CI coverage | Fail CI when generated CRDs are out of sync | [krkn-operator#86](https://github.com/krkn-chaos/krkn-operator/issues/86) |
| Improve API error handling in the console | Provide consistent 500 and permission error feedback | [krkn-operator-console#100](https://github.com/krkn-chaos/krkn-operator-console/issues/100) |
| Improve workflow replay navigation and preview mocks | Keep replay navigation and mock API responses aligned | [krkn-operator-console#95](https://github.com/krkn-chaos/krkn-operator-console/issues/95), [#132](https://github.com/krkn-chaos/krkn-operator-console/issues/132) |
| Clarify file mount-path configuration | Make file mounting behavior easier to understand | [krkn-operator-console#126](https://github.com/krkn-chaos/krkn-operator-console/issues/126) |
| Add health-check and global-parameter controls to Chaos Studio | Expose health-check options separately in Studio | [krkn-operator-console#104](https://github.com/krkn-chaos/krkn-operator-console/issues/104) |
| Hide secrets in scenario details | Prevent sensitive values from appearing in configuration views | [krkn-operator-console#93](https://github.com/krkn-chaos/krkn-operator-console/issues/93) |
| Fix duplicate pods per workflow node | Ensure each workflow node creates only its intended pod | [krkn-operator#128](https://github.com/krkn-chaos/krkn-operator/issues/128) |
| Create scenario pods as privileged only when requested | Reduce unnecessary pod privileges | [krkn-operator#125](https://github.com/krkn-chaos/krkn-operator/issues/125) |
---

## Release 4 — 2027-02-23
**Theme: Scale, Security, Community/Ecosystem**

| Issue | Summary | GitHub |
|-------|---------|--------|
| User management integration with OIDC | SSO via OpenID Connect for enterprise deployments | [krkn-operator#44](https://github.com/krkn-chaos/krkn-operator/issues/44) |
| Improve table sorting across views | Consistent, multi-column sorting on all data tables | [krkn-operator-console#61](https://github.com/krkn-chaos/krkn-operator-console/issues/61) |
| Cancel scenario permissions: toast → modal | Show a modal (not a toast) when user lacks cancel permission | [krkn-operator#9](https://github.com/krkn-chaos/krkn-operator/issues/9) |
| KrknOperatorTargetProviderConfig parameters grouping | Group target provider config params for better discoverability | [krkn-operator#45](https://github.com/krkn-chaos/krkn-operator/issues/45) |
| Scale testing / validate operator at scale | Stress test operator with many concurrent runs and clusters | [krkn-operator#46](https://github.com/krkn-chaos/krkn-operator/issues/46) |
| Explore OperatorHub distribution channel | Evaluate packaging and publishing via OperatorHub | [krkn-operator#47](https://github.com/krkn-chaos/krkn-operator/issues/47) |
| Release to Community OperatorHub.io | Publish operator to the community catalog | [krkn-operator#48](https://github.com/krkn-chaos/krkn-operator/issues/48) |

---

## Ongoing (not release-blocked)

| Issue | Summary | GitHub |
|-------|---------|--------|
| Blog post for Developer Preview | Announce the Developer Preview milestone publicly | |
| Graph-based scenarios + resiliency score | Support for graph scenario types with scoring | |
| Engage upstream ACM/OCM community | Build relationships and contributions in the OCM upstream | [krkn-operator#49](https://github.com/krkn-chaos/krkn-operator/issues/49) |
