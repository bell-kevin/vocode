package agentcontext

import (
	"fmt"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// DirectiveApplyBatch is one batch of directives the daemon returned and the host must apply.
// Wire: [protocol.VoiceTranscriptResult.applyBatchId] matches [protocol.VoiceTranscriptParams.reportApplyBatchId];
// [protocol.VoiceTranscriptParams.lastBatchApply] is parallel to [SourceIntents] and to result.directives.
type DirectiveApplyBatch struct {
	ID            string
	SourceIntents []intents.Intent
}

// ConsumeHostApplyReport validates the host report against this batch and returns succeeded vs
// failed intents for the next agent loop iteration. Caller must clear [VoiceSession.PendingDirectiveApply]
// after a successful return.
func (b *DirectiveApplyBatch) ConsumeHostApplyReport(
	reportBatchID string,
	items []protocol.VoiceTranscriptDirectiveApplyItem,
) ([]intents.Intent, []FailedIntent, error) {
	if b == nil {
		return nil, nil, fmt.Errorf("directive apply batch: nil batch")
	}
	if strings.TrimSpace(reportBatchID) != b.ID {
		return nil, nil, fmt.Errorf("directive apply batch: reportApplyBatchId mismatch")
	}
	if len(items) != len(b.SourceIntents) {
		return nil, nil, fmt.Errorf("directive apply batch: lastBatchApply length mismatch")
	}
	var extSucc []intents.Intent
	var extFail []FailedIntent
	for i, it := range items {
		intent := b.SourceIntents[i]
		if it.Ok {
			extSucc = append(extSucc, intent)
			continue
		}
		msg := strings.TrimSpace(it.Message)
		if msg == "" {
			msg = "extension failed to apply directive"
		}
		extFail = append(extFail, FailedIntent{
			Intent: intent,
			Phase:  PhaseExtension,
			Reason: msg,
		})
	}
	return extSucc, extFail, nil
}
