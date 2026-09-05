---

description: "Task list template for feature implementation"
---

# Tasks: Managed Files and Folders

**Input**: Design documents from `/specs/005-introduce-managed-files/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/cli-init.md](./contracts/cli-init.md), [quickstart.md](./quickstart.md)

**Tests**: Included and REQUIRED — this repository's constitution (Principle III, Testing
Standards, NON-NEGOTIABLE) mandates automated tests covering success and failure paths for every
behavior change before merge. Unlike feature 004, this feature requires real production code
changes (per [research.md](./research.md)): a new `managedFolders` parameter and pruning helper
in `internal/reposync`, a new `ManagedConfig` type in `internal/agentcatalog`, and new wiring in
`internal/cli/init.go`. Tests below are written to drive and then pin that new behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and
testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Single Go CLI project (existing structure from features 001-004 — no new project or directory is
introduced):

- `internal/agentcatalog/` — catalog parsing/validation
- `internal/reposync/` — clone/checkout/collect/filter/write/prune
- `internal/cli/` — `init` command wiring

---

## Phase 1: Setup

**Purpose**: Confirm a green baseline before making changes.

- [X] T001 Run `go build ./...` and `go test ./...` from the repo root to confirm the
      `004-folder-merge-overwrite` baseline (including its retroactive re-run-instruction change)
      is green before starting feature 005 changes

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add config-level support for declaring managed files/folders (FR-001), independent
of and blocking both user stories — per spec.md's own acceptance scenarios, both stories describe
managed folders as "declared in `config.yaml`."

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T002 [P] Add a `ManagedConfig` struct (`Files []string \`yaml:"files"\``,
      `Folders []string \`yaml:"folders"\``) and a `Managed ManagedConfig \`yaml:"managed"\`` field
      on `GitConfig` in internal/agentcatalog/catalog.go, with a doc comment explaining its
      purpose (FR-001)
- [X] T003 Extend path validation in internal/agentcatalog/catalog.go so `LoadFS` also validates
      `catalog.Git.Managed.Files` and `catalog.Git.Managed.Folders` using the existing
      `isValidDeclaredPath` check, returning `ErrInvalidDeclaredPath` wrapped with a
      managed-specific message (e.g. `managed file %q` / `managed folder %q`) on failure
- [X] T004 [P] Add `TestLoadFS_ManagedConfigParsesFromYAML` in
      internal/agentcatalog/catalog_test.go: load a fixture catalog whose `git.managed.files` and
      `git.managed.folders` are both populated; assert `catalog.Git.Managed.Files`/`Folders` match
      exactly what was declared
- [X] T005 [P] Add `TestLoadFS_AbsentManagedConfigIsBackwardCompatible` in
      internal/agentcatalog/catalog_test.go: load a fixture catalog with no `git.managed` section
      at all (matching features 001-004's existing fixtures); assert load succeeds and
      `catalog.Git.Managed.Files`/`Folders` are both empty
- [X] T006 [P] Add `TestLoadFS_InvalidManagedPathFails` (table-driven) in
      internal/agentcatalog/catalog_test.go covering an absolute path, an empty string, and a
      path containing a `..` segment, in both `managed.files` and `managed.folders`; assert
      `LoadFS` returns `ErrInvalidDeclaredPath` naming the offending managed path in each case
- [X] T007 [P] Run `go test ./internal/agentcatalog/... -v` and confirm T004–T006 pass

**Checkpoint**: `config.yaml` can declare managed files/folders and invalid entries are rejected
at load time. Neither `internal/reposync` nor `internal/cli` has been touched yet.

---

## Phase 3: User Story 1 - Managed folders exactly mirror the repository, including removing stale content (Priority: P1) 🎯 MVP

**Goal**: Prove that `reposync.Sync`, given a new `managedFolders` parameter, writes every file
the repository currently provides under that folder and removes every destination file under
that folder the repository does not provide — recursively, fail-fast on type conflicts without
deleting anything, and best-effort (no rollback) on partial failure — while leaving an agent's
own `declaredFolders` merge-only behavior (feature 004) completely unaffected.

**Independent Test**: Call `Sync` directly with a destination containing a stale file and a
still-provided file under a folder passed as `managedFolders`; verify the stale file is removed
and the provided file is written, without needing any `config.yaml` or CLI wiring.

### Tests for User Story 1

- [X] T008 [P] [US1] Add `TestSync_ManagedFolderPrunesStaleDestinationFile` in
      internal/reposync/reposync_test.go: pre-populate destination with `.highway/stale.md` (not
      provided by the fixture repo) and `.highway/keep.md` (provided, with different content);
      call `Sync` passing `.highway` as `managedFolders`; assert `.highway/stale.md` is removed
      and `.highway/keep.md` is overwritten with the repository's content (FR-003, Acceptance
      Scenario 1)
- [X] T009 [P] [US1] Add `TestSync_ManagedFolderMirrorsRecursively` in
      internal/reposync/reposync_test.go: pre-populate stale files at multiple nested depths
      under a managed folder; call `Sync`; assert every stale file at every depth is removed while
      every file the repository provides is written (FR-003, Acceptance Scenario 3)
- [X] T010 [P] [US1] Add `TestSync_ManagedFolderRemovedEntirelyWhenRepoProvidesNoFiles` in
      internal/reposync/reposync_test.go: pre-populate several files under a managed folder;
      configure the fixture repo to provide zero files under that folder; call `Sync`; assert
      every previously-existing file under the managed folder is removed (Acceptance Scenario 4)
- [X] T011 [P] [US1] Add `TestSync_ManagedFolderDoesNotAffectDeclaredFolderPruning` in
      internal/reposync/reposync_test.go: pre-populate a stale file under a folder passed only as
      `declaredFolders` (not `managedFolders`); call `Sync`; assert that stale file is **not**
      removed — a regression guard proving feature 004's merge-only guarantee for agent-declared
      folders is unaffected by this feature
- [X] T012 [P] [US1] Add `TestSync_ManagedFolderTypeConflictFileWhereDestinationHasDirectory` in
      internal/reposync/reposync_test.go: pre-create a directory at a destination path a managed
      folder's file needs to occupy; call `Sync`; assert an error naming that path, and the
      destination directory still exists afterward (FR-007)
- [X] T013 [P] [US1] Add `TestSync_ManagedFolderTypeConflictDirectoryWhereDestinationHasFile` in
      internal/reposync/reposync_test.go: pre-create a plain file at a destination path that
      needs to become a parent directory for a nested managed-folder entry; call `Sync`; assert an
      error naming that path, and the destination file still exists unchanged (FR-007)
- [X] T014 [P] [US1] Add `TestSync_ManagedFolderMatchingZeroFilesIsNotAnError` in
      internal/reposync/reposync_test.go: pass a `managedFolders` entry that matches zero files in
      the fixture repo (which still provides other content elsewhere, e.g. a plain declared file);
      call `Sync`; assert it succeeds without error — confirms managed folders are exempt from the
      missing-declared-path check (FR-008 revision; a managed folder matching nothing is a valid
      "mirror to nothing" outcome, not a configuration error; see research.md)
- [X] T015 [P] [US1] Add `TestSync_ManagedFolderWriteFailureLeavesPruningUnattempted` in
      internal/reposync/reposync_test.go: pre-populate a stale file under a managed folder
      alongside a type conflict elsewhere in that same managed folder's write set; call `Sync`;
      assert it returns an error and the stale file is still present afterward — proving pruning
      never runs when the write step itself fails (FR-006; confirms the write-then-prune ordering
      decided in [research.md](./research.md))

### Implementation for User Story 1

- [X] T016 [US1] In internal/reposync/reposync.go, add a `managedFolders []string` parameter to
      `Sync`; union it with `declaredFolders` for the existing `selectWriteSet` call only (so
      managed-folder content is written through the same existing code path) — do **not** union it
      into `checkDeclaredPathsExist`, since a managed folder matching zero repository files is a
      valid mirror-to-nothing outcome, not a missing-path error (depends on T008–T015 existing as
      failing tests)
- [X] T017 [US1] In internal/reposync/reposync.go, add
      `pruneStaleManagedFiles(destination string, collected map[string][]byte, managedFolders []string) error`
      using `filepath.WalkDir` to remove any destination file under each managed folder whose
      repository-relative path is not a key in `collected`; call it immediately after the write
      loop completes successfully, returning any error wrapped with the failing path (depends on
      T016)
- [X] T018 [US1] Update `Sync`'s doc comment in internal/reposync/reposync.go to describe the new
      `managedFolders` parameter and the mirror/prune guarantee it adds (depends on T017)
- [X] T019 [P] [US1] Run `go test ./internal/reposync/... -run TestSync -v` and confirm T008–T015
      all pass (depends on T016–T018)

**Checkpoint**: User Story 1 is independently complete and testable — `reposync.Sync` fully
implements mirror/prune/type-conflict/no-rollback semantics for managed folders, with no
dependency on `config.yaml` or CLI wiring.

---

## Phase 4: User Story 2 - Managed files and folders apply automatically on every `init` run, regardless of agent selection (Priority: P2)

**Goal**: Wire `catalog.Git.Managed` into `internal/cli/init.go` so managed files and folders are
always retrieved and applied in addition to whichever agent is selected during a given `init` run.

**Independent Test**: Run `runInit` twice against two destinations with a fixture catalog whose
`Managed.Files`/`Managed.Folders` are populated, selecting a different agent each time; verify the
fake sync stub receives the managed files/folders in both runs, alongside each run's respective
selected agent's own files/folders.

### Tests for User Story 2

- [X] T020 [P] [US2] Add `TestRunInit_ManagedFilesAndFoldersAppliedRegardlessOfSelectedAgent` in
      internal/cli/init_test.go: fixture catalog declares `Managed.Files`/`Managed.Folders` plus
      two agents with distinct declared files/folders; run `runInit` twice, selecting a different
      agent each time; assert the fake sync stub is invoked with the managed files/folders unioned
      with that run's selected agent's own files/folders in both cases (FR-002, FR-005; US2
      Acceptance Scenarios 1-2)
- [X] T021 [P] [US2] Add `TestRunInit_ManagedFilesUnionedWithAgentFilesNoDuplicateWrites` in
      internal/cli/init_test.go: fixture catalog's `Managed.Files` includes a path already present
      in the selected agent's own `Files`; run `runInit`; assert it succeeds and the fake sync
      stub receives that path (without erroring or panicking on the duplicate) — confirms the
      caller-side union tolerates overlapping declarations
- [X] T022 [P] [US2] Add `TestRunInit_NoManagedConfigStillSucceeds` in internal/cli/init_test.go:
      fixture catalog has no `Managed` section (zero value, matching features 001-004's existing
      fixtures); run `runInit`; assert it succeeds exactly as before this feature (backward
      compatibility)

### Implementation for User Story 2

- [X] T023 [US2] In internal/cli/init.go, extend `syncRepo`'s function type and its call to
      `reposync.Sync` to accept and forward a new `managedFolders []string` parameter (depends on
      Phase 3's `Sync` signature change, T016)
- [X] T024 [US2] In internal/cli/init.go's `runInit`, compute the union of the selected agent's
      `Files` and `catalog.Git.Managed.Files` (cloning the agent's slice before appending, to
      avoid mutating shared catalog state) and pass it as `syncRepo`'s `files` argument, alongside
      `catalog.Git.Managed.Folders` as the new `managedFolders` argument (depends on T023)
- [X] T025 [US2] Update `noopSync` and any other `syncRepo`-shaped test stubs in
      internal/cli/init_test.go to match the new signature (depends on T023)
- [X] T026 [P] [US2] Run `go test ./internal/cli/... -run TestRunInit -v` and confirm T020–T022,
      and every pre-existing `TestRunInit_*` test, all pass (depends on T024–T025)

**Checkpoint**: User Story 1 AND User Story 2 both pass — managed content mirrors correctly and
is applied on every `init` run regardless of the selected agent.

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Close out repo-wide validation.

- [X] T027 [P] Run `gofmt -l .` and `go vet ./...` across the repo; fix any formatting/vet issues
      surfaced by the new code and tests
- [X] T028 [P] Run `go test ./...` and confirm every package passes, including T004–T006 and
      T008–T022
- [X] T029 [P] Run `go mod tidy` and confirm `go.mod`/`go.sum` are unchanged (no new third-party
      dependency is required by this feature, per [research.md](./research.md))
- [ ] T030 Manually run the [quickstart.md](./quickstart.md) validation scenarios against a local
      build (`go build -o bin/highway ./cmd/highway`) using the production
      internal/agentcatalog/config.yaml's existing `git.managed.files`/`folders` declarations
      (`file1.md`, `.highway`) to confirm end-to-end behavior

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories (both
  stories' acceptance scenarios describe managed folders as declared in `config.yaml`).
- **User Story 1 (Phase 3)**: Depends on Foundational completion. No dependency on US2 — tests
  and implementation operate entirely within `internal/reposync`, independent of `config.yaml`
  parsing or CLI wiring.
- **User Story 2 (Phase 4)**: Depends on Foundational completion AND on User Story 1's `Sync`
  signature change (T016) being in place, since `internal/cli/init.go` must forward the new
  `managedFolders` parameter. Not independent of US1 in implementation order, but independently
  testable once both are complete.
- **Polish (Phase 5)**: Depends on User Story 1 and User Story 2 completion.

### User Story Dependencies

- **User Story 1 (P1)**: No dependencies on other stories — MVP.
- **User Story 2 (P2)**: Depends on User Story 1's `reposync.Sync` signature change to exist
  before it can be wired into `internal/cli/init.go`; otherwise independently testable.

### Within Each User Story

- All test tasks within a story are marked `[P]` — they add independent test functions to the
  same file but do not depend on each other's completion, only on the story's implementation
  tasks (or, for Foundational, on Phase 2's config support).

---

## Parallel Execution Examples

```text
# Phase 2 (Foundational) — struct addition and both new tests can be authored in parallel;
# T003 (validation) should land before T006 (which exercises it), but can be written alongside:
T002, T004, T005, T006

# Phase 3 (User Story 1) — all eight tests can be authored in parallel before implementation:
T008, T009, T010, T011, T012, T013, T014, T015

# Phase 4 (User Story 2) — all three tests can be authored in parallel before implementation:
T020, T021, T022

# Phase 5 (Polish) — independent in principle, though practically sequential since each
# depends on the previous step's cleanliness:
T027, T028, T029
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational — `config.yaml` can declare managed files/folders
3. Complete Phase 3: User Story 1 — T008-T019
4. **STOP and VALIDATE**: Run `go test ./internal/reposync/... -run TestSync -v` and confirm
   T008-T015 pass against the new `managedFolders` parameter and pruning helper
5. This is a deployable/demonstrable increment: `reposync.Sync` can now fully mirror a declared
   managed folder, independent of any CLI wiring

### Incremental Delivery

1. Add User Story 2 (T020-T026) → wires managed config into `init` so it applies automatically,
   regardless of agent selection
2. Add Polish (T027-T030) → closes out repo-wide validation (fmt/vet/test/mod tidy/quickstart)
3. Each phase leaves the repository in a fully green, independently valuable state
