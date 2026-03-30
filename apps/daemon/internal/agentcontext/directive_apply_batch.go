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

// ConsumeHostApplyReport validates the host report against this batch and returns succeeded,
// failed (attempted), and skipped (not attempted) intents for the next agent loop iteration.
// Caller must clear [VoiceSession.PendingDirectiveApply] after a successful return.
func (b *DirectiveApplyBatch) ConsumeHostApplyReport(
	reportBatchID string,
	items []protocol.VoiceTranscriptDirectiveApplyItem,
) ([]intents.Intent, []FailedIntent, []intents.Intent, error) {
	if b == nil {
		return nil, nil, nil, fmt.Errorf("directive apply batch: nil batch")
	}
	if strings.TrimSpace(reportBatchID) != b.ID {
		return nil, nil, nil, fmt.Errorf("directive apply batch: reportApplyBatchId mismatch")
	}
	if len(items) != len(b.SourceIntents) {
		return nil, nil, nil, fmt.Errorf("directive apply batch: lastBatchApply length mismatch")
	}
	var extSucc []intents.Intent
	var extFail []FailedIntent
	var extSkipped []intents.Intent
	for i, it := range items {
		intent := b.SourceIntents[i]
		switch strings.TrimSpace(it.Status) {
		case ApplyItemStatusOK:
			extSucc = append(extSucc, intent)
		case ApplyItemStatusSkipped:
			extSkipped = append(extSkipped, intent)
		case ApplyItemStatusFailed:
			msg := strings.TrimSpace(it.Message)
			if msg == "" {
				msg = "extension failed to apply directive"
			}
			extFail = append(extFail, FailedIntent{
				Intent: intent,
				Phase:  PhaseExtension,
				Reason: msg,
			})
		default:
			return nil, nil, nil, fmt.Errorf("directive apply batch: unknown status %q", it.Status)
		}
	}
	return extSucc, extFail, extSkipped, nil
}
