package agentcontext

import "vocoding.net/vocode/v2/apps/daemon/internal/intents"

// CloneVoiceSession returns a deep enough copy for storing a [VoiceSession] between RPCs.
func CloneVoiceSession(v VoiceSession) VoiceSession {
	out := VoiceSession{
		Gathered:              v.Gathered,
		NextApplyBatchOrdinal: v.NextApplyBatchOrdinal,
	}
	if v.PendingDirectiveApply != nil {
		p := *v.PendingDirectiveApply
		p.SourceIntents = append([]intents.Intent(nil), v.PendingDirectiveApply.SourceIntents...)
		out.PendingDirectiveApply = &p
	}
	if len(v.IntentApplyHistory) > 0 {
		out.IntentApplyHistory = append([]IntentApplyRecord(nil), v.IntentApplyHistory...)
	}
	return out
}
