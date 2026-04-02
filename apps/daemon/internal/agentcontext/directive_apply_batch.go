package agentcontext

import (
	"fmt"
	"strings"

	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

const (
	ApplyItemStatusOK      = "ok"
	ApplyItemStatusFailed  = "failed"
	ApplyItemStatusSkipped = "skipped"
)

// DirectiveApplyBatch is one batch of directives the daemon returned and the host must apply.
// Wire: [protocol.HostApplyParams.applyBatchId] is used to correlate the host's response
// back to this batch during duplex apply inside the same voice.transcript RPC.
type DirectiveApplyBatch struct {
	ID            string
	NumDirectives int
}

// ConsumeHostApplyReport validates the host report against this batch.
func (b *DirectiveApplyBatch) ConsumeHostApplyReport(
	reportBatchID string,
	items []protocol.VoiceTranscriptDirectiveApplyItem,
) error {
	if b == nil {
		return fmt.Errorf("directive apply batch: nil batch")
	}
	if strings.TrimSpace(reportBatchID) != b.ID {
		return fmt.Errorf("directive apply batch: applyBatchId mismatch")
	}
	if len(items) != b.NumDirectives {
		return fmt.Errorf("directive apply batch: apply items length mismatch")
	}
	for _, it := range items {
		switch strings.TrimSpace(it.Status) {
		case ApplyItemStatusOK, ApplyItemStatusSkipped, ApplyItemStatusFailed:
			// valid
		default:
			return fmt.Errorf("directive apply batch: unknown status %q", it.Status)
		}
	}
	return nil
}
