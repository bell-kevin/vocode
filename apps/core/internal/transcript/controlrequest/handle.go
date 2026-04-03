package controlrequest

import (
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
		vs.SearchResults = nil
		vs.ActiveSearchIndex = 0
		vs.PendingDirectiveApply = nil
		if vs.BasePhase == session.BasePhaseSelection {
			vs.BasePhase = session.BasePhaseMain
		}
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "Search session closed",
			Search:        &protocol.VoiceTranscriptSearchState{Closed: true},
			UiDisposition: "hidden",
		}, true
	default:
		return protocol.VoiceTranscriptCompletion{}, false
	}
}
