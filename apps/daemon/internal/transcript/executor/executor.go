package executor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/hostcaps"
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

// FlowMode selects routing before the instruction pipeline ([ExecuteOptions.Mode]).
type FlowMode string

const (
	// FlowModeMain runs the classifier then instruction heuristics / scoped edit.
	FlowModeMain FlowMode = ""
	// FlowModeSelection skips classifier; utterances apply to the locked code match (selection flow).
	FlowModeSelection FlowMode = "selection"
)

// ExecuteOptions configures [Executor.Execute].
type ExecuteOptions struct {
	Mode FlowMode
}

// Execute runs one voice.transcript through the operation pipeline.
func (e *Executor) Execute(
	params protocol.VoiceTranscriptParams,
	gatheredIn agentcontext.Gathered,
	opt ExecuteOptions,
) (protocol.VoiceTranscriptCompletion, []protocol.VoiceTranscriptDirective, agentcontext.Gathered, *agentcontext.DirectiveApplyBatch, bool, string) {
	text := strings.TrimSpace(params.Text)
	if text == "" {
		return protocol.VoiceTranscriptCompletion{}, nil, gatheredIn, nil, false, "empty transcript text"
	}

	if opt.Mode == FlowModeSelection {
		return e.executeInstructionPath(params, gatheredIn, text, agentcontext.FlowKindSelection)
	}

	// First-pass classifier: route transcript intent (instruction/search/question/file_selection/irrelevant).
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
		return protocol.VoiceTranscriptCompletion{Success: true, TranscriptOutcome: "irrelevant", UiDisposition: "skipped"}, nil, gatheredIn, nil, true, ""
	case agent.TranscriptQuestion:
		ans := strings.TrimSpace(clsRes.AnswerText)
		// Put the answer in both answerText (structured) and summary (UI fallback).
		// Summary is what the panel shows even if answer-specific UI isn't available yet.
		return protocol.VoiceTranscriptCompletion{Success: true, TranscriptOutcome: "answer", UiDisposition: "hidden", AnswerText: ans, Summary: ans}, nil, gatheredIn, nil, true, ""
	case agent.TranscriptSearch:
		query := strings.TrimSpace(clsRes.SearchQuery)
		if query == "" {
			if q, ok := searchLikeQueryFromText(text); ok {
				query = q
			}
		}
		if query == "" {
			return protocol.VoiceTranscriptCompletion{Success: false}, nil, gatheredIn, nil, true, "search classification missing searchQuery"
		}
		return workspaceSearch(params, gatheredIn, query)
	case agent.TranscriptFileSelection:
		if !params.WorkspaceFolderOpen {
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				Summary:           "Open a folder in VS Code to browse and change files by voice.",
				TranscriptOutcome: "needs_workspace_folder",
				UiDisposition:     "shown",
			}, nil, gatheredIn, nil, true, ""
		}
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           "File selection",
			TranscriptOutcome: "file_selection",
			UiDisposition:     "hidden",
		}, nil, gatheredIn, nil, true, ""
	default:
		// Model may pick "instruction" when the user clearly asked to find something in the repo
		// (e.g. cursor already on a match). Still run ripgrep + search UI so other files show up.
		if q, ok := searchLikeQueryFromText(text); ok {
			return workspaceSearch(params, gatheredIn, q)
		}
	}

	return e.executeInstructionPath(params, gatheredIn, text, agentcontext.FlowKindMain)
}

func (e *Executor) executeInstructionPath(
	params protocol.VoiceTranscriptParams,
	gatheredIn agentcontext.Gathered,
	text string,
	clarifyParentFlow string,
) (protocol.VoiceTranscriptCompletion, []protocol.VoiceTranscriptDirective, agentcontext.Gathered, *agentcontext.DirectiveApplyBatch, bool, string) {
	// Heuristic: host code actions for deterministic refactors and fixes.
	if cad, ok := parseCodeAction(text, params); ok {
		batchID, err := newDirectiveApplyBatchID()
		if err != nil {
			return protocol.VoiceTranscriptCompletion{Success: false}, nil, gatheredIn, nil, true, fmt.Sprintf("failed to create applyBatchId: %v", err)
		}
		dir := protocol.VoiceTranscriptDirective{
			Kind: "code_action",
			CodeActionDirective: &struct {
				Path  string `json:"path"`
				Range *struct {
					StartLine int64 `json:"startLine"`
					StartChar int64 `json:"startChar"`
					EndLine   int64 `json:"endLine"`
					EndChar   int64 `json:"endChar"`
				} `json:"range,omitempty"`
				ActionKind             string `json:"actionKind"`
				PreferredTitleIncludes string `json:"preferredTitleIncludes,omitempty"`
			}{
				Path:                   cad.path,
				Range:                  cad.rng,
				ActionKind:             cad.kind,
				PreferredTitleIncludes: cad.pref,
			},
		}
		pending := &agentcontext.DirectiveApplyBatch{ID: batchID, NumDirectives: 1}
		return protocol.VoiceTranscriptCompletion{Success: true, Summary: cad.summary, UiDisposition: "shown"}, []protocol.VoiceTranscriptDirective{dir}, gatheredIn, pending, true, ""
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
		return protocol.VoiceTranscriptCompletion{Success: true, Summary: fd.summary, UiDisposition: "shown"}, []protocol.VoiceTranscriptDirective{dir}, gatheredIn, pending, true, ""
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
				Path     string `json:"path"`
				Position struct {
					Line      int64 `json:"line"`
					Character int64 `json:"character"`
				} `json:"position"`
				NewName string `json:"newName"`
			}{
				Path: filepath.Clean(active),
				Position: struct {
					Line      int64 `json:"line"`
					Character int64 `json:"character"`
				}{Line: params.CursorPosition.Line, Character: params.CursorPosition.Character},
				NewName: newName,
			},
		}
		pending := &agentcontext.DirectiveApplyBatch{ID: batchID, NumDirectives: 1}
		return protocol.VoiceTranscriptCompletion{Success: true, Summary: "rename", UiDisposition: "shown"}, []protocol.VoiceTranscriptDirective{dir}, agentcontext.SeedGatheredActiveFile(gatheredIn, active), pending, true, ""
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
		targetRes := agentcontext.ClarifyTargetInstruction
		if clarifyParentFlow == agentcontext.FlowKindSelection {
			targetRes = agentcontext.ClarifyTargetEdit
		}
		if err := agentcontext.ValidateClarifyTargetResolution(clarifyParentFlow, targetRes); err != nil {
			return protocol.VoiceTranscriptCompletion{Success: false}, nil, g, nil, true, err.Error()
		}
		return protocol.VoiceTranscriptCompletion{
			Success:                 true,
			Summary:                 q,
			TranscriptOutcome:       "clarify",
			UiDisposition:           "hidden",
			ClarifyTargetResolution: targetRes,
		}, nil, g, nil, true, ""
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
					NewText:        out.ReplacementText,
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
	return protocol.VoiceTranscriptCompletion{Success: true, Summary: "scoped edit", UiDisposition: "shown"}, []protocol.VoiceTranscriptDirective{dir}, g, pending, true, ""
}
