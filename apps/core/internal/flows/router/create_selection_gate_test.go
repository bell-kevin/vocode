package router

import (
	"testing"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func TestRejectCreateWhenEditorSelection_blocksWhenNonempty(t *testing.T) {
	t.Parallel()
	p := protocol.VoiceTranscriptParams{
		ActiveSelection: &struct {
			StartLine int64 `json:"startLine"`
			StartChar int64 `json:"startChar"`
			EndLine   int64 `json:"endLine"`
			EndChar   int64 `json:"endChar"`
		}{StartLine: 0, StartChar: 0, EndLine: 0, EndChar: 1},
	}
	c, msg, ok := RejectCreateWhenEditorSelection(p)
	if !ok {
		t.Fatal("expected blocked")
	}
	if c.Success || msg == "" || c.Summary == "" {
		t.Fatalf("completion=%+v msg=%q", c, msg)
	}
}

func TestRejectCreateWhenEditorSelection_allowsCaret(t *testing.T) {
	t.Parallel()
	p := protocol.VoiceTranscriptParams{
		ActiveSelection: &struct {
			StartLine int64 `json:"startLine"`
			StartChar int64 `json:"startChar"`
			EndLine   int64 `json:"endLine"`
			EndChar   int64 `json:"endChar"`
		}{StartLine: 2, StartChar: 5, EndLine: 2, EndChar: 5},
	}
	_, _, ok := RejectCreateWhenEditorSelection(p)
	if ok {
		t.Fatal("expected not blocked for empty selection")
	}
}
