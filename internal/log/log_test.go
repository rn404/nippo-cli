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

func TestAddRejectsEmptyContent(t *testing.T) {
	l := newTestLog()
	want := len(l.Items)

	for _, content := range []string{"", "   ", "\t\n"} {
		if _, err := Add(&l, content, false); !errors.Is(err, ErrEmptyContent) {
			t.Errorf("Add(%q, false) err = %v, want ErrEmptyContent", content, err)
		}
		if _, err := Add(&l, content, true); !errors.Is(err, ErrEmptyContent) {
			t.Errorf("Add(%q, true) err = %v, want ErrEmptyContent", content, err)
		}
	}
	if len(l.Items) != want {
		t.Errorf("items = %d, want unchanged %d", len(l.Items), want)
	}
}

func TestAddKeepsContentUntrimmed(t *testing.T) {
	l := newTestLog()

	memo, err := Add(&l, "  padded memo  ", false)
	if err != nil {
		t.Fatal(err)
	}
	if memo.Content != "  padded memo  " {
		t.Errorf("Content = %q, want leading/trailing whitespace preserved", memo.Content)
	}
}

func TestCarryForward(t *testing.T) {
	tagged := newTestLog()
	tagged.Items[0].Tags = []string{"cli"} // task-open

	carried := CarryForward(tagged.Items, "2026-07-05")

	if len(carried) != 1 {
		t.Fatalf("carried = %+v, want exactly 1 (the open task; memo and done task excluded)", carried)
	}

	item := carried[0]
	if item.Content != "open task" {
		t.Errorf("Content = %q, want %q", item.Content, "open task")
	}
	if item.Hash == "task-open" {
		t.Errorf("carried copy should get a fresh hash, not reuse the source hash")
	}
	if item.CarriedFrom == nil || *item.CarriedFrom != "2026-07-05:task-open" {
		t.Errorf("CarriedFrom = %v, want %q", item.CarriedFrom, "2026-07-05:task-open")
	}
	if item.IsStarted() {
		t.Errorf("carried copy should not inherit StartedAt: %+v", item)
	}
	if item.IsClosed() {
		t.Errorf("carried copy should start open, not closed: %+v", item)
	}
	if len(item.Tags) != 1 || item.Tags[0] != "cli" {
		t.Errorf("Tags = %+v, want carried over unchanged", item.Tags)
	}
	if item.CreatedAt == "2026-07-05T02:00:00.000Z" {
		t.Errorf("carried copy should get a fresh CreatedAt, not the source's")
	}
}

func TestCarryForwardEmpty(t *testing.T) {
	if got := CarryForward(nil, "2026-07-05"); got != nil {
		t.Errorf("CarryForward(nil, ...) = %+v, want nil", got)
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

// editTestLog returns a log covering every lifecycle state Edit must
// support regardless of status: an open task, a started task, a
// closed task, and a memo. Each item also carries a tag so Edit's
// promise to leave Tags untouched can be checked.
func editTestLog() model.Log {
	open := false
	started := false
	closedFlag := true
	startedAt := "2026-07-05T01:30:00.000Z"

	return model.Log{
		Hash:    "filehash",
		Freezed: false,
		Items: []model.Item{
			{Hash: "task-open", Content: "open task", CreatedAt: "2026-07-05T02:00:00.000Z", UpdatedAt: "2026-07-05T02:00:00.000Z", Closed: &open, Tags: []string{"cli"}},
			{Hash: "task-started", Content: "started task", CreatedAt: "2026-07-05T01:15:00.000Z", UpdatedAt: "2026-07-05T01:30:00.000Z", Closed: &started, StartedAt: &startedAt, Tags: []string{"cli"}},
			{Hash: "task-done", Content: "done task", CreatedAt: "2026-07-05T01:00:00.000Z", UpdatedAt: "2026-07-05T01:30:00.000Z", Closed: &closedFlag, Tags: []string{"cli"}},
			{Hash: "memo-1", Content: "a memo", CreatedAt: "2026-07-05T03:00:00.000Z", UpdatedAt: "2026-07-05T03:00:00.000Z", Tags: []string{"cli"}},
		},
	}
}

func TestFind_ReturnsItem(t *testing.T) {
	l := editTestLog()

	item, err := Find(&l, "memo-1")
	if err != nil {
		t.Fatal(err)
	}
	if item.Content != "a memo" {
		t.Errorf("Content = %q, want %q", item.Content, "a memo")
	}

	// Find must not mutate the log.
	if len(l.Items) != 4 {
		t.Errorf("Find should not change the log: items = %d, want 4", len(l.Items))
	}
}

func TestFind_NotFound(t *testing.T) {
	l := editTestLog()

	if _, err := Find(&l, "no-such-hash"); err == nil {
		t.Errorf("finding unknown hash should fail")
	}
}

func TestEdit_UpdatesContentAndUpdatedAt(t *testing.T) {
	l := editTestLog()

	edited, err := Edit(&l, "memo-1", "fixed typo")
	if err != nil {
		t.Fatal(err)
	}
	if edited.Content != "fixed typo" {
		t.Errorf("Content = %q, want %q", edited.Content, "fixed typo")
	}
	if edited.UpdatedAt == "2026-07-05T03:00:00.000Z" {
		t.Errorf("UpdatedAt should be renewed on edit")
	}
	if l.Items[3].Content != "fixed typo" {
		t.Errorf("log should hold the edited item: %+v", l.Items[3])
	}
}

// TestEdit_PreservesHashCreatedAtAndStatus checks that an open task, a
// started task, a closed task, and a memo can all be edited, and that
// editing never changes Hash/CreatedAt/Closed/StartedAt/Tags -- only
// Content and UpdatedAt move.
func TestEdit_PreservesHashCreatedAtAndStatus(t *testing.T) {
	cases := []string{"task-open", "task-started", "task-done", "memo-1"}

	for _, hash := range cases {
		t.Run(hash, func(t *testing.T) {
			l := editTestLog()
			before, err := Find(&l, hash)
			if err != nil {
				t.Fatal(err)
			}

			edited, err := Edit(&l, hash, "edited content")
			if err != nil {
				t.Fatal(err)
			}

			if edited.Content != "edited content" {
				t.Errorf("Content = %q, want %q", edited.Content, "edited content")
			}
			if edited.UpdatedAt == before.UpdatedAt {
				t.Errorf("UpdatedAt should be renewed on edit")
			}

			if edited.Hash != before.Hash {
				t.Errorf("Hash changed: got %q, want %q", edited.Hash, before.Hash)
			}
			if edited.CreatedAt != before.CreatedAt {
				t.Errorf("CreatedAt changed: got %q, want %q", edited.CreatedAt, before.CreatedAt)
			}
			if (edited.Closed == nil) != (before.Closed == nil) {
				t.Fatalf("Closed nil-ness changed: got %v, want %v", edited.Closed, before.Closed)
			}
			if edited.Closed != nil && *edited.Closed != *before.Closed {
				t.Errorf("Closed value changed: got %v, want %v", *edited.Closed, *before.Closed)
			}
			if (edited.StartedAt == nil) != (before.StartedAt == nil) {
				t.Fatalf("StartedAt nil-ness changed: got %v, want %v", edited.StartedAt, before.StartedAt)
			}
			if edited.StartedAt != nil && *edited.StartedAt != *before.StartedAt {
				t.Errorf("StartedAt value changed: got %v, want %v", *edited.StartedAt, *before.StartedAt)
			}
			if len(edited.Tags) != len(before.Tags) {
				t.Fatalf("Tags changed: got %+v, want %+v", edited.Tags, before.Tags)
			}
			for i := range before.Tags {
				if edited.Tags[i] != before.Tags[i] {
					t.Errorf("Tags changed: got %+v, want %+v", edited.Tags, before.Tags)
					break
				}
			}

			// Also check the log's own copy, not just the returned value.
			stored := l.Items[indexOf(l.Items, hash)]
			if stored.Content != "edited content" {
				t.Errorf("log should hold the edited content: %+v", stored)
			}
		})
	}
}

// TestEdit_AllowsEmptyContent is a deliberate guard, not an oversight:
// Edit must not validate newContent at all, consistent with Add's
// lack of content validation. An empty string is accepted and applied
// as-is.
func TestEdit_AllowsEmptyContent(t *testing.T) {
	l := editTestLog()

	edited, err := Edit(&l, "memo-1", "")
	if err != nil {
		t.Fatalf("Edit with empty content should not error, got %v", err)
	}
	if edited.Content != "" {
		t.Errorf("Content = %q, want empty string applied as-is", edited.Content)
	}
	if l.Items[3].Content != "" {
		t.Errorf("log should hold the emptied content: %+v", l.Items[3])
	}
}

func TestEdit_NotFound(t *testing.T) {
	l := editTestLog()

	if _, err := Edit(&l, "no-such-hash", "new content"); err == nil {
		t.Errorf("editing unknown hash should fail")
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

// TestSplitByStatus_PartitionsByStatus checks that a log mixing closed,
// open, started tasks, and a memo is classified into the right bucket
// for each item.
func TestSplitByStatus_PartitionsByStatus(t *testing.T) {
	closed := true
	open := false
	startedAt := "2026-07-05T01:30:00.000Z"
	l := model.Log{
		Items: []model.Item{
			{Hash: "task-open", Content: "open task", CreatedAt: "2026-07-05T02:00:00.000Z", Closed: &open},
			{Hash: "task-done", Content: "done task", CreatedAt: "2026-07-05T01:00:00.000Z", Closed: &closed},
			{Hash: "task-started", Content: "started task", CreatedAt: "2026-07-05T01:15:00.000Z", Closed: &open, StartedAt: &startedAt},
			{Hash: "memo-1", Content: "a memo", CreatedAt: "2026-07-05T03:00:00.000Z"},
		},
	}

	closedTasks, openTasks, memos := SplitByStatus(l)

	if len(closedTasks) != 1 || closedTasks[0].Hash != "task-done" {
		t.Errorf("closedTasks = %+v, want only task-done", closedTasks)
	}
	if len(openTasks) != 2 {
		t.Fatalf("openTasks = %+v, want 2 (task-open and task-started)", openTasks)
	}
	gotOpenHashes := []string{openTasks[0].Hash, openTasks[1].Hash}
	wantOpenHashes := []string{"task-started", "task-open"} // sorted by createdAt ascending
	for i := range wantOpenHashes {
		if gotOpenHashes[i] != wantOpenHashes[i] {
			t.Errorf("openTasks = %+v, want order %+v", gotOpenHashes, wantOpenHashes)
			break
		}
	}
	if len(memos) != 1 || memos[0].Hash != "memo-1" {
		t.Errorf("memos = %+v, want only memo-1", memos)
	}

	total := len(closedTasks) + len(openTasks) + len(memos)
	if total != len(l.Items) {
		t.Errorf("total returned items = %d, want %d", total, len(l.Items))
	}

	for _, item := range closedTasks {
		if !item.IsClosed() {
			t.Errorf("closedTasks contains a non-closed item: %+v", item)
		}
	}
	for _, item := range openTasks {
		if !item.IsTask() || item.IsClosed() {
			t.Errorf("openTasks contains an item that is not an open/started task: %+v", item)
		}
	}
	for _, item := range memos {
		if item.IsTask() {
			t.Errorf("memos contains a task: %+v", item)
		}
	}
}

// TestSplitByStatus_OrdersByCreatedAtAscending checks that each bucket
// is sorted by CreatedAt ascending independently of the others.
func TestSplitByStatus_OrdersByCreatedAtAscending(t *testing.T) {
	closed := true
	l := model.Log{
		Items: []model.Item{
			{Hash: "closed-late", Content: "closed late", CreatedAt: "2026-07-05T05:00:00.000Z", Closed: &closed},
			{Hash: "closed-early", Content: "closed early", CreatedAt: "2026-07-05T01:00:00.000Z", Closed: &closed},
		},
	}

	closedTasks, openTasks, memos := SplitByStatus(l)

	if len(openTasks) != 0 || len(memos) != 0 {
		t.Fatalf("openTasks = %+v, memos = %+v, want both empty", openTasks, memos)
	}
	if len(closedTasks) != 2 || closedTasks[0].Hash != "closed-early" || closedTasks[1].Hash != "closed-late" {
		t.Errorf("closedTasks = %+v, want [closed-early, closed-late]", closedTasks)
	}
}

// TestSplitByStatus_AllClosed checks a log where every item is a closed
// task: openTasks and memos should come back empty while closedTasks
// holds everything.
func TestSplitByStatus_AllClosed(t *testing.T) {
	closed := true
	l := model.Log{
		Items: []model.Item{
			{Hash: "a", Content: "a", CreatedAt: "2026-07-05T02:00:00.000Z", Closed: &closed},
			{Hash: "b", Content: "b", CreatedAt: "2026-07-05T01:00:00.000Z", Closed: &closed},
		},
	}

	closedTasks, openTasks, memos := SplitByStatus(l)

	if len(closedTasks) != 2 {
		t.Fatalf("closedTasks = %+v, want 2", closedTasks)
	}
	if closedTasks[0].Hash != "b" || closedTasks[1].Hash != "a" {
		t.Errorf("closedTasks = %+v, want sorted [b, a]", closedTasks)
	}
	if openTasks != nil {
		t.Errorf("openTasks = %+v, want nil/empty", openTasks)
	}
	if memos != nil {
		t.Errorf("memos = %+v, want nil/empty", memos)
	}
}

// TestSplitByStatus_EmptyLog checks that an empty log yields three
// empty buckets rather than panicking or returning nil-vs-empty
// inconsistently.
func TestSplitByStatus_EmptyLog(t *testing.T) {
	closedTasks, openTasks, memos := SplitByStatus(model.Log{})

	if len(closedTasks) != 0 || len(openTasks) != 0 || len(memos) != 0 {
		t.Errorf("SplitByStatus(empty) = %+v, %+v, %+v, want all empty", closedTasks, openTasks, memos)
	}
}

// TestTimelineTiesKeepInsertionOrder guards against a non-deterministic
// tie-break: items sharing the exact same CreatedAt (possible within
// the same millisecond) must keep their original relative order.
func TestTimelineTiesKeepInsertionOrder(t *testing.T) {
	l := model.Log{Items: []model.Item{
		{Hash: "first", CreatedAt: "2026-07-05T02:00:00.000Z"},
		{Hash: "second", CreatedAt: "2026-07-05T02:00:00.000Z"},
		{Hash: "third", CreatedAt: "2026-07-05T02:00:00.000Z"},
	}}

	items := Timeline(l)
	got := []string{items[0].Hash, items[1].Hash, items[2].Hash}
	want := []string{"first", "second", "third"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("items = %+v, want insertion order %+v", got, want)
			break
		}
	}
}
