# Feature Specification: Agent Repository Clone

**Feature Branch**: `002-agent-repo-clone`

**Created**: 2026-09-05

**Status**: Draft

**Input**: User description: "I want to add a configurable git repository and commit to the agents.yaml file. After the user selects the Agent, I want the CLI to clone the git repo and write the files to a path either specified by the user or to the working directory on the local machine."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Fetching configured files after selection (Priority: P1)

A developer runs `init` and completes the existing agent selection step. The
tool then retrieves the configured git repository at its configured commit
and writes the resulting files into the developer's current working
directory. Which specific agent was selected does not affect which
repository is retrieved at this stage — the repository and commit are a
single, tool-wide configuration; mapping a repository to a specific selected
agent is out of scope for this feature and is expected to be addressed in a
future spec.

**Why this priority**: This is the core value of the feature — without it,
completing `init` produces no usable output beyond a confirmation message.

**Independent Test**: Can be fully tested by running `init`, completing
agent selection, and verifying the expected files appear in the current
working directory after the command completes.

**Acceptance Scenarios**:

1. **Given** a git repository URL and commit reference are configured,
   **When** the developer completes agent selection, **Then** the tool
   clones/fetches that repository at that exact commit and writes its files
   into the current working directory.
2. **Given** the files have been written, **When** the command finishes,
   **Then** the tool confirms to the developer where the files were written.

---

### User Story 2 - Choosing a destination path (Priority: P2)

A developer runs `init` and wants the retrieved files written somewhere other
than the current working directory, so they provide an explicit destination
path.

**Why this priority**: Useful for scaffolding into a specific project
directory, but the feature is still valuable without it (User Story 1's
working-directory default covers the common case).

**Independent Test**: Can be fully tested by running `init` with an explicit
destination path, selecting an agent, and verifying the files are written to
that path instead of the current working directory.

**Acceptance Scenarios**:

1. **Given** the developer supplies an explicit destination path, **When**
   they complete agent selection, **Then** the tool writes the retrieved
   files into that path instead of the current working directory.
2. **Given** the developer supplies a destination path that does not yet
   exist, **When** they complete agent selection, **Then** the tool creates
   the path and writes the files into it.

---

### User Story 3 - Handling a failed or invalid retrieval (Priority: P2)

A developer runs `init`, but the configured git repository cannot be
retrieved (unreachable, invalid URL, invalid or missing commit reference).
The tool must not crash, must not write partial files, and must clearly
report what went wrong.

**Why this priority**: Reliability of the core flow — without clear error
handling, a network or configuration problem could leave a developer's
directory in a broken or confusing state.

**Independent Test**: Can be fully tested with a configured repository or
commit reference that is invalid/unreachable and verifying the tool reports
a clear, actionable error and leaves no partial files behind.

**Acceptance Scenarios**:

1. **Given** the configured repository URL is unreachable or invalid,
   **When** the developer completes agent selection, **Then** the tool
   reports a clear, actionable error and exits without writing any files.
2. **Given** the configured commit reference does not exist in the
   repository, **When** the developer completes agent selection, **Then**
   the tool reports a clear, actionable error and exits without writing any
   files.

---

### Edge Cases

- What happens when the destination directory already contains files? Retrieved files overwrite any existing files at colliding paths; non-colliding existing files are left untouched.
- What happens when no git repository is configured at all? The tool MUST report a clear error rather than silently doing nothing.
- What happens when the local machine cannot reach the git repository (no network, DNS failure, timeout)? Reported as a retrieval error per User Story 3.
- What happens when the developer cancels (Ctrl+C) while the clone/write is in progress? The tool MUST NOT leave a partially-written destination that looks complete; it MUST report cancellation and exit non-zero.
- What happens when the configured git repository requires authentication? Out of scope: the configured repository is assumed to be publicly accessible without credentials; authenticated/private repository access is not supported by this feature.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The embedded catalog configuration MUST allow a single git repository URL and a single commit reference (branch, tag, or commit SHA) to be configured, independent of any specific agent entry.
- **FR-002**: After the developer completes agent selection, the system MUST retrieve the configured git repository at its configured commit reference. Which agent was selected MUST NOT affect which repository or commit is retrieved at this stage.
- **FR-003**: The system MUST write the retrieved repository's files to a destination directory: an explicit path supplied by the developer, or the current working directory if no path is supplied.
- **FR-004**: The system MUST create the destination directory if it does not already exist.
- **FR-005**: The system MUST report a clear, actionable error and exit non-zero without writing any files if the repository cannot be reached, the URL is invalid, or the configured commit reference does not exist.
- **FR-006**: The system MUST report a clear, actionable error and exit non-zero if no git repository is configured.
- **FR-007**: The system MUST report a clear, actionable error and exit non-zero, with no partial files left in a completed-looking state, if retrieval or writing is cancelled or interrupted before it completes.
- **FR-008**: The system MUST confirm to the developer, on success, where the retrieved files were written.
- **FR-009**: The system MUST overwrite any existing file at the destination whose path collides with a retrieved file; existing files at non-colliding paths MUST be left untouched.
- **FR-010**: The system MUST assume the configured git repository is publicly accessible without credentials; authenticated/private repository retrieval is out of scope for this feature.
- **FR-011**: The system MUST accept an optional destination path parameter on `init`; when supplied, it MUST be used as the destination directory instead of the current working directory.

### Key Entities

- **Repository Configuration**: The single, tool-wide git repository URL and commit reference retrieved after agent selection; configured independently of any specific agent entry (a future spec may map specific repositories to specific agents).
- **Destination Path**: The directory on the local machine where retrieved files are written; either explicitly supplied by the developer or defaulted to the current working directory.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of successful `init` runs that complete agent selection while a valid repository and commit are configured result in that repository's files being present in the destination directory.
- **SC-002**: 100% of retrieval failures (unreachable repository, invalid URL, invalid commit, cancellation) result in a clear, actionable error and zero files written, never a partial or silently broken destination.
- **SC-003**: A developer who supplies an explicit destination path finds the files there, not in the current working directory, 100% of the time.
- **SC-004**: A developer can go from completing agent selection to having the files available in the destination directory in a time proportional to the repository's size, with no unexplained hangs.

## Assumptions

- The git repository and commit reference are a single, tool-wide configuration maintained by the tool's maintainers (per the existing embedded catalog configuration file), not entered by the developer at runtime, and not tied to any specific agent entry. Associating a specific repository with a specific selected agent is explicitly out of scope for this feature and is expected to be addressed in a future spec.
- "Clone the git repo and write the files" means the retrieved repository's file contents are made available at the destination path; it does not imply the destination must remain a fully version-controlled git working copy (no `.git` metadata is written to the destination).
- The configured git repository is assumed to be publicly accessible; no credential handling is in scope for this feature.
- The developer's selection of which agent to use is still not persisted as project state (consistent with the existing `init` feature); the durable outcome of this feature is the files written to disk, not a tracked "active agent" record.
- The destination path, when explicitly supplied, is a local filesystem path (not a remote location).
