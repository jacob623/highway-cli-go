# Research: Agent Repository Clone

All items below were resolved from the user-provided technical context; no
`NEEDS CLARIFICATION` markers remain in the plan.

## Git client library

- **Decision**: `github.com/go-git/go-git/v6`, cloning into a `memory.Storage` object store with
  a `go-billy` `memfs` worktree filesystem.
- **Rationale**: Pure-Go git implementation, so the CLI does not require a system `git` binary to
  be installed — preserving the constitution's cross-platform build/run guarantee. Supports
  cloning entirely to in-memory storage, satisfying the "clone to internal memory" requirement
  with no on-disk `.git` metadata or temporary directory. Actively maintained (latest commit 2
  days old at time of writing), satisfying the Technology & Dependency Constraints "commit no more
  than 1 month old" bar.
- **Alternatives considered**: Shelling out to the system `git` binary via `os/exec` (rejected —
  requires `git` to be installed and on `PATH`, which the CLI cannot guarantee across macOS/Linux/
  Windows, and would need a temporary on-disk clone rather than an in-memory one).

## In-memory filesystem for the worktree

- **Decision**: `github.com/go-git/go-billy/v6`'s `memfs` package, used as the `git.Clone`
  worktree filesystem.
- **Rationale**: `go-git`'s in-memory clone API requires a `billy.Filesystem` for the worktree;
  `go-billy` is the reference implementation `go-git` itself uses in its own in-memory example,
  and is maintained by the same project. Actively maintained (latest commit 3 weeks old).
- **Alternatives considered**: Writing the clone to a temporary on-disk directory and deleting it
  afterward (rejected — briefly creates a real `.git` working copy on disk, contradicting the
  in-memory requirement, and adds cleanup-on-error/interrupt complexity).

## Resolving an exact commit SHA

- **Decision**: Perform a full (non-shallow) `git.Clone` of the configured repository's default
  branch, then call `Worktree.Checkout(&git.CheckoutOptions{Hash: plumbing.NewHash(ref)})` to move
  to the configured commit SHA.
- **Rationale**: Shallow/partial fetch of an arbitrary commit SHA is not universally supported by
  git servers (support for "want sha1" varies by host), while a full clone guarantees the target
  commit's object is present locally as long as it is reachable from a fetched branch/tag. Given
  the configured repository is expected to be modestly sized (a small skills/config repository),
  the extra history transferred by a full clone is an acceptable, simple trade-off.
- **Alternatives considered**: Shallow clone with `Depth: 1` pinned directly to the ref (rejected —
  `go-git` cannot shallow-fetch an arbitrary SHA that isn't a branch/tag tip on all git hosting
  providers; unreliable across the range of repositories this config could point to).

## Writing retrieved files to the destination

- **Decision**: Walk the in-memory worktree filesystem recursively; for each file, create any
  missing destination parent directories, then write the file's bytes to the destination path,
  truncating/overwriting any existing file at that exact path. Files at destination paths that do
  not collide with a retrieved file are left untouched.
- **Rationale**: Directly satisfies FR-009 (overwrite colliding files only) and keeps the
  destination filesystem interaction limited to the standard library (`os`, `path/filepath`), with
  no new dependency needed for this step.
- **Alternatives considered**: Deleting the entire destination directory before writing (rejected —
  would delete unrelated developer files not present in the retrieved repository, violating
  FR-009's "non-colliding files untouched" requirement).

## Destination path argument

- **Decision**: Accept an optional positional argument on `init` (`init [path]`) for the
  destination directory; when omitted, the destination defaults to the current working directory
  (`os.Getwd()`). Implemented with cobra's `cobra.MaximumNArgs(1)` so the command still accepts
  zero args.
- **Rationale**: Directly satisfies FR-003/FR-011. A positional argument keeps the common case
  (`highway init /some/path`) terse, matching how similar scaffolding tools (e.g. `git clone
  <repo> [path]`) accept an optional destination directory.
- **Alternatives considered**: A `--path`/`-p` flag (rejected — the destination is the primary,
  single piece of optional input this command takes, so a flag adds unnecessary verbosity
  compared to a positional argument).

## Embedded catalog file rename and git configuration placement

- **Decision**: Rename `internal/agentcatalog/agents.yaml` to `internal/agentcatalog/config.yaml`
  and add a top-level `git:` block (`repository`, `ref`) alongside the existing `agents:` list,
  both loaded and validated by the same `LoadFS` function.
- **Rationale**: The embedded file now holds tool-wide configuration beyond just the agent list;
  `config.yaml` reflects that broader scope. Per FR-001/FR-002, the git repository/commit is
  explicitly independent of any specific agent entry, so it belongs at the top level rather than
  nested under an agent.
- **Alternatives considered**: A second, separate embedded file for git configuration (rejected —
  unnecessary split for two related pieces of tool-wide configuration; adds a second `go:embed`
  target and a second missing/malformed-file failure mode for no real benefit).

## Testing git retrieval without real network access

- **Decision**: In `internal/reposync`'s tests, create a temporary local repository with
  `go-git`'s `PlainInit`, commit fixture files to it, and clone from its local filesystem path
  (no `http(s)://` or real network access involved).
- **Rationale**: Keeps the test suite fully offline and deterministic, consistent with Constitution
  Principle III (Testing Standards) and avoids flaky CI runs caused by real network calls.
- **Alternatives considered**: Mocking the `go-git` API directly (rejected — `go-git`'s clone/
  checkout surface is large; a real local repository fixture exercises the actual code path with
  far less test-only scaffolding).
