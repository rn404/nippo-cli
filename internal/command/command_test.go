package command

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rn404/nippo-cli/internal/index"
	"github.com/rn404/nippo-cli/internal/logfile"
	"github.com/rn404/nippo-cli/internal/model"
)

func todayItems(t *testing.T, dir string) []model.Item {
	t.Helper()
	file, err := logfile.Stat(dir, "")
	if errors.Is(err, logfile.ErrNotFound) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return file.Body.Items
}

func TestAddEndDelFlow(t *testing.T) {
	dir := t.TempDir()

	if err := Add(dir, "buy cabbage", AddOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := Add(dir, "a memo", AddOptions{Memo: true}); err != nil {
		t.Fatal(err)
	}

	items := todayItems(t, dir)
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	task, memo := items[0], items[1]
	if !task.IsTask() || memo.IsTask() {
		t.Fatalf("expected one task and one memo: %+v", items)
	}

	var out strings.Builder
	if err := End(&out, dir, task.Hash); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Finished!!") {
		t.Errorf("End output = %q", out.String())
	}
	if items := todayItems(t, dir); !items[0].IsClosed() {
		t.Errorf("task should be closed after End: %+v", items[0])
	}

	if err := Del(dir, memo.Hash); err != nil {
		t.Fatal(err)
	}
	if items := todayItems(t, dir); len(items) != 1 {
		t.Errorf("items after Del = %+v, want only the task", items)
	}
}

func TestEndErrors(t *testing.T) {
	dir := t.TempDir()
	if err := Add(dir, "a memo", AddOptions{Memo: true}); err != nil {
		t.Fatal(err)
	}
	memo := todayItems(t, dir)[0]

	var out strings.Builder
	if err := End(&out, dir, "no-such-hash"); err == nil {
		t.Errorf("End with unknown hash should fail")
	}
	if err := End(&out, dir, memo.Hash); err == nil {
		t.Errorf("End on memo should fail")
	}
}

func TestStartFlow(t *testing.T) {
	dir := t.TempDir()

	if err := Add(dir, "slice cabbage", AddOptions{}); err != nil {
		t.Fatal(err)
	}
	task := todayItems(t, dir)[0]

	var out strings.Builder
	if err := Start(&out, dir, task.Hash); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Started!!") {
		t.Errorf("Start output = %q", out.String())
	}
	if items := todayItems(t, dir); !items[0].IsStarted() {
		t.Errorf("task should be started after Start: %+v", items[0])
	}

	if err := Start(&out, dir, task.Hash); err == nil {
		t.Errorf("starting the same task twice should fail")
	}
}

func TestAddWithStart(t *testing.T) {
	dir := t.TempDir()

	if err := Add(dir, "feed the shrimp", AddOptions{Start: true}); err != nil {
		t.Fatal(err)
	}
	if items := todayItems(t, dir); !items[0].IsStarted() {
		t.Errorf("task added with start should be started: %+v", items[0])
	}

	if err := Add(dir, "a memo", AddOptions{Memo: true, Start: true}); err == nil {
		t.Errorf("memo with start should fail")
	}
}

func TestTagFlow(t *testing.T) {
	dir := t.TempDir()
	if err := Add(dir, "buy cabbage", AddOptions{Tags: []string{"cabbage", "shopping"}}); err != nil {
		t.Fatal(err)
	}
	item := todayItems(t, dir)[0]
	if len(item.Tags) != 2 {
		t.Fatalf("tags = %+v, want 2", item.Tags)
	}
	if _, err := os.Stat(index.Path(dir)); err != nil {
		t.Errorf("add with tags should write the index: %v", err)
	}

	var out strings.Builder
	if err := Tag(&out, dir, item.Hash, []string{"food"}, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Tags updated!!") || !strings.Contains(out.String(), "#food") {
		t.Errorf("Tag output = %q", out.String())
	}

	out.Reset()
	if err := Tag(&out, dir, item.Hash, []string{"shopping"}, true); err != nil {
		t.Fatal(err)
	}
	if item := todayItems(t, dir)[0]; item.HasTag("shopping") || !item.HasTag("food") {
		t.Errorf("shopping should be removed, food kept: %+v", item.Tags)
	}

	if err := Tag(&out, dir, "no-such-hash", []string{"x"}, false); err == nil {
		t.Errorf("tagging unknown hash should fail")
	}
}

func TestTagList(t *testing.T) {
	dir := t.TempDir()

	var out strings.Builder
	if err := TagList(&out, dir); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "There is no tags...") {
		t.Errorf("empty TagList output = %q", out.String())
	}

	if err := Add(dir, "buy cabbage", AddOptions{Tags: []string{"cabbage"}}); err != nil {
		t.Fatal(err)
	}
	if err := Add(dir, "more cabbage", AddOptions{Tags: []string{"cabbage"}}); err != nil {
		t.Fatal(err)
	}

	out.Reset()
	if err := TagList(&out, dir); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "- cabbage (2)") {
		t.Errorf("TagList output = %q", out.String())
	}
}

func TestListWithTagFilter(t *testing.T) {
	dir := t.TempDir()
	for content, tags := range map[string][]string{
		"tagged both":  {"go", "cli"},
		"tagged one":   {"go"},
		"tagged other": {"web"},
	} {
		if err := Add(dir, content, AddOptions{Tags: tags}); err != nil {
			t.Fatal(err)
		}
	}

	var out strings.Builder
	if err := List(&out, strings.NewReader(""), dir, ListOptions{Tags: []string{"go", "cli"}}); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); !strings.Contains(got, "tagged both") || strings.Contains(got, "tagged one") {
		t.Errorf("AND filter output = %q", got)
	}

	out.Reset()
	if err := List(&out, strings.NewReader(""), dir, ListOptions{Tags: []string{"go", "cli"}, Or: true}); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); !strings.Contains(got, "tagged one") || strings.Contains(got, "tagged other") {
		t.Errorf("OR filter output = %q", got)
	}

	if err := List(&out, strings.NewReader(""), dir, ListOptions{All: true, Tags: []string{"go"}}); err == nil {
		t.Errorf("tag filter with --all should fail")
	}
}

func TestListToday(t *testing.T) {
	dir := t.TempDir()
	if err := Add(dir, "buy cabbage", AddOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := Add(dir, "shrimp memo", AddOptions{Memo: true}); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	if err := List(&out, strings.NewReader(""), dir, ListOptions{}); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"Today's logs are...", "Task ->", "buy cabbage", "Memo ->", "shrimp memo"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("list output should contain %q:\n%s", want, out.String())
		}
	}
}

func TestListEmptyAndInvalidDate(t *testing.T) {
	dir := t.TempDir()

	var out strings.Builder
	if err := List(&out, strings.NewReader(""), dir, ListOptions{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "There is no body...") {
		t.Errorf("empty list output = %q", out.String())
	}

	if err := List(&out, strings.NewReader(""), dir, ListOptions{Date: "not-a-date"}); err == nil {
		t.Errorf("invalid date should return an error")
	}
}

func TestListStatAndAll(t *testing.T) {
	dir := t.TempDir()
	if err := Add(dir, "buy cabbage", AddOptions{}); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	if err := List(&out, strings.NewReader(""), dir, ListOptions{Stat: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Task: 1 (unfinished: 1), Memo: 0") {
		t.Errorf("stat output = %q", out.String())
	}

	out.Reset()
	if err := List(&out, strings.NewReader(""), dir, ListOptions{All: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), model.Today()) {
		t.Errorf("list -a output should contain today's file name: %q", out.String())
	}

	out.Reset()
	if err := List(&out, strings.NewReader(""), dir, ListOptions{All: true, Stat: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "There are 1 total.") {
		t.Errorf("list -a -s output = %q", out.String())
	}
}

func TestListAllStatConfirmDeclined(t *testing.T) {
	dir := t.TempDir()
	// Create more files than fileStatsLimit to trigger the prompt.
	for month := 1; month <= fileStatsLimit+1; month++ {
		day := time2date(2026, month, 1)
		if _, err := logfile.Get(dir, day); err != nil {
			t.Fatal(err)
		}
	}

	var out strings.Builder
	if err := List(&out, strings.NewReader("n\n"), dir, ListOptions{All: true, Stat: true}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Task:") {
		t.Errorf("stats should not be shown after declining: %q", out.String())
	}

	// With Yes the prompt is skipped.
	out.Reset()
	if err := List(&out, strings.NewReader(""), dir, ListOptions{All: true, Stat: true, Yes: true}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(out.String(), "Task:"); got != fileStatsLimit+1 {
		t.Errorf("stat lines = %d, want %d", got, fileStatsLimit+1)
	}
}

func TestClearOld(t *testing.T) {
	dir := t.TempDir()
	if _, err := logfile.Get(dir, "2000-01-01"); err != nil {
		t.Fatal(err)
	}
	if err := Add(dir, "recent", AddOptions{}); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	if err := Clear(&out, strings.NewReader(""), dir, false, false); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out.String(), "Deleted... 2000-01-01 logs.") {
		t.Errorf("clear output = %q", out.String())
	}
	refs, err := logfile.List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || refs[0].Name != model.Today() {
		t.Errorf("only today's file should remain, got %+v", refs)
	}
	if _, err := os.Stat(filepath.Join(dir, "2000-01-01.json")); !os.IsNotExist(err) {
		t.Errorf("old file should actually be removed from disk (Deno version bug)")
	}
}

func TestClearAll(t *testing.T) {
	dir := t.TempDir()
	if err := Add(dir, "content", AddOptions{}); err != nil {
		t.Fatal(err)
	}

	// Declined: nothing happens.
	var out strings.Builder
	if err := Clear(&out, strings.NewReader("n\n"), dir, true, false); err != nil {
		t.Fatal(err)
	}
	if refs, _ := logfile.List(dir); len(refs) != 1 {
		t.Errorf("declining should keep files, got %+v", refs)
	}

	// Accepted: files removed.
	out.Reset()
	if err := Clear(&out, strings.NewReader("y\n"), dir, true, false); err != nil {
		t.Fatal(err)
	}
	if refs, _ := logfile.List(dir); len(refs) != 0 {
		t.Errorf("all files should be deleted, got %+v", refs)
	}
	if !strings.Contains(out.String(), "Deleted all files.") {
		t.Errorf("clear -a output = %q", out.String())
	}

	// --yes skips the prompt entirely.
	out.Reset()
	if err := Clear(&out, strings.NewReader(""), dir, true, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "There is no log files.") {
		t.Errorf("clear -a on empty dir output = %q", out.String())
	}
}

func time2date(year, month, day int) string {
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}
