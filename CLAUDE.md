# CLAUDE.md - Project Instructions

## ⚠️ IMPORTANT: Read Project Configuration First

**Before following these instructions, you MUST:**

1. **Read `PROJECT.toml`** in this directory
2. Extract `project.name` → This is your **project identifier** (used as beads `issue_prefix` and in all references)
3. Extract `project.type` → Determines which technology-specific sections apply to you
4. Extract `ecosystem_projects` → Related projects you may need to read or modify

**Throughout this file:**
- `{{PROJECT_NAME}}` refers to `project.name` from PROJECT.toml
- Technology-specific sections (e.g., "Go Operator Best Practices", "React Best Practices") only apply if `project.type` matches

---

## Project Overview

This project is part of an ecosystem. Check PROJECT.toml for:
- **Project name and description** - What this project does
- **Project type** - Technology stack and conventions
- **Ecosystem relationships** - Related projects that may be affected by your changes

**Important**: If you're working on shared/reusable code (check PROJECT.toml for public API paths), always consider the impact on other ecosystem projects before making changes.

---

## Critical Rules

### Task Management

**MANDATORY: Use beads exclusively for all task tracking**
- ✅ ALWAYS use `bd create` to create tasks before starting work
- ✅ ALWAYS use `bd ready` to find available work
- ✅ ALWAYS use `bd update <id> --status=in_progress` when starting a task
- ✅ ALWAYS use `bd close <id>` when completing work
- ❌ NEVER start coding without creating a beads issue first

**Beads Configuration:**
- This project uses a **centralized beads database** via symlink: `.beads -> $HOME/Projects/beads/.beads`
- All projects in the ecosystem share the same central database
- **Issue prefix**: `tsebastiani` (shared across all projects)
- **Project separation**: Each project uses label `project:{{FOLDER_NAME}}` (from PROJECT.toml `beads.project_label`)
- **CRITICAL**: ALWAYS include `-l {{PROJECT_LABEL}}` when creating issues
- **Viewing issues**: ALWAYS filter by label: `bd list -l {{PROJECT_LABEL}}`
- Local `./beads/` folder contains JSON exports (gitignored, for offline reference)

**Cross-Project Workflow:**
```bash
# Create issue for THIS project (read PROJECT_LABEL from PROJECT.toml):
bd create --title="Fix bug" -l project:krkn-operator --type=task

# Create issue for ANOTHER project (cross-project requirement):
bd create --title="Add login UI" \
  -l project:krkn-operator-console \
  -l created-by:krkn-operator \
  --description="Backend auth ready, console needs UI"

# View only THIS project's issues:
bd list -l project:krkn-operator

# View requirements from other projects:
bd list -l project:krkn-operator -l created-by:krkn-operator-acm
```

**PROHIBITED - Do NOT create these files for task tracking:**
- ❌ NEVER use TodoWrite or TaskCreate tools
- ❌ NEVER create TODO.md, tasks.md, TASKS.md, PROGRESS.md, or any markdown files for tasks
- ❌ NEVER create checklists in markdown files
- ❌ NEVER create issue lists or tracking files
- ✅ Use ONLY beads (`bd` commands) for all task and issue tracking

### Code Quality Standards

**This is PRODUCTION code. Every change must meet these standards:**

**Testing:**
- ✅ MANDATORY: Write tests for ALL new code
- ✅ Unit tests for all public functions
- ✅ Table-driven tests for functions with multiple scenarios
- ✅ Mock external dependencies (APIs, databases, external services)
- ✅ Minimum test coverage as specified in PROJECT.toml (typically 80%)
- ✅ Run project test command (check PROJECT.toml) before committing
- ✅ Add integration tests for critical workflows

**Documentation:**
- ✅ MANDATORY: Document all public APIs, types, and functions
- ✅ Module/package-level documentation explaining purpose and usage
- ✅ Explain WHY, not just WHAT (include context and rationale)
- ✅ Document error conditions and edge cases
- ✅ Include usage examples for complex APIs
- ✅ Update README.md when adding new features

**Code Reviews & Quality:**
- ✅ Follow language/framework best practices and idioms
- ✅ Handle ALL errors explicitly - never ignore errors
- ✅ Use meaningful variable and function names
- ✅ Keep functions small and focused (Single Responsibility Principle)
- ✅ Add validation for all inputs
- ✅ Use structured logging (never use print statements in production code)
- ✅ Run project lint command (check PROJECT.toml) and fix all issues before committing

### Code Organization & Reusability

Check PROJECT.toml for project-specific paths (e.g., `pkg_public_api`, `internal_api` for Go projects).

**Code Duplication:**
- ❌ NEVER duplicate code - always refactor common logic into reusable functions
- ✅ If code appears in 2+ places, extract it into a shared function/package
- ✅ Identify reusable patterns early and move them to public API paths if they could benefit other projects
- ✅ Use composition and interfaces to promote reusability

---

## Development Workflow

### Before Writing Code
1. Read PROJECT.toml to get `beads.project_label` (e.g., `project:krkn-operator`)
2. Create a beads issue with project label:
   ```bash
   bd create --title="..." --description="..." -l {{PROJECT_LABEL}} --type=task --priority=2
   ```
3. Review the issue: `bd show <id>`
4. Mark as in progress: `bd update <id> --status=in_progress`
5. Read existing code to understand patterns and avoid duplication
6. Check PROJECT.toml to understand where new code should go (public vs private API)

### While Writing Code
1. Write the minimum code needed to solve the problem
2. Identify and extract duplicated logic immediately
3. Write tests alongside the implementation
4. Add documentation for all public APIs
5. Run tests frequently (use command from PROJECT.toml)

### Before Committing
1. Run full test suite (command from PROJECT.toml)
2. Run linters (command from PROJECT.toml)
3. Verify code coverage is adequate (check minimum in PROJECT.toml)
4. Review your own changes for duplication
5. Ensure all public APIs are documented
6. Close beads issue: `bd close <id>`
7. Export beads snapshot (filtered by project label):
   ```bash
   bd export -l {{PROJECT_LABEL}} -o beads/{{FOLDER_NAME}}-issues.jsonl
   ```
8. Commit with meaningful message

### Session End Protocol
**CRITICAL - Always complete these steps:**
```bash
git status                                                    # Check what changed
git add <files>                                               # Stage code changes
bd export -l {{PROJECT_LABEL}} -o beads/{{FOLDER_NAME}}-issues.jsonl  # Export filtered snapshot
git commit -m "type(scope): description"                      # Commit code
git push                                                      # Push to remote
```

**Note**:
- The centralized beads database (`.beads` symlink) is automatically synced via git hooks
- The local `beads/` export is filtered by project label and is gitignored
- DO NOT add `beads/` to git - it's for offline reference only

---

## Technology-Specific Best Practices

### [Go Operator] - Only applies if project.type = "go-operator"

**Package Structure:**
- **`pkg/`** - Public, reusable code shared across the ecosystem
  - All code that other projects need to import MUST go here
  - This is the public API
  - Examples: shared types, utilities, client libraries, common interfaces
  - Must be well-documented with godoc comments
  - Must maintain backward compatibility or use proper versioning

- **`internal/`** - Private implementation details
  - Code specific to this project only
  - Cannot be imported by other projects
  - Controllers, webhooks, internal handlers

**Kubernetes Operator Best Practices:**
- Use controller-runtime patterns and conventions
- Implement proper reconciliation loops with requeue logic
- Always update status subresources separately from spec
- Handle finalizers correctly for cleanup logic
- Use owner references for dependent resources
- Implement proper RBAC with minimal permissions
- Add validation webhooks for CRDs when needed
- Use exponential backoff for retries
- Log at appropriate levels (Info for normal operations, Error for failures)

**Go-Specific Quality:**
- Use `golangci-lint` for linting
- Follow effective Go guidelines
- Use context.Context for cancellation and timeouts
- Prefer standard library when possible
- Use table-driven tests

### [React Frontend] - Only applies if project.type = "react-frontend"

**Project Structure:**
- **`src/components/`** - Reusable UI components
- **`src/pages/`** - Page-level components
- **`src/hooks/`** - Custom React hooks
- **`src/utils/`** - Utility functions
- **`src/api/`** - API client code

**React Best Practices:**
- Use functional components and hooks
- Implement proper error boundaries
- Use TypeScript for type safety
- Follow React naming conventions (PascalCase for components)
- Implement proper loading and error states
- Use proper key props in lists
- Avoid inline function definitions in JSX
- Use React.memo for expensive components

**Frontend Quality:**
- Use ESLint and Prettier
- Write component tests with React Testing Library
- Test user interactions, not implementation details
- Use semantic HTML
- Ensure accessibility (a11y) compliance
- Optimize bundle size

---

## Architecture Principles

- **Separation of Concerns**: Keep different aspects of the system separate
- **Dependency Injection**: Pass dependencies explicitly, avoid global state
- **Interface-Based Design**: Use interfaces/contracts for flexibility
- **Testability**: Design code to be easily testable with mocks
- **Fail Fast**: Validate early, return errors immediately
- **Idempotency**: Operations should be safe to retry

---

## When in Doubt

- Check existing code patterns in the project
- Review PROJECT.toml for project-specific guidance
- Favor simplicity over cleverness
- Ask for clarification before making architectural decisions
- Document assumptions and trade-offs
- Create a beads issue to track follow-up work

---

## Communication

- All git commits should reference beads issues when relevant
- Use conventional commit format: `type(scope): description`
  - Examples: `feat(api): add user authentication`, `fix(ui): resolve login button bug`
- PR descriptions should explain WHY, not just WHAT
- Include test results in PR descriptions

---

## Migrating CLAUDE.md to a New Project

This CLAUDE.md is designed to be **completely reusable**. To use it in a new project:

1. **Copy this CLAUDE.md** to your new project root (no edits needed!)

2. **Create PROJECT.toml** with your project details:
   ```toml
   [project]
   name = "your-project-name"
   type = "go-operator"  # or "react-frontend", etc.
   description = "Brief description"

   [beads]
   central_database = "/Users/tsebasti/Projects/beads"
   export_path = "./beads/"
   ```

3. **Add to .gitignore**:
   ```
   beads/
   ```

4. **Configure beads multi-repo** in `.beads/config.yaml`:
   ```yaml
   repos:
     primary: "/Users/tsebasti/Projects/beads"
     additional: []
   ```

5. **Configure beads prefix**:
   ```bash
   bd config set issue_prefix your-project-name
   ```

6. **Done!** CLAUDE.md now works with your project name automatically.

---

**Remember**: This is production code. Quality, documentation, and reusability are not optional.
