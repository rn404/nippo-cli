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
	if err := Add(&l, "new task", true); err != nil {
		t.Fatal(err)
	}
	if err := Add(&l, "new memo", false); err != nil {
		t.Fatal(err)
	}

	if len(l.Items) != 5 {
		t.Fatalf("items = %d, want 5", len(l.Items))
	}
	if !l.Items[3].IsTask() {
		t.Errorf("appended item should be a task: %+v", l.Items[3])
	}
	if l.Items[4].IsTask() {
		t.Errorf("appended item should be a memo: %+v", l.Items[4])
	}
}

func TestAddToFreezedLog(t *testing.T) {
	l := newTestLog()
	l.Freezed = true
	if err := Add(&l, "content", true); !errors.Is(err, ErrFreezed) {
		t.Errorf("err = %v, want ErrFreezed", err)
	}
}

func TestDelete(t *testing.T) {
	l := newTestLog()
	Delete(&l, "memo-1")
	if len(l.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(l.Items))
	}
	for _, item := range l.Items {
		if item.Hash == "memo-1" {
			t.Errorf("memo-1 should be deleted")
		}
	}

	Delete(&l, "no-such-hash")
	if len(l.Items) != 2 {
		t.Errorf("delete with unknown hash should be a no-op")
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
