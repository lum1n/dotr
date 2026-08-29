package tui

import (
	"strings"
)

// fuzzyScore returns a match score for query against hay (both already lowercased).
// Higher is better; -1 means no match. Empty query matches everything with score 0.
func fuzzyScore(query, hay string) int {
	query = strings.TrimSpace(query)
	if query == "" {
		return 0
	}
	parts := strings.Fields(query)
	if len(parts) > 1 {
		total := 0
		for _, p := range parts {
			s := fuzzyScore(p, hay)
			if s < 0 {
				return -1
			}
			total += s
		}
		return total
	}

	q := []rune(parts[0])
	hayRunes := []rune(hay)

	// Prefer contiguous substring.
	if i := strings.Index(hay, parts[0]); i >= 0 {
		score := 2000 - i
		if i == 0 || (i > 0 && (hay[i-1] == '/' || hay[i-1] == '.' || hay[i-1] == ' ')) {
			score += 80
		}
		return score
	}

	qi := 0
	score := 0
	consecutive := 0
	for hi, r := range hayRunes {
		if qi >= len(q) {
			break
		}
		if r != q[qi] {
			consecutive = 0
			continue
		}
		score += 10 + consecutive*5
		if hi == 0 || hayRunes[hi-1] == '/' || hayRunes[hi-1] == '.' || hayRunes[hi-1] == ' ' {
			score += 25
		}
		consecutive++
		qi++
	}
	if qi < len(q) {
		return -1
	}
	return score
}

func searchHaystack(app, rel, abs string) string {
	base := abs
	if i := strings.LastIndexAny(abs, "/\\"); i >= 0 {
		base = abs[i+1:]
	}
	return strings.ToLower(app + "/" + rel + " " + abs + " " + base)
}

func normalizeQuery(q string) string {
	return strings.ToLower(strings.TrimSpace(q))
}
