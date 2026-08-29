package tui

import (
	"sort"
	"testing"
)

func TestFuzzyScoreSubstring(t *testing.T) {
	hay := searchHaystack("alacritty", "alacritty.toml", "/home/u/.config/alacritty/alacritty.toml")
	if fuzzyScore("alac", hay) < 0 {
		t.Fatal("expected match")
	}
	if fuzzyScore("zzz", hay) >= 0 {
		t.Fatal("expected miss")
	}
}

func TestFuzzyScoreRanksBetter(t *testing.T) {
	a := searchHaystack("alacritty", "alacritty.toml", "/x/alacritty/alacritty.toml")
	b := searchHaystack("other", "notes.md", "/x/other/notes-about-alacritty.md")
	sa := fuzzyScore("alacritty", a)
	sb := fuzzyScore("alacritty", b)
	if sa <= sb {
		t.Fatalf("want alacritty app ranked higher: %d vs %d", sa, sb)
	}
}

func TestFuzzyMultiTerm(t *testing.T) {
	hay := searchHaystack("nvim", "lua/config.lua", "/x/nvim/lua/config.lua")
	if fuzzyScore("nvim lua", hay) < 0 {
		t.Fatal("expected multi-term match")
	}
	if fuzzyScore("nvim missing", hay) >= 0 {
		t.Fatal("expected multi-term miss")
	}
}

func TestFuzzySortStable(t *testing.T) {
	type row struct {
		name  string
		score int
	}
	rows := []row{
		{"b", fuzzyScore("nv", searchHaystack("env", "x", "/env"))},
		{"a", fuzzyScore("nv", searchHaystack("nvim", "init.lua", "/nvim/init.lua"))},
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].score > rows[j].score })
	if rows[0].name != "a" {
		t.Fatalf("got %+v", rows)
	}
}
