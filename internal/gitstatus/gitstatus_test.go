package gitstatus_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/lum1n/dotr/internal/gitstatus"
)

func TestMapCleanAndModified(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "t@example.com")
	run(t, dir, "git", "config", "user.name", "t")

	f := filepath.Join(dir, "a.toml")
	if err := os.WriteFile(f, []byte("a=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", "a.toml")
	run(t, dir, "git", "commit", "-m", "init")

	m := gitstatus.Map([]string{f})
	if m[f] != gitstatus.Clean {
		t.Fatalf("want clean, got %v", m[f])
	}

	if err := os.WriteFile(f, []byte("a=2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m = gitstatus.Map([]string{f})
	if m[f] != gitstatus.Modified {
		t.Fatalf("want modified, got %v (%s)", m[f], m[f].Mark())
	}
}

func TestMapManyFilesSameRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "t@example.com")
	run(t, dir, "git", "config", "user.name", "t")

	var paths []string
	for i := 0; i < 40; i++ {
		p := filepath.Join(dir, "f"+strconv.Itoa(i)+".toml")
		if err := os.WriteFile(p, []byte("a=1\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "init")

	dirty := paths[0]
	if err := os.WriteFile(dirty, []byte("a=2\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	m := gitstatus.Map(paths)
	if m[dirty] != gitstatus.Modified {
		t.Fatalf("dirty: want modified, got %v", m[dirty])
	}
	for _, p := range paths[1:] {
		if m[p] != gitstatus.Clean {
			t.Fatalf("%s: want clean, got %v", p, m[p])
		}
	}
}

func TestMapOutsideRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "alone.toml")
	if err := os.WriteFile(p, []byte("a=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := gitstatus.Map([]string{p})
	if m[p] != gitstatus.None {
		t.Fatalf("want none, got %v", m[p])
	}
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}
