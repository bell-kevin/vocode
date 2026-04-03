package workspaceselectflow

import (
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// HandleRename runs LSP symbol rename (all references) at the active workspace search hit.
// The host applies VoiceTranscriptDirective kind "rename" (vscode.executeDocumentRenameProvider).
func HandleRename(deps *SelectionDeps, params protocol.VoiceTranscriptParams, vs *session.VoiceSession, text string) (protocol.VoiceTranscriptCompletion, string) {
	if deps == nil || deps.HostApply == nil || deps.NewBatchID == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "rename: host apply client not configured"
	}
	if vs == nil || len(vs.SearchResults) == 0 {
		return protocol.VoiceTranscriptCompletion{Success: false}, "rename: no search results"
	}
	if vs.ActiveSearchIndex < 0 || vs.ActiveSearchIndex >= len(vs.SearchResults) {
		return protocol.VoiceTranscriptCompletion{Success: false}, "rename: invalid active hit"
	}
	newName, ok := parseSymbolRenameNewName(strings.TrimSpace(text))
	if !ok {
		return protocol.VoiceTranscriptCompletion{Success: false}, `rename: use "rename … to <newName>" (single identifier)`
	}
	hit := vs.SearchResults[vs.ActiveSearchIndex]
	path := strings.TrimSpace(hit.Path)
	if path == "" {
		return protocol.VoiceTranscriptCompletion{Success: false}, "rename: empty hit path"
	}

	batchID := deps.NewBatchID()
	dir := []protocol.VoiceTranscriptDirective{
		{
			Kind: "rename",
			RenameDirective: &struct {
				Path     string `json:"path"`
				Position struct {
					Line      int64 `json:"line"`
					Character int64 `json:"character"`
				} `json:"position"`
				NewName string `json:"newName"`
			}{
				Path: path,
				Position: struct {
					Line      int64 `json:"line"`
					Character int64 `json:"character"`
				}{
					Line:      int64(hit.Line),
					Character: int64(hit.Character),
				},
				NewName: newName,
			},
		},
	}
	pending := &session.DirectiveApplyBatch{ID: batchID, NumDirectives: len(dir)}
	vs.PendingDirectiveApply = pending
	hostRes, err := deps.HostApply.ApplyDirectives(protocol.HostApplyParams{
		ApplyBatchId: batchID,
		ActiveFile:   params.ActiveFile,
		Directives:   dir,
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

	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       "renamed symbol to " + newName,
		UiDisposition: "hidden",
	}, ""
}

func parseSymbolRenameNewName(text string) (string, bool) {
	t := strings.ToLower(text)
	if !strings.Contains(t, "rename") {
		return "", false
	}
	idx := strings.LastIndex(t, " to ")
	if idx < 0 {
		return "", false
	}
	newName := strings.TrimSpace(text[idx+4:])
	newName = strings.Trim(newName, "\"'`")
	if newName == "" {
		return "", false
	}
	if strings.ContainsAny(newName, " \t\r\n") {
		return "", false
	}
	return newName, true
}
