package tui

import (
	"strings"
	"testing"

	"github.com/lum1n/dotr/internal/gitstatus"
	"github.com/lum1n/dotr/internal/scan"
)

func TestBrowseHelpHintLabels(t *testing.T) {
	m := Model{mode: modeBrowse}
	got := m.helpHint()
	if !strings.Contains(got, "? help") {
		t.Fatalf("missing ? help: %q", got)
	}
	if !strings.Contains(got, "q quit") {
		t.Fatalf("missing q quit: %q", got)
	}
}

func TestListMarksAreColored(t *testing.T) {
	s := newStyles()
	e := scan.Entry{App: "nvim", RelPath: "init.lua", AbsPath: "/tmp/init.lua", Symlink: true}
	line := formatListLine(s, e, map[string]gitstatus.Kind{
		e.AbsPath: gitstatus.Modified,
	}, nil, 40, false)
	if !strings.Contains(line, "init.lua") {
		t.Fatalf("missing name: %q", line)
	}
	plain := stripANSI(line)
	if !strings.Contains(plain, "M") {
		t.Fatalf("missing git mark: %q", plain)
	}
	if !strings.Contains(plain, "↗") {
		t.Fatalf("missing symlink mark: %q", plain)
	}
	if !strings.Contains(line, "\x1b") {
		t.Fatal("expected ANSI color on list marks")
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		if r == '\x1b' {
			in = true
			continue
		}
		if in {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				in = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
