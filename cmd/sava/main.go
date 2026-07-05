package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const appName = "sava"

// version is overridable at build time:
// go build -ldflags "-X main.version=vX.Y.Z"
var version = "v0.1.0"

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
