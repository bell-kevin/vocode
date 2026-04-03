package workspaceselectflow

import (
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// HandleDelete handles the "delete" route (delete selection — stub until executor is ported).
func HandleDelete(_ *SelectionDeps, _ protocol.VoiceTranscriptParams, _ *session.VoiceSession, _ string) (protocol.VoiceTranscriptCompletion, string) {
	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       "core transcript (stub)",
		UiDisposition: "hidden",
	}, ""
}
