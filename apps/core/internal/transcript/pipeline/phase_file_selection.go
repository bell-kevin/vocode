package pipeline

import (
	"strings"

	global "vocoding.net/vocode/v2/apps/core/internal/flows/global"
	selectfileflow "vocoding.net/vocode/v2/apps/core/internal/flows/selectfile"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/outcome"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/run"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func runFileSelectionPhase(
	e *run.Env,
	key string,
	params protocol.VoiceTranscriptParams,
	vs *session.VoiceSession,
	text string,
	pre preOpts,
) (protocol.VoiceTranscriptCompletion, bool, string) {
	if global.IsExitPhrase(text) {
		vs.FileSelectionPaths = nil
		vs.FileSelectionIndex = 0
		vs.FileSelectionFocus = ""
		vs.BasePhase = session.BasePhaseMain
		persist(e, key, *vs)
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "File selection closed",
			UiDisposition: "hidden",
		}, true, ""
	}

	if len(vs.FileSelectionPaths) == 0 {
		vs.BasePhase = session.BasePhaseMain
		persist(e, key, *vs)
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "No file hits in this session; use find file or your assistant file search to get a path list first.",
			UiDisposition: "shown",
		}, true, ""
	}

	if focused := strings.TrimSpace(params.FocusedWorkspacePath); focused != "" {
		for i, p := range vs.FileSelectionPaths {
			if p == focused {
				vs.FileSelectionIndex = i
				vs.FileSelectionFocus = p
				break
			}
		}
	}

	route, searchQuery, rOK, clsErrMsg := resolveSelectFileRoute(e, text, pre)
	if !rOK {
		persist(e, key, *vs)
		if strings.TrimSpace(clsErrMsg) != "" {
			return protocol.VoiceTranscriptCompletion{}, true, clsErrMsg
		}
		return protocol.VoiceTranscriptCompletion{}, true, "flow classifier: empty route"
	}

	frRes, frFail := selectfileflow.DispatchRoute(selectFileDeps(e), params, vs, text, route, searchQuery)
	if strings.TrimSpace(frFail) != "" {
		persist(e, key, *vs)
		if !frRes.Success {
			return frRes, true, frFail
		}
		return protocol.VoiceTranscriptCompletion{Success: false}, true, frFail
	}
	outcome.Apply(vs, params, frRes)
	persist(e, key, *vs)
	return frRes, true, ""
}
