// Package command implements the CLI subcommands. Each function takes
// the log directory and writer/reader explicitly so behavior is
// testable without touching the real home directory.
package command

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/rn404/nippo-cli/internal/index"
	"github.com/rn404/nippo-cli/internal/log"
	"github.com/rn404/nippo-cli/internal/logfile"
	"github.com/rn404/nippo-cli/internal/model"
	"github.com/rn404/nippo-cli/internal/view"
)

const (
	// storagePeriodDays is how long daily logs are kept by clear.
	storagePeriodDays = 30
	// fileStatsLimit is the file count above which list -a -s asks
	// for confirmation before loading everything.
	fileStatsLimit = 10
)

// Add appends a memo to today's log.
func Add(w io.Writer, dir, content string, tags []string) error {
	file, err := logfile.Get(dir, "")
	if err != nil {
		return err
	}
	item, err := log.Add(&file.Body, content, false)
	if err != nil {
		return err
	}
	if len(tags) > 0 {
		item, err = log.AddTags(&file.Body, item.Hash, tags)
		if err != nil {
			return err
		}
	}

	if err := logfile.Update(dir, file.Name, file.Body); err != nil {
		return err
	}
	view.Added(w, item)

	if len(tags) > 0 {
		if _, err := index.Rebuild(dir); err != nil {
			return err
		}
	}
	return nil
}

// TodoOptions controls the todo command behavior.
type TodoOptions struct {
	Start bool     // mark the task as started right away
	Tags  []string // tags to put on the new item
}

// Todo appends a TODO item (task) to today's log.
func Todo(w io.Writer, dir, content string, opts TodoOptions) error {
	file, err := logfile.Get(dir, "")
	if err != nil {
		return err
	}
	item, err := log.Add(&file.Body, content, true)
	if err != nil {
		return err
	}
	if opts.Start {
		item, err = log.Start(&file.Body, item.Hash)
		if err != nil {
			return err
		}
	}
	if len(opts.Tags) > 0 {
		item, err = log.AddTags(&file.Body, item.Hash, opts.Tags)
		if err != nil {
			return err
		}
	}

	if err := logfile.Update(dir, file.Name, file.Body); err != nil {
		return err
	}
	view.Added(w, item)

	if len(opts.Tags) > 0 {
		if _, err := index.Rebuild(dir); err != nil {
			return err
		}
	}
	return nil
}

// Tag adds tags to (or removes them from, when remove is true) the
// item matching hash in today's log, then refreshes the index.
func Tag(w io.Writer, dir, hash string, tags []string, remove bool) error {
	file, err := logfile.Get(dir, "")
	if err != nil {
		return err
	}

	var item model.Item
	if remove {
		item, err = log.RemoveTags(&file.Body, hash, tags)
	} else {
		item, err = log.AddTags(&file.Body, hash, tags)
	}
	if err != nil {
		return err
	}

	if err := logfile.Update(dir, file.Name, file.Body); err != nil {
		return err
	}
	view.TagsUpdated(w, item)

	if _, err := index.Rebuild(dir); err != nil {
		return err
	}
	return nil
}

// TagList prints every known tag with its item count, refreshing the
// index as a side effect.
func TagList(w io.Writer, dir string) error {
	idx, err := index.Rebuild(dir)
	if err != nil {
		return err
	}

	view.Header(w, "Known tags are...")
	if len(idx.Tags) == 0 {
		fmt.Fprintln(w, "There is no tags...")
		return nil
	}

	names := make([]string, 0, len(idx.Tags))
	for name := range idx.Tags {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		view.ListItem(w, fmt.Sprintf("%s (%d)", name, len(idx.Tags[name])))
	}
	return nil
}

// Diff prints the elapsed time between the creation of two items,
// resolving each hash across all daily logs via the index. The index
// is rebuilt once when a hash is not found, so stale cache entries
// heal themselves.
func Diff(w io.Writer, dir, hashA, hashB string) error {
	idx, err := index.Load(dir)
	if err != nil {
		return err
	}

	rebuilt := false
	resolve := func(hash string) (model.Item, error) {
		for {
			if item, ok := lookup(dir, idx, hash); ok {
				return item, nil
			}
			if rebuilt {
				return model.Item{}, fmt.Errorf("target item %q is not found", hash)
			}
			idx, err = index.Rebuild(dir)
			if err != nil {
				return model.Item{}, err
			}
			rebuilt = true
		}
	}

	itemA, err := resolve(hashA)
	if err != nil {
		return err
	}
	itemB, err := resolve(hashB)
	if err != nil {
		return err
	}

	elapsed, err := elapsedBetween(itemA, itemB)
	if err != nil {
		return err
	}

	view.Diff(w, itemA, itemB, elapsed)
	return nil
}

// lookup finds the item behind hash using the index. A stale entry
// (missing file or hash no longer in it) reports a miss.
func lookup(dir string, idx index.Index, hash string) (model.Item, bool) {
	date, ok := idx.Hashes[hash]
	if !ok {
		return model.Item{}, false
	}

	file, err := logfile.Stat(dir, date)
	if err != nil {
		return model.Item{}, false
	}
	for _, item := range file.Body.Items {
		if item.Hash == hash {
			return item, true
		}
	}
	return model.Item{}, false
}

// elapsedBetween returns the absolute distance between the creation
// times of two items.
func elapsedBetween(a, b model.Item) (time.Duration, error) {
	createdA, err := time.Parse(time.RFC3339, a.CreatedAt)
	if err != nil {
		return 0, fmt.Errorf("broken createdAt on item %q: %w", a.Hash, err)
	}
	createdB, err := time.Parse(time.RFC3339, b.CreatedAt)
	if err != nil {
		return 0, fmt.Errorf("broken createdAt on item %q: %w", b.Hash, err)
	}

	elapsed := createdB.Sub(createdA)
	if elapsed < 0 {
		elapsed = -elapsed
	}
	return elapsed, nil
}

// Start marks the task matching hash in today's log as started.
func Start(w io.Writer, dir, hash string) error {
	file, err := logfile.Get(dir, "")
	if err != nil {
		return err
	}

	started, err := log.Start(&file.Body, hash)
	if err != nil {
		return err
	}

	if err := logfile.Update(dir, file.Name, file.Body); err != nil {
		return err
	}

	view.StartedTask(w, started)
	return nil
}

// End closes the tasks matching hashes in today's log. Duplicate
// hashes are collapsed to one. All hashes must resolve to open tasks
// or none of them are persisted.
func End(w io.Writer, dir string, hashes []string) error {
	file, err := logfile.Get(dir, "")
	if err != nil {
		return err
	}

	finished := make([]model.Item, 0, len(hashes))
	for _, hash := range dedupe(hashes) {
		item, err := log.Finish(&file.Body, hash)
		if err != nil {
			return err
		}
		finished = append(finished, item)
	}

	if err := logfile.Update(dir, file.Name, file.Body); err != nil {
		return err
	}

	for _, item := range finished {
		view.FinishedTask(w, item)
	}
	return nil
}

// dedupe returns hashes with repeats removed, keeping first occurrence order.
func dedupe(hashes []string) []string {
	seen := make(map[string]bool, len(hashes))
	out := make([]string, 0, len(hashes))
	for _, hash := range hashes {
		if !seen[hash] {
			seen[hash] = true
			out = append(out, hash)
		}
	}
	return out
}

// Del removes the item matching hash from today's log.
func Del(w io.Writer, dir, hash string) error {
	file, err := logfile.Get(dir, "")
	if err != nil {
		return err
	}

	item, ok := findItem(file.Body, hash)
	if !ok {
		return fmt.Errorf("target item %q is not found", hash)
	}

	log.Delete(&file.Body, hash)
	if err := logfile.Update(dir, file.Name, file.Body); err != nil {
		return err
	}

	view.Deleted(w, item)
	return nil
}

// findItem returns the item matching hash in l, if any.
func findItem(l model.Log, hash string) (model.Item, bool) {
	for _, item := range l.Items {
		if item.Hash == hash {
			return item, true
		}
	}
	return model.Item{}, false
}

// ListOptions controls the list command behavior.
type ListOptions struct {
	Date string // yyyy-MM-dd; empty means today
	All  bool
	Stat bool
	Yes  bool     // skip confirmation prompts
	Tags []string // show only items carrying the tags
	Or   bool     // match any tag instead of all
}

// List shows the items of one day, or summaries across all log files.
func List(w io.Writer, r io.Reader, dir string, opts ListOptions) error {
	if !opts.All {
		return listOneDay(w, dir, opts)
	}
	if len(opts.Tags) > 0 {
		return errors.New("tag filter cannot be combined with --all")
	}

	refs, err := logfile.List(dir)
	if err != nil {
		return err
	}

	if !opts.Stat {
		view.Header(w, "The logs here are...")
		for _, ref := range refs {
			view.ListItem(w, ref.Name)
		}
		return nil
	}

	view.Header(w, fmt.Sprintf("View all log statistics. There are %d total.", len(refs)))
	if len(refs) > fileStatsLimit && !opts.Yes {
		message := fmt.Sprintf(
			"There are more than %d log files. It may take some time to display all of them. Are you sure you want to view them?",
			fileStatsLimit,
		)
		if !confirm(w, r, message) {
			return nil
		}
	}

	for _, ref := range refs {
		file, err := logfile.Get(dir, ref.Name)
		if err != nil {
			return err
		}
		writeFileStat(w, file)
	}
	return nil
}

func listOneDay(w io.Writer, dir string, opts ListOptions) error {
	file, err := logfile.Stat(dir, opts.Date)
	if err != nil && !errors.Is(err, logfile.ErrNotFound) {
		return err
	}

	if opts.Stat {
		if opts.Date == "" {
			view.Header(w, "Today's log stats are...")
		} else {
			view.Header(w, fmt.Sprintf("Log stats for %s are...", opts.Date))
		}
		if file == nil {
			fmt.Fprintln(w, "There is no body...")
			return nil
		}
		writeFileStat(w, file)
		return nil
	}

	if opts.Date == "" {
		view.Header(w, "Today's logs are...")
	} else {
		view.Header(w, fmt.Sprintf("Log for %s are...", opts.Date))
	}
	if file == nil {
		fmt.Fprintln(w, "There is no body...")
		return nil
	}

	tasks, memos := log.Split(file.Body)
	if len(opts.Tags) > 0 {
		tasks = log.FilterByTags(tasks, opts.Tags, opts.Or)
		memos = log.FilterByTags(memos, opts.Tags, opts.Or)
	}
	view.ItemList(w, tasks, memos)
	return nil
}

func writeFileStat(w io.Writer, file *logfile.LogFile) {
	tasks, memos := log.Split(file.Body)
	view.FileStat(w, file.Name, file.Body.Freezed, tasks, memos, log.CountUnfinished(tasks))
}

// Clear deletes old log files, or all of them when all is true.
func Clear(w io.Writer, r io.Reader, dir string, all, yes bool) error {
	if all {
		return clearAll(w, r, dir, yes)
	}
	return clearOld(w, dir)
}

func clearAll(w io.Writer, r io.Reader, dir string, yes bool) error {
	if !yes && !confirm(w, r, "Do you want to delete all the files?") {
		return nil
	}

	refs, err := logfile.List(dir)
	if err != nil {
		return err
	}
	if len(refs) == 0 {
		fmt.Fprintln(w, "There is no log files.")
		return nil
	}

	for _, ref := range refs {
		if err := logfile.Remove(ref); err != nil {
			return err
		}
		fmt.Fprintf(w, "Deleted... %s logs.\n", ref.Name)
	}

	fmt.Fprintln(w, "Deleted all files.")
	return index.Remove(dir)
}

func clearOld(w io.Writer, dir string) error {
	view.Header(w, fmt.Sprintf("Delete logs that are past their storage period. ( Storage period: %d days )", storagePeriodDays))

	refs, err := logfile.List(dir)
	if err != nil {
		return err
	}

	deadline := time.Now().AddDate(0, 0, -storagePeriodDays)
	for _, ref := range refs {
		date, err := model.ParseDate(ref.Name)
		if err != nil {
			continue
		}
		if date.Before(deadline) {
			if err := logfile.Remove(ref); err != nil {
				return err
			}
			fmt.Fprintf(w, "Deleted... %s logs.\n", ref.Name)
		}
	}
	return nil
}

// confirm asks a yes/no question and returns true only on an explicit
// yes. Any read failure (e.g. closed stdin) counts as no.
func confirm(w io.Writer, r io.Reader, message string) bool {
	fmt.Fprintf(w, "%s [y/N] ", message)

	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && line == "" {
		fmt.Fprintln(w)
		return false
	}

	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
