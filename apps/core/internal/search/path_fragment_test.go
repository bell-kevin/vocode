package search

import (
	"os"
	"path/filepath"
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
