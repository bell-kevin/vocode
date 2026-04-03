package router

import (
	"testing"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func TestClassifyFlow_stubWorkspaceSelect_prefersEditWhenSelectionAndImperative(t *testing.T) {
	t.Parallel()
	fr := NewFlowRouter(nil)
	p := protocol.VoiceTranscriptParams{
		ActiveFile: "/x/test.js",
		ActiveSelection: &struct {
			StartLine int64 `json:"startLine"`
			StartChar int64 `json:"startChar"`
			EndLine   int64 `json:"endLine"`
			EndChar   int64 `json:"endChar"`
		}{StartLine: 0, StartChar: 0, EndLine: 10, EndChar: 1},
	}
	ctx := ContextForClassification(flows.WorkspaceSelect, "make it pass delta time into render", p)
	res, err := fr.ClassifyFlow(t.Context(), ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.Route != "edit" {
		t.Fatalf("got route %q want edit", res.Route)
	}
}

func TestClassifyFlow_stubWorkspaceSelect_renameToBeforeEdit(t *testing.T) {
	t.Parallel()
	fr := NewFlowRouter(nil)
	p := protocol.VoiceTranscriptParams{
		ActiveFile: "/x/test.js",
		ActiveSelection: &struct {
			StartLine int64 `json:"startLine"`
			StartChar int64 `json:"startChar"`
			EndLine   int64 `json:"endLine"`
			EndChar   int64 `json:"endChar"`
		}{StartLine: 0, StartChar: 0, EndLine: 10, EndChar: 1},
	}
	ctx := ContextForClassification(flows.WorkspaceSelect, "rename deltaTime to dt", p)
	res, err := fr.ClassifyFlow(t.Context(), ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.Route != "rename" {
		t.Fatalf("got route %q want rename", res.Route)
	}
}
