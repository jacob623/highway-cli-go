# Feature Specification: Merge Declared Folders Into Destination

**Feature Branch**: `004-folder-merge-overwrite`

**Created**: 2026-09-05

**Status**: Draft

**Input**: User description: "when the init command is run, I want to merge the folders with the destination. Only overwrite files on the local file system that exist within the declared folders. Example: If the declared folder in the config.yaml file is .github/skills and the git repository contains .github/skills/highway-activities and the destination contains .github/skills/highway-activities and .github/skills/speckit-specify, .github/skills/highway-activities is overwritten and .github/skills/speckit-specify is left alone."

**Scope**: This feature's merge/preserve guarantee applies only to the `files` and `folders`
paths declared for the **agent selected in the current run**, as read from `config.yaml`
(`internal/agentcatalog/config.yaml`). It does not change, and does not make any claim about,
behavior for destination paths outside those declared entries — including paths that happen to
match a *different* agent's declared folder, or any path not declared for any agent. Those paths
are already, and remain, entirely outside `init`'s write set and are never touched, independent
of this feature.

## Clarifications

### Session 2026-09-05

- Q: If writing the merge partway through fails (e.g. a disk error after some files have already been overwritten), should `init` roll back the files it already wrote during that run, or is it acceptable for the destination to be left with a partial merge? → A: Best-effort — leave whatever was already written before the failure; a partial merge is acceptable, matching current sequential-write behavior. No rollback is required.
- Q: If a path the repository wants to write as a file is already a directory at the destination (or vice versa), should `init` fail with an error, or should it forcibly replace the conflicting entry? → A: Fail with an actionable error naming the conflicting path; write nothing for that run, consistent with existing fail-fast behavior. `init` never deletes an existing directory or file to make room.
- Q: In both the partial-merge-failure and type-conflict-failure cases above, what should the resulting error communicate to the user? → A: The error MUST instruct the user to re-run the command — since neither failure leaves the destination in a state requiring manual cleanup (best-effort partial merge and fail-fast-no-delete are both safe to retry as-is), re-running `init` is always the correct, sufficient recovery action.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Retrieved folder contents overwrite only their own paths at the destination (Priority: P1)

A user re-runs `init` against a destination that already contains prior output for a declared
folder. Only the files that exist within that declared folder in the retrieved repository are
created or overwritten at the destination; anything else already present under that same
declared folder — but not present in the retrieved repository's version of it — is left exactly
as it was.

**Why this priority**: This is the entire feature: without it, re-running `init` risks silently
destroying content the user added or kept at the destination that happens to live under a
declared folder's path, which would make `init` unsafe to re-run.

**Independent Test**: Populate a destination with two entries under a declared folder, where the
retrieved repository only contains one of the two. Run `init`, select the agent declaring that
folder, and verify the entry present in the repository is overwritten with the repository's
content while the other entry is untouched (same content, same modification behavior as before
the run).

**Acceptance Scenarios**:

1. **Given** a destination containing `.github/skills/highway-activities` and
   `.github/skills/speckit-specify`, and a retrieved repository whose `.github/skills` declared
   folder contains only `highway-activities`, **When** `init` runs and the agent declaring
   `.github/skills` is selected, **Then** `.github/skills/highway-activities` is overwritten with
   the repository's content and `.github/skills/speckit-specify` is left unchanged.
2. **Given** a destination that does not yet contain any files under a declared folder,
   **When** `init` runs, **Then** every file the retrieved repository has under that declared
   folder is written to the destination.
3. **Given** a destination containing a file at a path that is not under any of the selected
   agent's declared folders or files, **When** `init` runs, **Then** that file is left unchanged
   regardless of what the retrieved repository contains.

---

### User Story 2 - Nested subfolders merge the same way (Priority: P2)

A declared folder contains nested subfolders in both the retrieved repository and the
destination. Merging happens recursively: only the specific nested files the repository provides
are written; sibling files and sibling subfolders already at the destination are left alone.

**Why this priority**: Declared folders are expected to contain nested structure (e.g. multiple
skill subfolders); the merge guarantee only delivers real value if it holds at every depth, not
just at the top level of the declared folder.

**Independent Test**: Populate a destination with a declared folder containing two subfolders,
where the retrieved repository's version of that declared folder only contains one of the two
subfolders (plus a new file inside the other, already-present subfolder). Run `init` and verify
the repository's file is written, the pre-existing sibling subfolder is untouched, and other
pre-existing files inside the shared subfolder that the repository doesn't provide are untouched.

**Acceptance Scenarios**:

1. **Given** a destination containing `.github/skills/a/existing.md` and
   `.github/skills/b/existing.md`, and a retrieved repository whose declared `.github/skills`
   folder contains only `.github/skills/a/new.md`, **When** `init` runs, **Then**
   `.github/skills/a/new.md` is written, and both `.github/skills/a/existing.md` and
   `.github/skills/b/existing.md` are left unchanged.

---

### Edge Cases

- What happens when the destination has no pre-existing content at all under a declared folder?
  The full retrieved folder contents are written (Acceptance Scenario 2 above) — this is not a
  special case, just the merge with an empty starting set.
- What happens when a retrieved file's content is byte-for-byte identical to what's already at
  the destination? It is still (over)written; this feature does not introduce a "skip if
  unchanged" optimization.
- What happens when the selected agent declares a `files` entry (not a folder) that collides with
  a path already at the destination? Existing exact-match overwrite behavior is unchanged — this
  feature only changes behavior for entries that were previously ambiguous: files that exist at
  the destination under a declared folder but do not exist in the retrieved repository's version
  of that folder.
- What happens to destination files at paths that overlap the parent directory of a declared
  folder but are not under it (e.g. a file directly inside a shared parent directory)? They are
  outside the declared folder's path prefix, so they are always left untouched, independent of
  this feature.
- What happens to a destination path that matches a folder declared for a *different* agent than
  the one selected in this run (e.g. another agent also declares `.github/skills` but a
  different agent is selected)? Out of scope for this feature — the merge/preserve guarantee
  applies only to the paths declared in `config.yaml` for the agent selected in the current run;
  paths declared only for other agents are never part of the write set and are already untouched
  under existing (feature 003) behavior.
- What happens if writing fails partway through a merge (e.g. a disk error after some files have
  already been overwritten)? The files already written before the failure remain at the
  destination — no rollback is performed; the run still fails and reports an error, but the
  destination is left with a partial merge rather than reverted to its pre-run state.
- What happens when a path the repository wants to write as a file is already a directory at the
  destination, or vice versa? `init` fails with an actionable error naming the conflicting path
  and writes nothing for that run; it never deletes an existing file or directory to make room
  for the other type.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: When `init` writes files under a folder declared for the selected agent in
  `config.yaml`, the system MUST create or overwrite only the paths that exist within that
  declared folder in the retrieved repository at the destination.
- **FR-002**: The system MUST NOT delete, modify, or otherwise touch any file already present at
  the destination that is not among the paths the retrieved repository provides for the selected
  agent's `files`/`folders` entries declared in `config.yaml`. This includes paths declared only
  for a different agent than the one selected in the current run.
- **FR-003**: Merging MUST apply recursively through every level of nesting within a declared
  folder — a file several directories deep that the repository doesn't provide is left untouched
  even if sibling or ancestor directories are otherwise being written to.
- **FR-004**: When a path within a declared folder exists both in the retrieved repository and at
  the destination, the destination copy MUST be overwritten with the repository's content
  (existing overwrite-on-collision behavior is preserved, not replaced).
- **FR-005**: This merge behavior MUST apply identically regardless of how many declared folders
  or files the selected agent has in `config.yaml`, and regardless of whether the destination
  directory already existed before this run.
- **FR-006**: If a failure occurs partway through writing a declared folder's files, the system
  MUST NOT roll back files it already wrote during that run — a partial merge with a reported
  error is acceptable and expected, consistent with existing sequential-write behavior.
- **FR-007**: If a path the repository provides as a file already exists at the destination as a
  directory, or a path the repository provides as a directory already exists at the destination
  as a file, the system MUST fail with an actionable error naming the conflicting path and MUST
  NOT delete the existing file or directory to resolve the conflict.
- **FR-008**: Whenever `init` fails because of a partial-merge failure (FR-006) or a type conflict
  (FR-007), the error output MUST instruct the user to re-run the command.

### Key Entities

- **Declared Folder**: A folder path declared for a specific agent in `config.yaml`'s `folders`
  list (existing entity from feature 003) whose contents are retrieved from the repository and
  merged — not wholesale replacing — into the corresponding destination path. Only the folders
  declared for the agent selected in the current run are in scope.
- **Destination**: The local directory `init` writes into; may contain pre-existing files from
  prior runs or from the user, some of which fall under a declared folder's path without being
  part of the repository's current contents for that folder.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: After running `init` against a destination with pre-existing content under a
  declared folder, 100% of destination files that are not part of the retrieved repository's
  version of that folder remain byte-for-byte unchanged.
- **SC-002**: After running `init`, 100% of the retrieved repository's files under a selected
  agent's declared folders are present at the destination with matching content.
- **SC-003**: Re-running `init` against the same destination and repository state twice in a row
  produces identical destination contents on both runs (idempotent merge).

## Assumptions

- This feature's merge/preserve guarantee is scoped exclusively to the `files` and `folders`
  entries declared for the selected agent in `config.yaml`. Any destination path not among those
  declared entries — regardless of why it exists at the destination — is out of scope and is
  never read, written, or otherwise reasoned about by this feature.
- This feature refines behavior for **declared folders** specifically; declared **files** already
  only ever affect their own exact path and are unaffected by this change.
- No deletion or "pruning" of destination files that no longer exist in the repository is in
  scope — `init` only ever adds or overwrites; removing stale destination files the repository no
  longer provides is explicitly out of scope for this feature.
- The destination is a local directory the user controls; concurrent external modification of the
  destination while `init` is running is out of scope.
