package pipeline

import (
	"strings"

	globalflow "vocoding.net/vocode/v2/apps/core/internal/flows/global"
	workspaceselectflow "vocoding.net/vocode/v2/apps/core/internal/flows/workspaceselect"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/outcome"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/run"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func runWorkspaceSelectPhase(
	e *run.Env,
	key string,
	params protocol.VoiceTranscriptParams,
	vs *session.VoiceSession,
	text string,
	pre preOpts,
) (protocol.VoiceTranscriptCompletion, bool, string) {
	route, searchQuery, searchSymbolKind, ok := resolveWorkspaceSelectRoute(e, params, text, pre)
	if !ok {
		persist(e, key, *vs)
		c := protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "Voice routing failed for workspace search — try rephrasing or say next, previous, or pick a result number.",
			UiDisposition: "hidden",
		}
		if s := globalflow.WorkspaceSearchStateFromSession(vs); s != nil {
			c.Search = s
			c.UiDisposition = "browse"
		}
		return c, true, ""
	}
	execRes, failure := workspaceselectflow.DispatchRoute(selectionDeps(e), params, vs, text, route, searchQuery, searchSymbolKind)
	if strings.TrimSpace(failure) != "" {
		persist(e, key, *vs)
		return protocol.VoiceTranscriptCompletion{
			Success: false,
			Summary: failure,
		}, true, failure
	}
	if !execRes.Success {
		persist(e, key, *vs)
		return execRes, true, transcriptFailureReason(execRes)
	}
	outcome.Apply(vs, params, execRes)
	persist(e, key, *vs)
	return execRes, true, ""
}
