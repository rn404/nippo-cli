// Package view renders command output.
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

// Timeline prints items (tasks and memos mixed) as a single list in
// the order given: a GFM checklist prefix (or a plain bullet for
// memos), the creation time, the content, and the hash/tags for
// reference. The hash is always wrapped as "(`hash`)" so it can be
// extracted from any line with the same pattern. When fullText is
// false, multi-line content is shown as its first line only, so one
// line always maps to one item; when true, content is shown in full.
func Timeline(w io.Writer, items []model.Item, fullText bool) {
	if len(items) == 0 {
		fmt.Fprintln(w, "There is no body...")
		return
	}

	for _, item := range items {
		content := item.Content
		if !fullText {
			content = firstLine(content)
		}
		fmt.Fprintf(w, "%s %s %s (`%s`)%s\n", checklistPrefix(item.Status()), formatTime(item.CreatedAt), content, item.Hash, formatTags(item.Tags))
	}
}

// checklistPrefix renders the leading bullet and, for tasks, GFM
// checklist syntax for an item's lifecycle status: "[ ]" for both
// open and started (GFM has no third checkbox state), with a
// literal "`in-progress`" token distinguishing started from open;
// "[x]" for closed; no checkbox at all for memos.
func checklistPrefix(status model.Status) string {
	switch status {
	case model.StatusClosed:
		return bullet + " [x]"
	case model.StatusStarted:
		return bullet + " [ ] `in-progress`"
	case model.StatusOpen:
		return bullet + " [ ]"
	default:
		return bullet
	}
}

// firstLine returns content's first line, or content itself if it
// has no newline.
func firstLine(content string) string {
	line, _, _ := strings.Cut(content, "\n")
	return line
}

// Carried prints the automatic-carry notice, ahead of whatever output
// the write command that triggered it goes on to print, so a carry
// never happens invisibly.
func Carried(w io.Writer, count int, sourceDate string) {
	fmt.Fprintf(w, "Carried %d items from %s (that day is now frozen).\n", count, sourceDate)
}

// Added prints the newly created item confirmation, including its
// hash so it can be used right away without a separate list call.
func Added(w io.Writer, item model.Item) {
	fmt.Fprintln(w, "Added!!")
	fmt.Fprintf(w, "> %s (%s) %s%s\n", item.Content, formatTime(item.CreatedAt), item.Hash, formatTags(item.Tags))
}

// Deleted prints the deleted item confirmation.
func Deleted(w io.Writer, item model.Item) {
	confirmItem(w, "Deleted!!", item)
}

// FinishedTask prints the closed task confirmation.
func FinishedTask(w io.Writer, item model.Item) {
	confirmItem(w, "Finished!!", item)
}

// StartedTask prints the started task confirmation.
func StartedTask(w io.Writer, item model.Item) {
	confirmItem(w, "Started!!", item)
}

// confirmItem prints a "<label>" header followed by the item's
// content and time. Shared by Deleted, FinishedTask, and StartedTask.
func confirmItem(w io.Writer, label string, item model.Item) {
	fmt.Fprintln(w, label)
	fmt.Fprintf(w, "> %s (%s)\n", item.Content, formatTime(item.CreatedAt))
}

// Edited prints the edited item confirmation.
func Edited(w io.Writer, item model.Item) {
	confirmItem(w, "Edited!!", item)
}

// EditAborted prints a notice that an edit was aborted (the editor
// was saved with no changes, or an empty content) and no content was
// rewritten.
func EditAborted(w io.Writer) {
	fmt.Fprintln(w, "Edit aborted: no changes to save.")
}

// AddAborted prints a notice that an add was aborted (the editor was
// saved with empty, or whitespace-only, content) and no memo was
// created.
func AddAborted(w io.Writer) {
	fmt.Fprintln(w, "Add aborted: nothing to save.")
}

// TodoAborted prints a notice that a todo was aborted (the editor was
// saved with empty, or whitespace-only, content) and no task was
// created.
func TodoAborted(w io.Writer) {
	fmt.Fprintln(w, "Todo aborted: nothing to save.")
}

// TagsUpdated prints the item's tags after a tag change.
func TagsUpdated(w io.Writer, item model.Item) {
	fmt.Fprintln(w, "Tags updated!!")
	fmt.Fprintf(w, "> %s (%s)%s\n", item.Content, formatTime(item.CreatedAt), formatTags(item.Tags))
}

// Diff prints two items with full creation dates and the elapsed time
// between them.
func Diff(w io.Writer, a, b model.Item, elapsed time.Duration) {
	fmt.Fprintln(w, "Diff...")
	fmt.Fprintf(w, "> %s (%s) %s\n", a.Content, formatDateTime(a.CreatedAt), a.Hash)
	fmt.Fprintf(w, "> %s (%s) %s\n", b.Content, formatDateTime(b.CreatedAt), b.Hash)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Elapsed: %s\n", formatDuration(elapsed))
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

// formatDateTime renders an ISO timestamp as local yyyy-MM-dd HH:mm.
func formatDateTime(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	return t.Local().Format("2006-01-02 15:04")
}

// formatDuration renders a duration as its non-zero components, e.g.
// "1d 2h 30m", "45m 10s" or "0s".
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d < 0 {
		d = -d
	}

	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute
	seconds := (d - minutes*time.Minute) / time.Second

	var parts []string
	for _, part := range []struct {
		value int64
		unit  string
	}{
		{int64(days), "d"},
		{int64(hours), "h"},
		{int64(minutes), "m"},
		{int64(seconds), "s"},
	} {
		if part.value > 0 {
			parts = append(parts, fmt.Sprintf("%d%s", part.value, part.unit))
		}
	}
	if len(parts) == 0 {
		return "0s"
	}
	return strings.Join(parts, " ")
}
