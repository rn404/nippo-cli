package command

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	if err := Todo(io.Discard, dir, "buy cabbage", TodoOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := Add(io.Discard, dir, "a memo", AddOptions{}); err != nil {
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
	if err := End(&out, dir, []string{task.Hash}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Finished!!") {
		t.Errorf("End output = %q", out.String())
	}
	if items := todayItems(t, dir); !items[0].IsClosed() {
		t.Errorf("task should be closed after End: %+v", items[0])
	}

	out.Reset()
	if err := Del(&out, dir, memo.Hash, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Deleted!!") {
		t.Errorf("Del output = %q", out.String())
	}
	if items := todayItems(t, dir); len(items) != 1 {
		t.Errorf("items after Del = %+v, want only the task", items)
	}

	if err := Del(&out, dir, "no-such-hash", false); err == nil {
		t.Errorf("deleting unknown hash should fail")
	}
}

func TestEndMultiple(t *testing.T) {
	dir := t.TempDir()
	if err := Todo(io.Discard, dir, "buy cabbage", TodoOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := Todo(io.Discard, dir, "feed the shrimp", TodoOptions{}); err != nil {
		t.Fatal(err)
	}
	items := todayItems(t, dir)
	hashA, hashB := items[0].Hash, items[1].Hash

	var out strings.Builder
	if err := End(&out, dir, []string{hashA, hashB}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(out.String(), "Finished!!"); got != 2 {
		t.Errorf("Finished!! count = %d, want 2:\n%s", got, out.String())
	}
	items = todayItems(t, dir)
	if !items[0].IsClosed() || !items[1].IsClosed() {
		t.Errorf("both tasks should be closed: %+v", items)
	}
}

func TestEndDuplicateHash(t *testing.T) {
	dir := t.TempDir()
	if err := Todo(io.Discard, dir, "buy cabbage", TodoOptions{}); err != nil {
		t.Fatal(err)
	}
	hash := todayItems(t, dir)[0].Hash

	var out strings.Builder
	if err := End(&out, dir, []string{hash, hash}); err != nil {
		t.Fatalf("End with a duplicate hash should not error: %v", err)
	}
	if got := strings.Count(out.String(), "Finished!!"); got != 1 {
		t.Errorf("Finished!! count = %d, want 1 (duplicate collapsed):\n%s", got, out.String())
	}
	if items := todayItems(t, dir); !items[0].IsClosed() {
		t.Errorf("task should be closed: %+v", items[0])
	}
}

func TestEndPartialFailureIsAtomic(t *testing.T) {
	dir := t.TempDir()
	if err := Todo(io.Discard, dir, "buy cabbage", TodoOptions{}); err != nil {
		t.Fatal(err)
	}
	hash := todayItems(t, dir)[0].Hash

	var out strings.Builder
	if err := End(&out, dir, []string{hash, "no-such-hash"}); err == nil {
		t.Fatal("End with one unknown hash should fail")
	}
	if items := todayItems(t, dir); items[0].IsClosed() {
		t.Errorf("valid hash should not be persisted when the batch fails: %+v", items[0])
	}
}

func TestEndErrors(t *testing.T) {
	dir := t.TempDir()
	if err := Add(io.Discard, dir, "a memo", AddOptions{}); err != nil {
		t.Fatal(err)
	}
	memo := todayItems(t, dir)[0]

	var out strings.Builder
	if err := End(&out, dir, []string{"no-such-hash"}); err == nil {
		t.Errorf("End with unknown hash should fail")
	}
	if err := End(&out, dir, []string{memo.Hash}); err == nil {
		t.Errorf("End on memo should fail")
	}
}

func TestStartFlow(t *testing.T) {
	dir := t.TempDir()

	if err := Todo(io.Discard, dir, "slice cabbage", TodoOptions{}); err != nil {
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

func TestTodoWithStart(t *testing.T) {
	dir := t.TempDir()

	if err := Todo(io.Discard, dir, "feed the shrimp", TodoOptions{Start: true}); err != nil {
		t.Fatal(err)
	}
	if items := todayItems(t, dir); !items[0].IsStarted() {
		t.Errorf("todo added with start should be started: %+v", items[0])
	}
}

func TestAddOutputsAddedConfirmation(t *testing.T) {
	dir := t.TempDir()

	var out strings.Builder
	if err := Add(&out, dir, "buy cabbage", AddOptions{Tags: []string{"cabbage"}}); err != nil {
		t.Fatal(err)
	}
	item := todayItems(t, dir)[0]
	if !strings.Contains(out.String(), "Added!!") || !strings.Contains(out.String(), item.Hash) || !strings.Contains(out.String(), "#cabbage") {
		t.Errorf("Add output = %q", out.String())
	}
}

// TestAddConfirmsEvenWhenIndexRebuildFails guards against a bug where
// a tagged Add/Todo would durably persist the item but skip the
// Added!! confirmation if the follow-up index.Rebuild failed, making
// a successful write look like it never happened.
func TestAddConfirmsEvenWhenIndexRebuildFails(t *testing.T) {
	dir := t.TempDir()
	breakIndexRebuild(t, dir)

	var out strings.Builder
	if err := Add(&out, dir, "buy cabbage", AddOptions{Tags: []string{"cabbage"}}); err == nil {
		t.Fatal("Add should surface the index.Rebuild failure")
	}
	if !strings.Contains(out.String(), "Added!!") {
		t.Errorf("Add should still confirm the durable write: %q", out.String())
	}
	items := todayItems(t, dir)
	if len(items) != 1 || items[0].Content != "buy cabbage" {
		t.Errorf("item should still be persisted despite the index failure: %+v", items)
	}
}

func TestTagFlow(t *testing.T) {
	dir := t.TempDir()
	if err := Add(io.Discard, dir, "buy cabbage", AddOptions{Tags: []string{"cabbage", "shopping"}}); err != nil {
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

// TestTagRebuildsIndexOnLastTagRemoved guards against a naive shared
// "rebuild if item has tags" helper: removing an item's only tag
// leaves it with zero tags, but the index still must be rebuilt to
// purge that tag's now-stale entry.
func TestTagRebuildsIndexOnLastTagRemoved(t *testing.T) {
	dir := t.TempDir()
	if err := Add(io.Discard, dir, "buy cabbage", AddOptions{Tags: []string{"onlytag"}}); err != nil {
		t.Fatal(err)
	}
	item := todayItems(t, dir)[0]

	if err := Tag(io.Discard, dir, item.Hash, []string{"onlytag"}, true); err != nil {
		t.Fatal(err)
	}
	if updated := todayItems(t, dir)[0]; len(updated.Tags) != 0 {
		t.Fatalf("item should have no tags left: %+v", updated.Tags)
	}

	idx, err := index.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, stale := idx.Tags["onlytag"]; stale {
		t.Errorf("index should no longer list onlytag: %+v", idx.Tags)
	}
}

// TestTagConfirmsEvenWhenIndexRebuildFails mirrors
// TestAddConfirmsEvenWhenIndexRebuildFails: Tag must not skip its
// confirmation just because the follow-up index.Rebuild fails after
// the tag change was already durably persisted.
func TestTagConfirmsEvenWhenIndexRebuildFails(t *testing.T) {
	dir := t.TempDir()
	if err := Add(io.Discard, dir, "buy cabbage", AddOptions{Tags: []string{"cabbage"}}); err != nil {
		t.Fatal(err)
	}
	item := todayItems(t, dir)[0]
	breakIndexRebuild(t, dir)

	var out strings.Builder
	if err := Tag(&out, dir, item.Hash, []string{"food"}, false); err == nil {
		t.Fatal("Tag should surface the index.Rebuild failure")
	}
	if !strings.Contains(out.String(), "Tags updated!!") {
		t.Errorf("Tag should still confirm the durable write: %q", out.String())
	}
	if updated := todayItems(t, dir)[0]; !updated.HasTag("food") {
		t.Errorf("tag change should still be persisted despite the index failure: %+v", updated.Tags)
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

	if err := Add(io.Discard, dir, "buy cabbage", AddOptions{Tags: []string{"cabbage"}}); err != nil {
		t.Fatal(err)
	}
	if err := Add(io.Discard, dir, "more cabbage", AddOptions{Tags: []string{"cabbage"}}); err != nil {
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
		if err := Add(io.Discard, dir, content, AddOptions{Tags: tags}); err != nil {
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

// breakIndexRebuild writes a corrupt sibling log file so that
// index.Rebuild (which scans every daily log) fails, independently of
// today's file.
func breakIndexRebuild(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "2000-01-01.json"), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// writeDay stores items as the log of day, bypassing Add so tests can
// control hashes and timestamps.
func writeDay(t *testing.T, dir, day string, items []model.Item) {
	t.Helper()
	file, err := logfile.Get(dir, day)
	if err != nil {
		t.Fatal(err)
	}
	file.Body.Items = items
	if err := logfile.Update(dir, day, file.Body); err != nil {
		t.Fatal(err)
	}
}

func TestDiffAcrossDays(t *testing.T) {
	dir := t.TempDir()
	writeDay(t, dir, "2026-07-05", []model.Item{
		{Hash: "aaaa1111", Content: "buy cabbage", CreatedAt: "2026-07-05T10:00:00.000Z", UpdatedAt: "2026-07-05T10:00:00.000Z"},
	})
	writeDay(t, dir, "2026-07-06", []model.Item{
		{Hash: "bbbb2222", Content: "feed the shrimp", CreatedAt: "2026-07-06T12:30:00.000Z", UpdatedAt: "2026-07-06T12:30:00.000Z"},
	})

	var out strings.Builder
	if err := Diff(&out, dir, "2026-07-05:aaaa1111", "2026-07-06:bbbb2222"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Elapsed: 1d 2h 30m") {
		t.Errorf("Diff output = %q", out.String())
	}

	// Reversed order measures the same distance.
	out.Reset()
	if err := Diff(&out, dir, "2026-07-06:bbbb2222", "2026-07-05:aaaa1111"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Elapsed: 1d 2h 30m") {
		t.Errorf("reversed Diff output = %q", out.String())
	}

	if err := Diff(&out, dir, "2026-07-05:aaaa1111", "2026-07-05:no-such-hash"); err == nil {
		t.Errorf("Diff with unknown hash should fail")
	}
	if err := Diff(&out, dir, "aaaa1111", "2026-07-06:bbbb2222"); err == nil {
		t.Errorf("Diff with a bare hash (no date) should fail")
	}
}

func TestDelSearchesWithinStoragePeriod(t *testing.T) {
	dir := t.TempDir()
	recent := time.Now().AddDate(0, 0, -5).Format("2006-01-02")
	writeDay(t, dir, recent, []model.Item{
		{Hash: "aaaa1111", Content: "recent memo", CreatedAt: "2026-01-01T00:00:00.000Z", UpdatedAt: "2026-01-01T00:00:00.000Z"},
	})

	var out strings.Builder
	if err := Del(&out, dir, "aaaa1111", false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Deleted!!") {
		t.Errorf("Del output = %q", out.String())
	}
	file, err := logfile.Stat(dir, recent)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Body.Items) != 0 {
		t.Errorf("item should be removed: %+v", file.Body.Items)
	}
}

func TestDelIgnoresOldDaysUnlessDeep(t *testing.T) {
	dir := t.TempDir()
	old := time.Now().AddDate(0, 0, -40).Format("2006-01-02")
	writeDay(t, dir, old, []model.Item{
		{Hash: "aaaa1111", Content: "ancient memo", CreatedAt: "2026-01-01T00:00:00.000Z", UpdatedAt: "2026-01-01T00:00:00.000Z"},
	})

	var out strings.Builder
	if err := Del(&out, dir, "aaaa1111", false); err == nil {
		t.Error("Del without --deep should not find an item older than the storage period")
	}
	if err := Del(&out, dir, "aaaa1111", true); err != nil {
		t.Fatalf("Del --deep should find the old item: %v", err)
	}
}

func TestDelAmbiguousHashAcrossDays(t *testing.T) {
	dir := t.TempDir()
	dayA := time.Now().AddDate(0, 0, -3).Format("2006-01-02")
	dayB := time.Now().AddDate(0, 0, -5).Format("2006-01-02")
	writeDay(t, dir, dayA, []model.Item{
		{Hash: "dup11111", Content: "on day A", CreatedAt: "2026-01-01T00:00:00.000Z", UpdatedAt: "2026-01-01T00:00:00.000Z"},
	})
	writeDay(t, dir, dayB, []model.Item{
		{Hash: "dup11111", Content: "on day B", CreatedAt: "2026-01-01T00:00:00.000Z", UpdatedAt: "2026-01-01T00:00:00.000Z"},
	})

	var out strings.Builder
	if err := Del(&out, dir, "dup11111", false); err == nil {
		t.Error("Del with an ambiguous hash should fail without deleting anything")
	}
	for _, day := range []string{dayA, dayB} {
		file, err := logfile.Stat(dir, day)
		if err != nil {
			t.Fatal(err)
		}
		if len(file.Body.Items) != 1 {
			t.Errorf("neither day should be touched by the ambiguous Del: %s has %+v", day, file.Body.Items)
		}
	}

	// The <date>:<hash> form disambiguates directly.
	if err := Del(&out, dir, dayA+":dup11111", false); err != nil {
		t.Fatal(err)
	}
	fileA, err := logfile.Stat(dir, dayA)
	if err != nil {
		t.Fatal(err)
	}
	if len(fileA.Body.Items) != 0 {
		t.Errorf("day A's item should be gone: %+v", fileA.Body.Items)
	}
	fileB, err := logfile.Stat(dir, dayB)
	if err != nil {
		t.Fatal(err)
	}
	if len(fileB.Body.Items) != 1 {
		t.Errorf("day B's item should be untouched: %+v", fileB.Body.Items)
	}
}

func TestListToday(t *testing.T) {
	dir := t.TempDir()
	if err := Todo(io.Discard, dir, "buy cabbage", TodoOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := Add(io.Discard, dir, "shrimp memo", AddOptions{}); err != nil {
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
	if err := Todo(io.Discard, dir, "buy cabbage", TodoOptions{}); err != nil {
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
	if err := Add(io.Discard, dir, "recent", AddOptions{}); err != nil {
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
		t.Errorf("old file should actually be removed from disk")
	}
}

func TestClearAll(t *testing.T) {
	dir := t.TempDir()
	if err := Add(io.Discard, dir, "content", AddOptions{}); err != nil {
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
