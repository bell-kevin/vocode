package clarify

import "vocoding.net/vocode/v2/apps/core/internal/transcript/session"

func ApplyResolvedBasePhaseTransition(vs *session.VoiceSession) {
	if vs.Clarify == nil {
		return
	}

	switch vs.Clarify.TargetResolution {
	case ClarifyTargetWorkspaceSelect:
		vs.BasePhase = session.BasePhaseSelection
		vs.FileSelectionPaths = nil
		vs.FileSelectionIndex = 0
		vs.FileSelectionFocus = ""
	case ClarifyTargetSelectFile:
		vs.BasePhase = session.BasePhaseFileSelection
		vs.SearchResults = nil
		vs.ActiveSearchIndex = 0
		vs.PendingDirectiveApply = nil
	case ClarifyTargetEdit:
		if vs.BasePhase == session.BasePhaseSelection {
			vs.BasePhase = session.BasePhaseMain
			vs.SearchResults = nil
			vs.ActiveSearchIndex = 0
			vs.PendingDirectiveApply = nil
		}
	default:
	}
}
