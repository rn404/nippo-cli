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

// TestAddRejectsEmptyContent guards against a regression where an
// empty-content Add would still persist an item: the error from
// log.Add must propagate before persistItem ever runs.
func TestAddRejectsEmptyContent(t *testing.T) {
	dir := t.TempDir()
	if err := Add(io.Discard, dir, "seed memo", AddOptions{}); err != nil {
		t.Fatal(err)
	}

	if err := Add(io.Discard, dir, "", AddOptions{}); err == nil {
		t.Errorf("Add with empty content should fail")
	}
	if items := todayItems(t, dir); len(items) != 1 {
		t.Errorf("items = %+v, want only the seed memo (empty content must not be persisted)", items)
	}
}

// TestTodoRejectsEmptyContentSkipsSideEffects guards against empty
// content slipping through Todo's Start/Tags handling: since log.Add
// fails before the new item exists, Start and AddTags must never run
// for it, and the tag index must not be rebuilt as a result of this
// call.
func TestTodoRejectsEmptyContentSkipsSideEffects(t *testing.T) {
	dir := t.TempDir()
	if err := Todo(io.Discard, dir, "seed task", TodoOptions{}); err != nil {
		t.Fatal(err)
	}

	err := Todo(io.Discard, dir, "", TodoOptions{Start: true, Tags: []string{"urgent"}})
	if err == nil {
		t.Errorf("Todo with empty content should fail")
	}

	items := todayItems(t, dir)
	if len(items) != 1 {
		t.Errorf("items = %+v, want only the seed task (empty content must not be persisted)", items)
	}
	if items[0].IsStarted() {
		t.Errorf("seed task should not have been started as a side effect: %+v", items[0])
	}
	if _, statErr := os.Stat(index.Path(dir)); statErr == nil {
		t.Errorf("tag index should not exist: AddTags must not have run for the rejected item")
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
// today's file. It also seeds a valid, empty, more recent day so that
// automatic carry (which also scans past log files, to find the most
// recent one before today) lands on that instead of the broken file:
// this helper's job is to break the index, not carry.
func breakIndexRebuild(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "2000-01-01.json"), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := logfile.Get(dir, "2000-01-02"); err != nil {
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

// TestParseRefRejectsEmptyHalves guards against a ref like ":hash" or
// "date:" being silently accepted with an empty date/hash instead of
// being rejected as malformed.
func TestParseRefRejectsEmptyHalves(t *testing.T) {
	for _, ref := range []string{":abcd1234", "2026-08-02:", ":"} {
		if _, _, ok := parseRef(ref); ok {
			t.Errorf("parseRef(%q) should not be ok", ref)
		}
	}
	if date, hash, ok := parseRef("2026-08-02:abcd1234"); !ok || date != "2026-08-02" || hash != "abcd1234" {
		t.Errorf("parseRef(well-formed) = %q, %q, %v", date, hash, ok)
	}
}

// TestDelRejectsMalformedRef guards against the exact bug found in
// review: a ref with an empty date half must not silently resolve
// against today's log.
func TestDelRejectsMalformedRef(t *testing.T) {
	dir := t.TempDir()
	if err := Add(io.Discard, dir, "keep me", AddOptions{}); err != nil {
		t.Fatal(err)
	}
	item := todayItems(t, dir)[0]

	if err := Del(io.Discard, dir, ":"+item.Hash, false); err == nil {
		t.Error("Del with an empty-date ref should fail")
	}
	if items := todayItems(t, dir); len(items) != 1 {
		t.Errorf("item should survive a rejected malformed ref: %+v", items)
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

	for _, want := range []string{"Today's logs are...", "[ ] ", "buy cabbage", "shrimp memo"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("list output should contain %q:\n%s", want, out.String())
		}
	}
}

// TestListToday_HidesClosedTasksByDefault proves the default (no
// --full) daily view now hides closed tasks per requirements 1.1 and
// 1.2: open tasks and memos still show, only closed tasks are
// dropped.
func TestListToday_HidesClosedTasksByDefault(t *testing.T) {
	dir := t.TempDir()
	closed, open := true, false
	writeDay(t, dir, "", []model.Item{
		{Hash: "closed11", Content: "closed task", CreatedAt: "2026-01-01T01:00:00.000Z", UpdatedAt: "2026-01-01T01:00:00.000Z", Closed: &closed},
		{Hash: "open1111", Content: "open task", CreatedAt: "2026-01-01T02:00:00.000Z", UpdatedAt: "2026-01-01T02:00:00.000Z", Closed: &open},
		{Hash: "memo1111", Content: "a memo", CreatedAt: "2026-01-01T03:00:00.000Z", UpdatedAt: "2026-01-01T03:00:00.000Z"},
	})

	var out strings.Builder
	if err := List(&out, strings.NewReader(""), dir, ListOptions{}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Contains(got, "closed task") {
		t.Errorf("default list output should hide closed tasks: %q", got)
	}
	if !strings.Contains(got, "open task") || !strings.Contains(got, "a memo") {
		t.Errorf("default list output should keep open tasks and memos: %q", got)
	}
}

// TestListToday_AllClosedShowsEmptyMessage proves requirement 1.3: a
// day whose only items are closed tasks renders the same "no body"
// message as a day with nothing at all, once those closed tasks are
// hidden by default.
func TestListToday_AllClosedShowsEmptyMessage(t *testing.T) {
	dir := t.TempDir()
	closed := true
	writeDay(t, dir, "", []model.Item{
		{Hash: "closed11", Content: "closed task", CreatedAt: "2026-01-01T01:00:00.000Z", UpdatedAt: "2026-01-01T01:00:00.000Z", Closed: &closed},
	})

	var out strings.Builder
	if err := List(&out, strings.NewReader(""), dir, ListOptions{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "There is no body...") {
		t.Errorf("all-closed default list output = %q, want the empty message", out.String())
	}
}

// TestListToday_FullShowsClosedTasksInOrder proves requirements 2.1,
// 2.2, and 4.1: --full includes closed tasks alongside everything
// else, ordered closed -> open -> memo.
func TestListToday_FullShowsClosedTasksInOrder(t *testing.T) {
	dir := t.TempDir()
	closed, open := true, false
	writeDay(t, dir, "", []model.Item{
		{Hash: "open1111", Content: "open task", CreatedAt: "2026-01-01T01:00:00.000Z", UpdatedAt: "2026-01-01T01:00:00.000Z", Closed: &open},
		{Hash: "memo1111", Content: "a memo", CreatedAt: "2026-01-01T02:00:00.000Z", UpdatedAt: "2026-01-01T02:00:00.000Z"},
		{Hash: "closed11", Content: "closed task", CreatedAt: "2026-01-01T03:00:00.000Z", UpdatedAt: "2026-01-01T03:00:00.000Z", Closed: &closed},
	})

	var out strings.Builder
	if err := List(&out, strings.NewReader(""), dir, ListOptions{Full: true}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"closed task", "open task", "a memo"} {
		if !strings.Contains(got, want) {
			t.Errorf("--full output should contain %q:\n%s", want, got)
		}
	}
	iClosed, iOpen, iMemo := strings.Index(got, "closed task"), strings.Index(got, "open task"), strings.Index(got, "a memo")
	if iClosed >= iOpen || iOpen >= iMemo {
		t.Errorf("--full output order should be closed -> open -> memo, got:\n%s", got)
	}
}

// TestListToday_TaskOnlyExcludesMemos proves requirements 3.1 and
// 3.2: --task drops memos, and without --full it still hides closed
// tasks too.
func TestListToday_TaskOnlyExcludesMemos(t *testing.T) {
	dir := t.TempDir()
	closed, open := true, false
	writeDay(t, dir, "", []model.Item{
		{Hash: "closed11", Content: "closed task", CreatedAt: "2026-01-01T01:00:00.000Z", UpdatedAt: "2026-01-01T01:00:00.000Z", Closed: &closed},
		{Hash: "open1111", Content: "open task", CreatedAt: "2026-01-01T02:00:00.000Z", UpdatedAt: "2026-01-01T02:00:00.000Z", Closed: &open},
		{Hash: "memo1111", Content: "a memo", CreatedAt: "2026-01-01T03:00:00.000Z", UpdatedAt: "2026-01-01T03:00:00.000Z"},
	})

	var out strings.Builder
	if err := List(&out, strings.NewReader(""), dir, ListOptions{TasksOnly: true}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Contains(got, "a memo") {
		t.Errorf("--task output should exclude memos: %q", got)
	}
	if strings.Contains(got, "closed task") {
		t.Errorf("--task without --full should still hide closed tasks: %q", got)
	}
	if !strings.Contains(got, "open task") {
		t.Errorf("--task output should keep open tasks: %q", got)
	}
}

// TestListToday_TaskAndFullShowsAllTasksNoMemos proves requirement
// 3.3: --task --full shows both closed and open tasks but still no
// memos.
func TestListToday_TaskAndFullShowsAllTasksNoMemos(t *testing.T) {
	dir := t.TempDir()
	closed, open := true, false
	writeDay(t, dir, "", []model.Item{
		{Hash: "closed11", Content: "closed task", CreatedAt: "2026-01-01T01:00:00.000Z", UpdatedAt: "2026-01-01T01:00:00.000Z", Closed: &closed},
		{Hash: "open1111", Content: "open task", CreatedAt: "2026-01-01T02:00:00.000Z", UpdatedAt: "2026-01-01T02:00:00.000Z", Closed: &open},
		{Hash: "memo1111", Content: "a memo", CreatedAt: "2026-01-01T03:00:00.000Z", UpdatedAt: "2026-01-01T03:00:00.000Z"},
	})

	var out strings.Builder
	if err := List(&out, strings.NewReader(""), dir, ListOptions{TasksOnly: true, Full: true}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "closed task") || !strings.Contains(got, "open task") {
		t.Errorf("--task --full output should contain both tasks: %q", got)
	}
	if strings.Contains(got, "a memo") {
		t.Errorf("--task --full output should still exclude memos: %q", got)
	}
}

// TestListToday_FullOrdersClosedGroupByCreatedAtAscending proves
// requirement 4.2: within a single status group (here, closed tasks),
// items are ordered by creation time ascending, independent of the
// order they were written to the log file.
func TestListToday_FullOrdersClosedGroupByCreatedAtAscending(t *testing.T) {
	dir := t.TempDir()
	closed := true
	// Written with the later-created item first, so a passing test
	// proves SplitByStatus actually sorts rather than merely
	// preserving input order.
	writeDay(t, dir, "", []model.Item{
		{Hash: "closedb1", Content: "closed second", CreatedAt: "2026-01-01T02:00:00.000Z", UpdatedAt: "2026-01-01T02:00:00.000Z", Closed: &closed},
		{Hash: "closeda1", Content: "closed first", CreatedAt: "2026-01-01T01:00:00.000Z", UpdatedAt: "2026-01-01T01:00:00.000Z", Closed: &closed},
	})

	var out strings.Builder
	if err := List(&out, strings.NewReader(""), dir, ListOptions{Full: true}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	iFirst, iSecond := strings.Index(got, "closed first"), strings.Index(got, "closed second")
	if iFirst == -1 || iSecond == -1 {
		t.Fatalf("--full output should contain both closed tasks:\n%s", got)
	}
	if iFirst >= iSecond {
		t.Errorf("within the closed-task group, items should be ordered by CreatedAt ascending (\"closed first\" before \"closed second\"), got:\n%s", got)
	}
}

// TestListToday_FullHasNoGroupHeadersBetweenSections proves
// requirement 4.3: the closed/open/memo grouping is expressed purely
// through item order, with no extra heading or divider line inserted
// between groups. With one item per group, every printed body line
// must be an item line (starting with view's bullet prefix "- "), and
// there must be exactly as many lines as items.
func TestListToday_FullHasNoGroupHeadersBetweenSections(t *testing.T) {
	dir := t.TempDir()
	closed, open := true, false
	writeDay(t, dir, "", []model.Item{
		{Hash: "closed11", Content: "closed task", CreatedAt: "2026-01-01T01:00:00.000Z", UpdatedAt: "2026-01-01T01:00:00.000Z", Closed: &closed},
		{Hash: "open1111", Content: "open task", CreatedAt: "2026-01-01T02:00:00.000Z", UpdatedAt: "2026-01-01T02:00:00.000Z", Closed: &open},
		{Hash: "memo1111", Content: "a memo", CreatedAt: "2026-01-01T03:00:00.000Z", UpdatedAt: "2026-01-01T03:00:00.000Z"},
	})

	var out strings.Builder
	if err := List(&out, strings.NewReader(""), dir, ListOptions{Full: true}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	// Strip the "Today's logs are..." section header (blank line +
	// title + blank line) that view.Header prints ahead of the
	// timeline body, so only view.Timeline's own output is inspected
	// for group headers.
	bodyStart := strings.Index(got, "Today's logs are...")
	if bodyStart == -1 {
		t.Fatalf("expected output to contain the day header, got:\n%s", got)
	}
	body := got[bodyStart+len("Today's logs are..."):]

	var lines []string
	for _, line := range strings.Split(body, "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) != 3 {
		t.Fatalf("--full body should have exactly 3 non-empty lines (one per item, no group headers), got %d lines:\n%s", len(lines), body)
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "- ") {
			t.Errorf("every body line should be an item line starting with the bullet prefix %q (no header/divider between groups), got line %q in:\n%s", "- ", line, body)
		}
	}
}

// TestListToday_TagFilterCombinesWithVisibility proves requirement
// 5.1: --tag applies on top of the visibility rules as an AND
// condition, and the closed -> open -> memo grouping order survives
// the filter.
func TestListToday_TagFilterCombinesWithVisibility(t *testing.T) {
	dir := t.TempDir()
	closed, open := true, false
	writeDay(t, dir, "", []model.Item{
		{Hash: "closed11", Content: "closed tagged", CreatedAt: "2026-01-01T01:00:00.000Z", UpdatedAt: "2026-01-01T01:00:00.000Z", Closed: &closed, Tags: []string{"cabbage"}},
		{Hash: "closed22", Content: "closed untagged", CreatedAt: "2026-01-01T01:30:00.000Z", UpdatedAt: "2026-01-01T01:30:00.000Z", Closed: &closed},
		{Hash: "open1111", Content: "open tagged", CreatedAt: "2026-01-01T02:00:00.000Z", UpdatedAt: "2026-01-01T02:00:00.000Z", Closed: &open, Tags: []string{"cabbage"}},
		{Hash: "memo1111", Content: "memo tagged", CreatedAt: "2026-01-01T03:00:00.000Z", UpdatedAt: "2026-01-01T03:00:00.000Z", Tags: []string{"cabbage"}},
		{Hash: "memo2222", Content: "memo untagged", CreatedAt: "2026-01-01T03:30:00.000Z", UpdatedAt: "2026-01-01T03:30:00.000Z"},
	})

	var out strings.Builder
	if err := List(&out, strings.NewReader(""), dir, ListOptions{Full: true, Tags: []string{"cabbage"}}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"closed tagged", "open tagged", "memo tagged"} {
		if !strings.Contains(got, want) {
			t.Errorf("tag-filtered --full output should contain %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"closed untagged", "memo untagged"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("tag-filtered output should exclude %q:\n%s", unwanted, got)
		}
	}
	iClosed, iOpen, iMemo := strings.Index(got, "closed tagged"), strings.Index(got, "open tagged"), strings.Index(got, "memo tagged")
	if iClosed >= iOpen || iOpen >= iMemo {
		t.Errorf("tag-filtered output should preserve closed -> open -> memo order, got:\n%s", got)
	}
}

// TestListStatAndAll_UnaffectedByNewFlags proves requirements 5.2 and
// 5.3: --stat and --all (without --stat) ignore Full/TasksOnly/FullText
// entirely, so passing them alongside produces byte-identical output
// to not passing them.
func TestListStatAndAll_UnaffectedByNewFlags(t *testing.T) {
	dir := t.TempDir()
	closed := true
	writeDay(t, dir, "", []model.Item{
		{Hash: "closed11", Content: "closed task\nsecond line", CreatedAt: "2026-01-01T01:00:00.000Z", UpdatedAt: "2026-01-01T01:00:00.000Z", Closed: &closed},
	})

	var withoutFlags, withFlags strings.Builder
	if err := List(&withoutFlags, strings.NewReader(""), dir, ListOptions{Stat: true}); err != nil {
		t.Fatal(err)
	}
	if err := List(&withFlags, strings.NewReader(""), dir, ListOptions{Stat: true, Full: true, TasksOnly: true, FullText: true}); err != nil {
		t.Fatal(err)
	}
	if withoutFlags.String() != withFlags.String() {
		t.Errorf("--stat output should be unaffected by Full/TasksOnly/FullText:\nwithout = %q\nwith = %q", withoutFlags.String(), withFlags.String())
	}

	withoutFlags.Reset()
	withFlags.Reset()
	if err := List(&withoutFlags, strings.NewReader(""), dir, ListOptions{All: true}); err != nil {
		t.Fatal(err)
	}
	if err := List(&withFlags, strings.NewReader(""), dir, ListOptions{All: true, Full: true, TasksOnly: true, FullText: true}); err != nil {
		t.Fatal(err)
	}
	if withoutFlags.String() != withFlags.String() {
		t.Errorf("--all output should be unaffected by Full/TasksOnly/FullText:\nwithout = %q\nwith = %q", withoutFlags.String(), withFlags.String())
	}
}

// TestListToday_FullTextFlagShowsMultilineContent proves requirements
// 4.1 and 4.2 end-to-end: in the daily (non-stat) view, a multi-line
// item's content is summarized to its first line by default, and
// shown in full (all lines) when FullText is set — while the other
// item remains single-line either way, proving the flag only changes
// how multi-line content renders, not single-line content.
func TestListToday_FullTextFlagShowsMultilineContent(t *testing.T) {
	dir := t.TempDir()
	open := false
	writeDay(t, dir, "", []model.Item{
		{Hash: "open1111", Content: "first line\nsecond line\nthird line", CreatedAt: "2026-01-01T01:00:00.000Z", UpdatedAt: "2026-01-01T01:00:00.000Z", Closed: &open},
		{Hash: "memo1111", Content: "single line memo", CreatedAt: "2026-01-01T02:00:00.000Z", UpdatedAt: "2026-01-01T02:00:00.000Z"},
	})

	var summarized strings.Builder
	if err := List(&summarized, strings.NewReader(""), dir, ListOptions{}); err != nil {
		t.Fatal(err)
	}
	got := summarized.String()
	if !strings.Contains(got, "first line") {
		t.Errorf("default output should contain the first line: %q", got)
	}
	if strings.Contains(got, "second line") || strings.Contains(got, "third line") {
		t.Errorf("default (FullText=false) output should not contain later lines:\n%s", got)
	}
	if !strings.Contains(got, "single line memo") {
		t.Errorf("default output should still contain the single-line memo: %q", got)
	}

	var full strings.Builder
	if err := List(&full, strings.NewReader(""), dir, ListOptions{FullText: true}); err != nil {
		t.Fatal(err)
	}
	got = full.String()
	for _, want := range []string{"first line", "second line", "third line", "single line memo"} {
		if !strings.Contains(got, want) {
			t.Errorf("FullText=true output should contain %q:\n%s", want, got)
		}
	}
}

// TestListYesterday proves "yesterday" resolves to an actual date
// before reaching logfile.Stat, both for the plain timeline and for
// --stat, and that it's case-insensitive.
func TestListYesterday(t *testing.T) {
	dir := t.TempDir()
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	writeDay(t, dir, yesterday, []model.Item{
		{Hash: "aaaa1111", Content: "yesterday's memo", CreatedAt: "2026-01-01T00:00:00.000Z", UpdatedAt: "2026-01-01T00:00:00.000Z"},
	})

	for _, keyword := range []string{"yesterday", "Yesterday", "YESTERDAY"} {
		var out strings.Builder
		if err := List(&out, strings.NewReader(""), dir, ListOptions{Date: keyword}); err != nil {
			t.Fatal(err)
		}
		want := "Log for " + yesterday + " are..."
		if !strings.Contains(out.String(), want) || !strings.Contains(out.String(), "yesterday's memo") {
			t.Errorf("List with Date=%q output = %q, want to contain %q and the item", keyword, out.String(), want)
		}
	}

	var stat strings.Builder
	if err := List(&stat, strings.NewReader(""), dir, ListOptions{Date: "yesterday", Stat: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stat.String(), "Log stats for "+yesterday+" are...") {
		t.Errorf("List --stat with Date=yesterday output = %q", stat.String())
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

// TestCarryOnFirstWriteOfDay proves the core Phase C behavior: the
// first write command of a new day copies yesterday's unfinished
// TODOs forward with fresh hashes, freezes yesterday, and prints a
// notice ahead of the command's own output. Closed tasks and memos
// are not carried, and the source item itself is left untouched.
func TestCarryOnFirstWriteOfDay(t *testing.T) {
	dir := t.TempDir()
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	closed := true
	open := false
	writeDay(t, dir, yesterday, []model.Item{
		{Hash: "open1111", Content: "unfinished todo", CreatedAt: "2026-01-01T02:00:00.000Z", UpdatedAt: "2026-01-01T02:00:00.000Z", Closed: &open, Tags: []string{"cli"}},
		{Hash: "done1111", Content: "finished todo", CreatedAt: "2026-01-01T01:00:00.000Z", UpdatedAt: "2026-01-01T01:00:00.000Z", Closed: &closed},
		{Hash: "memo1111", Content: "a memo", CreatedAt: "2026-01-01T03:00:00.000Z", UpdatedAt: "2026-01-01T03:00:00.000Z"},
	})

	var out strings.Builder
	if err := Add(&out, dir, "today's memo", AddOptions{}); err != nil {
		t.Fatal(err)
	}

	notice := "Carried 1 items from " + yesterday + " (that day is now frozen)."
	if !strings.Contains(out.String(), notice) {
		t.Errorf("carry notice = %q, want to contain %q", out.String(), notice)
	}
	if strings.Index(out.String(), "Carried") > strings.Index(out.String(), "Added!!") {
		t.Errorf("carry notice should print before the command's own output: %q", out.String())
	}

	items := todayItems(t, dir)
	if len(items) != 2 {
		t.Fatalf("today's items = %+v, want 2 (the carried todo and the new memo)", items)
	}
	carried := items[0]
	if carried.Content != "unfinished todo" {
		t.Errorf("carried item content = %q, want %q", carried.Content, "unfinished todo")
	}
	if carried.Hash == "open1111" {
		t.Errorf("carried item should get a fresh hash")
	}
	if carried.CarriedFrom == nil || *carried.CarriedFrom != yesterday+":open1111" {
		t.Errorf("CarriedFrom = %v, want %q", carried.CarriedFrom, yesterday+":open1111")
	}
	if carried.IsStarted() || carried.IsClosed() {
		t.Errorf("carried item should start open and untouched today: %+v", carried)
	}
	if len(carried.Tags) != 1 || carried.Tags[0] != "cli" {
		t.Errorf("tags should carry over: %+v", carried.Tags)
	}

	source, err := logfile.Stat(dir, yesterday)
	if err != nil {
		t.Fatal(err)
	}
	if !source.Body.Freezed {
		t.Errorf("source day should be frozen after carry")
	}
	if len(source.Body.Items) != 3 {
		t.Fatalf("source items should be untouched: %+v", source.Body.Items)
	}
	for _, item := range source.Body.Items {
		if item.Hash == "open1111" && (item.IsClosed() || item.CarriedFrom != nil) {
			t.Errorf("the source item itself must not be rewritten by carry: %+v", item)
		}
	}
}

// TestCarryFreezesEvenWithNothingToCarry proves that a source day with
// no unfinished TODOs (a memo only, here) is still frozen once it is
// passed over as "the most recent day before today": carry marks the
// day as visited regardless of whether there was anything to copy,
// and prints no notice for a zero-item carry.
func TestCarryFreezesEvenWithNothingToCarry(t *testing.T) {
	dir := t.TempDir()
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	writeDay(t, dir, yesterday, []model.Item{
		{Hash: "memo1111", Content: "just a memo", CreatedAt: "2026-01-01T00:00:00.000Z", UpdatedAt: "2026-01-01T00:00:00.000Z"},
	})

	var out strings.Builder
	if err := Add(&out, dir, "today's memo", AddOptions{}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Carried") {
		t.Errorf("no carry notice expected when nothing was carried: %q", out.String())
	}
	if items := todayItems(t, dir); len(items) != 1 {
		t.Fatalf("today's items = %+v, want only the new memo", items)
	}

	source, err := logfile.Stat(dir, yesterday)
	if err != nil {
		t.Fatal(err)
	}
	if !source.Body.Freezed {
		t.Errorf("source day should still be frozen even with nothing to carry")
	}
}

// TestCarrySkipsAlreadyFrozenDay proves a day already frozen (already
// carried from once) is left alone: no second carry, no notice, and
// today starts as a plain log with nothing extra in it.
func TestCarrySkipsAlreadyFrozenDay(t *testing.T) {
	dir := t.TempDir()
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	open := false
	writeDay(t, dir, yesterday, []model.Item{
		{Hash: "open1111", Content: "unfinished todo", CreatedAt: "2026-01-01T00:00:00.000Z", UpdatedAt: "2026-01-01T00:00:00.000Z", Closed: &open},
	})
	frozen, err := logfile.Stat(dir, yesterday)
	if err != nil {
		t.Fatal(err)
	}
	frozen.Body.Freezed = true
	if err := logfile.Update(dir, yesterday, frozen.Body); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	if err := Add(&out, dir, "today's memo", AddOptions{}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Carried") {
		t.Errorf("no carry should run against an already-frozen day: %q", out.String())
	}
	if items := todayItems(t, dir); len(items) != 1 {
		t.Errorf("today should only have the new memo, not a carried copy: %+v", items)
	}
}

// TestCarryFindsMostRecentDayAcrossAGap proves the source day is
// "whichever day actually has a log file, most recently, before
// today" rather than literally "yesterday": a multi-day gap (e.g. a
// weekend sava was never touched) is skipped without special-casing.
func TestCarryFindsMostRecentDayAcrossAGap(t *testing.T) {
	dir := t.TempDir()
	fiveDaysAgo := time.Now().AddDate(0, 0, -5).Format("2006-01-02")
	open := false
	writeDay(t, dir, fiveDaysAgo, []model.Item{
		{Hash: "open1111", Content: "unfinished todo", CreatedAt: "2026-01-01T00:00:00.000Z", UpdatedAt: "2026-01-01T00:00:00.000Z", Closed: &open},
	})

	var out strings.Builder
	if err := Add(&out, dir, "today's memo", AddOptions{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Carried 1 items from "+fiveDaysAgo) {
		t.Errorf("carry should reach across the gap to the most recent existing day: %q", out.String())
	}
}

// TestCarryRunsOnlyOncePerDay proves the one-carry-per-day guarantee:
// once today's file exists, a second write command the same day must
// not re-carry or re-freeze.
func TestCarryRunsOnlyOncePerDay(t *testing.T) {
	dir := t.TempDir()
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	open := false
	writeDay(t, dir, yesterday, []model.Item{
		{Hash: "open1111", Content: "unfinished todo", CreatedAt: "2026-01-01T00:00:00.000Z", UpdatedAt: "2026-01-01T00:00:00.000Z", Closed: &open},
	})

	if err := Add(io.Discard, dir, "first write", AddOptions{}); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	if err := Add(&out, dir, "second write", AddOptions{}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Carried") {
		t.Errorf("carry should not run again on the same day: %q", out.String())
	}

	if items := todayItems(t, dir); len(items) != 3 {
		t.Fatalf("items = %+v, want 3 (1 carried + 2 memos, no duplicate carry)", items)
	}
}

// TestEdit_DirectMode_UpdatesContent proves requirements 1.1 and 1.4:
// Edit rewrites the content of a today's item directly and bumps
// UpdatedAt.
func TestEdit_DirectMode_UpdatesContent(t *testing.T) {
	dir := t.TempDir()
	item := model.Item{Hash: "aaaa1111", Content: "buy cabbage", CreatedAt: "2026-01-01T00:00:00.000Z", UpdatedAt: "2026-01-01T00:00:00.000Z"}
	writeDay(t, dir, "", []model.Item{item})

	var out strings.Builder
	if err := Edit(&out, dir, item.Hash, "buy more cabbage"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Edited!!") || !strings.Contains(out.String(), "buy more cabbage") {
		t.Errorf("Edit output = %q", out.String())
	}

	updated := todayItems(t, dir)[0]
	if updated.Content != "buy more cabbage" {
		t.Errorf("content = %q, want %q", updated.Content, "buy more cabbage")
	}
	if updated.Hash != item.Hash {
		t.Errorf("hash should not change: got %q, want %q", updated.Hash, item.Hash)
	}
	if updated.UpdatedAt == item.UpdatedAt {
		t.Errorf("UpdatedAt should change after Edit")
	}
}

// TestEdit_AllowsEmptyContent proves requirement 1.2: an empty string
// is applied as-is, without validation.
func TestEdit_AllowsEmptyContent(t *testing.T) {
	dir := t.TempDir()
	if err := Add(io.Discard, dir, "buy cabbage", AddOptions{}); err != nil {
		t.Fatal(err)
	}
	item := todayItems(t, dir)[0]

	if err := Edit(io.Discard, dir, item.Hash, ""); err != nil {
		t.Fatal(err)
	}
	if updated := todayItems(t, dir)[0]; updated.Content != "" {
		t.Errorf("content = %q, want empty", updated.Content)
	}
}

// TestEdit_NotFoundForUnknownOrPastDayHash proves requirement 3.1 and
// 3.2: a hash that only exists on a past day, or does not exist at
// all, is an error when editing today's log.
func TestEdit_NotFoundForUnknownOrPastDayHash(t *testing.T) {
	dir := t.TempDir()
	past := time.Now().AddDate(0, 0, -3).Format("2006-01-02")
	writeDay(t, dir, past, []model.Item{
		{Hash: "aaaa1111", Content: "past memo", CreatedAt: "2026-01-01T00:00:00.000Z", UpdatedAt: "2026-01-01T00:00:00.000Z"},
	})
	if err := Add(io.Discard, dir, "today memo", AddOptions{}); err != nil {
		t.Fatal(err)
	}

	if err := Edit(io.Discard, dir, "aaaa1111", "new content"); err == nil {
		t.Error("Edit with a past-day-only hash should fail")
	}
	if err := Edit(io.Discard, dir, "no-such-hash", "new content"); err == nil {
		t.Error("Edit with an unknown hash should fail")
	}
}

// TestEdit_PreservesStatusAcrossKinds proves requirements 1.3, 4.1 and
// 4.2: a memo, an open task, a started task, and a closed task can all
// be edited, and none of their state fields (Hash, CreatedAt, Closed,
// StartedAt) change, only Content and UpdatedAt do.
func TestEdit_PreservesStatusAcrossKinds(t *testing.T) {
	dir := t.TempDir()
	closed, open := true, false
	startedAt := "2026-01-01T05:00:00.000Z"
	items := []model.Item{
		{Hash: "memo1111", Content: "a memo", CreatedAt: "2026-01-01T01:00:00.000Z", UpdatedAt: "2026-01-01T01:00:00.000Z"},
		{Hash: "open1111", Content: "open task", CreatedAt: "2026-01-01T02:00:00.000Z", UpdatedAt: "2026-01-01T02:00:00.000Z", Closed: &open},
		{Hash: "start111", Content: "started task", CreatedAt: "2026-01-01T03:00:00.000Z", UpdatedAt: "2026-01-01T03:00:00.000Z", Closed: &open, StartedAt: &startedAt},
		{Hash: "close111", Content: "closed task", CreatedAt: "2026-01-01T04:00:00.000Z", UpdatedAt: "2026-01-01T04:00:00.000Z", Closed: &closed},
	}
	writeDay(t, dir, "", items)

	for _, before := range items {
		if err := Edit(io.Discard, dir, before.Hash, "edited: "+before.Content); err != nil {
			t.Fatalf("Edit(%q) failed: %v", before.Hash, err)
		}

		var after model.Item
		for _, item := range todayItems(t, dir) {
			if item.Hash == before.Hash {
				after = item
			}
		}
		if after.Content != "edited: "+before.Content {
			t.Errorf("hash %q: content = %q, want %q", before.Hash, after.Content, "edited: "+before.Content)
		}
		if after.Hash != before.Hash {
			t.Errorf("hash %q: hash changed to %q", before.Hash, after.Hash)
		}
		if after.CreatedAt != before.CreatedAt {
			t.Errorf("hash %q: CreatedAt changed: %q -> %q", before.Hash, before.CreatedAt, after.CreatedAt)
		}
		if (after.Closed == nil) != (before.Closed == nil) || (after.Closed != nil && *after.Closed != *before.Closed) {
			t.Errorf("hash %q: Closed changed: %v -> %v", before.Hash, before.Closed, after.Closed)
		}
		if (after.StartedAt == nil) != (before.StartedAt == nil) || (after.StartedAt != nil && *after.StartedAt != *before.StartedAt) {
			t.Errorf("hash %q: StartedAt changed: %v -> %v", before.Hash, before.StartedAt, after.StartedAt)
		}
		if after.UpdatedAt == before.UpdatedAt {
			t.Errorf("hash %q: UpdatedAt should change after Edit", before.Hash)
		}
	}
}

// TestTodayItem_ReturnsCurrentContent proves requirement 2.1: TodayItem
// fetches the current content of an item that exists in today's log.
func TestTodayItem_ReturnsCurrentContent(t *testing.T) {
	dir := t.TempDir()
	if err := Add(io.Discard, dir, "buy cabbage", AddOptions{}); err != nil {
		t.Fatal(err)
	}
	want := todayItems(t, dir)[0]

	got, err := TodayItem(dir, want.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != want.Content || got.Hash != want.Hash {
		t.Errorf("TodayItem = %+v, want %+v", got, want)
	}
}

// TestTodayItem_NotFoundWhenNoTodayFileOrHash proves requirements 3.1
// and 3.2 for the read-only path: no today's log file at all, and an
// unknown hash within an existing today's log, are both errors.
func TestTodayItem_NotFoundWhenNoTodayFileOrHash(t *testing.T) {
	dir := t.TempDir()

	if _, err := TodayItem(dir, "no-such-hash"); err == nil {
		t.Error("TodayItem should fail when today's log file does not exist")
	}

	if err := Add(io.Discard, dir, "buy cabbage", AddOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := TodayItem(dir, "no-such-hash"); err == nil {
		t.Error("TodayItem should fail for an unknown hash within an existing today's log")
	}
}

// TestTodayItem_DoesNotTriggerCarryForward is the direct verification
// of this task's central design constraint: TodayItem must use
// logfile.Stat, never ensureToday, so merely looking up an item ahead
// of opening an editor must not create today's log file nor carry
// forward yesterday's unfinished TODOs.
func TestTodayItem_DoesNotTriggerCarryForward(t *testing.T) {
	dir := t.TempDir()
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	open := false
	writeDay(t, dir, yesterday, []model.Item{
		{Hash: "open1111", Content: "unfinished todo", CreatedAt: "2026-01-01T00:00:00.000Z", UpdatedAt: "2026-01-01T00:00:00.000Z", Closed: &open},
	})

	if _, err := TodayItem(dir, "open1111"); err == nil {
		t.Error("TodayItem should not find a hash that only exists on a past day")
	}

	if _, err := logfile.Stat(dir, ""); !errors.Is(err, logfile.ErrNotFound) {
		t.Errorf("today's log file should not have been created by TodayItem, err = %v", err)
	}
	source, err := logfile.Stat(dir, yesterday)
	if err != nil {
		t.Fatal(err)
	}
	if source.Body.Freezed {
		t.Error("yesterday's log should not be frozen by a read-only TodayItem call")
	}
}

// TestEditAborted_PrintsMessage proves EditAborted delegates to
// view.EditAborted and actually produces output.
func TestEditAborted_PrintsMessage(t *testing.T) {
	var out strings.Builder
	EditAborted(&out)
	if out.Len() == 0 {
		t.Error("EditAborted should print a message")
	}
}

// TestDelFailsOnFrozenDay proves del's own explicit freeze check:
// since logfile.Update no longer guards frozen writes internally, del
// must reject deleting from a day that carry has already frozen.
func TestDelFailsOnFrozenDay(t *testing.T) {
	dir := t.TempDir()
	day := time.Now().AddDate(0, 0, -5).Format("2006-01-02")
	writeDay(t, dir, day, []model.Item{
		{Hash: "aaaa1111", Content: "frozen memo", CreatedAt: "2026-01-01T00:00:00.000Z", UpdatedAt: "2026-01-01T00:00:00.000Z"},
	})
	file, err := logfile.Stat(dir, day)
	if err != nil {
		t.Fatal(err)
	}
	file.Body.Freezed = true
	if err := logfile.Update(dir, day, file.Body); err != nil {
		t.Fatal(err)
	}

	if err := Del(io.Discard, dir, day+":aaaa1111", false); !errors.Is(err, logfile.ErrFreezed) {
		t.Errorf("err = %v, want to wrap logfile.ErrFreezed", err)
	}

	reloaded, err := logfile.Stat(dir, day)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Body.Items) != 1 {
		t.Errorf("frozen day's item should survive the rejected delete: %+v", reloaded.Body.Items)
	}
}
