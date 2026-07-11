package logfile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rn404/nippo-cli/internal/log"
	"github.com/rn404/nippo-cli/internal/model"
)

func TestStatMissingFile(t *testing.T) {
	file, err := Stat(t.TempDir(), "2026-07-05")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if file != nil {
		t.Errorf("Stat for missing file should return nil, got %+v", file)
	}
}

func TestStatBrokenFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "2026-07-05.json"), []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Stat(dir, "2026-07-05"); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("broken JSON should surface a parse error, got %v", err)
	}
}

func TestUpdateInvalidDate(t *testing.T) {
	if err := Update(t.TempDir(), "not-a-date", model.NewLog()); err == nil {
		t.Error("Update with an invalid date should fail")
	}
}

func TestGetCreatesNewFile(t *testing.T) {
	dir := t.TempDir()
	file, err := Get(dir, "2026-07-05")
	if err != nil {
		t.Fatal(err)
	}

	if file.Name != "2026-07-05" {
		t.Errorf("Name = %q, want 2026-07-05", file.Name)
	}
	if len(file.Body.Items) != 0 {
		t.Errorf("new log should be empty: %+v", file.Body)
	}
	if _, err := os.Stat(filepath.Join(dir, "2026-07-05.json")); err != nil {
		t.Errorf("file should be created on disk: %v", err)
	}
}

func TestUpdateAndReload(t *testing.T) {
	dir := t.TempDir()
	file, err := Get(dir, "2026-07-05")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := log.Add(&file.Body, "buy cabbage", true); err != nil {
		t.Fatal(err)
	}
	if err := Update(dir, file.Name, file.Body); err != nil {
		t.Fatal(err)
	}

	reloaded, err := Stat(dir, "2026-07-05")
	if err != nil {
		t.Fatal(err)
	}
	if reloaded == nil || len(reloaded.Body.Items) != 1 {
		t.Fatalf("reloaded = %+v, want 1 item", reloaded)
	}
	if reloaded.Body.Items[0].Content != "buy cabbage" {
		t.Errorf("content = %q", reloaded.Body.Items[0].Content)
	}
}

func TestFilePermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "logs")
	file, err := Get(dir, "2026-07-05")
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(file.Path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("log file mode = %o, want 600 (owner-only)", got)
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Errorf("log dir mode = %o, want 700 (owner-only)", got)
	}
}

func TestUpdateFreezedLog(t *testing.T) {
	body := model.NewLog()
	body.Freezed = true
	if err := Update(t.TempDir(), "2026-07-05", body); !errors.Is(err, ErrFreezed) {
		t.Errorf("err = %v, want ErrFreezed", err)
	}
}

func TestInvalidDate(t *testing.T) {
	if _, err := Get(t.TempDir(), "not-a-date"); err == nil {
		t.Errorf("Get with invalid date should fail")
	}
}

func TestListSortedAndFiltered(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"2026-07-05", "2026-05-01", "2026-06-15"} {
		if _, err := Get(dir, name); err != nil {
			t.Fatal(err)
		}
	}
	// Files that must be ignored.
	for _, name := range []string{"not-a-date.json", "2026-07-05.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	refs, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"2026-05-01", "2026-06-15", "2026-07-05"}
	if len(refs) != len(want) {
		t.Fatalf("refs = %+v, want %v", refs, want)
	}
	for i, name := range want {
		if refs[i].Name != name {
			t.Errorf("refs[%d].Name = %q, want %q (ascending by date)", i, refs[i].Name, name)
		}
	}
}

func TestListMissingDir(t *testing.T) {
	refs, err := List(filepath.Join(t.TempDir(), "no-such-dir"))
	if err != nil {
		t.Fatal(err)
	}
	if refs != nil {
		t.Errorf("refs = %+v, want nil", refs)
	}
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	if _, err := Get(dir, "2026-07-05"); err != nil {
		t.Fatal(err)
	}

	refs, err := List(dir)
	if err != nil || len(refs) != 1 {
		t.Fatalf("refs = %+v, err = %v", refs, err)
	}

	if err := Remove(refs[0]); err != nil {
		t.Fatalf("Remove should delete the actual file: %v", err)
	}
	if _, err := os.Stat(refs[0].Path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file should be gone, got %v", err)
	}
}

func TestReadFormatSample(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "log-format")
	file, err := Stat(dir, "2026-07-05")
	if err != nil {
		t.Fatal(err)
	}
	if file == nil {
		t.Fatal("format sample should be readable")
	}
	if len(file.Body.Items) != 3 {
		t.Errorf("items = %d, want 3", len(file.Body.Items))
	}
}
