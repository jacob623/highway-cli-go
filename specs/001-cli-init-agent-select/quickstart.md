# Quickstart: CLI Init Agent Selection

Validates the `init` command end-to-end. See [data-model.md](./data-model.md)
for entity details and [contracts/cli-init.md](./contracts/cli-init.md) for
the full command contract.

## Prerequisites

- Go 1.23+ installed.
- A terminal (TTY) session — `init` requires an interactive prompt.

## Build

```sh
cd highway-cli-go
go build -o bin/highway ./cmd/highway
```

## Scenario 1 — First-time selection (User Story 1)

```sh
cd /tmp/some-empty-project
/path/to/bin/highway init
```

**Expected**: An interactive list showing every agent from the embedded
catalog (e.g., GitHub Copilot, Claude Code, Cursor). Selecting one prints a
confirmation naming that agent. Nothing is written to disk — the selection
only applies to this invocation.

## Scenario 2 — Missing/empty/malformed catalog (User Story 2)

Since the catalog is embedded in the binary, simulate this in unit/integration
tests by pointing the loader at fixture files rather than the real binary:

- Missing file → loader returns a "configuration not found" error.
- File with `agents: []` (or no `agents` key) → loader returns a "no
  supported agents available" error.
- File with invalid YAML syntax → loader returns a "failed to parse
  configuration" error.

**Expected** in all three cases: `init` exits non-zero and prints an
actionable error to stderr.

## Scenario 3 — No interactive terminal

```sh
/path/to/bin/highway init < /dev/null > /tmp/out.txt 2>/tmp/err.txt
echo $?
```

**Expected**: Non-zero exit, an actionable error in `/tmp/err.txt` about no
interactive terminal being available, and no hang.

## Scenario 4 — Cancel mid-prompt

```sh
/path/to/bin/highway init
# press Ctrl+C before choosing an agent
```

**Expected**: Non-zero exit and no confirmation message.
