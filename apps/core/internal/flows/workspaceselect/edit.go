package workspaceselectflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/agent"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// HandleEdit runs a scoped edit: resolve range from selection/cursor/symbols, ask the model for replacement text, apply replace_range.
func HandleEdit(deps *SelectionDeps, params protocol.VoiceTranscriptParams, vs *session.VoiceSession, text string) (protocol.VoiceTranscriptCompletion, string) {
	instr := strings.TrimSpace(text)
	if instr == "" {
		return protocol.VoiceTranscriptCompletion{Success: false}, "edit: empty instruction"
	}
	active := strings.TrimSpace(params.ActiveFile)
	if active == "" {
		return protocol.VoiceTranscriptCompletion{Success: false}, "edit: activeFile required"
	}
	if deps.ExtensionHost == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "extension host not configured"
	}
	if deps.EditModel == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "edit: no model configured (set VOCODE_AGENT_PROVIDER=openai and API keys)"
	}
	body, err := deps.ExtensionHost.ReadHostFile(active)
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "read file: " + err.Error()
	}
	sl, sc, el, ec, ok := resolveEditRange(params, body)
	if !ok {
		return protocol.VoiceTranscriptCompletion{Success: false}, "edit: could not resolve target range"
	}
	targetText, ok := extractRangeText(body, sl, sc, el, ec)
	if !ok {
		return protocol.VoiceTranscriptCompletion{Success: false}, "edit: invalid target range"
	}

	sum := sha256.Sum256([]byte(targetText))
	fp := hex.EncodeToString(sum[:])

	repl, err := callScopedEditModel(context.Background(), deps.EditModel, instr, active, sl, sc, el, ec, targetText)
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "edit model: " + err.Error()
	}

	if deps.HostApply == nil || deps.NewBatchID == nil {
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
						Kind:           "replace_range",
						Path:           active,
						NewText:        repl,
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
						EditId: "vocode-edit-" + batchID,
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
		Summary:       "applied edit",
		UiDisposition: "hidden",
	}, ""
}

func callScopedEditModel(ctx context.Context, m agent.ModelClient, instruction, activeFile string, sl, sc, el, ec int, targetText string) (string, error) {
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"replacementText"},
		"properties": map[string]any{
			"replacementText": map[string]any{"type": "string"},
		},
	}
	type targetPayload struct {
		Path      string `json:"path"`
		StartLine int    `json:"startLine"`
		StartChar int    `json:"startChar"`
		EndLine   int    `json:"endLine"`
		EndChar   int    `json:"endChar"`
	}
	userObj := map[string]any{
		"instruction": instruction,
		"activeFile":  activeFile,
		"target": targetPayload{
			Path:      activeFile,
			StartLine: sl,
			StartChar: sc,
			EndLine:   el,
			EndChar:   ec,
		},
		"targetText": targetText,
	}
	userBytes, err := json.MarshalIndent(userObj, "", "  ")
	if err != nil {
		return "", err
	}
	sys := strings.TrimSpace(`
You are Vocode's scoped edit model. You receive the user's instruction, active file path, a target range, and the exact current text in that range.
Respond with one JSON object: {"replacementText":"..."}. Only change what is needed inside the target range. No markdown fences or extra keys.
`)
	out, err := m.Call(ctx, agent.CompletionRequest{
		System:     sys,
		User:       string(userBytes),
		JSONSchema: schema,
	})
	if err != nil {
		return "", err
	}
	var parsed struct {
		ReplacementText string `json:"replacementText"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &parsed); err != nil {
		return "", fmt.Errorf("decode model json: %w", err)
	}
	return parsed.ReplacementText, nil
}
