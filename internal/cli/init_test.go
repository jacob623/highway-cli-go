package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/huh"

	"github.com/jacob623/highway-cli-go/internal/agentcatalog"
)

func withInitStubs(
	t *testing.T,
	load func() (*agentcatalog.AgentCatalog, error),
	terminal func(*os.File) bool,
	sel func([]huh.Option[string]) (string, error),
	sync func(context.Context, string, string, string, []string, []string, []string) error,
	wd func() (string, error),
) {
	t.Helper()
	origLoad, origTerminal, origSelect, origSync, origGetwd := loadCatalog, isTerminal, selectAgent, syncRepo, getwd
	t.Cleanup(func() {
		loadCatalog, isTerminal, selectAgent, syncRepo, getwd = origLoad, origTerminal, origSelect, origSync, origGetwd
	})
	if load != nil {
		loadCatalog = load
	}
	if terminal != nil {
		isTerminal = terminal
	}
	if sel != nil {
		selectAgent = sel
	}
	if sync != nil {
		syncRepo = sync
	}
	if wd != nil {
		getwd = wd
	}
}

func fakeCatalog(agents ...agentcatalog.AgentDefinition) func() (*agentcatalog.AgentCatalog, error) {
	return func() (*agentcatalog.AgentCatalog, error) {
		return &agentcatalog.AgentCatalog{
			Agents: agents,
			Git:    agentcatalog.GitConfig{Repository: "https://example.com/example/repo.git", Ref: "abc123"},
		}, nil
	}
}

func noopSync(context.Context, string, string, string, []string, []string, []string) error {
	return nil
}

func fakeGetwd(path string) func() (string, error) {
	return func() (string, error) { return path, nil }
}

func TestRunInit_NonInteractive(t *testing.T) {
	withInitStubs(t,
		fakeCatalog(agentcatalog.AgentDefinition{ID: "vscode", DisplayName: "GitHub Copilot"}),
		func(*os.File) bool { return false },
		nil, nil, nil,
	)

	var stdout, stderr bytes.Buffer
	err := runInit(context.Background(), &stdout, &stderr, "")

	if err == nil {
		t.Fatal("runInit() expected an error when no interactive terminal is available")
	}
	if stdout.Len() != 0 {
		t.Errorf("runInit() stdout = %q, want empty (no confirmation shown)", stdout.String())
	}
	if !strings.Contains(stderr.String(), "interactive terminal") {
		t.Errorf("runInit() stderr = %q, want message mentioning interactive terminal", stderr.String())
	}
}

func TestRunInit_HappyPath(t *testing.T) {
	withInitStubs(t,
		fakeCatalog(
			agentcatalog.AgentDefinition{ID: "vscode", DisplayName: "GitHub Copilot"},
			agentcatalog.AgentDefinition{ID: "cursor", DisplayName: "Cursor"},
		),
		func(*os.File) bool { return true },
		func([]huh.Option[string]) (string, error) { return "cursor", nil },
		noopSync,
		fakeGetwd("/tmp/current-dir"),
	)

	var stdout, stderr bytes.Buffer
	err := runInit(context.Background(), &stdout, &stderr, "")

	if err != nil {
		t.Fatalf("runInit() unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Cursor") {
		t.Errorf("runInit() stdout = %q, want confirmation naming Cursor", stdout.String())
	}
	if !strings.Contains(stdout.String(), "/tmp/current-dir") {
		t.Errorf("runInit() stdout = %q, want confirmation naming the current working directory", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("runInit() stderr = %q, want empty", stderr.String())
	}
}

func TestRunInit_ExplicitDestinationPath(t *testing.T) {
	var gotDestination string
	withInitStubs(t,
		fakeCatalog(agentcatalog.AgentDefinition{ID: "vscode", DisplayName: "GitHub Copilot"}),
		func(*os.File) bool { return true },
		func([]huh.Option[string]) (string, error) { return "vscode", nil },
		func(_ context.Context, _, _, destination string, _, _, _ []string) error {
			gotDestination = destination
			return nil
		},
		func() (string, error) {
			t.Fatal("getwd should not be called when an explicit path is supplied")
			return "", nil
		},
	)

	var stdout, stderr bytes.Buffer
	err := runInit(context.Background(), &stdout, &stderr, "/tmp/explicit-dest")

	if err != nil {
		t.Fatalf("runInit() unexpected error: %v", err)
	}
	if gotDestination != "/tmp/explicit-dest" {
		t.Errorf("runInit() destination = %q, want %q", gotDestination, "/tmp/explicit-dest")
	}
	if !strings.Contains(stdout.String(), "/tmp/explicit-dest") {
		t.Errorf("runInit() stdout = %q, want confirmation naming the explicit destination", stdout.String())
	}
}

func TestRunInit_PassesSelectedAgentFilesAndFolders(t *testing.T) {
	var gotFiles, gotFolders []string
	withInitStubs(t,
		fakeCatalog(
			agentcatalog.AgentDefinition{
				ID:          "vscode",
				DisplayName: "GitHub Copilot",
				Files:       []string{"file1.md", "file2.md"},
				Folders:     []string{".github/skills", ".idea/library"},
			},
			agentcatalog.AgentDefinition{ID: "cursor", DisplayName: "Cursor"},
		),
		func(*os.File) bool { return true },
		func([]huh.Option[string]) (string, error) { return "vscode", nil },
		func(_ context.Context, _, _, _ string, files, folders, _ []string) error {
			gotFiles = files
			gotFolders = folders
			return nil
		},
		fakeGetwd("/tmp/current-dir"),
	)

	var stdout, stderr bytes.Buffer
	if err := runInit(context.Background(), &stdout, &stderr, ""); err != nil {
		t.Fatalf("runInit() unexpected error: %v", err)
	}

	wantFiles := []string{"file1.md", "file2.md"}
	wantFolders := []string{".github/skills", ".idea/library"}
	if !slices.Equal(gotFiles, wantFiles) {
		t.Errorf("runInit() passed files = %v, want %v", gotFiles, wantFiles)
	}
	if !slices.Equal(gotFolders, wantFolders) {
		t.Errorf("runInit() passed folders = %v, want %v", gotFolders, wantFolders)
	}
}

func TestRunInit_NoDeclaredListsStillSucceeds(t *testing.T) {
	withInitStubs(t,
		fakeCatalog(agentcatalog.AgentDefinition{ID: "vscode", DisplayName: "GitHub Copilot"}),
		func(*os.File) bool { return true },
		func([]huh.Option[string]) (string, error) { return "vscode", nil },
		noopSync,
		fakeGetwd("/tmp/current-dir"),
	)

	var stdout, stderr bytes.Buffer
	err := runInit(context.Background(), &stdout, &stderr, "")

	if err != nil {
		t.Fatalf("runInit() unexpected error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Errorf("runInit() stderr = %q, want empty on success", stderr.String())
	}
	if !strings.Contains(stdout.String(), "/tmp/current-dir") {
		t.Errorf("runInit() stdout = %q, want confirmation naming the destination even though nothing was written", stdout.String())
	}
}

func TestRunInit_Cancellation(t *testing.T) {
	withInitStubs(t,
		fakeCatalog(agentcatalog.AgentDefinition{ID: "vscode", DisplayName: "GitHub Copilot"}),
		func(*os.File) bool { return true },
		func([]huh.Option[string]) (string, error) { return "", huh.ErrUserAborted },
		nil, nil,
	)

	var stdout, stderr bytes.Buffer
	err := runInit(context.Background(), &stdout, &stderr, "")

	if !errors.Is(err, huh.ErrUserAborted) {
		t.Errorf("runInit() error = %v, want huh.ErrUserAborted", err)
	}
	if stdout.Len() != 0 {
		t.Errorf("runInit() stdout = %q, want empty on cancellation", stdout.String())
	}
}

func TestRunInit_SyncCancellation(t *testing.T) {
	withInitStubs(t,
		fakeCatalog(agentcatalog.AgentDefinition{ID: "vscode", DisplayName: "GitHub Copilot"}),
		func(*os.File) bool { return true },
		func([]huh.Option[string]) (string, error) { return "vscode", nil },
		func(context.Context, string, string, string, []string, []string, []string) error {
			return context.Canceled
		},
		fakeGetwd("/tmp/current-dir"),
	)

	var stdout, stderr bytes.Buffer
	err := runInit(context.Background(), &stdout, &stderr, "")

	if !errors.Is(err, context.Canceled) {
		t.Errorf("runInit() error = %v, want it to wrap context.Canceled", err)
	}
	if strings.Contains(stdout.String(), "Files written") {
		t.Errorf("runInit() stdout = %q, want no success confirmation on cancellation", stdout.String())
	}
	if !strings.Contains(stderr.String(), "cancel") {
		t.Errorf("runInit() stderr = %q, want a message mentioning cancellation", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Re-run the command") {
		t.Errorf("runInit() stderr = %q, want it to instruct the user to re-run the command", stderr.String())
	}
}

func TestRunInit_CatalogError(t *testing.T) {
	withInitStubs(t,
		func() (*agentcatalog.AgentCatalog, error) { return nil, agentcatalog.ErrCatalogEmpty },
		func(*os.File) bool { return true },
		func([]huh.Option[string]) (string, error) {
			t.Fatal("selectAgent should not be called when the catalog fails to load")
			return "", nil
		},
		nil, nil,
	)

	var stdout, stderr bytes.Buffer
	err := runInit(context.Background(), &stdout, &stderr, "")

	if !errors.Is(err, agentcatalog.ErrCatalogEmpty) {
		t.Errorf("runInit() error = %v, want agentcatalog.ErrCatalogEmpty", err)
	}
	if stdout.Len() != 0 {
		t.Errorf("runInit() stdout = %q, want empty when catalog fails to load", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Error("runInit() stderr is empty, want an actionable error message")
	}
}

func TestRunInit_GitConfigMissing(t *testing.T) {
	withInitStubs(t,
		func() (*agentcatalog.AgentCatalog, error) { return nil, agentcatalog.ErrGitConfigMissing },
		func(*os.File) bool { return true },
		func([]huh.Option[string]) (string, error) {
			t.Fatal("selectAgent should not be called when no git repository is configured")
			return "", nil
		},
		nil, nil,
	)

	var stdout, stderr bytes.Buffer
	err := runInit(context.Background(), &stdout, &stderr, "")

	if !errors.Is(err, agentcatalog.ErrGitConfigMissing) {
		t.Errorf("runInit() error = %v, want agentcatalog.ErrGitConfigMissing", err)
	}
	if stdout.Len() != 0 {
		t.Errorf("runInit() stdout = %q, want empty when no git repository is configured", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Error("runInit() stderr is empty, want an actionable error message")
	}
}

func TestRunInit_SyncError(t *testing.T) {
	syncErr := errors.New("clone https://example.com/example/repo.git: repository not found")
	withInitStubs(t,
		fakeCatalog(agentcatalog.AgentDefinition{ID: "vscode", DisplayName: "GitHub Copilot"}),
		func(*os.File) bool { return true },
		func([]huh.Option[string]) (string, error) { return "vscode", nil },
		func(context.Context, string, string, string, []string, []string, []string) error { return syncErr },
		fakeGetwd("/tmp/current-dir"),
	)

	var stdout, stderr bytes.Buffer
	err := runInit(context.Background(), &stdout, &stderr, "")

	if !errors.Is(err, syncErr) {
		t.Errorf("runInit() error = %v, want it to wrap %v", err, syncErr)
	}
	if strings.Contains(stdout.String(), "Files written") {
		t.Errorf("runInit() stdout = %q, want no success confirmation on a sync error", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Error("runInit() stderr is empty, want an actionable error message")
	}
	if !strings.Contains(stderr.String(), "Re-run the command") {
		t.Errorf("runInit() stderr = %q, want it to instruct the user to re-run the command", stderr.String())
	}
}

func TestRunInit_MissingDeclaredPathSurfacesToStderr(t *testing.T) {
	syncErr := errors.New("declared path(s) not found in repository: does-not-exist.md")
	withInitStubs(t,
		fakeCatalog(agentcatalog.AgentDefinition{
			ID:          "vscode",
			DisplayName: "GitHub Copilot",
			Files:       []string{"does-not-exist.md"},
		}),
		func(*os.File) bool { return true },
		func([]huh.Option[string]) (string, error) { return "vscode", nil },
		func(context.Context, string, string, string, []string, []string, []string) error { return syncErr },
		fakeGetwd("/tmp/current-dir"),
	)

	var stdout, stderr bytes.Buffer
	err := runInit(context.Background(), &stdout, &stderr, "")

	if !errors.Is(err, syncErr) {
		t.Errorf("runInit() error = %v, want it to wrap %v", err, syncErr)
	}
	if !strings.Contains(stderr.String(), "does-not-exist.md") {
		t.Errorf("runInit() stderr = %q, want it to name the missing declared path", stderr.String())
	}
	if strings.Contains(stdout.String(), "Files written") {
		t.Errorf("runInit() stdout = %q, want no success confirmation when a declared path is missing", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Re-run the command") {
		t.Errorf("runInit() stderr = %q, want it to instruct the user to re-run the command", stderr.String())
	}
}

func fakeCatalogWithManaged(managed agentcatalog.ManagedConfig, agents ...agentcatalog.AgentDefinition) func() (*agentcatalog.AgentCatalog, error) {
	return func() (*agentcatalog.AgentCatalog, error) {
		return &agentcatalog.AgentCatalog{
			Agents: agents,
			Git: agentcatalog.GitConfig{
				Repository: "https://example.com/example/repo.git",
				Ref:        "abc123",
				Managed:    managed,
			},
		}, nil
	}
}

func TestRunInit_ManagedFilesAndFoldersAppliedRegardlessOfSelectedAgent(t *testing.T) {
	managed := agentcatalog.ManagedConfig{
		Files:   []string{"shared.md"},
		Folders: []string{".highway"},
	}
	agents := []agentcatalog.AgentDefinition{
		{ID: "vscode", DisplayName: "GitHub Copilot", Files: []string{"vscode-only.md"}, Folders: []string{"vscode-folder"}},
		{ID: "cursor", DisplayName: "Cursor", Files: []string{"cursor-only.md"}, Folders: []string{"cursor-folder"}},
	}

	runAndCapture := func(selectedID string) (files, folders, managedFolders []string) {
		withInitStubs(t,
			fakeCatalogWithManaged(managed, agents...),
			func(*os.File) bool { return true },
			func([]huh.Option[string]) (string, error) { return selectedID, nil },
			func(_ context.Context, _, _, _ string, f, d, m []string) error {
				files, folders, managedFolders = f, d, m
				return nil
			},
			fakeGetwd("/tmp/current-dir"),
		)
		var stdout, stderr bytes.Buffer
		if err := runInit(context.Background(), &stdout, &stderr, ""); err != nil {
			t.Fatalf("runInit() unexpected error: %v", err)
		}
		return files, folders, managedFolders
	}

	vscodeFiles, vscodeFolders, vscodeManagedFolders := runAndCapture("vscode")
	if !slices.Equal(vscodeFiles, []string{"vscode-only.md", "shared.md"}) {
		t.Errorf("runInit() (vscode) files = %v, want [vscode-only.md shared.md]", vscodeFiles)
	}
	if !slices.Equal(vscodeFolders, []string{"vscode-folder"}) {
		t.Errorf("runInit() (vscode) folders = %v, want [vscode-folder]", vscodeFolders)
	}
	if !slices.Equal(vscodeManagedFolders, []string{".highway"}) {
		t.Errorf("runInit() (vscode) managedFolders = %v, want [.highway]", vscodeManagedFolders)
	}

	cursorFiles, cursorFolders, cursorManagedFolders := runAndCapture("cursor")
	if !slices.Equal(cursorFiles, []string{"cursor-only.md", "shared.md"}) {
		t.Errorf("runInit() (cursor) files = %v, want [cursor-only.md shared.md]", cursorFiles)
	}
	if !slices.Equal(cursorFolders, []string{"cursor-folder"}) {
		t.Errorf("runInit() (cursor) folders = %v, want [cursor-folder]", cursorFolders)
	}
	if !slices.Equal(cursorManagedFolders, []string{".highway"}) {
		t.Errorf("runInit() (cursor) managedFolders = %v, want [.highway]", cursorManagedFolders)
	}
}

func TestRunInit_ManagedFilesUnionedWithAgentFilesNoDuplicateWrites(t *testing.T) {
	var gotFiles []string
	withInitStubs(t,
		fakeCatalogWithManaged(
			agentcatalog.ManagedConfig{Files: []string{"shared.md"}},
			agentcatalog.AgentDefinition{ID: "vscode", DisplayName: "GitHub Copilot", Files: []string{"shared.md", "agent-only.md"}},
		),
		func(*os.File) bool { return true },
		func([]huh.Option[string]) (string, error) { return "vscode", nil },
		func(_ context.Context, _, _, _ string, files, _, _ []string) error {
			gotFiles = files
			return nil
		},
		fakeGetwd("/tmp/current-dir"),
	)

	var stdout, stderr bytes.Buffer
	if err := runInit(context.Background(), &stdout, &stderr, ""); err != nil {
		t.Fatalf("runInit() unexpected error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Errorf("runInit() stderr = %q, want empty", stderr.String())
	}
	if !slices.Contains(gotFiles, "shared.md") {
		t.Errorf("runInit() files = %v, want it to contain the overlapping managed file shared.md", gotFiles)
	}
}

func TestRunInit_NoManagedConfigStillSucceeds(t *testing.T) {
	withInitStubs(t,
		fakeCatalog(agentcatalog.AgentDefinition{ID: "vscode", DisplayName: "GitHub Copilot", Files: []string{"agent-only.md"}}),
		func(*os.File) bool { return true },
		func([]huh.Option[string]) (string, error) { return "vscode", nil },
		noopSync,
		fakeGetwd("/tmp/current-dir"),
	)

	var stdout, stderr bytes.Buffer
	err := runInit(context.Background(), &stdout, &stderr, "")

	if err != nil {
		t.Fatalf("runInit() unexpected error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Errorf("runInit() stderr = %q, want empty on success", stderr.String())
	}
}
