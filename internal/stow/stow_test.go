package stow_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lum1n/dotr/internal/stow"
)

func TestReadRC(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".stowrc")
	body := "# comment\n--dir=/tmp/farm\n--target=/tmp/home\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	opts, err := stow.ReadRC(path)
	if err != nil {
		t.Fatal(err)
	}
	if opts.Dir != "/tmp/farm" || opts.Target != "/tmp/home" {
		t.Fatalf("%+v", opts)
	}
}

func TestAnalyzeFoldedAndLinked(t *testing.T) {
	root := t.TempDir()
	stowDir := filepath.Join(root, "repos")
	pkg := filepath.Join(stowDir, "dotfiles")
	target := filepath.Join(root, "home")
	configReal := filepath.Join(target, ".config")

	mustMkdir(t, filepath.Join(pkg, ".config", "alacritty"))
	mustWrite(t, filepath.Join(pkg, ".config", "alacritty", "a.toml"), "x")
	mustWrite(t, filepath.Join(pkg, ".zshenv"), "export X=1\n")
	mustMkdir(t, configReal)

	// Folded: ~/.config is real dir; app is symlink into package.
	mustSymlink(t, filepath.Join(pkg, ".config", "alacritty"), filepath.Join(configReal, "alacritty"))
	mustSymlink(t, filepath.Join(pkg, ".zshenv"), filepath.Join(target, ".zshenv"))

	opts := stow.Options{Dir: stowDir, Target: target}
	p, err := stow.Analyze(opts, "dotfiles")
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != stow.StatusLinked {
		t.Fatalf("status=%s linked=%d missing=%d conflict=%d links=%+v",
			p.Status, p.Linked, p.Missing, p.Conflicts, p.Links)
	}
	if p.Linked < 2 {
		t.Fatalf("linked=%d", p.Linked)
	}
}

func TestAnalyzeMissingAndConflict(t *testing.T) {
	root := t.TempDir()
	stowDir := filepath.Join(root, "stow")
	pkg := filepath.Join(stowDir, "pkg")
	target := filepath.Join(root, "home")
	mustMkdir(t, filepath.Join(pkg, "bin"))
	mustWrite(t, filepath.Join(pkg, "bin", "tool"), "#!/bin/sh\n")
	mustWrite(t, filepath.Join(pkg, "conflict.txt"), "pkg\n")
	mustMkdir(t, target)
	mustWrite(t, filepath.Join(target, "conflict.txt"), "other\n")

	opts := stow.Options{Dir: stowDir, Target: target}
	p, err := stow.Analyze(opts, "pkg")
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != stow.StatusConflict {
		t.Fatalf("status=%s %+v", p.Status, p)
	}
	if p.Missing < 1 || p.Conflicts < 1 {
		t.Fatalf("missing=%d conflicts=%d", p.Missing, p.Conflicts)
	}
}

func TestOwner(t *testing.T) {
	root := t.TempDir()
	stowDir := filepath.Join(root, "repos")
	pkg := filepath.Join(stowDir, "dotfiles", ".config", "nvim")
	target := filepath.Join(root, "home", ".config", "nvim")
	mustMkdir(t, pkg)
	mustWrite(t, filepath.Join(pkg, "init.lua"), "--\n")
	mustMkdir(t, filepath.Dir(target))
	mustSymlink(t, pkg, target)

	opts := stow.Options{Dir: stowDir, Target: filepath.Join(root, "home")}
	name, ok := stow.Owner(opts, target)
	if !ok || name != "dotfiles" {
		t.Fatalf("owner=%q ok=%v", name, ok)
	}
}

func TestNormalizePackageAsDir(t *testing.T) {
	root := t.TempDir()
	repos := filepath.Join(root, "repos")
	dotfiles := filepath.Join(repos, "dotfiles")
	target := filepath.Join(root, "home")
	mustMkdir(t, filepath.Join(dotfiles, ".config", "app"))
	mustWrite(t, filepath.Join(dotfiles, ".config", "app", "c"), "1")
	mustMkdir(t, filepath.Join(target, ".config"))
	mustSymlink(t, filepath.Join(dotfiles, ".config", "app"), filepath.Join(target, ".config", "app"))
	// sibling decoy package under "dotfiles" that would be wrong with --dir=dotfiles
	mustMkdir(t, filepath.Join(dotfiles, ".config", "other"))
	mustWrite(t, filepath.Join(dotfiles, ".config", "other", "x"), "1")

	// Simulate Resolve path: user .stowrc points at package dir itself.
	home := filepath.Join(root, "userhome")
	mustMkdir(t, home)
	rc := "--dir=" + dotfiles + "\n--target=" + target + "\n"
	if err := os.WriteFile(filepath.Join(home, ".stowrc"), []byte(rc), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	opts, err := stow.Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if opts.Dir != repos {
		t.Fatalf("dir=%s want %s", opts.Dir, repos)
	}
	pkgs, err := stow.Packages(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || pkgs[0] != "dotfiles" {
		t.Fatalf("pkgs=%v", pkgs)
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
