package symlink_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lum1n/dotr/internal/symlink"
)

func TestCreateReadRetargetRemove(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.txt")
	if err := os.WriteFile(target, []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.txt")
	if err := symlink.Create(link, target); err != nil {
		t.Fatal(err)
	}
	info, err := symlink.Read(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Target != target || info.Dangle {
		t.Fatalf("%+v", info)
	}

	other := filepath.Join(dir, "other.txt")
	if err := os.WriteFile(other, []byte("yo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := symlink.Retarget(link, other); err != nil {
		t.Fatal(err)
	}
	info, err = symlink.Read(link)
	if err != nil || info.Target != other {
		t.Fatalf("%+v %v", info, err)
	}

	if err := symlink.Remove(link); err != nil {
		t.Fatal(err)
	}
	if symlink.Is(link) {
		t.Fatal("still a symlink")
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatal("removed target by mistake")
	}
}

func TestCreateRelative(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.toml"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "b.toml")
	if err := symlink.Create(link, "a.toml"); err != nil {
		t.Fatal(err)
	}
	info, err := symlink.Read(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Target != "a.toml" || info.Dangle {
		t.Fatalf("%+v", info)
	}
}

func TestRefuseRemoveRegular(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := symlink.Remove(f); err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateExists(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := symlink.Create(f, "elsewhere"); err == nil {
		t.Fatal("expected error")
	}
}
