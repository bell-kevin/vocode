package fileselectflow

import (
	"testing"

	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
)

func TestApplyFileSelectionControlOp_wrapNextBack(t *testing.T) {
	vs := &session.VoiceSession{
		FileSelectionPaths: []string{"/a", "/b", "/c"},
		FileSelectionIndex: 0,
	}
	applyFileSelectionControlOp(vs, "next", 0)
	if vs.FileSelectionIndex != 1 || vs.FileSelectionFocus != "/b" {
		t.Fatalf("next: idx=%d focus=%q", vs.FileSelectionIndex, vs.FileSelectionFocus)
	}
	vs.FileSelectionIndex = 2
	applyFileSelectionControlOp(vs, "next", 0)
	if vs.FileSelectionIndex != 0 || vs.FileSelectionFocus != "/a" {
		t.Fatalf("next wrap: idx=%d focus=%q", vs.FileSelectionIndex, vs.FileSelectionFocus)
	}
	applyFileSelectionControlOp(vs, "back", 0)
	if vs.FileSelectionIndex != 2 || vs.FileSelectionFocus != "/c" {
		t.Fatalf("back wrap: idx=%d focus=%q", vs.FileSelectionIndex, vs.FileSelectionFocus)
	}
}
