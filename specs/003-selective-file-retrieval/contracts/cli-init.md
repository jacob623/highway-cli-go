# CLI Contract: `init` command (extended again)

This extends the `init` contract from `002-agent-repo-clone` (see
[`../../002-agent-repo-clone/contracts/cli-init.md`](../../002-agent-repo-clone/contracts/cli-init.md))
with the per-agent selective file-writing behavior added by this feature. Invocation, agent
selection, and destination-path resolution are unchanged.

## Preconditions (added)

- Each agent entry in the embedded catalog MAY declare a `files` list, a `folders` list, or both,
  in addition to the `id`/`display_name` required by feature 001. Every entry in either list MUST
  be a relative, forward-slash path with no `..` segment.

## Behavior (added, inserted between existing steps 4 and 5 of the 002 contract)

4a. After checkout succeeds, collect the full retrieved file tree in memory (unchanged internal
    step from feature 002).

4b. Determine the write set based on the **selected** agent's declared lists:
   - If the selected agent declares neither `files` nor `folders` (or both are empty): the write
     set is empty — no files are written for this agent (this is not an error; the run still
     completes successfully).
   - Otherwise: the write set is the union of (a) files exactly matching a declared `files` entry
     and (b) files found anywhere under a declared `folders` entry.

4c. Before writing anything, verify every declared `files`/`folders` entry for the selected agent
    matched at least one file in the retrieved tree.
   - If any entry matched nothing: print an actionable error to stderr naming every unmatched
     path, exit non-zero. **No files are written.**

Step 6 (write) then applies only to the computed write set, not necessarily the entire retrieved
tree; the existing overwrite-only-on-collision semantics are unchanged.

## Outputs (added)

| Channel | Content |
|---------|---------|
| stderr  | (added) Actionable error naming every declared `files`/`folders` path absent from the retrieved repository at the configured commit |

## Out of scope for this contract

- Glob/wildcard patterns in `files`/`folders` entries — entries are literal paths.
- Distinguishing an empty-but-present declared folder from an absent one (git does not track
  empty directories; both are treated as a missing declared path).
- Everything already out of scope for the `002-agent-repo-clone` contract.
