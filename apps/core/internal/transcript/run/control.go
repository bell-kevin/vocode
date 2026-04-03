package run

import (
	"time"

	"vocoding.net/vocode/v2/apps/core/internal/transcript/clarify"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func handleControlRequest(
	params protocol.VoiceTranscriptParams,
	key string,
	vs *session.VoiceSession,
	cr string,
) (protocol.VoiceTranscriptCompletion, bool) {
	_ = params

	switch cr {
	case "cancel_clarify":
		if vs.Clarify != nil {
			vs.Clarify = nil
		}
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           "Clarification cancelled",
			TranscriptOutcome: "completed",
			UiDisposition:     "hidden",
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
			Success:           true,
			Summary:           "Search session closed",
			TranscriptOutcome: "completed",
			UiDisposition:     "hidden",
		}, true
	default:
		return protocol.VoiceTranscriptCompletion{}, false
	}
}

func resumeFromClarification(vs *session.VoiceSession) {
	if vs.Clarify == nil {
		return
	}

	switch vs.Clarify.TargetResolution {
	case clarify.ClarifyTargetSelect:
		vs.BasePhase = session.BasePhaseSelection
		vs.FileSelectionPaths = nil
		vs.FileSelectionIndex = 0
		vs.FileSelectionFocus = ""
	case clarify.ClarifyTargetSelectFile:
		vs.BasePhase = session.BasePhaseFileSelection
		vs.SearchResults = nil
		vs.ActiveSearchIndex = 0
		vs.PendingDirectiveApply = nil
	case clarify.ClarifyTargetEdit:
		if vs.BasePhase == session.BasePhaseSelection {
			vs.BasePhase = session.BasePhaseMain
			vs.SearchResults = nil
			vs.ActiveSearchIndex = 0
			vs.PendingDirectiveApply = nil
		}
	default:
	}
}

func idleReset(params protocol.VoiceTranscriptParams) time.Duration {
	if params.DaemonConfig == nil || params.DaemonConfig.SessionIdleResetMs == nil {
		return 0
	}
	ms := *params.DaemonConfig.SessionIdleResetMs
	if ms <= 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}
