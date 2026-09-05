# Contract: `init` Managed Files and Folders

Extends the `init` contract from
[../004-folder-merge-overwrite/contracts/cli-init.md](../004-folder-merge-overwrite/contracts/cli-init.md),
which itself extends
[../003-selective-file-retrieval/contracts/cli-init.md](../003-selective-file-retrieval/contracts/cli-init.md).
No CLI flags, arguments, exit codes, or output formats change. This contract documents the new
managed-files/managed-folders write set that applies in addition to, and independent of, the
selected agent's own declared files/folders.

## Managed write step contract

On every `init` run, in addition to the selected agent's declared `files`/`folders` write set:

1. Every path declared under `config.yaml`'s `git.managed.files` is created or overwritten with
   the retrieved repository's content at that exact path, following the same write-step contract
   as an agent's declared files (feature 003/004) — including the type-conflict and
   partial-failure guarantees below. A managed file no longer present in the repository is simply
   not written; any previously-written copy at the destination is left in place (no deletion
   applies to managed files).
2. Every path declared under `config.yaml`'s `git.managed.folders` is created or overwritten the
   same way, **and then**, once every write for the current `init` run has succeeded, any file
   already at the destination under that managed folder whose path the repository does not
   currently provide is removed. This applies recursively at every depth of the managed folder.
   This step supersedes feature 004's non-goal that declared-folder pruning was unimplemented —
   pruning now applies, but **only** to paths declared under `git.managed.folders`, never to an
   agent's own declared folders.
3. Managed files and folders apply regardless of which agent is selected during the current
   `init` run — they are not scoped to any single agent, and are applied on every run in addition
   to that run's selected agent's own declared files/folders.
4. If a managed-folder path the repository provides as a file already exists at the destination
   as a directory (or vice versa), `init` fails with an actionable error naming the conflicting
   path; the conflicting destination entry is left exactly as it was — this step never deletes it.
5. If any individual write or prune-time removal fails partway through, `init` fails and reports
   the error; files already written or removed earlier in the same run are **not** rolled back.
   Pruning for a managed folder only ever begins after every write for that `init` run has already
   succeeded — a write failure means pruning for that run does not happen at all.
6. Whenever `init` fails per step 4 or step 5, the error output MUST instruct the user to re-run
   the command.
7. If a declared managed **file** does not match anything in the retrieved repository, `init`
   fails with an actionable error naming the missing path, without writing or removing any files
   — consistent with the existing validation for an agent's own declared files (feature 003).
   This check does not apply to managed folders: a managed folder matching zero files in the
   retrieved repository is a valid outcome (it mirrors to a fully empty destination folder per
   step 2), not a validation error.

## Non-goals (explicitly out of contract)

- Pruning never applies to an agent's own declared folders — only to paths declared under
  `git.managed.folders`. Feature 004's merge/no-prune guarantee for agent-declared folders is
  unchanged.
- Pruning never removes now-empty directories left behind after their files are removed; only
  files are pruned.
- `init` does not detect or skip byte-identical content when writing managed files/folders; a
  matching path is always (re)written, same as feature 003/004.
