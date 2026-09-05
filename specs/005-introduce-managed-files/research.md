# Phase 0 Research: Managed Files and Folders

No `[NEEDS CLARIFICATION]` markers remain in the Technical Context (all three clarification
questions were resolved directly during `/speckit-specify` and `/speckit-clarify`; see
[spec.md](./spec.md)'s Clarifications section). This document instead records the design
decisions needed to turn the resolved requirements into an implementation, following the
existing patterns established by features 003 and 004.

## Decision: Reuse the existing single clone; do not fetch managed content separately

**Decision**: Managed files/folders are retrieved from the exact same `git.CloneContext` call
`reposync.Sync` already performs for the selected agent's declared files/folders — no second
clone, no second network call.

**Rationale**: The repository and ref are tool-wide (`GitConfig.Repository`/`Ref`), identical for
every agent and for managed content. Cloning twice would double network latency and violate
Constitution Principle V (network calls must not be redundant/on a needless critical path).

**Alternatives considered**: A separate `SyncManaged` function with its own clone — rejected as
wasteful and redundant; would also require re-deriving the same `collectFiles` output twice.

## Decision: Managed files are unioned into the existing `declaredFiles` write-set input

**Decision**: `internal/cli/init.go` combines the selected agent's declared files with
`catalog.Git.Managed.Files` into a single slice before calling `reposync.Sync` — the same slice
Sync already treats as "create-or-overwrite, never delete." No change to `Sync`'s per-file
handling is needed.

**Rationale**: FR-004 requires managed files to behave exactly like an agent's declared files
(overwrite-only, no deletion on removal from the repo). Since the two categories have identical
semantics at the `reposync.Sync` layer, there is nothing to distinguish once the caller has
computed the combined set — introducing a separate parameter or type for "managed files" would
be a distinction without a behavioral difference, which Principle II (YAGNI) rules out.

**Alternatives considered**: Adding a `managedFiles []string` parameter to `Sync` that is
handled identically to `declaredFiles` internally — rejected because it would add API surface
with zero behavioral divergence from the existing parameter.

## Decision: Managed folders are unioned into the write-set/validation computation, plus one new pruning pass

**Decision**: `reposync.Sync` gains exactly one new parameter, `managedFolders []string`. It is:

1. Unioned with `declaredFolders` for the existing `selectWriteSet` call only (so managed-folder
   files are written/overwritten using the exact same code path, including the same OS-level
   `os.MkdirAll`/`os.WriteFile` type-conflict errors that already satisfy FR-007 — see below).
   `managedFolders` is deliberately **not** unioned into the `checkDeclaredPathsExist` call — see
   the missing-path decision below.
2. Used, on its own, as the input to one new function, `pruneStaleManagedFiles`, run only after
   the write loop completes without error.

**Rationale**: This is the minimal change that satisfies FR-003 (managed folders mirror
exactly) without duplicating the write-set selection or validation logic that already exists and
is already tested for features 003/004.

**Alternatives considered**: A wholly separate `MirrorManaged` function duplicating clone/write
logic — rejected as unnecessary duplication of already-correct, already-tested code (violates
Principle II).

## Decision: Managed folders are exempt from the missing-declared-path validation (FR-008 revision)

**Decision**: `checkDeclaredPathsExist` continues to validate `declaredFiles`, `declaredFolders`,
and (via the caller's union at the CLI layer) managed files — but is called with `declaredFolders`
alone, never unioned with `managedFolders`. A managed folder that currently matches zero files in
the retrieved repository is **not** an error.

**Rationale**: This corrects a contradiction discovered during implementation: FR-008, as
originally drafted, required an error whenever a declared managed folder matched nothing in the
repository — but Acceptance Scenario 4 (and the original feature request's "replaced in its
entirety" wording) requires that exact condition — a managed folder the repository maintainer has
fully emptied out — to succeed and prune every previously-written file. Since git trees do not
record empty directories, "this folder currently has zero files" and "this folder path never
existed" are indistinguishable from the collected file map alone, so both FR-008-as-originally-
drafted and Acceptance Scenario 4 could not simultaneously hold for a managed folder. Resolved in
favor of the explicit acceptance scenario; see spec.md's Clarifications section for the recorded
decision. Managed **files** are unaffected — a missing managed file is still a validation error,
since a single exact file path either exists in the repository or it doesn't, with no equivalent
"legitimately mirrors to nothing" case.

**Alternatives considered**: Requiring at least one file to remain under a managed folder at all
times — rejected as directly contradicting the feature's core "replace in its entirety" premise,
which explicitly includes replacement with nothing.

## Decision: Type-conflict handling (FR-007) requires no new code

**Decision**: Because managed folders are unioned into the same `selectWriteSet`/write-loop used
for declared folders, a file/directory type conflict at a path the repository wants to write
(e.g., destination has a file where the repo wants a directory) already produces the exact same
`os.MkdirAll`/`os.WriteFile` error feature 004 relies on for its own FR-007 — and, per that same
existing behavior, the conflicting entry is never deleted (the write loop only ever calls
`os.WriteFile` on paths the repository provides; it never removes anything).

**Rationale**: This mirrors feature 004's research finding that FR-007-equivalent behavior falls
out of the existing write loop for free. No new type-conflict-detection code is needed.

**Alternatives considered**: A dedicated pre-flight type-conflict scan before writing — rejected
as redundant; the OS already returns a clear, actionable error at the exact failing path.

## Decision: Prune stale managed-folder files only after all writes succeed ("write-then-prune")

**Decision**: `pruneStaleManagedFiles` runs strictly after the write loop returns successfully.
If the write loop fails partway through (FR-006), `Sync` returns immediately and no pruning is
attempted at all that run.

**Rationale**: This ordering minimizes worst-case damage from a partial failure. If pruning ran
first (delete-then-write), a mid-run failure could leave the destination missing content the
repository still wants there, with the replacement not yet written — an actively worse state than
before the run. Write-then-prune guarantees that, in the worst case, a partial failure only
leaves some now-stale files un-pruned (harmless extra content), never missing wanted content.
This is consistent with the resolved best-effort/no-rollback answer (FR-006): the run simply
stops, and whatever was already accomplished (written and/or pruned) stands.

**Alternatives considered**: Delete-then-write — rejected per the safety argument above.
Interleaving prune-per-file with write-per-file — rejected as needless complexity for no
additional guarantee, since both passes are already best-effort/no-rollback.

## Decision: Pruning walks only the declared managed-folder subtrees, comparing against `collected`

**Decision**: For each folder in `managedFolders`, walk the destination's existing directory
tree rooted at `filepath.Join(destination, folder)` using `filepath.WalkDir` (stdlib). For every
regular file found, compute its path relative to `destination` (matching the exact key format
`collected`/`writeSet` already use) and delete it via `os.Remove` if that key is not present in
`collected` (i.e., the repository's current retrieved content). Directories themselves are never
removed, even if they become empty — FR-003 is scoped to files ("every file... MUST be removed"),
and leaving empty directories behind satisfies that requirement while keeping the implementation
simple (Principle II).

**Rationale**: This directly satisfies FR-003 and Acceptance Scenario 4 (entire repository-side
removal of a managed folder's contents results in full destination removal) with an isolated,
easily testable helper. Scoping the walk to only the declared managed folders (not the whole
destination) keeps pruning cost proportional to managed content, not total destination size —
satisfying the Performance principle.

**Alternatives considered**: Removing now-empty directories after pruning their contents —
rejected as an unrequested behavior (YAGNI); the spec's success criteria are defined in terms of
files, not directory structure.

## Decision: Managed path validation reuses `isValidDeclaredPath` and the existing error type

**Decision**: `agentcatalog.LoadFS` validates `catalog.Git.Managed.Files`/`Folders` using the
same `isValidDeclaredPath` check already used for agents' declared paths, returning the existing
`ErrInvalidDeclaredPath` wrapped with a managed-specific message (e.g. `managed file %q` /
`managed folder %q` instead of `agent %q declares file %q`).

**Rationale**: The validation rule (non-empty, relative, no `..` segments) is identical
regardless of whether a path is agent-declared or managed; reusing the existing helper and error
sentinel avoids introducing a parallel validation system for no behavioral gain.

**Alternatives considered**: A distinct `ErrInvalidManagedPath` sentinel — rejected; callers
that already handle `ErrInvalidDeclaredPath` (if any exist outside this package) continue to
work unchanged, and the message text alone is sufficient to distinguish the source.

## Decision: `GitConfig.Managed` is optional and defaults to empty (backward compatible)

**Decision**: `ManagedConfig` is a plain struct with `Files []string` / `Folders []string`,
embedded as `Managed ManagedConfig` on `GitConfig`. When `git.managed` is absent from
`config.yaml`, YAML unmarshalling leaves both slices `nil`, and every code path that consumes
them (write-set union, validation, pruning) already treats `nil`/empty slices as "nothing to
do" — matching existing behavior for empty `declaredFiles`/`declaredFolders`.

**Rationale**: Satisfies the Assumption in spec.md that absent/empty managed configuration is
backward compatible with features 001-004, with zero special-casing required.

**Alternatives considered**: A pointer `*ManagedConfig` to distinguish "absent" from "present but
empty" — rejected; the spec draws no behavioral distinction between the two, so the added
complexity is unjustified (Principle II).
