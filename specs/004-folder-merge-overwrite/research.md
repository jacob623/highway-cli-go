# Phase 0 Research: Merge Declared Folders Into Destination

No `[NEEDS CLARIFICATION]` markers remain in the Technical Context (see [plan.md](./plan.md)), so
this document records the design decisions already settled — through the clarification session
and through direct verification of current behavior — rather than open research questions.

## Decision: No changes are needed to `reposync.Sync`'s write-set selection or write loop

**Rationale**: Feature 003 already changed `Sync` so that only paths matching a selected agent's
declared `files`/`folders` are written (`selectWriteSet`), and the write loop
(`for relPath, data := range writeSet { ... os.WriteFile(destPath, data, ...) }`) only ever
creates or overwrites paths present in that set — it never enumerates or deletes existing
destination content. Because the write set is already scoped to exactly "what the repository
provides for the declared folder," any destination file under that same folder that the
repository doesn't currently provide is never referenced by the loop and is therefore left
alone. This is precisely FR-001–FR-003 and FR-005; no new filtering logic is required.

**Alternatives considered**: Explicitly enumerating the destination directory and skipping
non-matching paths was considered and rejected — it would add code that duplicates behavior the
write-set-driven loop already provides for free, violating Constitution Principle II
(Simplicity/YAGNI).

## Decision: No changes are needed for partial-failure behavior (FR-006)

**Rationale**: The write loop iterates the write set and returns immediately on the first error
from `os.MkdirAll`/`os.WriteFile`. Files already written in earlier loop iterations are not
reverted. This already matches the clarified "best-effort, no rollback" behavior exactly.

**Alternatives considered**: Buffering all file contents and writing them only after every write
is validated (a two-phase commit style) was considered and rejected per the clarification answer
— it would add meaningful complexity (temp-file staging, atomic rename orchestration) for a
guarantee the spec explicitly does not require.

## Decision: No changes are needed for file/directory type-conflict handling (FR-007)

**Rationale**: Verified empirically (see below) that both conflict directions already produce a
clear, path-naming error through the existing wrapped-error path, and neither causes any
deletion:

- Retrieved **file** where the destination already has a **directory** at that exact path:
  `os.WriteFile` returns `open <path>: is a directory`, wrapped by `Sync` as
  `write <path>: open <path>: is a directory` — no deletion occurs, and the path is named twice
  over (by the OS error and by `Sync`'s wrap).
- Retrieved file nested under a destination path where an **ancestor** is already a **file** (not
  a directory): `os.MkdirAll` returns `mkdir <path>: not a directory`, wrapped by `Sync` as
  `create directory for <path>: mkdir <path>: not a directory` — no deletion occurs.

Verified with a throwaway Go program exercising both `os.WriteFile` and `os.MkdirAll` against
pre-created conflicting paths; both returned errors without creating, deleting, or modifying the
conflicting entry.

**Alternatives considered**: Adding an explicit `os.Stat`-based pre-check before writing (to
produce a custom, feature-specific error message) was considered and rejected — the existing
OS-level errors, once wrapped by `Sync`'s existing `fmt.Errorf(..., "%w", err)` calls, already
name the conflicting path and clearly state the nature of the conflict ("is a directory" / "not a
directory"), satisfying FR-007's "actionable error naming the conflicting path" requirement
without new code.

## Conclusion

This feature's implementation work is exclusively new regression tests in
`internal/reposync/reposync_test.go`:

1. A test pinning that pre-existing destination content under a declared folder, absent from the
   repository's current version of that folder, survives a `Sync` call unchanged (User Story 1).
2. A test pinning the same guarantee recursively through nested subfolders (User Story 2).
3. Tests pinning both type-conflict directions (FR-007): error returned, error names the
   conflicting path, and the conflicting destination entry is neither deleted nor modified.
