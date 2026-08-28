package scan_test

import (
	"testing"

	"github.com/lum1n/dotr/internal/preview"
	"github.com/lum1n/dotr/internal/scan"
)

func TestScanFindsConfigs(t *testing.T) {
	ents, err := scan.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) == 0 {
		t.Fatal("expected at least one config entry")
	}
	t.Logf("found %d entries", len(ents))
	for i, e := range ents {
		if i >= 8 {
			break
		}
		t.Logf("%s/%s symlink=%v size=%d", e.App, e.RelPath, e.Symlink, e.Size)
	}
}

func TestPreviewHomeDot(t *testing.T) {
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
		t.Skip("no home dots")
	}
	r := preview.Render(path, 80)
	if r.Err != nil {
		t.Fatal(r.Err)
	}
	if r.Content == "" {
		t.Fatal("empty preview")
	}
	t.Logf("preview %s lang=%s binary=%v len=%d", path, r.Language, r.Binary, len(r.Content))
}
