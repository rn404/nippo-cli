// Package logfile handles reading and writing daily log files stored
// under ~/.log/sava/<yyyy-MM-dd>.json.
package logfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rn404/nippo-cli/internal/model"
)

const (
	logDirName = ".log/sava"
	fileExt    = ".json"
)

var (
	// ErrFreezed is returned when attempting to update a frozen log file.
	ErrFreezed = errors.New("this log file is freezed, no updates")
	// ErrNotFound is returned by Stat when the day has no log file.
	ErrNotFound = errors.New("log file not found")
)

// LogFile is a loaded daily log file.
type LogFile struct {
	Path string // full path including file name
	Name string // yyyy-MM-dd (file name without extension)
	Body model.Log
}

// Ref points to a log file on disk without loading its body.
type Ref struct {
	Path string
	Name string // yyyy-MM-dd
}

// Dir returns the log directory. It prefers the home directory and
// falls back to the current working directory, like the Deno version.
func Dir() string {
	root, err := os.UserHomeDir()
	if err != nil || root == "" {
		root, _ = os.Getwd()
	}
	return filepath.Join(root, logDirName)
}

func pathFor(dir, name string) string {
	return filepath.Join(dir, name+fileExt)
}

// resolveName returns the file name (yyyy-MM-dd) for day, defaulting
// to today when day is empty.
func resolveName(day string) (string, error) {
	if day == "" {
		return model.Today(), nil
	}
	if _, err := model.ParseDate(day); err != nil {
		return "", err
	}
	return day, nil
}

// Stat loads the log file for day (today if empty). It returns an
// error wrapping ErrNotFound when the file does not exist.
func Stat(dir, day string) (*LogFile, error) {
	name, err := resolveName(day)
	if err != nil {
		return nil, err
	}

	path := pathFor(dir, name)
	data, err := os.ReadFile(path) //nolint:gosec // path is the log dir joined with a strictly validated date
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%s: %w", name, ErrNotFound)
	}
	if err != nil {
		return nil, err
	}

	var body model.Log
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, fmt.Errorf("broken log file %s: %w", path, err)
	}

	return &LogFile{Path: path, Name: name, Body: body}, nil
}

// Get loads the log file for day (today if empty), creating an empty
// one when it does not exist yet.
func Get(dir, day string) (*LogFile, error) {
	file, err := Stat(dir, day)
	if err == nil {
		return file, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	name, err := resolveName(day)
	if err != nil {
		return nil, err
	}

	body := model.NewLog()
	if err := Update(dir, name, body); err != nil {
		return nil, err
	}

	return &LogFile{Path: pathFor(dir, name), Name: name, Body: body}, nil
}

// Update writes body to the log file for day. Frozen logs are rejected.
func Update(dir, day string, body model.Log) error {
	if body.Freezed {
		return ErrFreezed
	}

	name, err := resolveName(day)
	if err != nil {
		return err
	}

	// Logs are personal notes: keep them readable by the owner only.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(pathFor(dir, name), data, 0o600)
}

// List returns refs to all daily log files in dir, sorted by date in
// ascending order. Files whose name is not a strict date are skipped.
func List(dir string) ([]Ref, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var refs []Ref
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), fileExt) {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), fileExt)
		if !model.IsDateString(name) {
			continue
		}
		refs = append(refs, Ref{Path: filepath.Join(dir, entry.Name()), Name: name})
	}

	sort.Slice(refs, func(i, j int) bool { return refs[i].Name < refs[j].Name })
	return refs, nil
}

// Remove deletes the log file that ref points to.
func Remove(ref Ref) error {
	return os.Remove(ref.Path)
}
