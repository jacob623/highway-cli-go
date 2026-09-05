# Implementation Plan: Agent Repository Clone

**Branch**: `002-agent-repo-clone` | **Date**: 2026-09-05 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/002-agent-repo-clone/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

After a developer completes the existing `init` agent-selection step, the CLI clones a single,
tool-wide configured git repository at a pinned commit SHA entirely in memory, then writes the
resulting files to a destination directory — an optional positional path argument to `init`, or
the current working directory by default — overwriting any existing files at colliding paths. The repository
URL and commit ref are configured once at the top level of the embedded catalog file, which is
renamed from `agents.yaml` to `config.yaml` to reflect that it now holds broader tool
configuration, not just the agent list. Which agent was selected has no bearing on which
repository/commit is retrieved at this stage (per-agent repository mapping is deferred to a
future spec). The configured repository is assumed to be publicly accessible; no authentication
is supported.

## Technical Context

**Language/Version**: Go 1.23

**Primary Dependencies**: `spf13/cobra` (command framework), `gopkg.in/yaml.v3` (YAML parsing for
the embedded catalog), `charmbracelet/huh` (interactive single-select prompt), `golang.org/x/term`
(TTY detection) — all carried over from the `init` feature — plus **NEW**: `github.com/go-git/go-git/v6`
(pure-Go git client, used to clone the configured repository into memory and check out the
configured commit) and `github.com/go-git/go-billy/v6` (in-memory filesystem backing that clone,
`memfs`), both required to satisfy the "clone to internal memory" requirement without shelling out
to a system `git` binary.

**Storage**: The embedded catalog file (renamed `internal/agentcatalog/config.yaml`, still bundled
via `go:embed`) gains a top-level `git: {repository, ref}` block alongside the existing `agents`
list. The git clone itself is held entirely in memory (`go-git` `memory.Storage` + `go-billy`
`memfs`) — no on-disk git working copy or `.git` metadata is ever created. The only on-disk output
is the plain files written to the destination directory.

**Testing**: `go test` with table-driven tests colocated as `_test.go` files per package; git
retrieval is tested against a local, temporary bare repository created with `go-git`'s `PlainInit`
and cloned via a local filesystem path, so tests never require real network access.

**Target Platform**: Cross-platform command-line binary (macOS, Linux, Windows), run interactively
in a terminal — unchanged from the `init` feature.

**Project Type**: Single project — CLI (extends the existing `highway-cli-go` project).

**Performance Goals**: The interactive portion (start-up through completing agent selection) must
remain well under 1 second, unchanged from Constitution Principle V. The subsequent clone/write
phase is expected to scale with the configured repository's size and network conditions; this is
an explicit, documented exception under Principle V ("network calls MUST NOT be on the critical
path ... unless that command's specified purpose requires network access") because retrieving the
configured repository is this feature's entire purpose.

**Constraints**: The configured repository is assumed to be publicly accessible (FR-010); no
credential handling is implemented. Retrieved files overwrite any destination file at a colliding
path; non-colliding existing files are left untouched (FR-009). No `.git` metadata is ever written
to the destination. The clone is performed in memory, not to a temporary directory on disk.

**Scale/Scope**: A single configured repository/commit, expected to be modestly sized (a small
skills/config repository), fetched as a full (non-shallow) clone so the exact configured commit
SHA is guaranteed to be resolvable locally before checkout.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Evaluated against `.specify/memory/constitution.md` v1.1.0:

| Principle / Constraint | Status | Notes |
|---|---|---|
| I. Code Quality | PASS | New package(s) get doc comments on exported identifiers; clone/write errors are returned and surfaced explicitly (FR-005, FR-006), never swallowed. |
| II. Simplicity | PASS | One new package (`internal/reposync`) with a single responsibility (fetch a configured repo into memory, write its files to disk); no speculative abstractions beyond the seam needed for offline testing (a small fetch/write interface, mirroring the existing overridable-function pattern already used in `internal/cli`). Two new dependencies are justified individually in Complexity Tracking. |
| III. Testing Standards (NON-NEGOTIABLE) | PASS | Fetch/write logic is tested against a local temporary bare repository (no real network calls), covering success, invalid URL, invalid ref, and destination-overwrite paths. |
| IV. User Experience Consistency | PASS | New optional positional path argument on `init` follows cobra's standard `Args` validation conventions; errors go to stderr with actionable messages; exit codes stay `0`/non-zero; no hang introduced (fetch failures are surfaced, not retried indefinitely). |
| V. Performance Requirements | PASS (documented exception) | Interactive portion remains sub-1-second; the clone/write phase is explicitly network-bound because that is this feature's specified purpose, per Principle V's own exception clause. |
| Technology & Dependency Constraints | PASS (with justification) | `go-git/go-git` (latest commit 2 days old at time of writing) and `go-git/go-billy` (latest commit 3 weeks old) both satisfy the "commit no more than 1 month old" bar; justified individually in Complexity Tracking. |
| Development Workflow & Quality Gates | PASS | This Constitution Check satisfies the required gate. Renaming the embedded catalog file (`agents.yaml` → `config.yaml`) is an internal, compiled-in resource, not a developer-facing CLI flag, output format, or persisted file, so it does not require a Development Workflow migration note. |

No unjustified violations. Proceed to Phase 0.

**Post-Phase 1 re-check**: Phase 0 (research.md) and Phase 1 (data-model.md, contracts/,
quickstart.md) did not introduce anything beyond the Technical Context and dependency set already
evaluated above. All rows above remain PASS after design.

## Project Structure

### Documentation (this feature)

```text
specs/002-agent-repo-clone/
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
    └── main.go                  # unchanged entrypoint

internal/
├── agentcatalog/
│   ├── config.yaml              # RENAMED from agents.yaml; adds top-level `git:` block
│   ├── catalog.go                # extended: GitConfig struct + AgentCatalog.Git field + validation
│   └── catalog_test.go
├── reposync/
│   ├── reposync.go               # NEW: in-memory clone (go-git + go-billy) + write-to-destination
│   └── reposync_test.go
└── cli/
    ├── root.go                   # unchanged
    ├── init.go                   # extended: optional positional path arg, wires reposync after agent confirmation
    └── init_test.go
```

**Structure Decision**: Extends the existing single Go project. Git retrieval and file-writing
logic is isolated into its own `internal/reposync` package (single responsibility: fetch + write),
kept separate from `internal/agentcatalog` (catalog parsing/validation) and `internal/cli`
(command wiring), consistent with Constitution Principle II.

## Complexity Tracking

> Required by the Technology & Dependency Constraints section: every new third-party dependency
> must be justified here.

| Dependency | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| `github.com/go-git/go-git/v6` | Clones the configured repository and checks out the configured commit SHA entirely in memory (FR-002); actively maintained (latest commit 2 days old at time of writing), satisfying the Technology & Dependency Constraints requirement | Shelling out to the system `git` binary (rejected — requires `git` to be installed on the developer's machine, which the constitution's cross-platform build/run requirement does not guarantee, and is harder to keep the clone confined to memory only) |
| `github.com/go-git/go-billy/v6` | Provides the in-memory filesystem (`memfs`) that `go-git` checks the repository out into, so no on-disk temporary directory or `.git` metadata is ever created; actively maintained (latest commit 3 weeks old) | Writing the clone to a temporary directory on disk and cleaning it up afterward (rejected — leaves a window where `.git` metadata and a full working copy exist on disk, contradicting the "clone to internal memory" requirement, and adds cleanup-on-error complexity) |
