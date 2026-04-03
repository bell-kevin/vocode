package workspaceselectflow

import (
	"testing"

	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
)

func TestApplySelectControlOp_wrapNextBack(t *testing.T) {
	vs := &session.VoiceSession{
		SearchResults: []session.SearchHit{
			{Path: "a"}, {Path: "b"}, {Path: "c"},
		},
		ActiveSearchIndex: 0,
	}
	applySelectControlOp(vs, "next", 0)
	if vs.ActiveSearchIndex != 1 {
		t.Fatalf("next: got %d want 1", vs.ActiveSearchIndex)
	}
	vs.ActiveSearchIndex = 2
	applySelectControlOp(vs, "next", 0)
	if vs.ActiveSearchIndex != 0 {
		t.Fatalf("next wrap: got %d want 0", vs.ActiveSearchIndex)
	}
	applySelectControlOp(vs, "back", 0)
	if vs.ActiveSearchIndex != 2 {
		t.Fatalf("back wrap from 0: got %d want 2", vs.ActiveSearchIndex)
	}
}

func TestApplySelectControlOp_singleItemWrap(t *testing.T) {
	vs := &session.VoiceSession{
		SearchResults:     []session.SearchHit{{Path: "only"}},
		ActiveSearchIndex: 0,
	}
	applySelectControlOp(vs, "next", 0)
	if vs.ActiveSearchIndex != 0 {
		t.Fatalf("single next: got %d want 0", vs.ActiveSearchIndex)
	}
	applySelectControlOp(vs, "back", 0)
	if vs.ActiveSearchIndex != 0 {
		t.Fatalf("single back: got %d want 0", vs.ActiveSearchIndex)
	}
}
