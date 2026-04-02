package session

import (
	"testing"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func TestConsumeHostApplyReport_validatesCountAndStatuses(t *testing.T) {
	b := DirectiveApplyBatch{ID: "b1", NumDirectives: 2}
	items := []protocol.VoiceTranscriptDirectiveApplyItem{
		{Status: ApplyItemStatusOK},
		{Status: ApplyItemStatusFailed, Message: "nope"},
	}

	// Host apply reports should fail when any item failed.
	if err := b.ConsumeHostApplyReport("b1", items); err == nil {
		t.Fatalf("expected error when item failed")
	}
}

func TestConsumeHostApplyReport_rejectsBatchIdMismatch(t *testing.T) {
	b := DirectiveApplyBatch{ID: "b1", NumDirectives: 1}
	items := []protocol.VoiceTranscriptDirectiveApplyItem{{Status: ApplyItemStatusOK}}
	if err := b.ConsumeHostApplyReport("wrong", items); err == nil {
		t.Fatalf("expected error on batch id mismatch")
	}
}

func TestConsumeHostApplyReport_rejectsItemCountMismatch(t *testing.T) {
	b := DirectiveApplyBatch{ID: "b1", NumDirectives: 2}
	items := []protocol.VoiceTranscriptDirectiveApplyItem{{Status: ApplyItemStatusOK}}
	if err := b.ConsumeHostApplyReport("b1", items); err == nil {
		t.Fatalf("expected error on item count mismatch")
	}
}
