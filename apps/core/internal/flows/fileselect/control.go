package fileselectflow

import (
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/flows/helpers"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/searchapply"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// HandleSelectFileControl handles the select-file flow "file_select_control" route only (path-list navigation via [selection.ParseNav]).
func HandleSelectFileControl(deps *SelectFileDeps, params protocol.VoiceTranscriptParams, vs *session.VoiceSession, text string) (protocol.VoiceTranscriptCompletion, string) {
	op, pick, ok := listNavOp(text)
	if !ok {
		c := protocol.VoiceTranscriptCompletion{
			Success:       true,
			UiDisposition: "skipped",
		}
		if strings.TrimSpace(vs.FileSelectionFocus) != "" {
			c.FileSelection = searchapply.FileSearchStateFromSinglePath(vs.FileSelectionFocus)
		}
		return c, ""
	}
	applyFileSelectionControlOp(vs, op, pick)

	if deps != nil && deps.HostApply != nil && deps.NewBatchID != nil &&
		len(vs.FileSelectionPaths) > 0 && !vs.FileFocusIsDir() {
		focus := strings.TrimSpace(vs.FileSelectionFocus)
		if focus != "" {
			dirs := searchapply.OpenFirstFileDirectivesForPath(focus)
			batchID := deps.NewBatchID()
			pending := &session.DirectiveApplyBatch{ID: batchID, NumDirectives: len(dirs)}
			vs.PendingDirectiveApply = pending
			hostRes, err := deps.HostApply.ApplyDirectives(protocol.HostApplyParams{
				ApplyBatchId: batchID,
				ActiveFile:   params.ActiveFile,
				Directives:   dirs,
			})
			if err != nil {
				vs.PendingDirectiveApply = nil
				return protocol.VoiceTranscriptCompletion{Success: false}, "host.applyDirectives failed: " + err.Error()
			}
			if err := pending.ConsumeHostApplyReport(batchID, hostRes.Items); err != nil {
				vs.PendingDirectiveApply = nil
				return protocol.VoiceTranscriptCompletion{Success: false}, "host apply failed: " + err.Error()
			}
			vs.PendingDirectiveApply = nil
		}
	}

	c := protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       "file focus updated",
		UiDisposition: "hidden",
	}
	if len(vs.FileSelectionPaths) > 0 {
		c.FileSelection = searchapply.FileSearchStateFromPathsWithDir(
			vs.FileSelectionPaths,
			vs.FileSelectionIsDir,
			vs.FileSelectionIndex,
		)
	} else if strings.TrimSpace(vs.FileSelectionFocus) != "" {
		c.FileSelection = searchapply.FileSearchStateFromSinglePath(vs.FileSelectionFocus)
	}
	return c, ""
}

func applyFileSelectionControlOp(vs *session.VoiceSession, op string, pick1Based int) {
	op = strings.ToLower(strings.TrimSpace(op))
	n := len(vs.FileSelectionPaths)
	switch op {
	case "next":
		if n > 0 {
			if vs.FileSelectionIndex < n-1 {
				vs.FileSelectionIndex++
			} else {
				vs.FileSelectionIndex = 0
			}
		}
	case "back":
		if n > 0 {
			if vs.FileSelectionIndex > 0 {
				vs.FileSelectionIndex--
			} else {
				vs.FileSelectionIndex = n - 1
			}
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
