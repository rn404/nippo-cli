package view

import (
	"strings"
	"testing"
	"time"

	"github.com/rn404/nippo-cli/internal/model"
)

func TestItemList(t *testing.T) {
	closed := true
	open := false
	startedAt := "2026-07-05T09:00:00.000Z"
	tasks := []model.Item{
		{Hash: "aaaa1111", Content: "buy cabbage", CreatedAt: "2026-07-05T08:43:04.971Z", Closed: &closed},
		{Hash: "bbbb2222", Content: "feed the shrimp", CreatedAt: "2026-07-05T08:43:05.026Z", Closed: &open},
		{Hash: "dddd4444", Content: "slice cabbage", CreatedAt: "2026-07-05T08:43:05.050Z", StartedAt: &startedAt, Closed: &open},
	}
	memos := []model.Item{
		{Hash: "cccc3333", Content: "shrimp looks happy today", CreatedAt: "2026-07-05T08:43:05.073Z", Tags: []string{"shrimp", "pet"}},
	}

	var buf strings.Builder
	ItemList(&buf, tasks, memos)
	out := buf.String()

	for _, want := range []string{
		"Task ->",
		"- [x] buy cabbage (",
		") aaaa1111",
		"- [ ] feed the shrimp (",
		"- [>] slice cabbage (",
		"Memo ->",
		"- shrimp looks happy today (",
		") cccc3333 #shrimp #pet",
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

func TestStartedTask(t *testing.T) {
	var buf strings.Builder
	StartedTask(&buf, model.Item{Content: "buy cabbage", CreatedAt: "2026-07-05T08:43:04.971Z"})
	out := buf.String()
	if !strings.Contains(out, "Started!!") || !strings.Contains(out, "> buy cabbage (") {
		t.Errorf("StartedTask output = %q", out)
	}
}

func TestDiff(t *testing.T) {
	a := model.Item{Hash: "aaaa1111", Content: "buy cabbage", CreatedAt: "2026-07-05T10:00:00.000Z"}
	b := model.Item{Hash: "bbbb2222", Content: "feed the shrimp", CreatedAt: "2026-07-06T12:30:00.000Z"}

	var buf strings.Builder
	Diff(&buf, a, b, 26*time.Hour+30*time.Minute)
	out := buf.String()

	for _, want := range []string{
		"Diff...",
		"> buy cabbage (2026-07-05",
		") aaaa1111",
		"> feed the shrimp (2026-07-06",
		") bbbb2222",
		"Elapsed: 1d 2h 30m",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output should contain %q:\n%s", want, out)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	cases := map[time.Duration]string{
		0:                             "0s",
		45 * time.Second:              "45s",
		time.Minute + 10*time.Second:  "1m 10s",
		2 * time.Hour:                 "2h",
		26*time.Hour + 30*time.Minute: "1d 2h 30m",
		-(time.Hour + time.Second):    "1h 1s",
		500 * time.Millisecond:        "1s", // rounded
		24*time.Hour + 5*time.Minute:  "1d 5m",
		48*time.Hour + 59*time.Second: "2d 59s",
	}
	for d, want := range cases {
		if got := formatDuration(d); got != want {
			t.Errorf("formatDuration(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestTagsUpdated(t *testing.T) {
	var buf strings.Builder
	TagsUpdated(&buf, model.Item{Content: "buy cabbage", CreatedAt: "2026-07-05T08:43:04.971Z", Tags: []string{"cabbage"}})
	out := buf.String()
	if !strings.Contains(out, "Tags updated!!") || !strings.Contains(out, "#cabbage") {
		t.Errorf("TagsUpdated output = %q", out)
	}

	buf.Reset()
	TagsUpdated(&buf, model.Item{Content: "buy cabbage", CreatedAt: "2026-07-05T08:43:04.971Z"})
	if strings.Contains(buf.String(), "#") {
		t.Errorf("item without tags should print no tag marks: %q", buf.String())
	}
}
