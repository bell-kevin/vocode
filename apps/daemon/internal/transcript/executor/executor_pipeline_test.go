package executor_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/executor"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type fakeModel struct {
	classifier agent.TranscriptClassifierResult
	scope agent.ScopeIntentResult
	edit  agent.ScopedEditResult
}

func (f fakeModel) ClassifyTranscript(ctx context.Context, in agentcontext.TranscriptClassifierContext) (agent.TranscriptClassifierResult, error) {
	_ = ctx
	_ = in
	if f.classifier.Kind == "" {
		return agent.TranscriptClassifierResult{Kind: agent.TranscriptInstruction}, nil
	}
	return f.classifier, nil
}

func (f fakeModel) ScopeIntent(ctx context.Context, in agentcontext.ScopeIntentContext) (agent.ScopeIntentResult, error) {
	_ = ctx
	_ = in
	return f.scope, nil
}

func (f fakeModel) ScopedEdit(ctx context.Context, in agentcontext.ScopedEditContext) (agent.ScopedEditResult, error) {
	_ = ctx
	_ = in
	return f.edit, nil
}

func TestExecutor_ScopedEdit_CurrentFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	active := filepath.Join(dir, "a.ts")
	if err := os.WriteFile(active, []byte("line0\nline1\nline2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := agent.New(fakeModel{
		classifier: agent.TranscriptClassifierResult{Kind: agent.TranscriptInstruction},
		scope: agent.ScopeIntentResult{ScopeKind: agent.ScopeCurrentFile},
		edit:  agent.ScopedEditResult{ReplacementText: "REPLACED\n"},
	})
	ex := executor.New(a, executor.Options{})
	res, dirs, _, pending, ok, reason := ex.Execute(protocol.VoiceTranscriptParams{
		Text:          "replace whole file",
		ActiveFile:    active,
		WorkspaceRoot: dir,
	}, agentcontext.Gathered{})
	if !ok || !res.Success || reason != "" {
		t.Fatalf("expected success, got ok=%v success=%v reason=%q", ok, res.Success, reason)
	}
	if pending == nil || pending.NumDirectives != 1 {
		t.Fatalf("expected pending batch of 1")
	}
	if len(dirs) != 1 || dirs[0].Kind != "edit" || dirs[0].EditDirective == nil {
		t.Fatalf("expected single edit directive")
	}
	act := dirs[0].EditDirective.Actions[0]
	if act.Kind != "replace_range" || act.Range == nil {
		t.Fatalf("expected replace_range action")
	}
	if act.Range.StartLine != 0 || act.Range.StartChar != 0 {
		t.Fatalf("expected range start at 0,0; got %+v", *act.Range)
	}
	if act.ExpectedSha256 == "" {
		t.Fatalf("expected expectedSha256 to be set")
	}
}

func TestExecutor_ScopedEdit_NamedSymbol_PicksSmallestRange(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	active := filepath.Join(dir, "b.ts")
	src := "l0\nl1\nl2\nl3\nl4\n"
	if err := os.WriteFile(active, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	a := agent.New(fakeModel{
		classifier: agent.TranscriptClassifierResult{Kind: agent.TranscriptInstruction},
		scope: agent.ScopeIntentResult{ScopeKind: agent.ScopeNamedSymbol, SymbolName: "foo"},
		edit:  agent.ScopedEditResult{ReplacementText: "X\n"},
	})
	ex := executor.New(a, executor.Options{})
	res, dirs, _, _, ok, reason := ex.Execute(protocol.VoiceTranscriptParams{
		Text:          "edit foo",
		ActiveFile:    active,
		WorkspaceRoot: dir,
		ActiveFileSymbols: []struct {
			Name string `json:"name"`
			Kind string `json:"kind"`
			Range struct {
				StartLine int64 `json:"startLine"`
				StartChar int64 `json:"startChar"`
				EndLine   int64 `json:"endLine"`
				EndChar   int64 `json:"endChar"`
			} `json:"range"`
			SelectionRange struct {
				StartLine int64 `json:"startLine"`
				StartChar int64 `json:"startChar"`
				EndLine   int64 `json:"endLine"`
				EndChar   int64 `json:"endChar"`
			} `json:"selectionRange"`
		}{
			// Bigger range.
			{
				Name: "foo",
				Kind: "function",
				Range: struct {
					StartLine int64 `json:"startLine"`
					StartChar int64 `json:"startChar"`
					EndLine   int64 `json:"endLine"`
					EndChar   int64 `json:"endChar"`
				}{StartLine: 1, StartChar: 0, EndLine: 4, EndChar: 0},
				SelectionRange: struct {
					StartLine int64 `json:"startLine"`
					StartChar int64 `json:"startChar"`
					EndLine   int64 `json:"endLine"`
					EndChar   int64 `json:"endChar"`
				}{StartLine: 1, StartChar: 0, EndLine: 1, EndChar: 1},
			},
			// Smaller range should win.
			{
				Name: "foo",
				Kind: "function",
				Range: struct {
					StartLine int64 `json:"startLine"`
					StartChar int64 `json:"startChar"`
					EndLine   int64 `json:"endLine"`
					EndChar   int64 `json:"endChar"`
				}{StartLine: 2, StartChar: 0, EndLine: 3, EndChar: 0},
				SelectionRange: struct {
					StartLine int64 `json:"startLine"`
					StartChar int64 `json:"startChar"`
					EndLine   int64 `json:"endLine"`
					EndChar   int64 `json:"endChar"`
				}{StartLine: 2, StartChar: 0, EndLine: 2, EndChar: 1},
			},
		},
	}, agentcontext.Gathered{})
	if !ok || !res.Success || reason != "" {
		t.Fatalf("expected success, got ok=%v success=%v reason=%q", ok, res.Success, reason)
	}
	act := dirs[0].EditDirective.Actions[0]
	if act.Range == nil {
		t.Fatalf("expected range")
	}
	if act.Range.StartLine != 2 || act.Range.EndLine != 3 {
		t.Fatalf("expected smallest foo range (2..3), got %+v", *act.Range)
	}
}

func TestExecutor_RenameHeuristic_ProducesRenameDirective(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	active := filepath.Join(dir, "c.ts")
	if err := os.WriteFile(active, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := agent.New(fakeModel{
		classifier: agent.TranscriptClassifierResult{Kind: agent.TranscriptInstruction},
		scope: agent.ScopeIntentResult{ScopeKind: agent.ScopeCurrentFile},
		edit:  agent.ScopedEditResult{ReplacementText: "x\n"},
	})
	ex := executor.New(a, executor.Options{})
	res, dirs, _, pending, ok, reason := ex.Execute(protocol.VoiceTranscriptParams{
		Text:          "rename foo to bar",
		ActiveFile:    active,
		WorkspaceRoot: dir,
		CursorPosition: &struct {
			Line      int64 `json:"line"`
			Character int64 `json:"character"`
		}{Line: 0, Character: 0},
	}, agentcontext.Gathered{})
	if !ok || !res.Success || reason != "" {
		t.Fatalf("expected success, got ok=%v success=%v reason=%q", ok, res.Success, reason)
	}
	if pending == nil || pending.NumDirectives != 1 {
		t.Fatalf("expected pending batch 1")
	}
	if len(dirs) != 1 || dirs[0].Kind != "rename" || dirs[0].RenameDirective == nil {
		t.Fatalf("expected rename directive")
	}
	if dirs[0].RenameDirective.NewName != "bar" {
		t.Fatalf("expected newName=bar, got %q", dirs[0].RenameDirective.NewName)
	}
}

func TestExecutor_ClassifierQuestion_ReturnsAnswerOutcome(t *testing.T) {
	t.Parallel()
	a := agent.New(fakeModel{
		classifier: agent.TranscriptClassifierResult{
			Kind:       agent.TranscriptQuestion,
			AnswerText: "Because.",
		},
		scope: agent.ScopeIntentResult{ScopeKind: agent.ScopeCurrentFile},
		edit:  agent.ScopedEditResult{ReplacementText: "x\n"},
	})
	ex := executor.New(a, executor.Options{})
	res, dirs, _, pending, ok, reason := ex.Execute(protocol.VoiceTranscriptParams{
		Text:          "why",
		ActiveFile:    "c:\\fake\\file.ts",
		WorkspaceRoot: "c:\\fake",
	}, agentcontext.Gathered{})
	if !ok || !res.Success || reason != "" {
		t.Fatalf("expected success, got ok=%v success=%v reason=%q", ok, res.Success, reason)
	}
	if len(dirs) != 0 || pending != nil {
		t.Fatalf("expected no directives for answer")
	}
	if res.TranscriptOutcome != "answer" || res.AnswerText != "Because." {
		t.Fatalf("expected answer outcome, got outcome=%q answer=%q", res.TranscriptOutcome, res.AnswerText)
	}
}

