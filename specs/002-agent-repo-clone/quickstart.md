# Quickstart: Agent Repository Clone

Validates the extended `init` command end-to-end. See [data-model.md](./data-model.md) for entity
details and [contracts/cli-init.md](./contracts/cli-init.md) for the full command contract. This
builds on the base `init` quickstart in `specs/001-cli-init-agent-select/quickstart.md`.

## Prerequisites

- Go 1.23+ installed.
- A terminal (TTY) session — `init` requires an interactive prompt.
- Network access to the configured git repository (this feature's retrieval step is not offline).

## Build

```sh
cd highway-cli-go
go build -o bin/highway ./cmd/highway
```

## Scenario 1 — Default destination (current working directory)

```sh
mkdir -p /tmp/some-empty-project && cd /tmp/some-empty-project
/path/to/bin/highway init
```

**Expected**: After selecting an agent and confirming, the tool clones the configured repository
at its configured commit and writes its files into `/tmp/some-empty-project`. A final message
confirms the destination directory used.

## Scenario 2 — Explicit destination path

```sh
/path/to/bin/highway init /tmp/another-project
```

**Expected**: Files are written to `/tmp/another-project` (created if it did not already exist),
not the current working directory.

## Scenario 3 — Destination already has files

```sh
mkdir -p /tmp/existing-project
echo "keep me" > /tmp/existing-project/untouched.txt
echo "overwrite me" > /tmp/existing-project/<a file that also exists in the configured repo>
/path/to/bin/highway init /tmp/existing-project
```

**Expected**: `untouched.txt` remains unchanged; any file whose path collides with a file in the
retrieved repository is overwritten with the repository's content.

## Scenario 4 — Missing or invalid git configuration

Since the catalog is embedded in the binary, simulate this in unit/integration tests by pointing
the loader at fixture files rather than the real binary:

- `git` block absent, or `repository`/`ref` empty → loader returns a "no git repository
  configured" error.

**Expected**: `init` exits non-zero and prints an actionable error to stderr before any prompt is
shown.

## Scenario 5 — Unreachable repository or invalid commit ref

```sh
# Point a test fixture catalog at an invalid URL or a ref that doesn't exist, then run init.
```

**Expected**: `init` exits non-zero, prints an actionable error to stderr, and writes no files to
the destination directory.
