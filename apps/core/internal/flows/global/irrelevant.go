package globalflow

import (
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// HandleIrrelevant handles the global "irrelevant" route.
func HandleIrrelevant(vs *session.VoiceSession, host flows.ID) (protocol.VoiceTranscriptCompletion, string) {
	c := protocol.VoiceTranscriptCompletion{
		Success:       true,
		UiDisposition: "skipped",
	}
	if host == flows.SelectFile && strings.TrimSpace(vs.FileSelectionFocus) != "" {
		c.FileSelection = &protocol.VoiceTranscriptFileSelectionState{FocusPath: vs.FileSelectionFocus}
	}
	return c, ""
}
