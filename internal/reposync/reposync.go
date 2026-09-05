// Package reposync clones a configured git repository at a pinned commit
// entirely in memory and writes its files to a destination directory on disk.
package reposync

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
func Sync(ctx context.Context, repository, ref, destination string) error {
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

	files, err := collectFiles(worktreeFS, "/")
	if err != nil {
		return fmt.Errorf("read retrieved files from %s: %w", repository, err)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if err := os.MkdirAll(destination, 0o755); err != nil {
		return fmt.Errorf("create destination %s: %w", destination, err)
	}

	for relPath, data := range files {
		destPath := filepath.Join(destination, relPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return fmt.Errorf("create directory for %s: %w", destPath, err)
		}
		if err := os.WriteFile(destPath, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", destPath, err)
		}
	}

	return nil
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
