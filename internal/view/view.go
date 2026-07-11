// Package view renders command output. The layout follows the former
// Deno implementation; wording equivalence is functional, not
// character-exact.
package view

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/rn404/nippo-cli/internal/model"
)

const bullet = "-"

// Header prints a section title surrounded by blank space.
func Header(w io.Writer, title string) {
	fmt.Fprintf(w, "\n    %s\n\n", title)
}

// ItemList prints tasks and memos grouped with headers.
func ItemList(w io.Writer, tasks, memos []model.Item) {
	if len(tasks) == 0 && len(memos) == 0 {
		fmt.Fprintln(w, "There is no body...")
		return
	}

	if len(tasks) > 0 {
		fmt.Fprintln(w, "Task ->")
		for _, item := range tasks {
			checkbox := "[ ]"
			switch {
			case item.IsClosed():
				checkbox = "[x]"
			case item.IsStarted():
				checkbox = "[>]"
			}
			fmt.Fprintf(w, "%s %s %s (%s) %s%s\n", bullet, checkbox, item.Content, formatTime(item.CreatedAt), item.Hash, formatTags(item.Tags))
		}
	}

	if len(tasks) > 0 && len(memos) > 0 {
		fmt.Fprintln(w)
	}

	if len(memos) > 0 {
		fmt.Fprintln(w, "Memo ->")
		for _, item := range memos {
			fmt.Fprintf(w, "%s %s (%s) %s%s\n", bullet, item.Content, formatTime(item.CreatedAt), item.Hash, formatTags(item.Tags))
		}
	}
}

// FinishedTask prints the closed task confirmation.
func FinishedTask(w io.Writer, item model.Item) {
	fmt.Fprintln(w, "Finished!!")
	fmt.Fprintf(w, "> %s (%s)\n", item.Content, formatTime(item.CreatedAt))
}

// StartedTask prints the started task confirmation.
func StartedTask(w io.Writer, item model.Item) {
	fmt.Fprintln(w, "Started!!")
	fmt.Fprintf(w, "> %s (%s)\n", item.Content, formatTime(item.CreatedAt))
}

// TagsUpdated prints the item's tags after a tag change.
func TagsUpdated(w io.Writer, item model.Item) {
	fmt.Fprintln(w, "Tags updated!!")
	fmt.Fprintf(w, "> %s (%s)%s\n", item.Content, formatTime(item.CreatedAt), formatTags(item.Tags))
}

// FileStat prints a one-line summary of a daily log file.
func FileStat(w io.Writer, name string, freezed bool, tasks, memos []model.Item, unfinished int) {
	freezedMark := " "
	if freezed {
		freezedMark = "*"
	}
	fmt.Fprintf(w, "%s %s%s Task: %d (unfinished: %d), Memo: %d\n",
		bullet, name, freezedMark, len(tasks), unfinished, len(memos))
}

// ListItem prints a single bullet line.
func ListItem(w io.Writer, message string) {
	fmt.Fprintf(w, "%s %s\n", bullet, message)
}

// formatTags renders tags as " #a #b", or "" when there are none.
func formatTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	var b strings.Builder
	for _, tag := range tags {
		b.WriteString(" #")
		b.WriteString(tag)
	}
	return b.String()
}

// formatTime renders an ISO timestamp as local HH:mm.
func formatTime(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	return t.Local().Format("15:04")
}
