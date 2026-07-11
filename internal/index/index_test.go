package index

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/rn404/nippo-cli/internal/log"
	"github.com/rn404/nippo-cli/internal/logfile"
)

func addTaggedItem(t *testing.T, dir, day, content string, tags []string) string {
	t.Helper()

	file, err := logfile.Get(dir, day)
	if err != nil {
		t.Fatal(err)
	}
	item, err := log.Add(&file.Body, content, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) > 0 {
		if _, err := log.AddTags(&file.Body, item.Hash, tags); err != nil {
			t.Fatal(err)
		}
	}
	if err := logfile.Update(dir, day, file.Body); err != nil {
		t.Fatal(err)
	}
	return item.Hash
}

func TestBuildAndRebuild(t *testing.T) {
	dir := t.TempDir()
	first := addTaggedItem(t, dir, "2026-07-10", "buy cabbage", []string{"cabbage"})
	second := addTaggedItem(t, dir, "2026-07-11", "feed the shrimp", []string{"shrimp", "pet"})
	plain := addTaggedItem(t, dir, "2026-07-11", "no tags here", nil)

	idx, err := Rebuild(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(idx.Tags) != 3 {
		t.Errorf("tags = %+v, want cabbage, shrimp, pet", idx.Tags)
	}
	if entries := idx.Tags["cabbage"]; len(entries) != 1 || entries[0].Hash != first || entries[0].Date != "2026-07-10" {
		t.Errorf("cabbage entries = %+v", entries)
	}
	for hash, date := range map[string]string{first: "2026-07-10", second: "2026-07-11", plain: "2026-07-11"} {
		if idx.Hashes[hash] != date {
			t.Errorf("Hashes[%s] = %q, want %q", hash, idx.Hashes[hash], date)
		}
	}

	// Rebuild persists the index as readable JSON next to the logs.
	data, err := os.ReadFile(Path(dir))
	if err != nil {
		t.Fatal(err)
	}
	var reloaded Index
	if err := json.Unmarshal(data, &reloaded); err != nil {
		t.Fatalf("index file should be valid JSON: %v", err)
	}
	if len(reloaded.Hashes) != 3 {
		t.Errorf("persisted hashes = %+v, want 3", reloaded.Hashes)
	}
}

func TestBuildEmptyDir(t *testing.T) {
	idx, err := Build(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Tags) != 0 || len(idx.Hashes) != 0 {
		t.Errorf("empty dir should yield an empty index: %+v", idx)
	}
}

func TestIndexFileIsNotListedAsLog(t *testing.T) {
	dir := t.TempDir()
	addTaggedItem(t, dir, "2026-07-11", "content", []string{"go"})
	if _, err := Rebuild(dir); err != nil {
		t.Fatal(err)
	}

	refs, err := logfile.List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || refs[0].Name != "2026-07-11" {
		t.Errorf("index.json must not be listed as a daily log: %+v", refs)
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()

	// Missing file yields an empty, usable index.
	idx, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if idx.Tags == nil || idx.Hashes == nil {
		t.Errorf("Load should return non-nil maps: %+v", idx)
	}

	// A broken file is treated as an empty cache, not an error.
	if err := os.WriteFile(Path(dir), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if idx, err = Load(dir); err != nil || idx.Tags == nil || idx.Hashes == nil {
		t.Errorf("broken index should load as empty: %+v, %v", idx, err)
	}

	// A persisted index round-trips.
	hash := addTaggedItem(t, dir, "2026-07-11", "content", []string{"go"})
	if _, err := Rebuild(dir); err != nil {
		t.Fatal(err)
	}
	idx, err = Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if idx.Hashes[hash] != "2026-07-11" || len(idx.Tags["go"]) != 1 {
		t.Errorf("loaded index mismatch: %+v", idx)
	}
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	if err := Remove(dir); err != nil {
		t.Errorf("removing a missing index should not fail: %v", err)
	}

	if _, err := Rebuild(dir); err != nil {
		t.Fatal(err)
	}
	if err := Remove(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(Path(dir)); !os.IsNotExist(err) {
		t.Errorf("index file should be gone, got %v", err)
	}
}
