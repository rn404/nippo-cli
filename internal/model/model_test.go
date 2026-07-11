package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestFormatSampleRoundTrip pins the storage format: the sample under
// testdata/log-format/ is the spec, and marshaling must reproduce it
// byte for byte. Item 1 carries only the required fields, proving that
// files written by older versions (before startedAt/tags) still load.
func TestFormatSampleRoundTrip(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "log-format", "2026-07-05.json")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read format sample: %v", err)
	}

	var body Log
	if err := json.Unmarshal(original, &body); err != nil {
		t.Fatalf("unmarshal format sample: %v", err)
	}

	if len(body.Items) != 3 {
		t.Fatalf("items = %d, want 3", len(body.Items))
	}
	if item := body.Items[0]; !item.IsTask() || !item.IsClosed() || !item.IsStarted() || len(item.Tags) != 2 {
		t.Errorf("item 0 should be a closed, started task with 2 tags: %+v", item)
	}
	if item := body.Items[1]; !item.IsTask() || item.IsClosed() || item.IsStarted() || item.Tags != nil {
		t.Errorf("item 1 should be an open task without optional fields: %+v", item)
	}
	if item := body.Items[2]; item.IsTask() || !item.HasTag("shrimp") {
		t.Errorf("item 2 should be a memo tagged shrimp: %+v", item)
	}

	remarshaled, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(remarshaled) != strings.TrimRight(string(original), "\n") {
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
