package run

import (
	"testing"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func TestInferTranscriptUIDisposition(t *testing.T) {
	t.Parallel()
	if inferTranscriptUIDisposition(&protocol.VoiceTranscriptCompletion{}) != "shown" {
		t.Fatal("minimal success completion → shown")
	}
	if inferTranscriptUIDisposition(&protocol.VoiceTranscriptCompletion{
		FileSelection: &protocol.VoiceTranscriptFileSearchState{
			Results:     []protocol.VoiceTranscriptFileListHit{{Path: "/x", Preview: "x"}},
			ActiveIndex: ptrInt64(0),
		},
	}) != "hidden" {
		t.Fatal("fileSelection navigation → hidden")
	}
	if inferTranscriptUIDisposition(&protocol.VoiceTranscriptCompletion{
		Search: &protocol.VoiceTranscriptWorkspaceSearchState{
			Results: []protocol.VoiceTranscriptSearchHit{{Path: "p", Line: 0, Character: 0, Preview: "x"}},
		},
	}) != "hidden" {
		t.Fatal("search hits → hidden")
	}
}

func TestApplyTranscriptUIDisposition_skippedInFileFlow(t *testing.T) {
	t.Parallel()
	res := protocol.VoiceTranscriptCompletion{
		Success:       true,
		UiDisposition: "skipped",
	}
	applyTranscriptUIDisposition(&res, agentcontext.FlowKindFileSelection, false)
	if res.UiDisposition != "hidden" {
		t.Fatalf("skipped in file flow should become hidden, got %q", res.UiDisposition)
	}
}

func TestApplyTranscriptUIDisposition_failureEmptyDisposition(t *testing.T) {
	t.Parallel()
	res := protocol.VoiceTranscriptCompletion{Success: false}
	applyTranscriptUIDisposition(&res, agentcontext.FlowKindMain, false)
	if res.UiDisposition != "hidden" {
		t.Fatalf("failed completion → hidden, got %q", res.UiDisposition)
	}
}
