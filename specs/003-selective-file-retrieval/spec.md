# Feature Specification: Selective File Retrieval

**Feature Branch**: `003-selective-file-retrieval`

**Created**: 2026-09-05

**Status**: Draft

**Input**: User description: "I only want to write a subset of the files pulled from the git repository to the local machine. I want to declare these files in the config.yaml"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Retrieving only the subset declared for the selected agent (Priority: P1)

A developer runs `init` and selects an agent, as before. Each agent entry in the embedded
configuration may declare its own files list and/or folders list. The tool retrieves the
configured git repository, but instead of writing every file from that repository to the
destination, it writes only the paths declared for the **selected** agent — individual files
named in that agent's declared files list, plus every file found recursively under each folder
named in that agent's declared folders list. Paths declared for other agents, and any other file
present in the retrieved repository, are not written to the destination.

**Why this priority**: This is the core value of the feature — without it, maintainers cannot
scope down `init`'s output to only the files relevant to the agent a developer actually selected,
and every consumer of the tool receives the entire configured repository's contents whether they
need it or not.

**Independent Test**: Configure two agents, each with a different declared files list and/or
folders list, naming subsets of the paths that exist in the configured repository at the
configured commit. Run `init`, select one agent, and verify that only the paths declared for that
agent appear at the destination — no path declared only for the other agent, and no other
retrieved file, is written.

**Acceptance Scenarios**:

1. **Given** the selected agent's entry declares a non-empty list of individual file paths,
   **When** agent selection completes, **Then** the tool writes only those declared files (found
   in the retrieved repository) to the destination.
2. **Given** the selected agent's entry declares a non-empty list of folder paths, **When** agent
   selection completes, **Then** the tool writes every file found recursively under each declared
   folder to the destination.
3. **Given** the selected agent's entry declares both a files list and a folders list, **When**
   agent selection completes, **Then** the tool writes the union of the declared individual files
   and every file recursively under each declared folder.
4. **Given** the selected agent's entry declares a non-empty files list, a non-empty folders list,
   or both, **When** the retrieved repository also contains paths not covered by either list, or
   paths declared only for a different agent, **Then** those other paths are not written to the
   destination.

---

### User Story 2 - Writing nothing when the selected agent declares no lists (Priority: P2)

A developer runs `init` and selects an agent whose entry does not declare a files list or a
folders list. The tool retrieves the configured repository but writes no files to the
destination, since the agent has not opted into receiving any of the repository's contents.

**Why this priority**: Explicit opt-in — an agent only receives files it has actually declared;
an agent with no declared lists is signaling it needs nothing from the repository, so `init` MUST
NOT fall back to writing everything on the agent's behalf.

**Independent Test**: Configure an agent entry with no declared files list and no declared folders
list (or both empty), run `init`, select that agent, and verify no files from the retrieved
repository are written to the destination, and the run still completes successfully.

**Acceptance Scenarios**:

1. **Given** the selected agent's entry declares no files list and no folders list (both keys
   absent), **When** agent selection completes, **Then** the tool writes no files to the
   destination and the run completes without error.
2. **Given** the selected agent's entry declares an empty files list and an empty folders list,
   **When** agent selection completes, **Then** the tool writes no files to the destination
   (empty lists are treated the same as absent lists), and the run completes without error.

---

### User Story 3 - Handling a declared entry that does not exist in the repository (Priority: P2)

A maintainer declares a file path or folder path for an agent in the configuration that does not
actually exist in the configured repository at the configured commit (for example, a typo or a
path that was later removed upstream). The tool must not crash and must clearly report the
problem.

**Why this priority**: Reliability of the configuration — without clear error handling, a
maintainer's mistake in an agent's declared list could silently produce incomplete or confusing
output for every developer selecting that agent.

**Independent Test**: Configure an agent's declared files list or folders list containing at least
one path that does not exist in the configured repository, run `init`, select that agent, and
verify the tool reports a clear, actionable error naming the missing path(s) and does not silently
continue as if nothing were wrong.

**Acceptance Scenarios**:

1. **Given** the selected agent's entry declares a file path that does not exist in the retrieved
   repository, **When** agent selection completes, **Then** the tool reports a clear, actionable
   error identifying the missing path.
2. **Given** the selected agent's entry declares a folder path that does not exist in the
   retrieved repository, **When** agent selection completes, **Then** the tool reports a clear,
   actionable error identifying the missing path.

---

### Edge Cases

- What happens when an agent's declared files list or declared folders list contains duplicate
  entries? Duplicates MUST be treated as a single entry; no duplicate-related error and no
  duplicate write attempt.
- What happens when a file named in an agent's declared files list also falls under a folder named
  in that same agent's declared folders list? The file MUST be written once; the overlap MUST NOT
  produce an error or a duplicate write attempt.
- What happens when a path is declared for one agent but not the selected agent? It MUST NOT be
  written; only the selected agent's declared paths determine what is written (per User Story 2,
  nothing is written if the selected agent declares no lists).
- What happens when a declared path (file or folder) attempts to reference a location outside the
  retrieved repository (for example, using `..` segments)? The tool MUST reject such a path as
  invalid configuration rather than writing to a location outside the intended destination
  directory.
- What happens when a declared path uses a different path-separator style than the local OS? The
  tool MUST interpret declared paths using the repository's own separator convention (forward
  slash), independent of the local operating system.
- What happens when the selected agent's declared lists are present but every entry is missing
  from the repository? The tool MUST report the failure per User Story 3 and write no files.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Each agent entry in the embedded catalog configuration MUST support an optional
  declared files list, naming individual file paths relative to the root of the configured git
  repository, that scopes which retrieved files are written to the destination when that agent is
  selected.
- **FR-001a**: Each agent entry in the embedded catalog configuration MUST support a separate,
  optional declared folders list, naming folder paths relative to the root of the configured git
  repository, whose contents scope which retrieved files are written to the destination when that
  agent is selected.
- **FR-002**: When the selected agent's declared files list, declared folders list, or both are
  non-empty, the system MUST write only the files at that agent's declared file paths and every
  file found recursively under that agent's declared folder paths, and MUST NOT write any other
  file retrieved from the repository (including paths declared only for a different agent).
- **FR-003**: When the selected agent's declared files list and declared folders list are both
  absent or empty, the system MUST write no files retrieved from the repository to the
  destination, and MUST still complete the run successfully (no error solely on this basis).
- **FR-004**: When a path in the selected agent's declared files list or declared folders list
  does not exist in the retrieved repository at the configured commit, the system MUST fail the
  entire retrieval, report a clear, actionable error identifying the missing path(s), exit
  non-zero, and write no files to the destination.
- **FR-005**: The system MUST treat duplicate entries within an agent's declared files list,
  within that agent's declared folders list, or a file that falls under both a declared file entry
  and a declared folder entry for that same agent, as producing a single write with no
  duplicate-related error.
- **FR-006**: The system MUST reject a declared path (file or folder), for any agent, that resolves
  outside the root of the retrieved repository (for example, containing `..` segments) as invalid
  configuration.
- **FR-007**: The system MUST continue to overwrite only destination files whose path collides
  with a file being written, leaving non-colliding existing destination files untouched, applied to
  the set of files actually written under this feature.
- **FR-008**: Each entry in an agent's declared files list MUST name an individual file; each
  entry in that agent's declared folders list MUST name a folder whose contents are included
  recursively (every file found at any depth beneath it).

### Key Entities

- **Declared Files List**: An optional, ordered set of individual file paths, relative to the
  retrieved repository's root, configured on a single agent entry alongside that agent's `id` and
  `display_name`. When present and non-empty, each entry scopes `init`'s file-writing step to
  include that exact file, only when this agent is the one selected.
- **Declared Folders List**: An optional, ordered set of folder paths, relative to the retrieved
  repository's root, configured on a single agent entry alongside its declared files list. When
  present and non-empty, each entry scopes `init`'s file-writing step to include every file found
  recursively beneath that folder, only when this agent is the one selected.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of successful `init` runs where the selected agent has a non-empty declared
  files list, a non-empty declared folders list, or both, result in exactly that agent's declared
  files and the files recursively under that agent's declared folders being present at the
  destination, and no other retrieved file.
- **SC-002**: 100% of successful `init` runs where the selected agent has no declared files list
  and no declared folders list (or both empty) result in no files being written to the
  destination, with the run still completing successfully.
- **SC-003**: 100% of `init` runs where the selected agent declares a file or folder path missing
  from the retrieved repository result in a clear, actionable error identifying the missing path,
  never a silent partial or confusing result.

## Assumptions

- The declared files list and declared folders list are maintained by the tool's maintainers on
  each agent entry in the embedded configuration, not entered by the developer at runtime —
  consistent with how each agent's `id` and `display_name` are configured today.
- Different agents may declare different files lists and folders lists; only the lists on the
  agent the developer selects apply to a given `init` run — an agent that declares neither list
  receives no files at all, rather than falling back to the full repository.
- Declared paths are plain, literal paths (no glob/wildcard pattern matching); folder entries
  include their full contents recursively rather than matching a pattern.
- This feature scopes down the existing file-writing step introduced by the git repository clone
  feature; it does not change agent selection, destination path resolution, or the overwrite-only-
  on-collision behavior for the files that are written.
