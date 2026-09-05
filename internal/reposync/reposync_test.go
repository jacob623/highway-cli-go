package reposync

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	git "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
)

// newFixtureRepo creates a local git repository on disk with the given files
// committed, returning the repository path and the commit SHA.
func newFixtureRepo(t *testing.T, files map[string]string) (path, ref string) {
	t.Helper()
	dir := t.TempDir()

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit() error: %v", err)
	}

	for relPath, content := range files {
		full := filepath.Join(dir, relPath)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("MkdirAll() error: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile() error: %v", err)
		}
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree() error: %v", err)
	}
	if _, err := wt.Add("."); err != nil {
		t.Fatalf("Add() error: %v", err)
	}

	sig := &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()}
	hash, err := wt.Commit("fixture commit", &git.CommitOptions{Author: sig, Committer: sig})
	if err != nil {
		t.Fatalf("Commit() error: %v", err)
	}

	return dir, hash.String()
}

func TestSync_Success(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		"README.md":       "hello\n",
		"nested/file.txt": "nested content\n",
	})
	dest := t.TempDir()

	if err := Sync(context.Background(), repoPath, ref, dest); err != nil {
		t.Fatalf("Sync() unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dest, "README.md"))
	if err != nil {
		t.Fatalf("ReadFile(README.md) error: %v", err)
	}
	if string(got) != "hello\n" {
		t.Errorf("README.md content = %q, want %q", got, "hello\n")
	}

	gotNested, err := os.ReadFile(filepath.Join(dest, "nested", "file.txt"))
	if err != nil {
		t.Fatalf("ReadFile(nested/file.txt) error: %v", err)
	}
	if string(gotNested) != "nested content\n" {
		t.Errorf("nested/file.txt content = %q, want %q", gotNested, "nested content\n")
	}

	if _, err := os.Stat(filepath.Join(dest, ".git")); !os.IsNotExist(err) {
		t.Errorf(".git metadata was written to destination, want none")
	}
}

func TestSync_OverwritesCollidingFilesOnly(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		"README.md": "new content\n",
	})
	dest := t.TempDir()

	if err := os.WriteFile(filepath.Join(dest, "README.md"), []byte("old content\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(README.md) error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "untouched.txt"), []byte("keep me\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(untouched.txt) error: %v", err)
	}

	if err := Sync(context.Background(), repoPath, ref, dest); err != nil {
		t.Fatalf("Sync() unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dest, "README.md"))
	if err != nil {
		t.Fatalf("ReadFile(README.md) error: %v", err)
	}
	if string(got) != "new content\n" {
		t.Errorf("README.md content = %q, want overwritten %q", got, "new content\n")
	}

	untouched, err := os.ReadFile(filepath.Join(dest, "untouched.txt"))
	if err != nil {
		t.Fatalf("ReadFile(untouched.txt) error: %v", err)
	}
	if string(untouched) != "keep me\n" {
		t.Errorf("untouched.txt content = %q, want unchanged %q", untouched, "keep me\n")
	}
}

func TestSync_InvalidRepository(t *testing.T) {
	dest := t.TempDir()
	before, _ := os.ReadDir(dest)

	err := Sync(context.Background(), filepath.Join(t.TempDir(), "does-not-exist"), "abc123", dest)
	if err == nil {
		t.Fatal("Sync() expected an error for an invalid repository, got nil")
	}

	after, _ := os.ReadDir(dest)
	if len(after) != len(before) {
		t.Errorf("Sync() wrote files to destination on failure, want none written")
	}
}

func TestSync_InvalidRef(t *testing.T) {
	repoPath, _ := newFixtureRepo(t, map[string]string{"README.md": "hello\n"})
	dest := t.TempDir()

	err := Sync(context.Background(), repoPath, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", dest)
	if err == nil {
		t.Fatal("Sync() expected an error for a nonexistent commit ref, got nil")
	}

	entries, _ := os.ReadDir(dest)
	if len(entries) != 0 {
		t.Errorf("Sync() wrote files to destination on invalid ref, want none written")
	}
}
