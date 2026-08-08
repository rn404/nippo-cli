package main

import (
	"strings"
	"testing"

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
// line in a `sava list` timeline, e.g. "- 09:30 [ ] fix bug (7ba24aef)".
func openTaskHashes(list string) []string {
	var hashes []string
	for _, line := range strings.Split(list, "\n") {
		if !strings.Contains(line, "[ ]") {
			continue
		}
		open, close := strings.Index(line, "("), strings.Index(line, ")")
		if open == -1 || close == -1 || close < open {
			continue
		}
		hashes = append(hashes, line[open+1:close])
	}
	return hashes
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
	for _, want := range []string{"[ ] buy cabbage", "・ shrimp memo"} {
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
	if !strings.Contains(out, "[>] slice cabbage") {
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
