package search

import (
	"sort"
	"strings"
)

// UniqueSortedPaths returns distinct paths from hits in sorted order, capped at maxPaths.
func UniqueSortedPaths(hits []Hit, maxPaths int) []string {
	if len(hits) == 0 || maxPaths <= 0 {
		return nil
	}
	raw := make([]string, 0, len(hits))
	for _, h := range hits {
		p := strings.TrimSpace(h.Path)
		if p != "" {
			raw = append(raw, p)
		}
	}
	if len(raw) == 0 {
		return nil
	}
	sort.Strings(raw)
	out := make([]string, 0, len(raw))
	var last string
	for i, p := range raw {
		if i == 0 || p != last {
			out = append(out, p)
			last = p
			if len(out) >= maxPaths {
				break
			}
		}
	}
	return out
}
