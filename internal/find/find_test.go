package find_test

import (
	"testing"

	"github.com/vegarringdal/dotr/internal/find"
	"github.com/vegarringdal/dotr/internal/scan"
)

func TestBestPrefersExact(t *testing.T) {
	ents := []scan.Entry{
		{App: "alacritty", RelPath: "alacritty.toml", AbsPath: "/c/alacritty/alacritty.toml"},
		{App: "tmux", RelPath: "plugins/x/osx_alacritty_start_tmux.sh", AbsPath: "/c/tmux/x.sh"},
	}
	best, ok := find.Best(ents, "alacritty.toml")
	if !ok || best.App != "alacritty" {
		t.Fatalf("got %+v", best)
	}
}
