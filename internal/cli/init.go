package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/jacob623/highway-cli-go/internal/agentcatalog"
	"github.com/jacob623/highway-cli-go/internal/reposync"
)

// loadCatalog is overridable in tests to avoid depending on the embedded catalog file.
var loadCatalog = agentcatalog.Load

// isTerminal is overridable in tests; reports whether f is an interactive terminal.
var isTerminal = func(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// selectAgent is overridable in tests; prompts the developer to choose one option.
var selectAgent = func(options []huh.Option[string]) (string, error) {
	var selectedID string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select your AI coding agent").
				Options(options...).
				Value(&selectedID),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	return selectedID, nil
}

// syncRepo is overridable in tests; clones the configured repository/ref and
// writes its files to destination, scoped to the selected agent's declared
// files/folders plus the git configuration's managed files/folders (applied
// regardless of the selected agent) and seeded files/folders (created only if
// missing at destination, also regardless of the selected agent) — see reposync.Sync.
var syncRepo = reposync.Sync

// getwd is overridable in tests; reports the current working directory.
var getwd = os.Getwd

func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init [path]",
		Short: "Select the AI coding agent to use",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var path string
			if len(args) == 1 {
				path = args[0]
			}
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
			defer cancel()
			return runInit(ctx, cmd.OutOrStdout(), cmd.ErrOrStderr(), path)
		},
	}
}

// runInit implements the `init` command contract: load the catalog, verify an
// interactive terminal is available, prompt for a selection, then clone the
// configured git repository/commit and write its files to the destination
// (path if supplied, otherwise the current working directory). The agent
// selection itself is never persisted (FR-011 covers the destination path).
func runInit(ctx context.Context, stdout, stderr io.Writer, path string) error {
	catalog, err := loadCatalog()
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return err
	}

	if !isTerminal(os.Stdin) {
		err := errors.New("no interactive terminal available to select an agent")
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return err
	}

	options := make([]huh.Option[string], 0, len(catalog.Agents))
	for _, agent := range catalog.Agents {
		options = append(options, huh.NewOption(agent.DisplayName, agent.ID))
	}

	selectedID, err := selectAgent(options)
	if err != nil {
		// Cancellation (e.g. Ctrl+C) or any other prompt failure: no confirmation, non-zero exit.
		return err
	}

	displayName := selectedID
	var files, folders []string
	for _, agent := range catalog.Agents {
		if agent.ID == selectedID {
			displayName = agent.DisplayName
			files = agent.Files
			folders = agent.Folders
			break
		}
	}

	// Managed files apply on every init run regardless of the selected agent (FR-002);
	// unioning them with the agent's own declared files reuses the existing write path
	// unchanged (FR-005). Cloning avoids mutating the catalog's own agent.Files slice.
	files = append(append([]string{}, files...), catalog.Git.Managed.Files...)

	// Seeded files/folders apply regardless of selected agent too (FR-002), but stay on
	// their own parameters since create-if-missing semantics differ from the
	// always-overwrite files/folders/managedFolders above.
	seededFiles := catalog.Git.Seeded.Files
	seededFolders := catalog.Git.Seeded.Folders

	fmt.Fprintf(stdout, "Selected agent: %s\n", displayName)

	destination := path
	if destination == "" {
		destination, err = getwd()
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return err
		}
	}

	if err := syncRepo(ctx, catalog.Git.Repository, catalog.Git.Ref, destination, files, folders, catalog.Git.Managed.Folders, seededFiles, seededFolders); err != nil {
		if errors.Is(err, context.Canceled) {
			err = fmt.Errorf("cancelled before files were written: %w", err)
		}
		fmt.Fprintf(stderr, "Error: %v\n", err)
		fmt.Fprintln(stderr, "Re-run the command to retry.")
		return err
	}

	fmt.Fprintf(stdout, "Files written to: %s\n", destination)
	return nil
}
