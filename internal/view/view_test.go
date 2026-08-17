package view

import (
	"strings"
	"testing"
	"time"

	"github.com/rn404/nippo-cli/internal/model"
)

func TestTimeline(t *testing.T) {
	closed := true
	open := false
	startedAt := "2026-07-05T09:00:00.000Z"
	items := []model.Item{
		{Hash: "aaaa1111", Content: "buy cabbage", CreatedAt: "2026-07-05T08:43:04.971Z", Closed: &closed},
		{Hash: "bbbb2222", Content: "feed the shrimp", CreatedAt: "2026-07-05T08:43:05.026Z", Closed: &open},
		{Hash: "dddd4444", Content: "slice cabbage", CreatedAt: "2026-07-05T08:43:05.050Z", StartedAt: &startedAt, Closed: &open},
		{Hash: "cccc3333", Content: "shrimp looks happy today", CreatedAt: "2026-07-05T08:43:05.073Z", Tags: []string{"shrimp", "pet"}},
	}

	var buf strings.Builder
	Timeline(&buf, items)
	out := buf.String()

	for _, want := range []string{
		"- ",
		"[x] buy cabbage (aaaa1111)",
		"[ ] feed the shrimp (bbbb2222)",
		"[>] slice cabbage (dddd4444)",
		"・ shrimp looks happy today (cccc3333) #shrimp #pet",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output should contain %q:\n%s", want, out)
		}
	}

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("lines = %d, want 4 (one per item):\n%s", len(lines), out)
	}
	if !strings.Contains(lines[3], "cccc3333") {
		t.Errorf("last line should be the latest item (shrimp memo): %q", lines[3])
	}
}

func TestTimelineEmpty(t *testing.T) {
	var buf strings.Builder
	Timeline(&buf, nil)
	if !strings.Contains(buf.String(), "There is no body...") {
		t.Errorf("empty timeline output = %q", buf.String())
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

func TestCarried(t *testing.T) {
	var buf strings.Builder
	Carried(&buf, 2, "2026-07-24")
	out := buf.String()
	if !strings.Contains(out, "Carried 2 items from 2026-07-24 (that day is now frozen).") {
		t.Errorf("Carried output = %q", out)
	}
}

func TestAdded(t *testing.T) {
	var buf strings.Builder
	Added(&buf, model.Item{Hash: "1ed29de4", Content: "review PR #123", CreatedAt: "2026-07-05T08:43:04.971Z", Tags: []string{"cli"}})
	out := buf.String()
	if !strings.Contains(out, "Added!!") || !strings.Contains(out, "> review PR #123 (") || !strings.Contains(out, "1ed29de4") || !strings.Contains(out, "#cli") {
		t.Errorf("Added output = %q", out)
	}
}

func TestDeleted(t *testing.T) {
	var buf strings.Builder
	Deleted(&buf, model.Item{Content: "buy cabbage", CreatedAt: "2026-07-05T08:43:04.971Z"})
	out := buf.String()
	if !strings.Contains(out, "Deleted!!") || !strings.Contains(out, "> buy cabbage (") {
		t.Errorf("Deleted output = %q", out)
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
