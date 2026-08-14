# Krkn-Operator Roadmap

**Updated:** 2026-08-14
**Cadence:** Release 1 on 2026-08-18, then 9-week cycles

---

## Release 1 — 2026-08-18
**Theme: Elasticsearch, Rerun, Workflow Templates**

| Issue | Summary | GitHub |
|-------|---------|--------|
| Save Elasticsearch details in admin view | Store ES connection config in the admin panel | [krkn-operator-console#45](https://github.com/krkn-chaos/krkn-operator-console/issues/45) |
| Replay scenario (rerunning) | Re-trigger a completed or failed scenario run | [krkn-operator-console#46](https://github.com/krkn-chaos/krkn-operator-console/issues/46) |
| Duplicate a failed run for re-run with edits | Clone a run's config and allow edits before re-submitting | [krkn-operator-console#9](https://github.com/krkn-chaos/krkn-operator-console/issues/9) |
| Create savable workflow templates | Save a workflow configuration as a reusable template | [krkn-operator-console#47](https://github.com/krkn-chaos/krkn-operator-console/issues/47) |
| ✅ Allow custom name/tag on Scenario Runs | Let users label runs for easier identification | [krkn-operator-console#7](https://github.com/krkn-chaos/krkn-operator-console/issues/7) |
| Download logs as HTML/PDF/JSON | Export run logs in multiple formats | [krkn-operator-console#48](https://github.com/krkn-chaos/krkn-operator-console/issues/48) |
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
| Toggle switches for enable/disable fields | Replace checkboxes with toggles; collapse disabled subsections | [krkn-operator-console#4](https://github.com/krkn-chaos/krkn-operator-console/issues/4) |
| Save/modify/delete scenario templates | Full CRUD for scenario templates | [krkn-operator#35](https://github.com/krkn-chaos/krkn-operator/issues/35) |
| Ability to rollback failed scenario | Rollback cluster state after a scenario fails | [krkn-operator#36](https://github.com/krkn-chaos/krkn-operator/issues/36) |
| Add resilience score per single run | Show a resiliency score for each individual scenario run | [krkn-operator-console#55](https://github.com/krkn-chaos/krkn-operator-console/issues/55) |
| Upload cloud provider credentials | Store AWS/GCP/Azure credentials for cloud-targeted scenarios | [krkn-operator#43](https://github.com/krkn-chaos/krkn-operator/issues/43) |
| Ability to save a backup of runs and restore in new cluster | Save a backup of current configurations and runs for migration to a new cluster | [krkn-operator#63](https://github.com/krkn-chaos/krkn-operator/issues/63) |
| Improve developer READMEs | Better onboarding docs for contributors | [krkn-operator#38](https://github.com/krkn-chaos/krkn-operator/issues/38) |
| REST API documentation via Swagger | Auto-generate and expose API docs at `/swagger` | [krkn-operator#37](https://github.com/krkn-chaos/krkn-operator/issues/37) |
| Improve run scenario UX by displaying all execution options | Make Run Scenario and Chaos Studio clearly visible as separate execution options | [krkn-operator#81](https://github.com/krkn-chaos/krkn-operator/issues/81) |
| Add paging to users and groups view | Paginate users and groups tables for better performance with many entries | [krkn-operator-console#89](https://github.com/krkn-chaos/krkn-operator-console/issues/89) |
| **Bug:** Fix 403 error in deploy preview workflow | Permissions error blocks deploy preview runs | [krkn-operator#39](https://github.com/krkn-chaos/krkn-operator/issues/39) |
| **Bug:** Duplicate logs at end of job completion | Logs repeat in UI after job completes and changes status | [krkn-operator-console#88](https://github.com/krkn-chaos/krkn-operator-console/issues/88) |
| **Bug:** Zone-outages workflow failing on coral | Workflow errors during perfconf on coral cluster | [krkn-operator#40](https://github.com/krkn-chaos/krkn-operator/issues/40) |

---

## Release 3 — 2026-12-22
**Theme: Multi-cluster, Advanced Features, Test Coverage**

| Issue | Summary | GitHub |
|-------|---------|--------|
| Import/export Scenario Runs with optional logs | Portability for run configs and results across environments | [krkn-operator-console#6](https://github.com/krkn-chaos/krkn-operator-console/issues/6) |
| Auto-populate pod/node names and labels from cluster | Pull live cluster data into scenario config fields | [krkn-operator-console#2](https://github.com/krkn-chaos/krkn-operator-console/issues/2) |
| Implement cluster explorer via gRPC server | Browse cluster resources through a gRPC-backed explorer | [krkn-operator#1](https://github.com/krkn-chaos/krkn-operator/issues/1) |
| Add cluster liveness check to frontend | Surface cluster health status in the UI | [krkn-operator-console#58](https://github.com/krkn-chaos/krkn-operator-console/issues/58) |
| Allowed scenarios per group configuration | Restrict which scenarios each user group can run | [krkn-operator#41](https://github.com/krkn-chaos/krkn-operator/issues/41) |
| Metrics and alert capturing/visualization on scenario details | Embed Prometheus metrics and alerts into scenario detail view | [krkn-operator-console#59](https://github.com/krkn-chaos/krkn-operator-console/issues/59) |
| Add workflow run history and resilience score trends | View past runs of saved workflows with resilience scores to identify regressions | [krkn-operator#82](https://github.com/krkn-chaos/krkn-operator/issues/82) |
| Weighted per-node resilience score in Chaos Studio | Assign weights to workflow nodes and calculate weighted overall resilience score | [krkn-operator#80](https://github.com/krkn-chaos/krkn-operator/issues/80) |
| Marketplace for Chaos Studio workflows | Share and discover workflow templates based on real-world outages | [krkn-operator-console#87](https://github.com/krkn-chaos/krkn-operator-console/issues/87) |
| Increase test coverage | Expand unit and integration test suite | [krkn-operator#42](https://github.com/krkn-chaos/krkn-operator/issues/42) |

---

## Release 4 — 2027-02-23
**Theme: Scale, Security, Community/Ecosystem**

| Issue | Summary | GitHub |
|-------|---------|--------|
| User management integration with OIDC | SSO via OpenID Connect for enterprise deployments | [krkn-operator#44](https://github.com/krkn-chaos/krkn-operator/issues/44) |
| Install krkn-visualize and save details in dashboard | Integrate krkn-visualize setup and config into the operator | [krkn-operator-console#60](https://github.com/krkn-chaos/krkn-operator-console/issues/60) |
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
| Test matrix for ACM releases | Coverage tracking across ACM versions | |
| Test matrix for OCM releases | Coverage tracking across OCM versions | |
| Blog post for Developer Preview | Announce the Developer Preview milestone publicly | |
| Graph-based scenarios + resiliency score | Support for graph scenario types with scoring | |
| Engage upstream ACM/OCM community | Build relationships and contributions in the OCM upstream | [krkn-operator#49](https://github.com/krkn-chaos/krkn-operator/issues/49) |
