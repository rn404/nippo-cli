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
	Timeline(&buf, items, false)
	out := buf.String()

	for _, want := range []string{
		"- [x] ",
		"buy cabbage (`aaaa1111`)",
		"- [ ] ",
		"feed the shrimp (`bbbb2222`)",
		"- [ ] `in-progress` ",
		"slice cabbage (`dddd4444`)",
		"shrimp looks happy today (`cccc3333`) #shrimp #pet",
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
	if strings.Contains(lines[3], "[") {
		t.Errorf("memo line should contain no checkbox syntax: %q", lines[3])
	}
}

func TestTimelineEmpty(t *testing.T) {
	var buf strings.Builder
	Timeline(&buf, nil, false)
	if !strings.Contains(buf.String(), "There is no body...") {
		t.Errorf("empty timeline output = %q", buf.String())
	}
}

func TestTimeline_OpenTaskChecklistFormat(t *testing.T) {
	open := false
	item := model.Item{Hash: "bbbb2222", Content: "feed the shrimp", CreatedAt: "2026-07-05T08:43:05.026Z", Closed: &open}

	var buf strings.Builder
	Timeline(&buf, []model.Item{item}, false)
	out := strings.TrimRight(buf.String(), "\n")

	want := "- [ ] " + formatTime(item.CreatedAt) + " feed the shrimp (`bbbb2222`)"
	if out != want {
		t.Errorf("Timeline open task = %q, want %q", out, want)
	}
}

func TestTimeline_StartedTaskInProgressToken(t *testing.T) {
	open := false
	startedAt := "2026-07-05T09:00:00.000Z"
	item := model.Item{Hash: "dddd4444", Content: "slice cabbage", CreatedAt: "2026-07-05T08:43:05.050Z", StartedAt: &startedAt, Closed: &open}

	var buf strings.Builder
	Timeline(&buf, []model.Item{item}, false)
	out := strings.TrimRight(buf.String(), "\n")

	want := "- [ ] `in-progress` " + formatTime(item.CreatedAt) + " slice cabbage (`dddd4444`)"
	if out != want {
		t.Errorf("Timeline started task = %q, want %q", out, want)
	}
}

func TestTimeline_ClosedTaskChecklistFormat(t *testing.T) {
	closed := true
	item := model.Item{Hash: "aaaa1111", Content: "buy cabbage", CreatedAt: "2026-07-05T08:43:04.971Z", Closed: &closed}

	var buf strings.Builder
	Timeline(&buf, []model.Item{item}, false)
	out := strings.TrimRight(buf.String(), "\n")

	want := "- [x] " + formatTime(item.CreatedAt) + " buy cabbage (`aaaa1111`)"
	if out != want {
		t.Errorf("Timeline closed task = %q, want %q", out, want)
	}
}

func TestTimeline_MemoNoCheckbox(t *testing.T) {
	item := model.Item{Hash: "cccc3333", Content: "shrimp looks happy today", CreatedAt: "2026-07-05T08:43:05.073Z"}

	var buf strings.Builder
	Timeline(&buf, []model.Item{item}, false)
	out := strings.TrimRight(buf.String(), "\n")

	want := "- " + formatTime(item.CreatedAt) + " shrimp looks happy today (`cccc3333`)"
	if out != want {
		t.Errorf("Timeline memo = %q, want %q", out, want)
	}
	if strings.Contains(out, "[") {
		t.Errorf("memo line should contain no checkbox syntax: %q", out)
	}
}

func TestTimeline_TagsAfterHash(t *testing.T) {
	item := model.Item{Hash: "cccc3333", Content: "shrimp looks happy today", CreatedAt: "2026-07-05T08:43:05.073Z", Tags: []string{"shrimp", "pet"}}

	var buf strings.Builder
	Timeline(&buf, []model.Item{item}, false)
	out := strings.TrimRight(buf.String(), "\n")

	want := "- " + formatTime(item.CreatedAt) + " shrimp looks happy today (`cccc3333`) #shrimp #pet"
	if out != want {
		t.Errorf("Timeline tags = %q, want %q", out, want)
	}
	if !strings.HasSuffix(out, "#shrimp #pet") {
		t.Errorf("tags should follow the hash token: %q", out)
	}
}

func TestTimeline_MultilineContentShowsFirstLineByDefault(t *testing.T) {
	// Short multi-line content (collapsed length <= 50 runes): newlines
	// are collapsed to spaces, all lines are shown, and no ellipsis is
	// appended. Requirements 1.1, 1.3.
	item := model.Item{Hash: "eeee5555", Content: "buy cabbage\nand also shrimp\nfor dinner", CreatedAt: "2026-07-05T08:43:05.073Z"}

	var buf strings.Builder
	Timeline(&buf, []model.Item{item}, false)
	out := buf.String()

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("lines = %d, want 1 (one line per item): %q", len(lines), out)
	}
	want := "buy cabbage and also shrimp for dinner (`eeee5555`)"
	if !strings.Contains(out, want) {
		t.Errorf("default output should collapse newlines to spaces and show the full (short) content: got %q, want to contain %q", out, want)
	}
	if strings.Contains(out, ellipsis) {
		t.Errorf("short content should not be truncated with an ellipsis: %q", out)
	}
}

func TestTimeline_LongContentTruncatedWithEllipsisByDefault(t *testing.T) {
	// Collapsed content over 50 runes: shown as the first 50 runes plus
	// an ellipsis. Requirements 1.1, 1.2.
	longContent := strings.Repeat("a", 60)
	item := model.Item{Hash: "ffff6666", Content: longContent, CreatedAt: "2026-07-05T08:43:05.073Z"}

	var buf strings.Builder
	Timeline(&buf, []model.Item{item}, false)
	out := strings.TrimRight(buf.String(), "\n")

	wantContent := strings.Repeat("a", 50) + ellipsis
	want := "- " + formatTime(item.CreatedAt) + " " + wantContent + " (`ffff6666`)"
	if out != want {
		t.Errorf("Timeline long content = %q, want %q", out, want)
	}
}

func TestTimeline_FullTextCollapsesNewlinesButShowsAllContent(t *testing.T) {
	// --full-text: newlines are collapsed to spaces (so the item is
	// still exactly one line), but the full content is shown with no
	// truncation and no ellipsis, even past 50 characters. Requirements
	// 2.1, 2.2, 2.3.
	item := model.Item{Hash: "eeee5555", Content: "buy cabbage\nand also shrimp\nfor dinner, which is a long enough sentence to pass fifty characters", CreatedAt: "2026-07-05T08:43:05.073Z"}

	var buf strings.Builder
	Timeline(&buf, []model.Item{item}, true)
	out := buf.String()

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("lines = %d, want 1 (one line per item): %q", len(lines), out)
	}
	for _, want := range []string{"buy cabbage", "and also shrimp", "for dinner, which is a long enough sentence to pass fifty characters", "(`eeee5555`)"} {
		if !strings.Contains(out, want) {
			t.Errorf("full-text output should contain %q:\n%s", want, out)
		}
	}
	if strings.Contains(strings.TrimRight(out, "\n"), "\n") {
		t.Errorf("full-text output should have newlines collapsed to spaces within the single line: %q", out)
	}
	if strings.Contains(out, ellipsis) {
		t.Errorf("full-text output should never be truncated with an ellipsis: %q", out)
	}
}

func TestSummarizeContent(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		fullText bool
		want     string
	}{
		{"single line short, default", "single line", false, "single line"},
		{"newline within limit, default: collapsed and untruncated", "first\nsecond", false, "first second"},
		{"consecutive newlines collapse to a single space", "paragraph A\n\nparagraph B", false, "paragraph A paragraph B"},
		{"exactly 50 runes after collapse: no ellipsis", strings.Repeat("a", 50), false, strings.Repeat("a", 50)},
		{"51 runes after collapse: truncated with ellipsis", strings.Repeat("a", 51), false, strings.Repeat("a", 50) + ellipsis},
		{"full-text: newlines collapsed, no truncation past 50", strings.Repeat("a", 30) + "\n\n" + strings.Repeat("b", 30), true, strings.Repeat("a", 30) + " " + strings.Repeat("b", 30)},
		{"full-text: consecutive newlines collapse to a single space", "paragraph A\n\nparagraph B", true, "paragraph A paragraph B"},
		{"empty content", "", false, ""},
	}
	for _, c := range cases {
		if got := summarizeContent(c.content, c.fullText); got != c.want {
			t.Errorf("%s: summarizeContent(%q, %v) = %q, want %q", c.name, c.content, c.fullText, got, c.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		name     string
		s        string
		maxRunes int
		want     string
	}{
		{"under limit: unchanged", "short", 50, "short"},
		{"exactly at limit: unchanged, no ellipsis", strings.Repeat("a", 50), 50, strings.Repeat("a", 50)},
		{"over limit: cut and ellipsis appended", strings.Repeat("a", 60), 50, strings.Repeat("a", 50) + ellipsis},
		{"multi-byte runes (Japanese) truncated without corrupting characters", strings.Repeat("日", 60), 50, strings.Repeat("日", 50) + ellipsis},
	}
	for _, c := range cases {
		if got := truncate(c.s, c.maxRunes); got != c.want {
			t.Errorf("%s: truncate(%q, %d) = %q, want %q", c.name, c.s, c.maxRunes, got, c.want)
		}
	}
}

func TestChecklistPrefix(t *testing.T) {
	cases := []struct {
		status model.Status
		want   string
	}{
		{model.StatusMemo, "-"},
		{model.StatusOpen, "- [ ]"},
		{model.StatusStarted, "- [ ] `in-progress`"},
		{model.StatusClosed, "- [x]"},
	}
	for _, c := range cases {
		if got := checklistPrefix(c.status); got != c.want {
			t.Errorf("checklistPrefix(%v) = %q, want %q", c.status, got, c.want)
		}
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

func TestEdited(t *testing.T) {
	var buf strings.Builder
	Edited(&buf, model.Item{Content: "buy cabbage", CreatedAt: "2026-07-05T08:43:04.971Z"})
	out := buf.String()
	if !strings.Contains(out, "Edited!!") || !strings.Contains(out, "> buy cabbage (") {
		t.Errorf("Edited output = %q", out)
	}
}

func TestEditAborted(t *testing.T) {
	var buf strings.Builder
	EditAborted(&buf)
	if buf.Len() == 0 {
		t.Error("EditAborted should print a message")
	}
}

func TestAddAborted(t *testing.T) {
	var buf strings.Builder
	AddAborted(&buf)
	if !strings.Contains(buf.String(), "Add aborted: nothing to save.") {
		t.Errorf("AddAborted output = %q", buf.String())
	}
}

func TestTodoAborted(t *testing.T) {
	var buf strings.Builder
	TodoAborted(&buf)
	if !strings.Contains(buf.String(), "Todo aborted: nothing to save.") {
		t.Errorf("TodoAborted output = %q", buf.String())
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
