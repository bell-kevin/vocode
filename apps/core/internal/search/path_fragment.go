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

// PathMatch is one file or directory path from path-fragment discovery.
type PathMatch struct {
	Path  string
	IsDir bool
}

func pathFragmentMatches(rel, baseName, lowerFragment string) bool {
	relLower := strings.ToLower(rel)
	baseLower := strings.ToLower(baseName)
	return strings.Contains(relLower, lowerFragment) || strings.Contains(baseLower, lowerFragment)
}

// PathFragmentMatches lists files and directories under root whose relative path or base name contains
// fragment (case-insensitive). Used for select_file — not content search.
// Returns up to maxPaths matches sorted by path. maxPaths <= 0 defaults to 20.
func PathFragmentMatches(root, fragment string, maxPaths int) ([]PathMatch, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	fragment = strings.TrimSpace(fragment)
	if root == "" || fragment == "" {
		return nil, nil
	}
	if maxPaths <= 0 {
		maxPaths = 20
	}
	lower := strings.ToLower(fragment)

	var matches []PathMatch
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
			if path == root {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			if pathFragmentMatches(rel, name, lower) {
				matches = append(matches, PathMatch{Path: filepath.Clean(path), IsDir: true})
			}
			if len(matches) >= maxPaths*4 {
				return filepath.SkipAll
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		base := filepath.Base(path)
		if !pathFragmentMatches(rel, base, lower) {
			return nil
		}
		matches = append(matches, PathMatch{Path: filepath.Clean(path), IsDir: false})
		if len(matches) >= maxPaths*4 {
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return uniqueSortedPathMatchesCap(matches, maxPaths), nil
}

func uniqueSortedPathMatchesCap(items []PathMatch, max int) []PathMatch {
	if len(items) == 0 {
		return nil
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	out := make([]PathMatch, 0, max)
	var last string
	for _, it := range items {
		if it.Path == "" {
			continue
		}
		if len(out) > 0 && it.Path == last {
			continue
		}
		out = append(out, it)
		last = it.Path
		if len(out) >= max {
			break
		}
	}
	return out
}
