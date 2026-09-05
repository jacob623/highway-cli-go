# Implementation Plan: Selective File Retrieval

**Branch**: `003-selective-file-retrieval` | **Date**: 2026-09-05 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/003-selective-file-retrieval/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

Each agent entry in the embedded catalog (`internal/agentcatalog/config.yaml`) gains two optional,
per-agent lists — declared `files` and declared `folders`, both relative to the configured git
repository's root. When the developer selects an agent, `init` writes only that agent's declared
files plus every file found recursively under its declared folders, instead of writing the entire
retrieved repository. If an agent declares neither list (or both are empty), `init` writes **no**
files for that agent — the declared lists are opt-in, not a fallback to "everything." If any
declared path is absent from the retrieved repository at the configured commit, the whole
retrieval fails before any file is written, with a clear error naming the missing path(s).

## Technical Context

**Language/Version**: Go 1.26.5 (per `go.mod`)

**Primary Dependencies**: `github.com/go-git/go-git/v6`, `github.com/go-git/go-billy/v6`
(existing, feature 002), `gopkg.in/yaml.v3`, `spf13/cobra`, `charmbracelet/huh` (existing, feature
001/002) — no new third-party dependency required for this feature.

**Storage**: N/A — embedded `config.yaml` (compiled into the binary via `go:embed`) plus the local
filesystem destination `init` writes to.

**Testing**: `go test ./...`, table-driven, colocated `_test.go` files, following the existing
patterns in `internal/agentcatalog/catalog_test.go`, `internal/reposync/reposync_test.go`, and
`internal/cli/init_test.go`.

**Target Platform**: CLI on macOS, Linux, and Windows (per constitution).

**Project Type**: Single Go CLI project (existing structure; no new top-level project).

**Performance Goals**: No new performance goal beyond existing feature 002 requirements — path
filtering is an in-memory set/prefix comparison over an already-cloned, already-in-memory file
tree, negligible relative to the clone itself.

**Constraints**: No network calls beyond the existing single repository clone; no telemetry;
destination writes remain limited to the files actually selected for the chosen agent.

**Scale/Scope**: Declared lists are maintainer-authored and expected to be small (tens of entries
per agent at most); no pagination, streaming, or indexing concerns.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Code Quality**: New/changed code will be `gofmt`-formatted, pass `go vet`/lint, handle
  errors explicitly (missing declared path is a returned, wrapped error — never silently
  dropped), and exported additions (`AgentDefinition.Files`/`.Folders`, any new `reposync` errors
  or parameters) will get doc comments. PASS.
- **II. Simplicity**: Reuses the existing `agentcatalog`/`reposync`/`cli` package boundaries — no
  new package is introduced. Declared-path validation (path-traversal rejection) is added to
  `agentcatalog`'s existing load-time validation (same place `ErrGitConfigMissing` is checked
  today); filtering is added to `reposync.Sync`'s existing collect-then-write flow. No new
  abstraction, interface, or configuration surface beyond the two YAML lists the feature requires.
  PASS.
- **III. Testing Standards**: New behavior (per-agent filtering, no-declared-lists writes nothing,
  missing-path failure, path-traversal rejection, folder recursion, files/folders overlap dedupe)
  will ship with table-driven tests in the same three `_test.go` files, covering success and
  failure paths, before merge. PASS.
- **IV. User Experience Consistency**: Missing declared path errors go to stderr, are actionable
  (name the missing path), and preserve the existing non-zero exit code convention; no new flags
  or output format are introduced. PASS.
- **V. Performance Requirements**: No new network calls; filtering cost scales with the size of
  the declared lists and the already-retrieved file tree, not fixed overhead. PASS.

No violations — Complexity Tracking table is not needed.

## Project Structure

### Documentation (this feature)

```text
specs/003-selective-file-retrieval/
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
│   ├── config.yaml          # adds per-agent `files`/`folders` lists
│   ├── catalog.go            # AgentDefinition gains Files/Folders; new path-traversal validation
│   └── catalog_test.go       # new cases for Files/Folders parsing and validation
├── reposync/
│   ├── reposync.go           # Sync gains a files/folders selection parameter and filtering step
│   └── reposync_test.go      # new cases for filtering, missing-path failure, folder recursion
└── cli/
    ├── init.go               # passes the selected agent's Files/Folders into syncRepo
    └── init_test.go          # updated stubs/tests for the new syncRepo signature

tests/                        # none — this repo colocates tests as `_test.go` next to source
```

**Structure Decision**: Existing single-project Go CLI layout is unchanged. This feature only
extends the three packages already touched by feature 002 (`agentcatalog`, `reposync`, `cli`); no
new package, directory, or project is introduced, consistent with Principle II (Simplicity).

## Complexity Tracking

> No entries — Constitution Check reported no violations.

