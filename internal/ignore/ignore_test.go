package ignore_test

import (
	"testing"

	"github.com/lum1n/dotr/internal/ignore"
)

func TestMatchPrefix(t *testing.T) {
	l := &ignore.List{Patterns: []string{"opencode/skills/"}}
	if !l.Match("opencode/skills/foo/SKILL.md", "") {
		t.Fatal("expected skills file ignored")
	}
	if l.Match("opencode/config.json", "") {
		t.Fatal("config should not be ignored")
	}
	if !l.MatchDir("opencode/skills") {
		t.Fatal("skills dir should be skipped")
	}
}

func TestMatchApp(t *testing.T) {
	l := &ignore.List{Patterns: []string{"cursor/"}}
	if !l.Match("cursor/argv.json", "") {
		t.Fatal("expected app file ignored")
	}
	if !l.MatchDir("cursor") {
		t.Fatal("expected app dir skipped")
	}
}

func TestMatchGlobStar(t *testing.T) {
	l := &ignore.List{Patterns: []string{"**/skills/"}}
	if !l.Match("opencode/skills/x.md", "") {
		t.Fatal("expected **/skills/ match")
	}
	if !l.Match("claude/skills/y.md", "") {
		t.Fatal("expected other app skills match")
	}
}

func TestSuggest(t *testing.T) {
	got := ignore.SuggestForEntry("opencode", "skills/foo/SKILL.md")
	if got != "opencode/skills/" {
		t.Fatalf("got %q", got)
	}
	got = ignore.SuggestForEntry("alacritty", "alacritty.toml")
	if got != "alacritty/alacritty.toml" {
		t.Fatalf("got %q", got)
	}
	got = ignore.SuggestApp("opencode", "skills/x")
	if got != "opencode/" {
		t.Fatalf("got %q", got)
	}
}

func TestAddRemove(t *testing.T) {
	l := &ignore.List{}
	if !l.Add("opencode/skills/") {
		t.Fatal("add")
	}
	if l.Add("opencode/skills/") {
		t.Fatal("duplicate should be false")
	}
	if !l.Remove("opencode/skills/") {
		t.Fatal("remove")
	}
	if len(l.Patterns) != 0 {
		t.Fatal("expected empty")
	}
}
