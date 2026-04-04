package search

import "strings"

// TrimSttTrailingSentenceDot removes trailing period(s) when the token already contains an inner dot,
// e.g. "index.tsx." → "index.tsx". Speech recognition often appends a sentence-ending period after a filename.
// Single-segment names like "README." are left unchanged (no inner dot).
func TrimSttTrailingSentenceDot(s string) string {
	s = strings.TrimSpace(s)
	for strings.HasSuffix(s, ".") {
		rest := strings.TrimSuffix(s, ".")
		if !strings.Contains(rest, ".") {
			break
		}
		s = rest
	}
	return s
}
