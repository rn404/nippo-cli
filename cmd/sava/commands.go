package main

import (
	"errors"

	"github.com/spf13/cobra"
)

var errNotImplemented = errors.New("not implemented yet")

func newAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <contents>",
		Short: "Add contents to nippo log.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return errNotImplemented
		},
	}
	cmd.Flags().BoolP("memo", "m", false, "Add contents like memo item.")
	return cmd
}

func newEndCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "end <hash>",
		Short: "end to task.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return errNotImplemented
		},
	}
}

func newDelCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "del <hash>",
		Short: "delete task.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return errNotImplemented
		},
	}
}

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [date]",
		Short: "list all logs.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return errNotImplemented
		},
	}
	cmd.Flags().BoolP("all", "a", false, "show all logs")
	cmd.Flags().BoolP("stat", "s", false, "show summary of list")
	return cmd
}

func newClearCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "delete log",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errNotImplemented
		},
	}
	cmd.Flags().BoolP("all", "a", false, "clear all logs")
	cmd.Flags().BoolP("yes", "y", false, "skip confirmation prompts")
	return cmd
}
