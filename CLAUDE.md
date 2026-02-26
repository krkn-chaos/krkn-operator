# CLAUDE.md - krkn-operator Project Instructions

## Project Overview

This is the **krkn-operator**, a Kubernetes operator part of the krkn-operator-ecosystem. It provides chaos engineering capabilities and serves as a core component that other operators in the ecosystem depend on.

### Ecosystem Architecture

The krkn-operator-ecosystem consists of multiple projects (located one folder above this project):

- **krkn-operator** (this project) - Core operator with shared reusable code in `pkg/`
- **krkn-operator-acm** (`../krkn-operator-acm`) - Integration operator with Red Hat Advanced Cluster Management (ACM)
- **krkn-operator-console** (`../krkn-operator-console`) - Frontend/UI for the ecosystem

**Important**: Decisions and changes made in krkn-operator often require reading or modifying the other ecosystem projects. Always consider the impact on krkn-operator-acm and krkn-operator-console when making changes to shared code in `pkg/`.

## Critical Rules

### Task Management

**MANDATORY: Use beads exclusively for all task tracking**
- ✅ ALWAYS use `bd create` to create tasks before starting work
- ✅ ALWAYS use `bd ready` to find available work
- ✅ ALWAYS use `bd update <id> --status=in_progress` when starting a task
- ✅ ALWAYS use `bd close <id>` when completing work
- ❌ NEVER start coding without creating a beads issue first

**PROHIBITED - Do NOT create these files for task tracking:**
- ❌ NEVER use TodoWrite or TaskCreate tools
- ❌ NEVER create TODO.md, tasks.md, TASKS.md, PROGRESS.md, or any markdown files for tasks
- ❌ NEVER create checklists in markdown files
- ❌ NEVER create issue lists or tracking files
- ✅ Use ONLY beads (`bd` commands) for all task and issue tracking

### Code Organization & Reusability

**Package Structure:**
- **`pkg/`** - Public, reusable code shared across the krkn-operator-ecosystem
  - All code that other operators in the ecosystem need to import MUST go here
  - This is the public API of krkn-operator
  - Examples: shared types, utilities, client libraries, common interfaces
  - Must be well-documented with godoc comments
  - Must maintain backward compatibility or use proper versioning

- **`internal/`** - Private implementation details
  - Code specific to this operator only
  - Cannot be imported by other operators
  - Controllers, webhooks, internal API handlers

**Code Duplication:**
- ❌ NEVER duplicate code - always refactor common logic into reusable functions
- ✅ If code appears in 2+ places, extract it into a shared function/package
- ✅ Identify reusable patterns early and move them to `pkg/` if they could benefit other operators
- ✅ Use composition and interfaces to promote reusability

### Code Quality Standards

**This is PRODUCTION code. Every change must meet these standards:**

**Testing:**
- ✅ MANDATORY: Write tests for ALL new code
- ✅ Unit tests for all public functions in `pkg/`
- ✅ Table-driven tests for functions with multiple scenarios
- ✅ Mock external dependencies (Kubernetes clients, APIs)
- ✅ Minimum 80% code coverage for new code
- ✅ Run `make test` before committing
- ✅ Add integration tests for controller reconciliation logic

**Documentation:**
- ✅ MANDATORY: Add godoc comments for all exported types, functions, and constants
- ✅ Package-level documentation in `doc.go` files
- ✅ Explain WHY, not just WHAT (include context and rationale)
- ✅ Document error conditions and edge cases
- ✅ Include usage examples for complex APIs
- ✅ Update README.md when adding new features

**Code Reviews & Quality:**
- ✅ Follow Go best practices and idioms
- ✅ Handle ALL errors explicitly - never ignore errors
- ✅ Use meaningful variable and function names
- ✅ Keep functions small and focused (Single Responsibility Principle)
- ✅ Add validation for all inputs
- ✅ Use structured logging (never fmt.Println in production code)
- ✅ Run `make lint` and fix all issues before committing

## Development Workflow

### Before Writing Code
1. Create a beads issue: `bd create --title="..." --description="..." --type=task --priority=2`
2. Review the issue: `bd show <id>`
3. Mark as in progress: `bd update <id> --status=in_progress`
4. Read existing code to understand patterns and avoid duplication
5. Identify if the code should go in `pkg/` (reusable) or `internal/` (private)

### While Writing Code
1. Write the minimum code needed to solve the problem
2. Identify and extract duplicated logic immediately
3. Write tests alongside the implementation
4. Add godoc comments for all exports
5. Run tests frequently: `make test`

### Before Committing
1. Run full test suite: `make test`
2. Run linters: `make lint`
3. Verify code coverage is adequate
4. Review your own changes for duplication
5. Ensure all exports in `pkg/` are documented
6. Close beads issue: `bd close <id>`
7. Sync beads: `bd sync`
8. Commit with meaningful message

### Session End Protocol
**CRITICAL - Always complete these steps:**
```bash
git status              # Check what changed
git add <files>         # Stage code changes
bd sync                 # Commit beads changes
git commit -m "..."     # Commit code
bd sync                 # Commit any new beads changes
git push                # Push to remote
```

## Kubernetes Operator Best Practices

- Use controller-runtime patterns and conventions
- Implement proper reconciliation loops with requeue logic
- Always update status subresources separately from spec
- Handle finalizers correctly for cleanup logic
- Use owner references for dependent resources
- Implement proper RBAC with minimal permissions
- Add validation webhooks for CRDs when needed
- Use exponential backoff for retries
- Log at appropriate levels (Info for normal operations, Error for failures)

## Architecture Principles

- **Separation of Concerns**: Controllers, API handlers, business logic should be separate
- **Dependency Injection**: Pass dependencies explicitly, avoid global state
- **Interface-Based Design**: Use interfaces in `pkg/` for flexibility
- **Testability**: Design code to be easily testable with mocks
- **Fail Fast**: Validate early, return errors immediately
- **Idempotency**: All reconciliation logic must be idempotent

## When in Doubt

- Check existing code patterns in the project
- Favor simplicity over cleverness
- Ask for clarification before making architectural decisions
- Document assumptions and trade-offs
- Create a beads issue to track follow-up work

## Communication

- All git commits should reference beads issues when relevant
- Use conventional commit format: `type(scope): description`
- PR descriptions should explain WHY, not just WHAT
- Include test results in PR descriptions

---

**Remember**: This is production code used by the ecosystem. Quality, documentation, and reusability are not optional.