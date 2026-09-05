# CLI Contract: `init` command (extended)

This extends the existing `init` contract (see the `001-cli-init-agent-select` feature) with the
git retrieval and file-write behavior added by this feature.

## Invocation

```text
highway init [path]
```

- `path` (optional, positional): destination directory for retrieved files. Defaults to the
  current working directory if omitted. At most one positional argument is accepted.

## Preconditions

- The binary has a valid embedded catalog (`internal/agentcatalog/config.yaml`), including a
  top-level `git:` block with a non-empty `repository` and `ref`.

## Behavior

1. Load and validate the embedded catalog (agents list and `git` block).
   - Missing/unparsable/empty agents list → print an actionable error to stderr, exit non-zero
     (unchanged from the `init` feature).
   - Missing/empty `git.repository` or `git.ref` → print an actionable error to stderr, exit
     non-zero. **No prompt is shown, nothing is cloned.**
2. Verify stdin is an interactive terminal (unchanged from the `init` feature).
3. Present the agent selection prompt and confirm the developer's choice (unchanged).
4. Clone the configured `git.repository` into memory and check out `git.ref`.
   - Unreachable/invalid repository URL, or a commit ref that does not exist → print an
     actionable error to stderr, exit non-zero. **No files are written.**
5. Resolve the destination directory: the positional `path` argument if supplied, otherwise the
   current working directory. Create it if it does not already exist.
6. Write every file from the retrieved repository into the destination directory, overwriting any
   existing file at a colliding path; files at non-colliding paths are left untouched. No `.git`
   metadata is ever written.
7. On success, print a confirmation naming the destination directory the files were written to.
   Exit 0.
8. On cancellation/interruption at any point before step 6 completes, print no success
   confirmation and exit non-zero.

## Outputs

| Channel | Content |
|---------|---------|
| stdout  | Interactive prompt UI; agent-selection confirmation; final confirmation naming the destination directory |
| stderr  | Actionable error messages for missing/empty/malformed catalog, missing git configuration, no-TTY environment, unreachable/invalid repository, invalid commit ref |
| exit code | `0` on completed retrieval and write; non-zero on any error or cancellation |

## Out of scope for this contract

- Authenticated/private repository access (FR-010).
- Mapping a specific repository to a specific selected agent (deferred to a future spec).
- Flags for non-interactive/unattended agent selection (unchanged from the `init` feature).
