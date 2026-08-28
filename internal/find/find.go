// Package find matches scanned config entries by query.
package find

import (
	"path/filepath"
	"strings"

	"github.com/lum1n/dotr/internal/scan"
)

// Display returns the UI-style path for an entry.
func Display(e scan.Entry) string {
	switch e.App {
	case "home", "config":
		return e.RelPath
	default:
		return filepath.Join(e.App, e.RelPath)
	}
}

// Match returns entries whose display path, app, or basename contains query (case-insensitive).
// Multi-word queries require all terms.
func Match(entries []scan.Entry, query string) []scan.Entry {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return append([]scan.Entry{}, entries...)
	}
	parts := strings.Fields(q)
	var out []scan.Entry
	for _, e := range entries {
		hay := strings.ToLower(Display(e) + " " + e.AbsPath + " " + e.App)
		ok := true
		for _, p := range parts {
			if !strings.Contains(hay, p) {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, e)
		}
	}
	return out
}

// Best returns the single best match, preferring exact basename then shortest display path.
func Best(entries []scan.Entry, query string) (scan.Entry, bool) {
	matches := Match(entries, query)
	if len(matches) == 0 {
		return scan.Entry{}, false
	}
	if len(matches) == 1 {
		return matches[0], true
	}
	q := strings.ToLower(strings.TrimSpace(query))
	var best scan.Entry
	bestScore := -1
	for _, e := range matches {
		score := 0
		base := strings.ToLower(filepath.Base(e.AbsPath))
		disp := strings.ToLower(Display(e))
		if base == q || disp == q {
			score += 100
		}
		if strings.HasPrefix(disp, q) {
			score += 40
		}
		if strings.Contains(base, q) {
			score += 20
		}
		score -= len(disp) // prefer shorter
		if score > bestScore {
			bestScore = score
			best = e
		}
	}
	return best, true
}
