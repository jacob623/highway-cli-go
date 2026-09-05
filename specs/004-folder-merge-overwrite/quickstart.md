# Quickstart: Merge Declared Folders Into Destination

Validates the destination-merge refinement to `init`'s write step. See
[data-model.md](./data-model.md) for the behavior summary and
[contracts/cli-init.md](./contracts/cli-init.md) for the full contract. This builds on the
`003-selective-file-retrieval` quickstart at
[`../003-selective-file-retrieval/quickstart.md`](../003-selective-file-retrieval/quickstart.md).

## Prerequisites

- Go 1.26+ installed.
- A terminal (TTY) session — `init` requires an interactive prompt.
- Network access to the configured git repository.
- A destination directory pre-populated with content under a declared folder, some of which the
  configured repository currently provides and some of which it does not.

## Build

```sh
cd highway-cli-go
go build -o bin/highway ./cmd/highway
```

## Scenario 1 — Pre-existing destination content outside the repository's current folder contents survives

Setup:

- Destination already contains `.github/skills/highway-activities/` (with files) and
  `.github/skills/speckit-specify/` (with files).
- The configured repository's `.github/skills` folder currently contains only
  `highway-activities`.

**Run**: `init` against that destination, selecting the agent declaring `.github/skills`.

**Expected**: `.github/skills/highway-activities/*` is overwritten with the repository's current
content; `.github/skills/speckit-specify/*` is byte-for-byte unchanged.

## Scenario 2 — Nested subfolders — merge holds at every depth

Setup:

- Destination contains `.github/skills/a/existing.md` and `.github/skills/b/existing.md`.
- The repository's `.github/skills` folder currently contains only `.github/skills/a/new.md`.

**Run**: `init`, selecting the agent declaring `.github/skills`.

**Expected**: `.github/skills/a/new.md` is written; both `.github/skills/a/existing.md` and
`.github/skills/b/existing.md` are unchanged.

## Scenario 3 — Destination has a directory where the repository provides a file

Setup: destination has a directory at `.github/skills/README.md/` (unusual, but possible if a
prior tool created it), and the repository provides a plain file at `.github/skills/README.md`.

**Run**: `init`, selecting the agent declaring `.github/skills`.

**Expected**: `init` exits non-zero, prints an actionable error naming
`.github/skills/README.md`, and the destination directory at that path is not deleted.

## Scenario 4 — Destination has a file where the repository provides a folder's contents

Setup: destination has a plain file at `.github/skills/nested` (no extension), and the
repository's declared folder contains `.github/skills/nested/inner.md` (i.e. `nested` must be a
directory to hold `inner.md`).

**Run**: `init`, selecting the agent declaring `.github/skills`.

**Expected**: `init` exits non-zero, prints an actionable error naming the conflicting path, and
the destination file at `.github/skills/nested` is not deleted.
