# Contract: `init` Destination Merge Semantics

Extends the `init` contract from
[../003-selective-file-retrieval/contracts/cli-init.md](../003-selective-file-retrieval/contracts/cli-init.md).
No CLI flags, arguments, exit codes, or output formats change. This contract documents the
merge-vs-replace guarantee for the write step already described there.

## Write step contract (refined)

When `init` writes the selected agent's declared `files`/`folders` write set to the destination:

1. For each path in the write set: create the path (and any missing parent directories) if it
   does not exist, or overwrite it in place if it does. This is unchanged from feature 003.
2. **Any destination path that is not in the write set is never read, modified, or deleted by
   this step** — regardless of whether that path happens to fall under one of the selected
   agent's declared folders. This is the guarantee this feature adds explicit regression coverage
   for.
3. If a path in the write set is a file but the destination already has a directory at that exact
   path (or vice versa — the write set implies a directory but the destination has a plain file
   at that path), `init` fails with an actionable error naming the conflicting path. The
   conflicting destination entry is left exactly as it was — `init` never deletes it to resolve
   the conflict.
4. If any individual write in step 1 fails partway through processing the write set, `init`
   fails and reports the error; files already written earlier in the same run are **not** rolled
   back. This is a best-effort, non-transactional write — consistent with `init`'s existing
   overwrite-on-collision behavior.
5. Whenever `init` fails per step 3 or step 4, the error output MUST instruct the user to re-run
   the command — both failure modes are safe to retry as-is, with no manual cleanup required.

## Non-goals (explicitly out of contract)

- `init` never deletes or prunes destination files that the repository no longer provides for a
  declared folder — that is a distinct, unimplemented feature.
- `init` does not detect or skip byte-identical content; a matching write-set path is always
  (re)written.
