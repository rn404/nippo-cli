package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestLegacyFileRoundTrip(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "legacy-samples", "2026-07-05.json")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read legacy sample: %v", err)
	}

	var body Log
	if err := json.Unmarshal(original, &body); err != nil {
		t.Fatalf("unmarshal legacy sample: %v", err)
	}

	if len(body.Items) != 3 {
		t.Fatalf("items = %d, want 3", len(body.Items))
	}
	if !body.Items[0].IsTask() || !body.Items[0].IsClosed() {
		t.Errorf("item 0 should be a closed task: %+v", body.Items[0])
	}
	if !body.Items[1].IsTask() || body.Items[1].IsClosed() {
		t.Errorf("item 1 should be an open task: %+v", body.Items[1])
	}
	if body.Items[2].IsTask() {
		t.Errorf("item 2 should be a memo: %+v", body.Items[2])
	}

	remarshaled, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(remarshaled) != string(original) {
		t.Errorf("round trip mismatch:\n--- original ---\n%s\n--- remarshaled ---\n%s", original, remarshaled)
	}
}

func TestHasTag(t *testing.T) {
	item := Item{Tags: []string{"go", "cli"}}
	if !item.HasTag("go") || item.HasTag("web") {
		t.Errorf("HasTag mismatch: %+v", item.Tags)
	}
	if (Item{}).HasTag("go") {
		t.Error("item without tags should not match")
	}
}

func TestNewID(t *testing.T) {
	pattern := regexp.MustCompile(`^[0-9a-f]{8}$`)
	seen := map[string]bool{}
	for range 100 {
		id := NewID()
		if !pattern.MatchString(id) {
			t.Fatalf("id %q does not match 8-char hex", id)
		}
		if seen[id] {
			t.Fatalf("duplicated id %q", id)
		}
		seen[id] = true
	}
}

func TestNewItems(t *testing.T) {
	task := NewTaskItem("task content")
	if !task.IsTask() || task.IsClosed() {
		t.Errorf("new task should be open task: %+v", task)
	}
	if task.CreatedAt != task.UpdatedAt {
		t.Errorf("timestamps should match on creation: %+v", task)
	}

	memo := NewMemoItem("memo content")
	if memo.IsTask() {
		t.Errorf("new memo should not be a task: %+v", memo)
	}
}

func TestNowISOFormat(t *testing.T) {
	pattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$`)
	if now := NowISO(); !pattern.MatchString(now) {
		t.Errorf("NowISO() = %q, want JS toISOString format", now)
	}
}

func TestParseDateStrict(t *testing.T) {
	valid := []string{"2026-07-05", "2000-01-31"}
	for _, value := range valid {
		if !IsDateString(value) {
			t.Errorf("IsDateString(%q) = false, want true", value)
		}
	}

	invalid := []string{"", "2026-7-5", "2026/07/05", "not-a-date", "2026-13-01", "20260705"}
	for _, value := range invalid {
		if IsDateString(value) {
			t.Errorf("IsDateString(%q) = true, want false", value)
		}
	}
}

func TestNewLogMarshalsEmptyItems(t *testing.T) {
	data, err := json.Marshal(NewLog())
	if err != nil {
		t.Fatal(err)
	}
	if want := `"items":[]`; !regexp.MustCompile(regexp.QuoteMeta(want)).Match(data) {
		t.Errorf("new log should marshal items as [], got %s", data)
	}
}
