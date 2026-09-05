---

description: "Task list template for feature implementation"
---

# Tasks: Merge Declared Folders Into Destination

**Input**: Design documents from `/specs/004-folder-merge-overwrite/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/cli-init.md](./contracts/cli-init.md), [quickstart.md](./quickstart.md)

**Tests**: Included and REQUIRED — this repository's constitution (Principle III, Testing
Standards, NON-NEGOTIABLE) mandates automated tests covering success and failure paths for every
behavior change before merge. Per [research.md](./research.md), no production code changes are
required for this feature: every test below is expected to **pass immediately** against the
existing `internal/reposync/reposync.go`, because it already implements the merge/no-delete/
fail-fast semantics this feature specifies. These tests exist to pin that behavior against
regression, not to drive new implementation.

**Organization**: Tasks are grouped by user story to enable independent implementation and
testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Single Go CLI project (existing structure from features 001/002/003 — no new project or
directory is introduced):

- `internal/reposync/` — clone/checkout/collect/filter/write; the only package touched

---

## Phase 1: Setup

**Purpose**: Confirm a green baseline before making changes; no new project scaffolding is needed
since this feature only adds tests to an existing package.

- [X] T001 Run `go build ./...` and `go test ./...` from the repo root to confirm the
      `003-selective-file-retrieval` baseline is green before starting feature 004 changes

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Confirm the research decision before writing tests against it.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T002 Re-read the write loop, `selectWriteSet`, and `checkDeclaredPathsExist` in
      internal/reposync/reposync.go alongside [research.md](./research.md) and confirm no
      production code changes are needed before writing any test in this feature — if a gap is
      found, stop and update research.md/plan.md before proceeding to Phase 3

**Checkpoint**: Foundation confirmed — proceed to User Story 1. No code changes expected from
this phase.

---

## Phase 3: User Story 1 - Retrieved folder contents overwrite only their own paths at the destination (Priority: P1) 🎯 MVP

**Goal**: Prove that when the selected agent's declared folder is written, only the paths the
retrieved repository currently provides are created/overwritten, and any other pre-existing
destination content — whether under the same declared folder or elsewhere — is left untouched.

**Independent Test**: Populate a destination with two entries under a declared folder where the
repository only provides one; run `Sync`; verify the provided entry is overwritten and the other
is byte-for-byte unchanged.

### Tests for User Story 1

- [X] T003 [P] [US1] Add `TestSync_MergePreservesDestinationFileNotProvidedByRepoFolder` in
      internal/reposync/reposync_test.go: pre-populate destination with
      `.github/skills/highway-activities/file.md` and
      `.github/skills/speckit-specify/file.md`; configure the fixture repository's declared
      `.github/skills` folder to contain only `highway-activities/file.md` with new content; run
      `Sync`; assert `highway-activities/file.md` is overwritten with the repository's content and
      `speckit-specify/file.md` is byte-for-byte unchanged (Acceptance Scenario 1)
- [X] T004 [P] [US1] Add `TestSync_MergeLeavesUnrelatedDestinationFileUntouched` in
      internal/reposync/reposync_test.go: pre-populate destination with a file at a path outside
      any of the selected agent's declared files/folders; run `Sync` with a non-empty declared
      folder that doesn't cover that path; assert the unrelated file is byte-for-byte unchanged
      (Acceptance Scenario 3)

### Implementation for User Story 1

- No implementation tasks — per [research.md](./research.md), `selectWriteSet`'s existing
  write-set-only-write loop already satisfies this story. Run T003–T004 and confirm both pass
  without modifying internal/reposync/reposync.go.

**Checkpoint**: User Story 1 is independently testable and passing — the core merge guarantee is
pinned by regression tests.

---

## Phase 4: User Story 2 - Nested subfolders merge the same way (Priority: P2)

**Goal**: Prove the same merge guarantee holds recursively: a pre-existing sibling subfolder (or
pre-existing file within a shared subfolder) that the repository doesn't currently provide is
left untouched, even while a neighboring file in the same declared folder tree is written.

**Independent Test**: Populate a destination with two subfolders under a declared folder where
the repository's current version of that folder only contains a new file in one of them; run
`Sync`; verify the new file is written and both pre-existing subfolders' files are unchanged.

### Tests for User Story 2

- [X] T005 [P] [US2] Add `TestSync_MergePreservesNestedSiblingSubfoldersAndFiles` in
      internal/reposync/reposync_test.go: pre-populate destination with
      `.github/skills/a/existing.md` and `.github/skills/b/existing.md`; configure the fixture
      repository's declared `.github/skills` folder to contain only `.github/skills/a/new.md`;
      run `Sync`; assert `.github/skills/a/new.md` is written, and both
      `.github/skills/a/existing.md` and `.github/skills/b/existing.md` are byte-for-byte
      unchanged (Acceptance Scenario 1)

### Implementation for User Story 2

- No implementation tasks — recursion falls out of `selectWriteSet`'s path-prefix matching
  against the full collected file map; no per-depth special-casing exists to break. Run T005 and
  confirm it passes without modifying internal/reposync/reposync.go.

**Checkpoint**: User Stories 1 AND 2 both pass — the merge guarantee is pinned at the top level
and recursively at every nesting depth.

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Pin the two clarified edge-case behaviors (type conflicts, partial-failure/no-
rollback) that apply across both user stories, and close out repo-wide validation.

- [X] T006 [P] Add `TestSync_TypeConflictFileWhereDestinationHasDirectory` in
      internal/reposync/reposync_test.go: pre-create a directory at the destination path that a
      declared `files` entry needs to occupy as a plain file; run `Sync`; assert it returns an
      error naming that path, and the destination directory still exists afterward (FR-007)
- [X] T007 [P] Add `TestSync_TypeConflictDirectoryWhereDestinationHasFile` in
      internal/reposync/reposync_test.go: pre-create a plain file at a destination path that
      needs to become a parent directory for a nested declared-folder entry; run `Sync`; assert it
      returns an error naming that path, and the destination file still exists afterward (FR-007)
- [X] T008 [P] Add `TestSync_FailedRunDoesNotRevertFilesFromEarlierSuccessfulRun` in
      internal/reposync/reposync_test.go: run `Sync` once successfully so
      `.github/skills/keep/a.md` is written; then run `Sync` again against the same destination
      with an additional declared path that triggers a type conflict (per T006/T007); assert the
      second (failing) run returns an error, and `.github/skills/keep/a.md` from the first run
      still exists with its original content — a failure does not revert previously-written
      content (FR-006)
- [X] T009 [P] Run `gofmt -l .` and `go vet ./...` across the repo; fix any formatting/vet issues
      surfaced by the new tests
- [X] T010 [P] Run `go test ./...` and confirm every package passes, including T003–T008
- [X] T011 [P] Run `go mod tidy` and confirm `go.mod`/`go.sum` are unchanged (no new third-party
      dependency is required by this feature)
- [ ] T012 Manually run the [quickstart.md](./quickstart.md) validation scenarios against a local
      build (`go build -o bin/highway ./cmd/highway`) using the production
      internal/agentcatalog/config.yaml declared folders to confirm end-to-end behavior
- [X] T013 [P] Retroactive addition (FR-008): in internal/cli/init.go, print "Re-run the command
      to retry." to stderr whenever `syncRepo` returns an error; update
      `TestRunInit_SyncError`, `TestRunInit_MissingDeclaredPathSurfacesToStderr`, and
      `TestRunInit_SyncCancellation` in internal/cli/init_test.go to assert this instruction is
      present

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories (confirms no
  code changes are needed before tests are written against current behavior).
- **User Story 1 (Phase 3)**: Depends on Foundational completion. No dependency on US2.
- **User Story 2 (Phase 4)**: Depends on Foundational completion. Independent of US1 — both
  exercise the same unmodified `selectWriteSet` logic from different angles.
- **Polish (Phase 5)**: Depends on User Story 1 and User Story 2 completion (T006–T008 reuse the
  same fixture-repository helpers established by T003–T005).

### User Story Dependencies

- **User Story 1 (P1)**: No dependencies on other stories — MVP.
- **User Story 2 (P2)**: No dependencies on User Story 1; both are independently testable and
  could be implemented in either order or in parallel.

### Within Each User Story

- All test tasks within a story are marked `[P]` — they add independent test functions to the
  same file but do not depend on each other's completion, only on Phase 2's confirmation.

---

## Parallel Execution Examples

```text
# Phase 3 (User Story 1) — both tests can be authored in parallel:
T003, T004

# Phase 4 (User Story 2) — single test, no parallel opportunity within the phase:
T005

# Phase 5 (Polish) — the three new regression tests can be authored in parallel;
# the four validation tasks (gofmt/vet, full test run, mod tidy, quickstart) are
# sequential in practice since each depends on the previous step's cleanliness,
# but are independent in principle:
T006, T007, T008
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (confirm no code changes needed)
3. Complete Phase 3: User Story 1 — T003, T004
4. **STOP and VALIDATE**: Run `go test ./internal/reposync/...` and confirm T003–T004 pass
   against the unmodified implementation
5. This is a deployable/demonstrable increment: the core "re-running `init` doesn't destroy
   unrelated declared-folder content" guarantee is now regression-tested

### Incremental Delivery

1. Add User Story 2 (T005) → confirms the guarantee holds recursively
2. Add Polish (T006–T012) → confirms the two clarified edge cases (type conflicts, no-rollback)
   and closes out repo-wide validation (fmt/vet/test/mod tidy/quickstart)
3. Each phase leaves the repository in a fully green, independently valuable state
