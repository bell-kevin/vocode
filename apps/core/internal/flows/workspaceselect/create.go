package workspaceselectflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"vocoding.net/vocode/v2/apps/core/internal/agent"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/hostdirectives"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// emptySHA256 is the hex SHA-256 of an empty string (zero-width replace_range).
const emptySHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// HandleCreate adds new text to the active file. Placement (beginning, end, before_line, after_line) and newText
// come from the placement model, which must interpret the full user transcript (including informal phrasing).
// On success it returns workspace Search state with a synthetic hit at the insertion (prepended, active index 0)
// so the host keeps the workspace-select surface open with the new content selected — including from root/file-select.
func HandleCreate(deps *SelectionDeps, params protocol.VoiceTranscriptParams, vs *session.VoiceSession, text string) (protocol.VoiceTranscriptCompletion, string) {
	instr := strings.TrimSpace(text)
	if instr == "" {
		return protocol.VoiceTranscriptCompletion{Success: false}, "create: empty instruction"
	}
	active := strings.TrimSpace(params.ActiveFile)
	if active == "" {
		return protocol.VoiceTranscriptCompletion{Success: false},
			"Open a file in the editor first. Create adds new content to the active editor buffer."
	}
	if deps.ExtensionHost == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "extension host not configured"
	}
	if deps.EditModel == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "create: no model configured (set VOCODE_AGENT_PROVIDER=openai and API keys)"
	}
	body, err := deps.ExtensionHost.ReadHostFile(active)
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "read file: " + err.Error()
	}
	lines := strings.Split(body, "\n")
	snippet := numberedFileSnippet(lines)

	addenda := StackPromptAddenda(params.WorkspaceSkillIds, params.WorkspacePromptAddendum)
	plan, err := callFileCreatePlanModel(context.Background(), deps.EditModel, instr, active, len(lines), snippet, addenda)
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "create model: " + err.Error()
	}
	if strings.TrimSpace(plan.NewText) == "" && !strings.Contains(plan.NewText, "\n") {
		return protocol.VoiceTranscriptCompletion{Success: false}, "create: model returned empty newText"
	}

	filteredImports := filterNewImportLines(body, plan.ImportLines)
	importBlock := importBlockForInsert(filteredImports)
	insertImpLn := importInsertLine(lines, active)
	lineOffImp := linesAddedByImportBlock(importBlock)
	imp := createImportContext{
		Block:      importBlock,
		InsertLine: insertImpLn,
		LineOff:    lineOffImp,
		Organize:   plan.shouldOrganizeImports(len(filteredImports) > 0),
	}

	placement := strings.ToLower(strings.TrimSpace(plan.Placement))
	switch placement {
	case "beginning", "start", "bof":
		return applyCreateBeginning(deps, params, vs, active, body, lines, plan.NewText, imp)
	case "end", "eof", "append":
		return applyCreateAppend(deps, params, vs, active, body, plan.NewText, imp)
	case "before_line", "line":
		line1 := plan.Line
		if line1 < 1 || line1 > len(lines) {
			return protocol.VoiceTranscriptCompletion{Success: false}, fmt.Sprintf("create: line %d out of range (file has %d line(s))", line1, len(lines))
		}
		return applyCreateBeforeLine(deps, params, vs, active, body, lines, line1, plan.NewText, imp)
	case "after_line":
		line1 := plan.Line
		if line1 < 1 || line1 > len(lines) {
			return protocol.VoiceTranscriptCompletion{Success: false}, fmt.Sprintf("create: line %d out of range (file has %d line(s))", line1, len(lines))
		}
		return applyCreateAfterLine(deps, params, vs, active, body, lines, line1, plan.NewText, imp)
	default:
		return protocol.VoiceTranscriptCompletion{Success: false}, `create: model placement must be "beginning", "end", "before_line", or "after_line"`
	}
}

func numberedFileSnippet(lines []string) string {
	const maxTotal = 120
	const headN = 80
	const tailN = 40
	n := len(lines)
	if n <= maxTotal {
		return formatNumberedLines(lines, 1)
	}
	var b strings.Builder
	b.WriteString(formatNumberedLines(lines[:headN], 1))
	b.WriteString(fmt.Sprintf("... (%d lines omitted) ...\n", n-headN-tailN))
	b.WriteString(formatNumberedLines(lines[n-tailN:], n-tailN+1))
	return b.String()
}

func formatNumberedLines(lines []string, startLine1 int) string {
	var b strings.Builder
	for i, ln := range lines {
		b.WriteString(fmt.Sprintf("%d|%s\n", startLine1+i, ln))
	}
	return b.String()
}

type fileCreatePlan struct {
	Placement       string   `json:"placement"`
	Line            int      `json:"line"`
	NewText         string   `json:"newText"`
	ImportLines     []string `json:"importLines"`
	OrganizeImports *bool    `json:"organizeImports,omitempty"`
}

func (p fileCreatePlan) shouldOrganizeImports(hasNewImports bool) bool {
	if !hasNewImports {
		return false
	}
	if p.OrganizeImports != nil {
		return *p.OrganizeImports
	}
	return true
}

// createImportContext carries optional top-of-file import insertion (same pipeline as scoped edit).
type createImportContext struct {
	Block      string
	InsertLine int
	LineOff    int
	Organize   bool
}

func (c createImportContext) active() bool { return c.Block != "" }

// syntheticFileLinesAfterImport returns the line list as it looks after inserting importBlock before insertLine.
func syntheticFileLinesAfterImport(lines []string, insertLine int, importBlock string) []string {
	importBlock = strings.TrimRight(importBlock, "\n")
	if importBlock == "" {
		return lines
	}
	chunks := strings.Split(importBlock, "\n")
	if insertLine < 0 {
		insertLine = 0
	}
	if insertLine > len(lines) {
		insertLine = len(lines)
	}
	out := make([]string, 0, len(lines)+len(chunks))
	out = append(out, lines[:insertLine]...)
	out = append(out, chunks...)
	out = append(out, lines[insertLine:]...)
	return out
}

func callFileCreatePlanModel(ctx context.Context, m agent.ModelClient, instruction, activeFile string, totalLines int, numberedSnippet, stackAddenda string) (fileCreatePlan, error) {
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"placement", "line", "newText", "importLines"},
		"properties": map[string]any{
			"placement": map[string]any{
				"type": "string",
				"enum": []string{"beginning", "end", "before_line", "after_line"},
			},
			"line": map[string]any{
				"type":        "integer",
				"description": "1-based line from the snippet: required for before_line and after_line (before_line = insert before that line; after_line = insert after that line). Use 0 for beginning or end.",
			},
			"newText": map[string]any{
				"type": "string",
				"description": "Only the new block to insert (no markdown fences). " +
					"MUST compile and typecheck for the language/stack implied by activeFile and the numbered snippet — every language, not optional. " +
					"No undefined identifiers: declare locals, types, and hooks in this block when the insertion is a full function/component, or rely on existing file context. " +
					"Do NOT put import or export-from lines in newText unless the insertion is into an import section: the host inserts importLines at the import region first, then inserts newText — duplicating imports causes doubles. ",
			},
			"importLines": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
				"description": `Always include this key (use [] if nothing new). When the file uses imports: required whenever newText references a symbol from another module that the file does not already import. ` +
					`Full lines (JS/TS: import { x } from "y"; host merges via organize imports). ` +
					`Use [] if every external symbol is already imported. In React function components, importing useState in importLines requires a useState(...) call inside newText when you use that state.`,
			},
			"organizeImports": map[string]any{
				"type":        "boolean",
				"description": "When importLines is non-empty, defaults to true (host merges/formats imports); set false to skip.",
			},
		},
	}
	userObj := map[string]any{
		"instruction":      instruction,
		"activeFile":       activeFile,
		"totalLines":       totalLines,
		"numberedSnippet":  numberedSnippet,
		"placementMeaning": "beginning: before line 1. end: after the last line (append). before_line: immediately before line `line`. after_line: immediately after line `line` (end of file if `line` is the last line). Line numbers are 1-based and must match the N| prefixes in numberedSnippet.",
	}
	userBytes, err := json.MarshalIndent(userObj, "", "  ")
	if err != nil {
		return fileCreatePlan{}, err
	}
	sys := strings.TrimSpace(`
You are Vocode's file-create helper. Produce newText and placement from the instruction and file context below.

From the user instruction and the numbered snippet (each line is "N|content"), decide exactly where NEW content should go in the active file. If they did not specify placement in speech, infer a reasonable placement from the snippet and conventions (e.g. append at end, after related code).

Respond with JSON: placement, line, newText, importLines (array, use [] if nothing new), organizeImports (optional).

You must map the user's wording to the correct placement yourself, including informal speech:
- "top", "start", "beginning of the file" → beginning (line 0).
- "bottom", "end of the file", "append" → end (line 0).
- "before line N", "above line N", "on line N" / "at line N" when they mean a new block landing on that line (insert before existing line N) → before_line with line N.
- "after line N", "below line N", "following line N" → after_line with line N.

If two readings are possible, prefer the one that matches common editor behavior: new code "on line 15" usually means before the current line-15 content (before_line 15), not end of file.

For "beginning" or "end", set line to 0. For "before_line" or "after_line", line must be a valid 1-based line number from the snippet (totalLines is the line count).

## Always (every language and stack)
- newText MUST compile and typecheck as real code for that file — no placeholders, no invented APIs, no undefined names (Go, Python, Rust, Java, JS/TS, etc.).
- If the insertion is a whole function, component, or class body, you may include statements before a return or JSX inside newText when that matches the instruction.
- Match idioms, naming, and patterns visible in the snippet.

## Imports (when the language uses them)
- The host applies importLines at the import area first, then inserts newText. Do not put import or export-from lines in newText unless you are inserting into an import section.
- importLines is a required JSON key: use [] when the file already imports every symbol you use; otherwise list full lines (including hooks like useState).
- React/TSX: listing useState in importLines is not enough — invoke it in the component (e.g. const [n, setN] = useState(0);) before JSX uses n or setN.
- organizeImports defaults to true when importLines is non-empty; set false to skip.

newText is ONLY the new block — no markdown fences or explanation. Match indentation to neighboring lines when obvious from the snippet.

No extra JSON keys beyond placement, line, newText, importLines, organizeImports.
`) + stackAddenda
	out, err := m.Call(ctx, agent.CompletionRequest{
		System:     sys,
		User:       string(userBytes),
		JSONSchema: schema,
	})
	if err != nil {
		return fileCreatePlan{}, err
	}
	var plan fileCreatePlan
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &plan); err != nil {
		return fileCreatePlan{}, fmt.Errorf("decode model json: %w", err)
	}
	return plan, nil
}

func applyCreateBeginning(deps *SelectionDeps, params protocol.VoiceTranscriptParams, vs *session.VoiceSession, active, body string, lines []string, newText string, imp createImportContext) (protocol.VoiceTranscriptCompletion, string) {
	core := strings.TrimRight(newText, "\n")
	sl, sc, el, ec := 0, 0, 0, 0
	workLines := lines
	sl2, sc2, el2, ec2 := sl, sc, el, ec
	if imp.active() {
		sl2, sc2, el2, ec2 = shiftRangeAfterImportInsert(sl, sc, el, ec, imp.InsertLine, imp.LineOff)
		workLines = syntheticFileLinesAfterImport(lines, imp.InsertLine, imp.Block)
	}
	pfx, suf := insertAffixForZeroWidth(workLines, sl2, sc2, core)
	payload := pfx + core + suf
	actions := buildCreateEditActions(active, imp, emptySHA256, sl2, sc2, el2, ec2, payload)
	return applyCreateEditActions(deps, params, vs, active, body, actions, imp, pfx, payload, core, "added at beginning of file", sl2, sc2, imp.active())
}

func applyCreateBeforeLine(deps *SelectionDeps, params protocol.VoiceTranscriptParams, vs *session.VoiceSession, active, body string, lines []string, line1 int, newText string, imp createImportContext) (protocol.VoiceTranscriptCompletion, string) {
	idx := line1 - 1
	core := strings.TrimRight(newText, "\n")
	sl, sc, el, ec := idx, 0, idx, 0
	workLines := lines
	sl2, sc2, el2, ec2 := sl, sc, el, ec
	if imp.active() {
		sl2, sc2, el2, ec2 = shiftRangeAfterImportInsert(sl, sc, el, ec, imp.InsertLine, imp.LineOff)
		workLines = syntheticFileLinesAfterImport(lines, imp.InsertLine, imp.Block)
	}
	pfx, suf := insertAffixForZeroWidth(workLines, sl2, sc2, core)
	payload := pfx + core + suf
	msg := fmt.Sprintf("added before line %d", line1)
	actions := buildCreateEditActions(active, imp, emptySHA256, sl2, sc2, el2, ec2, payload)
	return applyCreateEditActions(deps, params, vs, active, body, actions, imp, pfx, payload, core, msg, sl2, sc2, imp.active())
}

// applyCreateAfterLine inserts after 1-based line line1: before the next line if any, else at end of the last line.
// line1 is validated: 1 <= line1 <= len(lines).
func applyCreateAfterLine(deps *SelectionDeps, params protocol.VoiceTranscriptParams, vs *session.VoiceSession, active, body string, lines []string, line1 int, newText string, imp createImportContext) (protocol.VoiceTranscriptCompletion, string) {
	core := strings.TrimRight(newText, "\n")
	sl, sc, el, ec := rangeAfterLine(lines, line1)
	workLines := lines
	sl2, sc2, el2, ec2 := sl, sc, el, ec
	if imp.active() {
		sl2, sc2, el2, ec2 = shiftRangeAfterImportInsert(sl, sc, el, ec, imp.InsertLine, imp.LineOff)
		workLines = syntheticFileLinesAfterImport(lines, imp.InsertLine, imp.Block)
	}
	pfx, suf := insertAffixForZeroWidth(workLines, sl2, sc2, core)
	payload := pfx + core + suf
	msg := fmt.Sprintf("added after line %d", line1)
	actions := buildCreateEditActions(active, imp, emptySHA256, sl2, sc2, el2, ec2, payload)
	return applyCreateEditActions(deps, params, vs, active, body, actions, imp, pfx, payload, core, msg, sl2, sc2, imp.active())
}

func buildCreateEditActions(active string, imp createImportContext, fp string, sl, sc, el, ec int, createPayload string) []protocol.EditAction {
	rng := func(slo, sco, elo, eco int) *struct {
		StartLine int64 `json:"startLine"`
		StartChar int64 `json:"startChar"`
		EndLine   int64 `json:"endLine"`
		EndChar   int64 `json:"endChar"`
	} {
		return &struct {
			StartLine int64 `json:"startLine"`
			StartChar int64 `json:"startChar"`
			EndLine   int64 `json:"endLine"`
			EndChar   int64 `json:"endChar"`
		}{
			StartLine: int64(slo),
			StartChar: int64(sco),
			EndLine:   int64(elo),
			EndChar:   int64(eco),
		}
	}
	out := make([]protocol.EditAction, 0, 2)
	if imp.active() {
		out = append(out, protocol.EditAction{
			Kind:           "replace_range",
			Path:           active,
			NewText:        imp.Block,
			ExpectedSha256: emptySHA256,
			Range:          rng(imp.InsertLine, 0, imp.InsertLine, 0),
		})
	}
	out = append(out, protocol.EditAction{
		Kind:           "replace_range",
		Path:           active,
		NewText:        createPayload,
		ExpectedSha256: fp,
		Range:          rng(sl, sc, el, ec),
	})
	return out
}

func applyCreateEditActions(
	deps *SelectionDeps,
	params protocol.VoiceTranscriptParams,
	vs *session.VoiceSession,
	active, preBody string,
	actions []protocol.EditAction,
	imp createImportContext,
	insertPrefix, fullPayload, coreBlock, summary string,
	anchorSl, anchorSc int,
	multiStep bool,
) (protocol.VoiceTranscriptCompletion, string) {
	if deps.HostApply == nil || deps.NewBatchID == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "host apply client not configured"
	}
	batchID := deps.NewBatchID()
	for i := range actions {
		if imp.active() && i == 0 {
			actions[i].EditId = "vocode-create-import-" + batchID
		} else {
			actions[i].EditId = "vocode-create-" + batchID
		}
	}
	dir := []protocol.VoiceTranscriptDirective{
		{
			Kind: "edit",
			EditDirective: &protocol.EditDirective{
				Kind:    "success",
				Actions: actions,
			},
		},
	}
	if imp.active() && imp.Organize {
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
	line0, char0 := anchorSl, anchorSc
	if coreBlock != "" && deps.ExtensionHost != nil {
		if full, rerr := deps.ExtensionHost.ReadHostFile(active); rerr == nil {
			fn := normalizeEOL(full)
			if off, ok := anchorCreateCoreByteOffset(preBody, fn, coreBlock, insertPrefix, fullPayload, anchorSl, anchorSc, multiStep); ok {
				line0, char0 = lineCharUTF16FromByteIndex(fn, off)
			}
		}
	}
	return finishCreateWithWorkspaceSelection(deps, params, vs, active, line0, char0, coreBlock, summary)
}

// anchorCreateCoreByteOffset returns the byte offset of coreBlock in postFull (normalized).
func anchorCreateCoreByteOffset(preBody, postFull, coreBlock, insertPrefix, fullPayload string, sl, sc int, multiStep bool) (int, bool) {
	postN := normalizeEOL(postFull)
	if coreBlock == "" {
		return 0, false
	}
	if multiStep {
		off := strings.Index(postN, coreBlock)
		if off < 0 {
			return 0, false
		}
		return alignCoreAnchorByteOffset(postN, off, coreBlock), true
	}
	off, ok := createReplaceCoreByteOffset(preBody, postFull, sl, sc, insertPrefix, fullPayload, coreBlock)
	if !ok {
		return 0, false
	}
	return off, true
}

// rangeAfterLine returns a zero-width range (0-based line/char) for inserting after 1-based line line1.
func rangeAfterLine(lines []string, line1 int) (sl, sc, el, ec int) {
	n := len(lines)
	if line1 < n {
		// After 1-based line line1 → start of line (line1+1) in 1-based = 0-based index line1.
		return line1, 0, line1, 0
	}
	last := n - 1
	endCol := len(lines[last])
	return last, endCol, last, endCol
}

func normalizeEOL(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

// insertAffixForZeroWidth returns text before/after core for a zero-width replace at (sl, sc).
// It does not add “pretty” blank lines — only a single '\n' when needed so core is not glued to
// adjacent line text on the same row (BOF, start-of-line, or end-of-line / EOF inserts).
func insertAffixForZeroWidth(lines []string, sl, sc int, core string) (pfx, suf string) {
	if core == "" {
		return "", ""
	}
	n := len(lines)
	if n == 0 {
		return "", "\n"
	}
	last := n - 1

	// Beginning of file.
	if sl == 0 && sc == 0 {
		if lines[0] != "" && !strings.HasSuffix(core, "\n") {
			suf = "\n"
		}
		return "", suf
	}

	// Start of a line (before_line, after_line before next line).
	if sc == 0 && sl < n {
		if lines[sl] != "" && !strings.HasSuffix(core, "\n") {
			suf = "\n"
		}
		return "", suf
	}

	// End of line sl (after_line on last line, or rare tail of line).
	if sl < n && sc == len(lines[sl]) {
		if lines[sl] != "" && !strings.HasPrefix(core, "\n") {
			pfx = "\n"
		}
		if sl == last {
			suf = "\n"
		}
		return pfx, suf
	}

	return "", ""
}

// byteOffsetAtLineChar returns the byte offset in body for 0-based line and LSP UTF-16 character within that line.
func byteOffsetAtLineChar(body string, line0, char0 int) int {
	if line0 < 0 {
		line0 = 0
	}
	lines := strings.Split(body, "\n")
	off := 0
	for i := 0; i < line0 && i < len(lines); i++ {
		off += len(lines[i]) + 1
	}
	if line0 >= len(lines) {
		return len(body)
	}
	line := lines[line0]
	maxCol := LineUTF16Len(line)
	c := char0
	if c > maxCol {
		c = maxCol
	}
	return off + UTF16ColToByteOffset(line, c)
}

// alignCoreAnchorByteOffset returns the byte offset where coreBlock actually starts in postN, fixing off-by-one
// blank lines so LSP expansion sees a point inside the new code (not on an empty line above it).
func alignCoreAnchorByteOffset(postN string, off int, coreBlock string) int {
	if coreBlock == "" || off < 0 {
		return off
	}
	if off > len(postN) {
		off = len(postN)
	}
	tryAt := func(at int) (int, bool) {
		if at >= 0 && at+len(coreBlock) <= len(postN) && postN[at:at+len(coreBlock)] == coreBlock {
			return at, true
		}
		return 0, false
	}
	if a, ok := tryAt(off); ok {
		return a
	}
	j := off
	for j < len(postN) && (postN[j] == '\n' || postN[j] == '\r') {
		j++
	}
	if a, ok := tryAt(j); ok {
		return a
	}
	if off < len(postN) {
		if idx := strings.Index(postN[off:], coreBlock); idx >= 0 {
			return off + idx
		}
	}
	if idx := strings.Index(postN, coreBlock); idx >= 0 {
		return idx
	}
	return off
}

// createReplaceCoreByteOffset finds the start byte of coreBlock in the post-edit file after a replace_range insert.
func createReplaceCoreByteOffset(preBody, postFull string, sl, sc int, insertPrefix, fullPayload, coreBlock string) (int, bool) {
	preN := normalizeEOL(preBody)
	postN := normalizeEOL(postFull)
	insertAt := byteOffsetAtLineChar(preN, sl, sc)
	if insertAt > len(preN) {
		insertAt = len(preN)
	}
	expected := preN[:insertAt] + fullPayload + preN[insertAt:]
	if postN == expected {
		start := insertAt + len(insertPrefix)
		if start+len(coreBlock) <= len(postN) && postN[start:start+len(coreBlock)] == coreBlock {
			return alignCoreAnchorByteOffset(postN, start, coreBlock), true
		}
	}
	searchFrom := insertAt
	if searchFrom > len(postN) {
		searchFrom = 0
	}
	idx := strings.Index(postN[searchFrom:], coreBlock)
	if idx < 0 {
		idx = strings.Index(postN, coreBlock)
		if idx < 0 {
			return 0, false
		}
		return alignCoreAnchorByteOffset(postN, idx, coreBlock), true
	}
	return alignCoreAnchorByteOffset(postN, searchFrom+idx, coreBlock), true
}

// appendCreateToAppend builds the append chunk, the logical core block (trailing newlines trimmed), and
// the byte length of any prefix before core (a single '\n' when the file does not end with a newline).
func appendCreateToAppend(body, newText string) (toAppend string, core string, prefixLen int) {
	core = strings.TrimRight(newText, "\n")
	bn := normalizeEOL(body)
	if core == "" {
		if len(bn) == 0 {
			return "\n", "", 0
		}
		if strings.HasSuffix(bn, "\n") {
			return "", "", 0
		}
		return "\n", "", 0
	}
	if len(bn) == 0 {
		return core + "\n", core, 0
	}
	var sb strings.Builder
	if !strings.HasSuffix(bn, "\n") && !strings.HasPrefix(core, "\n") {
		sb.WriteString("\n")
	}
	pl := sb.Len()
	sb.WriteString(core)
	sb.WriteString("\n")
	return sb.String(), core, pl
}

func utf16CodeUnitsInString(s string) int {
	n := 0
	for _, r := range s {
		if r > 0xffff {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// lineCharUTF16FromByteIndex maps a byte offset in s (UTF-8) to VS Code-style 0-based line and UTF-16 character offset on that line.
func lineCharUTF16FromByteIndex(s string, byteIdx int) (line, char int) {
	if byteIdx < 0 {
		byteIdx = 0
	}
	if byteIdx > len(s) {
		byteIdx = len(s)
	}
	lineStart := 0
	line = 0
	for i := 0; i < byteIdx; {
		if s[i] == '\n' {
			line++
			lineStart = i + 1
			i++
			continue
		}
		_, sz := utf8.DecodeRuneInString(s[i:])
		if sz == 0 {
			break
		}
		i += sz
	}
	seg := s[lineStart:byteIdx]
	char = utf16CodeUnitsInString(seg)
	return line, char
}

func appendInsertionByteOffset(body, full string, prefixLen int, core string) (int, bool) {
	if core == "" {
		return 0, false
	}
	bn := normalizeEOL(body)
	fn := normalizeEOL(full)
	start := len(bn) + prefixLen
	if start+len(core) <= len(fn) && fn[start:start+len(core)] == core {
		return start, true
	}
	if len(bn) > len(fn) {
		return 0, false
	}
	idx := strings.Index(fn[len(bn):], core)
	if idx < 0 {
		return 0, false
	}
	return len(bn) + idx, true
}

// insertionAnchorAfterAppend locates the first line of coreBlock in the file after an append (scan from bottom).
func insertionAnchorAfterAppend(deps *SelectionDeps, active, coreBlock string) (line0, char0 int, ok bool) {
	coreBlock = strings.TrimRight(coreBlock, "\n")
	if coreBlock == "" || deps == nil || deps.ExtensionHost == nil {
		return 0, 0, false
	}
	firstLine, _, _ := strings.Cut(coreBlock, "\n")
	firstLine = strings.TrimSpace(firstLine)
	if firstLine == "" {
		return 0, 0, false
	}
	full, err := deps.ExtensionHost.ReadHostFile(active)
	if err != nil {
		return 0, 0, false
	}
	fileLines := strings.Split(full, "\n")
	for i := len(fileLines) - 1; i >= 0; i-- {
		line := fileLines[i]
		if idx := strings.Index(line, firstLine); idx >= 0 {
			return i, idx, true
		}
	}
	return 0, 0, false
}

// documentSymbolsForCreateReveal loads flattened LSP symbols after the file was edited.
// params.ActiveFileSymbols are from before the transcript and miss newly appended/inserted code, so expansion would fall back to a one-line range.
func documentSymbolsForCreateReveal(deps *SelectionDeps, path string) []hostdirectives.DocumentSymbol {
	if deps == nil || deps.ExtensionHost == nil {
		return nil
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	res, err := deps.ExtensionHost.GetDocumentSymbols(protocol.HostGetDocumentSymbolsParams{Path: path})
	if err != nil || len(res.Symbols) == 0 {
		return nil
	}
	return hostdirectives.DocumentSymbolsFromHostResult(res)
}

func applyHostRevealAfterCreate(deps *SelectionDeps, params protocol.VoiceTranscriptParams, vs *session.VoiceSession, path string, line0, char0, needleLen int, freshSyms []hostdirectives.DocumentSymbol) string {
	if deps == nil || deps.HostApply == nil || deps.NewBatchID == nil {
		return ""
	}
	if needleLen < 1 {
		needleLen = 1
	}
	var dirs []protocol.VoiceTranscriptDirective
	if len(freshSyms) > 0 {
		dirs = hostdirectives.HitNavigateDirectivesExpandWithSymbols(path, line0, char0, needleLen, freshSyms)
	} else if deps.HitNavigateDirectives != nil {
		dirs = deps.HitNavigateDirectives(params, path, line0, char0, needleLen)
	} else {
		dirs = hostdirectives.HitNavigateDirectives(path, line0, char0, needleLen)
	}
	batchID := deps.NewBatchID()
	pending := &session.DirectiveApplyBatch{ID: batchID, NumDirectives: len(dirs)}
	if vs != nil {
		vs.PendingDirectiveApply = pending
	}
	hostRes, err := deps.HostApply.ApplyDirectives(protocol.HostApplyParams{
		ApplyBatchId: batchID,
		ActiveFile:   params.ActiveFile,
		Directives:   dirs,
	})
	if err != nil {
		if vs != nil {
			vs.PendingDirectiveApply = nil
		}
		return "create: reveal navigation failed: " + err.Error()
	}
	if err := pending.ConsumeHostApplyReport(batchID, hostRes.Items); err != nil {
		if vs != nil {
			vs.PendingDirectiveApply = nil
		}
		return "create: reveal navigation failed: " + err.Error()
	}
	if vs != nil {
		vs.PendingDirectiveApply = nil
	}
	return ""
}

func finishCreateWithWorkspaceSelection(
	deps *SelectionDeps,
	params protocol.VoiceTranscriptParams,
	vs *session.VoiceSession,
	active string,
	anchorLine, anchorChar int,
	coreBlock string,
	summary string,
) (protocol.VoiceTranscriptCompletion, string) {
	coreBlock = strings.TrimRight(coreBlock, "\n")
	line0, char0 := anchorLine, anchorChar
	firstLine := coreBlock
	if i := strings.Index(coreBlock, "\n"); i >= 0 {
		firstLine = coreBlock[:i]
	}
	if strings.TrimSpace(firstLine) == "" {
		firstLine = "(added)"
	}
	preview := firstLine
	const pvMax = 200
	if len(preview) > pvMax {
		preview = preview[:pvMax] + "…"
	}
	freshSyms := documentSymbolsForCreateReveal(deps, active)
	preview = hostdirectives.CreateFlowHitPreview(freshSyms, line0, char0, preview)
	ml := int64(utf16CodeUnitsInString(strings.TrimSpace(firstLine)))
	if ml < 1 {
		ml = 1
	}
	p := ml
	syn := protocol.VoiceTranscriptSearchHit{
		Path:        active,
		Line:        int64(line0),
		Character:   int64(char0),
		Preview:     preview,
		MatchLength: &p,
	}
	if coreBlock != "" {
		if msg := applyHostRevealAfterCreate(deps, params, vs, active, line0, char0, int(ml), freshSyms); msg != "" {
			return protocol.VoiceTranscriptCompletion{Success: false}, msg
		}
	}
	var z int64
	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       summary,
		UiDisposition: "browse",
		Search: &protocol.VoiceTranscriptWorkspaceSearchState{
			Results:     []protocol.VoiceTranscriptSearchHit{syn},
			ActiveIndex: &z,
		},
	}, ""
}

func applyCreateAppend(deps *SelectionDeps, params protocol.VoiceTranscriptParams, vs *session.VoiceSession, active, body, newText string, imp createImportContext) (protocol.VoiceTranscriptCompletion, string) {
	toAppend, block, prefixLen := appendCreateToAppend(body, newText)
	if deps.HostApply == nil || deps.NewBatchID == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "host apply client not configured"
	}
	batchID := deps.NewBatchID()
	actions := make([]protocol.EditAction, 0, 2)
	if imp.active() {
		rng := &struct {
			StartLine int64 `json:"startLine"`
			StartChar int64 `json:"startChar"`
			EndLine   int64 `json:"endLine"`
			EndChar   int64 `json:"endChar"`
		}{
			StartLine: int64(imp.InsertLine),
			StartChar: 0,
			EndLine:   int64(imp.InsertLine),
			EndChar:   0,
		}
		actions = append(actions, protocol.EditAction{
			Kind:           "replace_range",
			Path:           active,
			NewText:        imp.Block,
			ExpectedSha256: emptySHA256,
			Range:          rng,
			EditId:         "vocode-create-import-" + batchID,
		})
	}
	actions = append(actions, protocol.EditAction{
		Kind:   "append_to_file",
		Path:   active,
		Text:   toAppend,
		EditId: "vocode-create-" + batchID,
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
	if imp.active() && imp.Organize {
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
	line0, char0 := 0, 0
	if full, rerr := deps.ExtensionHost.ReadHostFile(active); rerr == nil {
		fn := normalizeEOL(full)
		if block != "" {
			if imp.active() {
				if off := strings.Index(fn, block); off >= 0 {
					off = alignCoreAnchorByteOffset(fn, off, block)
					line0, char0 = lineCharUTF16FromByteIndex(fn, off)
				} else {
					line0, char0, _ = insertionAnchorAfterAppend(deps, active, block)
				}
			} else if off, ok := appendInsertionByteOffset(body, full, prefixLen, block); ok {
				off = alignCoreAnchorByteOffset(fn, off, block)
				line0, char0 = lineCharUTF16FromByteIndex(fn, off)
			} else {
				line0, char0, _ = insertionAnchorAfterAppend(deps, active, block)
			}
		}
	} else if block != "" {
		line0, char0, _ = insertionAnchorAfterAppend(deps, active, block)
	}
	return finishCreateWithWorkspaceSelection(deps, params, vs, active, line0, char0, block, "appended to file")
}
