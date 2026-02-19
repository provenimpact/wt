<!-- Sync Impact Report
  Version change: (new) → 1.0.0
  Modified principles: N/A (initial creation)
  Added sections:
    - Core Principles (6): Git Safety First, CLI Simplicity,
      Test-First, Single Responsibility Packages,
      Prefer Libraries Over Command Wrapping,
      Minimal Dependencies & Fast Startup
    - Distribution & Compatibility
    - Development Workflow (includes Conventional Commits)
    - Governance
  Removed sections: N/A
  Templates requiring updates:
    - .specify/templates/plan-template.md ✅ no updates needed
    - .specify/templates/spec-template.md ✅ no updates needed
    - .specify/templates/tasks-template.md ✅ no updates needed
  Command files reviewed:
    - .opencode/command/speckit.*.md (9 files) ✅ no updates needed
  Follow-up TODOs: none
-->
# wt Constitution

## Core Principles

### I. Git Safety First

The tool MUST prevent data loss by default. All destructive
operations — removing worktrees with uncommitted changes, deleting
branches, pruning stale worktrees — MUST require explicit
confirmation or the `--force` flag.

- The tool MUST verify worktree and branch state before any
  mutation.
- Uncommitted changes, untracked files, and unpushed commits MUST
  trigger a blocking warning with a clear description of what
  would be lost.
- The `--force` flag MUST be the only way to bypass safety checks;
  no silent destructive defaults are permitted.

**Rationale**: `wt` wraps git worktree to simplify it.
Simplification MUST NOT come at the cost of safety. Users trust
the tool to protect their work.

### II. CLI Simplicity

Every command MUST do one thing well. The 80% use case MUST require
zero flags.

- Human-readable output by default; machine-parseable output (JSON)
  via `--json`.
- Exit codes MUST follow POSIX conventions: 0 = success,
  1 = general error, 2 = usage error.
- Error messages MUST include actionable guidance (what went wrong
  and what to do next).
- Help text MUST be concise and include examples for every
  subcommand.

**Rationale**: A worktree helper that requires memorizing flags
defeats its purpose. Opinionated defaults make the common path
fast.

### III. Test-First (NON-NEGOTIABLE)

TDD is mandatory. The development cycle is: write tests, verify
they fail (red), implement until they pass (green), then refactor.

- Every exported function MUST have unit test coverage.
- Every CLI subcommand MUST have an integration test that exercises
  real git operations against a temporary repository.
- Test helpers for creating/tearing down temporary git repos MUST
  be provided as a shared test utility package.
- `go test ./...` MUST pass before any code is merged.

**Rationale**: A tool that manages git worktrees operates on
irreplaceable user data. Rigorous testing is the primary defense
against regressions that cause data loss.

### IV. Single Responsibility Packages

Code MUST be organized into focused Go packages, each with a clear,
singular purpose.

- The `cmd/` layer MUST be thin: argument parsing and output
  formatting only. Business logic MUST live in library packages.
- Library packages MUST be independently testable without importing
  `cmd/`.
- No circular dependencies between packages.
- New packages MUST be justified — prefer extending an existing
  package over creating a new one unless the responsibility is
  clearly distinct.

**Rationale**: Thin CLI layers and focused packages enable thorough
unit testing and make the codebase navigable as it grows.

### V. Prefer Libraries Over Command Wrapping

`wt` MUST NOT reimplement git internals from scratch. When
interacting with git repositories, the implementation MUST prefer
well-maintained Go libraries (e.g., `go-git`) over shelling out to
the `git` binary when the library provides a better solution in
terms of correctness, testability, or error handling.

- Go libraries MUST be preferred when they offer: type-safe return
  values, structured error handling, or elimination of output
  parsing.
- Shelling out to the `git` binary is acceptable when: no adequate
  library support exists, the library has known correctness issues
  for the operation, or the git CLI is the canonical interface for
  that operation (e.g., `git worktree add` if library support is
  incomplete).
- Worktree state MUST always be derived from git at runtime; the
  tool MUST NOT maintain its own cache or state file.
- Users MUST be able to use raw `git worktree` commands alongside
  `wt` without conflict or corruption.
- The tool MUST fail gracefully if required git dependencies
  (binary or library) are missing or below the minimum supported
  version.

**Rationale**: Go libraries provide type safety, structured errors,
and eliminate fragile output parsing. However, not all git
operations have mature library support. The decision to use a
library vs. the git binary MUST be made per-operation based on
solution quality, not dogma.

### VI. Minimal Dependencies & Fast Startup

The binary MUST start in under 100ms on commodity hardware.

- External Go module dependencies MUST be justified and audited
  before adoption. Prefer the Go standard library except where
  Principle V mandates a library for superior correctness or
  testability (e.g., `go-git` for repository operations).
- No runtime configuration files are required — the tool MUST work
  with zero configuration out of the box.
- Optional configuration (e.g., default worktree base directory)
  MAY be supported but MUST NOT be mandatory.

**Rationale**: CLI tools are invoked frequently and often in
scripts. Slow startup or complex setup erodes trust and adoption.

## Distribution & Compatibility

- The primary distribution channels are **Homebrew**
  (`brew install wt`) and **`go install`**
  (`go install github.com/provenimpact/wt@latest`).
- The build MUST produce statically linked binaries for at minimum
  macOS (amd64, arm64) and Linux (amd64, arm64).
- The tool MUST support the two most recent major Go versions.
- The tool MUST function correctly on macOS and Linux. Windows
  support is a non-goal for v1 but the design MUST NOT introduce
  platform-specific assumptions that would block future Windows
  support.
- The minimum supported git version MUST be documented and tested
  against in CI.

## Development Workflow

- **Conventional Commits**: All commits MUST follow the
  [Conventional Commits](https://www.conventionalcommits.org/)
  specification. Valid types include: `feat`, `fix`, `docs`,
  `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`,
  `revert`. Breaking changes MUST use the `!` suffix or a
  `BREAKING CHANGE:` footer.
- **Branch naming**: Feature branches MUST follow the pattern
  `<number>-<short-name>` (e.g., `1-add-create-command`).
- **Pull requests**: Every PR MUST pass CI (tests, lint, build)
  before merge. PRs MUST reference the relevant spec or issue.
- **CI gates**: `go test ./...`, `go vet ./...`, and the project
  linter MUST pass on every push. Builds MUST succeed for all
  target platforms.
- **Release tagging**: Releases MUST use semantic versioning tags
  (e.g., `v1.2.3`). Tags MUST correspond to a passing CI build.

## Governance

This constitution is the highest-authority document for the `wt`
project. It supersedes all other practices, conventions, and
informal agreements.

- **Amendments** require: (1) a documented rationale, (2) review
  by at least one maintainer, and (3) a migration plan if the
  change affects existing code or workflows.
- **Versioning**: The constitution follows semantic versioning:
  - MAJOR: Principle removal or backward-incompatible redefinition.
  - MINOR: New principle or materially expanded guidance.
  - PATCH: Clarifications, wording, or non-semantic refinements.
- **Compliance review**: All PRs and code reviews MUST verify
  adherence to constitution principles. Violations MUST be
  resolved before merge.

**Version**: 1.0.0 | **Ratified**: 2026-02-18 | **Last Amended**: 2026-02-18
