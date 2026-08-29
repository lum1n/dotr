package preview_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lum1n/dotr/internal/preview"
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

func TestPreviewCacheHitAndInvalidate(t *testing.T) {
	preview.Clear()
	t.Cleanup(preview.Clear)

	dir := t.TempDir()
	p := filepath.Join(dir, "a.toml")
	if err := os.WriteFile(p, []byte("a = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r1 := preview.Render(p, 80)
	if r1.Err != nil {
		t.Fatal(r1.Err)
	}
	hit, ok := preview.Lookup(p, 80)
	if !ok {
		t.Fatal("expected cache hit after render")
	}
	if hit.Content != r1.Content {
		t.Fatal("lookup content mismatch")
	}

	preview.Invalidate(p)
	if _, ok := preview.Lookup(p, 80); ok {
		t.Fatal("expected miss after invalidate")
	}
	if err := os.WriteFile(p, []byte("value = 99\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r2 := preview.Render(p, 80)
	if r2.Err != nil {
		t.Fatal(r2.Err)
	}
	if !strings.Contains(r2.Content, "99") {
		t.Fatalf("expected rewritten content, got %q", r2.Content)
	}
}

func TestMarkdownPreview(t *testing.T) {
	dir := t.TempDir()
	md := filepath.Join(dir, "note.md")
	if err := os.WriteFile(md, []byte("# hi\n\nbody\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := preview.Render(md, 80)
	if r.Err != nil {
		t.Fatalf("markdown preview: %v", r.Err)
	}
	if r.Language != "markdown" {
		t.Fatalf("markdown language: %q", r.Language)
	}
	if !strings.Contains(r.Content, "hi") {
		t.Fatalf("markdown content missing heading: %q", r.Content)
	}
}
