package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rn404/nippo-cli/internal/command"
	"github.com/rn404/nippo-cli/internal/logfile"
)

func newAddCommand() *cobra.Command {
	opts := command.AddOptions{}
	cmd := &cobra.Command{
		Use:   "add <contents>",
		Short: "Add contents to nippo log.",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return command.Add(logfile.Dir(), args[0], opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.Memo, "memo", "m", false, "Add contents like memo item.")
	cmd.Flags().BoolVarP(&opts.Start, "start", "s", false, "start the task right away")
	cmd.Flags().StringSliceVarP(&opts.Tags, "tag", "t", nil, "put tags on the new item")
	cmd.MarkFlagsMutuallyExclusive("memo", "start")
	return cmd
}

func newTagCommand() *cobra.Command {
	var remove, list bool
	cmd := &cobra.Command{
		Use:   "tag <hash> <tag>...",
		Short: "manage tags of an item.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if list {
				if len(args) != 0 {
					return fmt.Errorf("--list takes no arguments")
				}
				return command.TagList(cmd.OutOrStdout(), logfile.Dir())
			}
			if len(args) < 2 {
				return fmt.Errorf("requires <hash> and at least one <tag>")
			}
			return command.Tag(cmd.OutOrStdout(), logfile.Dir(), args[0], args[1:], remove)
		},
	}
	cmd.Flags().BoolVarP(&remove, "delete", "d", false, "remove the tags instead of adding")
	cmd.Flags().BoolVarP(&list, "list", "l", false, "list all known tags")
	cmd.MarkFlagsMutuallyExclusive("delete", "list")
	return cmd
}

func newStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start <hash>",
		Short: "start to task.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return command.Start(cmd.OutOrStdout(), logfile.Dir(), args[0])
		},
	}
}

func newEndCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "end <hash>",
		Short: "end to task.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return command.End(cmd.OutOrStdout(), logfile.Dir(), args[0])
		},
	}
}

func newDelCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "del <hash>",
		Short: "delete task.",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return command.Del(logfile.Dir(), args[0])
		},
	}
}

func newDiffCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <hashA>...<hashB>",
		Short: "show elapsed time between two items.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			hashA, hashB, err := splitDiffArgs(args)
			if err != nil {
				return err
			}
			return command.Diff(cmd.OutOrStdout(), logfile.Dir(), hashA, hashB)
		},
	}
}

// splitDiffArgs accepts either "<hashA>...<hashB>" (also "..") as one
// argument or two separate hash arguments.
func splitDiffArgs(args []string) (string, string, error) {
	if len(args) == 2 {
		return args[0], args[1], nil
	}
	for _, sep := range []string{"...", ".."} {
		parts := strings.SplitN(args[0], sep, 2)
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return parts[0], parts[1], nil
		}
	}
	return "", "", fmt.Errorf("expected <hashA>...<hashB> or two hashes")
}

func newListCommand() *cobra.Command {
	opts := command.ListOptions{}
	cmd := &cobra.Command{
		Use:   "list [date]",
		Short: "list all logs.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				opts.Date = args[0]
			}
			return command.List(cmd.OutOrStdout(), cmd.InOrStdin(), logfile.Dir(), opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.All, "all", "a", false, "show all logs")
	cmd.Flags().BoolVarP(&opts.Stat, "stat", "s", false, "show summary of list")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "skip confirmation prompts")
	cmd.Flags().StringSliceVarP(&opts.Tags, "tag", "t", nil, "show only items carrying the tags")
	cmd.Flags().BoolVar(&opts.Or, "or", false, "match any tag instead of all")
	return cmd
}

func newClearCommand() *cobra.Command {
	var all, yes bool
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "delete log",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return command.Clear(cmd.OutOrStdout(), cmd.InOrStdin(), logfile.Dir(), all, yes)
		},
	}
	cmd.Flags().BoolVarP(&all, "all", "a", false, "clear all logs")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompts")
	return cmd
}
