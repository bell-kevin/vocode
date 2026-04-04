package search

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
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

// PathFragmentOptions tweaks ranking (and related behavior) for [PathFragmentMatches].
// The zero value is the default; extra arguments are optional.
type PathFragmentOptions struct {
	// PreferFiles is true when the spoken query clearly asked for a file (e.g. "the app file")
	// and not a folder/directory. Suffix-only directory matches rank lower.
	PreferFiles bool
}

// PreferFilesFromSelectQuery reports whether the raw file-select utterance favors files over
// directories. It is conservative: "profile" must not count as "file".
func PreferFilesFromSelectQuery(q string) bool {
	q = strings.TrimSpace(strings.ToLower(q))
	if q == "" {
		return false
	}
	for _, w := range []string{"folder", "folders", "directory", "directories"} {
		if selectQueryHasWholeWord(q, w) {
			return false
		}
	}
	return selectQueryHasWholeWord(q, "file")
}

func selectQueryHasWholeWord(q, w string) bool {
	for _, tok := range strings.FieldsFunc(q, func(r rune) bool {
		return r == '_' || unicode.IsPunct(r) || unicode.IsSpace(r)
	}) {
		if tok == w {
			return true
		}
	}
	return false
}

// dirSegmentStrongBasenameMatch is true when a path segment is an exact/spoken match for the
// fragment (not a mere suffix like vocoded-app for app).
func dirSegmentStrongBasenameMatch(seg, fragment string) bool {
	seg = strings.TrimSpace(seg)
	fragment = strings.TrimSpace(fragment)
	if seg == "" || fragment == "" {
		return false
	}
	if strings.EqualFold(seg, fragment) {
		return true
	}
	return fragmentRefersToBasename(seg, fragment)
}

// dirSegmentWeakSuffixOnlyMatch is true for directory names that match the fragment only via
// HasSuffix (e.g. vocoded-app for app), used to ignore those segments when matching files by path.
func dirSegmentWeakSuffixOnlyMatch(seg, fragment string) bool {
	if !dirBasenameMatchesSelectFileFragment(seg, fragment) {
		return false
	}
	return !dirSegmentStrongBasenameMatch(seg, fragment)
}

// filePathFragmentMatches decides whether a file path matches the fragment. Unlike naive rel
// substring match, parent folders that only match by suffix (vocoded-app for app) do not make
// every file under them match; the basename must match, or the fragment must appear in rel outside
// those weak segments.
func filePathFragmentMatches(rel, baseName, lowerFragment, fragment string) bool {
	if pathContainsFold(baseName, lowerFragment) {
		return true
	}
	segs := strings.Split(rel, "/")
	if len(segs) == 0 {
		return false
	}
	var b strings.Builder
	for i, seg := range segs {
		if i < len(segs)-1 && dirSegmentWeakSuffixOnlyMatch(seg, fragment) {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('/')
		}
		b.WriteString(seg)
	}
	return pathContainsFold(b.String(), lowerFragment)
}

// fragmentLooksLikeFileOrExtQuery is true when the query names a file-like token (any "." in the fragment),
// e.g. "app.tsx". Directory discovery must not treat that as a folder name query.
func fragmentLooksLikeFileOrExtQuery(fragment string) bool {
	return strings.Contains(strings.TrimSpace(fragment), ".")
}

// dirBasenameMatchesSelectFileFragment is true when the directory's own name ends with the query
// (case-insensitive), e.g. assets, my-assets, vocoded-app for "app". It does not match parent path
// segments (assets/images for "assets") nor infix-only names (xassetsy for "assets").
func dirBasenameMatchesSelectFileFragment(dirName, fragment string) bool {
	fragment = strings.TrimSpace(fragment)
	dirName = strings.TrimSpace(dirName)
	if fragment == "" || dirName == "" {
		return false
	}
	return strings.HasSuffix(strings.ToLower(dirName), strings.ToLower(fragment))
}

// PathFragmentMatches lists paths for file_select. The fragment is trimmed with [TrimSttTrailingSentenceDot]
// (e.g. "index.tsx." → "index.tsx"). If the fragment contains ".", it is treated as a file-style query:
// no directory matches (and workspace-root prepend is skipped). Otherwise every matching directory and file
// is kept: basename or relative path substring (case-insensitive), including similarly named files next to
// folders (e.g. both app/ and app.ts for "app"). Files do not match solely because a parent folder name
// ends with the fragment (e.g. vocoded-app for app must not match every file inside).
// Directories match when their basename ends with the fragment (case-insensitive): src/assets, my-assets,
// vocoded-app for "app". Exact / spoken-token names rank above suffix-only names.
// Per-segment resolution uses [ResolveWorkspaceRelativePath].
// Every matching directory is kept, including nested dirs with the same name (e.g. Res/Res or
// App/some_stuff/App). Pruning never removes a directory from the list—only file entries that lie
// strictly inside a strong folder hit (basename equals fragment, case-insensitive, or spoken-token
// match—not a mere suffix like vocoded-app for "app") when the file matched only via that path—
// except files whose own basename still strongly matches the fragment (e.g. app/App.tsx for "app").
// Prepending the workspace root (when the query names that folder) runs before pruning.
// maxPaths <= 0 defaults to 20.
func PathFragmentMatches(root, fragment string, maxPaths int, opts ...PathFragmentOptions) ([]PathMatch, error) {
	var opt PathFragmentOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	root = filepath.Clean(strings.TrimSpace(root))
	fragment = TrimSttTrailingSentenceDot(strings.TrimSpace(fragment))
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
			// Directory hits: basename ends with fragment, or spoken phrase names this segment
			// (e.g. "find the evade directory" + folder Evade). File-like queries (contain ".") never match dirs.
			if !fragmentLooksLikeFileOrExtQuery(fragment) &&
				(dirBasenameMatchesSelectFileFragment(name, fragment) || fragmentRefersToBasename(name, fragment)) {
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
		if !filePathFragmentMatches(rel, base, lower, fragment) {
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
	matches = prependWorkspaceRootIfBasenameMatches(root, fragment, matches)
	matches = pruneDescendantFilesUnderStrongDirs(matches, fragment, root)
	rankPathFragmentMatches(matches, fragment, opt)
	return uniqueSortedPathMatchesCap(matches, maxPaths), nil
}

// isStrictDescendant reports whether child is inside parent (not equal).
func isStrictDescendant(child, parent string) bool {
	child = filepath.Clean(child)
	parent = filepath.Clean(parent)
	if child == parent || parent == "" || child == "" {
		return false
	}
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	if rel == "." {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}

// dirStrongBasenameForPruningFiles is true when this directory is a strong name match for the
// fragment (equal fold or spoken-token basename), not merely a suffix (e.g. vocoded-app for "app").
// Only then do we drop file hits that lie strictly inside it—never subdirectory hits.
func dirStrongBasenameForPruningFiles(dirPath, fragment string) bool {
	return dirSegmentStrongBasenameMatch(filepath.Base(dirPath), fragment)
}

// fileBasenameStrongFragmentMatch is true when the file's own basename matches the fragment on
// merit (stem or spoken token), not only because a parent path segment contained the fragment.
// Such files are kept even inside a strong-matching folder (e.g. app/App.tsx for "app").
func fileBasenameStrongFragmentMatch(baseName, fragment string) bool {
	if fragmentRefersToBasename(baseName, fragment) {
		return true
	}
	return fileStemFragmentMatchTier(baseName, fragment) >= 1
}

// pruneDescendantFilesUnderStrongDirs removes file entries that lie strictly inside a directory
// that is a strong basename match (see [dirStrongBasenameForPruningFiles]), except files whose
// basenames are themselves a strong match ([fileBasenameStrongFragmentMatch]) when that directory
// is not the search root. Under the search root itself (e.g. prepended workspace folder Evade),
// all descendant files are still pruned so listing the repo folder does not flood with children.
func pruneDescendantFilesUnderStrongDirs(matches []PathMatch, fragment, searchRoot string) []PathMatch {
	searchRoot = filepath.Clean(strings.TrimSpace(searchRoot))
	var dirPaths []string
	for _, m := range matches {
		if m.IsDir && m.Path != "" && dirStrongBasenameForPruningFiles(m.Path, fragment) {
			dirPaths = append(dirPaths, filepath.Clean(m.Path))
		}
	}
	if len(dirPaths) == 0 {
		return matches
	}
	out := make([]PathMatch, 0, len(matches))
	for _, m := range matches {
		p := filepath.Clean(m.Path)
		if p == "" {
			continue
		}
		drop := false
		if !m.IsDir {
			base := filepath.Base(p)
			for _, d := range dirPaths {
				if p == d {
					break
				}
				if isStrictDescendant(p, d) {
					if filepath.Clean(d) == searchRoot {
						drop = true
					} else if !fileBasenameStrongFragmentMatch(base, fragment) {
						drop = true
					}
					break
				}
			}
		}
		if !drop {
			out = append(out, m)
		}
	}
	return out
}

// StripFileSearchSpokenFiller removes common speech tokens so path matching sees name tokens
// (e.g. "go to the explore file" → "explore"). Used by [NormalizeSelectFileSearchQuery].
func StripFileSearchSpokenFiller(q string) string {
	var b strings.Builder
	for _, raw := range strings.Fields(strings.TrimSpace(q)) {
		w := strings.Trim(raw, ".,;:!?")
		if w == "" {
			continue
		}
		if _, stop := fileSearchQueryStopWords[strings.ToLower(w)]; stop {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(w)
	}
	return b.String()
}

// fileSearchQueryStopWords strips spoken noise from file_select search queries when matching basenames.
var fileSearchQueryStopWords = map[string]struct{}{
	"find": {}, "the": {}, "a": {}, "an": {}, "open": {}, "show": {}, "please": {},
	"goto": {}, "go": {}, "to": {}, "folder": {}, "folders": {}, "directory": {}, "directories": {},
	"file": {}, "path": {}, "need": {}, "want": {}, "where": {}, "is": {}, "my": {},
	"this": {}, "that": {}, "me": {}, "can": {}, "you": {},
}

var pathSearchBinaryLikeExt = map[string]struct{}{
	".exe": {}, ".dll": {}, ".dylib": {}, ".so": {}, ".bin": {}, ".pdb": {},
	".obj": {}, ".o": {}, ".class": {}, ".jar": {}, ".wasm": {},
	".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".webp": {}, ".ico": {},
	".pdf": {}, ".zip": {}, ".gz": {}, ".7z": {}, ".rar": {},
}

func isPathSearchBinaryLikeBase(name string) bool {
	_, ok := pathSearchBinaryLikeExt[strings.ToLower(filepath.Ext(name))]
	return ok
}

// IsBinaryLikeFileName reports extensions we should not auto-open as editor text (executables, archives, images).
func IsBinaryLikeFileName(name string) bool {
	return isPathSearchBinaryLikeBase(name)
}

// fragmentRefersToBasename reports whether a spoken query is really about this path segment
// (e.g. "find the evade directory" → base "Evade").
func fragmentRefersToBasename(baseName, fragment string) bool {
	baseName = strings.TrimSpace(baseName)
	fragment = strings.TrimSpace(fragment)
	if baseName == "" || fragment == "" {
		return false
	}
	if strings.EqualFold(baseName, fragment) {
		return true
	}
	if NormalizePathTokenForMatch(baseName) == NormalizePathTokenForMatch(fragment) {
		return true
	}
	lowerB := strings.ToLower(baseName)
	for _, w := range strings.Fields(strings.ToLower(fragment)) {
		if len(w) < 2 {
			continue
		}
		if _, stop := fileSearchQueryStopWords[w]; stop {
			continue
		}
		if lowerB == w {
			return true
		}
		if NormalizePathTokenForMatch(baseName) == NormalizePathTokenForMatch(w) {
			return true
		}
	}
	return false
}

// prependWorkspaceRootIfBasenameMatches adds the workspace folder itself when the walk never lists
// path==root but the user is clearly asking for that folder (e.g. repo "Evade" + "Evade Exe.exe" inside).
func prependWorkspaceRootIfBasenameMatches(root, fragment string, matches []PathMatch) []PathMatch {
	root = filepath.Clean(strings.TrimSpace(root))
	fragment = strings.TrimSpace(fragment)
	if root == "" || fragment == "" {
		return matches
	}
	if fragmentLooksLikeFileOrExtQuery(fragment) {
		return matches
	}
	base := filepath.Base(root)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return matches
	}
	if !fragmentRefersToBasename(base, fragment) {
		return matches
	}
	for _, m := range matches {
		if strings.EqualFold(filepath.Clean(m.Path), root) {
			return matches
		}
	}
	return append([]PathMatch{{Path: root, IsDir: true}}, matches...)
}

// fileStemFragmentMatchTier: 2 = stem equals fragment (fold), 1 = stem starts with fragment, 0 = other.
func fileStemFragmentMatchTier(baseName, fragment string) int {
	lowerF := strings.ToLower(strings.TrimSpace(fragment))
	if lowerF == "" {
		return 0
	}
	ext := filepath.Ext(baseName)
	stem := strings.ToLower(strings.TrimSuffix(baseName, ext))
	if stem == lowerF {
		return 2
	}
	if len(lowerF) >= 1 && strings.HasPrefix(stem, lowerF) {
		return 1
	}
	return 0
}

func pathFragmentMatchScore(m PathMatch, fragment string, opts PathFragmentOptions) int {
	score := 0
	base := filepath.Base(m.Path)
	lowerF := strings.ToLower(strings.TrimSpace(fragment))
	lowerB := strings.ToLower(base)
	depth := strings.Count(filepath.ToSlash(filepath.Clean(m.Path)), "/")
	if m.IsDir {
		score += 120
		if fragmentRefersToBasename(base, fragment) || strings.EqualFold(strings.TrimSpace(base), strings.TrimSpace(fragment)) {
			score += 720
		} else if lowerF != "" && strings.HasSuffix(lowerB, lowerF) {
			// Ends-with only (e.g. my-assets, vocoded-app): below direct file-name hits.
			score += 60
		}
		if opts.PreferFiles {
			score -= 320
		}
	} else {
		score += 260
		if isPathSearchBinaryLikeBase(base) {
			score -= 400
		}
		switch fileStemFragmentMatchTier(base, fragment) {
		case 2:
			score += 560
		case 1:
			score += 380
		}
		if fragmentRefersToBasename(base, fragment) {
			score += 120
		} else if pathContainsFold(base, lowerF) && fileStemFragmentMatchTier(base, fragment) == 0 {
			// Basename substring only (e.g. mapper.go for "app" if it ever matched).
			score += 80
		}
	}
	score -= depth * 3
	return score
}

func rankPathFragmentMatches(matches []PathMatch, fragment string, opts PathFragmentOptions) {
	sort.SliceStable(matches, func(i, j int) bool {
		si := pathFragmentMatchScore(matches[i], fragment, opts)
		sj := pathFragmentMatchScore(matches[j], fragment, opts)
		if si != sj {
			return si > sj
		}
		return matches[i].Path < matches[j].Path
	})
}

func uniqueSortedPathMatchesCap(items []PathMatch, max int) []PathMatch {
	if len(items) == 0 {
		return nil
	}
	// Preserve order from [rankPathFragmentMatches] (exact directory names before substring-only).
	seen := make(map[string]struct{}, len(items))
	out := make([]PathMatch, 0, max)
	for _, it := range items {
		if it.Path == "" {
			continue
		}
		k := filepath.Clean(it.Path)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, it)
		if len(out) >= max {
			break
		}
	}
	return out
}
