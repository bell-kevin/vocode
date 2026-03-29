package workspace

import (
	"path/filepath"
	"testing"
)

func TestEffectiveWorkspaceRoot(t *testing.T) {
	t.Parallel()
	wd := t.TempDir()
	abs := filepath.Join(wd, "deep", "file.go")

	if got := EffectiveWorkspaceRoot("/ws", abs); got != "/ws" {
		t.Fatalf("got %q want /ws", got)
	}
	wantDir := filepath.Dir(filepath.Clean(abs))
	if got := EffectiveWorkspaceRoot("", abs); got != wantDir {
		t.Fatalf("got %q want %q", got, wantDir)
	}
	if got := EffectiveWorkspaceRoot("", ""); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}

func TestResolveTargetPath(t *testing.T) {
	t.Parallel()
	wd := t.TempDir()
	abs := filepath.Join(wd, "deep", "file.go")
	rel := filepath.Join("pkg", "a.go")

	t.Run("empty target is active file", func(t *testing.T) {
		got := ResolveTargetPath("/ws", abs, "")
		if got != filepath.Clean(abs) {
			t.Fatalf("got %q want %q", got, filepath.Clean(abs))
		}
	})

	t.Run("absolute target", func(t *testing.T) {
		got := ResolveTargetPath("/ws", abs, abs)
		if got != filepath.Clean(abs) {
			t.Fatalf("got %q want %q", got, filepath.Clean(abs))
		}
	})

	t.Run("relative joins workspace", func(t *testing.T) {
		got := ResolveTargetPath(wd, abs, rel)
		want := filepath.Clean(filepath.Join(wd, rel))
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("empty workspace root falls back to active file directory", func(t *testing.T) {
		want := filepath.Clean(filepath.Join(filepath.Dir(abs), rel))
		got := ResolveTargetPath("", abs, rel)
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})
}
