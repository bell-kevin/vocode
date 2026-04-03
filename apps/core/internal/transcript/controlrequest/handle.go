package controlrequest

import (
	"vocoding.net/vocode/v2/apps/core/internal/flows/helpers"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func Handle(
	params protocol.VoiceTranscriptParams,
	key string,
	vs *session.VoiceSession,
	cr string,
) (protocol.VoiceTranscriptCompletion, bool) {
	_ = params
	_ = key

	switch cr {
	case "cancel_clarify":
		if vs.Clarify != nil {
			vs.Clarify = nil
		}
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "Clarification cancelled",
			UiDisposition: "hidden",
		}, true

	case "cancel_selection":
		vs.Clarify = nil
		helpers.CloseSelectionPhase(vs, false)
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "Search session closed",
			Search:        &protocol.VoiceTranscriptWorkspaceSearchState{Closed: true},
			UiDisposition: "hidden",
		}, true

	case "cancel_file_selection":
		helpers.CloseFileSelectionPhase(vs, false)
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "File selection closed",
			FileSelection: &protocol.VoiceTranscriptFileSearchState{Closed: true},
			UiDisposition: "hidden",
		}, true

	default:
		return protocol.VoiceTranscriptCompletion{}, false
	}
}
