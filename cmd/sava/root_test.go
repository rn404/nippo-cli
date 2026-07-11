package main

import (
	"strings"
	"testing"
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

	mustExecute(t, "add", "buy cabbage")
	mustExecute(t, "add", "-m", "shrimp memo")

	out := mustExecute(t, "list")
	for _, want := range []string{"Task ->", "buy cabbage", "Memo ->", "shrimp memo"} {
		if !strings.Contains(out, want) {
			t.Errorf("list output should contain %q:\n%s", want, out)
		}
	}

	stat := mustExecute(t, "list", "-s")
	if !strings.Contains(stat, "Task: 1 (unfinished: 1), Memo: 1") {
		t.Errorf("list -s output = %q", stat)
	}
}

func TestStartFlow(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	mustExecute(t, "add", "-s", "slice cabbage")

	out := mustExecute(t, "list")
	if !strings.Contains(out, "[>] slice cabbage") {
		t.Errorf("task added with -s should be shown as started:\n%s", out)
	}

	if _, err := execute(t, "add", "-m", "-s", "impossible"); err == nil {
		t.Error("add -m -s should fail as mutually exclusive")
	}

	if _, err := execute(t, "start", "no-such-hash"); err == nil {
		t.Error("start with an unknown hash should fail")
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
