package main

import (
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
	cmd.MarkFlagsMutuallyExclusive("memo", "start")
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
