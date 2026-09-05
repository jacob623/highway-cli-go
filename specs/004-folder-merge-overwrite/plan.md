# Implementation Plan: Merge Declared Folders Into Destination

**Branch**: `004-folder-merge-overwrite` | **Date**: 2026-09-05 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/004-folder-merge-overwrite/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

`init` must merge a selected agent's declared folders into the destination rather than treat the
destination as disposable: only the paths the retrieved repository actually provides under a
declared folder are created or overwritten; any other pre-existing destination content under that
same folder is left untouched, recursively at every nesting depth. Research (Phase 0) confirms
`internal/reposync.Sync`'s existing write-set-only-write loop (from feature 003) and the Go
standard library's file/directory type-conflict errors already satisfy every functional
requirement in this spec, including the two behaviors clarified in this session (best-effort
partial-write on failure, fail-fast actionable error on file/directory type conflicts, no
deletion in either case). No production code changes are required; the deliverable is a set of
new regression tests in `internal/reposync/reposync_test.go` that pin this behavior so it cannot
silently regress.

## Technical Context

**Language/Version**: Go 1.26.5 (unchanged)

**Primary Dependencies**: None new — uses only `os`/`path/filepath` from the standard library,
already imported by `internal/reposync`

**Storage**: Local filesystem (destination directory); N/A otherwise

**Testing**: `go test ./...`; table-driven tests colocated in `internal/reposync/reposync_test.go`

**Target Platform**: CLI on macOS, Linux, and Windows (unchanged)

**Project Type**: Single Go CLI project — extends `internal/reposync` test coverage only

**Performance Goals**: No change; merge write path is the same file-count-bound loop as before

**Constraints**: MUST NOT regress feature 002/003 behavior (declared path validation, write-set
filtering, overwrite-on-collision, opt-in-only empty-list semantics)

**Scale/Scope**: Same as existing — bounded by the size of the configured skill catalog repository

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Code Quality**: PASS — no production code changes; new tests are gofmt/vet-clean and
  colocated per existing convention.
- **II. Simplicity**: PASS — no new abstractions, dependencies, or configuration; this plan
  explicitly avoids adding speculative rollback/type-conflict-handling machinery the research
  showed is unnecessary (YAGNI).
- **III. Testing Standards (NON-NEGOTIABLE)**: PASS — this feature's entire deliverable is new
  success-path (merge preserves untouched files, recursively) and failure-path (type conflict
  fails fast, no deletion) tests, satisfying the "ship with tests" requirement even though no
  behavior is changing.
- **IV. UX Consistency**: PASS — errors continue to flow through the existing wrapped-error →
  stderr path in `runInit`; exit codes unchanged.
- **V. Performance**: PASS — no new dependencies, no network calls added, no change to the
  offline-by-default clone-then-write flow.

No Complexity Tracking entries are needed — no violations to justify.

## Project Structure

### Documentation (this feature)

```text
specs/004-folder-merge-overwrite/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   └── cli-init.md
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
# Option 1: Single project (existing structure from features 001/002/003 — unchanged)
internal/
├── agentcatalog/   # unchanged by this feature
├── reposync/       # only reposync_test.go gains new regression tests
│   ├── reposync.go
│   └── reposync_test.go
└── cli/            # unchanged by this feature
    ├── init.go
    └── init_test.go

cmd/highway/        # unchanged by this feature
```

**Structure Decision**: Single Go CLI project, unchanged from features 001–003. This feature adds
no new files, packages, or dependencies — only new test cases in the existing
`internal/reposync/reposync_test.go`.

## Complexity Tracking

> No entries — Constitution Check reported no violations.

