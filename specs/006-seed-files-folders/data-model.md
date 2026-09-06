# Phase 1 Data Model: Seed Files and Folders

## `SeededConfig` (new)

Location: `internal/agentcatalog/catalog.go`

```go
// SeededConfig is the set of files and folders that are created at the destination on init,
// regardless of which agent is selected, but only the first time nothing already exists at
// their exact destination path (feature 006). Unlike ManagedConfig (feature 005), seeded
// content is never overwritten or removed once it exists — this is what makes it safe to
// customize locally. A zero-value SeededConfig (both fields nil) is valid and means no seeded
// files/folders are declared.
type SeededConfig struct {
	Files   []string `yaml:"files"`
	Folders []string `yaml:"folders"`
}
```

**Validation rules** (`validateSeededPaths`, mirrors `validateManagedPaths`):
- Every entry in `Files` and `Folders` MUST pass `isValidDeclaredPath` (non-empty, relative, no
  `..` traversal) — same rule already applied to `Agents[].Files/Folders` and
  `Git.Managed.Files/Folders`.

## `GitConfig` (extended)

```go
type GitConfig struct {
	Repository string        `yaml:"repository"`
	Ref        string        `yaml:"ref"`
	Managed    ManagedConfig `yaml:"managed"`
	Seeded     SeededConfig  `yaml:"seeded"`
}
```

No change to `Repository`/`Ref`/`Managed` semantics. `GitConfig` was already non-comparable via
`==` after feature 005 added `Managed` (a struct containing slices); adding `Seeded` (same shape)
does not introduce any new comparability concern — existing field-by-field test assertions
continue to apply, extended with two more field checks.

## `reposync.Sync` (extended signature)

```go
func Sync(
	ctx context.Context,
	repository, ref, destination string,
	declaredFiles, declaredFolders []string,
	managedFolders []string,
	seededFiles, seededFolders []string,
) error
```

Two new trailing parameters, appended after feature 005's `managedFolders` to minimize diff to
existing call sites' argument order (same rationale feature 005 used when appending
`managedFolders` after `declaredFolders`). Internally:
- `checkDeclaredPathsExist` is called with `declaredFiles ∪ seededFiles` and `declaredFolders ∪
  seededFolders` (a missing seeded path is a validation error, same as a missing declared path;
  see research.md).
- `selectWriteSet` is called exactly as in feature 005 (`declaredFolders ∪ managedFolders`) —
  unaffected by this feature.
- A new call to `selectSeedWriteSet(destination, collected, seededFiles, seededFolders)` produces
  a second write-set map, merged into the first before the single existing write loop runs.
- `pruneStaleManagedFiles` is called exactly as in feature 005 — unaffected; seeded content is
  never pruned.

## `selectSeedWriteSet` (new)

Location: `internal/reposync/reposync.go`

```go
// selectSeedWriteSet returns the subset of collected whose path exactly matches an entry in
// seededFiles, or has an entry in seededFolders as a path-prefix, AND for which nothing
// currently exists at the corresponding destination path (os.Lstat reports IsNotExist).
// Matching entries that already exist at destination are silently excluded — this is the
// mechanism that makes seeding create-if-missing rather than always-overwrite (FR-003, FR-004,
// FR-005). Returns an error if an Lstat call fails for any reason other than IsNotExist.
func selectSeedWriteSet(destination string, collected map[string][]byte, seededFiles, seededFolders []string) (map[string][]byte, error)
```

## Key Entities (from spec.md, restated with implementation mapping)

- **Seeded File** → an entry in `SeededConfig.Files`; matched exactly against `collected`'s keys;
  included in the write set only when absent at destination.
- **Seeded Folder** → an entry in `SeededConfig.Folders`; matched as a path-prefix against
  `collected`'s keys (recursively, since `collected` keys already encode full nested paths);
  each matching file independently included in the write set only when absent at destination.
- **Destination** → unchanged existing entity; this feature adds a read-only existence check
  (`os.Lstat`) against it before writing, in addition to the existing write operations.
