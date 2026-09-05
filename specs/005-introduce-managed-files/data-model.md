# Phase 1 Data Model: Managed Files and Folders

## Entities

### ManagedConfig (new)

Represents the `git.managed` section of `config.yaml`: files and folders that apply on every
`init` run, independent of agent selection.

| Field   | Type       | YAML key  | Notes                                                             |
|---------|------------|-----------|--------------------------------------------------------------------|
| Files   | `[]string` | `files`   | Exact repository-relative paths; create-or-overwrite only, never deleted (FR-004). |
| Folders | `[]string` | `folders` | Repository-relative folder path prefixes; mirrored exactly, including deletion of stale destination files (FR-003). |

**Validation rules** (FR-001, reusing feature 003's `isValidDeclaredPath`):
- Each entry MUST be non-empty, relative (no leading `/`), and MUST NOT contain a `..` segment.
- An invalid entry fails catalog load with `ErrInvalidDeclaredPath`, naming the offending managed
  file or folder.

**Relationships**:
- Declared independently of `AgentDefinition` (no reference to any agent id).
- Embedded as a new field on the existing `GitConfig` entity (see below).

**State / lifecycle**: Read once per `init` invocation from `config.yaml`; not persisted or
mutated at runtime.

### GitConfig (existing, extended)

| Field      | Type            | YAML key    | Notes                                                    |
|------------|-----------------|-------------|-----------------------------------------------------------|
| Repository | `string`        | `repository`| Unchanged from features 001-004.                          |
| Ref        | `string`        | `ref`       | Unchanged from features 001-004.                           |
| Managed    | `ManagedConfig` | `managed`   | **New.** Zero value (`Files`/`Folders` both `nil`) when absent from `config.yaml` — fully backward compatible with features 001-004. |

### Managed File (conceptual, from spec.md)

At the `reposync.Sync` layer, a managed file has no distinct representation from an agent's
declared file — both are unioned by the caller (`internal/cli/init.go`) into the same `files`
argument, since both share identical create-or-overwrite/never-delete semantics (see
[research.md](./research.md)).

### Managed Folder (conceptual, from spec.md)

At the `reposync.Sync` layer, represented by the new `managedFolders []string` parameter.
Distinct from an agent's `declaredFolders` only in that it additionally participates in the new
post-write pruning pass (delete-stale-files), not in how its content is written (which reuses the
exact same write-set/type-conflict machinery as `declaredFolders`).

### Destination (existing concept, unchanged)

The on-disk directory `init` writes into. No new fields; managed folders/files write into and
prune from the same destination root as agent-declared content.

## Function Signature Changes

### `internal/agentcatalog/catalog.go`

```go
// ManagedConfig is the set of files and folders that apply on every init run,
// independent of which agent is selected (feature 005).
type ManagedConfig struct {
	Files   []string `yaml:"files"`
	Folders []string `yaml:"folders"`
}

type GitConfig struct {
	Repository string        `yaml:"repository"`
	Ref        string        `yaml:"ref"`
	Managed    ManagedConfig `yaml:"managed"`
}
```

`validateDeclaredPaths` (or a small new sibling function called alongside it from `LoadFS`) also
validates `catalog.Git.Managed.Files`/`Folders` using the existing `isValidDeclaredPath` check,
returning `ErrInvalidDeclaredPath` on failure.

### `internal/reposync/reposync.go`

```go
func Sync(ctx context.Context, repository, ref, destination string,
	declaredFiles, declaredFolders, managedFolders []string) error
```

One new parameter, `managedFolders`, appended at the end to minimize diff to existing call
sites' argument order. Internally:
- `selectWriteSet` is called with `declaredFolders` and `managedFolders` unioned (order does not
  matter; both feed the same "which folders' files should be in the write set" computation).
- `checkDeclaredPathsExist` is called with `declaredFolders` only — `managedFolders` is
  deliberately excluded, since a managed folder matching zero repository files is a valid
  "mirror to nothing" outcome (FR-003/Acceptance Scenario 4), not a missing-path error (FR-008
  applies to managed files only; see research.md).
- After a successful write loop, a new step calls `pruneStaleManagedFiles(destination, collected,
  managedFolders)`.

```go
// pruneStaleManagedFiles removes any regular file already at destination under one of
// managedFolders whose corresponding repository-relative path is not present in collected,
// making each managed folder an exact mirror of the retrieved repository content (FR-003).
// Directories are never removed, even if left empty. Runs only after every write in the
// current Sync call has already succeeded (best-effort, no rollback — FR-006).
func pruneStaleManagedFiles(destination string, collected map[string][]byte, managedFolders []string) error
```

### `internal/cli/init.go`

No new exported signatures. `runInit` computes:

```go
files := append(append([]string{}, agent.Files...), catalog.Git.Managed.Files...)
```

and passes `catalog.Git.Managed.Folders` as the new `managedFolders` argument to `syncRepo`,
alongside the existing `agent.Folders` as `declaredFolders`.

## Not Modeled / Explicitly Out of Scope

- No new persisted state — managed configuration is read fresh from `config.yaml` every run.
- No new CLI flags or output formats.
- No change to `AgentDefinition` — managed files/folders remain entirely separate from any
  agent's own declared files/folders (FR-001, FR-005).
