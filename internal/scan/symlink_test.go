package scan_test

import (
	"strings"
	"testing"

	"github.com/lum1n/dotr/internal/scan"
)

func TestScanIncludesStowSymlinkApps(t *testing.T) {
	ents, err := scan.Scan()
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	home := 0
	for _, e := range ents {
		joined += e.App + "/" + e.RelPath + "\n"
		if e.App == "home" {
			home++
		}
	}
	for _, want := range []string{"alacritty", "ghostty", "home/.zshrc", "home/.stowrc"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q (total %d)", want, len(ents))
		}
	}
	if home < 3 {
		t.Errorf("expected home dots, got %d", home)
	}
}
