# Phase 0 Research: Selective File Retrieval

No open `[NEEDS CLARIFICATION]` markers remain in [spec.md](./spec.md) — both were resolved
directly with the user before planning (see FR-004 and FR-008/FR-001a). This document records the
resulting design decisions and the alternatives considered.

## Decision: Declared lists live on each agent entry, not on the tool-wide `git` block

**Rationale**: The user confirmed the declared files/folders scope which paths are written *based
on the agent selected*, not a single tool-wide subset. `config.yaml`'s `git:` block (repository +
ref) stays a single, tool-wide setting; `files`/`folders` are added to each entry under `agents:`.

**Alternatives considered**:
- A single tool-wide `files`/`folders` list under `git:` — rejected: contradicts the user's
  explicit requirement that the subset depends on which agent is selected.
- A separate top-level map keyed by agent id (e.g. `file_selections: {vscode: {...}}`) — rejected:
  splits one agent's configuration across two unrelated YAML locations for no benefit; nesting
  under the existing per-agent entry is simpler (Principle II).

## Decision: Two separate lists (`files`, `folders`) rather than one mixed list

**Rationale**: The user explicitly asked for two separate entries. This also removes any need to
infer from a bare path string whether it names a file or a directory (which would otherwise
require a filesystem lookup before validation could even start).

**Alternatives considered**:
- One list with directory-detection by trailing slash or repository lookup — rejected per user's
  explicit instruction, and it would have required resolving against the retrieved tree before
  validating configuration shape, adding complexity for no requirement.

## Decision: Missing declared path fails the entire retrieval (fail-fast)

**Rationale**: The user chose strict fail-fast behavior: any declared file or folder absent from
the retrieved repository at the configured commit aborts the whole `init` run with a clear error,
writing nothing. This matches the existing fail-fast precedent from feature 002 (clone/checkout
failure also writes nothing).

**Alternatives considered**:
- Write what exists, warn about the rest, exit 0 — rejected by user (risks silently-incomplete
  output being treated as success).
- Write what exists, warn, but exit non-zero — rejected by user in favor of writing nothing.

## Decision: Folder entries are matched by path-prefix against the already-collected file tree

**Rationale**: `reposync.Sync` already walks the entire checked-out tree into an in-memory
`map[string]([]byte)` keyed by repo-relative path before writing anything to disk (feature 002).
Filtering can reuse that same map: a file is included if its path exactly matches a declared file
entry, or if its path has a declared folder entry as a path-prefix (folder-then-separator). A
declared folder with zero matching entries in the tree is treated as "missing" for FR-004 purposes
— this is a deliberate simplification (an empty-but-present directory in the git tree is
indistinguishable from an absent one via this method, and git trees generally don't track empty
directories at all, making the distinction moot in practice).

**Alternatives considered**:
- Separately stat/list the directory in the checked-out worktree filesystem to distinguish
  "exists but empty" from "does not exist" — rejected: adds a second traversal mechanism for a
  distinction git itself does not meaningfully preserve (git does not track empty directories).

## Decision: Path-traversal validation happens in `agentcatalog` at load time

**Rationale**: `agentcatalog.LoadFS` already performs FR-driven structural validation (non-empty
agents, non-empty git config) and returns sentinel errors for each. Rejecting a declared path
containing `..` segments (FR-006) is the same kind of "invalid configuration" check, and doing it
at load time means `reposync` can trust every path it receives, keeping that package focused on
clone/checkout/write (Principle II: single responsibility per package).

**Alternatives considered**:
- Validate inside `reposync.Sync` right before filtering — rejected: would duplicate the
  "configuration validation" responsibility `agentcatalog` already owns, and would only be caught
  after paying the cost of a full clone.

## Decision: `reposync.Sync` gains two new parameters (`files`, `folders []string`)

**Rationale**: Keeps `Sync`'s existing responsibility (clone, checkout, collect, write) and simply
threads the caller-selected agent's declared lists through as additional inputs, mirroring how
`repository`/`ref`/`destination` are already passed as plain strings. When both slices are empty,
`Sync` writes no files at all (FR-003/User Story 2) — the declared lists are opt-in, so an agent
that declares neither list receives nothing, rather than falling back to the pre-003 "write
everything" behavior. The clone and checkout still happen (so a missing/invalid repository or ref
is still reported), but the write step is a no-op.

**Alternatives considered**:
- A `Selection` struct parameter — considered for readability with more fields, but two `[]string`
  parameters is simpler for exactly two related inputs and avoids introducing a new exported type
  for no behavior beyond grouping (Principle II).
- Falling back to "write everything" when both lists are empty (the feature's original working
  assumption) — rejected: the user explicitly requires an agent with no declared lists to receive
  no files, making the declared lists a strict opt-in mechanism rather than an optional narrowing
  of a default "everything" behavior.
