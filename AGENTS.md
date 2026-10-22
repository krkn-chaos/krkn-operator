# Agent Instructions

## Issue Tracking

This project uses **bd (beads)** for issue tracking.
Run `bd prime` for workflow context, or install hooks (`bd hooks install`) for auto-injection.

**Quick reference:**
- `bd ready` - Find unblocked work
- `bd create "Title" --type task --priority 2` - Create issue
- `bd close <id>` - Complete work
- `bd sync` - Sync with git (run at session end)

For full workflow details: `bd prime`


<!-- bv-agent-instructions-v1 -->

---

## Beads Workflow Integration

This project uses [beads_viewer](https://github.com/Dicklesworthstone/beads_viewer) for issue tracking. Issues are stored in `.beads/` and tracked in git.

### Essential Commands

```bash
# View issues (launches TUI - avoid in automated sessions)
bv

# CLI commands for agents (use these instead)
bd ready              # Show issues ready to work (no blockers)
bd list --status=open # All open issues
bd show <id>          # Full issue details with dependencies
bd create --title="..." --type=task --priority=2
bd update <id> --status=in_progress
bd close <id> --reason="Completed"
bd close <id1> <id2>  # Close multiple issues at once
bd sync               # Commit and push changes
```

### Workflow Pattern

1. **Start**: Run `bd ready` to find actionable work
2. **Claim**: Use `bd update <id> --status=in_progress`
3. **Work**: Implement the task
4. **Complete**: Use `bd close <id>`
5. **Sync**: Always run `bd sync` at session end

### Key Concepts

- **Dependencies**: Issues can block other issues. `bd ready` shows only unblocked work.
- **Priority**: P0=critical, P1=high, P2=medium, P3=low, P4=backlog (use numbers, not words)
- **Types**: task, bug, feature, epic, question, docs
- **Blocking**: `bd dep add <issue> <depends-on>` to add dependencies

### Session Protocol

**Before ending any session, run this checklist:**

```bash
git status              # Check what changed
git add <files>         # Stage code changes
bd sync                 # Commit beads changes
git commit -m "..."     # Commit code
bd sync                 # Commit any new beads changes
git push                # Push to remote
```

### Best Practices

- Check `bd ready` at session start to find available work
- Update status as you work (in_progress → closed)
- Create new issues with `bd create` when you discover tasks
- Use descriptive titles and set appropriate priority/type
- Always `bd sync` before ending session

<!-- end-bv-agent-instructions -->

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

## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

When the user types `$graphify`, use the installed graphify skill or instructions before doing anything else.

Rules:
- When graphify-out/graph.json exists, run `graphify query "<question>"` before your first grep, search or source-file read for a codebase question, and put this rule in every subagent prompt that explores code. Read raw files only for lines the graph or an earlier read already located. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- Dirty graphify-out/ files are expected after hooks or incremental updates; dirty graph files are not a reason to skip graphify. Only skip graphify if the task is about stale or incorrect graph output, or the user explicitly says not to use it.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).
- A committed graphify-out/graph.json is rewritten whole on every build, so a large diff to it in a pull request is expected. Do not flag it as churn or suggest skipping the graph rebuild or its hooks.
