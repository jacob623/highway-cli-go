// Package reposync clones a configured git repository at a pinned commit
// entirely in memory and writes its files to a destination directory on disk.
package reposync

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	git "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/storage/memory"

	"github.com/go-git/go-billy/v6"
	"github.com/go-git/go-billy/v6/memfs"
)

// Sync clones repository at ref entirely in memory and writes its resulting
// files into destination, overwriting any existing file at a colliding path
// (FR-009); files at non-colliding paths are left untouched. destination is
// created if it does not already exist (FR-004). No files are written to
// destination if the clone, checkout, or read of the repository's contents
// fails, or if ctx is cancelled before writing begins (FR-005, FR-007).
//
// declaredFiles and declaredFolders are the selected agent's declared paths (feature
// 003): when both are empty, nothing is written; when either is non-empty, only the
// matching subset is written, and Sync fails without writing anything if any declared
// entry does not match at least one retrieved file.
//
// managedFolders (feature 005) are additional folders applied regardless of the
// selected agent: their matching files are written using the same code path as
// declaredFolders, but a managedFolders entry matching zero retrieved files is not an
// error (unlike declaredFiles/declaredFolders) since a managed folder legitimately
// mirrors to nothing when the repository no longer provides anything under it. After a
// successful write, every managed folder's destination contents are pruned to exactly
// match the retrieved repository: any destination file under a managed folder that the
// repository does not provide is removed (FR-003). Pruning only runs after every write
// in this call has already succeeded (FR-006).
//
// seededFiles and seededFolders (feature 006) are additional paths applied regardless
// of the selected agent, like managedFolders, but with the opposite write behavior:
// each matching file is created only the first time nothing already exists at its exact
// destination path (create-if-missing), and is never overwritten, removed, or reported
// as a conflict on any later call. A seededFiles/seededFolders entry matching zero
// retrieved files is a validation error, same as declaredFiles/declaredFolders (unlike
// managedFolders), since seeded content has no legitimate "matches nothing" outcome.
func Sync(ctx context.Context, repository, ref, destination string, declaredFiles, declaredFolders, managedFolders, seededFiles, seededFolders []string) error {
	storer := memory.NewStorage()
	worktreeFS := memfs.New()

	repo, err := git.CloneContext(ctx, storer, worktreeFS, &git.CloneOptions{URL: repository})
	if err != nil {
		return fmt.Errorf("clone %s: %w", repository, err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("open worktree for %s: %w", repository, err)
	}

	if err := worktree.Checkout(&git.CheckoutOptions{Hash: plumbing.NewHash(ref)}); err != nil {
		return fmt.Errorf("checkout %s at commit %s: %w", repository, ref, err)
	}

	collected, err := collectFiles(worktreeFS, "/")
	if err != nil {
		return fmt.Errorf("read retrieved files from %s: %w", repository, err)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if err := checkDeclaredPathsExist(collected, append(append([]string{}, declaredFiles...), seededFiles...), append(append([]string{}, declaredFolders...), seededFolders...)); err != nil {
		return err
	}

	allFolders := make([]string, 0, len(declaredFolders)+len(managedFolders))
	allFolders = append(allFolders, declaredFolders...)
	allFolders = append(allFolders, managedFolders...)
	writeSet := selectWriteSet(collected, declaredFiles, allFolders)

	seedWriteSet, err := selectSeedWriteSet(destination, collected, seededFiles, seededFolders)
	if err != nil {
		return err
	}
	for relPath, data := range seedWriteSet {
		writeSet[relPath] = data
	}

	if err := os.MkdirAll(destination, 0o755); err != nil {
		return fmt.Errorf("create destination %s: %w", destination, err)
	}

	for relPath, data := range writeSet {
		destPath := filepath.Join(destination, relPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return fmt.Errorf("create directory for %s: %w", destPath, err)
		}
		if err := os.WriteFile(destPath, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", destPath, err)
		}
	}

	return pruneStaleManagedFiles(destination, collected, managedFolders)
}

// checkDeclaredPathsExist verifies every entry in declaredFiles and declaredFolders
// matches at least one path in collected: declaredFiles entries must match exactly,
// declaredFolders entries must be a path-prefix of at least one collected path. If any
// entry matches nothing, it returns an error naming every unmatched path (FR-004).
func checkDeclaredPathsExist(collected map[string][]byte, declaredFiles, declaredFolders []string) error {
	var missing []string

	for _, f := range declaredFiles {
		if _, ok := collected[filepath.FromSlash(f)]; !ok {
			missing = append(missing, f)
		}
	}

	for _, folder := range declaredFolders {
		prefix := filepath.FromSlash(folder) + string(filepath.Separator)
		found := false
		for relPath := range collected {
			if strings.HasPrefix(relPath, prefix) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, folder)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("declared path(s) not found in repository: %s", strings.Join(missing, ", "))
	}
	return nil
}

// selectWriteSet returns the subset of collected whose path exactly matches an entry in
// declaredFiles, or has an entry in declaredFolders as a path-prefix. When both
// declaredFiles and declaredFolders are empty, no files are selected (FR-003): declaring
// neither list is opt-in to writing nothing, not a fallback to writing everything.
func selectWriteSet(collected map[string][]byte, declaredFiles, declaredFolders []string) map[string][]byte {
	if len(declaredFiles) == 0 && len(declaredFolders) == 0 {
		return map[string][]byte{}
	}

	fileSet := make(map[string]bool, len(declaredFiles))
	for _, f := range declaredFiles {
		fileSet[filepath.FromSlash(f)] = true
	}

	folderPrefixes := make([]string, len(declaredFolders))
	for i, f := range declaredFolders {
		folderPrefixes[i] = filepath.FromSlash(f) + string(filepath.Separator)
	}

	result := make(map[string][]byte)
	for relPath, data := range collected {
		if fileSet[relPath] {
			result[relPath] = data
			continue
		}
		for _, prefix := range folderPrefixes {
			if strings.HasPrefix(relPath, prefix) {
				result[relPath] = data
				break
			}
		}
	}
	return result
}

// selectSeedWriteSet returns the subset of collected whose path exactly matches an
// entry in seededFiles, or has an entry in seededFolders as a path-prefix, AND for
// which nothing currently exists at the corresponding destination path. This is what
// makes seeding create-if-missing rather than always-overwrite (FR-003, FR-004,
// FR-005): a matching path that already exists at destination, of any type, is silently
// excluded rather than written or reported as a conflict.
func selectSeedWriteSet(destination string, collected map[string][]byte, seededFiles, seededFolders []string) (map[string][]byte, error) {
	candidates := selectWriteSet(collected, seededFiles, seededFolders)

	result := make(map[string][]byte, len(candidates))
	for relPath, data := range candidates {
		_, err := os.Lstat(filepath.Join(destination, relPath))
		if err == nil {
			continue
		}
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("check existing seed path %s: %w", relPath, err)
		}
		result[relPath] = data
	}
	return result, nil
}

// collectFiles recursively reads every regular file under dir in fsys into
// memory, keyed by its OS-native path relative to the initial dir.
func collectFiles(fsys billy.Filesystem, dir string) (map[string][]byte, error) {
	files := make(map[string][]byte)
	entries, err := fsys.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		path := fsys.Join(dir, entry.Name())
		if entry.IsDir() {
			sub, err := collectFiles(fsys, path)
			if err != nil {
				return nil, err
			}
			for relPath, data := range sub {
				files[filepath.Join(entry.Name(), relPath)] = data
			}
			continue
		}

		data, err := readFile(fsys, path)
		if err != nil {
			return nil, err
		}
		files[entry.Name()] = data
	}

	return files, nil
}

// readFile reads the full contents of the file at path in fsys.
func readFile(fsys billy.Filesystem, path string) ([]byte, error) {
	f, err := fsys.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// pruneStaleManagedFiles removes any regular file already at destination under one of
// managedFolders whose corresponding repository-relative path is not present in
// collected, making each managed folder an exact mirror of the retrieved repository
// content (FR-003). Directories are never removed, even if left empty. A managedFolders
// entry that does not yet exist at destination is skipped without error. This is called
// only after every write in the current Sync call has already succeeded, and is
// best-effort: it does not roll back files already removed if a later removal fails
// (FR-006).
func pruneStaleManagedFiles(destination string, collected map[string][]byte, managedFolders []string) error {
	for _, folder := range managedFolders {
		root := filepath.Join(destination, filepath.FromSlash(folder))
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if path == root && os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if d.IsDir() {
				return nil
			}
			relPath, err := filepath.Rel(destination, path)
			if err != nil {
				return err
			}
			if _, ok := collected[relPath]; !ok {
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("remove stale managed file %s: %w", path, err)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
