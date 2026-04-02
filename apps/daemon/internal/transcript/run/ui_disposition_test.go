package run

import (
	"testing"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func TestInferTranscriptUIDisposition(t *testing.T) {
	t.Parallel()
	if inferTranscriptUIDisposition("completed", false) != "shown" {
		t.Fatal("completed → shown")
	}
	if inferTranscriptUIDisposition("file_selection_control", false) != "hidden" {
		t.Fatal("file_selection_control → hidden")
	}
	if inferTranscriptUIDisposition("irrelevant", true) != "hidden" {
		t.Fatal("irrelevant with active search hits → hidden")
	}
	if inferTranscriptUIDisposition("irrelevant", false) != "skipped" {
		t.Fatal("irrelevant without search → skipped")
	}
}

func TestApplyTranscriptUIDisposition_irrelevantInFileFlow(t *testing.T) {
	t.Parallel()
	res := protocol.VoiceTranscriptCompletion{
		Success:           true,
		TranscriptOutcome: "irrelevant",
		UiDisposition:     "",
	}
	applyTranscriptUIDisposition(&res, agentcontext.FlowKindFileSelection, false)
	if res.UiDisposition != "hidden" {
		t.Fatalf("file_selection irrelevant should be hidden, got %q", res.UiDisposition)
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
