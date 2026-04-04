package search

import (
	"path/filepath"
	"strings"
)

// NormalizeSelectFileSearchQuery returns a single path segment (file or folder basename) for
// [PathFragmentMatches]. The classifier must output only a filename-style token (no slashes);
// this still strips mistakes such as a full absolute path or "Res/game.js" down to "game.js".
func NormalizeSelectFileSearchQuery(root, q string) string {
	_ = root // search scope is the caller's workspace root; basename does not depend on it
	q = strings.TrimSpace(StripFileSearchSpokenFiller(strings.TrimSpace(q)))
	if q == "" {
		return ""
	}
	slash := filepath.ToSlash(q)
	for len(slash) > 1 && strings.HasSuffix(slash, "/") {
		slash = strings.TrimSuffix(slash, "/")
	}
	native := filepath.FromSlash(slash)
	if native == "" {
		return ""
	}
	native = filepath.Clean(native)
	base := filepath.Base(native)
	if base == "." || base == ".." {
		return ""
	}
	base = TrimSttTrailingSentenceDot(base)
	return filepath.ToSlash(base)
}
