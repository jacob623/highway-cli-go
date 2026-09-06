# Implementation Plan: Seed Files and Folders

**Branch**: `006-seed-files-folders` | **Date**: 2026-09-06 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/006-seed-files-folders/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

Add a `git.seeded.files`/`git.seeded.folders` section to `config.yaml`, applied on every `init`
run regardless of agent selection. Unlike managed files/folders (feature 005), which always
overwrite and mirror-prune, seeded files/folders are create-if-missing only: a seeded path is
written exactly once — the first time nothing already exists there — and is never touched again on
any later run, preserving any local customization. Implemented by extending `reposync.Sync` with
two new parameters (`seededFiles`, `seededFolders`) and one new destination-existence-aware
selection function, reusing the existing validation and write-loop code paths.

## Technical Context

**Language/Version**: Go 1.26.5 (per `go.mod`)

**Primary Dependencies**: `spf13/cobra`, `gopkg.in/yaml.v3`, `charmbracelet/huh`,
`golang.org/x/term`, `github.com/go-git/go-git/v6`, `github.com/go-git/go-billy/v6` — all
pre-existing; this feature adds no new dependency (stdlib `os.Lstat`/`path/filepath` suffice for
the destination-existence check).

**Storage**: Local filesystem (destination directory the CLI writes into) and an in-memory git
clone (existing `reposync` mechanism); no database.

**Testing**: `go test` with table-driven, colocated `_test.go` files (Principle III).

**Target Platform**: macOS, Linux, Windows (existing cross-platform Go CLI).

**Project Type**: Single-project Go CLI (existing `internal/` package layout).

**Performance Goals**: No new goal beyond existing Principle V baseline (interactive start well
under 1s); this feature adds only O(number of declared seeded paths) local `os.Lstat` calls to an
already-in-memory computation, which is negligible.

**Constraints**: Fully offline-capable aside from the existing git clone step (unchanged);
no new third-party dependency (Principle II / Technology & Dependency Constraints).

**Scale/Scope**: Bounded by the size of a single git repository's tracked files, same as the
existing `reposync.Sync` mechanism (features 002-005).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Code Quality**: PASS. New code follows existing `reposync`/`agentcatalog`/`cli` package
  conventions; `gofmt`/`go vet` are part of Phase 5 polish; doc comments added for new exported
  types/parameters; no unreviewed merge (handled by PR process, outside this plan's scope).
- **II. Simplicity**: PASS. No new package, no new dependency, no new abstraction/interface. The
  seeded write set is merged into the existing write loop and validated with the existing
  `checkDeclaredPathsExist`/`selectWriteSet`-shaped functions; only one genuinely new function
  (`selectSeedWriteSet`) is added, because "only write if the destination path doesn't already
  exist" has no equivalent in the existing code.
- **III. Testing (NON-NEGOTIABLE)**: PASS. Every new behavior (create-if-missing for files and
  folders, missing-path validation, type-conflict-as-skip, partial-failure no-rollback, CLI
  wiring regardless of agent) ships with table-driven tests colocated in the existing
  `reposync_test.go`/`catalog_test.go`/`init_test.go` files, mirroring feature 005's test
  structure.
- **IV. UX Consistency**: PASS. Reuses the existing error-to-stderr, actionable-message,
  re-run-instruction, and exit-code conventions verbatim — no new error UX pattern introduced.
- **V. Performance**: PASS. Adds only local filesystem `Lstat` calls proportional to the number of
  declared seeded paths; no new network calls; the existing single in-memory clone is unchanged.

No violations. Complexity Tracking table intentionally left empty.

## Project Structure

### Documentation (this feature)

```text
specs/006-seed-files-folders/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
internal/
├── agentcatalog/
│   ├── catalog.go        # Add SeededConfig, GitConfig.Seeded, validateSeededPaths
│   ├── catalog_test.go   # Tests for parsing/validating git.seeded.files/folders
│   └── config.yaml       # Production config (already has a `seeded:` section staged)
├── reposync/
│   ├── reposync.go       # Sync gains seededFiles/seededFolders params + selectSeedWriteSet
│   └── reposync_test.go  # Tests for create-if-missing file/folder seeding behavior
└── cli/
    ├── init.go           # runInit forwards catalog.Git.Seeded.Files/.Folders to syncRepo
    └── init_test.go      # Tests for seeded content applied regardless of selected agent
```

**Structure Decision**: Single-project Go CLI layout (existing). This feature only extends the
three existing packages above — no new package or directory is introduced, consistent with
Principle II.

## Complexity Tracking

> Not applicable — Constitution Check reported no violations.

