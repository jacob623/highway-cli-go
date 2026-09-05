<!--
Sync Impact Report
- Version change: 1.0.0 → 1.1.0
- Rationale: Materially expanded guidance added to Technology & Dependency
  Constraints, making the existing "actively maintained" requirement an
  explicit, checkable pre-proposal gate (MINOR).
- Modified principles: n/a
- Added sections: none (existing Technology & Dependency Constraints section
  gained one new bullet)
- Removed sections: none
- Templates checked for alignment:
  - .specify/templates/plan-template.md — Constitution Check gate references
    this file directly; no update needed (reads at runtime). ✅
  - .specify/templates/spec-template.md — no constitution-specific references. ✅
  - .specify/templates/tasks-template.md — no constitution-specific references. ✅
  - .specify/templates/checklist-template.md — no constitution-specific references. ✅
- Follow-up TODOs: none
-->

# Highway CLI Constitution

## Core Principles

### I. Code Quality

Code merged into this repository MUST be readable, idiomatic, and maintainable
before it is merged. Specifically:

- All Go code MUST be formatted with `gofmt` and MUST pass `go vet` and the
  project's linter with zero warnings before merge; these checks are
  non-negotiable CI gates, not advisory.
- Errors MUST be handled explicitly at the point they occur (returned,
  wrapped with context, or logged); errors MUST NOT be silently discarded.
- Exported identifiers (types, functions, packages) MUST have doc comments
  explaining their purpose when that purpose is not obvious from the name.
- Every change MUST be reviewed by at least one other contributor before
  merge; the author MUST NOT merge their own unreviewed change.

**Rationale**: A CLI tool that grows over many iterations (as this one is
expected to) accumulates technical debt fastest through inconsistent
formatting, swallowed errors, and unreviewed changes. Automating and
enforcing these checks keeps the codebase approachable as it grows.

### II. Simplicity

Solve the current, specified problem with the least amount of structure that
does the job:

- Prefer the Go standard library; introduce a third-party dependency only
  when it removes meaningfully more complexity than it adds, and record the
  justification in the relevant plan's Complexity Tracking section.
- MUST NOT add configuration options, abstractions, interfaces, or
  extensibility hooks for hypothetical future needs (YAGNI); add them when a
  real, specified requirement needs them.
- Each package MUST have a single, clearly stated responsibility. Prefer
  splitting an overloaded package over adding unrelated responsibilities to
  an existing one.

**Rationale**: Simplicity keeps the tool easy to extend as new agents,
commands, and configuration are added over time, and keeps the review burden
in Principle I manageable.

### III. Testing Standards (NON-NEGOTIABLE)

Automated tests are a required part of every change, not an afterthought:

- Every new function, command, or behavior change MUST ship with automated
  tests covering both success and expected-failure paths before merge.
- Bug fixes MUST include a regression test that fails before the fix and
  passes after it.
- Tests MUST be table-driven where multiple input/output cases apply,
  following idiomatic Go testing conventions, and colocated as `_test.go`
  files next to the code under test.
- CI MUST run the full test suite (`go test ./...`) on every change, and a
  failing test suite MUST block merge.

**Rationale**: A CLI that persists state and reads user-editable/embedded
configuration has many edge cases (missing files, malformed input,
cancellation); tests are the only reliable way to keep those paths correct
as the tool evolves.

### IV. User Experience Consistency

The CLI MUST behave predictably across all commands so developers can build
a reliable mental model of the tool:

- Command names, flags, and help text MUST follow a single, consistent
  naming and formatting convention across the whole CLI.
- Errors MUST be written to stderr, MUST be actionable (state what went
  wrong and, where possible, how to fix it), and normal output MUST be
  written to stdout.
- Exit codes MUST be consistent: `0` for success, non-zero for any failure
  or user cancellation, across every command.
- The CLI MUST NOT hang waiting for input it cannot receive (e.g., no
  interactive terminal available); it MUST fail fast with a clear message
  instead.
- Human-readable output formats MUST remain stable across releases; breaking
  an existing output format requires a documented migration note (see
  Development Workflow below).

**Rationale**: Consistency is what makes a multi-command CLI learnable —
users should be able to predict how a new command behaves from how existing
ones behave.

### V. Performance Requirements

The CLI MUST stay fast and lightweight as it grows:

- Interactive commands MUST start and reach the point of user interaction
  in well under 1 second on typical developer hardware, excluding time spent
  waiting on user input.
- The CLI MUST work fully offline by default; network calls MUST NOT be on
  the critical path of a command unless that command's specified purpose
  requires network access.
- Resource usage (memory, file I/O) MUST scale with the size of the
  configuration/input actually being processed, not with unrelated fixed
  overhead.
- Performance-sensitive changes MUST be justified with a measurement
  (benchmark or timing) rather than assumed; regressions found by
  measurement MUST be fixed or explicitly justified before merge.

**Rationale**: A CLI tool is invoked frequently and interactively; slow or
network-dependent startup directly degrades the developer experience it
exists to support.

## Technology & Dependency Constraints

- The CLI targets Go (current stable minor release or newer) and MUST build
  and run on macOS, Linux, and Windows.
- New third-party dependencies MUST be minimal in scope, actively
  maintained, and justified in the introducing change's plan (Complexity
  Tracking); prefer one well-established library over multiple overlapping
  ones for the same concern.
- Before proposing any new third-party library, the proposer MUST check that
  library's public commit history and confirm its most recent commit is
  dated no more than 1 month before the proposal date. A library that does
  not meet this bar MUST NOT be proposed unless the plan's Complexity
  Tracking section documents an explicit justification for the exception.
- The tool MUST NOT transmit telemetry, usage data, or configuration content
  over the network without explicit, documented user opt-in.

## Development Workflow & Quality Gates

- All changes are proposed via pull request; CI MUST enforce formatting
  (`gofmt`), `go vet`, linting, and the full test suite as required, blocking
  merge on any failure.
- Every feature plan produced by `/speckit-plan` MUST evaluate its design
  against this constitution's Constitution Check gate, both before Phase 0
  research and again after Phase 1 design; unresolved violations MUST be
  justified in that plan's Complexity Tracking table or the design MUST be
  simplified.
- Any change that breaks existing CLI output, flags, exit codes, or
  persisted file formats MUST document the break and a migration path in
  the pull request description.

## Governance

This constitution supersedes any conflicting practice, convention, or prior
undocumented norm in this repository. All pull requests and reviews MUST
verify compliance with the principles above; any requested complexity that
conflicts with Principle II (Simplicity) MUST be justified in writing in the
relevant plan.

Amendments are made by editing this file via pull request. Version bumps
follow semantic versioning:

- **MAJOR**: Backward-incompatible governance changes or removal/redefinition
  of an existing principle.
- **MINOR**: A new principle or materially expanded guidance is added.
- **PATCH**: Wording, typo, or other non-semantic clarifications.

Each amendment PR MUST update the version and `Last Amended` date below and
MUST include a Sync Impact Report (as an HTML comment at the top of this
file) summarizing what changed.

**Version**: 1.1.0 | **Ratified**: 2026-09-05 | **Last Amended**: 2026-09-05
