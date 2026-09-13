// Package editor resolves new item content by round-tripping it
// through an external editor process. It deals only with plain
// strings and an injected launch function, and does not depend on
// any other internal package.
package editor

import (
	"os"
	"strings"
)

// EditorName returns the editor to invoke: $EDITOR, else $VISUAL,
// else "vi".
func EditorName() string {
	if v := os.Getenv("EDITOR"); v != "" {
		return v
	}
	if v := os.Getenv("VISUAL"); v != "" {
		return v
	}
	return "vi"
}

// Resolve lets the user edit current via an external editor. launch
// actually invokes the editor (injected for testability -- in
// production it execs EditorName() against the real terminal).
// ok is false when the saved content is identical to current or is
// empty, mirroring git commit's empty-message abort; err is non-nil
// only when launch itself failed (e.g. non-zero exit).
func Resolve(current string, launch func(editorName, path string) error) (content string, ok bool, err error) {
	f, err := os.CreateTemp("", "sava-edit-*")
	if err != nil {
		return "", false, err
	}
	path := f.Name()
	defer os.Remove(path)

	if _, err := f.WriteString(current); err != nil {
		f.Close()
		return "", false, err
	}
	if err := f.Close(); err != nil {
		return "", false, err
	}

	if err := launch(EditorName(), path); err != nil {
		return "", false, err
	}

	saved, err := os.ReadFile(path)
	if err != nil {
		return "", false, err
	}

	candidate := strings.TrimSuffix(string(saved), "\n")
	if candidate == current || candidate == "" {
		return "", false, nil
	}
	return candidate, true, nil
}
