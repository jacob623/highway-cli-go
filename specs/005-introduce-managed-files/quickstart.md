# Quickstart: Managed Files and Folders

Validates `init`'s new managed-files/managed-folders behavior. See
[data-model.md](./data-model.md) for the behavior summary and
[contracts/cli-init.md](./contracts/cli-init.md) for the full contract. This builds on the
`004-folder-merge-overwrite` quickstart at
[`../004-folder-merge-overwrite/quickstart.md`](../004-folder-merge-overwrite/quickstart.md).

## Prerequisites

- Go 1.26+ installed.
- A terminal (TTY) session — `init` requires an interactive prompt.
- Network access to the configured git repository.
- `config.yaml`'s `git.managed` section declares at least one file and one folder (the production
  catalog already declares `files: [file1.md]` and `folders: [.highway]`).
- A destination directory pre-populated with a file under the declared managed folder that the
  configured repository does **not** currently provide.

## Build

```sh
cd highway-cli-go
go build -o bin/highway ./cmd/highway
```

## Scenario 1 — Managed folder is mirrored exactly, including removal of stale files

Setup:

- Destination already contains `.highway/stale.md` (a file the repository's current `.highway`
  folder does not provide) and `.highway/keep.md` (which the repository's current `.highway`
  folder does provide, with different content).

**Run**: `init` against that destination, selecting any agent.

**Expected**: `.highway/stale.md` is removed; `.highway/keep.md` is overwritten with the
repository's current content.

## Scenario 2 — Managed content applies regardless of selected agent

Setup: two separate destination directories, no managed content written yet in either.

**Run**: `init` once selecting Agent A, once selecting Agent B (against the two separate
destinations).

**Expected**: both destinations receive an identical copy of every declared managed file and
folder, in addition to each destination receiving that run's selected agent's own declared
files/folders.

## Scenario 3 — Managed file behaves like an agent's declared file (overwrite-only, no deletion)

Setup: destination already contains `file1.md` (managed file), and the repository's current
version no longer provides `file1.md` at all.

**Run**: `init`, selecting any agent.

**Expected**: `file1.md` at the destination is left exactly as it was (not deleted) — managed
files are never removed, only created or overwritten when the repository provides them.

## Scenario 4 — Type conflict inside a managed folder fails fast without deleting anything

Setup: destination has a directory at `.highway/conflict/`, and the repository's current
`.highway` folder provides a plain file at `.highway/conflict`.

**Run**: `init`, selecting any agent.

**Expected**: `init` exits non-zero, prints an actionable error naming `.highway/conflict`,
followed by "Re-run the command to retry."; the destination directory at that path is not
deleted.

## Scenario 5 — Missing declared managed path fails validation before any write

Setup: `config.yaml`'s `git.managed.files` (or `folders`) declares a path that does not exist
anywhere in the configured repository.

**Run**: `init`, selecting any agent.

**Expected**: `init` exits non-zero before writing or removing anything, printing an actionable
error naming the missing declared path, followed by "Re-run the command to retry."
