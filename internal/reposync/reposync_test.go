package reposync

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

	err := Sync(context.Background(), repoPath, ref, dest, []string{"README.md"}, []string{"nested"}, nil)
	if err != nil {
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

	if err := Sync(context.Background(), repoPath, ref, dest, []string{"README.md"}, nil, nil); err != nil {
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

	err := Sync(context.Background(), filepath.Join(t.TempDir(), "does-not-exist"), "abc123", dest, nil, nil, nil)
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

	err := Sync(context.Background(), repoPath, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", dest, nil, nil, nil)
	if err == nil {
		t.Fatal("Sync() expected an error for a nonexistent commit ref, got nil")
	}

	entries, _ := os.ReadDir(dest)
	if len(entries) != 0 {
		t.Errorf("Sync() wrote files to destination on invalid ref, want none written")
	}
}

func TestSync_FiltersToDeclaredFiles(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		"README.md":       "keep\n",
		"nested/file.txt": "drop me\n",
	})
	dest := t.TempDir()

	if err := Sync(context.Background(), repoPath, ref, dest, []string{"README.md"}, nil, nil); err != nil {
		t.Fatalf("Sync() unexpected error: %v", err)
	}

	if _, err := os.ReadFile(filepath.Join(dest, "README.md")); err != nil {
		t.Errorf("ReadFile(README.md) error: %v, want declared file written", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "nested", "file.txt")); !os.IsNotExist(err) {
		t.Errorf("nested/file.txt was written, want only the declared file")
	}
}

func TestSync_FiltersToDeclaredFolders(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		"README.md":          "drop me\n",
		"skills/a.md":        "keep a\n",
		"skills/sub/b.md":    "keep b\n",
		"other/untouched.md": "drop me too\n",
	})
	dest := t.TempDir()

	if err := Sync(context.Background(), repoPath, ref, dest, nil, []string{"skills"}, nil); err != nil {
		t.Fatalf("Sync() unexpected error: %v", err)
	}

	if _, err := os.ReadFile(filepath.Join(dest, "skills", "a.md")); err != nil {
		t.Errorf("ReadFile(skills/a.md) error: %v, want it written", err)
	}
	if _, err := os.ReadFile(filepath.Join(dest, "skills", "sub", "b.md")); err != nil {
		t.Errorf("ReadFile(skills/sub/b.md) error: %v, want it written recursively", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "README.md")); !os.IsNotExist(err) {
		t.Errorf("README.md was written, want only files under the declared folder")
	}
	if _, err := os.Stat(filepath.Join(dest, "other", "untouched.md")); !os.IsNotExist(err) {
		t.Errorf("other/untouched.md was written, want only files under the declared folder")
	}
}

func TestSync_FiltersUnionOfFilesAndFolders(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		"README.md":       "keep via files\n",
		"skills/a.md":     "keep via folders\n",
		"other/ignore.md": "drop me\n",
	})
	dest := t.TempDir()

	err := Sync(context.Background(), repoPath, ref, dest, []string{"README.md"}, []string{"skills"}, nil)
	if err != nil {
		t.Fatalf("Sync() unexpected error: %v", err)
	}

	if _, err := os.ReadFile(filepath.Join(dest, "README.md")); err != nil {
		t.Errorf("ReadFile(README.md) error: %v, want it written", err)
	}
	if _, err := os.ReadFile(filepath.Join(dest, "skills", "a.md")); err != nil {
		t.Errorf("ReadFile(skills/a.md) error: %v, want it written", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "other", "ignore.md")); !os.IsNotExist(err) {
		t.Errorf("other/ignore.md was written, want only the union of declared files/folders")
	}
}

func TestSync_DuplicateAndOverlappingEntriesWriteOnce(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		"skills/a.md": "content\n",
	})
	dest := t.TempDir()

	err := Sync(context.Background(), repoPath, ref, dest,
		[]string{"skills/a.md", "skills/a.md"}, // duplicate file entry
		[]string{"skills", "skills"},           // duplicate + overlapping folder entry
		nil,
	)
	if err != nil {
		t.Fatalf("Sync() unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dest, "skills", "a.md"))
	if err != nil {
		t.Fatalf("ReadFile(skills/a.md) error: %v", err)
	}
	if string(got) != "content\n" {
		t.Errorf("skills/a.md content = %q, want %q", got, "content\n")
	}
}

func TestSync_NoDeclaredListsWritesNothing(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		"README.md":       "hello\n",
		"nested/file.txt": "nested content\n",
	})
	dest := t.TempDir()

	if err := Sync(context.Background(), repoPath, ref, dest, nil, nil, nil); err != nil {
		t.Fatalf("Sync() unexpected error: %v", err)
	}

	entries, err := os.ReadDir(dest)
	if err != nil {
		t.Fatalf("ReadDir(dest) error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Sync() wrote %d entries to destination, want none when no files/folders are declared", len(entries))
	}
}

func TestSync_MissingDeclaredFileFails(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{"README.md": "hello\n"})
	dest := t.TempDir()

	err := Sync(context.Background(), repoPath, ref, dest, []string{"does-not-exist.md"}, nil, nil)
	if err == nil {
		t.Fatal("Sync() expected an error for a missing declared file, got nil")
	}
	if !strings.Contains(err.Error(), "does-not-exist.md") {
		t.Errorf("Sync() error = %q, want it to name the missing path", err)
	}

	entries, _ := os.ReadDir(dest)
	if len(entries) != 0 {
		t.Errorf("Sync() wrote files to destination on missing declared file, want none written")
	}
}

func TestSync_MissingDeclaredFolderFails(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{"README.md": "hello\n"})
	dest := t.TempDir()

	err := Sync(context.Background(), repoPath, ref, dest, nil, []string{"does-not-exist"}, nil)
	if err == nil {
		t.Fatal("Sync() expected an error for a missing declared folder, got nil")
	}
	if !strings.Contains(err.Error(), "does-not-exist") {
		t.Errorf("Sync() error = %q, want it to name the missing path", err)
	}

	entries, _ := os.ReadDir(dest)
	if len(entries) != 0 {
		t.Errorf("Sync() wrote files to destination on missing declared folder, want none written")
	}
}

func TestSync_MultipleMissingPathsNamedInError(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{"README.md": "hello\n"})
	dest := t.TempDir()

	err := Sync(context.Background(), repoPath, ref, dest,
		[]string{"missing-file.md"},
		[]string{"missing-folder"},
		nil,
	)
	if err == nil {
		t.Fatal("Sync() expected an error for missing declared paths, got nil")
	}
	if !strings.Contains(err.Error(), "missing-file.md") {
		t.Errorf("Sync() error = %q, want it to name missing-file.md", err)
	}
	if !strings.Contains(err.Error(), "missing-folder") {
		t.Errorf("Sync() error = %q, want it to name missing-folder", err)
	}
}

func TestSync_MissingPathLeavesDestinationUntouched(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		"README.md":   "hello\n",
		"skills/a.md": "content\n",
	})
	dest := t.TempDir()
	if err := os.WriteFile(filepath.Join(dest, "existing.txt"), []byte("keep me\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(existing.txt) error: %v", err)
	}

	err := Sync(context.Background(), repoPath, ref, dest, []string{"README.md", "missing.md"}, []string{"skills"}, nil)
	if err == nil {
		t.Fatal("Sync() expected an error for a missing declared path, got nil")
	}

	if _, statErr := os.Stat(filepath.Join(dest, "README.md")); !os.IsNotExist(statErr) {
		t.Errorf("README.md was written despite a missing declared path elsewhere, want no files written")
	}
	if _, statErr := os.Stat(filepath.Join(dest, "skills", "a.md")); !os.IsNotExist(statErr) {
		t.Errorf("skills/a.md was written despite a missing declared path elsewhere, want no files written")
	}
	existing, err := os.ReadFile(filepath.Join(dest, "existing.txt"))
	if err != nil {
		t.Fatalf("ReadFile(existing.txt) error: %v", err)
	}
	if string(existing) != "keep me\n" {
		t.Errorf("existing.txt content = %q, want unchanged %q", existing, "keep me\n")
	}
}

func TestSync_MergePreservesDestinationFileNotProvidedByRepoFolder(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		"skills/highway-activities/file.md": "new content\n",
	})
	dest := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dest, "skills", "highway-activities"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "skills", "highway-activities", "file.md"), []byte("old content\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(highway-activities/file.md) error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dest, "skills", "speckit-specify"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "skills", "speckit-specify", "file.md"), []byte("keep me\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(speckit-specify/file.md) error: %v", err)
	}

	if err := Sync(context.Background(), repoPath, ref, dest, nil, []string{"skills"}, nil); err != nil {
		t.Fatalf("Sync() unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dest, "skills", "highway-activities", "file.md"))
	if err != nil {
		t.Fatalf("ReadFile(highway-activities/file.md) error: %v", err)
	}
	if string(got) != "new content\n" {
		t.Errorf("highway-activities/file.md content = %q, want overwritten %q", got, "new content\n")
	}

	untouched, err := os.ReadFile(filepath.Join(dest, "skills", "speckit-specify", "file.md"))
	if err != nil {
		t.Fatalf("ReadFile(speckit-specify/file.md) error: %v", err)
	}
	if string(untouched) != "keep me\n" {
		t.Errorf("speckit-specify/file.md content = %q, want unchanged %q", untouched, "keep me\n")
	}
}

func TestSync_MergeLeavesUnrelatedDestinationFileUntouched(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		"skills/a.md": "keep a\n",
	})
	dest := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dest, "other"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "other", "unrelated.md"), []byte("keep me\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(other/unrelated.md) error: %v", err)
	}

	if err := Sync(context.Background(), repoPath, ref, dest, nil, []string{"skills"}, nil); err != nil {
		t.Fatalf("Sync() unexpected error: %v", err)
	}

	untouched, err := os.ReadFile(filepath.Join(dest, "other", "unrelated.md"))
	if err != nil {
		t.Fatalf("ReadFile(other/unrelated.md) error: %v", err)
	}
	if string(untouched) != "keep me\n" {
		t.Errorf("other/unrelated.md content = %q, want unchanged %q", untouched, "keep me\n")
	}
}

func TestSync_MergePreservesNestedSiblingSubfoldersAndFiles(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		"skills/a/new.md": "new content\n",
	})
	dest := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dest, "skills", "a"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "skills", "a", "existing.md"), []byte("keep a\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(a/existing.md) error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dest, "skills", "b"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "skills", "b", "existing.md"), []byte("keep b\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(b/existing.md) error: %v", err)
	}

	if err := Sync(context.Background(), repoPath, ref, dest, nil, []string{"skills"}, nil); err != nil {
		t.Fatalf("Sync() unexpected error: %v", err)
	}

	newFile, err := os.ReadFile(filepath.Join(dest, "skills", "a", "new.md"))
	if err != nil {
		t.Fatalf("ReadFile(a/new.md) error: %v", err)
	}
	if string(newFile) != "new content\n" {
		t.Errorf("a/new.md content = %q, want %q", newFile, "new content\n")
	}

	aExisting, err := os.ReadFile(filepath.Join(dest, "skills", "a", "existing.md"))
	if err != nil {
		t.Fatalf("ReadFile(a/existing.md) error: %v", err)
	}
	if string(aExisting) != "keep a\n" {
		t.Errorf("a/existing.md content = %q, want unchanged %q", aExisting, "keep a\n")
	}

	bExisting, err := os.ReadFile(filepath.Join(dest, "skills", "b", "existing.md"))
	if err != nil {
		t.Fatalf("ReadFile(b/existing.md) error: %v", err)
	}
	if string(bExisting) != "keep b\n" {
		t.Errorf("b/existing.md content = %q, want unchanged %q", bExisting, "keep b\n")
	}
}

func TestSync_TypeConflictFileWhereDestinationHasDirectory(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		"skills/conflict": "should be a file\n",
	})
	dest := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dest, "skills", "conflict"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}

	err := Sync(context.Background(), repoPath, ref, dest, nil, []string{"skills"}, nil)
	if err == nil {
		t.Fatal("Sync() expected an error for a file/directory type conflict, got nil")
	}
	if !strings.Contains(err.Error(), filepath.Join("skills", "conflict")) {
		t.Errorf("Sync() error = %q, want it to name the conflicting path", err)
	}

	info, statErr := os.Stat(filepath.Join(dest, "skills", "conflict"))
	if statErr != nil {
		t.Fatalf("Stat(skills/conflict) error: %v, want the destination directory to still exist", statErr)
	}
	if !info.IsDir() {
		t.Error("skills/conflict is no longer a directory, want it left untouched")
	}
}

func TestSync_TypeConflictDirectoryWhereDestinationHasFile(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		"skills/conflict/inner.md": "should live inside a directory\n",
	})
	dest := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dest, "skills"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "skills", "conflict"), []byte("should stay a file\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(skills/conflict) error: %v", err)
	}

	err := Sync(context.Background(), repoPath, ref, dest, nil, []string{"skills"}, nil)
	if err == nil {
		t.Fatal("Sync() expected an error for a directory/file type conflict, got nil")
	}
	if !strings.Contains(err.Error(), filepath.Join("skills", "conflict")) {
		t.Errorf("Sync() error = %q, want it to name the conflicting path", err)
	}

	got, readErr := os.ReadFile(filepath.Join(dest, "skills", "conflict"))
	if readErr != nil {
		t.Fatalf("ReadFile(skills/conflict) error: %v, want the destination file to still exist", readErr)
	}
	if string(got) != "should stay a file\n" {
		t.Errorf("skills/conflict content = %q, want unchanged %q", got, "should stay a file\n")
	}
}

func TestSync_FailedRunDoesNotRevertFilesFromEarlierSuccessfulRun(t *testing.T) {
	firstRepoPath, firstRef := newFixtureRepo(t, map[string]string{
		"skills/keep/a.md": "content\n",
	})
	dest := t.TempDir()

	if err := Sync(context.Background(), firstRepoPath, firstRef, dest, nil, []string{"skills"}, nil); err != nil {
		t.Fatalf("Sync() first run unexpected error: %v", err)
	}
	if _, err := os.ReadFile(filepath.Join(dest, "skills", "keep", "a.md")); err != nil {
		t.Fatalf("ReadFile(skills/keep/a.md) error after first run: %v", err)
	}

	secondRepoPath, secondRef := newFixtureRepo(t, map[string]string{
		"skills/keep/a.md": "content\n",
		"skills/conflict":  "should be a file\n",
	})
	if err := os.MkdirAll(filepath.Join(dest, "skills", "conflict"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}

	err := Sync(context.Background(), secondRepoPath, secondRef, dest, nil, []string{"skills"}, nil)
	if err == nil {
		t.Fatal("Sync() second run expected an error for the type conflict, got nil")
	}

	got, readErr := os.ReadFile(filepath.Join(dest, "skills", "keep", "a.md"))
	if readErr != nil {
		t.Fatalf("ReadFile(skills/keep/a.md) error after failed second run: %v, want it left in place", readErr)
	}
	if string(got) != "content\n" {
		t.Errorf("skills/keep/a.md content = %q, want unchanged %q", got, "content\n")
	}
}

func TestSync_ManagedFolderPrunesStaleDestinationFile(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		".highway/keep.md": "new content\n",
	})
	dest := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dest, ".highway"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, ".highway", "stale.md"), []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(.highway/stale.md) error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, ".highway", "keep.md"), []byte("old content\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(.highway/keep.md) error: %v", err)
	}

	if err := Sync(context.Background(), repoPath, ref, dest, nil, nil, []string{".highway"}); err != nil {
		t.Fatalf("Sync() unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dest, ".highway", "stale.md")); !os.IsNotExist(err) {
		t.Errorf(".highway/stale.md was not removed, want it pruned")
	}

	got, err := os.ReadFile(filepath.Join(dest, ".highway", "keep.md"))
	if err != nil {
		t.Fatalf("ReadFile(.highway/keep.md) error: %v", err)
	}
	if string(got) != "new content\n" {
		t.Errorf(".highway/keep.md content = %q, want overwritten %q", got, "new content\n")
	}
}

func TestSync_ManagedFolderMirrorsRecursively(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		".highway/sub/new.md": "new content\n",
	})
	dest := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dest, ".highway", "sub"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, ".highway", "sub", "stale1.md"), []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(.highway/sub/stale1.md) error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dest, ".highway", "other"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, ".highway", "other", "stale2.md"), []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(.highway/other/stale2.md) error: %v", err)
	}

	if err := Sync(context.Background(), repoPath, ref, dest, nil, nil, []string{".highway"}); err != nil {
		t.Fatalf("Sync() unexpected error: %v", err)
	}

	if _, err := os.ReadFile(filepath.Join(dest, ".highway", "sub", "new.md")); err != nil {
		t.Errorf("ReadFile(.highway/sub/new.md) error: %v, want it written", err)
	}
	if _, err := os.Stat(filepath.Join(dest, ".highway", "sub", "stale1.md")); !os.IsNotExist(err) {
		t.Errorf(".highway/sub/stale1.md was not removed, want it pruned")
	}
	if _, err := os.Stat(filepath.Join(dest, ".highway", "other", "stale2.md")); !os.IsNotExist(err) {
		t.Errorf(".highway/other/stale2.md was not removed, want it pruned")
	}
}

func TestSync_ManagedFolderRemovedEntirelyWhenRepoProvidesNoFiles(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		"README.md": "hello\n",
	})
	dest := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dest, ".highway", "sub"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, ".highway", "a.md"), []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(.highway/a.md) error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, ".highway", "sub", "b.md"), []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(.highway/sub/b.md) error: %v", err)
	}

	if err := Sync(context.Background(), repoPath, ref, dest, []string{"README.md"}, nil, []string{".highway"}); err != nil {
		t.Fatalf("Sync() unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dest, ".highway", "a.md")); !os.IsNotExist(err) {
		t.Errorf(".highway/a.md was not removed, want it pruned")
	}
	if _, err := os.Stat(filepath.Join(dest, ".highway", "sub", "b.md")); !os.IsNotExist(err) {
		t.Errorf(".highway/sub/b.md was not removed, want it pruned")
	}
}

func TestSync_ManagedFolderDoesNotAffectDeclaredFolderPruning(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		"skills/a.md": "keep a\n",
	})
	dest := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dest, "skills"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "skills", "stale.md"), []byte("stale but not managed\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(skills/stale.md) error: %v", err)
	}

	if err := Sync(context.Background(), repoPath, ref, dest, nil, []string{"skills"}, nil); err != nil {
		t.Fatalf("Sync() unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dest, "skills", "stale.md"))
	if err != nil {
		t.Fatalf("ReadFile(skills/stale.md) error: %v, want it left in place (feature 004 merge-only semantics)", err)
	}
	if string(got) != "stale but not managed\n" {
		t.Errorf("skills/stale.md content = %q, want unchanged %q", got, "stale but not managed\n")
	}
}

func TestSync_ManagedFolderTypeConflictFileWhereDestinationHasDirectory(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		".highway/conflict": "should be a file\n",
	})
	dest := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dest, ".highway", "conflict"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}

	err := Sync(context.Background(), repoPath, ref, dest, nil, nil, []string{".highway"})
	if err == nil {
		t.Fatal("Sync() expected an error for a file/directory type conflict, got nil")
	}
	if !strings.Contains(err.Error(), filepath.Join(".highway", "conflict")) {
		t.Errorf("Sync() error = %q, want it to name the conflicting path", err)
	}

	info, statErr := os.Stat(filepath.Join(dest, ".highway", "conflict"))
	if statErr != nil {
		t.Fatalf("Stat(.highway/conflict) error: %v, want the destination directory to still exist", statErr)
	}
	if !info.IsDir() {
		t.Error(".highway/conflict is no longer a directory, want it left untouched")
	}
}

func TestSync_ManagedFolderTypeConflictDirectoryWhereDestinationHasFile(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		".highway/conflict/inner.md": "should live inside a directory\n",
	})
	dest := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dest, ".highway"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, ".highway", "conflict"), []byte("should stay a file\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(.highway/conflict) error: %v", err)
	}

	err := Sync(context.Background(), repoPath, ref, dest, nil, nil, []string{".highway"})
	if err == nil {
		t.Fatal("Sync() expected an error for a directory/file type conflict, got nil")
	}
	if !strings.Contains(err.Error(), filepath.Join(".highway", "conflict")) {
		t.Errorf("Sync() error = %q, want it to name the conflicting path", err)
	}

	got, readErr := os.ReadFile(filepath.Join(dest, ".highway", "conflict"))
	if readErr != nil {
		t.Fatalf("ReadFile(.highway/conflict) error: %v, want the destination file to still exist", readErr)
	}
	if string(got) != "should stay a file\n" {
		t.Errorf(".highway/conflict content = %q, want unchanged %q", got, "should stay a file\n")
	}
}

func TestSync_ManagedFolderMatchingZeroFilesIsNotAnError(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		"README.md": "hello\n",
	})
	dest := t.TempDir()

	err := Sync(context.Background(), repoPath, ref, dest, []string{"README.md"}, nil, []string{".highway"})
	if err != nil {
		t.Fatalf("Sync() unexpected error: %v, want a managed folder matching zero files to succeed", err)
	}

	if _, err := os.ReadFile(filepath.Join(dest, "README.md")); err != nil {
		t.Errorf("ReadFile(README.md) error: %v, want it written", err)
	}
}

func TestSync_ManagedFolderWriteFailureLeavesPruningUnattempted(t *testing.T) {
	repoPath, ref := newFixtureRepo(t, map[string]string{
		".highway/keep.md":  "new content\n",
		".highway/conflict": "should be a file\n",
	})
	dest := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dest, ".highway", "conflict"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, ".highway", "stale.md"), []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(.highway/stale.md) error: %v", err)
	}

	err := Sync(context.Background(), repoPath, ref, dest, nil, nil, []string{".highway"})
	if err == nil {
		t.Fatal("Sync() expected an error for a type conflict, got nil")
	}

	if _, statErr := os.Stat(filepath.Join(dest, ".highway", "stale.md")); statErr != nil {
		t.Errorf("Stat(.highway/stale.md) error: %v, want it left in place because pruning must not run after a write failure", statErr)
	}
}
