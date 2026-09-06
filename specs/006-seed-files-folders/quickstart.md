# Quickstart: Seed Files and Folders

Validates `init`'s new seeded-files/seeded-folders behavior. See [data-model.md](./data-model.md)
for the behavior summary and [contracts/cli-init.md](./contracts/cli-init.md) for the full
contract. This builds on the `005-introduce-managed-files` quickstart at
[`../005-introduce-managed-files/quickstart.md`](../005-introduce-managed-files/quickstart.md).

## Prerequisites

- Go 1.26+ installed.
- A terminal (TTY) session — `init` requires an interactive prompt.
- Network access to the configured git repository.
- `config.yaml`'s `git.seeded` section declares at least one file and one folder (the production
  catalog already declares `files: [architecture.yaml]` and `folders: [custom]`).

## Build

```sh
cd highway-cli-go
go build -o bin/highway ./cmd/highway
```

## Scenario 1 — Seed content appears on a fresh destination

Setup: an empty destination directory.

**Run**: `init` against that destination, selecting any agent.

**Expected**: `architecture.yaml` and every file the repository currently provides under
`custom/` are created at the destination.

## Scenario 2 — Local customizations survive a re-run

Setup: destination already has `architecture.yaml` with content that differs from the
repository's current version (simulating a local edit made after a first `init` run).

**Run**: `init` again against that same destination, selecting any agent.

**Expected**: `architecture.yaml`'s content at the destination is byte-for-byte unchanged — not
overwritten, and no error is reported.

## Scenario 3 — Newly added upstream seed files are adopted without disturbing existing ones

Setup: destination already has `custom/existing.md` (previously seeded, since customized); the
repository's current `custom/` folder now additionally provides `custom/new.md`, which does not
yet exist at the destination.

**Run**: `init` against that destination, selecting any agent.

**Expected**: `custom/new.md` is created; `custom/existing.md`'s content is left completely
unchanged.

## Scenario 4 — A directory occupying a seeded file's path is left alone, not an error

Setup: destination has a directory at the exact path a seeded file (or a seeded folder's file)
would occupy.

**Run**: `init`, selecting any agent.

**Expected**: `init` succeeds; the destination directory at that path is left exactly as it was;
no seed file is written there and no error is reported for that path.

## Scenario 5 — Missing declared seed path fails validation before any write

Setup: `config.yaml`'s `git.seeded.files` (or `folders`) declares a path that does not exist
anywhere in the configured repository.

**Run**: `init`, selecting any agent.

**Expected**: `init` exits non-zero before writing anything, printing an actionable error naming
the missing declared path, followed by "Re-run the command to retry."
