package workspace

import (
	"path/filepath"
	"testing"
)

func TestPathSearchWorkspaceRoot_usesExplicitPathSearchWhenSet(t *testing.T) {
	outer := filepath.FromSlash("/ws/parent")
	inner := filepath.FromSlash("/ws/parent/child")
	got := PathSearchWorkspaceRoot(inner, outer, filepath.FromSlash("/ws/parent/child/file.ts"))
	if got != outer {
		t.Fatalf("got %q want outer %q", got, outer)
	}
}

func TestPathSearchWorkspaceRoot_fallsBackToEffective(t *testing.T) {
	want := filepath.Clean(filepath.FromSlash("/ws"))
	got := PathSearchWorkspaceRoot(want, "", filepath.FromSlash("/ws/a/b/c.ts"))
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
