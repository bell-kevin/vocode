package selectfileflow

import (
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// HandleOpen handles the "open" route.
func HandleOpen(deps *SelectFileDeps, params protocol.VoiceTranscriptParams, vs *session.VoiceSession, text string) (protocol.VoiceTranscriptCompletion, string) {
	_ = text
	return fileSelectionOpenPath(deps, params, vs, strings.TrimSpace(vs.FileSelectionFocus))
}

func fileSelectionOpenPath(deps *SelectFileDeps, params protocol.VoiceTranscriptParams, vs *session.VoiceSession, path string) (protocol.VoiceTranscriptCompletion, string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return protocol.VoiceTranscriptCompletion{Success: false}, "open: no file path"
	}
	if deps.HostApply == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "daemon has directives but no host apply client is configured"
	}
	dirs := []protocol.VoiceTranscriptDirective{
		{
			Kind: "navigate",
			NavigationDirective: &protocol.NavigationDirective{
				Kind: "success",
				Action: &protocol.NavigationAction{
					Kind: "open_file",
					OpenFile: &struct {
						Path string `json:"path"`
					}{Path: path},
				},
			},
		},
	}
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
	vs.FileSelectionFocus = path
	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       "open file",
		UiDisposition: "hidden",
		FileSelection: &protocol.VoiceTranscriptFileSelectionState{
			FocusPath:      path,
			NavigatingList: true,
		},
	}, ""
}
