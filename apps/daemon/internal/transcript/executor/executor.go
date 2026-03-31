package executor

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/hostcaps"
	"vocoding.net/vocode/v2/apps/daemon/internal/workspace"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Executor runs one voice.transcript through the operation pipeline.
type Executor struct {
	agent          *agent.Agent
	symbolProvider hostcaps.SymbolProvider
}

// Options configures caps and optional symbol resolution for [Executor].
type Options struct {
	SymbolProvider hostcaps.SymbolProvider
}

// New constructs an [Executor].
func New(a *agent.Agent, opts Options) *Executor {
	if opts.SymbolProvider == nil {
		opts.SymbolProvider = hostcaps.ParamsSymbolProvider{}
	}
	return &Executor{
		agent:          a,
		symbolProvider: opts.SymbolProvider,
	}
}

// Execute runs one scoped edit operation and returns directives for the host to apply.
func (e *Executor) Execute(
	params protocol.VoiceTranscriptParams,
	gatheredIn agentcontext.Gathered,
) (protocol.VoiceTranscriptCompletion, []protocol.VoiceTranscriptDirective, agentcontext.Gathered, *agentcontext.DirectiveApplyBatch, bool, string) {
	text := strings.TrimSpace(params.Text)
	if text == "" {
		return protocol.VoiceTranscriptCompletion{}, nil, gatheredIn, nil, false, "empty transcript text"
	}

	// First-pass classifier: route transcript intent (instruction/search/question/irrelevant).
	clsIn := agentcontext.TranscriptClassifierContext{
		Instruction: text,
		Editor:      agentcontext.EditorSnapshotFromParams(params, resolveHostCursorSymbol(e.symbolProvider, params)),
	}
	clsRes, err := e.agent.ClassifyTranscript(context.Background(), clsIn)
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, nil, gatheredIn, nil, true, fmt.Sprintf("classifier failed: %v", err)
	}
	switch clsRes.Kind {
	case agent.TranscriptIrrelevant:
		return protocol.VoiceTranscriptCompletion{Success: true, TranscriptOutcome: "irrelevant"}, nil, gatheredIn, nil, true, ""
	case agent.TranscriptQuestion:
		ans := strings.TrimSpace(clsRes.AnswerText)
		// Put the answer in both answerText (structured) and summary (UI fallback).
		// Summary is what the panel shows even if answer-specific UI isn't available yet.
		return protocol.VoiceTranscriptCompletion{Success: true, TranscriptOutcome: "answer", AnswerText: ans, Summary: ans}, nil, gatheredIn, nil, true, ""
	case agent.TranscriptSearch:
		query := strings.TrimSpace(clsRes.SearchQuery)
		root := workspace.EffectiveWorkspaceRoot(params.WorkspaceRoot, params.ActiveFile)
		if strings.TrimSpace(root) == "" {
			return protocol.VoiceTranscriptCompletion{Success: false}, nil, gatheredIn, nil, true, "search requires workspaceRoot or activeFile"
		}
		hits, err := rgSearch(root, query, 20)
		if err != nil {
			return protocol.VoiceTranscriptCompletion{Success: false}, nil, gatheredIn, nil, true, fmt.Sprintf("search failed: %v", err)
		}
		if len(hits) == 0 {
			return protocol.VoiceTranscriptCompletion{Success: true, Summary: fmt.Sprintf("no matches for %q", query), TranscriptOutcome: "search"}, nil, gatheredIn, nil, true, ""
		}

		first := hits[0]
		wireHits := make([]struct {
			Path      string `json:"path"`
			Line      int64  `json:"line"`
			Character int64  `json:"character"`
			Preview   string `json:"preview"`
		}, 0, len(hits))
		for _, h := range hits {
			wireHits = append(wireHits, struct {
				Path      string `json:"path"`
				Line      int64  `json:"line"`
				Character int64  `json:"character"`
				Preview   string `json:"preview"`
			}{
				Path:      h.Path,
				Line:      int64(h.Line0),
				Character: int64(h.Char0),
				Preview:   h.Preview,
			})
		}

		open := protocol.VoiceTranscriptDirective{
			Kind: "navigate",
			NavigationDirective: &protocol.NavigationDirective{
				Kind: "success",
				Action: &protocol.NavigationAction{
					Kind: "open_file",
					OpenFile: &struct {
						Path string `json:"path"`
					}{Path: first.Path},
				},
			},
		}
		sel := protocol.VoiceTranscriptDirective{
			Kind: "navigate",
			NavigationDirective: &protocol.NavigationDirective{
				Kind: "success",
				Action: &protocol.NavigationAction{
					Kind: "select_range",
					SelectRange: &struct {
						Target struct {
							Path      string `json:"path,omitempty"`
							StartLine int64  `json:"startLine"`
							StartChar int64  `json:"startChar"`
							EndLine   int64  `json:"endLine"`
							EndChar   int64  `json:"endChar"`
						} `json:"target"`
					}{
						Target: struct {
							Path      string `json:"path,omitempty"`
							StartLine int64  `json:"startLine"`
							StartChar int64  `json:"startChar"`
							EndLine   int64  `json:"endLine"`
							EndChar   int64  `json:"endChar"`
						}{
							Path:      first.Path,
							StartLine: int64(first.Line0),
							StartChar: int64(first.Char0),
							EndLine:   int64(first.Line0),
							EndChar:   int64(first.Char0 + first.Len),
						},
					},
				},
			},
		}
		batchID, err := newDirectiveApplyBatchID()
		if err != nil {
			return protocol.VoiceTranscriptCompletion{Success: false}, nil, gatheredIn, nil, true, fmt.Sprintf("failed to create applyBatchId: %v", err)
		}
		pending := &agentcontext.DirectiveApplyBatch{ID: batchID, NumDirectives: 2}
		z := int64(0)
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           fmt.Sprintf("found %d matches for %q; opened first", len(hits), query),
			TranscriptOutcome: "search",
			SearchResults:     wireHits,
			ActiveSearchIndex: &z,
		}, []protocol.VoiceTranscriptDirective{open, sel}, gatheredIn, pending, true, ""
	default:
		// continue as instruction
	}

	// Heuristic: host code actions for deterministic refactors and fixes.
	if cad, ok := parseCodeAction(text, params); ok {
		batchID, err := newDirectiveApplyBatchID()
		if err != nil {
			return protocol.VoiceTranscriptCompletion{Success: false}, nil, gatheredIn, nil, true, fmt.Sprintf("failed to create applyBatchId: %v", err)
		}
		dir := protocol.VoiceTranscriptDirective{
			Kind: "code_action",
			CodeActionDirective: &struct {
				Path string `json:"path"`
				Range *struct {
					StartLine int64 `json:"startLine"`
					StartChar int64 `json:"startChar"`
					EndLine   int64 `json:"endLine"`
					EndChar   int64 `json:"endChar"`
				} `json:"range,omitempty"`
				ActionKind string `json:"actionKind"`
				PreferredTitleIncludes string `json:"preferredTitleIncludes,omitempty"`
			}{
				Path: cad.path,
				Range: cad.rng,
				ActionKind: cad.kind,
				PreferredTitleIncludes: cad.pref,
			},
		}
		pending := &agentcontext.DirectiveApplyBatch{ID: batchID, NumDirectives: 1}
		return protocol.VoiceTranscriptCompletion{Success: true, Summary: cad.summary}, []protocol.VoiceTranscriptDirective{dir}, gatheredIn, pending, true, ""
	}
	// Heuristic: host formatting.
	if fd, ok := parseFormat(text, params); ok {
		batchID, err := newDirectiveApplyBatchID()
		if err != nil {
			return protocol.VoiceTranscriptCompletion{Success: false}, nil, gatheredIn, nil, true, fmt.Sprintf("failed to create applyBatchId: %v", err)
		}
		dir := protocol.VoiceTranscriptDirective{
			Kind: "format",
			FormatDirective: &struct {
				Path  string `json:"path"`
				Scope string `json:"scope"`
				Range *struct {
					StartLine int64 `json:"startLine"`
					StartChar int64 `json:"startChar"`
					EndLine   int64 `json:"endLine"`
					EndChar   int64 `json:"endChar"`
				} `json:"range,omitempty"`
			}{
				Path:  fd.path,
				Scope: fd.scope,
				Range: fd.rng,
			},
		}
		pending := &agentcontext.DirectiveApplyBatch{ID: batchID, NumDirectives: 1}
		return protocol.VoiceTranscriptCompletion{Success: true, Summary: fd.summary}, []protocol.VoiceTranscriptDirective{dir}, gatheredIn, pending, true, ""
	}
	// Heuristic: deterministic rename when utterance looks like "rename X to Y".
	if newName, ok := parseRenameNewName(text); ok {
		active := strings.TrimSpace(params.ActiveFile)
		if active == "" || params.CursorPosition == nil {
			return protocol.VoiceTranscriptCompletion{Success: false}, nil, gatheredIn, nil, true, "rename requires activeFile and cursorPosition"
		}
		batchID, err := newDirectiveApplyBatchID()
		if err != nil {
			return protocol.VoiceTranscriptCompletion{Success: false}, nil, gatheredIn, nil, true, fmt.Sprintf("failed to create applyBatchId: %v", err)
		}
		dir := protocol.VoiceTranscriptDirective{
			Kind: "rename",
			RenameDirective: &struct {
				Path string `json:"path"`
				Position struct {
					Line int64 `json:"line"`
					Character int64 `json:"character"`
				} `json:"position"`
				NewName string `json:"newName"`
			}{
				Path: filepath.Clean(active),
				Position: struct {
					Line int64 `json:"line"`
					Character int64 `json:"character"`
				}{Line: params.CursorPosition.Line, Character: params.CursorPosition.Character},
				NewName: newName,
			},
		}
		pending := &agentcontext.DirectiveApplyBatch{ID: batchID, NumDirectives: 1}
		return protocol.VoiceTranscriptCompletion{Success: true, Summary: "rename"}, []protocol.VoiceTranscriptDirective{dir}, agentcontext.SeedGatheredActiveFile(gatheredIn, active), pending, true, ""
	}

	active := strings.TrimSpace(params.ActiveFile)
	if active == "" {
		return protocol.VoiceTranscriptCompletion{Success: false}, nil, gatheredIn, nil, true, "activeFile is required for scoped edit"
	}
	b, err := os.ReadFile(active)
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, nil, gatheredIn, nil, true, fmt.Sprintf("read active file: %v", err)
	}
	fileText := string(b)

	g := agentcontext.SeedGatheredActiveFile(gatheredIn, active)

	// Voice-only scope bridge: ask the model which scope to edit before requesting replacement text.
	scopeIn := agentcontext.ScopeIntentContext{
		Instruction: text,
		Editor:      agentcontext.EditorSnapshotFromParams(params, resolveHostCursorSymbol(e.symbolProvider, params)),
	}
	for i := range params.ActiveFileSymbols {
		s := params.ActiveFileSymbols[i]
		scopeIn.ActiveFileSymbols = append(scopeIn.ActiveFileSymbols, struct {
			Name string `json:"name"`
			Kind string `json:"kind"`
		}{
			Name: strings.TrimSpace(s.Name),
			Kind: strings.TrimSpace(s.Kind),
		})
		if len(scopeIn.ActiveFileSymbols) >= 32 {
			break
		}
	}
	scopeRes, err := e.agent.ScopeIntent(context.Background(), scopeIn)
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, nil, g, nil, true, fmt.Sprintf("scope intent failed: %v", err)
	}
	if scopeRes.ScopeKind == agent.ScopeClarify {
		q := strings.TrimSpace(scopeRes.ClarifyQuestion)
		if q == "" {
			q = "Which function or file should I edit?"
		}
		return protocol.VoiceTranscriptCompletion{Success: true, Summary: q, TranscriptOutcome: "clarify"}, nil, g, nil, true, ""
	}

	target, targetText, err := resolveScopedTarget(params, fileText, scopeRes)
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, nil, g, nil, true, err.Error()
	}
	sum := sha256.Sum256([]byte(targetText))
	target.Fingerprint = hex.EncodeToString(sum[:])

	ctx := agentcontext.ScopedEditContext{
		Instruction: text,
		Editor:      agentcontext.EditorSnapshotFromParams(params, nil),
		Target:      target,
		TargetText:  targetText,
	}
	out, err := e.agent.ScopedEdit(context.Background(), ctx)
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, nil, g, nil, true, fmt.Sprintf("model scoped edit failed: %v", err)
	}

	dir := protocol.VoiceTranscriptDirective{
		Kind: "edit",
		EditDirective: &protocol.EditDirective{
			Kind: "success",
			Actions: []protocol.EditAction{
				{
					Kind: "replace_range",
					Path: target.Path,
					Range: &struct {
						StartLine int64 `json:"startLine"`
						StartChar int64 `json:"startChar"`
						EndLine   int64 `json:"endLine"`
						EndChar   int64 `json:"endChar"`
					}{
						StartLine: int64(target.Range.StartLine),
						StartChar: int64(target.Range.StartChar),
						EndLine:   int64(target.Range.EndLine),
						EndChar:   int64(target.Range.EndChar),
					},
					NewText: out.ReplacementText,
					ExpectedSha256: target.Fingerprint,
				},
			},
		},
	}

	batchID, err := newDirectiveApplyBatchID()
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, nil, g, nil, true, fmt.Sprintf("failed to create applyBatchId: %v", err)
	}
	pending := &agentcontext.DirectiveApplyBatch{ID: batchID, NumDirectives: 1}
	return protocol.VoiceTranscriptCompletion{Success: true, Summary: "scoped edit"}, []protocol.VoiceTranscriptDirective{dir}, g, pending, true, ""
}

type rgHit struct {
	Path  string
	Line0 int
	Char0 int
	Len   int
	Preview string
}

func rgBinary() string {
	if p := strings.TrimSpace(os.Getenv("VOCODE_RG_BIN")); p != "" {
		return p
	}
	return "rg"
}

var rgLineRe = regexp.MustCompile(`^(.*):(\d+):(\d+):(.*)$`)

func rgSearch(root, query string, maxHits int) ([]rgHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if maxHits <= 0 {
		maxHits = 10
	}
	// Use fixed-string search (ripgrep supports -F / --fixed-strings).
	cmd := exec.Command(rgBinary(), "--column", "-n", "--fixed-strings", query, root)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		// ripgrep exit code 1 means "no matches", not an execution error.
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 {
			return nil, nil
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s", msg)
	}
	out := make([]rgHit, 0, maxHits)
	sc := bufio.NewScanner(bytes.NewReader(stdout.Bytes()))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		m := rgLineRe.FindStringSubmatch(line)
		if len(m) != 5 {
			continue
		}
		path := filepath.Clean(strings.TrimSpace(m[1]))
		ln1 := atoiSafe(m[2])
		col1 := atoiSafe(m[3])
		if ln1 <= 0 || col1 <= 0 {
			continue
		}
		out = append(out, rgHit{
			Path:  path,
			Line0: ln1 - 1,
			Char0: col1 - 1,
			Len:   len(query),
			Preview: strings.TrimSpace(m[4]),
		})
		if len(out) >= maxHits {
			break
		}
	}
	return out, nil
}

func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}


type parsedCodeAction struct {
	path    string
	rng     *struct {
		StartLine int64 `json:"startLine"`
		StartChar int64 `json:"startChar"`
		EndLine   int64 `json:"endLine"`
		EndChar   int64 `json:"endChar"`
	}
	kind    string
	pref    string
	summary string
}

func parseCodeAction(text string, params protocol.VoiceTranscriptParams) (parsedCodeAction, bool) {
	l := strings.ToLower(strings.TrimSpace(text))
	active := filepath.Clean(strings.TrimSpace(params.ActiveFile))
	if active == "" {
		return parsedCodeAction{}, false
	}
	sel := params.ActiveSelection
	var rng *struct {
		StartLine int64 `json:"startLine"`
		StartChar int64 `json:"startChar"`
		EndLine   int64 `json:"endLine"`
		EndChar   int64 `json:"endChar"`
	}
	if sel != nil {
		rng = &struct {
			StartLine int64 `json:"startLine"`
			StartChar int64 `json:"startChar"`
			EndLine   int64 `json:"endLine"`
			EndChar   int64 `json:"endChar"`
		}{StartLine: sel.StartLine, StartChar: sel.StartChar, EndLine: sel.EndLine, EndChar: sel.EndChar}
	}

	switch {
	case strings.Contains(l, "extract function"):
		if rng == nil || (rng.StartLine == rng.EndLine && rng.StartChar == rng.EndChar) {
			return parsedCodeAction{summary: "extract function requires a selection"}, false
		}
		return parsedCodeAction{path: active, rng: rng, kind: "refactor.extract.function", summary: "extract function"}, true
	case strings.Contains(l, "extract variable"):
		if rng == nil || (rng.StartLine == rng.EndLine && rng.StartChar == rng.EndChar) {
			return parsedCodeAction{summary: "extract variable requires a selection"}, false
		}
		return parsedCodeAction{path: active, rng: rng, kind: "refactor.extract.variable", summary: "extract variable"}, true
	case strings.Contains(l, "extract constant"):
		if rng == nil || (rng.StartLine == rng.EndLine && rng.StartChar == rng.EndChar) {
			return parsedCodeAction{summary: "extract constant requires a selection"}, false
		}
		return parsedCodeAction{path: active, rng: rng, kind: "refactor.extract.constant", summary: "extract constant"}, true
	case strings.Contains(l, "inline"):
		return parsedCodeAction{path: active, rng: rng, kind: "refactor.inline", summary: "inline"}, true
	case strings.Contains(l, "organize imports"):
		return parsedCodeAction{path: active, kind: "source.organizeImports", summary: "organize imports"}, true
	case strings.Contains(l, "fix all"):
		return parsedCodeAction{path: active, kind: "source.fixAll", summary: "fix all"}, true
	case strings.Contains(l, "quick fix") || strings.Contains(l, "quickfix"):
		return parsedCodeAction{path: active, rng: rng, kind: "quickfix", summary: "quick fix"}, true
	default:
		return parsedCodeAction{}, false
	}
}

type parsedFormat struct {
	path    string
	scope   string
	rng     *struct {
		StartLine int64 `json:"startLine"`
		StartChar int64 `json:"startChar"`
		EndLine   int64 `json:"endLine"`
		EndChar   int64 `json:"endChar"`
	}
	summary string
}

func parseFormat(text string, params protocol.VoiceTranscriptParams) (parsedFormat, bool) {
	l := strings.ToLower(strings.TrimSpace(text))
	if !strings.Contains(l, "format") {
		return parsedFormat{}, false
	}
	active := filepath.Clean(strings.TrimSpace(params.ActiveFile))
	if active == "" {
		return parsedFormat{}, false
	}
	// Prefer selection if user says selection/range, else document.
	if strings.Contains(l, "selection") || strings.Contains(l, "selected") {
		sel := params.ActiveSelection
		if sel == nil {
			return parsedFormat{}, false
		}
		return parsedFormat{
			path:  active,
			scope: "selection",
			rng: &struct {
				StartLine int64 `json:"startLine"`
				StartChar int64 `json:"startChar"`
				EndLine   int64 `json:"endLine"`
				EndChar   int64 `json:"endChar"`
			}{StartLine: sel.StartLine, StartChar: sel.StartChar, EndLine: sel.EndLine, EndChar: sel.EndChar},
			summary: "format selection",
		}, true
	}
	return parsedFormat{path: active, scope: "document", summary: "format document"}, true
}

func parseRenameNewName(text string) (string, bool) {
	t := strings.ToLower(text)
	if !strings.Contains(t, "rename") {
		return "", false
	}
	// Very small heuristic: find the last " to " and use the suffix as new name.
	idx := strings.LastIndex(t, " to ")
	if idx < 0 {
		return "", false
	}
	newName := strings.TrimSpace(text[idx+4:])
	newName = strings.Trim(newName, "\"'`")
	if newName == "" {
		return "", false
	}
	// Avoid multi-word names.
	if strings.ContainsAny(newName, " \t\r\n") {
		return "", false
	}
	return newName, true
}

func resolveScopedTarget(params protocol.VoiceTranscriptParams, fileText string, scope agent.ScopeIntentResult) (agentcontext.ResolvedTarget, string, error) {
	active := filepath.Clean(strings.TrimSpace(params.ActiveFile))
	if active == "" {
		return agentcontext.ResolvedTarget{}, "", fmt.Errorf("activeFile is required")
	}
	lines := strings.Split(fileText, "\n")
	lastLine := len(lines) - 1
	lastChar := 0
	if lastLine >= 0 {
		lastChar = len(lines[lastLine])
	}
	// Default: whole file.
	startLine, endLine := 0, lastLine

	if scope.ScopeKind == agent.ScopeCurrentFile {
		startLine, endLine = 0, lastLine
	} else if scope.ScopeKind == agent.ScopeNamedSymbol && len(params.ActiveFileSymbols) > 0 {
		want := strings.ToLower(strings.TrimSpace(scope.SymbolName))
		bestIdx := -1
		bestSize := 0
		for i := range params.ActiveFileSymbols {
			s := params.ActiveFileSymbols[i]
			if strings.ToLower(strings.TrimSpace(s.Name)) != want {
				continue
			}
			r := s.Range
			sz := (int(r.EndLine)-int(r.StartLine))*100000 + (int(r.EndChar) - int(r.StartChar))
			if bestIdx == -1 || sz < bestSize {
				bestIdx = i
				bestSize = sz
			}
		}
		if bestIdx != -1 {
			r := params.ActiveFileSymbols[bestIdx].Range
			startLine = int(r.StartLine)
			endLine = int(r.EndLine)
		}
	} else if scope.ScopeKind == agent.ScopeCurrentFunction && len(params.ActiveFileSymbols) > 0 {
		if cp := params.CursorPosition; cp != nil {
			line := int(cp.Line)
			char := int(cp.Character)
			bestIdx := -1
			bestSize := 0
			for i := range params.ActiveFileSymbols {
				s := params.ActiveFileSymbols[i]
				r := s.Range
				if line < int(r.StartLine) || line > int(r.EndLine) {
					continue
				}
				if line == int(r.StartLine) && char < int(r.StartChar) {
					continue
				}
				if line == int(r.EndLine) && char > int(r.EndChar) {
					continue
				}
				sz := (int(r.EndLine)-int(r.StartLine))*100000 + (int(r.EndChar) - int(r.StartChar))
				if bestIdx == -1 || sz < bestSize {
					bestIdx = i
					bestSize = sz
				}
			}
			if bestIdx != -1 {
				r := params.ActiveFileSymbols[bestIdx].Range
				startLine = int(r.StartLine)
				endLine = int(r.EndLine)
			}
		}
	}
	if startLine < 0 {
		startLine = 0
	}
	if endLine < startLine {
		endLine = startLine
	}
	if endLine > lastLine {
		endLine = lastLine
	}

	// Normalize to whole lines to avoid UTF-16 slicing issues in Go.
	text := ""
	if lastLine >= 0 && startLine <= endLine {
		text = strings.Join(lines[startLine:endLine+1], "\n")
	}
	t := agentcontext.ResolvedTarget{
		Path: filepath.Clean(active),
		Range: agentcontext.Range{
			StartLine: startLine,
			StartChar: 0,
			EndLine:   endLine,
			EndChar:   lastChar,
		},
	}
	return t, text, nil
}
