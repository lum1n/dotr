package backup_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vegarringdal/dotr/internal/backup"
)

func TestCreateListRestore(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "config.yaml")
	if err := os.WriteFile(src, []byte("a: 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	snap, err := backup.Create(src, 0)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Path == "" {
		t.Fatal("empty snapshot path")
	}

	list, err := backup.List(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("want 1 snapshot, got %d", len(list))
	}

	if err := os.WriteFile(src, []byte("a: 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := backup.Restore(list[0]); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "a: 1\n" {
		t.Fatalf("restore failed: %q", data)
	}
}
