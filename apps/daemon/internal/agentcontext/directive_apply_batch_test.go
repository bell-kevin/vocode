package agentcontext

import (
	"testing"

	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func TestConsumeHostApplyReportSplitsStatuses(t *testing.T) {
	t.Parallel()

	b := &DirectiveApplyBatch{
		ID: "batch-1",
		SourceIntents: []intents.Intent{
			intents.ControlDone(),
			intents.ControlDone(),
			intents.ControlDone(),
		},
	}
	items := []protocol.VoiceTranscriptDirectiveApplyItem{
		{Status: ApplyItemStatusOK},
		{Status: ApplyItemStatusFailed, Message: "boom"},
		{Status: ApplyItemStatusSkipped, Message: "not attempted"},
	}
	ok, fail, skip, err := b.ConsumeHostApplyReport("batch-1", items)
	if err != nil {
		t.Fatal(err)
	}
	if len(ok) != 1 || len(fail) != 1 || len(skip) != 1 {
		t.Fatalf("ok=%d fail=%d skip=%d", len(ok), len(fail), len(skip))
	}
	if fail[0].Reason != "boom" {
		t.Fatalf("reason: %q", fail[0].Reason)
	}
}
