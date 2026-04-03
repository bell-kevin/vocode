package search

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

var pathSearchSkipDirNames = map[string]struct{}{
	".git":         {},
	"node_modules": {},
	".pnpm-store":  {},
	"vendor":       {},
	"dist":         {},
	"bin":          {},
	".turbo":       {},
	"__pycache__":  {},
	".venv":        {},
	"target":       {}, // Rust
	".idea":        {},
}

// PathFragmentSearch lists files under root whose relative path or base name contains
// fragment (case-insensitive). Used for select_file / path discovery — not content search.
// Returns up to maxPaths sorted paths (clean, absolute). maxPaths <= 0 defaults to 20.
func PathFragmentSearch(root, fragment string, maxPaths int) ([]string, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	fragment = strings.TrimSpace(fragment)
	if root == "" || fragment == "" {
		return nil, nil
	}
	if maxPaths <= 0 {
		maxPaths = 20
	}
	lower := strings.ToLower(fragment)

	var matches []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if path != root {
				if _, skip := pathSearchSkipDirNames[name]; skip {
					return filepath.SkipDir
				}
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		base := strings.ToLower(filepath.Base(path))
		relLower := strings.ToLower(rel)
		if !strings.Contains(relLower, lower) && !strings.Contains(base, lower) {
			return nil
		}
		matches = append(matches, filepath.Clean(path))
		if len(matches) >= maxPaths*4 {
			// Collect extras then unique+sort+cap (simple bound on walk work).
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return uniqueSortedPathsCap(matches, maxPaths), nil
}

func uniqueSortedPathsCap(paths []string, max int) []string {
	if len(paths) == 0 {
		return nil
	}
	sort.Strings(paths)
	out := make([]string, 0, max)
	var last string
	for _, p := range paths {
		if p == "" {
			continue
		}
		if len(out) > 0 && p == last {
			continue
		}
		out = append(out, p)
		last = p
		if len(out) >= max {
			break
		}
	}
	return out
}
