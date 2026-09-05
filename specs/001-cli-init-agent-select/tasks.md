---

description: "Task list template for feature implementation"
---

# Tasks: CLI Init Agent Selection

**Input**: Design documents from `/specs/001-cli-init-agent-select/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli-init.md, quickstart.md

**Tests**: Included. Constitution Principle III (Testing Standards) is NON-NEGOTIABLE, requiring
automated tests for every behavior change; the plan's Project Structure already designates
`catalog_test.go` and `init_test.go` as required files.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing
of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2)
- Paths below follow plan.md's Project Structure (single Go project: `cmd/highway/`, `internal/`)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Initialize the Go module and create the directory skeleton (`cmd/highway/`, `internal/agentcatalog/`, `internal/cli/`) per plan.md Project Structure
- [X] T002 Add and fetch dependencies in go.mod: `github.com/spf13/cobra`, `gopkg.in/yaml.v3`, `github.com/charmbracelet/huh`, `golang.org/x/term`
- [X] T003 [P] Add golangci-lint configuration (`.golangci.yml`) enforcing zero-warning `gofmt`/`go vet`-equivalent checks per Constitution Principle I (Code Quality)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Create `internal/agentcatalog/agents.yaml` with the initial catalog entries (`vscode`/"GitHub Copilot", `claude-code`/"Claude Code", `cursor`/"Cursor") per data-model.md
- [X] T005 [P] Define `AgentDefinition` (`id`, `display_name`) and `AgentCatalog` (`agents []AgentDefinition`) structs with yaml struct tags in `internal/agentcatalog/catalog.go`
- [X] T006 Embed `agents.yaml` via `go:embed` into an `embed.FS` in `internal/agentcatalog/catalog.go`, and implement `LoadFS(fsys fs.FS, name string) (*AgentCatalog, error)` plus a `Load()` wrapper over the embedded FS, so tests can inject fixture catalogs without touching the real embedded file (per quickstart.md Scenario 2)
- [X] T007 Implement parsing and validation inside `LoadFS`: non-empty `id`/`display_name` required, first-occurrence-wins de-duplication by `id` (FR-006–FR-009)
- [X] T008 [P] Create `cmd/highway/main.go` entrypoint that constructs and executes the cobra root command
- [X] T009 [P] Create `internal/cli/root.go` defining the cobra root command

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - First-time agent selection (Priority: P1) 🎯 MVP

**Goal**: A developer runs `init`, sees every supported agent from the embedded catalog as a
selectable list, picks one, and sees a confirmation naming that agent. Nothing is persisted.

**Independent Test**: Run `init` in an interactive terminal against the embedded catalog, choose
an agent from the presented list, and verify the tool prints a confirmation naming that agent
(quickstart.md Scenario 1).

### Tests for User Story 1 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T010 [P] [US1] Table-driven test for `LoadFS` success path (valid multi-agent fixture, including a duplicate `id` case) in `internal/agentcatalog/catalog_test.go`
- [X] T011 [P] [US1] Test that `init` exits non-zero with an actionable stderr error and shows no prompt when stdin is not a TTY, in `internal/cli/init_test.go`
- [X] T012 [P] [US1] Test that `init`'s happy path (selecting an agent) prints a confirmation naming the selected `display_name` and exits 0, in `internal/cli/init_test.go`

### Implementation for User Story 1

- [X] T013 [US1] Implement `internal/cli/init.go`: cobra `init` command skeleton (FR-001)
- [X] T014 [US1] Add TTY detection using `golang.org/x/term.IsTerminal` before prompting; print an actionable stderr error and exit non-zero if stdin is not a TTY (FR-010)
- [X] T015 [US1] Load the catalog via `agentcatalog.Load()` and integrate `huh`'s `Select` field to present the de-duplicated `display_name` list (FR-002, FR-003, FR-004)
- [X] T016 [US1] Handle prompt cancellation (Ctrl+C): show no confirmation message, exit non-zero
- [X] T017 [US1] Print a confirmation message naming the selected agent's `display_name` and exit 0 (FR-005); confirm no selection is written to disk anywhere (FR-011)
- [X] T018 [US1] Register the `init` command on the cobra root command in `internal/cli/root.go`

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently
(quickstart.md Scenarios 1, 3, and 4)

---

## Phase 4: User Story 2 - Handling a missing or invalid configuration file (Priority: P2)

**Goal**: When the embedded catalog is missing, empty, or malformed, `init` reports a clear,
actionable error instead of crashing or silently proceeding to selection.

**Independent Test**: Point the loader at fixture files that are missing, empty (`agents: []` or
no `agents` key), and malformed YAML, and verify `init` exits non-zero with an actionable stderr
error and never shows a prompt (quickstart.md Scenario 2).

### Tests for User Story 2 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T019 [P] [US2] Table-driven tests for `LoadFS` error paths: missing file, empty/absent `agents` list, malformed YAML syntax, in `internal/agentcatalog/catalog_test.go`
- [X] T020 [P] [US2] Test that `init` surfaces catalog load errors to stderr with a non-zero exit and shows no prompt, in `internal/cli/init_test.go`

### Implementation for User Story 2

- [X] T021 [US2] Add distinct, actionable error messages in `LoadFS` for a missing file (FR-006), malformed YAML (FR-007), and an empty agent list (FR-008)
- [X] T022 [US2] In `internal/cli/init.go`, catch catalog load errors before invoking the prompt, print the actionable error to stderr, and exit non-zero

**Checkpoint**: All user stories should now be independently functional

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T023 [P] Add doc comments to exported identifiers in `internal/agentcatalog` and `internal/cli` per Constitution Principle I (Code Quality)
- [X] T024 Run `gofmt -l .`, `go vet ./...`, and the configured linter; fix any reported issues
- [X] T025 Run `go mod tidy` to finalize go.mod/go.sum
- [X] T026 Execute quickstart.md Scenarios 1-4 manually against the built binary and confirm expected behavior

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends only on Foundational completion
- **User Story 2 (Phase 4)**: Depends only on Foundational completion; independent of US1 (both read the same catalog loader, but exercise different code paths)
- **Polish (Phase 5)**: Depends on both user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - No dependencies on US1; independently testable via fixture catalogs

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- `catalog.go`/`catalog_test.go` changes before `init.go`/`init_test.go` changes that depend on them
- Story complete before moving to the next priority (if working sequentially)

### Parallel Opportunities

- T003 (lint config) can run in parallel with T001/T002
- T005, T008, T009 can run in parallel with each other (different files) once T004 exists
- All three US1 test tasks (T010-T012) can run in parallel
- Both US2 test tasks (T019-T020) can run in parallel
- Once Foundational (Phase 2) completes, User Story 1 and User Story 2 can be implemented in parallel by different contributors

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together:
Task: "Table-driven test for LoadFS success path in internal/agentcatalog/catalog_test.go"
Task: "Test that init exits non-zero with no prompt when stdin is not a TTY in internal/cli/init_test.go"
Task: "Test that init's happy path prints a confirmation and exits 0 in internal/cli/init_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Run quickstart.md Scenarios 1, 3, and 4 independently
5. Ship the MVP: developers can select an agent from the embedded catalog

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → MVP ready
3. Add User Story 2 → Test independently → Full feature complete (quickstart.md Scenario 2 passes)
4. Finish with Polish phase (lint/vet/tidy + full quickstart re-run)

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (interactive selection + confirmation)
   - Developer B: User Story 2 (catalog error handling)
3. Stories integrate independently since both only touch `internal/cli/init.go` and `internal/agentcatalog/catalog.go` in non-overlapping ways (selection UX vs. error paths)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story (US1, US2) for traceability
- Verify tests fail before implementing the corresponding behavior
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
