package fileselectflow

import (
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// HandleMove handles the "move" route (stub until executor is ported).
func HandleMove(_ *SelectFileDeps, _ protocol.VoiceTranscriptParams, _ *session.VoiceSession, _ string) (protocol.VoiceTranscriptCompletion, string) {
	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       "core transcript (stub)",
		UiDisposition: "hidden",
	}, ""
}
