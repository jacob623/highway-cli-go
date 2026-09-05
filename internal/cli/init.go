package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/jacob623/highway-cli-go/internal/agentcatalog"
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

func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Select the AI coding agent to use",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
}

// runInit implements the `init` command contract: load the catalog, verify an
// interactive terminal is available, prompt for a selection, and confirm it.
// The selection is never persisted (FR-011).
func runInit(stdout, stderr io.Writer) error {
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
	for _, agent := range catalog.Agents {
		if agent.ID == selectedID {
			displayName = agent.DisplayName
			break
		}
	}

	fmt.Fprintf(stdout, "Selected agent: %s\n", displayName)
	return nil
}
