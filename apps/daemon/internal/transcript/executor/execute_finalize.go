package executor

import (
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func finalizeExecute(st *agentLoopState) (protocol.VoiceTranscriptResult, agentcontext.Gathered, *agentcontext.DirectiveApplyBatch, bool) {
	result := protocol.VoiceTranscriptResult{
		Success:    true,
		Directives: st.directives,
		Summary:    st.transcriptSummary,
	}
	var pending *agentcontext.DirectiveApplyBatch
	if len(st.directives) > 0 {
		bid, err := newDirectiveApplyBatchID()
		if err != nil {
			return protocol.VoiceTranscriptResult{Success: false}, st.gathered, nil, true
		}
		result.ApplyBatchId = bid
		pending = &agentcontext.DirectiveApplyBatch{
			ID:            bid,
			SourceIntents: append([]intents.Intent(nil), st.batchSourceIntents...),
		}
	}
	if err := result.Validate(); err != nil {
		return protocol.VoiceTranscriptResult{Success: false}, st.gathered, nil, true
	}
	return result, st.gathered, pending, true
}
