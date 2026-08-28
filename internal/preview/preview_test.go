package preview_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vegarringdal/dotr/internal/preview"
)

func TestParseJSONYAMLTOML(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		body string
		want preview.ParseStatus
	}{
		{"ok.json", `{"a":1}`, preview.ParseOK},
		{"bad.json", `{a:`, preview.ParseFail},
		{"ok.yaml", "a: 1\n", preview.ParseOK},
		{"bad.yaml", ":\n  -", preview.ParseFail},
		{"ok.toml", "a = 1\n", preview.ParseOK},
		{"bad.toml", "a = [\n", preview.ParseFail},
		{"plain.txt", "hello", preview.ParseNone},
	}
	for _, tc := range cases {
		path := filepath.Join(dir, tc.name)
		if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
			t.Fatal(err)
		}
		r := preview.Render(path, 80)
		if r.Parse != tc.want {
			t.Errorf("%s: parse=%v want=%v err=%q", tc.name, r.Parse, tc.want, r.ParseErr)
		}
	}
}
