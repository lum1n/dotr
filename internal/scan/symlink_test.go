package scan_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lum1n/dotr/internal/scan"
)

func TestScanIncludesStowSymlinkApps(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	cfg := filepath.Join(home, ".config")
	farm := filepath.Join(root, "dotfiles", ".config")

	mustMkdir(t, home)
	mustMkdir(t, cfg)
	mustWrite(t, filepath.Join(home, ".zshrc"), "export Z=1\n")
	mustWrite(t, filepath.Join(home, ".stowrc"), "--dir=/tmp\n")
	mustWrite(t, filepath.Join(farm, "alacritty", "alacritty.toml"), "[general]\n")
	mustWrite(t, filepath.Join(farm, "ghostty", "config"), "theme = dark\n")
	mustSymlink(t, filepath.Join(farm, "alacritty"), filepath.Join(cfg, "alacritty"))
	mustSymlink(t, filepath.Join(farm, "ghostty"), filepath.Join(cfg, "ghostty"))

	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", cfg)

	ents, err := scan.Scan()
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	homeN := 0
	for _, e := range ents {
		joined += e.App + "/" + e.RelPath + "\n"
		if e.App == "home" {
			homeN++
		}
	}
	for _, want := range []string{"alacritty", "ghostty", "home/.zshrc", "home/.stowrc"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q\n%s", want, joined)
		}
	}
	if homeN < 2 {
		t.Errorf("expected home dots, got %d", homeN)
	}

	var sawSymlinkApp bool
	for _, e := range ents {
		if e.App == "alacritty" && e.Symlink {
			sawSymlinkApp = true
			break
		}
	}
	if !sawSymlinkApp {
		t.Fatal("expected symlink mark on alacritty entry")
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	mustMkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustSymlink(t *testing.T, oldname, newname string) {
	t.Helper()
	mustMkdir(t, filepath.Dir(newname))
	if err := os.Symlink(oldname, newname); err != nil {
		t.Fatal(err)
	}
}
