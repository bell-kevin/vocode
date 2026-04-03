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

	got, err := PathFragmentSearch(root, "test.js", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != filepath.Clean(want) {
		t.Fatalf("got %v want [%s]", got, filepath.Clean(want))
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

	got, err := PathFragmentSearch(root, "test.js", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != filepath.Clean(visible) {
		t.Fatalf("got %v want only root test.js", got)
	}
}
