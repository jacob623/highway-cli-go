# Contract: `init` Seed Files and Folders

Extends the `init` contract from
[../005-introduce-managed-files/contracts/cli-init.md](../005-introduce-managed-files/contracts/cli-init.md),
which itself extends features 003/004's `init` contracts. No CLI flags, arguments, exit codes, or
output formats change. This contract documents the new seeded-files/seeded-folders write set,
which applies in addition to, and independent of, both the selected agent's own declared
files/folders and the managed files/folders write set.

## Seed write step contract

On every `init` run, in addition to the selected agent's declared write set and the managed write
set (feature 005):

1. Every path declared under `config.yaml`'s `git.seeded.files` is created with the retrieved
   repository's content at that exact path **only if** nothing already exists there. If a file or
   directory already occupies that exact path, it is left completely untouched — no overwrite, no
   error.
2. Every path declared under `config.yaml`'s `git.seeded.folders` is seeded the same way,
   recursively: each file the repository provides under that folder is created at its
   corresponding destination path only if nothing already exists there. Existing files under the
   folder — whether previously seeded and then customized, or unrelated — are never overwritten
   or removed.
3. Seeded files and folders apply regardless of which agent is selected during the current `init`
   run — they are not scoped to any single agent, and are applied on every run in addition to
   that run's agent-declared and managed write sets.
4. A directory already occupying a seeded file's destination path (or a file already occupying
   where a seeded folder's file would go) is treated as "already exists" per steps 1-2 — this is
   not reported as an error, and the existing entry is left exactly as it was.
5. If any individual seed write fails partway through, `init` fails and reports the error; seed
   files already written earlier in the same run are **not** rolled back — a partial mix of
   seeded and not-yet-seeded content is acceptable, consistent with the existing best-effort
   precedent for managed folders (feature 005).
6. Whenever `init` fails per step 5, or per the missing-path validation in step 7, the error
   output MUST instruct the user to re-run the command.
7. If a declared seeded file or seeded folder does not match anything in the retrieved
   repository, `init` fails with an actionable error naming the missing path, without writing any
   seed content — unlike managed folders (feature 005), a seeded folder matching zero files is
   always treated as a configuration error, since seed content has no legitimate "matches nothing"
   success outcome (it is never deleted or mirrored).

## Non-goals (explicitly out of contract)

- Seed content is never overwritten or removed once it exists at the destination, under any
  circumstance — there is no "force reseed" mechanism in this feature.
- `init` does not compare an existing destination file's content against the repository's version
  to decide whether to write; the check is existence-only (a file present with different, equal,
  or empty content is equally left untouched).
- Declaring the same path as both seeded and managed, or both seeded and an agent's own declared
  file/folder, has no defined interaction in this contract — such overlapping declarations are a
  configuration choice outside this feature's scope.
