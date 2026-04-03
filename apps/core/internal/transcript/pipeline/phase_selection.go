package pipeline

import (
	"strings"

	selectflow "vocoding.net/vocode/v2/apps/core/internal/flows/select"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/outcome"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/run"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func runSelectionPhase(
	e *run.Env,
	key string,
	params protocol.VoiceTranscriptParams,
	vs *session.VoiceSession,
	text string,
	pre preOpts,
) (protocol.VoiceTranscriptCompletion, bool, string) {
	route, searchQuery, ok := resolveSelectRoute(e, text, pre)
	if !ok {
		persist(e, key, *vs)
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "core transcript (stub)",
			UiDisposition: "hidden",
		}, true, ""
	}
	execRes, failure := selectflow.DispatchRoute(selectionDeps(e), params, vs, text, route, searchQuery)
	if strings.TrimSpace(failure) != "" {
		persist(e, key, *vs)
		return protocol.VoiceTranscriptCompletion{Success: false}, true, failure
	}
	outcome.Apply(vs, params, execRes)
	persist(e, key, *vs)
	return execRes, true, ""
}
