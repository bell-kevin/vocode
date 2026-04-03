package globalflow

import (
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/searchapply"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// TryHandleWorkspaceSelectSearch runs workspace text search using the classifier-provided rg query.
// ok is true when SearchFromQuery ran as a hit (including error reasons in fail).
// ok is false when the query was empty or SearchFromQuery did not apply (e.g. root may fall through).
func TryHandleWorkspaceSelectSearch(
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

// HandleWorkspaceSelect handles the global "workspace_select" route for a sub-flow host (not root fallthrough).
func HandleWorkspaceSelect(
	deps *RouteDeps,
	params protocol.VoiceTranscriptParams,
	vs *session.VoiceSession,
	host flows.ID,
	searchQuery string,
) (protocol.VoiceTranscriptCompletion, string) {
	if res, fail, ok := TryHandleWorkspaceSelectSearch(deps, params, vs, searchQuery); ok {
		return res, fail
	}
	return workspaceSelectSearchMiss(host, vs), ""
}

func workspaceSelectSearchMiss(host flows.ID, vs *session.VoiceSession) protocol.VoiceTranscriptCompletion {
	switch host {
	case flows.SelectFile:
		c := protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "file focus updated",
			UiDisposition: "hidden",
		}
		if len(vs.FileSelectionPaths) > 0 {
			c.FileSelection = searchapply.FileSearchStateFromPaths(vs.FileSelectionPaths, vs.FileSelectionIndex)
		} else if strings.TrimSpace(vs.FileSelectionFocus) != "" {
			c.FileSelection = searchapply.FileSearchStateFromSinglePath(vs.FileSelectionFocus)
		}
		return c
	default: // flows.WorkspaceSelect
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "core transcript (stub)",
			UiDisposition: "hidden",
		}
	}
}
