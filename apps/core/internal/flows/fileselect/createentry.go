package fileselectflow

import (
	"fmt"
	"path/filepath"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/transcript/searchapply"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// HandleCreateEntry creates a new file under the focused file-selection row: sibling of a file,
// or inside a selected directory.
func HandleCreateEntry(deps *SelectFileDeps, params protocol.VoiceTranscriptParams, vs *session.VoiceSession, text string) (protocol.VoiceTranscriptCompletion, string) {
	focus := strings.TrimSpace(vs.FileSelectionFocus)
	if focus == "" {
		return protocol.VoiceTranscriptCompletion{Success: false}, "create_entry: no file or folder selected"
	}
	parent := focus
	if !vs.FileFocusIsDir() {
		parent = filepath.Dir(focus)
	}
	name := sanitizeNewFileName(text)
	if name == "" {
		return protocol.VoiceTranscriptCompletion{Success: false}, "create_entry: could not derive a file name from the transcript"
	}
	newPath := filepath.Join(parent, name)

	if deps == nil || deps.HostApply == nil || deps.NewBatchID == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "host apply client not configured"
	}

	batchID := deps.NewBatchID()
	dir := []protocol.VoiceTranscriptDirective{
		{
			Kind: "edit",
			EditDirective: &protocol.EditDirective{
				Kind: "success",
				Actions: []protocol.EditAction{
					{
						Kind:    "create_file",
						Path:    newPath,
						Content: "",
						EditId:  "vocode-create-" + batchID,
					},
				},
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

	openDirs := searchapply.OpenFirstFileDirectivesForPath(newPath)
	ob := deps.NewBatchID()
	p2 := &session.DirectiveApplyBatch{ID: ob, NumDirectives: len(openDirs)}
	vs.PendingDirectiveApply = p2
	hostRes2, err := deps.HostApply.ApplyDirectives(protocol.HostApplyParams{
		ApplyBatchId: ob,
		ActiveFile:   params.ActiveFile,
		Directives:   openDirs,
	})
	if err != nil {
		vs.PendingDirectiveApply = nil
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       fmt.Sprintf("created %s (open failed: %v)", name, err),
			UiDisposition: "hidden",
		}, ""
	}
	_ = p2.ConsumeHostApplyReport(ob, hostRes2.Items)
	vs.PendingDirectiveApply = nil

	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       fmt.Sprintf("created %s", name),
		UiDisposition: "hidden",
	}, ""
}

func sanitizeNewFileName(text string) string {
	t := strings.TrimSpace(text)
	t = strings.Trim(t, `"'`)
	t = strings.ReplaceAll(t, "\\", "/")
	base := filepath.Base(t)
	base = strings.TrimSpace(base)
	if base == "." || base == "/" {
		return ""
	}
	if strings.Contains(base, "..") {
		return ""
	}
	return base
}
