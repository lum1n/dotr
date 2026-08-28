// Package ignore loads and matches user ignore patterns for dotr.
//
// Patterns live in $XDG_CONFIG_HOME/dotr/ignore (one per line).
// Matching is against config-relative paths like "opencode/skills/foo.md"
// and home paths like "home/.zshrc".
//
// Rules:
//   - blank lines and # comments are ignored
//   - trailing "/" means directory prefix (matches that path and children)
//   - "*" / "?" use filepath.Match on the full relative path
//   - otherwise exact match, or prefix if pattern has no slash and equals app name
package ignore

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const fileName = "ignore"

// List holds user ignore patterns.
type List struct {
	Patterns []string
	path     string
}

// Path returns the ignore file path.
func Path() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}

func configDir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "dotr"), nil
}

// Load reads the ignore file. Missing file yields an empty list.
func Load() (*List, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	l := &List{path: path}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return l, nil
		}
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		l.Patterns = append(l.Patterns, line)
	}
	return l, sc.Err()
}

// Save writes patterns back (creates parent dirs). Preserves a short header.
func (l *List) Save() error {
	if l.path == "" {
		path, err := Path()
		if err != nil {
			return err
		}
		l.path = path
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o700); err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("# dotr ignore patterns (one per line)\n")
	b.WriteString("# Examples:\n")
	b.WriteString("#   opencode/skills/\n")
	b.WriteString("#   **/skills/\n")
	b.WriteString("#   cursor/\n")
	b.WriteString("#   home/.nvidia-settings-rc\n")
	b.WriteString("#\n")
	for _, p := range l.Patterns {
		b.WriteString(p)
		b.WriteByte('\n')
	}
	return os.WriteFile(l.path, []byte(b.String()), 0o600)
}

// Add appends pattern if not already present. Returns true if added.
func (l *List) Add(pattern string) bool {
	pattern = normalize(pattern)
	if pattern == "" {
		return false
	}
	for _, p := range l.Patterns {
		if p == pattern {
			return false
		}
	}
	l.Patterns = append(l.Patterns, pattern)
	return true
}

// Remove deletes an exact pattern. Returns true if removed.
func (l *List) Remove(pattern string) bool {
	pattern = normalize(pattern)
	out := l.Patterns[:0]
	found := false
	for _, p := range l.Patterns {
		if p == pattern {
			found = true
			continue
		}
		out = append(out, p)
	}
	l.Patterns = out
	return found
}

// RemoveAt removes by index.
func (l *List) RemoveAt(i int) bool {
	if i < 0 || i >= len(l.Patterns) {
		return false
	}
	l.Patterns = append(l.Patterns[:i], l.Patterns[i+1:]...)
	return true
}

func normalize(p string) string {
	p = strings.TrimSpace(p)
	p = strings.ReplaceAll(p, "\\", "/")
	return p
}

// RelPath builds the match key for an entry.
func RelPath(app, rel string) string {
	switch app {
	case "home":
		return "home/" + rel
	case "config":
		return rel
	default:
		if rel == "" || rel == "." {
			return app
		}
		return app + "/" + rel
	}
}

// Match reports whether rel (config-style path) or abs is ignored.
func (l *List) Match(rel, abs string) bool {
	if l == nil || len(l.Patterns) == 0 {
		return false
	}
	rel = strings.ReplaceAll(rel, "\\", "/")
	abs = strings.ReplaceAll(abs, "\\", "/")
	for _, p := range l.Patterns {
		if matchPattern(p, rel, abs) {
			return true
		}
	}
	return false
}

// MatchDir reports whether a directory (config-relative) should be skipped entirely.
func (l *List) MatchDir(rel string) bool {
	if l == nil || len(l.Patterns) == 0 {
		return false
	}
	rel = strings.ReplaceAll(strings.Trim(rel, "/"), "\\", "/")
	if rel == "" {
		return false
	}
	// Treat as directory path for matching.
	dirRel := rel + "/"
	for _, p := range l.Patterns {
		if matchPattern(p, dirRel, "") || matchPattern(p, rel, "") {
			return true
		}
		// pattern "foo/" should skip walking into foo
		np := normalize(p)
		if strings.HasSuffix(np, "/") {
			prefix := strings.TrimSuffix(np, "/")
			if rel == prefix || strings.HasPrefix(rel, prefix+"/") {
				return true
			}
		}
		if np == rel {
			return true
		}
	}
	return false
}

func matchPattern(pattern, rel, abs string) bool {
	pattern = normalize(pattern)
	if pattern == "" {
		return false
	}

	// **/foo/ → match if /foo/ or prefix foo/ appears in path
	if strings.HasPrefix(pattern, "**/") {
		rest := strings.TrimPrefix(pattern, "**/")
		return matchPattern(rest, rel, abs) ||
			strings.Contains("/"+strings.TrimSuffix(rel, "/")+"/", "/"+strings.TrimSuffix(rest, "/")+"/")
	}

	if strings.HasSuffix(pattern, "/") {
		prefix := strings.TrimSuffix(pattern, "/")
		r := strings.TrimSuffix(rel, "/")
		if r == prefix || strings.HasPrefix(r, prefix+"/") {
			return true
		}
		if abs != "" {
			base := filepath.Base(prefix)
			if strings.Contains(abs, string(os.PathSeparator)+prefix+string(os.PathSeparator)) ||
				strings.HasSuffix(abs, string(os.PathSeparator)+prefix) {
				return true
			}
			_ = base
		}
		return false
	}

	if strings.ContainsAny(pattern, "*?[") {
		ok, err := filepath.Match(pattern, rel)
		if err == nil && ok {
			return true
		}
		// also try match on path segments for patterns like "skills"
		for _, part := range strings.Split(rel, "/") {
			ok, err := filepath.Match(pattern, part)
			if err == nil && ok {
				return true
			}
		}
		return false
	}

	if rel == pattern {
		return true
	}
	// bare app name
	if !strings.Contains(pattern, "/") {
		app, _, _ := strings.Cut(rel, "/")
		if app == pattern {
			return true
		}
	}
	if abs != "" && (abs == pattern || strings.HasSuffix(abs, "/"+pattern)) {
		return true
	}
	return false
}

// SuggestForEntry picks a useful ignore pattern for a selected entry.
// Nested files → ignore their parent dir (e.g. opencode/skills/).
// Top-level app files → ignore that file. Home → home/<name>.
func SuggestForEntry(app, rel string) string {
	key := RelPath(app, rel)
	parts := strings.Split(key, "/")
	if app == "home" {
		return key
	}
	if len(parts) <= 2 {
		// app/file.toml → ignore just that file
		return key
	}
	// app/dir/.../file → ignore app/dir/
	return parts[0] + "/" + parts[1] + "/"
}

// SuggestApp ignores the whole app.
func SuggestApp(app, rel string) string {
	if app == "home" {
		return RelPath(app, rel)
	}
	if app == "config" {
		return rel
	}
	return app + "/"
}

// FormatStatus is a short user-facing summary.
func (l *List) FormatStatus() string {
	n := 0
	if l != nil {
		n = len(l.Patterns)
	}
	return fmt.Sprintf("%d ignores", n)
}
