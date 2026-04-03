package globalflow

import (
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/searchapply"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// HandleIrrelevant handles the global "irrelevant" route.
func HandleIrrelevant(vs *session.VoiceSession, host flows.ID) (protocol.VoiceTranscriptCompletion, string) {
	c := protocol.VoiceTranscriptCompletion{
		Success:       true,
		UiDisposition: "skipped",
	}
	if host == flows.SelectFile {
		if len(vs.FileSelectionPaths) > 0 {
			c.FileSelection = searchapply.FileSearchStateFromPathsWithDir(vs.FileSelectionPaths, vs.FileSelectionIsDir, vs.FileSelectionIndex)
		} else if p := strings.TrimSpace(vs.FileSelectionFocus); p != "" {
			c.FileSelection = searchapply.FileSearchStateFromSinglePath(p)
		}
	}
	return c, ""
}
