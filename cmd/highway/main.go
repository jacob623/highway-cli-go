// Command highway is the entrypoint for the Highway CLI.
package main

import (
	"os"

	"github.com/jacob623/highway-cli-go/internal/cli"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
