package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rn404/nippo-cli/internal/logfile"
	"github.com/rn404/nippo-cli/internal/model"
)

// execute runs the root command with args and returns combined output.
func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()

	root := newRootCommand()
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)

	err := root.Execute()
	return buf.String(), err
}

// openTaskHashes extracts the hash of every open (unchecked) task
// line in a `sava list` timeline, e.g.
// "- [ ] 09:30 fix bug (`7ba24aef`)". It matches the "[ ]" marker
// only at its fixed position right after the leading bullet (not
// anywhere in the line, and not "[x]" for a closed task) and reads
// the hash from the LAST backtick-wrapped parenthesized group, so
// item content that happens to contain "[ ]" or literal parentheses
// doesn't produce a false match.
func openTaskHashes(list string) []string {
	const markerPrefix = "- [ ]"
	var hashes []string
	for _, line := range strings.Split(list, "\n") {
		if !strings.HasPrefix(line, markerPrefix) {
			continue
		}
		openParen, closeParen := strings.LastIndex(line, "(`"), strings.LastIndex(line, "`)")
		if openParen == -1 || closeParen == -1 || closeParen < openParen {
			continue
		}
		hashes = append(hashes, line[openParen+2:closeParen])
	}
	return hashes
}

// TestOpenTaskHashesIgnoresContentThatLooksLikeAMarker guards against
// two false-match bugs found in review: a memo whose content contains
// the literal substring "[ ]" must not be mistaken for an open task,
// and a task whose content contains parentheses before the trailing
// (hash) must still yield the real hash, not the content's own text.
func TestOpenTaskHashesIgnoresContentThatLooksLikeAMarker(t *testing.T) {
	list := "- 09:12 use [ ] for checkboxes (`aaaa1111`)\n" +
		"- [ ] 09:30 call (urgent) client (`bbbb2222`)\n"

	got := openTaskHashes(list)
	if len(got) != 1 || got[0] != "bbbb2222" {
		t.Errorf("openTaskHashes = %+v, want exactly [bbbb2222]", got)
	}
}

// addedHash extracts the hash from a "sava add"/"sava todo" output's
// "> <content> (<time>) <hash>[<tags>]" confirmation line, taking the
// last whitespace-separated field so it still works when the content
// itself contains spaces.
func addedHash(out string) string {
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "> ") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				return fields[len(fields)-1]
			}
		}
	}
	return ""
}

// writeFakeEditor writes an executable shell script at a path under
// dir and returns that path, for use as $EDITOR in editor-mode tests.
func writeFakeEditor(t *testing.T, dir, name, script string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func mustExecute(t *testing.T, args ...string) string {
	t.Helper()

	out, err := execute(t, args...)
	if err != nil {
		t.Fatalf("sava %s: %v", strings.Join(args, " "), err)
	}
	return out
}

func TestVersion(t *testing.T) {
	out := mustExecute(t, "--version")
	if !strings.Contains(out, version) {
		t.Errorf("version output = %q, want to contain %q", out, version)
	}
	if version == "" {
		t.Error("embedded version should not be empty")
	}
}

func TestAddListFlow(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	mustExecute(t, "todo", "buy cabbage")
	mustExecute(t, "add", "shrimp memo")

	out := mustExecute(t, "list")
	for _, want := range []string{"[ ] ", "buy cabbage", "shrimp memo"} {
		if !strings.Contains(out, want) {
			t.Errorf("list output should contain %q:\n%s", want, out)
		}
	}

	stat := mustExecute(t, "list", "-s")
	if !strings.Contains(stat, "Task: 1 (unfinished: 1), Memo: 1") {
		t.Errorf("list -s output = %q", stat)
	}
}

func TestAddOutputsHash(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	out := mustExecute(t, "add", "buy cabbage")
	if !strings.Contains(out, "Added!!") {
		t.Errorf("add output should confirm the addition:\n%s", out)
	}
}

func TestStartFlow(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	mustExecute(t, "todo", "-s", "slice cabbage")

	out := mustExecute(t, "list")
	if !strings.Contains(out, "[ ] `in-progress`") || !strings.Contains(out, "slice cabbage") {
		t.Errorf("todo added with -s should be shown as started:\n%s", out)
	}

	if _, err := execute(t, "start", "no-such-hash"); err == nil {
		t.Error("start with an unknown hash should fail")
	}
}

func TestEndMultipleHashes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	mustExecute(t, "todo", "first task")
	mustExecute(t, "todo", "second task")

	list := mustExecute(t, "list")
	hashes := openTaskHashes(list)
	if len(hashes) != 2 {
		t.Fatalf("hashes = %+v, want 2:\n%s", hashes, list)
	}

	out := mustExecute(t, "end", hashes[0], hashes[1])
	if got := strings.Count(out, "Finished!!"); got != 2 {
		t.Errorf("Finished!! count = %d, want 2:\n%s", got, out)
	}

	if _, err := execute(t, "end", "no-such-hash"); err == nil {
		t.Error("end with an unknown hash should fail")
	}
}

// TestTodoContentCanBeStartOrEnd guards against regressing to nesting
// start/end as todo subcommands, which made it impossible to create a
// TODO whose entire content is literally "start" or "end".
func TestTodoContentCanBeStartOrEnd(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	for _, content := range []string{"start", "end"} {
		out := mustExecute(t, "todo", content)
		if !strings.Contains(out, "Added!!") {
			t.Errorf("todo %q should create an item, not dispatch to a subcommand:\n%s", content, out)
		}
	}
}

func TestTagFlow(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	mustExecute(t, "add", "-t", "cabbage,food", "buy cabbage")

	out := mustExecute(t, "list", "-t", "cabbage")
	if !strings.Contains(out, "buy cabbage") || !strings.Contains(out, "#cabbage #food") {
		t.Errorf("tagged item should be listed with tags:\n%s", out)
	}

	out = mustExecute(t, "list", "-t", "no-such-tag")
	if strings.Contains(out, "buy cabbage") {
		t.Errorf("unmatched tag filter should hide the item:\n%s", out)
	}

	out = mustExecute(t, "tag", "--list")
	if !strings.Contains(out, "- cabbage (1)") || !strings.Contains(out, "- food (1)") {
		t.Errorf("tag --list output:\n%s", out)
	}

	if _, err := execute(t, "tag", "only-hash"); err == nil {
		t.Error("tag without tags should fail")
	}
}

func TestDiffFlow(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	mustExecute(t, "todo", "first task")
	mustExecute(t, "todo", "second task")

	list := mustExecute(t, "list")
	hashes := openTaskHashes(list)
	if len(hashes) != 2 {
		t.Fatalf("hashes = %+v, want 2:\n%s", hashes, list)
	}
	today := model.Today()
	refA, refB := today+":"+hashes[0], today+":"+hashes[1]

	out := mustExecute(t, "diff", refA+"..."+refB)
	if !strings.Contains(out, "Diff...") || !strings.Contains(out, "Elapsed: ") {
		t.Errorf("diff output:\n%s", out)
	}

	// Two-argument form works as well.
	out = mustExecute(t, "diff", refA, refB)
	if !strings.Contains(out, "Elapsed: ") {
		t.Errorf("two-arg diff output:\n%s", out)
	}

	if _, err := execute(t, "diff", "lonely-ref"); err == nil {
		t.Error("diff without a separator should fail")
	}
	if _, err := execute(t, "diff", hashes[0], hashes[1]); err == nil {
		t.Error("diff with bare hashes (no date) should fail")
	}
	if _, err := execute(t, "diff", refA+"..."+today+":no-such-hash"); err == nil {
		t.Error("diff with an unknown hash should fail")
	}
}

func TestClearAllWithYes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	mustExecute(t, "add", "temporary")
	mustExecute(t, "clear", "-a", "--yes")

	out := mustExecute(t, "list", "-a")
	if strings.Contains(out, "-Task") || strings.Count(out, "\n- ") > 0 {
		t.Errorf("no log files should remain after clear -a --yes:\n%s", out)
	}
}

// TestCarryFlow exercises Phase C end-to-end through the CLI: an
// unfinished TODO left over from a previous day should reappear in
// today's list under a new hash, with that source day now frozen,
// the moment the first command of a new day is run.
func TestCarryFlow(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := logfile.Dir()

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	file, err := logfile.Get(dir, yesterday)
	if err != nil {
		t.Fatal(err)
	}
	open := false
	file.Body.Items = []model.Item{
		{Hash: "open1111", Content: "unfinished todo", CreatedAt: "2026-01-01T00:00:00.000Z", UpdatedAt: "2026-01-01T00:00:00.000Z", Closed: &open},
	}
	if err := logfile.Update(dir, yesterday, file.Body); err != nil {
		t.Fatal(err)
	}

	out := mustExecute(t, "add", "today's memo")
	if !strings.Contains(out, "Carried 1 items from "+yesterday+" (that day is now frozen).") {
		t.Errorf("carry notice missing from CLI output:\n%s", out)
	}

	list := mustExecute(t, "list")
	if !strings.Contains(list, "[ ] ") || !strings.Contains(list, "unfinished todo") {
		t.Errorf("carried todo should appear in today's list:\n%s", list)
	}

	stat := mustExecute(t, "list", yesterday, "-s")
	if !strings.Contains(stat, yesterday+"*") {
		t.Errorf("yesterday should show as frozen in stats:\n%s", stat)
	}
}

// TestListYesterdayKeyword proves "yesterday" works as a literal date
// argument on the actual CLI, not just at the command-layer.
func TestListYesterdayKeyword(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := logfile.Dir()

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	file, err := logfile.Get(dir, yesterday)
	if err != nil {
		t.Fatal(err)
	}
	file.Body.Items = []model.Item{
		{Hash: "aaaa1111", Content: "yesterday's memo", CreatedAt: "2026-01-01T00:00:00.000Z", UpdatedAt: "2026-01-01T00:00:00.000Z"},
	}
	if err := logfile.Update(dir, yesterday, file.Body); err != nil {
		t.Fatal(err)
	}

	out := mustExecute(t, "list", "yesterday")
	if !strings.Contains(out, "Log for "+yesterday+" are...") || !strings.Contains(out, "yesterday's memo") {
		t.Errorf("list yesterday output:\n%s", out)
	}
}

// TestListFullAndTaskFlags proves "--full"/"-f" and "--task" are
// registered on the list command and bridged to the right
// command.ListOptions fields: a closed task is hidden by default,
// shown with --full, and a memo is excluded with --task.
func TestListFullAndTaskFlags(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	mustExecute(t, "todo", "buy cabbage")
	mustExecute(t, "add", "shrimp memo")

	list := mustExecute(t, "list")
	hashes := openTaskHashes(list)
	if len(hashes) != 1 {
		t.Fatalf("hashes = %+v, want 1:\n%s", hashes, list)
	}
	mustExecute(t, "end", hashes[0])

	out := mustExecute(t, "list")
	if strings.Contains(out, "buy cabbage") {
		t.Errorf("closed task should be hidden by default:\n%s", out)
	}
	if !strings.Contains(out, "shrimp memo") {
		t.Errorf("memo should still be shown by default:\n%s", out)
	}

	out = mustExecute(t, "list", "--full")
	if !strings.Contains(out, "buy cabbage") {
		t.Errorf("--full should include the closed task:\n%s", out)
	}

	out = mustExecute(t, "list", "-f")
	if !strings.Contains(out, "buy cabbage") {
		t.Errorf("-f should include the closed task:\n%s", out)
	}

	out = mustExecute(t, "list", "--task")
	if strings.Contains(out, "shrimp memo") {
		t.Errorf("--task should exclude memos:\n%s", out)
	}

	out = mustExecute(t, "list", "--full", "--task")
	if !strings.Contains(out, "buy cabbage") || strings.Contains(out, "shrimp memo") {
		t.Errorf("--full --task should show the closed task but exclude memos:\n%s", out)
	}
}

// TestFullTextFlag proves "--full-text" is registered on the list
// command and bridged to command.ListOptions.FullText: a multi-line
// memo shows only its first line by default, and its full content
// (all lines) once --full-text is given.
func TestFullTextFlag(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	mustExecute(t, "add", "first line\nsecond line")

	out := mustExecute(t, "list")
	if !strings.Contains(out, "first line") {
		t.Errorf("list output should contain the first line:\n%s", out)
	}
	if strings.Contains(out, "second line") {
		t.Errorf("list without --full-text should not show the second line:\n%s", out)
	}

	out = mustExecute(t, "list", "--full-text")
	if !strings.Contains(out, "first line\nsecond line") {
		t.Errorf("list --full-text should show the full multi-line content:\n%s", out)
	}
}

func TestInvalidDateFails(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if _, err := execute(t, "list", "not-a-date"); err == nil {
		t.Error("list with an invalid date should fail")
	}
}

func TestUnknownHashFails(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if _, err := execute(t, "end", "no-such-hash"); err == nil {
		t.Error("end with an unknown hash should fail")
	}
}

// TestEditDirectMode proves "sava edit <hash> <new content>" rewrites
// the item's content immediately, without touching $EDITOR at all.
func TestEditDirectMode(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	added := mustExecute(t, "add", "original content")
	hash := addedHash(added)
	if hash == "" {
		t.Fatalf("could not extract hash from add output:\n%s", added)
	}

	out := mustExecute(t, "edit", hash, "fixed content")
	if !strings.Contains(out, "Edited!!") || !strings.Contains(out, "fixed content") {
		t.Errorf("edit output = %q", out)
	}

	list := mustExecute(t, "list")
	if !strings.Contains(list, "fixed content") {
		t.Errorf("list after edit should show the new content:\n%s", list)
	}
	if strings.Contains(list, "original content") {
		t.Errorf("list after edit should not show the old content:\n%s", list)
	}
	if !strings.Contains(list, hash) {
		t.Errorf("list after edit should still show hash %q:\n%s", hash, list)
	}
}

// TestEditEditorMode proves "sava edit <hash>" (content omitted)
// launches $EDITOR against a temp file seeded with the current
// content, and applies whatever that script saves.
func TestEditEditorMode(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	added := mustExecute(t, "add", "original content")
	hash := addedHash(added)
	if hash == "" {
		t.Fatalf("could not extract hash from add output:\n%s", added)
	}

	script := writeFakeEditor(t, t.TempDir(), "fake-editor.sh", "#!/bin/sh\necho \"edited via script\" > \"$1\"\n")
	t.Setenv("EDITOR", script)

	out := mustExecute(t, "edit", hash)
	if !strings.Contains(out, "Edited!!") || !strings.Contains(out, "edited via script") {
		t.Errorf("edit output = %q", out)
	}

	list := mustExecute(t, "list")
	if !strings.Contains(list, "edited via script") {
		t.Errorf("list after editor-mode edit should show the new content:\n%s", list)
	}
	if strings.Contains(list, "original content") {
		t.Errorf("list after editor-mode edit should not show the old content:\n%s", list)
	}
}

// TestEditEditorMode_AbortsOnUnchanged proves that when the $EDITOR
// script leaves the temp file's content unchanged (a no-op save), the
// edit is aborted: no error, an abort message, and the item's content
// stays exactly as it was.
func TestEditEditorMode_AbortsOnUnchanged(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	added := mustExecute(t, "add", "original content")
	hash := addedHash(added)
	if hash == "" {
		t.Fatalf("could not extract hash from add output:\n%s", added)
	}

	script := writeFakeEditor(t, t.TempDir(), "noop-editor.sh", "#!/bin/sh\ntrue\n")
	t.Setenv("EDITOR", script)

	out := mustExecute(t, "edit", hash)
	if !strings.Contains(out, "Edit aborted") {
		t.Errorf("edit output should report an abort:\n%s", out)
	}
	if strings.Contains(out, "Edited!!") {
		t.Errorf("edit output should not confirm an edit on abort:\n%s", out)
	}

	list := mustExecute(t, "list")
	if !strings.Contains(list, "original content") {
		t.Errorf("content should be unchanged after an aborted edit:\n%s", list)
	}
}

// TestAddEditorMode_CreatesMemo proves "sava add" (content argument
// omitted) launches $EDITOR against an empty temp file and creates a
// memo from whatever content that script saves.
func TestAddEditorMode_CreatesMemo(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	script := writeFakeEditor(t, t.TempDir(), "fake-editor.sh", "#!/bin/sh\necho \"buy cabbage via editor\" > \"$1\"\n")
	t.Setenv("EDITOR", script)

	out := mustExecute(t, "add")
	if !strings.Contains(out, "Added!!") || !strings.Contains(out, "buy cabbage via editor") {
		t.Errorf("add output = %q", out)
	}

	list := mustExecute(t, "list")
	if !strings.Contains(list, "buy cabbage via editor") {
		t.Errorf("list after editor-mode add should show the new content:\n%s", list)
	}
}

// TestAddEditorMode_AbortsWhenEmpty proves that saving the editor's
// temp file with no content at all aborts the add: no error, an abort
// message, and no memo created.
func TestAddEditorMode_AbortsWhenEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	script := writeFakeEditor(t, t.TempDir(), "noop-editor.sh", "#!/bin/sh\ntrue\n")
	t.Setenv("EDITOR", script)

	out := mustExecute(t, "add")
	if !strings.Contains(out, "Add aborted") {
		t.Errorf("add output should report an abort:\n%s", out)
	}
	if strings.Contains(out, "Added!!") {
		t.Errorf("add output should not confirm an add on abort:\n%s", out)
	}

	list := mustExecute(t, "list")
	if !strings.Contains(list, "There is no body...") {
		t.Errorf("no memo should be created after an aborted add:\n%s", list)
	}
}

// TestAddEditorMode_AbortsWhenWhitespaceOnly proves that saving the
// editor's temp file with only whitespace also aborts the add, even
// though editor.Resolve's own ok-check (candidate == "" or ==
// current) alone would not catch it (current is always "" here, so a
// whitespace candidate is != current and != "").
func TestAddEditorMode_AbortsWhenWhitespaceOnly(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	script := writeFakeEditor(t, t.TempDir(), "whitespace-editor.sh", "#!/bin/sh\nprintf '   \\n\\t \\n' > \"$1\"\n")
	t.Setenv("EDITOR", script)

	out := mustExecute(t, "add")
	if !strings.Contains(out, "Add aborted") {
		t.Errorf("add output should report an abort for a whitespace-only save:\n%s", out)
	}

	list := mustExecute(t, "list")
	if !strings.Contains(list, "There is no body...") {
		t.Errorf("no memo should be created after a whitespace-only aborted add:\n%s", list)
	}
}

// TestAddEditorMode_PropagatesEditorError proves that a non-zero exit
// from the $EDITOR script fails the command and creates no memo.
func TestAddEditorMode_PropagatesEditorError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	script := writeFakeEditor(t, t.TempDir(), "failing-editor.sh", "#!/bin/sh\nexit 1\n")
	t.Setenv("EDITOR", script)

	if _, err := execute(t, "add"); err == nil {
		t.Error("add should fail when the editor exits non-zero")
	}

	list := mustExecute(t, "list")
	if !strings.Contains(list, "There is no body...") {
		t.Errorf("no memo should be created when the editor errors:\n%s", list)
	}
}

// TestAddEditorMode_WithTagOption proves --tag is applied to a memo
// created via the editor path, same as direct mode.
func TestAddEditorMode_WithTagOption(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	script := writeFakeEditor(t, t.TempDir(), "fake-editor.sh", "#!/bin/sh\necho \"tagged via editor\" > \"$1\"\n")
	t.Setenv("EDITOR", script)

	mustExecute(t, "add", "-t", "cabbage")

	out := mustExecute(t, "list", "-t", "cabbage")
	if !strings.Contains(out, "tagged via editor") {
		t.Errorf("editor-mode add with --tag should tag the created memo:\n%s", out)
	}
}

// TestAddEditorMode_AbortedSkipsTagOption proves that --tag processing
// does not run when the editor path aborts: no tag ends up registered
// in the tag index at all.
func TestAddEditorMode_AbortedSkipsTagOption(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	script := writeFakeEditor(t, t.TempDir(), "noop-editor.sh", "#!/bin/sh\ntrue\n")
	t.Setenv("EDITOR", script)

	mustExecute(t, "add", "-t", "cabbage")

	out := mustExecute(t, "tag", "--list")
	if !strings.Contains(out, "There is no tags...") {
		t.Errorf("aborted editor-mode add should not register any tag:\n%s", out)
	}
}

// TestAddDirectMode_UnaffectedByEditorChange proves content given
// directly still bypasses the editor entirely: a broken/unusable
// $EDITOR does not stop the add from succeeding, because it is never
// invoked.
func TestAddDirectMode_UnaffectedByEditorChange(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("EDITOR", "/no/such/editor-binary")

	out := mustExecute(t, "add", "direct content")
	if !strings.Contains(out, "Added!!") || !strings.Contains(out, "direct content") {
		t.Errorf("add output = %q", out)
	}
}
