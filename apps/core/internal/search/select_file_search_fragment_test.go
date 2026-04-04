package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeSelectFileSearchQuery_basenameOnly(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Evade")
	resDir := filepath.Join(root, "Res")
	if err := os.MkdirAll(resDir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(resDir, "game.js")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if g := NormalizeSelectFileSearchQuery(root, file); g != "game.js" {
		t.Fatalf("abs file: got %q want game.js", g)
	}
	if g := NormalizeSelectFileSearchQuery(root, filepath.Join("Res", "game.js")); g != "game.js" {
		t.Fatalf("relative multi-segment: got %q want game.js", g)
	}
	if g := NormalizeSelectFileSearchQuery(root, "game.js"); g != "game.js" {
		t.Fatalf("basename: got %q", g)
	}
	if g := NormalizeSelectFileSearchQuery(root, "index.tsx."); g != "index.tsx" {
		t.Fatalf("stt trailing dot: got %q want index.tsx", g)
	}
	if g := NormalizeSelectFileSearchQuery(root, root); g != filepath.Base(root) {
		t.Fatalf("abs workspace dir: got %q want %q", g, filepath.Base(root))
	}
	outside := filepath.Join(root, "..", "other", "x.js")
	if err := os.MkdirAll(filepath.Dir(outside), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if g := NormalizeSelectFileSearchQuery(root, outside); g != "x.js" {
		t.Fatalf("outside workspace abs: got %q want x.js", g)
	}
}

func TestNormalizeSelectFileSearchQuery_stripsSpokenGoTo(t *testing.T) {
	root := t.TempDir()
	if g := NormalizeSelectFileSearchQuery(root, "go to the explore file"); g != "explore" {
		t.Fatalf("got %q want explore", g)
	}
	if g := NormalizeSelectFileSearchQuery(root, "find the explore file"); g != "explore" {
		t.Fatalf("got %q want explore", g)
	}
}
