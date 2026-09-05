# Implementation Plan: CLI Init Agent Selection

**Branch**: `001-cli-init-agent-select` | **Date**: 2026-09-05 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-cli-init-agent-select/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

A Go CLI provides an `init` command that loads a catalog of supported AI coding agents from a YAML file embedded in the binary at build time, presents the agents as an interactive single-select list, and prints a confirmation of the developer's choice. The selection is not persisted anywhere; each `init` run is independent. The embedded catalog is intentionally kept as its own YAML file (not hardcoded structs) because the set of agents — and the configuration captured per agent — is expected to grow over future iterations.

## Technical Context

**Language/Version**: Go 1.23

**Primary Dependencies**: `spf13/cobra` (command framework), `gopkg.in/yaml.v3` (YAML parsing for the embedded catalog), `charmbracelet/huh` (interactive single-select list prompt), `golang.org/x/term` (TTY detection)

**Storage**: One YAML file, `agents.yaml`, embedded into the binary via `go:embed`, ships the catalog of supported agents. No other persistent storage is used — the developer's selection is not saved anywhere (FR-011)

**Testing**: `go test` with table-driven tests colocated as `_test.go` files per package

**Target Platform**: Cross-platform command-line binary (macOS, Linux, Windows), run interactively in a terminal

**Project Type**: Single project — CLI

**Performance Goals**: Command must start and reach the point of user interaction in well under 1 second, excluding time spent waiting on developer input (Constitution Principle V — Performance Requirements)

**Constraints**: Must not hang when no interactive terminal is available (FR-010, Principle IV — User Experience Consistency); embedded catalog file must remain easy to extend with new fields per agent as the feature iterates (Principle II — Simplicity: extend via data, not new abstractions). This feature does not itself require network access; a future feature is expected to add git-repo-sourced files, so no offline-only constraint is being locked in here.

**Scale/Scope**: Single subcommand (`init`) today; catalog expected to hold on the order of tens of agent entries, not hundreds

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Evaluated against `.specify/memory/constitution.md` v1.0.0:

| Principle / Constraint | Status | Notes |
|---|---|---|
| I. Code Quality | PASS | Plan requires `gofmt`/`go vet`/lint-clean code and explicit error handling (missing/empty/malformed catalog are surfaced, not swallowed, per FR-006–FR-008); review happens via normal PR flow — no plan-level exception needed. |
| II. Simplicity | PASS | Catalog grows by editing YAML data, not by adding abstractions; each package (`agentcatalog`, `cli`) has one responsibility. No persisted-state package is needed since selections are not saved (FR-011). Four third-party dependencies are introduced — justified individually in Complexity Tracking below per the Technology & Dependency Constraints section. |
| III. Testing Standards (NON-NEGOTIABLE) | PASS | Testing strategy uses table-driven, colocated `_test.go` files and covers both success and failure paths (missing/empty/malformed catalog, cancellation, no-TTY) per quickstart.md Scenarios 2–4. |
| IV. User Experience Consistency | PASS | contracts/cli-init.md fixes stdout/stderr separation, non-zero exit on any failure/cancellation, and fail-fast (no hang) behavior for the no-TTY case (FR-010). |
| V. Performance Requirements | PASS | No network calls required for this feature's scope (embedded catalog only); sub-1-second startup goal stated above; no unbounded resource use (catalog scoped to tens of entries). A later feature is expected to add network access (git-sourced files) and will need its own evaluation against this principle. |
| Technology & Dependency Constraints | PASS (with justification) | Targets current stable Go, cross-platform (macOS/Linux/Windows); no telemetry. New dependencies (cobra, yaml.v3, huh, x/term) are justified in Complexity Tracking, each actively maintained per the constraint below. |
| Development Workflow & Quality Gates | PASS | This Constitution Check itself satisfies the required pre- and post-Phase-1 gate evaluation; no breaking output/flag/format changes exist yet to require a migration note. |

No unjustified violations. Proceed to Phase 0.

**Post-Phase 1 re-check**: Phase 0 (research.md) and Phase 1 (data-model.md, contracts/, quickstart.md) did not introduce anything beyond the Technical Context and dependency set already evaluated above. All rows above remain PASS after design; no new violations were introduced by the data model, contracts, or quickstart.

## Project Structure

### Documentation (this feature)

```text
specs/001-cli-init-agent-select/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
cmd/
└── highway/
    └── main.go              # entrypoint; wires the cobra root command

internal/
├── agentcatalog/
│   ├── agents.yaml          # embedded (go:embed) catalog: supported AI coding agents
│   ├── catalog.go           # embed + parse + validate (missing/empty/malformed/duplicates)
│   └── catalog_test.go
└── cli/
    ├── root.go              # cobra root command
    ├── init.go              # `init` command: load catalog, prompt, print confirmation
    └── init_test.go

go.mod
go.sum
```

**Structure Decision**: Single Go project (Option 1), adapted to idiomatic Go conventions: a
thin `cmd/highway` entrypoint delegates to a `cli` package built on cobra, with the agent
catalog logic isolated into its own `internal` package so it is independently unit-testable.
Tests are colocated as `_test.go` files next to the code they cover (standard Go practice)
rather than under a separate top-level `tests/` tree.

## Complexity Tracking

> Required by the Technology & Dependency Constraints section: every new third-party dependency
> must be justified here. No constitutional principle is being violated — this table documents
> why each dependency is the minimal viable choice rather than recording an exception.

| Dependency | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| `spf13/cobra` | Structures the `init` command today and the additional subcommands/flags expected as the tool iterates (per spec Assumptions and Technical Context) | Standard library `flag` package (rejected — no subcommand/help-text conventions, would need hand-rolled scaffolding that duplicates cobra as more commands are added) |
| `gopkg.in/yaml.v3` | Parses the embedded `agents.yaml` catalog | No YAML support in the standard library; hand-rolling a parser is far more complex than depending on the de facto standard library |
| `charmbracelet/huh` | Provides the interactive single-select list with built-in cancel (Ctrl+C) and invalid-input re-prompt handling required by FR-003/FR-004; actively maintained (commits within the last month), satisfying the Technology & Dependency Constraints requirement | `manifoldco/promptui` (rejected — no commits/releases in 5 years, no longer actively maintained); `AlecAivazis/survey` (rejected — officially archived/unmaintained, README recommends migrating away); hand-rolled numbered stdin prompt (rejected — reimplements cancellation and input-validation edge cases huh already handles) |
| `golang.org/x/term` | Detects a non-interactive terminal so `init` can fail fast instead of hanging (FR-010) | No portable, cross-platform TTY-detection primitive exists in the standard library; this is the standard extension used for exactly this check |
