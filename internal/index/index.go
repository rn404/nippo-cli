// Package index maintains index.json in the log directory: a cache
// mapping tags and item hashes to the daily log files containing them.
// The file is rebuildable from the logs at any time, so it may go
// stale after item deletion; readers should rebuild on a miss instead
// of trusting it blindly.
package index

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/rn404/nippo-cli/internal/logfile"
)

const fileName = "index.json"

// Entry points to one item inside a daily log file.
type Entry struct {
	Date string `json:"date"`
	Hash string `json:"hash"`
}

// Index is the persisted index body.
type Index struct {
	// Tags maps a tag to the items carrying it.
	Tags map[string][]Entry `json:"tags"`
	// Hashes maps an item hash to the date (file name) holding it.
	Hashes map[string]string `json:"hashes"`
}

// Path returns the index file location inside the log directory.
func Path(dir string) string {
	return filepath.Join(dir, fileName)
}

// Build scans every daily log file and returns a fresh index.
func Build(dir string) (Index, error) {
	idx := Index{
		Tags:   map[string][]Entry{},
		Hashes: map[string]string{},
	}

	refs, err := logfile.List(dir)
	if err != nil {
		return Index{}, err
	}

	for _, ref := range refs {
		file, err := logfile.Stat(dir, ref.Name)
		if err != nil {
			return Index{}, err
		}
		for _, item := range file.Body.Items {
			idx.Hashes[item.Hash] = file.Name
			for _, tag := range item.Tags {
				idx.Tags[tag] = append(idx.Tags[tag], Entry{Date: file.Name, Hash: item.Hash})
			}
		}
	}

	return idx, nil
}

// Load reads the persisted index. A missing or broken file yields an
// empty index rather than an error: the file is a cache, and callers
// are expected to Rebuild on a miss anyway.
func Load(dir string) (Index, error) {
	empty := Index{Tags: map[string][]Entry{}, Hashes: map[string]string{}}

	data, err := os.ReadFile(Path(dir))
	if errors.Is(err, os.ErrNotExist) {
		return empty, nil
	}
	if err != nil {
		return Index{}, err
	}

	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return empty, nil
	}
	if idx.Tags == nil {
		idx.Tags = map[string][]Entry{}
	}
	if idx.Hashes == nil {
		idx.Hashes = map[string]string{}
	}
	return idx, nil
}

// Rebuild builds a fresh index and persists it.
func Rebuild(dir string) (Index, error) {
	idx, err := Build(dir)
	if err != nil {
		return Index{}, err
	}
	if err := save(dir, idx); err != nil {
		return Index{}, err
	}
	return idx, nil
}

// Remove deletes the index file; a missing file is not an error.
func Remove(dir string) error {
	err := os.Remove(Path(dir))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func save(dir string, idx Index) error {
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}

	// Same ownership rules as the daily logs themselves.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(Path(dir), data, 0o600)
}
