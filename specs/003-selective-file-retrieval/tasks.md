---

description: "Task list template for feature implementation"
---

# Tasks: Selective File Retrieval

**Input**: Design documents from `/specs/003-selective-file-retrieval/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/cli-init.md](./contracts/cli-init.md), [quickstart.md](./quickstart.md)

**Tests**: Included and REQUIRED — this repository's constitution (Principle III, Testing
Standards, NON-NEGOTIABLE) mandates automated tests covering success and failure paths for every
behavior change before merge.

**Organization**: Tasks are grouped by user story to enable independent implementation and
testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Single Go CLI project (existing structure from features 001/002 — no new project or directory is
introduced):

- `internal/agentcatalog/` — catalog struct, parsing, validation
- `internal/reposync/` — clone/checkout/collect/filter/write
- `internal/cli/` — `init` command wiring

---

## Phase 1: Setup

**Purpose**: Confirm a green baseline before making changes; no new project scaffolding is needed
since this feature only extends existing packages.

- [X] T001 Run `go build ./...` and `go test ./...` from the repo root to confirm the
      `002-agent-repo-clone` baseline is green before starting feature 003 changes

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Struct/signature changes every user story depends on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T002 [P] Add `Files []string` (`yaml:"files"`) and `Folders []string`
      (`yaml:"folders"`) fields to `AgentDefinition` in
      internal/agentcatalog/catalog.go
- [X] T003 Add an `ErrInvalidDeclaredPath` sentinel error and a `validateDeclaredPaths`
      helper in internal/agentcatalog/catalog.go that rejects an empty, absolute, or
      `..`-containing entry in any agent's `Files`/`Folders` (depends on T002)
- [X] T004 Call `validateDeclaredPaths` for every agent from `LoadFS` in
      internal/agentcatalog/catalog.go, immediately after `dedupeAndValidate` and before
      the existing `Git.Repository`/`Git.Ref` check, returning `ErrInvalidDeclaredPath`
      wrapped with the offending agent id and path on failure (depends on T003)
- [X] T005 [P] Add table-driven tests in internal/agentcatalog/catalog_test.go covering
      `Files`/`Folders` YAML parsing and `ErrInvalidDeclaredPath` rejection for empty,
      absolute, and `..`-containing entries (depends on T004)
- [X] T006 [P] Extend `Sync`'s signature to
      `Sync(ctx context.Context, repository, ref, destination string, files, folders []string) error`
      in internal/reposync/reposync.go; the new parameters are threaded through but not
      yet used for filtering (plumbing only at this stage)
- [X] T007 Update the `syncRepo` var type and the `runInit` call site in
      internal/cli/init.go: capture the selected agent's `Files`/`Folders` in the
      existing agent-lookup loop and pass them to `syncRepo` (depends on T002, T006)
- [X] T008 Update the stubs/helpers in internal/cli/init_test.go (`fakeCatalog`,
      `noopSync`, `withInitStubs`, and any existing call sites) to match the new
      `syncRepo` signature so all existing tests compile and pass (depends on T007)

**Checkpoint**: `go build ./... && go test ./...` is green; `Files`/`Folders` parse and validate;
`Sync`/`init` plumbing compiles. `Sync` still writes every retrieved file regardless of
`files`/`folders` until US1/US2 land — this is expected intermediate state, not user-facing yet.

---

## Phase 3: User Story 1 - Retrieving only the subset declared for the selected agent (Priority: P1) 🎯 MVP

**Goal**: When the selected agent declares a non-empty `files` list, `folders` list, or both,
`init` writes only that subset — nothing else from the retrieved repository.

**Independent Test**: Configure two agents with different declared `files`/`folders`, run `init`,
select one, and verify only that agent's declared paths appear at the destination.

### Tests for User Story 1

- [X] T009 [P] [US1] Add table-driven tests in internal/reposync/reposync_test.go:
      `TestSync_FiltersToDeclaredFiles`, `TestSync_FiltersToDeclaredFolders`,
      `TestSync_FiltersUnionOfFilesAndFolders`, `TestSync_DuplicateAndOverlappingEntriesWriteOnce`
      (write these first; they FAIL until T010 lands)

### Implementation for User Story 1

- [X] T010 [US1] Implement write-set filtering in `Sync` (internal/reposync/reposync.go):
      when `files` and/or `folders` is non-empty, compute the write set as the union of
      (a) collected paths exactly matching a `files` entry and (b) collected paths with a
      `folders` entry as a path-prefix, deduplicated, and write only that set instead of
      every collected file (depends on T006, T009)
- [X] T011 [P] [US1] Add an internal/cli/init_test.go case verifying `runInit` passes the
      selected agent's `Files`/`Folders` through to `syncRepo` unchanged (depends on T007, T008)

**Checkpoint**: User Story 1 is independently functional and testable — declared subsets are
correctly written when non-empty. (The empty-list default is still "write everything" at this
point; that flips in User Story 2.)

---

## Phase 4: User Story 2 - Writing nothing when the selected agent declares no lists (Priority: P2)

**Goal**: When the selected agent declares neither `files` nor `folders` (or both are empty),
`init` writes no files, and still completes successfully.

**Independent Test**: Configure an agent with no declared lists, run `init`, select it, and
verify the destination receives no new files and the run exits 0.

### Tests for User Story 2

- [X] T012 [P] [US2] Add an internal/reposync/reposync_test.go case
      `TestSync_NoDeclaredListsWritesNothing` verifying clone/checkout still succeed, `Sync`
      returns `nil`, and the destination ends up with no files written (write this first; it
      FAILS until T013 lands)

### Implementation for User Story 2

- [X] T013 [US2] Change the default branch in `Sync` (internal/reposync/reposync.go) so that
      when both `files` and `folders` are empty, the write step is skipped entirely (zero
      files written) instead of writing every collected file, while `Sync` still returns `nil`
      on success (depends on T010, T012)
- [X] T014 [P] [US2] Add an internal/cli/init_test.go case verifying `runInit` still prints its
      destination confirmation and exits 0 when the selected agent declares neither list, even
      though nothing is written (depends on T007, T008)

**Checkpoint**: User Stories 1 AND 2 both work independently — declared subsets are honored, and
agents with no declared lists correctly receive nothing.

---

## Phase 5: User Story 3 - Handling a declared entry that does not exist in the repository (Priority: P2)

**Goal**: If any declared `files`/`folders` path for the selected agent is absent from the
retrieved repository at the configured commit, the whole retrieval fails with a clear error and
writes no files.

**Independent Test**: Configure an agent with a declared path that doesn't exist in the fixture
repository, run `init`, select it, and verify a clear error naming the missing path and no files
written.

### Tests for User Story 3

- [X] T015 [P] [US3] Add table-driven tests in internal/reposync/reposync_test.go:
      `TestSync_MissingDeclaredFileFails`, `TestSync_MissingDeclaredFolderFails`,
      `TestSync_MultipleMissingPathsNamedInError`, `TestSync_MissingPathLeavesDestinationUntouched`
      (write these first; they FAIL until T016 lands)

### Implementation for User Story 3

- [X] T016 [US3] Before writing, validate in `Sync` (internal/reposync/reposync.go) that every
      declared `files`/`folders` entry matched at least one collected path; if any entry matched
      nothing, return an error naming every unmatched path and write no files (depends on T010,
      T013, T015)
- [X] T017 [P] [US3] Add an internal/cli/init_test.go case verifying a `Sync` failure from a
      missing declared path surfaces to stderr and exits non-zero, consistent with `runInit`'s
      existing error handling (depends on T007, T008)

**Checkpoint**: All three user stories are independently functional and testable.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Repo-wide validation and closing out maintainer-facing configuration.

- [X] T018 [P] Run `gofmt -l .` and `go vet ./...` across the repo; fix any formatting/vet
      issues surfaced by the new code
- [X] T019 [P] Run `go test ./...` and confirm every package passes
- [X] T020 Update the production internal/agentcatalog/config.yaml with maintainer-authored
      `files`/`folders` declarations for the existing `vscode`, `claude-code`, and `cursor`
      agents, so `init` continues to write the parts of the `highway-skills` repository each
      agent needs now that an agent with no declared lists writes nothing (depends on T004)
- [X] T021 [P] Run `go mod tidy` and confirm `go.mod`/`go.sum` are unchanged (no new
      third-party dependency is required by this feature)
- [X] T022 Manually run the [quickstart.md](./quickstart.md) validation scenarios against a
      local build (`go build -o bin/highway ./cmd/highway`) to confirm end-to-end behavior

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational completion. No dependency on US2/US3.
- **User Story 2 (Phase 4)**: Depends on Foundational completion; its implementation task (T013)
  builds directly on US1's filtering code (T010), so in practice US2 is implemented after US1
  even though it has no independent-testability dependency on US1's *behavior*.
- **User Story 3 (Phase 5)**: Depends on Foundational completion; its implementation task (T016)
  builds on both US1's (T010) and US2's (T013) code in the same function.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### Within Each User Story

- Tests are written first and MUST fail before the corresponding implementation task.
- Implementation task follows its test task in the same story.

### Parallel Opportunities

- T002 (catalog.go struct fields) and T006 (reposync.go signature) touch different packages and
  can be done in parallel.
- T005, T009, T012, T015 (test-writing tasks) are marked [P] relative to tasks in *other* files,
  but note T009/T012/T015 all add cases to the same internal/reposync/reposync_test.go file across
  phases — treat them as parallel only across different contributors picking up different phases
  sequentially, not as simultaneous edits to the same file.
- T011, T014, T017 (init_test.go cases) are marked [P] for the same reason relative to their
  sibling reposync test tasks, not for literal simultaneous edits within init_test.go.
- T018, T019, T021 (repo-wide checks) and T020 (config.yaml content) can run in parallel with each
  other.

---

## Parallel Example: Foundational Phase

```bash
# Launch independent-package foundational tasks together:
Task: "Add Files/Folders fields to AgentDefinition in internal/agentcatalog/catalog.go"
Task: "Extend Sync's signature to accept files, folders []string in internal/reposync/reposync.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational (CRITICAL — blocks all stories).
3. Complete Phase 3: User Story 1.
4. **STOP and VALIDATE**: Confirm declared-subset filtering works independently (note: agents with
   empty lists still write everything until US2 lands — acceptable for an MVP demo of the core
   filtering value, but not yet spec-complete).

### Incremental Delivery

1. Setup + Foundational → baseline ready.
2. Add User Story 1 → declared subsets are honored → validate independently.
3. Add User Story 2 → empty-list agents write nothing → validate independently.
4. Add User Story 3 → missing declared paths fail fast with a clear error → validate independently.
5. Polish → repo-wide checks green, production config.yaml populated, quickstart validated.

---

## Notes

- [P] tasks = different files, no dependencies (see caveats above about test-file tasks across
  phases).
- [Story] label maps task to specific user story for traceability.
- Every user story is independently completable and testable per its own Independent Test.
- Verify each story's tests fail before implementing that story's code.
- Commit after each task or logical group.
- Stop at any checkpoint to validate a story independently.
