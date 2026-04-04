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
		return protocol.VoiceTranscriptCompletion{Success: false},
			"Open a file in the editor first. Edit changes code in the active editor."
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

	addenda := StackPromptAddenda(params.WorkspaceSkillIds, params.WorkspacePromptAddendum)
	modelOut, err := callScopedEditModel(context.Background(), deps.EditModel, instr, active, sl, sc, el, ec, targetText, body, addenda)
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "edit model: " + err.Error()
	}
	if rest, peeled := mergePeelLeadingImportLines(modelOut.ReplacementText, targetText); len(peeled) > 0 {
		modelOut.ReplacementText = rest
		modelOut.ImportLines = append(peeled, modelOut.ImportLines...)
	}

	if deps.HostApply == nil || deps.NewBatchID == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "host apply client not configured"
	}
	batchID := deps.NewBatchID()

	filteredImports := filterNewImportLines(body, modelOut.ImportLines)
	importBlock := importBlockForInsert(filteredImports)
	insertLine := importInsertLine(strings.Split(body, "\n"), active)
	lineOff := linesAddedByImportBlock(importBlock)

	sl2, sc2, el2, ec2 := sl, sc, el, ec
	if importBlock != "" {
		sl2, sc2, el2, ec2 = shiftRangeAfterImportInsert(sl, sc, el, ec, insertLine, lineOff)
	}

	actions := make([]protocol.EditAction, 0, 2)
	if importBlock != "" {
		actions = append(actions, protocol.EditAction{
			Kind:           "replace_range",
			Path:           active,
			NewText:        importBlock,
			ExpectedSha256: emptySHA256,
			Range: &struct {
				StartLine int64 `json:"startLine"`
				StartChar int64 `json:"startChar"`
				EndLine   int64 `json:"endLine"`
				EndChar   int64 `json:"endChar"`
			}{
				StartLine: int64(insertLine),
				StartChar: 0,
				EndLine:   int64(insertLine),
				EndChar:   0,
			},
			EditId: "vocode-imports-" + batchID,
		})
	}
	actions = append(actions, protocol.EditAction{
		Kind:           "replace_range",
		Path:           active,
		NewText:        modelOut.ReplacementText,
		ExpectedSha256: fp,
		Range: &struct {
			StartLine int64 `json:"startLine"`
			StartChar int64 `json:"startChar"`
			EndLine   int64 `json:"endLine"`
			EndChar   int64 `json:"endChar"`
		}{
			StartLine: int64(sl2),
			StartChar: int64(sc2),
			EndLine:   int64(el2),
			EndChar:   int64(ec2),
		},
		EditId: "vocode-edit-" + batchID,
	})

	dir := []protocol.VoiceTranscriptDirective{
		{
			Kind: "edit",
			EditDirective: &protocol.EditDirective{
				Kind:    "success",
				Actions: actions,
			},
		},
	}

	shouldOrganize := len(filteredImports) > 0
	if shouldOrganize && modelOut.OrganizeImports != nil && !*modelOut.OrganizeImports {
		shouldOrganize = false
	}
	if shouldOrganize {
		dir = append(dir, protocol.VoiceTranscriptDirective{
			Kind: "code_action",
			CodeActionDirective: &struct {
				Path   string `json:"path"`
				Range  *struct {
					StartLine int64 `json:"startLine"`
					StartChar int64 `json:"startChar"`
					EndLine   int64 `json:"endLine"`
					EndChar   int64 `json:"endChar"`
				} `json:"range,omitempty"`
				ActionKind             string `json:"actionKind"`
				PreferredTitleIncludes string `json:"preferredTitleIncludes,omitempty"`
			}{
				Path:       active,
				ActionKind: "source.organizeImports",
			},
		})
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

const scopedEditFullFileContextMaxBytes = 120_000

func truncateScopedEditFileContext(s string) string {
	const max = scopedEditFullFileContextMaxBytes
	if len(s) <= max {
		return s
	}
	s = s[:max]
	for len(s) > 0 && s[len(s)-1]&0xC0 == 0x80 {
		s = s[:len(s)-1]
	}
	return s + "\n…(truncated for model context)…"
}

type scopedEditModelOut struct {
	ReplacementText string
	ImportLines     []string
	// OrganizeImports is nil unless the model set organizeImports; when nil and filteredImports is non-empty, default is organize (see HandleEdit).
	OrganizeImports *bool
}

func callScopedEditModel(ctx context.Context, m agent.ModelClient, instruction, activeFile string, sl, sc, el, ec int, targetText, fullFile, stackAddenda string) (scopedEditModelOut, error) {
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"replacementText", "importLines"},
		"properties": map[string]any{
			"replacementText": map[string]any{
				"type": "string",
				"description": "New source for targetText only. " +
					"MUST compile and typecheck for the language/stack implied by activeFile, fullFile, and path (every language — not optional). " +
					"No undefined identifiers: declare locals, types, and closures before use inside the span; do not reference symbols that are not in scope. " +
					"For languages with separate import sites (JS/TS, etc.), symbols from another module must already appear in fullFile or be added via importLines — never use a name from 'react', 'react-native', stdlib, or any package without an import. " +
					"Do NOT put import or export-from lines in replacementText unless the selection is only an import section: the host inserts importLines at the file import region first, then applies replacementText; duplicating imports here causes double imports. " +
					"Additional stack-specific rules may appear after the main system prompt (e.g. React Native).",
			},
			"importLines": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
				"description": `Always include this key (use [] if nothing new). For any file that uses import/include statements: required whenever replacementText uses a name from another module that fullFile does not already import (types, functions, hooks, components, constants). ` +
					`JS/TS: full lines, e.g. import { foo } from "bar"; the host inserts these lines, then runs organize imports when enabled — prefer importLines over pasting imports into replacementText. ` +
					`Go: import "path" or grouped spec lines inside import ( ). Python: import lines or from x import y. ` +
					`Use [] only when every external symbol you use is already imported in fullFile.`,
			},
			"organizeImports": map[string]any{
				"type":        "boolean",
				"description": `When importLines is non-empty: if true (default), the host runs the editor "organize imports" action after applying (works for TypeScript/JavaScript via tsserver and Go via gopls when installed). Set false to skip.`,
			},
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
		"fullFile":   truncateScopedEditFileContext(fullFile),
	}
	userBytes, err := json.MarshalIndent(userObj, "", "  ")
	if err != nil {
		return scopedEditModelOut{}, err
	}
	sys := strings.TrimSpace(`
You are Vocode's scoped edit model. You receive the user's instruction, active file path, a target range, targetText (exact source in that range), and fullFile (entire buffer, possibly truncated at the end).

## Always (every language and stack)
- replacementText MUST compile and typecheck as real code for that file — no placeholders, no invented APIs, no undefined names. This is mandatory for Go, Python, Rust, Java, JS/TS, etc., not only for one framework.
- If the selection spans a whole function or block, you may add statements above an inner return (e.g. hooks or locals) inside replacementText whenever that text includes those lines.
- Declare everything you use: if you reference count, setCount, helper types, or callbacks, define them inside the edited span or ensure they already exist in targetText/fullFile.
- Match idioms, naming, and patterns visible in fullFile and targetText.

## Imports and modules (when the language uses them)
- The host applies importLines at the top import area first, then replaces the selection with replacementText. Do not put import or export-from lines in replacementText unless the selection is only imports — use importLines for every new module symbol (including hooks like useState).
- importLines is a required JSON key: use [] when fullFile already imports every symbol you use; otherwise list full import lines.
- Importing a hook or function does not declare state: in React/TSX, if JSX or handlers use count/setCount (or similar), you must include the hook call in replacementText in the same function (e.g. const [count, setCount] = useState(0);) before the return — not only import useState in importLines.
- organizeImports (optional): when importLines is non-empty, defaults to true for hosts that can merge/format imports; set false to skip.

Output JSON keys only: replacementText, importLines (required array, may be empty []), organizeImports (optional).

No markdown fences or extra keys.
`) + stackAddenda
	out, err := m.Call(ctx, agent.CompletionRequest{
		System:     sys,
		User:       string(userBytes),
		JSONSchema: schema,
	})
	if err != nil {
		return scopedEditModelOut{}, err
	}
	var parsed struct {
		ReplacementText string   `json:"replacementText"`
		ImportLines     []string `json:"importLines"`
		OrganizeImports *bool    `json:"organizeImports"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &parsed); err != nil {
		return scopedEditModelOut{}, fmt.Errorf("decode model json: %w", err)
	}
	return scopedEditModelOut{
		ReplacementText: parsed.ReplacementText,
		ImportLines:     parsed.ImportLines,
		OrganizeImports: parsed.OrganizeImports,
	}, nil
}
