// Package cli wires together the highway CLI's cobra command tree.
package cli

import "github.com/spf13/cobra"

// NewRootCommand constructs the highway CLI's root cobra command.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "highway",
		Short:         "Highway CLI",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.AddCommand(newInitCommand())
	return root
}
