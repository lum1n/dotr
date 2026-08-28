package scan_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lum1n/dotr/internal/preview"
	"github.com/lum1n/dotr/internal/scan"
)

func TestScanFindsConfigs(t *testing.T) {
	home, cfg := setupFakeHome(t)
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", cfg)

	ents, err := scan.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) == 0 {
		t.Fatal("expected at least one config entry")
	}
}

func TestPreviewHomeDot(t *testing.T) {
	home, cfg := setupFakeHome(t)
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", cfg)

	ents, err := scan.Scan()
	if err != nil {
		t.Fatal(err)
	}
	var path string
	for _, e := range ents {
		if e.App == "home" {
			path = e.AbsPath
			break
		}
	}
	if path == "" {
		t.Fatal("expected a home dot")
	}
	r := preview.Render(path, 80)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if r.Content == "" {
		t.Fatal("empty preview")
	}
}

func setupFakeHome(t *testing.T) (home, cfg string) {
	t.Helper()
	root := t.TempDir()
	home = filepath.Join(root, "home")
	cfg = filepath.Join(home, ".config")
	if err := os.MkdirAll(filepath.Join(cfg, "nvim"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export Z=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "nvim", "init.lua"), []byte("-- nvim\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return home, cfg
}
