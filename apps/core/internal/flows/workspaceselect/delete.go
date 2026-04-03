package workspaceselectflow

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// HandleDelete removes the active editor selection, the symbol at the cursor, the current workspace
// search hit (when it lies in the active file), or refuses a whole-file delete without a narrow target.
func HandleDelete(deps *SelectionDeps, params protocol.VoiceTranscriptParams, vs *session.VoiceSession, _ string) (protocol.VoiceTranscriptCompletion, string) {
	active := strings.TrimSpace(params.ActiveFile)
	if active == "" {
		return protocol.VoiceTranscriptCompletion{Success: false},
			"Open a file in the editor first. Delete removes text or a target in the active file."
	}
	if deps == nil || deps.ExtensionHost == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "extension host not configured"
	}
	body, err := deps.ExtensionHost.ReadHostFile(active)
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "read file: " + err.Error()
	}

	sl, sc, el, ec, ok := resolveEditRange(params, body)
	if !ok {
		return protocol.VoiceTranscriptCompletion{Success: false}, "delete: could not resolve target range"
	}

	if IsWholeFileRange(body, sl, sc, el, ec) && vs != nil && len(vs.SearchResults) > 0 {
		if idx := vs.ActiveSearchIndex; idx >= 0 && idx < len(vs.SearchResults) {
			hit := vs.SearchResults[idx]
			if pathsMatchForFlow(hit.Path, active) {
				if hsl, hsc, hel, hec, hok := RangeForSearchHit(body, hit); hok {
					sl, sc, el, ec = hsl, hsc, hel, hec
				}
			}
		}
	}

	if IsWholeFileRange(body, sl, sc, el, ec) {
		return protocol.VoiceTranscriptCompletion{Success: false}, "delete: select text, a symbol, or a search hit — whole-file delete is not allowed"
	}

	targetText, ok := extractRangeText(body, sl, sc, el, ec)
	if !ok {
		return protocol.VoiceTranscriptCompletion{Success: false}, "delete: invalid target range"
	}

	if deps.HostApply == nil || deps.NewBatchID == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "host apply client not configured"
	}

	sum := sha256.Sum256([]byte(targetText))
	fp := hex.EncodeToString(sum[:])
	batchID := deps.NewBatchID()
	dir := []protocol.VoiceTranscriptDirective{
		{
			Kind: "edit",
			EditDirective: &protocol.EditDirective{
				Kind: "success",
				Actions: []protocol.EditAction{
					{
						Kind:           "replace_range",
						Path:           active,
						NewText:        "",
						ExpectedSha256: fp,
						Range: &struct {
							StartLine int64 `json:"startLine"`
							StartChar int64 `json:"startChar"`
							EndLine   int64 `json:"endLine"`
							EndChar   int64 `json:"endChar"`
						}{
							StartLine: int64(sl),
							StartChar: int64(sc),
							EndLine:   int64(el),
							EndChar:   int64(ec),
						},
						EditId: "vocode-delete-" + batchID,
					},
				},
			},
		},
	}
	pending := &session.DirectiveApplyBatch{ID: batchID, NumDirectives: len(dir)}
	if vs != nil {
		vs.PendingDirectiveApply = pending
	}
	hostRes, err := deps.HostApply.ApplyDirectives(protocol.HostApplyParams{
		ApplyBatchId: batchID,
		ActiveFile:   params.ActiveFile,
		Directives:   dir,
	})
	if err != nil {
		if vs != nil {
			vs.PendingDirectiveApply = nil
		}
		return protocol.VoiceTranscriptCompletion{Success: false}, "host.applyDirectives failed: " + err.Error()
	}
	if err := pending.ConsumeHostApplyReport(batchID, hostRes.Items); err != nil {
		if vs != nil {
			vs.PendingDirectiveApply = nil
		}
		return protocol.VoiceTranscriptCompletion{Success: false}, "host apply failed: " + err.Error()
	}
	if vs != nil {
		vs.PendingDirectiveApply = nil
	}

	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       "deleted selection",
		UiDisposition: "hidden",
	}, ""
}

func pathsMatchForFlow(a, b string) bool {
	a = filepath.Clean(strings.TrimSpace(a))
	b = filepath.Clean(strings.TrimSpace(b))
	return a != "" && b != "" && strings.EqualFold(a, b)
}
