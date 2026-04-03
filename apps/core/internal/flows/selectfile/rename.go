package selectfileflow

import (
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// HandleRename handles the "rename" route (stub until executor is ported).
func HandleRename(_ *SelectFileDeps, _ protocol.VoiceTranscriptParams, _ *session.VoiceSession, _ string) (protocol.VoiceTranscriptCompletion, string) {
	return protocol.VoiceTranscriptCompletion{
		Success:           true,
		Summary:           "core transcript (stub)",
		TranscriptOutcome: "completed",
		UiDisposition:     "hidden",
	}, ""
}
