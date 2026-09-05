package cli

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/huh"

	"github.com/jacob623/highway-cli-go/internal/agentcatalog"
)

func withInitStubs(t *testing.T, load func() (*agentcatalog.AgentCatalog, error), terminal func(*os.File) bool, sel func([]huh.Option[string]) (string, error)) {
	t.Helper()
	origLoad, origTerminal, origSelect := loadCatalog, isTerminal, selectAgent
	t.Cleanup(func() {
		loadCatalog, isTerminal, selectAgent = origLoad, origTerminal, origSelect
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
}

func fakeCatalog(agents ...agentcatalog.AgentDefinition) func() (*agentcatalog.AgentCatalog, error) {
	return func() (*agentcatalog.AgentCatalog, error) {
		return &agentcatalog.AgentCatalog{Agents: agents}, nil
	}
}

func TestRunInit_NonInteractive(t *testing.T) {
	withInitStubs(t,
		fakeCatalog(agentcatalog.AgentDefinition{ID: "vscode", DisplayName: "GitHub Copilot"}),
		func(*os.File) bool { return false },
		nil,
	)

	var stdout, stderr bytes.Buffer
	err := runInit(&stdout, &stderr)

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
	)

	var stdout, stderr bytes.Buffer
	err := runInit(&stdout, &stderr)

	if err != nil {
		t.Fatalf("runInit() unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Cursor") {
		t.Errorf("runInit() stdout = %q, want confirmation naming Cursor", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("runInit() stderr = %q, want empty", stderr.String())
	}
}

func TestRunInit_Cancellation(t *testing.T) {
	withInitStubs(t,
		fakeCatalog(agentcatalog.AgentDefinition{ID: "vscode", DisplayName: "GitHub Copilot"}),
		func(*os.File) bool { return true },
		func([]huh.Option[string]) (string, error) { return "", huh.ErrUserAborted },
	)

	var stdout, stderr bytes.Buffer
	err := runInit(&stdout, &stderr)

	if !errors.Is(err, huh.ErrUserAborted) {
		t.Errorf("runInit() error = %v, want huh.ErrUserAborted", err)
	}
	if stdout.Len() != 0 {
		t.Errorf("runInit() stdout = %q, want empty on cancellation", stdout.String())
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
	)

	var stdout, stderr bytes.Buffer
	err := runInit(&stdout, &stderr)

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
