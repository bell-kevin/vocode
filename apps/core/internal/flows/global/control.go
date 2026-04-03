package globalflow

import (
	"regexp"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

var exitRe = regexp.MustCompile(`\b(cancel|exit|close|stop|done|quit|leave|end|abort)\b`)

// IsExitPhrase is true when the utterance clearly requests leaving the flow.
func IsExitPhrase(transcript string) bool {
	t := strings.TrimSpace(strings.ToLower(transcript))
	return exitRe.MatchString(t)
}

// HandleControl handles the global "control" route (exit / leave) for the given host flow.
func HandleControl(host flows.ID, _ protocol.VoiceTranscriptParams, vs *session.VoiceSession, text string) (protocol.VoiceTranscriptCompletion, string) {
	if !IsExitPhrase(text) {
		return nonExitControlIrrelevant(host, vs), ""
	}

	switch host {
	case flows.Select:
		closeSelectionPhase(vs)
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "Search session closed",
			Search:        &protocol.VoiceTranscriptSearchState{Closed: true},
			UiDisposition: "hidden",
		}, ""

	case flows.SelectFile:
		closeFileSelectionPhase(vs)
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "File selection closed",
			UiDisposition: "hidden",
		}, ""

	default: // flows.Root
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "core transcript (stub)",
			UiDisposition: "hidden",
		}, ""
	}
}

func nonExitControlIrrelevant(host flows.ID, vs *session.VoiceSession) protocol.VoiceTranscriptCompletion {
	c := protocol.VoiceTranscriptCompletion{
		Success:       true,
		UiDisposition: "skipped",
	}
	if host == flows.SelectFile && strings.TrimSpace(vs.FileSelectionFocus) != "" {
		c.FileSelection = &protocol.VoiceTranscriptFileSelectionState{FocusPath: vs.FileSelectionFocus}
	}
	return c
}

func closeSelectionPhase(vs *session.VoiceSession) {
	vs.SearchResults = nil
	vs.ActiveSearchIndex = 0
	vs.PendingDirectiveApply = nil
	vs.BasePhase = session.BasePhaseMain
}

func closeFileSelectionPhase(vs *session.VoiceSession) {
	vs.FileSelectionPaths = nil
	vs.FileSelectionIndex = 0
	vs.FileSelectionFocus = ""
	vs.BasePhase = session.BasePhaseMain
}
