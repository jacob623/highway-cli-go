# Quickstart: Selective File Retrieval

Validates the further-extended `init` command end-to-end. See [data-model.md](./data-model.md)
for entity details and [contracts/cli-init.md](./contracts/cli-init.md) for the full command
contract. This builds on the `002-agent-repo-clone` quickstart at
[`../002-agent-repo-clone/quickstart.md`](../002-agent-repo-clone/quickstart.md).

## Prerequisites

- Go 1.26+ installed.
- A terminal (TTY) session — `init` requires an interactive prompt.
- Network access to the configured git repository.
- A fixture catalog (`config.yaml`) where at least one agent entry declares `files` and/or
  `folders`, and at least one agent entry declares neither — used via `agentcatalog.LoadFS`
  fixtures in tests rather than the embedded production file.

## Build

```sh
cd highway-cli-go
go build -o bin/highway ./cmd/highway
```

## Scenario 1 — Selected agent declares a files list only

Fixture catalog:

```yaml
git:
  repository: https://example.com/example/repo.git
  ref: <commit>
agents:
  - id: vscode
    display_name: GitHub Copilot
    files:
      - README.md
      - docs/guide.md
```

**Expected**: After selecting `vscode`, only `README.md` and `docs/guide.md` (if present in the
retrieved repository) are written to the destination — no other retrieved file appears.

## Scenario 2 — Selected agent declares a folders list only

Fixture catalog agent entry:

```yaml
  - id: cursor
    display_name: Cursor
    folders:
      - .github/skills
```

**Expected**: After selecting `cursor`, every file found recursively under `.github/skills` in the
retrieved repository is written to the destination — no file outside that folder appears.

## Scenario 3 — Selected agent declares both files and folders, with overlap

Fixture catalog agent entry:

```yaml
  - id: claude-code
    display_name: Claude Code
    folders:
      - .github/skills
    files:
      - .github/skills/idea-init/SKILL.md
      - file1.md
```

**Expected**: The union of every file under `.github/skills` plus `file1.md` is written exactly
once each — the overlap between the `files` entry and the `folders` entry produces no error and
no duplicate write.

## Scenario 4 — Selected agent declares neither list

**Expected**: `init` still clones and checks out the configured repository/ref successfully, but
writes **no** files to the destination for this agent, and the run completes successfully (exit
0) — declared lists are opt-in, not a narrowing of a "write everything" default.

## Scenario 5 — Selected agent declares a path missing from the repository

Fixture catalog agent entry:

```yaml
  - id: vscode
    display_name: GitHub Copilot
    files:
      - does/not/exist.md
```

**Expected**: `init` exits non-zero, prints an actionable error to stderr naming
`does/not/exist.md`, and writes **no** files to the destination — even if other declared entries
for the same agent would otherwise have matched.

## Scenario 6 — Declared path attempts path traversal

Fixture catalog agent entry:

```yaml
  - id: vscode
    display_name: GitHub Copilot
    files:
      - ../outside-repo.md
```

**Expected**: Catalog loading itself fails (before any prompt is shown) with an actionable error
identifying the invalid declared path.
