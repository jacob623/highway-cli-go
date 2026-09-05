package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
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
	sync func(context.Context, string, string, string) error,
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

func noopSync(context.Context, string, string, string) error { return nil }

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
		func(_ context.Context, _, _, destination string) error {
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
		func(context.Context, string, string, string) error { return context.Canceled },
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
		func(context.Context, string, string, string) error { return syncErr },
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
}
