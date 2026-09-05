# Feature Specification: Managed Files and Folders

**Feature Branch**: `005-introduce-managed-files`

**Created**: 2026-09-05

**Status**: Draft

**Input**: User description: "I want to introduce managed files and folders to the git configuration in config.yaml. When init is run, these files and folders are managed as a default, regardless of the agent selection. The folders declared as managed folders maintain their exact structure. If a file exists in the destination that does not exist in the source, the file in the destination is removed. Essentially, the managed folder can be replaced in its entirety."

## Clarifications

### Session 2026-09-05

- Q: If replacing a managed folder fails partway through, should the destination be left exactly
  as it was before the run started (all-or-nothing), or is a partial mix of removed/written/
  untouched files acceptable, matching the existing best-effort precedent for per-agent declared
  folders (feature 004)? → A: Best-effort, partial replacement acceptable — matches feature 004's
  existing no-rollback precedent; a failed run may leave the managed folder in a mixed state.
- Q: If a path inside a declared managed folder is a plain file at the destination but the
  repository provides a directory there (or vice versa), should `init` fail with an actionable
  error and leave the conflicting entry untouched, or forcibly remove the conflicting entry and
  replace it? → A: Fail-fast, never delete — same as feature 004's precedent for per-agent
  declared folders; a single conflicting path fails the run rather than being forcibly replaced.
- Q: In both the partial-failure and type-conflict cases above, what should the resulting error
  communicate to the user? → A: The error MUST instruct the user to re-run the command — the
  same requirement retroactively applied to feature 004's analogous partial-failure and
  type-conflict errors (see feature 004's FR-008).
- Q: FR-008 (as originally drafted) required an error whenever a declared managed file *or
  folder* matched nothing in the retrieved repository — but Acceptance Scenario 4 requires a
  managed folder that currently matches nothing (because the repository's maintainer emptied it
  out) to succeed and remove all previously-written destination content. Since a git tree cannot
  distinguish "this folder path never existed" from "this folder currently has zero files," these
  two requirements directly conflicted for the same observable repository state. → A: Resolved in
  favor of the explicit acceptance scenario (and the original feature request's "replaced in its
  entirety" wording, which includes replacement with nothing): FR-008's missing-path validation
  now applies only to managed **files**, never to managed folders. A managed folder matching zero
  repository files always succeeds as a full-removal mirror.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Managed folders exactly mirror the repository, including removing stale content (Priority: P1)

The configuration declares one or more "managed" folders (in addition to, and separate from, any
agent's own declared folders). When `init` runs, each managed folder at the destination is made
to exactly match the retrieved repository's current version of that folder: every file the
repository provides is written, and any file already at the destination under that folder that
the repository does not provide is removed. The managed folder can therefore be fully replaced by
a single `init` run, not merely merged into.

**Why this priority**: This is the defining behavior the user asked for — a way to designate
specific folders as fully repository-owned, where the destination is never left with orphaned or
stale content the repository no longer provides. Without it, this feature is indistinguishable
from the existing per-agent merge behavior (feature 004).

**Independent Test**: Declare a managed folder, populate the destination with a file under that
folder that the repository's current version does not contain, run `init`, and verify that file
is removed while every file the repository does provide is present with matching content.

**Acceptance Scenarios**:

1. **Given** a managed folder declared in `config.yaml`, and a destination containing a file
   under that folder that the retrieved repository's current version does not contain,
   **When** `init` runs, **Then** that file is removed from the destination and every file the
   repository provides under that folder is present with matching content.
2. **Given** a managed folder whose destination copy does not yet exist, **When** `init` runs,
   **Then** the destination gains a full copy of the repository's current version of that folder.
3. **Given** a managed folder containing nested subfolders, **When** `init` runs, **Then** the
   mirroring (including removal of destination files the repository no longer provides) applies
   recursively at every depth, not only at the top level of the managed folder.
4. **Given** the retrieved repository's current version of a managed folder no longer contains
   any files at all, **When** `init` runs, **Then** every file previously at the destination under
   that managed folder is removed.

---

### User Story 2 - Managed files and folders apply automatically on every `init` run, regardless of agent selection (Priority: P2)

Managed files and folders are declared once, independent of any specific agent, and are always
retrieved and applied every time `init` runs — no matter which agent the user selects during that
run. This is in addition to, not a replacement for, whatever files and folders the selected agent
itself declares.

**Why this priority**: Establishes the scope and trigger for User Story 1: without this, a user
could reasonably assume managed content only applies to certain agents, or requires extra steps
to opt in. Confirming it happens unconditionally, alongside the agent-specific write set, is
what makes managed folders a reliable default rather than a per-agent option.

**Independent Test**: Configure a managed folder and at least two agents with different declared
folders. Run `init` selecting each agent in turn, and verify the managed folder's contents are
written to the destination identically regardless of which agent was selected, alongside that
agent's own declared files/folders.

**Acceptance Scenarios**:

1. **Given** a managed folder declared in `config.yaml` and two agents, Agent A and Agent B, each
   with their own distinct declared folders, **When** `init` runs and Agent A is selected,
   **Then** the managed folder's contents are written to the destination in addition to Agent A's
   declared folders.
2. **Given** the same configuration, **When** `init` runs again and Agent B is selected instead,
   **Then** the managed folder's contents are written identically, in addition to Agent B's
   declared folders — the managed folder's presence and content do not depend on which agent was
   selected.

---

### Edge Cases

- What happens when no managed files or folders are declared in `config.yaml`? `init` behaves
  exactly as it does today (features 001-004): only the selected agent's declared files/folders
  are written, with no managed-folder processing at all.
- What happens when a managed folder overlaps a path also covered by the currently selected
  agent's own declared folder? The managed folder's mirror-and-remove-stale-files behavior applies
  to that overlapping path — managed processing is authoritative over any path it covers,
  regardless of what the selected agent separately declares for that same path.
- What happens to a destination file that sits in the same parent directory as a managed folder,
  but not inside it? It is outside the managed folder's path prefix, so it is left untouched,
  consistent with the equivalent precedent for per-agent declared folders (feature 004).
- What happens to a declared managed **file** (not a folder) that no longer exists in the
  retrieved repository? Only the write is skipped; any previously-written copy already at the
  destination is left in place. Managed files behave like existing per-agent declared files
  (create-or-overwrite only) — the "replace in entirety" / stale-file-removal behavior is scoped
  specifically to managed **folders**, per the feature description.
- What happens if a declared managed file or folder does not exist anywhere in the retrieved
  repository at all? `init` fails with an actionable error naming the missing path, consistent
  with the existing validation for per-agent declared files/folders (feature 003) — a managed
  entry that matches nothing in the repository is treated as a configuration error, not silently
  skipped.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Configuration MUST support declaring "managed" files and "managed" folders as part
  of the git configuration, independent of any specific agent's own declared files/folders.
- **FR-002**: On every `init` run, the system MUST retrieve and apply all declared managed files
  and folders, regardless of which agent is selected during that run.
- **FR-003**: For each declared managed folder, the destination MUST be made to exactly mirror the
  retrieved repository's current version of that folder: every file the repository provides under
  it MUST be created or overwritten, and every file already at the destination under that folder
  that the repository does not provide MUST be removed. This mirroring MUST apply recursively
  through every level of nesting within the managed folder.
- **FR-004**: Declared managed files (not folders) MUST be created or overwritten with the
  repository's content at their exact declared path; a managed file no longer present in the
  repository MUST simply not be written — any previously-written copy at the destination is left
  in place (no deletion applies to managed files).
- **FR-005**: Managed files and folders MUST apply in addition to, not instead of, the currently
  selected agent's own declared files/folders — both write sets are applied during the same
  `init` run.
- **FR-006**: If replacing a managed folder fails partway through, the system MUST NOT roll back
  files or removals it already performed during that run — a partial mix of removed, written, and
  untouched files is acceptable and expected, consistent with the existing best-effort precedent
  for per-agent declared folders (feature 004).
- **FR-007**: If a path inside a declared managed folder is a plain file at the destination but
  the repository provides a directory there (or vice versa), the system MUST fail with an
  actionable error naming the conflicting path and MUST NOT delete the existing file or directory
  to resolve the conflict, consistent with the existing precedent for per-agent declared folders
  (feature 004).
- **FR-008**: If a declared managed **file** does not match anything in the retrieved
  repository, `init` MUST fail with an actionable error naming the missing path, without writing
  or removing any files, consistent with existing validation for per-agent declared files. This
  check does not apply to managed **folders**: a managed folder currently matching zero files in
  the retrieved repository is indistinguishable from one that was fully emptied out (git trees do
  not record empty directories), and per FR-003/Acceptance Scenario 4 this MUST be treated as a
  valid "mirror to nothing" outcome — removing every previously-written file under that folder —
  not a configuration error.
- **FR-009**: Whenever `init` fails because of a partial-replacement failure (FR-006) or a type
  conflict (FR-007) involving a managed file or folder, the error output MUST instruct the user
  to re-run the command.

### Key Entities

- **Managed File**: A file path declared once in `config.yaml`'s git configuration, independent
  of any agent, that is created or overwritten (never removed) at its exact destination path on
  every `init` run.
- **Managed Folder**: A folder path declared once in `config.yaml`'s git configuration,
  independent of any agent, whose destination contents are made to exactly mirror the retrieved
  repository's current version of that folder on every `init` run — including removal of
  destination files the repository no longer provides.
- **Destination**: The local directory `init` writes into (existing entity, features 001-004);
  now additionally subject to managed file/folder mirroring on every run.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: After any `init` run, 100% of a declared managed folder's destination contents
  exactly match the retrieved repository's current version of that folder — no extra files, no
  missing files, no stale content.
- **SC-002**: Declared managed files and folders are present at the destination after 100% of
  `init` runs, regardless of which configured agent was selected during each run.
- **SC-003**: Re-running `init` against the same destination and repository state twice in a row
  produces identical destination contents for every declared managed file and folder (idempotent
  mirroring).

## Assumptions

- Managed files and folders are declared independently of agents in `config.yaml` (e.g., under a
  `managed` section of the existing `git` configuration), rather than being tied to any agent's
  `files`/`folders` list.
- This feature does not change the existing per-agent behavior established in features 001-004
  for any path that is not covered by a declared managed folder or file — the selected agent's own
  declared files/folders continue to use the existing merge/never-delete semantics (feature 004).
- Declaring no managed files or folders (an absent or empty `managed` configuration) is
  fully backward compatible: `init` behaves exactly as it does today.
- Managed folders/files declared in `config.yaml` are validated against the retrieved repository
  the same way per-agent declared files/folders already are (feature 003) — a declared managed
  path that matches nothing in the repository is a configuration error, not a silent no-op.
