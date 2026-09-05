---

description: "Task list template for feature implementation"
---

# Tasks: Agent Repository Clone

**Input**: Design documents from `/specs/002-agent-repo-clone/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli-init.md, quickstart.md

**Tests**: Included. Constitution Principle III (Testing Standards) is NON-NEGOTIABLE, requiring
automated tests for every behavior change; git retrieval is tested against local fixture
repositories (no real network access) per research.md.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing
of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Paths below follow plan.md's Project Structure (single Go project: `internal/agentcatalog/`,
  `internal/reposync/`, `internal/cli/`)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Add `github.com/go-git/go-git/v6` and `github.com/go-git/go-billy/v6` to go.mod via `go get`
- [X] T002 [P] Rename `internal/agentcatalog/agents.yaml` to `internal/agentcatalog/config.yaml`, updating the `//go:embed` directive and `Load()`'s `LoadFS(embeddedFS, "config.yaml")` call in `internal/agentcatalog/catalog.go` accordingly

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T003 Add a top-level `git: {repository, ref}` block to `internal/agentcatalog/config.yaml` with `repository: https://github.com/jacob623/ihighway-skills.git` and `ref: 309b50d309478e3b2417120396a8fc9179c99a30`, alongside the existing `agents` list, per data-model.md
- [X] T004 [P] Define the `GitConfig` struct (`Repository`, `Ref` string, yaml tags `repository`/`ref`) in `internal/agentcatalog/catalog.go`
- [X] T005 Add a `Git GitConfig` field (yaml tag `git`) to the `AgentCatalog` struct in `internal/agentcatalog/catalog.go`
- [X] T006 Implement git config validation in `LoadFS`/`dedupeAndValidate`: introduce `ErrGitConfigMissing` and return it when `git.repository` or `git.ref` is empty, independent of agents-list validation (FR-006)
- [X] T007 [P] Create the `internal/reposync` package skeleton in `internal/reposync/reposync.go` with an exported function signature (repository URL, commit ref, destination path) returning an error, per research.md's in-memory clone decisions

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Fetching configured files after selection (Priority: P1) 🎯 MVP

**Goal**: After a developer completes agent selection, the tool clones the configured git
repository at its configured commit entirely in memory and writes the resulting files into the
current working directory, then confirms where the files were written.

**Independent Test**: Run `init`, complete agent selection, and verify the expected files appear
in the current working directory after the command completes (quickstart.md Scenario 1).

### Tests for User Story 1 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T008 [P] [US1] Table-driven test for `LoadFS` parsing a valid `git:` block into `GitConfig` in `internal/agentcatalog/catalog_test.go`
- [X] T009 [P] [US1] Test that the `internal/reposync` fetch function clones a local fixture repository (created with `go-git`'s `PlainInit`) at a specific commit SHA and writes its files to a temp destination directory, in `internal/reposync/reposync_test.go`
- [X] T010 [P] [US1] Test that `init`'s happy path clones the configured repository/ref and writes files into the current working directory, printing a destination confirmation, in `internal/cli/init_test.go`

### Implementation for User Story 1

- [X] T011 [US1] Implement in-memory clone in `internal/reposync/reposync.go` using `git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{URL: ...})` (FR-002)
- [X] T012 [US1] Implement checkout of the configured commit via `Worktree.Checkout(&git.CheckoutOptions{Hash: plumbing.NewHash(ref)})` in `internal/reposync/reposync.go` (FR-002)
- [X] T013 [US1] Implement recursive write-to-destination logic in `internal/reposync/reposync.go`: walk the in-memory `billy.Filesystem`, create missing destination directories, and write each file's bytes to disk (FR-003, FR-004)
- [X] T014 [US1] Wire `internal/reposync` into `internal/cli/init.go`: after the existing agent-selection confirmation, resolve the destination (default `os.Getwd()`), invoke the fetch/write function, and print a success confirmation naming the destination (FR-002, FR-003, FR-008)

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently
(quickstart.md Scenario 1)

---

## Phase 4: User Story 2 - Choosing a destination path (Priority: P2)

**Goal**: A developer can supply an explicit destination path so retrieved files are written there
instead of the current working directory, with the path created if it doesn't already exist.

**Independent Test**: Run `init <path>`, complete agent selection, and verify the files are
written to `<path>` instead of the current working directory (quickstart.md Scenario 2).

### Tests for User Story 2 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T015 [P] [US2] Test that `init <path>` writes retrieved files to the given positional path instead of the current working directory, creating the path if it does not exist, in `internal/cli/init_test.go`

### Implementation for User Story 2

- [X] T016 [US2] Add an optional positional `path` argument to the `init` cobra command using `cobra.MaximumNArgs(1)` in `internal/cli/init.go` (FR-011)
- [X] T017 [US2] Resolve the destination in `internal/cli/init.go`: use the positional argument if provided, otherwise `os.Getwd()`; create the destination directory if it does not exist (FR-003, FR-004)

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Handling a failed or invalid retrieval (Priority: P2)

**Goal**: When the configured repository/commit cannot be retrieved, or no git repository is
configured, the tool reports a clear, actionable error, writes no partial files, and exits
non-zero — including on cancellation.

**Independent Test**: Point the loader/reposync at fixtures with a missing git config, an
unreachable/invalid repository URL, and an invalid commit ref, and verify `init` reports a clear
error and leaves no partial files (quickstart.md Scenarios 4 and 5).

### Tests for User Story 3 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T018 [P] [US3] Test that the reposync fetch function returns an error and writes no files when the repository URL is invalid/unreachable, in `internal/reposync/reposync_test.go`
- [X] T019 [P] [US3] Test that the reposync fetch function returns an error and writes no files when the configured commit ref does not exist in the repository, in `internal/reposync/reposync_test.go`
- [X] T020 [P] [US3] Test that `init` reports an actionable stderr error and exits non-zero, showing no prompt, when the catalog's `git` block is missing/empty (`ErrGitConfigMissing`), in `internal/cli/init_test.go`
- [X] T021 [P] [US3] Test that `init` reports an actionable stderr error and exits non-zero, with no files written, when clone/checkout fails, in `internal/cli/init_test.go`

### Implementation for User Story 3

- [X] T022 [US3] Ensure the reposync fetch/write function in `internal/reposync/reposync.go` only writes files to the destination after clone and checkout have both fully succeeded, so no partial files are written on failure (FR-005, FR-007)
- [X] T023 [US3] In `internal/cli/init.go`, catch `ErrGitConfigMissing` before showing the agent prompt, print an actionable stderr error, and exit non-zero (FR-006)
- [X] T024 [US3] In `internal/cli/init.go`, catch reposync errors (unreachable/invalid URL, invalid ref), print an actionable stderr error, exit non-zero, and confirm no destination-directory changes are made (FR-005)
- [X] T025 [US3] Handle cancellation (Ctrl+C) during clone/checkout/write in `internal/cli/init.go` and `internal/reposync/reposync.go`: ensure no partial files remain in a completed-looking state, report cancellation to stderr, exit non-zero (FR-007)

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T026 [P] Add doc comments to exported identifiers in `internal/reposync` and the extended parts of `internal/agentcatalog` per Constitution Principle I (Code Quality)
- [X] T027 Run `gofmt -l .`, `go vet ./...`, and the configured linter; fix any reported issues
- [X] T028 Run `go mod tidy` to finalize go.mod/go.sum
- [X] T029 Execute quickstart.md Scenarios 1-5 manually against the built binary and confirm expected behavior

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P2)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Adds the positional path argument on top of US1's fetch/write logic; independently testable once US1's fetch/write exists
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - Adds failure handling around US1's fetch/write logic; independently testable with fixtures once US1's fetch/write exists

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Fetch/checkout before write-to-destination
- Core implementation before error-handling integration
- Story complete before moving to next priority

### Parallel Opportunities

- T002 (rename to config.yaml) can run in parallel with T001 (add dependencies)
- T004 and T007 (Foundational) marked [P] can run in parallel
- All tests for a user story marked [P] can run in parallel
- Different user stories can be worked on in parallel by different team members once Foundational is complete

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together:
Task: "Table-driven test for LoadFS parsing a valid git: block into GitConfig in internal/agentcatalog/catalog_test.go"
Task: "Test that the internal/reposync fetch function clones a local fixture repository at a specific commit SHA in internal/reposync/reposync_test.go"
Task: "Test that init's happy path clones the configured repository/ref and writes files into the current working directory in internal/cli/init_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently (quickstart.md Scenario 1)
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!)
3. Add User Story 2 → Test independently → Deploy/Demo
4. Add User Story 3 → Test independently → Deploy/Demo
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1
   - Developer B: User Story 2
   - Developer C: User Story 3
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
