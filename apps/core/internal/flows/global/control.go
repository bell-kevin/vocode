package globalflow

import (
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
	"vocoding.net/vocode/v2/apps/core/internal/flows/helpers"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/searchapply"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// HandleControl handles the global "control" route (exit / leave) for the given host flow.
func HandleControl(host flows.ID, _ protocol.VoiceTranscriptParams, vs *session.VoiceSession, text string) (protocol.VoiceTranscriptCompletion, string) {
	if !helpers.IsExitPhrase(text) {
		return nonExitControlIrrelevant(host, vs), ""
	}

	switch host {
	case flows.WorkspaceSelect:
		helpers.CloseSelectionPhase(vs, true)
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "Search session closed",
			Search:        &protocol.VoiceTranscriptWorkspaceSearchState{Closed: true},
			UiDisposition: "hidden",
		}, ""

	case flows.SelectFile:
		helpers.CloseFileSelectionPhase(vs, true)
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "File selection closed",
			FileSelection: &protocol.VoiceTranscriptFileSearchState{Closed: true},
			UiDisposition: "hidden",
		}, ""

	default: // flows.Root
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "core transcript (stub)",
			UiDisposition: "hidden",
		}, ""
	}
}

func nonExitControlIrrelevant(host flows.ID, vs *session.VoiceSession) protocol.VoiceTranscriptCompletion {
	c := protocol.VoiceTranscriptCompletion{
		Success:       true,
		UiDisposition: "skipped",
	}
	if host == flows.SelectFile && strings.TrimSpace(vs.FileSelectionFocus) != "" {
		if len(vs.FileSelectionPaths) > 0 {
			c.FileSelection = searchapply.FileSearchStateFromPathsWithDir(vs.FileSelectionPaths, vs.FileSelectionIsDir, vs.FileSelectionIndex)
		} else {
			c.FileSelection = searchapply.FileSearchStateFromSinglePath(vs.FileSelectionFocus)
		}
	}
	return c
}
