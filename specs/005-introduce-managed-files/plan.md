# Implementation Plan: Managed Files and Folders

**Branch**: `005-introduce-managed-files` | **Date**: 2026-09-05 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/005-introduce-managed-files/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

Add a `git.managed.files` / `git.managed.folders` section to `config.yaml` that applies on
every `init` run regardless of which agent is selected. Managed files behave exactly like an
agent's declared files (create-or-overwrite, never deleted). Managed folders go further: after
writing every file the repository currently provides under that folder, `init` removes any file
already at the destination under that folder that the repository does not provide — a full
mirror, not a merge. Per the resolved clarifications, a partial-failure run is left as-is (no
rollback), a file/directory type conflict fails fast without deleting anything, and any resulting
error instructs the user to re-run the command.

The implementation reuses feature 003/004's existing write-set machinery in
`internal/reposync.Sync` almost unchanged: managed folders are unioned into the same
declared-folder write-set computation (so writing and type-conflict detection are already
correct for free), and a new, separate pruning pass — run only after all writes succeed — walks
each managed folder's destination subtree and removes any file not present in the freshly
retrieved repository content.

## Technical Context

**Language/Version**: Go 1.26.5 (per go.mod; matches Constitution's "current stable minor
release or newer")

**Primary Dependencies**: `github.com/go-git/go-git/v6` (already used for the in-memory clone;
no new dependency), Go standard library only for the new pruning logic
(`path/filepath`, `os`, `io/fs`)

**Storage**: Local filesystem only (destination directory); no database or external storage

**Testing**: `go test` (table-driven, colocated `_test.go` files), matching existing
`internal/agentcatalog` and `internal/reposync` test conventions

**Target Platform**: macOS, Linux, Windows (per Constitution's Technology & Dependency
Constraints)

**Project Type**: Single Go CLI project (existing `cmd/highway`, `internal/cli`,
`internal/agentcatalog`, `internal/reposync` layout — no new top-level structure needed)

**Performance Goals**: No new network calls (managed files/folders are retrieved from the same
single clone already performed for the selected agent); pruning cost scales with the number of
files already at the destination under declared managed folders only, not the whole destination
tree

**Constraints**: Must remain backward compatible when `git.managed` is absent from
`config.yaml` (existing features 001-004 behavior unchanged); must not introduce a second clone
or network round-trip

**Scale/Scope**: Two new struct fields, one new `reposync.Sync` parameter, one new internal
pruning helper, wiring in `internal/cli/init.go`; no new package

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Code Quality**: No new abstractions beyond one small `ManagedConfig` struct and one
  pruning helper function; both get doc comments per Principle I. Reuses existing
  `checkDeclaredPathsExist`/`selectWriteSet` rather than duplicating validation/selection logic. ✅
- **II. Simplicity**: Deliberately avoids introducing a new exported type/interface for
  "write-set member" — managed files are unioned into the existing `declaredFiles` list by the
  caller, and only one new parameter (`managedFolders`) is added to `Sync`, because managed
  folders are the only case needing new (pruning) behavior. No speculative extensibility hooks
  added. ✅
- **III. Testing Standards**: New pruning behavior, the type-conflict-never-deletes guarantee,
  the best-effort/no-rollback guarantee, and the missing-declared-managed-path validation error
  all require new table-driven tests in `internal/reposync` and `internal/agentcatalog`, plus
  `internal/cli` wiring tests — matching the NON-NEGOTIABLE testing principle. ✅
- **IV. UX Consistency**: Reuses the existing stderr/exit-code conventions and the "Re-run the
  command to retry." instruction already added for feature 004; no new output format introduced. ✅
- **V. Performance**: No new network calls — managed content comes from the same single clone
  already performed per `init` run; pruning only walks declared managed-folder subtrees, not the
  entire destination. ✅

No violations. Complexity Tracking section is not needed.

## Project Structure

### Documentation (this feature)

```text
specs/005-introduce-managed-files/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md         # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
internal/
├── agentcatalog/
│   ├── config.yaml           # existing; git.managed.files/folders already scaffolded
│   ├── catalog.go             # add ManagedConfig struct, GitConfig.Managed field,
│   │                          # extend validateDeclaredPaths to cover managed paths
│   └── catalog_test.go        # add coverage for managed path validation
├── reposync/
│   ├── reposync.go            # add managedFolders param to Sync; add pruneStaleManagedFiles
│   └── reposync_test.go       # add coverage for mirror/prune/type-conflict/no-rollback cases
└── cli/
    ├── init.go                # union managed files into files passed to syncRepo; pass
    │                          # catalog.Git.Managed.Folders as the new managedFolders arg
    └── init_test.go           # add coverage confirming managed files/folders are applied
                                # regardless of selected agent
```

**Structure Decision**: Single Go CLI project; feature is implemented entirely within the three
existing internal packages above (`agentcatalog`, `reposync`, `cli`). No new package is
warranted — the new behavior is a modest extension of each package's existing responsibility
(catalog parsing/validation, filesystem sync, CLI wiring), consistent with Principle II.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No violations — this section is intentionally empty.

