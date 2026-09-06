# Phase 0 Research: Seed Files and Folders

No `NEEDS CLARIFICATION` markers remain in the Technical Context — this feature reuses the
existing `reposync`/`agentcatalog`/`cli` mechanism established by features 002-005, so the
research below documents design decisions rather than resolving unknown technology choices.

## Decision: Config schema mirrors `managed`, under a new `seeded` key

**Decision**: Add `Seeded SeededConfig` to `GitConfig` (`internal/agentcatalog/catalog.go`),
where `SeededConfig` has `Files []string` and `Folders []string`, tagged `yaml:"seeded"` /
`yaml:"files"` / `yaml:"folders"` — structurally identical to feature 005's `ManagedConfig`. The
production `config.yaml` already stages this shape (`git.seeded.folders: [custom]`,
`git.seeded.files: [architecture.yaml]`), confirming the expected schema.

**Rationale**: Consistency with the existing `managed` key (Principle IV-adjacent: predictable
config shape) and Principle II (no new abstraction — same struct shape, different semantics
applied downstream).

**Alternatives considered**: A single `git.files`/`git.folders` list with a `mode: seed|manage`
per-entry flag — rejected as a bigger schema change for no added clarity, and it would complicate
the simple "which Go field did the user populate" validation this codebase already uses.

## Decision: Path validation reuses `isValidDeclaredPath` unchanged

**Decision**: A new `validateSeededPaths(seeded SeededConfig) error` function, structurally
identical to feature 005's `validateManagedPaths`, rejects empty/absolute/path-traversing entries
in `Seeded.Files`/`Seeded.Folders` using the existing `isValidDeclaredPath` helper. Called from
`LoadFS` alongside the existing `validateDeclaredPaths`/`validateManagedPaths` calls.

**Rationale**: There is only one sensible answer here (the same validation already applied to
every other declared-path category) — not a genuine design decision, so no clarification question
was warranted during `/speckit-clarify`. Reuses existing, already-tested code (Principle II).

**Alternatives considered**: None — skipping validation for seeded paths would be an inconsistent,
unjustified carve-out with no rationale.

## Decision: `reposync.Sync` gains `seededFiles, seededFolders []string`, validated like declared paths but written only if missing

**Decision**: Two new trailing parameters are added to `Sync` (after feature 005's
`managedFolders`): `seededFiles, seededFolders []string`.

- **Missing-path validation**: Unlike `managedFolders` (exempted in feature 005 because an empty
  managed folder is a valid mirror-to-nothing outcome), `seededFiles`/`seededFolders` behave like
  `declaredFiles`/`declaredFolders` for validation — `checkDeclaredPathsExist` is called with
  `declaredFiles ∪ seededFiles` and `declaredFolders ∪ seededFolders` (unioned only for this
  validation call). A seeded path matching zero retrieved files is a configuration error (FR-007),
  because seed content is never deleted or mirrored — there is no legitimate "matches nothing"
  outcome for it, unlike managed folders.
- **Write-set selection**: A new function, `selectSeedWriteSet(destination string, collected
  map[string][]byte, seededFiles, seededFolders []string) (map[string][]byte, error)`, mirrors
  `selectWriteSet`'s file/prefix matching but additionally calls `os.Lstat` on each candidate's
  destination path and includes it **only if** `os.IsNotExist(err)` is true for that `Lstat` call.
  If `Lstat` fails with any other error, it is wrapped and returned (fails the whole run, no
  partial seed writes attempted past that point — consistent with the existing validate-before-any-
  write ordering).
- **Merge into the existing write loop**: The seed write set is merged into the same `writeSet`
  map already computed for declared/managed content, before the single existing write loop runs.
  This reuses the existing `os.MkdirAll`/`os.WriteFile` loop unchanged (Principle II — no
  duplicated write logic) and means a type conflict at write time (e.g., permission error) is
  reported the same way as for declared/managed writes.
- **Type conflicts as silent skip, not error**: Because `selectSeedWriteSet` only includes a path
  when `Lstat` reports it does not exist, a directory occupying a seeded file's path (or vice
  versa) is already excluded from the write set by the existence check itself — no separate
  type-conflict error path is needed for seeded content, unlike managed folders (feature 005's
  FR-007, which applies only to always-overwrite content).
- **Pruning is unaffected**: `pruneStaleManagedFiles` continues to run only over `managedFolders`;
  seeded content is never pruned, matching FR-005/FR-004 of this feature (never delete).

**Rationale**: Reuses every existing function shape and the single write loop; the only genuinely
new logic is the destination-side existence check, which has no prior equivalent in this codebase.

**Alternatives considered**: A wholly separate `SeedFiles`/`SeedFolders` function duplicating
clone/write logic — rejected as unnecessary duplication (Principle II), same rationale feature
005 used for rejecting a separate `MirrorManaged` function.

## Decision: `os.Lstat`, not `os.Stat`, for the existence check

**Decision**: Use `os.Lstat` (not `os.Stat`) when checking whether something already exists at a
candidate seeded destination path.

**Rationale**: `Lstat` reports the existence of the path entry itself without following a
symlink, which is the more conservative and correct check for "is there already something at this
exact path" — a dangling symlink still counts as "already exists" and must not be overwritten,
matching the spirit of FR-005 (never touch what's already there).

**Alternatives considered**: `os.Stat` (follows symlinks) — rejected because a dangling symlink
would report `IsNotExist`, causing `Sync` to attempt to write through/replace it, contradicting
the "leave completely untouched" guarantee.

## Decision: CLI wiring keeps seeded paths as separate parameters, not unioned into agent/managed sets

**Decision**: In `internal/cli/init.go`'s `runInit`, `catalog.Git.Seeded.Files` and
`catalog.Git.Seeded.Folders` are passed to `syncRepo` as their own `seededFiles`/`seededFolders`
arguments — **not** unioned into the agent's `files`/`folders` the way feature 005 unions managed
files into the agent's own `files`.

**Rationale**: Managed files could be unioned into the agent's declared files because both use
identical always-overwrite semantics — the union was purely a code-reuse shortcut. Seeded files
have different (create-if-missing) semantics from the agent's own declared files, so they must
stay on their own parameter to reach the new `selectSeedWriteSet` path instead of the
always-overwrite `selectWriteSet` path.

**Alternatives considered**: Giving every agent's own declared files "seed" semantics too —
rejected; out of scope and contradicts existing, already-shipped feature 003/004 behavior where an
agent's declared files are always refreshed.
