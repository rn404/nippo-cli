// Package command implements the CLI subcommands. Each function takes
// the log directory and writer/reader explicitly so behavior is
// testable without touching the real home directory.
package command

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

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

// Add appends a task (or a memo when memo is true) to today's log.
func Add(dir, content string, memo bool) error {
	file, err := logfile.Get(dir, "")
	if err != nil {
		return err
	}
	if err := log.Add(&file.Body, content, !memo); err != nil {
		return err
	}
	return logfile.Update(dir, file.Name, file.Body)
}

// End closes the task matching hash in today's log.
func End(w io.Writer, dir, hash string) error {
	file, err := logfile.Get(dir, "")
	if err != nil {
		return err
	}

	finished, err := log.Finish(&file.Body, hash)
	if err != nil {
		return err
	}

	view.FinishedTask(w, finished)
	return logfile.Update(dir, file.Name, file.Body)
}

// Del removes the item matching hash from today's log.
func Del(dir, hash string) error {
	file, err := logfile.Get(dir, "")
	if err != nil {
		return err
	}

	log.Delete(&file.Body, hash)
	return logfile.Update(dir, file.Name, file.Body)
}

// ListOptions controls the list command behavior.
type ListOptions struct {
	Date string // yyyy-MM-dd; empty means today
	All  bool
	Stat bool
	Yes  bool // skip confirmation prompts
}

// List shows the items of one day, or summaries across all log files.
func List(w io.Writer, r io.Reader, dir string, opts ListOptions) error {
	if !opts.All {
		return listOneDay(w, dir, opts)
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
	if err != nil {
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
	return nil
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
