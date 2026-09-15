package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rn404/nippo-cli/internal/command"
	"github.com/rn404/nippo-cli/internal/editor"
	"github.com/rn404/nippo-cli/internal/logfile"
)

func newAddCommand() *cobra.Command {
	opts := command.AddOptions{}
	cmd := &cobra.Command{
		Use:   "add [contents]",
		Short: "Add a memo to nippo log.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			content, ok, err := resolveNewContent(args)
			if err != nil {
				return err
			}
			if !ok {
				command.AddAborted(w)
				return nil
			}
			return command.Add(w, logfile.Dir(), content, opts)
		},
	}
	cmd.Flags().StringSliceVarP(&opts.Tags, "tag", "t", nil, "put tags on the new item")
	return cmd
}

// resolveNewContent resolves the content to create a new add/todo item
// with. When args already supplies it directly (len(args) == 1), ok is
// always true and content is returned unvalidated -- the existing
// empty-content rejection for direct mode happens downstream in
// command.Add/command.Todo (via log.Add), unchanged by this function.
//
// When args is empty, it launches the editor via editor.Resolve with
// an empty initial content. ok is false when the editor left it
// unchanged (still empty, editor.Resolve's own judgment) or when the
// saved content is whitespace-only: editor.Resolve's own emptiness
// check only catches an exact "" (its current is always "" here, so
// even a single space differs from both), so the whitespace-only case
// is caught here instead, to stay consistent with log.Add's trim-based
// validation without surfacing it as a hard error from the editor
// path.
func resolveNewContent(args []string) (content string, ok bool, err error) {
	if len(args) == 1 {
		return args[0], true, nil
	}

	content, ok, err = editor.Resolve("", runEditor)
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}
	if strings.TrimSpace(content) == "" {
		return "", false, nil
	}
	return content, true, nil
}

// newTodoCommand and newAddCommand must never gain subcommands: cobra
// resolves a matching child command name before falling back to
// <contents>, so a subcommand named e.g. "start" would make it
// impossible to create an item whose content is literally "start".
func newTodoCommand() *cobra.Command {
	opts := command.TodoOptions{}
	cmd := &cobra.Command{
		Use:   "todo <contents>",
		Short: "Add a TODO item to nippo log.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return command.Todo(cmd.OutOrStdout(), logfile.Dir(), args[0], opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.Start, "start", "s", false, "start the task right away")
	cmd.Flags().StringSliceVarP(&opts.Tags, "tag", "t", nil, "put tags on the new item")
	return cmd
}

func newStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start <hash>",
		Short: "start an existing TODO item.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return command.Start(cmd.OutOrStdout(), logfile.Dir(), args[0])
		},
	}
}

func newEndCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "end <hash>...",
		Short: "finish one or more TODO items.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return command.End(cmd.OutOrStdout(), logfile.Dir(), args)
		},
	}
}

func newEditCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <hash> [new content]",
		Short: "edit the content of an existing item.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := logfile.Dir()
			w := cmd.OutOrStdout()

			if len(args) == 2 {
				return command.Edit(w, dir, args[0], args[1])
			}

			item, err := command.TodayItem(dir, args[0])
			if err != nil {
				return err
			}

			content, ok, err := editor.Resolve(item.Content, runEditor)
			if err != nil {
				return err
			}
			if !ok {
				command.EditAborted(w)
				return nil
			}
			return command.Edit(w, dir, args[0], content)
		},
	}
}

// runEditor is the production launch function injected into
// editor.Resolve: it execs the named editor against path with the
// real terminal's stdin/stdout/stderr attached.
func runEditor(name, path string) error {
	cmd := exec.Command(name, path) //nolint:gosec // name comes from the user's own $EDITOR/$VISUAL (or "vi"), the same trust boundary as `git commit` invoking $EDITOR; path is our own temp file
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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

func newDelCommand() *cobra.Command {
	var deep bool
	cmd := &cobra.Command{
		Use:   "del <hash>|<date>:<hash>",
		Short: "delete item.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return command.Del(cmd.OutOrStdout(), logfile.Dir(), args[0], deep)
		},
	}
	cmd.Flags().BoolVar(&deep, "deep", false, "search all logs instead of just the last 30 days")
	return cmd
}

func newDiffCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <date>:<hashA>...<date>:<hashB>",
		Short: "show elapsed time between two items.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			refA, refB, err := splitDiffArgs(args)
			if err != nil {
				return err
			}
			return command.Diff(cmd.OutOrStdout(), logfile.Dir(), refA, refB)
		},
	}
}

// splitDiffArgs accepts either "<refA>...<refB>" (also "..") as one
// argument or two separate ref arguments.
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
	return "", "", fmt.Errorf("expected <refA>...<refB> or two refs")
}

func newListCommand() *cobra.Command {
	opts := command.ListOptions{}
	cmd := &cobra.Command{
		Use:   "list [date|yesterday]",
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
	cmd.Flags().BoolVarP(&opts.Full, "full", "f", false, "include completed tasks")
	cmd.Flags().BoolVar(&opts.TasksOnly, "task", false, "show only tasks (exclude memos)")
	cmd.Flags().BoolVar(&opts.FullText, "full-text", false, "show full content instead of the first line only")
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
