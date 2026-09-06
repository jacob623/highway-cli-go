---

description: "Task list template for feature implementation"
---

# Tasks: Seed Files and Folders

**Input**: Design documents from `/specs/006-seed-files-folders/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/cli-init.md](./contracts/cli-init.md), [quickstart.md](./quickstart.md)

**Tests**: Included and REQUIRED — this repository's constitution (Principle III, Testing
Standards, NON-NEGOTIABLE) mandates automated tests covering success and failure paths for every
behavior change before merge. Per [research.md](./research.md), this feature requires real
production code changes: a new `seededFiles`/`seededFolders` parameter pair and a new
`selectSeedWriteSet` function in `internal/reposync`, a new `SeededConfig` type in
`internal/agentcatalog`, and new wiring in `internal/cli/init.go`. Tests below are written to
drive and then pin that new behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and
testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Single Go CLI project (existing structure from features 001-005 — no new project or directory is
introduced):

- `internal/agentcatalog/` — catalog parsing/validation
- `internal/reposync/` — clone/checkout/collect/filter/write/prune
- `internal/cli/` — `init` command wiring

---

## Phase 1: Setup

**Purpose**: Confirm a green baseline before making changes.

- [X] T001 Run `go build ./...` and `go test ./...` from the repo root to confirm the
      `005-introduce-managed-files` baseline is green before starting feature 006 changes

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add config-level support for declaring seeded files/folders (FR-001), independent of
and blocking all three user stories — every acceptance scenario in spec.md describes seeded
content as "declared in `config.yaml`."

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T002 [P] Add a `SeededConfig` struct (`Files []string \`yaml:"files"\``,
      `Folders []string \`yaml:"folders"\``) and a `Seeded SeededConfig \`yaml:"seeded"\`` field on
      `GitConfig` in internal/agentcatalog/catalog.go, with a doc comment explaining its
      create-if-missing purpose (FR-001)
- [X] T003 Extend path validation in internal/agentcatalog/catalog.go so `LoadFS` also validates
      `catalog.Git.Seeded.Files` and `catalog.Git.Seeded.Folders` using the existing
      `isValidDeclaredPath` check, returning `ErrInvalidDeclaredPath` wrapped with a
      seeded-specific message (e.g. `seeded file %q` / `seeded folder %q`) on failure
- [X] T004 [P] Add `TestLoadFS_SeededConfigParsesFromYAML` in
      internal/agentcatalog/catalog_test.go: load a fixture catalog whose `git.seeded.files` and
      `git.seeded.folders` are both populated; assert `catalog.Git.Seeded.Files`/`Folders` match
      exactly what was declared
- [X] T005 [P] Add `TestLoadFS_AbsentSeededConfigIsBackwardCompatible` in
      internal/agentcatalog/catalog_test.go: load a fixture catalog with no `git.seeded` section
      at all (matching features 001-005's existing fixtures); assert load succeeds and
      `catalog.Git.Seeded.Files`/`Folders` are both empty
- [X] T006 [P] Add `TestLoadFS_InvalidSeededPathFails` (table-driven) in
      internal/agentcatalog/catalog_test.go covering an absolute path, an empty string, and a
      path containing a `..` segment, in both `seeded.files` and `seeded.folders`; assert
      `LoadFS` returns `ErrInvalidDeclaredPath` naming the offending seeded path in each case
- [X] T007 [P] Run `go test ./internal/agentcatalog/... -v` and confirm T004–T006 pass

**Checkpoint**: `config.yaml` can declare seeded files/folders and invalid entries are rejected at
load time. Neither `internal/reposync` nor `internal/cli` has been touched yet.

---

## Phase 3: User Story 1 - Scaffold a new project from declared seed content (Priority: P1) 🎯 MVP

**Goal**: Prove that `reposync.Sync`, given new `seededFiles`/`seededFolders` parameters, creates
every declared seed file and every file the repository currently provides under a declared seed
folder when nothing already exists at the destination — recursively for folders — and that this
happens on every `init` run regardless of which agent is selected. A declared seed path matching
nothing in the repository fails validation before any write, and a partial write failure does not
roll back seed files already written earlier in the same run.

**Independent Test**: Call `Sync` directly with an empty destination, passing seeded files/folders;
verify every declared seed path is created with the repository's content. Also run `runInit`
twice against two destinations selecting a different agent each time; verify the fake sync stub
receives the same seeded files/folders in both runs.

### Tests for User Story 1

- [X] T008 [P] [US1] Add `TestSync_SeededFileCreatedWhenDestinationEmpty` in
      internal/reposync/reposync_test.go: call `Sync` against an empty destination with a
      `seededFiles` entry the fixture repo provides; assert the file is created with the
      repository's content (FR-003, US1 Acceptance Scenario 1)
- [X] T009 [P] [US1] Add `TestSync_SeededFolderFilesCreatedRecursively` in
      internal/reposync/reposync_test.go: call `Sync` against an empty destination with a
      `seededFolders` entry whose fixture repo content is nested at multiple depths; assert every
      nested file is created (FR-004, US1 Acceptance Scenario 2)
- [X] T010 [P] [US1] Add `TestSync_SeededMissingDeclaredPathFails` in
      internal/reposync/reposync_test.go: pass a `seededFiles` or `seededFolders` entry that
      matches zero files in the fixture repo; call `Sync`; assert an error naming that path and
      that no files were written (FR-007 — unlike managed folders, a seeded folder matching
      nothing is always a configuration error, since seed content has no legitimate
      "matches nothing" outcome; see research.md)
- [X] T011 [P] [US1] Add `TestSync_SeededWriteFailureDoesNotRollbackAlreadyWrittenSeedFiles` in
      internal/reposync/reposync_test.go: configure a seeded write set with one path that
      succeeds in an earlier run and one path that hits an intermediate path-segment conflict
      (a plain file occupies a directory a later seeded path needs) in a later run; call `Sync`;
      assert the later run returns an error and the earlier run's seed file is still present
      afterward (edge case: partial write failure has no rollback)
- [X] T012 [P] [US1] Add `TestRunInit_SeededFilesAndFoldersAppliedRegardlessOfSelectedAgent` in
      internal/cli/init_test.go: fixture catalog declares `Seeded.Files`/`Seeded.Folders` plus two
      agents with distinct declared files/folders; run `runInit` twice, selecting a different
      agent each time; assert the fake sync stub is invoked with the same seeded files/folders in
      both cases (FR-002, FR-006; US1 Acceptance Scenario 3)

### Implementation for User Story 1

- [X] T013 [US1] In internal/reposync/reposync.go, add `seededFiles, seededFolders []string`
      parameters to `Sync`; extend the existing `checkDeclaredPathsExist` call to also validate
      `declaredFiles ∪ seededFiles` and `declaredFolders ∪ seededFolders` (a missing seeded path
      is a validation error, unlike managed folders) (depends on T008–T011 existing as failing
      tests)
- [X] T014 [US1] In internal/reposync/reposync.go, add
      `selectSeedWriteSet(destination string, collected map[string][]byte, seededFiles, seededFolders []string) (map[string][]byte, error)`
      that matches candidates the same way `selectWriteSet` does but additionally calls
      `os.Lstat` on each candidate's destination path, including it only when
      `os.IsNotExist(err)` is true; merge its result into the existing write-set map before the
      single existing write loop runs (depends on T013)
- [X] T015 [US1] Update `Sync`'s doc comment in internal/reposync/reposync.go to describe the new
      `seededFiles`/`seededFolders` parameters and the create-if-missing guarantee they add
      (depends on T014)
- [X] T016 [P] [US1] Run `go test ./internal/reposync/... -run TestSync -v` and confirm
      T008–T011 all pass (depends on T013–T015)
- [X] T017 [US1] In internal/cli/init.go, extend `syncRepo`'s function type and its call to
      `reposync.Sync` to accept and forward new `seededFiles, seededFolders []string` parameters,
      kept as their own arguments rather than unioned into the agent's `files`/`folders` (depends
      on T014)
- [X] T018 [US1] In internal/cli/init.go's `runInit`, pass `catalog.Git.Seeded.Files` and
      `catalog.Git.Seeded.Folders` as `syncRepo`'s new `seededFiles`/`seededFolders` arguments
      (depends on T017)
- [X] T019 [US1] Update `noopSync` and any other `syncRepo`-shaped test stubs in
      internal/cli/init_test.go to match the new signature (depends on T017)
- [X] T020 [P] [US1] Run `go test ./internal/cli/... -run TestRunInit -v` and confirm T012, and
      every pre-existing `TestRunInit_*` test, all pass (depends on T018–T019)

**Checkpoint**: User Story 1 is independently complete and testable — seeded content is created
on a fresh destination, recursively for folders, regardless of the selected agent.

---

## Phase 4: User Story 2 - Local customizations always survive a re-run (Priority: P2)

**Goal**: Prove that a seeded file or a file under a seeded folder that already exists at the
destination is left completely unchanged on any later `init` run, even when the repository's
current content at that path differs — with no error reported.

**Independent Test**: Call `Sync` with a destination pre-populated with content at a declared seed
path that differs from the fixture repo's version; verify the destination content is unchanged
afterward and `Sync` returns no error.

### Tests for User Story 2

- [X] T021 [P] [US2] Add `TestSync_SeededFileWithExistingContentLeftUnchanged` in
      internal/reposync/reposync_test.go: pre-populate a seeded file's destination path with
      content that differs from the fixture repo's version; call `Sync`; assert the destination
      content is byte-for-byte unchanged and no error is returned (FR-005, US2 Acceptance
      Scenario 1)
- [X] T022 [P] [US2] Add `TestSync_SeededFolderExistingFileLeftUnchangedEvenWhenRepoContentDiffers`
      in internal/reposync/reposync_test.go: pre-populate a file under a seeded folder with
      content that differs from what the fixture repo currently provides at that path; call
      `Sync`; assert the destination content is unchanged (FR-005, US2 Acceptance Scenario 2)
- [X] T023 [P] [US2] Add `TestSync_SeededPathOccupiedByDirectoryIsSkippedNotError` in
      internal/reposync/reposync_test.go: pre-create a directory at the exact destination path a
      seeded file needs to occupy; call `Sync`; assert it succeeds, the directory is left
      unchanged, and no seed file is written there (edge case: a type conflict at a seeded path is
      a silent skip, not an error, unlike managed folders)
- [X] T024 [P] [US2] Run `go test ./internal/reposync/... -run TestSync -v` and confirm
      T021–T023 pass (depends on T014, T021–T023 existing as tests)

**Checkpoint**: User Story 1 AND User Story 2 both pass — seeded content is created fresh and
never disturbs anything already present at its destination path.

---

## Phase 5: User Story 3 - Newly added seed files are adopted without disturbing existing ones (Priority: P3)

**Goal**: Prove that when the repository's current content under a declared seed folder includes
a file not yet present at the destination, alongside a file that is already present (and
customized), only the missing file is created — the already-present file is left untouched.

**Independent Test**: Call `Sync` with a destination that already has one file under a seeded
folder, while the fixture repo provides that same file (with different content) plus one
additional file not yet at the destination; verify only the additional file is created and the
existing file is unchanged.

### Tests for User Story 3

- [X] T025 [P] [US3] Add `TestSync_SeededFolderAdoptsNewlyAddedFileWithoutDisturbingExisting` in
      internal/reposync/reposync_test.go: pre-populate `custom/existing.md` with customized
      content; configure the fixture repo to provide `custom/existing.md` (different content) and
      `custom/new.md` (not yet at the destination); call `Sync`; assert `custom/new.md` is created
      with the repository's content and `custom/existing.md` remains unchanged (FR-004, US3
      Acceptance Scenario 1)
- [X] T026 [P] [US3] Run `go test ./internal/reposync/... -run TestSync -v` and confirm T025
      passes (depends on T014, T025 existing as a test)

**Checkpoint**: All three user stories pass — seed content scaffolds a fresh destination,
survives every re-run untouched, and adopts newly added upstream files incrementally.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Close out repo-wide validation.

- [X] T027 [P] Run `gofmt -l .` and `go vet ./...` across the repo; fix any formatting/vet issues
      surfaced by the new code and tests
- [X] T028 [P] Run `go test ./...` and confirm every package passes, including T004–T006 and
      T008–T025
- [X] T029 [P] Run `go mod tidy` and confirm `go.mod`/`go.sum` are unchanged (no new third-party
      dependency is required by this feature, per [research.md](./research.md))
- [ ] T030 Manually run the [quickstart.md](./quickstart.md) validation scenarios against a local
      build (`go build -o bin/highway ./cmd/highway`) using the production
      internal/agentcatalog/config.yaml's existing `git.seeded.files`/`folders` declarations
      (`architecture.yaml`, `custom`) to confirm end-to-end behavior

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories (every
  story's acceptance scenarios describe seeded content as declared in `config.yaml`).
- **User Story 1 (Phase 3)**: Depends on Foundational completion. Introduces the
  `seededFiles`/`seededFolders` parameters, `selectSeedWriteSet`, and the CLI wiring that makes
  seeded content apply regardless of agent — the MVP.
- **User Story 2 (Phase 4)**: Depends on Foundational completion AND on User Story 1's
  `selectSeedWriteSet` (T014) existing, since the create-if-missing mechanism it tests is
  implemented there; adds no new production code, only tests locking in the preservation
  guarantee.
- **User Story 3 (Phase 5)**: Same dependency as User Story 2 — depends on T014; adds no new
  production code, only a test locking in incremental adoption.
- **Polish (Phase 6)**: Depends on User Story 1, 2, and 3 completion.

### User Story Dependencies

- **User Story 1 (P1)**: No dependencies on other stories — MVP.
- **User Story 2 (P2)**: Depends on User Story 1's `selectSeedWriteSet` (T014) existing before its
  tests can pass; otherwise independently testable and adds no implementation of its own.
- **User Story 3 (P3)**: Same as User Story 2 — depends on T014, independently testable, adds no
  implementation of its own.

### Within Each User Story

- All test tasks within a story are marked `[P]` — they add independent test functions to the
  same file but do not depend on each other's completion, only on the relevant implementation
  tasks (or, for Foundational, on Phase 2's config support).

---

## Parallel Execution Examples

```text
# Phase 2 (Foundational) — struct addition and both new tests can be authored in parallel;
# T003 (validation) should land before T006 (which exercises it), but can be written alongside:
T002, T004, T005, T006

# Phase 3 (User Story 1) — all five tests can be authored in parallel before implementation:
T008, T009, T010, T011, T012

# Phase 4 (User Story 2) — all three tests can be authored in parallel:
T021, T022, T023

# Phase 5 (User Story 3) — single test, no parallelism needed:
T025

# Phase 6 (Polish) — independent in principle, though practically sequential since each
# depends on the previous step's cleanliness:
T027, T028, T029
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational — `config.yaml` can declare seeded files/folders
3. Complete Phase 3: User Story 1 — T008-T020
4. **STOP and VALIDATE**: Run `go test ./internal/reposync/... -run TestSync -v` and
   `go test ./internal/cli/... -run TestRunInit -v`, confirming T008-T012 pass against the new
   `seededFiles`/`seededFolders` parameters, `selectSeedWriteSet`, and CLI wiring
5. This is a deployable/demonstrable increment: `init` now scaffolds declared seed content on a
   fresh destination, regardless of selected agent

### Incremental Delivery

1. Add User Story 2 (T021-T024) → locks in the never-overwrite guarantee with dedicated tests,
   no new production code required
2. Add User Story 3 (T025-T026) → locks in incremental adoption of newly added upstream seed
   files with a dedicated test, no new production code required
3. Add Polish (T027-T030) → closes out repo-wide validation (fmt/vet/test/mod tidy/quickstart)
4. Each phase leaves the repository in a fully green, independently valuable state
