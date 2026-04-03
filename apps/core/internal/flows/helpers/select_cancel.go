package helpers

import (
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
)

// CloseSelectionPhase clears workspace search hit state. If voiceExit is true (spoken cancel
// from the select flow), BasePhase is set to main. If false (host cancel_selection), BasePhase
// moves to main only when currently in the selection phase so file-selection mode is preserved.
func CloseSelectionPhase(vs *session.VoiceSession, voiceExit bool) {
	if vs == nil {
		return
	}
	vs.SearchResults = nil
	vs.ActiveSearchIndex = 0
	vs.PendingDirectiveApply = nil
	if voiceExit {
		vs.BasePhase = session.BasePhaseMain
		return
	}
	if vs.BasePhase == session.BasePhaseSelection {
		vs.BasePhase = session.BasePhaseMain
	}
}

// CloseFileSelectionPhase clears file path list state. If voiceExit is true, BasePhase is set
// to main. If false (host cancel_file_selection), BasePhase moves to main only when in file selection.
func CloseFileSelectionPhase(vs *session.VoiceSession, voiceExit bool) {
	if vs == nil {
		return
	}
	vs.FileSelectionPaths = nil
	vs.FileSelectionIndex = 0
	vs.FileSelectionFocus = ""
	if voiceExit {
		vs.BasePhase = session.BasePhaseMain
		return
	}
	if vs.BasePhase == session.BasePhaseFileSelection {
		vs.BasePhase = session.BasePhaseMain
	}
}
