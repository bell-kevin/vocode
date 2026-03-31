package executor

import (
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func finalizeExecute(st *agentLoopState) (protocol.VoiceTranscriptCompletion, []protocol.VoiceTranscriptDirective, agentcontext.Gathered, *agentcontext.DirectiveApplyBatch, bool) {
	result := protocol.VoiceTranscriptCompletion{
		Success: true,
		Summary: st.transcriptSummary,
	}
	if strings.TrimSpace(st.transcriptOutcome) == "irrelevant" {
		result.TranscriptOutcome = "irrelevant"
	}
	dirs := append([]protocol.VoiceTranscriptDirective(nil), st.directives...)
	var pending *agentcontext.DirectiveApplyBatch
	if len(dirs) > 0 {
		bid, err := newDirectiveApplyBatchID()
		if err != nil {
			return protocol.VoiceTranscriptCompletion{Success: false}, nil, st.gathered, nil, false
		}
		pending = &agentcontext.DirectiveApplyBatch{
			ID:            bid,
			SourceIntents: append([]intents.Intent(nil), st.batchSourceIntents...),
		}
	}
	if err := result.Validate(); err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, nil, st.gathered, nil, false
	}
	return result, dirs, st.gathered, pending, true
}
