package pipeline

import (
	"strings"

	flowhelpers "vocoding.net/vocode/v2/apps/core/internal/flows/helpers"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/clarify"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/controlrequest"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/idle"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/run"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Execute runs the voice transcript accept pipeline: session load, control requests, clarify, then phase dispatch.
func Execute(e *run.Env, params protocol.VoiceTranscriptParams, opts *Opts) (protocol.VoiceTranscriptCompletion, bool, string) {
	if e == nil || e.Ephemeral == nil || e.Sessions == nil {
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "core transcript env not initialized",
			UiDisposition: "hidden",
		}, true, ""
	}

	key := strings.TrimSpace(params.ContextSessionId)
	var vs session.VoiceSession
	if key == "" {
		vs = session.CloneVoiceSession(*e.Ephemeral)
	} else {
		vs = e.Sessions.Get(key, idle.SessionResetDuration(params))
	}

	if cr := strings.TrimSpace(params.ControlRequest); cr != "" {
		out, ok := controlrequest.Handle(params, key, &vs, cr)
		persist(e, key, vs)
		return out, ok, ""
	}

	text := strings.TrimSpace(params.Text)
	if text == "" {
		return protocol.VoiceTranscriptCompletion{}, false, ""
	}

	if handled, res := tryClarifyTurn(e, key, &vs, text); handled {
		return res, true, ""
	}

	if vs.BasePhase == "" {
		vs.BasePhase = session.BasePhaseMain
	}

	pre := preFromOpts(opts)

	switch vs.BasePhase {
	case session.BasePhaseSelection:
		return runWorkspaceSelectPhase(e, key, params, &vs, text, pre)
	case session.BasePhaseFileSelection:
		return runFileSelectionPhase(e, key, params, &vs, text, pre)
	default:
		return runMainPhase(e, key, params, &vs, text, pre)
	}
}

func tryClarifyTurn(
	e *run.Env,
	key string,
	vs *session.VoiceSession,
	text string,
) (handled bool, res protocol.VoiceTranscriptCompletion) {
	if vs.Clarify == nil {
		return false, res
	}

	if flowhelpers.IsExitPhrase(text) {
		vs.Clarify = nil
		persist(e, key, *vs)
		return true, protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "Clarification cancelled",
			UiDisposition: "hidden",
		}
	}

	ov := *vs.Clarify
	cc := clarify.BuildClarificationContext(ov, text)
	_ = cc
	clarify.ApplyResolvedBasePhaseTransition(vs)
	vs.Clarify = nil
	persist(e, key, *vs)
	return true, protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       "Clarification resolved",
		UiDisposition: "hidden",
	}
}
