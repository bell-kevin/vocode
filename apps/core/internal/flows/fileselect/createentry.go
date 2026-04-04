package fileselectflow

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/search"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/searchapply"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

var spokenDotWord = regexp.MustCompile(`(?i)(\w+)\s+dot\s+(\w+)`)

// HandleCreateEntry creates a new file under the focused file-selection row: workspace root,
// a selected directory, or next to a selected file (sibling). It does not create empty directories;
// use move to a path whose parent chain should exist — the host creates missing parent dirs for move_path.
func HandleCreateEntry(deps *SelectFileDeps, params protocol.VoiceTranscriptParams, vs *session.VoiceSession, text string) (protocol.VoiceTranscriptCompletion, string) {
	focus := strings.TrimSpace(vs.FileSelectionFocus)
	if focus == "" {
		return protocol.VoiceTranscriptCompletion{Success: false}, "create_entry: no file or folder selected"
	}
	parent := focus
	if !vs.FileFocusIsDir() {
		parent = filepath.Dir(focus)
	}
	text = strings.TrimSpace(text)
	var name string
	if deps != nil && deps.Editor != nil && deps.Editor.EditModel != nil {
		extracted, err := extractCreateEntryFileName(context.Background(), deps.Editor.EditModel, params, parent, text)
		if err == nil && strings.TrimSpace(extracted) != "" {
			name = extracted
		}
	}
	if strings.TrimSpace(name) == "" {
		name = parseCreateEntryNameHeuristic(text)
	}
	name = sanitizeNewFileName(name)
	if name == "" {
		return protocol.VoiceTranscriptCompletion{Success: false},
			`create_entry: could not derive a file name — say e.g. "file called what.js" or configure an agent model`
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

// parseCreateEntryNameHeuristic derives a basename without a model (spoken "dot", "called …").
func parseCreateEntryNameHeuristic(text string) string {
	s := strings.TrimSpace(text)
	s = strings.Trim(s, `"'`)
	lower := strings.ToLower(s)
	for _, needle := range []string{" called ", " named "} {
		if i := strings.LastIndex(lower, needle); i >= 0 {
			s = strings.TrimSpace(s[i+len(needle):])
			lower = strings.ToLower(s)
			break
		}
	}
	if len(lower) >= 6 && strings.HasPrefix(lower, "named ") {
		s = strings.TrimSpace(s[6:])
	}
	for spokenDotWord.MatchString(s) {
		s = spokenDotWord.ReplaceAllString(s, "$1.$2")
	}
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'`)
	s = trimCreateEntryTrailingCourtesy(s)
	return s
}

func trimCreateEntryTrailingCourtesy(s string) string {
	s = strings.TrimSpace(s)
	for {
		lower := strings.ToLower(s)
		cut := false
		for _, suf := range []string{" please", " thanks", " thank you"} {
			if strings.HasSuffix(lower, suf) {
				s = strings.TrimSpace(s[:len(s)-len(suf)])
				cut = true
				break
			}
		}
		if !cut {
			return s
		}
	}
}

func sanitizeNewFileName(text string) string {
	t := strings.TrimSpace(text)
	t = strings.Trim(t, `"'`)
	t = strings.ReplaceAll(t, "\\", "/")
	if strings.Contains(t, "..") {
		return ""
	}
	base := filepath.Base(t)
	base = strings.TrimSpace(base)
	if base == "." || base == "/" || base == "" {
		return ""
	}
	if strings.Contains(base, "..") {
		return ""
	}
	if strings.ContainsAny(base, `/\:`) {
		return ""
	}
	return search.TrimSttTrailingSentenceDot(base)
}
