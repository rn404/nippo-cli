package log

import (
	"errors"
	"testing"

	"github.com/rn404/nippo-cli/internal/model"
)

func newTestLog() model.Log {
	closed := true
	open := false
	return model.Log{
		Hash:    "filehash",
		Freezed: false,
		Items: []model.Item{
			{Hash: "task-open", Content: "open task", CreatedAt: "2026-07-05T02:00:00.000Z", UpdatedAt: "2026-07-05T02:00:00.000Z", Closed: &open},
			{Hash: "task-done", Content: "done task", CreatedAt: "2026-07-05T01:00:00.000Z", UpdatedAt: "2026-07-05T01:30:00.000Z", Closed: &closed},
			{Hash: "memo-1", Content: "a memo", CreatedAt: "2026-07-05T03:00:00.000Z", UpdatedAt: "2026-07-05T03:00:00.000Z"},
		},
	}
}

func TestAdd(t *testing.T) {
	l := newTestLog()
	task, err := Add(&l, "new task", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Add(&l, "new memo", false); err != nil {
		t.Fatal(err)
	}

	if len(l.Items) != 5 {
		t.Fatalf("items = %d, want 5", len(l.Items))
	}
	if !l.Items[3].IsTask() {
		t.Errorf("appended item should be a task: %+v", l.Items[3])
	}
	if l.Items[3].Hash != task.Hash {
		t.Errorf("Add should return the appended item: %+v", task)
	}
	if l.Items[4].IsTask() {
		t.Errorf("appended item should be a memo: %+v", l.Items[4])
	}
}

func TestAddToFreezedLog(t *testing.T) {
	l := newTestLog()
	l.Freezed = true
	if _, err := Add(&l, "content", true); !errors.Is(err, ErrFreezed) {
		t.Errorf("err = %v, want ErrFreezed", err)
	}
}

func TestDelete(t *testing.T) {
	l := newTestLog()
	deleted, err := Delete(&l, "memo-1")
	if err != nil {
		t.Fatal(err)
	}
	if deleted.Hash != "memo-1" {
		t.Errorf("Delete should return the removed item: %+v", deleted)
	}
	if len(l.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(l.Items))
	}
	for _, item := range l.Items {
		if item.Hash == "memo-1" {
			t.Errorf("memo-1 should be deleted")
		}
	}

	if _, err := Delete(&l, "no-such-hash"); err == nil {
		t.Errorf("deleting an unknown hash should fail")
	}
	if len(l.Items) != 2 {
		t.Errorf("a failed delete should not change the log")
	}
}

// TestDeleteRemovesOnlyFirstMatch guards against a hash collision (see
// TestUniqueIDRetriesOnCollision for the Add-side retry that prevents
// same-day collisions) making Delete remove more than the one item it
// reports.
func TestDeleteRemovesOnlyFirstMatch(t *testing.T) {
	l := model.Log{Items: []model.Item{
		{Hash: "dup", Content: "first"},
		{Hash: "dup", Content: "second"},
	}}

	deleted, err := Delete(&l, "dup")
	if err != nil {
		t.Fatal(err)
	}
	if deleted.Content != "first" {
		t.Errorf("Delete should remove the first match: %+v", deleted)
	}
	if len(l.Items) != 1 || l.Items[0].Content != "second" {
		t.Errorf("the second colliding item should be left alone: %+v", l.Items)
	}
}

func TestHashExists(t *testing.T) {
	if !HashExists([]model.Item{{Hash: "x"}}, "x") {
		t.Error("HashExists should find a matching hash")
	}
	if HashExists([]model.Item{{Hash: "x"}}, "y") {
		t.Error("HashExists should not match a different hash")
	}
	if HashExists(nil, "x") {
		t.Error("HashExists on an empty slice should be false")
	}
}

// TestUniqueIDRetriesOnCollision proves the retry loop itself, since a
// real crypto/rand collision can't be forced from a test: a stub
// generator returns two colliding IDs before a fresh one.
func TestUniqueIDRetriesOnCollision(t *testing.T) {
	items := []model.Item{{Hash: "dup"}}

	calls := []string{"dup", "dup", "fresh"}
	next := 0
	generator := func() string {
		id := calls[next]
		next++
		return id
	}

	got := uniqueID(items, generator)
	if got != "fresh" {
		t.Errorf("uniqueID = %q, want %q after retrying past collisions", got, "fresh")
	}
	if next != len(calls) {
		t.Errorf("generator call count = %d, want %d (retries then success)", next, len(calls))
	}
}

func TestFinish(t *testing.T) {
	l := newTestLog()
	finished, err := Finish(&l, "task-open")
	if err != nil {
		t.Fatal(err)
	}
	if !finished.IsClosed() {
		t.Errorf("finished item should be closed: %+v", finished)
	}
	if finished.UpdatedAt == finished.CreatedAt {
		t.Errorf("updatedAt should be renewed on finish")
	}
	if !l.Items[0].IsClosed() {
		t.Errorf("log should hold the closed item: %+v", l.Items[0])
	}
}

func TestFinishErrors(t *testing.T) {
	l := newTestLog()

	if _, err := Finish(&l, "no-such-hash"); err == nil {
		t.Errorf("finishing unknown hash should fail")
	}
	if _, err := Finish(&l, "memo-1"); !errors.Is(err, ErrNotTask) {
		t.Errorf("err = %v, want ErrNotTask", err)
	}
	if _, err := Finish(&l, "task-done"); !errors.Is(err, ErrAlreadyFinished) {
		t.Errorf("err = %v, want ErrAlreadyFinished", err)
	}
}

func TestStart(t *testing.T) {
	l := newTestLog()
	started, err := Start(&l, "task-open")
	if err != nil {
		t.Fatal(err)
	}
	if !started.IsStarted() {
		t.Errorf("started item should have startedAt: %+v", started)
	}
	if started.UpdatedAt != *started.StartedAt {
		t.Errorf("updatedAt should match startedAt on start: %+v", started)
	}
	if !l.Items[0].IsStarted() {
		t.Errorf("log should hold the started item: %+v", l.Items[0])
	}
}

func TestStartErrors(t *testing.T) {
	l := newTestLog()

	if _, err := Start(&l, "no-such-hash"); err == nil {
		t.Errorf("starting unknown hash should fail")
	}
	if _, err := Start(&l, "memo-1"); !errors.Is(err, ErrNotTask) {
		t.Errorf("err = %v, want ErrNotTask", err)
	}
	if _, err := Start(&l, "task-done"); !errors.Is(err, ErrAlreadyFinished) {
		t.Errorf("err = %v, want ErrAlreadyFinished", err)
	}

	if _, err := Start(&l, "task-open"); err != nil {
		t.Fatal(err)
	}
	if _, err := Start(&l, "task-open"); !errors.Is(err, ErrAlreadyStarted) {
		t.Errorf("err = %v, want ErrAlreadyStarted", err)
	}
}

func TestAddAndRemoveTags(t *testing.T) {
	l := newTestLog()

	tagged, err := AddTags(&l, "memo-1", []string{"shrimp", " pet ", "shrimp"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tagged.Tags) != 2 || !tagged.HasTag("shrimp") || !tagged.HasTag("pet") {
		t.Errorf("tags should be trimmed and deduplicated: %+v", tagged.Tags)
	}
	if tagged.UpdatedAt == tagged.CreatedAt {
		t.Errorf("updatedAt should be renewed on tagging")
	}

	// Adding an existing tag is a no-op for that tag.
	tagged, err = AddTags(&l, "memo-1", []string{"shrimp", "happy"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tagged.Tags) != 3 {
		t.Errorf("tags = %+v, want 3 entries", tagged.Tags)
	}

	removed, err := RemoveTags(&l, "memo-1", []string{"pet", "unknown"})
	if err != nil {
		t.Fatal(err)
	}
	if len(removed.Tags) != 2 || removed.HasTag("pet") {
		t.Errorf("pet should be removed: %+v", removed.Tags)
	}

	removed, err = RemoveTags(&l, "memo-1", []string{"shrimp", "happy"})
	if err != nil {
		t.Fatal(err)
	}
	if removed.Tags != nil {
		t.Errorf("emptied tags should marshal away entirely: %+v", removed.Tags)
	}
}

func TestTagErrors(t *testing.T) {
	l := newTestLog()

	if _, err := AddTags(&l, "no-such-hash", []string{"tag"}); err == nil {
		t.Errorf("tagging unknown hash should fail")
	}
	if _, err := AddTags(&l, "memo-1", []string{""}); !errors.Is(err, ErrEmptyTag) {
		t.Errorf("err = %v, want ErrEmptyTag", err)
	}
	if _, err := AddTags(&l, "memo-1", []string{"has space"}); err == nil {
		t.Errorf("tag with whitespace should fail")
	}
	if _, err := RemoveTags(&l, "no-such-hash", []string{"tag"}); err == nil {
		t.Errorf("untagging unknown hash should fail")
	}
}

func TestFilterByTags(t *testing.T) {
	items := []model.Item{
		{Hash: "a", Tags: []string{"go", "cli"}},
		{Hash: "b", Tags: []string{"go"}},
		{Hash: "c"},
	}

	and := FilterByTags(items, []string{"go", "cli"}, false)
	if len(and) != 1 || and[0].Hash != "a" {
		t.Errorf("AND filter = %+v, want only a", and)
	}

	or := FilterByTags(items, []string{"go", "cli"}, true)
	if len(or) != 2 {
		t.Errorf("OR filter = %+v, want a and b", or)
	}
}

func TestSplit(t *testing.T) {
	tasks, memos := Split(newTestLog())

	if len(tasks) != 2 || len(memos) != 1 {
		t.Fatalf("tasks = %d, memos = %d, want 2 and 1", len(tasks), len(memos))
	}
	// Sorted by createdAt ascending: task-done (01:00) before task-open (02:00).
	if tasks[0].Hash != "task-done" || tasks[1].Hash != "task-open" {
		t.Errorf("tasks should be sorted by createdAt: %+v", tasks)
	}

	if got := CountUnfinished(tasks); got != 1 {
		t.Errorf("CountUnfinished = %d, want 1", got)
	}
}

func TestTimeline(t *testing.T) {
	items := Timeline(newTestLog())

	if len(items) != 3 {
		t.Fatalf("items = %d, want 3", len(items))
	}
	// Sorted by createdAt ascending, tasks and memos interleaved:
	// task-done (01:00), task-open (02:00), memo-1 (03:00).
	got := []string{items[0].Hash, items[1].Hash, items[2].Hash}
	want := []string{"task-done", "task-open", "memo-1"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("items = %+v, want order %+v", got, want)
			break
		}
	}
}
