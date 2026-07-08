package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const appName = "sava"

// versionFile is the single source of truth for the version, bumped
// by the release PR workflow (.github/workflows/release-pr.yml).
//
//go:embed version.txt
var versionFile string

var version = strings.TrimSpace(versionFile)

func main() {
	root := newRootCommand()
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", appName, err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           appName,
		Short:         "A memo tool to support daily reports",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		newAddCommand(),
		newEndCommand(),
		newDelCommand(),
		newListCommand(),
		newClearCommand(),
	)

	return root
}
