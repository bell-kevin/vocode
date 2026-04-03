package selectfileflow

import (
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/flows/helpers"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// HandleSelectFileControl handles the select-file flow "select_file_control" route only (path-list navigation via [selection.ParseNav]).
func HandleSelectFileControl(_ *SelectFileDeps, _ protocol.VoiceTranscriptParams, vs *session.VoiceSession, text string) (protocol.VoiceTranscriptCompletion, string) {
	op, pick, ok := listNavOp(text)
	if !ok {
		c := protocol.VoiceTranscriptCompletion{
			Success:       true,
			UiDisposition: "skipped",
		}
		if strings.TrimSpace(vs.FileSelectionFocus) != "" {
			c.FileSelection = &protocol.VoiceTranscriptFileSelectionState{FocusPath: vs.FileSelectionFocus}
		}
		return c, ""
	}
	applyFileSelectionControlOp(vs, op, pick)
	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       "file focus updated",
		UiDisposition: "hidden",
		FileSelection: &protocol.VoiceTranscriptFileSelectionState{
			FocusPath:      vs.FileSelectionFocus,
			NavigatingList: true,
		},
	}, ""
}

func applyFileSelectionControlOp(vs *session.VoiceSession, op string, pick1Based int) {
	op = strings.ToLower(strings.TrimSpace(op))
	n := len(vs.FileSelectionPaths)
	switch op {
	case "next":
		if n > 0 && vs.FileSelectionIndex < n-1 {
			vs.FileSelectionIndex++
		}
	case "back":
		if vs.FileSelectionIndex > 0 {
			vs.FileSelectionIndex--
		}
	case "pick":
		if pick1Based >= 1 && pick1Based <= n {
			vs.FileSelectionIndex = pick1Based - 1
		}
	}
	if n > 0 && vs.FileSelectionIndex >= 0 && vs.FileSelectionIndex < n {
		vs.FileSelectionFocus = vs.FileSelectionPaths[vs.FileSelectionIndex]
	}
}

func listNavOp(text string) (op string, pick1Based int, ok bool) {
	k, ord, ok := helpers.ParseNav(text)
	if !ok {
		return "", 0, false
	}
	switch k {
	case "next":
		return "next", 0, true
	case "back":
		return "back", 0, true
	case "pick":
		return "pick", ord, true
	default:
		return "", 0, false
	}
}
