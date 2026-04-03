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

func TestClassifyFlow_stubWorkspaceSelect_createOnLine(t *testing.T) {
	t.Parallel()
	fr := NewFlowRouter(nil)
	p := protocol.VoiceTranscriptParams{
		ActiveFile: "/x/test.go",
		ActiveSelection: &struct {
			StartLine int64 `json:"startLine"`
			StartChar int64 `json:"startChar"`
			EndLine   int64 `json:"endLine"`
			EndChar   int64 `json:"endChar"`
		}{StartLine: 0, StartChar: 0, EndLine: 10, EndChar: 1},
	}
	ctx := ContextForClassification(flows.WorkspaceSelect, "insert a package comment on line 1", p)
	res, err := fr.ClassifyFlow(t.Context(), ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.Route != "create" {
		t.Fatalf("got route %q want create", res.Route)
	}
}

func TestClassifyFlow_stubWorkspaceSelect_createAtEnd(t *testing.T) {
	t.Parallel()
	fr := NewFlowRouter(nil)
	ctx := ContextForClassification(flows.WorkspaceSelect, "add a helper function at the end of the file", protocol.VoiceTranscriptParams{ActiveFile: "/x/a.go"})
	res, err := fr.ClassifyFlow(t.Context(), ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.Route != "create" {
		t.Fatalf("got route %q want create", res.Route)
	}
}

func TestClassifyFlow_stubRoot_createAtEnd(t *testing.T) {
	t.Parallel()
	fr := NewFlowRouter(nil)
	ctx := ContextForClassification(flows.Root, "append a newline at the end of the file", protocol.VoiceTranscriptParams{ActiveFile: "/x/a.go"})
	res, err := fr.ClassifyFlow(t.Context(), ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.Flow != flows.Root || res.Route != "create" {
		t.Fatalf("got flow=%q route %q want root/create", res.Flow, res.Route)
	}
}

func TestClassifyFlow_stubSelectFile_editorCreate(t *testing.T) {
	t.Parallel()
	fr := NewFlowRouter(nil)
	ctx := ContextForClassification(flows.SelectFile, "add a comment at the end of the file", protocol.VoiceTranscriptParams{ActiveFile: "/x/a.go"})
	res, err := fr.ClassifyFlow(t.Context(), ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.Flow != flows.SelectFile || res.Route != "create" {
		t.Fatalf("got flow=%q route %q want select_file/create", res.Flow, res.Route)
	}
}

func TestClassifyFlow_stubSelectFile_createEntry(t *testing.T) {
	t.Parallel()
	fr := NewFlowRouter(nil)
	ctx := ContextForClassification(flows.SelectFile, "new file util.go", protocol.VoiceTranscriptParams{})
	res, err := fr.ClassifyFlow(t.Context(), ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.Flow != flows.SelectFile || res.Route != "create_entry" {
		t.Fatalf("got flow=%q route %q want select_file/create_entry", res.Flow, res.Route)
	}
}
