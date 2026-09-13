package editor

import (
	"errors"
	"os"
	"testing"
)

// writeLaunch returns a fake launch func that overwrites the temp file
// with newContent and reports success, simulating an editor session
// that saves newContent and exits cleanly.
func writeLaunch(newContent string) func(editorName, path string) error {
	return func(_, path string) error {
		return os.WriteFile(path, []byte(newContent), 0o644)
	}
}

// errLaunch returns a fake launch func that reports failure without
// touching the file, simulating an editor that exits non-zero.
func errLaunch(err error) func(editorName, path string) error {
	return func(_, _ string) error {
		return err
	}
}

func TestResolve_AppliesChangedNonEmptyContent(t *testing.T) {
	content, ok, err := Resolve("original", writeLaunch("updated\n"))
	if err != nil {
		t.Fatalf("Resolve returned unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("Resolve: ok = false, want true for changed non-empty content")
	}
	if content != "updated" {
		t.Fatalf("Resolve: content = %q, want %q (trailing newline stripped)", content, "updated")
	}
}

func TestResolve_AbortsWhenUnchanged(t *testing.T) {
	// Editor saves the file back with the exact same content (with the
	// usual single trailing newline an editor adds).
	content, ok, err := Resolve("original", writeLaunch("original\n"))
	if err != nil {
		t.Fatalf("Resolve returned unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("Resolve: ok = true, want false for unchanged content")
	}
	if content != "" {
		t.Fatalf("Resolve: content = %q, want empty when ok is false", content)
	}
}

func TestResolve_AbortsWhenEmpty(t *testing.T) {
	content, ok, err := Resolve("original", writeLaunch(""))
	if err != nil {
		t.Fatalf("Resolve returned unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("Resolve: ok = true, want false for empty saved content")
	}
	if content != "" {
		t.Fatalf("Resolve: content = %q, want empty when ok is false", content)
	}
}

func TestResolve_PropagatesLaunchError(t *testing.T) {
	wantErr := errors.New("editor exited with status 1")
	_, ok, err := Resolve("original", errLaunch(wantErr))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Resolve: err = %v, want %v", err, wantErr)
	}
	if ok {
		t.Fatalf("Resolve: ok = true, want false when launch fails")
	}
}

func TestResolve_RemovesTempFileOnSuccess(t *testing.T) {
	var capturedPath string
	launch := func(_, path string) error {
		capturedPath = path
		return os.WriteFile(path, []byte("updated\n"), 0o644)
	}

	if _, _, err := Resolve("original", launch); err != nil {
		t.Fatalf("Resolve returned unexpected error: %v", err)
	}
	if capturedPath == "" {
		t.Fatalf("launch was not invoked with a path")
	}
	if _, statErr := os.Stat(capturedPath); !os.IsNotExist(statErr) {
		t.Fatalf("temp file %q still exists after Resolve returned (stat err: %v)", capturedPath, statErr)
	}
}

func TestResolve_RemovesTempFileOnLaunchError(t *testing.T) {
	var capturedPath string
	launch := func(_, path string) error {
		capturedPath = path
		return errors.New("boom")
	}

	if _, _, err := Resolve("original", launch); err == nil {
		t.Fatalf("Resolve: err = nil, want non-nil")
	}
	if capturedPath == "" {
		t.Fatalf("launch was not invoked with a path")
	}
	if _, statErr := os.Stat(capturedPath); !os.IsNotExist(statErr) {
		t.Fatalf("temp file %q still exists after Resolve returned (stat err: %v)", capturedPath, statErr)
	}
}

func TestResolve_WritesCurrentContentToTempFileBeforeLaunch(t *testing.T) {
	var seenDuringLaunch string
	launch := func(_, path string) error {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		seenDuringLaunch = string(b)
		return os.WriteFile(path, []byte("updated\n"), 0o644)
	}

	if _, _, err := Resolve("original", launch); err != nil {
		t.Fatalf("Resolve returned unexpected error: %v", err)
	}
	if seenDuringLaunch != "original" {
		t.Fatalf("temp file content seen by launch = %q, want %q", seenDuringLaunch, "original")
	}
}

func TestName_PrefersEditorEnv(t *testing.T) {
	t.Setenv("EDITOR", "my-editor")
	t.Setenv("VISUAL", "my-visual")

	if got := Name(); got != "my-editor" {
		t.Fatalf("Name() = %q, want %q", got, "my-editor")
	}
}

func TestName_FallsBackToVisual(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "my-visual")

	if got := Name(); got != "my-visual" {
		t.Fatalf("Name() = %q, want %q", got, "my-visual")
	}
}

func TestName_FallsBackToVi(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")

	if got := Name(); got != "vi" {
		t.Fatalf("Name() = %q, want %q", got, "vi")
	}
}
