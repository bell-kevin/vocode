package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathFragmentSearch_findsByFileName(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "pkg", "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(sub, "test.js")
	if err := os.WriteFile(want, []byte("// empty\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(root, "other.go")
	if err := os.WriteFile(other, []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := PathFragmentMatches(root, "test.js", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Path != filepath.Clean(want) || got[0].IsDir {
		t.Fatalf("got %v want file [%s]", got, filepath.Clean(want))
	}
}

func TestPathFragmentSearch_strongDirKeepsStrongBasenameFilesInside(t *testing.T) {
	root := t.TempDir()
	appDir := filepath.Join(root, "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	appTSX := filepath.Join(appDir, "App.tsx")
	other := filepath.Join(appDir, "other.go")
	if err := os.WriteFile(appTSX, []byte("export {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := PathFragmentMatches(root, "app", 20)
	if err != nil {
		t.Fatal(err)
	}
	var sawDir, sawTSX, sawOther bool
	for _, m := range got {
		switch filepath.Clean(m.Path) {
		case filepath.Clean(appDir):
			sawDir = m.IsDir
		case filepath.Clean(appTSX):
			sawTSX = !m.IsDir
		case filepath.Clean(other):
			sawOther = !m.IsDir
		}
	}
	if !sawDir || !sawTSX {
		t.Fatalf("want app/ directory and App.tsx inside it, got %#v", got)
	}
	if sawOther {
		t.Fatalf("other.go matched only via parent path; should be pruned, got %#v", got)
	}
}

func TestPathFragmentSearch_folderMatchDoesNotListAllDescendants(t *testing.T) {
	root := t.TempDir()
	resDir := filepath.Join(root, "Res")
	if err := os.MkdirAll(resDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.ogg", "b.png", "test.js"} {
		if err := os.WriteFile(filepath.Join(resDir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := PathFragmentMatches(root, "Res", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d matches, want 1 (folder only): %#v", len(got), got)
	}
	if !got[0].IsDir || filepath.Clean(got[0].Path) != filepath.Clean(resDir) {
		t.Fatalf("want only dir %s, got %#v", resDir, got[0])
	}
}

func TestPathFragmentSearch_assetsSuffixRanksExactBeforeMyAssets(t *testing.T) {
	root := t.TempDir()
	exact := filepath.Join(root, "src", "assets")
	myAssets := filepath.Join(root, "src", "my-assets")
	if err := os.MkdirAll(exact, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(myAssets, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := PathFragmentMatches(root, "assets", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 dirs, got %#v", got)
	}
	if filepath.Clean(got[0].Path) != filepath.Clean(exact) {
		t.Fatalf("want exact assets first, got %#v", got)
	}
	if filepath.Clean(got[1].Path) != filepath.Clean(myAssets) {
		t.Fatalf("want my-assets second, got %#v", got)
	}
}

func TestPathFragmentSearch_dirMatchesBasenameNotParentPathSegment(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "assets")
	images := filepath.Join(assets, "images")
	if err := os.MkdirAll(images, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := PathFragmentMatches(root, "assets", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].IsDir || filepath.Clean(got[0].Path) != filepath.Clean(assets) {
		t.Fatalf("want only assets/ (not images/), got %#v", got)
	}
}

func TestPathFragmentSearch_nestedAppDirsUnderSomeStuffBothListed(t *testing.T) {
	root := t.TempDir()
	outer := filepath.Join(root, "App")
	inner := filepath.Join(outer, "some_stuff", "App")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := PathFragmentMatches(root, "app", 20)
	if err != nil {
		t.Fatal(err)
	}
	var sawOuter, sawInner bool
	for _, m := range got {
		if !m.IsDir {
			continue
		}
		switch filepath.Clean(m.Path) {
		case filepath.Clean(outer):
			sawOuter = true
		case filepath.Clean(inner):
			sawInner = true
		}
	}
	if !sawOuter || !sawInner {
		t.Fatalf("want both App directories (outer and inner), got %#v", got)
	}
}

func TestPathFragmentSearch_nestedSameNamedDirsBothListed(t *testing.T) {
	root := t.TempDir()
	outer := filepath.Join(root, "Res")
	inner := filepath.Join(outer, "Res")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inner, "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := PathFragmentMatches(root, "Res", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want outer+inner Res dirs (2), got %d: %#v", len(got), got)
	}
	for _, m := range got {
		if !m.IsDir {
			t.Fatalf("expected only directories, got %#v", got)
		}
	}
}

func TestPathFragmentSearch_prependedWorkspaceRootPrunesChildFiles(t *testing.T) {
	// Regression: prepend must run before prune, otherwise .exe/.jar under the repo root stay listed.
	parent := t.TempDir()
	evadeRoot := filepath.Join(parent, "Evade")
	if err := os.MkdirAll(evadeRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Evade Exe.exe", "Evade.jar"} {
		if err := os.WriteFile(filepath.Join(evadeRoot, name), []byte{0}, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := PathFragmentMatches(evadeRoot, "evade", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].IsDir || filepath.Clean(got[0].Path) != filepath.Clean(evadeRoot) {
		t.Fatalf("want only workspace root dir, got %#v", got)
	}
}

func TestPathFragmentSearch_workspaceRootPreferredOverExe(t *testing.T) {
	// Workspace is the repo folder itself; walk never lists root, only children like "Evade Exe.exe".
	parent := t.TempDir()
	evadeRoot := filepath.Join(parent, "Evade")
	if err := os.MkdirAll(evadeRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evadeRoot, "Evade Exe.exe"), []byte{0x4d, 0x5a}, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := PathFragmentMatches(evadeRoot, "find the evade directory", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].IsDir || filepath.Clean(got[0].Path) != filepath.Clean(evadeRoot) {
		t.Fatalf("want only workspace root dir, got %#v", got)
	}
}

func TestPathFragmentSearch_directoryBeatsBinaryWhenBothMatch(t *testing.T) {
	root := t.TempDir()
	evadeDir := filepath.Join(root, "Evade")
	if err := os.MkdirAll(evadeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evadeDir, "Evade Exe.exe"), []byte{0x4d, 0x5a}, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := PathFragmentMatches(root, "evade", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 1 || !got[0].IsDir || filepath.Clean(got[0].Path) != filepath.Clean(evadeDir) {
		t.Fatalf("want top match Evade folder, got %#v", got)
	}
}

func TestPathFragmentSearch_shortQueryAppPrefersInnerAppDirNotParentSubstring(t *testing.T) {
	// Regression: "app" must surface the real app/ folder (exact) before vocoded-app (substring only),
	// and must not prune app/ just because vocoded-app matched.
	parent := t.TempDir()
	ws := filepath.Join(parent, "vocoded-workspace")
	va := filepath.Join(ws, "vocoded-app")
	appDir := filepath.Join(va, "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "index.ts"), []byte("// x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := PathFragmentMatches(ws, "app", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 1 {
		t.Fatalf("expected at least app dir, got %#v", got)
	}
	if !got[0].IsDir || filepath.Clean(got[0].Path) != filepath.Clean(appDir) {
		t.Fatalf("want first hit inner app folder %s, got %#v", appDir, got)
	}
	var sawVA bool
	for _, m := range got {
		if filepath.Clean(m.Path) == filepath.Clean(va) {
			sawVA = true
		}
	}
	if !sawVA {
		t.Fatalf("expected vocoded-app in results (substring dir) after exact app/, got %#v", got)
	}
}

func TestPathFragmentSearch_bareFragmentFindsIndexTsx(t *testing.T) {
	root := t.TempDir()
	tabs := filepath.Join(root, "app", "(tabs)")
	if err := os.MkdirAll(tabs, 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tabs, "index.tsx")
	if err := os.WriteFile(want, []byte("export {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := PathFragmentMatches(root, "index", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].IsDir || filepath.Clean(got[0].Path) != filepath.Clean(want) {
		t.Fatalf("want %s, got %#v", want, got)
	}
}

func TestPathFragmentSearch_bareFragmentReturnsDirAndStemExtFile(t *testing.T) {
	root := t.TempDir()
	appDir := filepath.Join(root, "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	appTS := filepath.Join(root, "app.ts")
	if err := os.WriteFile(appTS, []byte("// x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := PathFragmentMatches(root, "app", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want app/ and app.ts (2 hits), got %#v", got)
	}
	var sawDir, sawTS bool
	for _, m := range got {
		switch filepath.Clean(m.Path) {
		case filepath.Clean(appDir):
			if m.IsDir {
				sawDir = true
			}
		case filepath.Clean(appTS):
			if !m.IsDir {
				sawTS = true
			}
		}
	}
	if !sawDir || !sawTS {
		t.Fatalf("want both directory and file, got %#v", got)
	}
}

func TestPathFragmentSearch_bareFragmentReturnsIndexDirAndIndexTsx(t *testing.T) {
	root := t.TempDir()
	idxDir := filepath.Join(root, "pkg", "index")
	if err := os.MkdirAll(idxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	idxTS := filepath.Join(root, "pkg", "index.tsx")
	if err := os.WriteFile(idxTS, []byte("// x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := PathFragmentMatches(root, "index", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want index/ and index.tsx (2 hits), got %#v", got)
	}
	var sawDir, sawTS bool
	for _, m := range got {
		switch filepath.Clean(m.Path) {
		case filepath.Clean(idxDir):
			if m.IsDir {
				sawDir = true
			}
		case filepath.Clean(idxTS):
			if !m.IsDir {
				sawTS = true
			}
		}
	}
	if !sawDir || !sawTS {
		t.Fatalf("want both directory and file, got %#v", got)
	}
}

func TestPathFragmentSearch_extensionQueryMatchesFileNotDirectory(t *testing.T) {
	root := t.TempDir()
	appDir := filepath.Join(root, "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	appTS := filepath.Join(root, "app.ts")
	if err := os.WriteFile(appTS, []byte("// x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := PathFragmentMatches(root, "app.ts", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].IsDir || filepath.Clean(got[0].Path) != filepath.Clean(appTS) {
		t.Fatalf("want only app.ts file, got %#v", got)
	}
}

func TestPathFragmentSearch_sttTrailingDotOnFileQuery(t *testing.T) {
	root := t.TempDir()
	want := filepath.Join(root, "index.tsx")
	if err := os.WriteFile(want, []byte("export {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := PathFragmentMatches(root, "index.tsx.", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Path != filepath.Clean(want) || got[0].IsDir {
		t.Fatalf("got %v want file [%s]", got, filepath.Clean(want))
	}
}

func TestPreferFilesFromSelectQuery(t *testing.T) {
	if !PreferFilesFromSelectQuery("find the app file") {
		t.Fatal("expected file intent")
	}
	if PreferFilesFromSelectQuery("open the app folder") {
		t.Fatal("folder mention should disable prefer-files")
	}
	if PreferFilesFromSelectQuery("profile.ts") {
		t.Fatal("substring 'file' in profile must not count")
	}
}

func TestPathFragmentSearch_weakParentDirNameDoesNotMatchUnrelatedFiles(t *testing.T) {
	parent := t.TempDir()
	va := filepath.Join(parent, "vocoded-app")
	if err := os.MkdirAll(va, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{".gitignore", "index.ts", "package-lock.json", "readme.md"} {
		if err := os.WriteFile(filepath.Join(va, name), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	appJSON := filepath.Join(va, "app.json")
	if err := os.WriteFile(appJSON, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := PathFragmentMatches(parent, "app", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want only app.json and vocoded-app dir (2), got %#v", got)
	}
	var sawJSON, sawDir bool
	for _, m := range got {
		switch filepath.Clean(m.Path) {
		case filepath.Clean(appJSON):
			sawJSON = !m.IsDir
		case filepath.Clean(va):
			sawDir = m.IsDir
		}
	}
	if !sawJSON || !sawDir {
		t.Fatalf("want app.json file and vocoded-app directory, got %#v", got)
	}
}

func TestPathFragmentSearch_vocodedAppListsFilesBeforeSuffixOnlyDir(t *testing.T) {
	parent := t.TempDir()
	ws := filepath.Join(parent, "vocoded-workspace")
	va := filepath.Join(ws, "vocoded-app")
	if err := os.MkdirAll(va, 0o755); err != nil {
		t.Fatal(err)
	}
	appJSON := filepath.Join(va, "app.json")
	appTSX := filepath.Join(va, "App.tsx")
	for _, p := range []string{appJSON, appTSX} {
		if err := os.WriteFile(p, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := PathFragmentMatches(ws, "app", 20, PathFragmentOptions{PreferFiles: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("want app.json, App.tsx, and vocoded-app (3), got %#v", got)
	}
	var sawJSON, sawTSX, sawDir bool
	for _, m := range got {
		switch filepath.Clean(m.Path) {
		case filepath.Clean(appJSON):
			sawJSON = !m.IsDir
		case filepath.Clean(appTSX):
			sawTSX = !m.IsDir
		case filepath.Clean(va):
			sawDir = m.IsDir
		}
	}
	if !sawJSON || !sawTSX || !sawDir {
		t.Fatalf("want all three paths, got %#v", got)
	}
	dirIdx := -1
	for i, m := range got {
		if filepath.Clean(m.Path) == filepath.Clean(va) {
			dirIdx = i
			break
		}
	}
	if dirIdx != 2 {
		t.Fatalf("want suffix-only dir last (idx 2), got index %d in %#v", dirIdx, got)
	}
	// Both concrete files should precede the weak directory match.
	if got[0].IsDir || got[1].IsDir {
		t.Fatalf("want first two hits to be files, got %#v", got)
	}
	if !strings.EqualFold(filepath.Base(got[0].Path), "app.json") && !strings.EqualFold(filepath.Base(got[0].Path), "App.tsx") {
		t.Fatalf("unexpected first hit %#v", got[0])
	}
}

func TestPathFragmentSearch_skipsNodeModules(t *testing.T) {
	root := t.TempDir()
	nm := filepath.Join(root, "node_modules", "pkg")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	hidden := filepath.Join(nm, "test.js")
	if err := os.WriteFile(hidden, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	visible := filepath.Join(root, "test.js")
	if err := os.WriteFile(visible, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := PathFragmentMatches(root, "test.js", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Path != filepath.Clean(visible) {
		t.Fatalf("got %v want only root test.js", got)
	}
}
