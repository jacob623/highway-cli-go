# Feature Specification: Seed Files and Folders

**Feature Branch**: `006-seed-files-folders`

**Created**: 2026-09-06

**Status**: Draft

**Input**: User description: "I want to add support for files and folders that are seeded when the
init command is run. These files and folders are declared in the config.yaml file and seeded
regardless of the agent selection. The files are and folders are only written to the destination
if they do not exist in the destination."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Scaffold a new project from declared seed content (Priority: P1)

A developer runs `init` for the first time against an empty (or mostly empty) destination
directory. The declared seed files and folders from the git configuration are created at the
destination automatically, regardless of which agent the developer selects, giving them a
ready-to-customize starting point without manually copying anything.

**Why this priority**: Without this, seed content never appears anywhere — there is no feature at
all.

**Independent Test**: Can be fully tested by running `init` against an empty destination with
declared seed files/folders and verifying every one of them is created with the repository's
content.

**Acceptance Scenarios**:

1. **Given** an empty destination and a git configuration declaring a seeded file, **When**
   `init` runs, **Then** that file is created at the destination with the retrieved repository's
   content.
2. **Given** an empty destination and a git configuration declaring a seeded folder, **When**
   `init` runs, **Then** every file the repository provides under that folder is created at the
   destination, recursively through every level of nesting.
3. **Given** a git configuration declaring seed content, **When** `init` runs with any agent
   selected, **Then** the seed content is created the same way regardless of which agent was
   chosen.

---

### User Story 2 - Local customizations always survive a re-run (Priority: P2)

A developer has already run `init` once and has since edited one of the seeded files (or added
content inside a seeded folder). They run `init` again — perhaps because they changed their
selected agent, or the repository's other declared content changed. Their edits to the
already-existing seeded file must remain completely untouched, no matter what the repository's
current version of that file looks like.

**Why this priority**: This is the defining guarantee that distinguishes "seeded" content from
"managed" content (which always overwrites) — without it, seed content isn't safe to customize
and the feature loses its main value, but the core scaffolding behavior (User Story 1) is already
independently useful on its own.

**Independent Test**: Can be fully tested by pre-populating a destination file at a path declared
as seeded (with content different from the repository's), running `init`, and verifying the file's
content is unchanged afterward.

**Acceptance Scenarios**:

1. **Given** a destination that already has a file at a declared seeded file's path, **When**
   `init` runs, **Then** that file's content is left completely unchanged and no error is
   reported.
2. **Given** a destination that already has a file at a path under a declared seeded folder,
   **When** `init` runs, **Then** that file's content is left completely unchanged even though the
   repository provides a different version of it.

---

### User Story 3 - Newly added seed files are adopted without disturbing existing ones (Priority: P3)

Some time after a developer first scaffolded their project, the repository's maintainers add a new
file under a seed folder the developer already has locally. The developer runs `init` again. The
new file appears at their destination, while every file they already have under that same folder —
customized or not — remains exactly as it was.

**Why this priority**: This confirms the create-if-missing behavior operates per file rather than
per folder, which matters for long-lived projects that periodically re-run `init`, but it is a
refinement of User Stories 1 and 2 rather than essential to initial delivery.

**Independent Test**: Can be fully tested by pre-populating a destination folder with one existing
file, running `init` against a repository that provides both that file (with different content)
and a second, new file under the same declared seed folder, and verifying only the new file is
created while the existing file is untouched.

**Acceptance Scenarios**:

1. **Given** a destination folder under a declared seed folder that already contains one file,
   **When** `init` runs against a repository version of that folder containing an additional file
   not yet present at the destination, **Then** only the missing file is created and the existing
   file is left unchanged.

---

### Edge Cases

- What happens when a directory already occupies the exact path where a seeded file would be
  written (or vice versa)? The path already has something at it, so it counts as "already exists"
  and is left completely untouched — no error, no write, no deletion.
- What happens when an existing destination file at a seeded path is empty (zero bytes)? It still
  counts as existing; seeding does not inspect or compare content, only presence.
- What happens if a declared seeded file or folder does not match anything in the retrieved
  repository at all (e.g., a typo in the declared path)? `init` fails with an actionable error
  naming the missing path, consistent with existing validation for agent-declared and managed
  paths.
- What happens if writing one seeded file fails partway through a run that would have seeded
  several? Files already created earlier in that same run are not rolled back; a partial mix of
  created and not-yet-created seed content is acceptable, consistent with existing best-effort
  behavior for managed folders.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST support declaring seeded files and seeded folders in the git
  configuration of `config.yaml`, independent of any specific agent's own declared files/folders.
- **FR-002**: On every `init` run, the system MUST attempt to seed all declared seeded files and
  folders, regardless of which agent is selected during that run.
- **FR-003**: For each declared seeded file, the system MUST create it at the destination with the
  retrieved repository's content only if nothing (no file, no directory) currently exists at that
  exact destination path.
- **FR-004**: For each declared seeded folder, every file the repository provides under it MUST be
  created at its corresponding destination path only if nothing currently exists at that exact
  path, applied recursively through every level of nesting within the folder.
- **FR-005**: If something already exists at a destination path a seeded file or a seeded folder's
  file would occupy, the system MUST leave it completely untouched — no overwrite, no deletion,
  and no error reported for that path.
- **FR-006**: Seeded files and folders MUST apply in addition to, not instead of, the currently
  selected agent's own declared files/folders and any declared managed files/folders — all three
  write behaviors are applied during the same `init` run.
- **FR-007**: If a declared seeded file or seeded folder does not match anything in the retrieved
  repository, `init` MUST fail with an actionable error naming the missing path, without writing
  any seed content, consistent with existing validation for agent-declared and managed
  files/folders.
- **FR-008**: If seeding fails partway through a run, the system MUST NOT roll back seed files it
  already created during that run — a partial mix of created and not-yet-created seed content is
  acceptable and expected, consistent with the existing best-effort precedent for managed folders.
- **FR-009**: Whenever `init` fails because of a partial-seeding failure (FR-008) or a missing
  declared seed path (FR-007), the error output MUST instruct the user to re-run the command.

### Key Entities

- **Seeded File**: A file path declared once in `config.yaml`'s git configuration, independent of
  any agent, that is created at its exact destination path only the first time nothing exists
  there — never overwritten or removed on any later `init` run.
- **Seeded Folder**: A folder path declared once in `config.yaml`'s git configuration, independent
  of any agent, whose files are each created at the destination only if that exact file does not
  already exist there — existing files under the folder, customized or not, are never overwritten
  or removed on any later `init` run.
- **Destination**: The local directory `init` writes into (existing entity, features 001-005); now
  additionally subject to seed file/folder create-if-missing behavior on every run.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: After running `init` against a completely empty destination, 100% of declared
  seeded files and the repository's current files under declared seeded folders are present at
  the destination.
- **SC-002**: After running `init` against a destination containing a previously-seeded file that
  has since been locally modified, that file's content is preserved byte-for-byte 100% of the
  time.
- **SC-003**: When the repository adds a new file under an already-seeded folder, the next `init`
  run creates exactly that new file while leaving 100% of the folder's pre-existing destination
  files unchanged.
- **SC-004**: Developers can re-run `init` at any time with zero risk of losing local edits made
  to any seeded file or any file under a seeded folder.

## Assumptions

- The "already exists" check is based on path presence only (a file or a directory occupying that
  exact path), not on comparing content — an existing destination file is left alone regardless of
  how its content differs from the repository's version.
- Seeded folders apply the create-if-missing check per individual file recursively, not to the
  top-level folder path as a whole — this matches the per-file recursive approach already used for
  managed folders (feature 005) and is what allows User Story 3's incremental adoption of newly
  added upstream files.
- Declaring the same path as both seeded and managed, or both seeded and an agent's own declared
  file/folder, is a configuration choice outside this feature's scope — no specific interaction or
  conflict-resolution behavior between those categories is defined or validated here.
- A declared seeded folder that matches zero files in the retrieved repository is treated as a
  configuration error (FR-007), the same as a declared seeded file — unlike managed folders, seed
  content is never deleted or mirrored, so there is no legitimate scenario where a seeded folder is
  expected to match nothing.
