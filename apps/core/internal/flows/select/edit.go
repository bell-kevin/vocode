package selectflow

import (
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// HandleEdit handles the "edit" route (scoped edit — stub until executor is ported).
func HandleEdit(_ *SelectionDeps, _ protocol.VoiceTranscriptParams, _ *session.VoiceSession, _ string) (protocol.VoiceTranscriptCompletion, string) {
	return protocol.VoiceTranscriptCompletion{
		Success:           true,
		Summary:           "core transcript (stub)",
		TranscriptOutcome: "completed",
		UiDisposition:     "hidden",
	}, ""
}
