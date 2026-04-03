package globalflow

import (
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// TryHandleSelectSearch runs workspace text search using the classifier-provided rg query.
// ok is true when SearchFromQuery ran as a hit (including error reasons in fail).
// ok is false when the query was empty or SearchFromQuery did not apply (e.g. root may fall through).
func TryHandleSelectSearch(
	deps *RouteDeps,
	params protocol.VoiceTranscriptParams,
	vs *session.VoiceSession,
	searchQuery string,
) (protocol.VoiceTranscriptCompletion, string, bool) {
	q := strings.TrimSpace(searchQuery)
	if q == "" {
		return protocol.VoiceTranscriptCompletion{}, "", false
	}
	if deps == nil || deps.Search == nil {
		return protocol.VoiceTranscriptCompletion{}, "", false
	}
	if res, hit, reason := deps.Search.SearchFromQuery(params, q, vs); hit {
		if strings.TrimSpace(reason) != "" {
			return protocol.VoiceTranscriptCompletion{Success: false}, reason, true
		}
		return res, "", true
	}
	return protocol.VoiceTranscriptCompletion{}, "", false
}

// HandleSelect handles the global "select" route for a sub-flow host (not root fallthrough).
func HandleSelect(
	deps *RouteDeps,
	params protocol.VoiceTranscriptParams,
	vs *session.VoiceSession,
	host flows.ID,
	searchQuery string,
) (protocol.VoiceTranscriptCompletion, string) {
	if res, fail, ok := TryHandleSelectSearch(deps, params, vs, searchQuery); ok {
		return res, fail
	}
	return selectSearchMiss(host, vs), ""
}

func selectSearchMiss(host flows.ID, vs *session.VoiceSession) protocol.VoiceTranscriptCompletion {
	switch host {
	case flows.SelectFile:
		return protocol.VoiceTranscriptCompletion{
			Success:                true,
			Summary:                "file focus updated",
			TranscriptOutcome:      "file_selection_control",
			UiDisposition:          "hidden",
			FileSelectionFocusPath: vs.FileSelectionFocus,
		}
	default: // flows.Select
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           "core transcript (stub)",
			TranscriptOutcome: "completed",
			UiDisposition:     "hidden",
		}
	}
}
