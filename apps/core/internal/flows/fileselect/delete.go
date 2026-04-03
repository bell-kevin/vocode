package fileselectflow

import (
	"os"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// HandleDelete handles the "delete" route.
func HandleDelete(deps *SelectFileDeps, params protocol.VoiceTranscriptParams, vs *session.VoiceSession, text string) (protocol.VoiceTranscriptCompletion, string) {
	_ = text
	return fileSelectionDeletePath(deps, params, vs, strings.TrimSpace(vs.FileSelectionFocus))
}

func fileSelectionDeletePath(deps *SelectFileDeps, params protocol.VoiceTranscriptParams, vs *session.VoiceSession, path string) (protocol.VoiceTranscriptCompletion, string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return protocol.VoiceTranscriptCompletion{Success: false}, "delete: no file path"
	}
	if deps.HostApply == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "host apply client not configured"
	}
	st, err := os.Stat(path)
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "delete: " + err.Error()
	}
	if st.IsDir() {
		return protocol.VoiceTranscriptCompletion{Success: false}, "delete folder not supported; delete a file"
	}
	dirs := []protocol.VoiceTranscriptDirective{
		{Kind: "delete_file", DeleteFileDirective: &protocol.DeleteFileDirective{Path: path}},
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
	vs.FileSelectionPaths = nil
	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       "delete file",
		UiDisposition: "shown",
	}, ""
}
