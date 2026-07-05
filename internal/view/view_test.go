package view

import (
	"strings"
	"testing"

	"github.com/rn404/nippo-cli/internal/model"
)

func TestItemList(t *testing.T) {
	closed := true
	open := false
	tasks := []model.Item{
		{Hash: "aaaa1111", Content: "buy cabbage", CreatedAt: "2026-07-05T08:43:04.971Z", Closed: &closed},
		{Hash: "bbbb2222", Content: "feed the shrimp", CreatedAt: "2026-07-05T08:43:05.026Z", Closed: &open},
	}
	memos := []model.Item{
		{Hash: "cccc3333", Content: "shrimp looks happy today", CreatedAt: "2026-07-05T08:43:05.073Z"},
	}

	var buf strings.Builder
	ItemList(&buf, tasks, memos)
	out := buf.String()

	for _, want := range []string{
		"Task ->",
		"- [x] buy cabbage (",
		") aaaa1111",
		"- [ ] feed the shrimp (",
		"Memo ->",
		"- shrimp looks happy today (",
		") cccc3333",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output should contain %q:\n%s", want, out)
		}
	}
}

func TestItemListEmpty(t *testing.T) {
	var buf strings.Builder
	ItemList(&buf, nil, nil)
	if !strings.Contains(buf.String(), "There is no body...") {
		t.Errorf("empty list output = %q", buf.String())
	}
}

func TestFileStat(t *testing.T) {
	open := false
	tasks := []model.Item{{Hash: "a", Closed: &open}}
	memos := []model.Item{{Hash: "b"}, {Hash: "c"}}

	var buf strings.Builder
	FileStat(&buf, "2026-07-05", false, tasks, memos, 1)
	if got, want := buf.String(), "- 2026-07-05  Task: 1 (unfinished: 1), Memo: 2\n"; got != want {
		t.Errorf("FileStat = %q, want %q", got, want)
	}

	buf.Reset()
	FileStat(&buf, "2026-07-05", true, tasks, memos, 1)
	if !strings.Contains(buf.String(), "2026-07-05*") {
		t.Errorf("freezed mark missing: %q", buf.String())
	}
}

func TestFinishedTask(t *testing.T) {
	var buf strings.Builder
	FinishedTask(&buf, model.Item{Content: "buy cabbage", CreatedAt: "2026-07-05T08:43:04.971Z"})
	out := buf.String()
	if !strings.Contains(out, "Finished!!") || !strings.Contains(out, "> buy cabbage (") {
		t.Errorf("FinishedTask output = %q", out)
	}
}
