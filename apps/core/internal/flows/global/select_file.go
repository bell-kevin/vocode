package globalflow

import (
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/searchapply"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// TryHandleSelectFileSearch runs file-path search using the classifier-provided rg query.
func TryHandleSelectFileSearch(
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
	if res, hit, reason := deps.Search.FileSearchFromQuery(params, q, vs); hit {
		if strings.TrimSpace(reason) != "" {
			return protocol.VoiceTranscriptCompletion{Success: false}, reason, true
		}
		return res, "", true
	}
	return protocol.VoiceTranscriptCompletion{}, "", false
}

// HandleSelectFile handles the global "select_file" route for a sub-flow host or root.
func HandleSelectFile(
	deps *RouteDeps,
	params protocol.VoiceTranscriptParams,
	vs *session.VoiceSession,
	host flows.ID,
	searchQuery string,
) (protocol.VoiceTranscriptCompletion, string) {
	if res, fail, ok := TryHandleSelectFileSearch(deps, params, vs, searchQuery); ok {
		return res, fail
	}
	return selectFileSearchMiss(host, vs), ""
}

func selectFileSearchMiss(host flows.ID, vs *session.VoiceSession) protocol.VoiceTranscriptCompletion {
	switch host {
	case flows.Root:
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			UiDisposition: "skipped",
		}
	case flows.SelectFile:
		c := protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "file focus updated",
			UiDisposition: "hidden",
		}
		if len(vs.FileSelectionPaths) > 0 {
			c.FileSelection = searchapply.FileSearchStateFromPathsWithDir(vs.FileSelectionPaths, vs.FileSelectionIsDir, vs.FileSelectionIndex)
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
