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

// AddOptions controls the add command behavior.
type AddOptions struct {
	Tags []string // tags to put on the new item
}

// Add appends a memo to today's log.
func Add(w io.Writer, dir, content string, opts AddOptions) error {
	file, err := ensureToday(w, dir)
	if err != nil {
		return err
	}
	item, err := log.Add(&file.Body, content, false)
	if err != nil {
		return err
	}
	// AddTags is a no-op on empty tags, so this is safe to call
	// unconditionally.
	item, err = log.AddTags(&file.Body, item.Hash, opts.Tags)
	if err != nil {
		return err
	}
	return persistItem(w, dir, file, item, len(opts.Tags) > 0, view.Added)
}

// persistItem writes file, reports item to w via confirm, and
// rebuilds the tag index when tagsChanged. tagsChanged must be true
// whenever tags were added OR removed, even if the item ends up with
// zero tags: removing the last tag still leaves a stale index entry
// that needs purging, so "item has tags now" is not a safe substitute
// for "this call touched tags." Shared by Add, Todo, and Tag.
func persistItem(w io.Writer, dir string, file *logfile.LogFile, item model.Item, tagsChanged bool, confirm func(io.Writer, model.Item)) error {
	if err := logfile.Update(dir, file.Name, file.Body); err != nil {
		return err
	}
	confirm(w, item)

	if tagsChanged {
		if _, err := index.Rebuild(dir); err != nil {
			return err
		}
	}
	return nil
}

// ensureToday returns today's log file, running the automatic carry
// engine first if today has not been written to yet. Carry copies
// every unfinished TODO on the most recent existing log day before
// today forward into today's (about to be created) log with fresh
// hashes, then freezes that source day so it is never carried again.
// A day that is already frozen, or no prior day at all, leaves today
// as a plain empty log, same as before Phase C. Once today's file
// exists, every later call within the same day just returns it as-is:
// the file's existence is itself the one-carry-per-day guarantee, so
// this never re-runs carry.
func ensureToday(w io.Writer, dir string) (*logfile.LogFile, error) {
	if today, err := logfile.Stat(dir, ""); err == nil {
		return today, nil
	} else if !errors.Is(err, logfile.ErrNotFound) {
		return nil, err
	}

	body := model.NewLog()
	carried, sourceDate, err := carryFromPreviousDay(dir)
	if err != nil {
		return nil, err
	}
	if len(carried) > 0 {
		body.Items = carried
		view.Carried(w, len(carried), sourceDate)
	}

	if err := logfile.Update(dir, "", body); err != nil {
		return nil, err
	}
	return logfile.Get(dir, "")
}

// carryFromPreviousDay finds the most recent existing log day strictly
// before today, copies its unfinished TODOs forward, and freezes that
// day so it becomes permanently read-only. It is a no-op (nil items,
// empty sourceDate) when no such day exists, or that day is already
// frozen (already carried from).
func carryFromPreviousDay(dir string) (carried []model.Item, sourceDate string, err error) {
	refs, err := logfile.List(dir)
	if err != nil {
		return nil, "", err
	}

	today := model.Today()
	var source *logfile.Ref
	for i := range refs {
		if refs[i].Name >= today {
			break // refs is sorted ascending, so nothing further back stays before today.
		}
		source = &refs[i]
	}
	if source == nil {
		return nil, "", nil
	}

	file, err := logfile.Stat(dir, source.Name)
	if err != nil {
		return nil, "", err
	}
	if file.Body.Freezed {
		return nil, "", nil
	}

	carried = log.CarryForward(file.Body.Items, source.Name)

	file.Body.Freezed = true
	if err := logfile.Update(dir, source.Name, file.Body); err != nil {
		return nil, "", err
	}

	return carried, source.Name, nil
}

// TodoOptions controls the todo command behavior.
type TodoOptions struct {
	Start bool     // mark the task as started right away
	Tags  []string // tags to put on the new item
}

// Todo appends a TODO item (task) to today's log.
func Todo(w io.Writer, dir, content string, opts TodoOptions) error {
	file, err := ensureToday(w, dir)
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
	// AddTags is a no-op on empty tags, so this is safe to call
	// unconditionally.
	item, err = log.AddTags(&file.Body, item.Hash, opts.Tags)
	if err != nil {
		return err
	}
	return persistItem(w, dir, file, item, len(opts.Tags) > 0, view.Added)
}

// Tag adds tags to (or removes them from, when remove is true) the
// item matching hash in today's log, then refreshes the index.
func Tag(w io.Writer, dir, hash string, tags []string, remove bool) error {
	file, err := ensureToday(w, dir)
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

	// Always true: even removing the last tag needs the index rebuilt
	// to purge its now-stale entry.
	return persistItem(w, dir, file, item, true, view.TagsUpdated)
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

// parseRef splits a "<date>:<hash>" reference into its parts. ok is
// false when ref has no colon (a bare hash, not a full reference) or
// either half is empty (e.g. ":hash" or "date:") — a malformed
// reference must not be mistaken for a valid one.
func parseRef(ref string) (date, hash string, ok bool) {
	date, hash, found := strings.Cut(ref, ":")
	if !found || date == "" || hash == "" {
		return "", ref, false
	}
	return date, hash, true
}

// resolveRef finds the item referenced by "<date>:<hash>".
func resolveRef(dir, ref string) (model.Item, error) {
	date, hash, ok := parseRef(ref)
	if !ok {
		return model.Item{}, fmt.Errorf("expected <date>:<hash>, got %q", ref)
	}

	file, err := logfile.Stat(dir, date)
	if err != nil {
		return model.Item{}, err
	}
	for _, item := range file.Body.Items {
		if item.Hash == hash {
			return item, nil
		}
	}
	return model.Item{}, fmt.Errorf("target item %q is not found on %s", hash, date)
}

// Diff prints the elapsed time between the creation of two items,
// each given as "<date>:<hash>".
func Diff(w io.Writer, dir, refA, refB string) error {
	itemA, err := resolveRef(dir, refA)
	if err != nil {
		return err
	}
	itemB, err := resolveRef(dir, refB)
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
	file, err := ensureToday(w, dir)
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
// hashes are collapsed to one. Within this call, all hashes must
// resolve to open tasks or none of them are persisted (this says
// nothing about two concurrent sava processes racing on the same
// file — there is no file locking anywhere in this codebase).
func End(w io.Writer, dir string, hashes []string) error {
	file, err := ensureToday(w, dir)
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

// Del removes the item matching ref from a daily log. ref may be a
// bare hash, searched across the last storagePeriodDays days (or
// every day, when deep is true), or a "<date>:<hash>" reference that
// is resolved directly, skipping the search.
func Del(w io.Writer, dir, ref string, deep bool) error {
	if date, hash, ok := parseRef(ref); ok {
		return deleteOn(w, dir, date, hash)
	}
	hash := ref

	refs, err := logfile.List(dir)
	if err != nil {
		return err
	}
	if !deep {
		refs = withinStoragePeriod(refs)
	}

	var found []string
	for _, r := range refs {
		file, err := logfile.Stat(dir, r.Name)
		if err != nil {
			return err
		}
		if log.HashExists(file.Body.Items, hash) {
			found = append(found, r.Name)
		}
	}

	switch len(found) {
	case 0:
		return fmt.Errorf("target item %q is not found", hash)
	case 1:
		return deleteOn(w, dir, found[0], hash)
	default:
		return fmt.Errorf("hash %q exists on multiple days (%s); specify as <date>:<hash>", hash, strings.Join(found, ", "))
	}
}

// deleteOn removes the item matching hash from the log for date. date
// may be any past day, not just today (see Del), so unlike the other
// write commands this must check Freezed itself: logfile.Update no
// longer guards against writing to a frozen day, and a carried-from
// day being permanently read-only is the whole point of freezing it.
func deleteOn(w io.Writer, dir, date, hash string) error {
	file, err := logfile.Stat(dir, date)
	if err != nil {
		return err
	}
	if file.Body.Freezed {
		return fmt.Errorf("%s: %w", date, logfile.ErrFreezed)
	}

	item, err := log.Delete(&file.Body, hash)
	if err != nil {
		return err
	}

	if err := logfile.Update(dir, file.Name, file.Body); err != nil {
		return err
	}

	view.Deleted(w, item)
	return nil
}

// withinStoragePeriod filters refs down to the last storagePeriodDays
// days, mirroring clearOld's retention window.
func withinStoragePeriod(refs []logfile.Ref) []logfile.Ref {
	deadline := time.Now().AddDate(0, 0, -storagePeriodDays)
	kept := refs[:0]
	for _, ref := range refs {
		date, err := model.ParseDate(ref.Name)
		if err != nil {
			continue
		}
		if !date.Before(deadline) {
			kept = append(kept, ref)
		}
	}
	return kept
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
	date := model.ResolveRelativeDate(opts.Date)

	file, err := logfile.Stat(dir, date)
	if err != nil && !errors.Is(err, logfile.ErrNotFound) {
		return err
	}

	if opts.Stat {
		if date == "" {
			view.Header(w, "Today's log stats are...")
		} else {
			view.Header(w, fmt.Sprintf("Log stats for %s are...", date))
		}
		if file == nil {
			fmt.Fprintln(w, "There is no body...")
			return nil
		}
		writeFileStat(w, file)
		return nil
	}

	if date == "" {
		view.Header(w, "Today's logs are...")
	} else {
		view.Header(w, fmt.Sprintf("Log for %s are...", date))
	}
	if file == nil {
		fmt.Fprintln(w, "There is no body...")
		return nil
	}

	items := log.Timeline(file.Body)
	if len(opts.Tags) > 0 {
		items = log.FilterByTags(items, opts.Tags, opts.Or)
	}
	view.Timeline(w, items)
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
