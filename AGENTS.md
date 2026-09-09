# AGENTS.md

Reusable instructions for Krkn ecosystem repositories. `PROJECT.toml` selects the
project identity and technology-specific section; the ecosystem block is shared.

## Project Configuration

Before implementation, read `PROJECT.toml` completely and identify:

- `project.name`, `project.type`, description, and `ecosystem_projects`;
- configured public/private API paths, build/test/lint commands, and coverage policy;
- `beads.project_label`, central database, and export settings.

Verify commands and versions against current manifests, task definitions, and CI.
Do not invent missing configuration, labels, paths, or a default coverage target.
If required configuration is missing, read-only investigation can continue; ask
for the missing detail before tracked implementation or an ambiguous mutation.
Read any more-specific instructions before editing the affected directory.

## Shared Krkn Ecosystem Structure

<!-- SHARED ECOSYSTEM CONTEXT
Copy this entire section unchanged into each ecosystem AGENTS.md.
Keep repository-specific instructions outside these markers.
-->

Paths are relative to the common workspace; confirm which checkouts are available.

| Repository path | Responsibility |
| --- | --- |
| `krkn-operator-ecosystem/krkn-operator` | Main Kubernetes/OpenShift operator and REST API. |
| `krkn-operator-ecosystem/krkn-operator-console` | Operator web UI. |
| `krkn-operator-ecosystem/krkn-operator-acm` | ACM integration operator. |
| `krknctl` | Krkn CLI and shared Go library used across the Go projects. |
| `krkn` | Python core, containerized and orchestrated by `krknctl` and `krkn-operator`. |

- Put a change in the repository that owns the behavior. Share Go logic through
  existing `krknctl` APIs when appropriate; do not duplicate it in consumers or
  introduce speculative shared abstractions.
- For exported Go API changes, inspect affected consumers and their pinned module
  versions. A sibling checkout does not mean a build uses its local code.
- Changes to core scenario/container contracts can affect both orchestrators.
  REST API changes require checking affected console and ACM consumers.
- Preserve compatibility unless the task includes a coordinated breaking change.
  Read each affected repository's instructions before working there; this map
  does not authorize unrelated edits, dependency upgrades, or releases.
- Inspect only consumers relevant to the changed contract. Report unavailable
  checkouts and unverified integration assumptions. Do not commit temporary local
  module replacements.
- When explicitly updating this shared block, keep authorized copies consistent
  and identify copies that could not be updated.

<!-- END SHARED ECOSYSTEM CONTEXT -->

## Scope and Working Agreements

- Read-only questions, reviews, and plans do not authorize edits or issue writes.
  Implementation requests authorize focused local edits and safe relevant checks.
- Preserve existing contracts and unrelated user changes. Ask before a breaking
  change, new dependency, or material expansion into another repository.
- Do not stage, commit, push, publish, or alter live infrastructure merely because
  a session is ending. Perform those actions only when requested.
- Never disclose credentials, tokens, or kubeconfigs. Run chaos/integration tests
  only against a designated disposable target, not an arbitrary current context.
- Reuse existing patterns when they fit. Extract genuinely shared behavior, not
  every repeated line; do not expand the public API without a concrete consumer.
- Match the project's logging conventions. Handle errors with useful context;
  propagate cancellation/timeouts and avoid hiding failures behind success logs.

## Beads Task Tracking

Beads is the only persistent task tracker for implementation work. Do not create
parallel TODO/task/progress Markdown files or use a competing task system.
A short plan in the conversation is fine.

### Database and project identity

- Resolve the existing `.beads` target and configured database before writes.
  The supplied setup shares a central database across repositories; do not replace
  its symlink, run `bd init`, migrate it, or install hooks as incidental setup.
- Keep the existing shared issue prefix (`tsebastiani` in the supplied setup).
  `project.name` is not a command to change that prefix. Separate projects using
  the exact `beads.project_label` from `PROJECT.toml`.
- Use the installed version's `bd --help` / subcommand help when needed. Preserve
  the configured backend and sync mechanism; do not assume every release uses
  the same export format or hooks.
- In the supplied ecosystem workspace, `.beads` is an existing symlink to the
  shared central Beads database. Treat the symlink target as authoritative:
  inspect it before writes, and never replace the symlink, run `bd init`, or
  create a repository-local database as incidental setup.
- The repository's `beads/` directory is only a JSONL export for local reference
  or the configured synchronization workflow; it is not an independent Beads
  database. Do not use it as a substitute database or import it automatically.

### Issue lifecycle

1. Filter list/ready queries by the current project label. Reuse a matching issue
   before creating another; verify its scope with `bd show`.
2. Before coding, create an issue if necessary using `bd create` with the project
   label, a clear title/description, and acceptance criteria.
3. Mark it in progress with the installed version's supported update/claim flow.
   Check ownership first; do not take work already assigned to another person/agent.
4. Record progress, evidence, and blockers in that issue. Close it with `bd close`
   only when its acceptance criteria are satisfied, not simply when the session ends.
5. For an authorized cross-project task, use the destination project label and the
   existing `created-by:<source-project>` convention; link the related issues.

If Beads is required but unavailable, report the blocker before coding instead
of silently skipping tracking, creating a new database, or inventing issue IDs.

### Exports and Git

Follow the checkout's actual tracking/ignore policy for `beads/`. The legacy
instructions conflict on whether exports are tracked; do not change `.gitignore`
or force-add ignored files to resolve that conflict. Verify documented hooks
before relying on them. Issue closure does not imply a commit or remote sync.
When a commit is requested, use the project's conventional commit style and
reference the relevant issue.

## Search and RTK

- RTK is mandatory whenever it provides a wrapper for the command being run.
  Use `rtk` for all supported search, filesystem, Git, test, lint, build,
  package-manager, and language-tool commands to minimize human-readable output
  and token usage. This includes `rtk rg`, `rtk find`, `rtk git`, `rtk test`,
  `rtk npm`/`rtk npx`, and `rtk go` where applicable.
- Do not use the native command merely out of habit when an RTK wrapper exists.
  Use the native command only when no suitable wrapper exists, exact unfiltered
  output is required, or the command is a file-content/script input operation.
  For a supported command that needs raw output, use `rtk proxy` and state why.
- Start with scoped `rg --files` and `rg -n`; avoid dumping entire repositories.
  Read applicable instruction files completely and inspect relevant code bodies.
- Check RTK availability once when needed. Prefer supported wrappers for noisy
  human-readable output, such as `rtk git status`, `rtk git log -n 10`, or
  `rtk go test` with the project's original arguments.
- Preserve environment variables, build tags, package selection, and exit status.
  Do not add a new linter or change test scope just because RTK supports it.
- Use native commands, or `rtk proxy` when bypassing an installed rewrite hook,
  for exact file contents, final diff review, and output consumed by scripts or
  JSON parsers. Do not use signature-only summaries as editing evidence.
- If a summary is insufficient, inspect the reported raw/tee log first. Rerun only
  a safe, narrow diagnostic; never replay a mutating command just to recover output.
- If RTK is missing or incompatible, use the original command. Do not install or
  reconfigure it as an incidental part of an unrelated task.

## Quality and Verification

- Use build, test, lint, formatting, and coverage commands defined by the project;
  do not invent tool names, targets, package managers, or coverage percentages.
- Test new public behavior and bug regressions, including failure paths. Use
  table-driven cases where useful; isolate external dependencies in unit tests and
  add integration coverage where mocks cannot validate the contract.
- Run focused tests during iteration and broader checks proportional to impact.
  Use the full configured suite before an authorized commit when its prerequisites
  are available; report unavailable checks explicitly.
- Document changed public APIs, error cases, and complex usage. Update relevant
  user documentation when behavior changes, not unrelated README sections.
- Review complete diffs, including generated files. Do not weaken tests, validation,
  RBAC, TLS, or quality thresholds to hide failures; distinguish existing failures
  from regressions introduced by the change.

## Go Operators â€” Only for `project.type = "go-operator"`

- Follow configured package boundaries: reusable public APIs normally live in
  `pkg/`; private controllers/handlers in `internal/`. Keep CLI/process concerns
  out of imported libraries.
- Keep reconciliation idempotent and safe under retries. Use the existing
  controller-runtime error/requeue strategy; do not add retry loops that multiply
  its backoff or block reconciliation.
- Update status through the appropriate status operation, handle resource-version
  conflicts, and preserve spec ownership.
- Make finalizer cleanup retry-safe; use valid owner references and least-privilege
  RBAC. Test deletion, partial failure, and repeated reconciliation paths.
- Propagate `context.Context`; document exported contracts. Regenerate CRDs/RBAC
  and related outputs with the repository's configured targets when inputs change.

## React Frontends â€” Only for `project.type = "react-frontend"`

- Follow existing TypeScript, component, routing, API-client, styling, and state
  conventions. Do not reorganize the app just to match an example folder layout.
- Preserve loading, empty, error, and success states, stable list keys, semantic
  HTML, keyboard access, and responsive behavior.
- Keep request/state effects correct under re-renders and cancellation. Use
  memoization only when justified by an actual performance need; inline callbacks
  are not inherently forbidden.
- Test user interactions with the configured tools. Run type/lint/build checks as
  applicable and inspect changed UI at relevant viewports when available.
  Report when visual verification could not be performed.

For other project types, use that repository's specific instructions and tooling;
do not apply either technology section by analogy.

## Handoff

Report the implemented behavior, exact checks and outcomes, relevant Beads issue,
and remaining blockers or cross-repository follow-ups. Distinguish local changes
from committed/pushed work. Stop when the requested scope is complete.
