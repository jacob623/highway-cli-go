# Phase 1 Data Model: Merge Declared Folders Into Destination

This feature introduces no new entities, fields, or schema changes. It reuses the entities
established in feature 003 ([../003-selective-file-retrieval/data-model.md](../003-selective-file-retrieval/data-model.md)):

- **`AgentDefinition.Files`** and **`AgentDefinition.Folders`** (`internal/agentcatalog`) —
  unchanged.
- **`reposync.Sync`'s write set** (the `map[string][]byte` returned by `selectWriteSet`) —
  unchanged in shape; this feature only pins, via new tests, an already-correct emergent property
  of how that write set is used (destination paths outside the write set are never touched).

## Behavior clarified by this feature (no data model impact)

| Scenario | Destination outcome |
|----------|---------------------|
| Path in write set, also exists at destination | Overwritten with repository content (unchanged from feature 003) |
| Path in write set, does not yet exist at destination | Created (unchanged from feature 003) |
| Path under a declared folder, NOT in write set, already exists at destination | Left untouched — this feature's regression tests pin this |
| Path in write set is a file, destination has a directory at that exact path | `Sync` fails with an actionable error naming the path; destination directory is not deleted |
| Path in write set is nested under a destination path that is a file (not a directory) | `Sync` fails with an actionable error naming the path; destination file is not deleted |
| A write in the middle of the write set fails | Files already written earlier in the same `Sync` call remain; no rollback |

No validation rules, state transitions, or relationships change as part of this feature.
