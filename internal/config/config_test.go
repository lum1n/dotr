package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lum1n/dotr/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BackupKeep != 20 {
		t.Fatalf("backup_keep=%d", cfg.BackupKeep)
	}
	if cfg.ChromaStyle != "dracula" {
		t.Fatalf("style=%s", cfg.ChromaStyle)
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "dotr", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	body := "backup_keep: 5\nchroma_style: monokai\nconfirm_secrets: false\nmouse: false\nextra_ignores:\n  - foo/\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BackupKeep != 5 || cfg.ChromaStyle != "monokai" {
		t.Fatalf("%+v", cfg)
	}
	if cfg.ConfirmSecrets || cfg.Mouse {
		t.Fatalf("expected false flags: %+v", cfg)
	}
	if len(cfg.ExtraIgnores) != 1 || cfg.ExtraIgnores[0] != "foo/" {
		t.Fatalf("ignores=%v", cfg.ExtraIgnores)
	}
}

func TestEnsureFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := config.EnsureFile()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	// second call no-op
	path2, err := config.EnsureFile()
	if err != nil || path2 != path {
		t.Fatalf("%s %v", path2, err)
	}
}
