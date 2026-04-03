package pipeline

import (
	"strings"

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
	route, searchQuery, ok := resolveWorkspaceSelectRoute(e, text, pre)
	if !ok {
		persist(e, key, *vs)
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "core transcript (stub)",
			UiDisposition: "hidden",
		}, true, ""
	}
	execRes, failure := workspaceselectflow.DispatchRoute(selectionDeps(e), params, vs, text, route, searchQuery)
	if strings.TrimSpace(failure) != "" {
		persist(e, key, *vs)
		return protocol.VoiceTranscriptCompletion{Success: false}, true, failure
	}
	outcome.Apply(vs, params, execRes)
	persist(e, key, *vs)
	return execRes, true, ""
}
