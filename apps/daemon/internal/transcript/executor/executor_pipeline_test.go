package executor_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/executor"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type fakeModel struct {
	classifier agent.TranscriptClassifierResult
	scope      agent.ScopeIntentResult
	edit       agent.ScopedEditResult
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
		scope:      agent.ScopeIntentResult{ScopeKind: agent.ScopeCurrentFile},
		edit:       agent.ScopedEditResult{ReplacementText: "REPLACED\n"},
	})
	ex := executor.New(a, executor.Options{})
	res, dirs, _, pending, ok, reason := ex.Execute(protocol.VoiceTranscriptParams{
		Text:          "replace whole file",
		ActiveFile:    active,
		WorkspaceRoot: dir,
	}, agentcontext.Gathered{}, executor.ExecuteOptions{})
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
		scope:      agent.ScopeIntentResult{ScopeKind: agent.ScopeNamedSymbol, SymbolName: "foo"},
		edit:       agent.ScopedEditResult{ReplacementText: "X\n"},
	})
	ex := executor.New(a, executor.Options{})
	res, dirs, _, _, ok, reason := ex.Execute(protocol.VoiceTranscriptParams{
		Text:          "edit foo",
		ActiveFile:    active,
		WorkspaceRoot: dir,
		ActiveFileSymbols: []struct {
			Name  string `json:"name"`
			Kind  string `json:"kind"`
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
	}, agentcontext.Gathered{}, executor.ExecuteOptions{})
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
	// EndChar must be the width of the target end line, not the file's last line.
	if want := int64(len("l3")); act.Range.EndChar != want {
		t.Fatalf("expected EndChar=%d (len of end line), got %d", want, act.Range.EndChar)
	}
}

func TestExecutor_ScopedEdit_CurrentFunction_EndCharMatchesTargetEndLine(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	active := filepath.Join(dir, "scoped-fn.ts")
	// Function `f` ends at line 3 ("}"). Last line is much longer — old bug used last-line length as EndChar.
	src := "preamble\nfunc f() {\n  return 1;\n}\nthis_is_a_much_longer_trailing_line_than_the_closing_brace\n"
	if err := os.WriteFile(active, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	a := agent.New(fakeModel{
		classifier: agent.TranscriptClassifierResult{Kind: agent.TranscriptInstruction},
		scope:      agent.ScopeIntentResult{ScopeKind: agent.ScopeCurrentFunction},
		edit:       agent.ScopedEditResult{ReplacementText: "func f() {\n  return 2;\n}\n"},
	})
	ex := executor.New(a, executor.Options{})
	res, dirs, _, _, ok, reason := ex.Execute(protocol.VoiceTranscriptParams{
		Text:          "change return in this function",
		ActiveFile:    active,
		WorkspaceRoot: dir,
		CursorPosition: &struct {
			Line      int64 `json:"line"`
			Character int64 `json:"character"`
		}{Line: 2, Character: 2},
		ActiveFileSymbols: []struct {
			Name  string `json:"name"`
			Kind  string `json:"kind"`
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
			{
				Name: "f",
				Kind: "function",
				Range: struct {
					StartLine int64 `json:"startLine"`
					StartChar int64 `json:"startChar"`
					EndLine   int64 `json:"endLine"`
					EndChar   int64 `json:"endChar"`
				}{StartLine: 1, StartChar: 0, EndLine: 3, EndChar: 1},
				SelectionRange: struct {
					StartLine int64 `json:"startLine"`
					StartChar int64 `json:"startChar"`
					EndLine   int64 `json:"endLine"`
					EndChar   int64 `json:"endChar"`
				}{StartLine: 1, StartChar: 0, EndLine: 1, EndChar: 1},
			},
		},
	}, agentcontext.Gathered{}, executor.ExecuteOptions{})
	if !ok || !res.Success || reason != "" {
		t.Fatalf("expected success, got ok=%v success=%v reason=%q", ok, res.Success, reason)
	}
	act := dirs[0].EditDirective.Actions[0]
	if act.Kind != "replace_range" || act.Range == nil {
		t.Fatalf("expected replace_range, got %+v", act)
	}
	if act.Range.StartLine != 1 || act.Range.EndLine != 3 {
		t.Fatalf("expected function lines 1..3, got %+v", *act.Range)
	}
	if want := int64(len("}")); act.Range.EndChar != want {
		t.Fatalf("expected EndChar=%d (len of closing brace line), got %d", want, act.Range.EndChar)
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
		scope:      agent.ScopeIntentResult{ScopeKind: agent.ScopeCurrentFile},
		edit:       agent.ScopedEditResult{ReplacementText: "x\n"},
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
	}, agentcontext.Gathered{}, executor.ExecuteOptions{})
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

func TestExecutor_FileSelection_RequiresWorkspaceFolder(t *testing.T) {
	t.Parallel()
	a := agent.New(fakeModel{
		classifier: agent.TranscriptClassifierResult{Kind: agent.TranscriptFileSelection},
	})
	ex := executor.New(a, executor.Options{})
	res, dirs, _, pending, ok, reason := ex.Execute(protocol.VoiceTranscriptParams{
		Text:                "delete that file",
		ActiveFile:          "c:\\fake\\file.ts",
		WorkspaceRoot:       "c:\\fake",
		WorkspaceFolderOpen: false,
	}, agentcontext.Gathered{}, executor.ExecuteOptions{})
	if !ok || !res.Success || reason != "" {
		t.Fatalf("expected success gate, got ok=%v success=%v reason=%q", ok, res.Success, reason)
	}
	if len(dirs) != 0 || pending != nil {
		t.Fatalf("expected no directives")
	}
	if res.Workspace == nil || !res.Workspace.NeedsFolder {
		t.Fatalf("expected workspace.needsFolder, got %+v", res.Workspace)
	}
}

func TestExecutor_ForceSearchQuery_skipsClassifierAndScopedEdit(t *testing.T) {
	rgPath := os.Getenv("VOCODE_RG_BIN")
	if rgPath == "" {
		var err error
		rgPath, err = exec.LookPath("rg")
		if err != nil {
			t.Skip("ripgrep not available (set VOCODE_RG_BIN or install rg): ", err)
		}
	}
	t.Setenv("VOCODE_RG_BIN", rgPath)

	dir := t.TempDir()
	active := filepath.Join(dir, "a.go")
	if err := os.WriteFile(active, []byte("func stuff() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := agent.New(fakeModel{
		classifier: agent.TranscriptClassifierResult{Kind: agent.TranscriptInstruction},
		scope:      agent.ScopeIntentResult{ScopeKind: agent.ScopeCurrentFile},
		edit:       agent.ScopedEditResult{ReplacementText: "SHOULD_NOT_APPLY\n"},
	})
	ex := executor.New(a, executor.Options{})
	res, dirs, _, pending, ok, reason := ex.Execute(protocol.VoiceTranscriptParams{
		Text:          "this text is ignored for routing",
		ActiveFile:    active,
		WorkspaceRoot: dir,
	}, agentcontext.Gathered{}, executor.ExecuteOptions{ForceSearchQuery: "NO_MATCH_TOKEN_XYZ_12345"})
	if !ok || !res.Success || reason != "" {
		t.Fatalf("expected success, got ok=%v success=%v reason=%q", ok, res.Success, reason)
	}
	// No matches avoids requiring ripgrep to parse output; still proves classifier/edit path was skipped.
	if res.Search == nil || !res.Search.NoHits {
		t.Fatalf("expected search.noHits (no hits), got %+v summary=%q", res.Search, res.Summary)
	}
	if len(dirs) != 0 {
		t.Fatalf("expected no directives for empty search, got %d", len(dirs))
	}
	if pending != nil {
		t.Fatalf("expected no pending batch for empty search")
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
	}, agentcontext.Gathered{}, executor.ExecuteOptions{})
	if !ok || !res.Success || reason != "" {
		t.Fatalf("expected success, got ok=%v success=%v reason=%q", ok, res.Success, reason)
	}
	if len(dirs) != 0 || pending != nil {
		t.Fatalf("expected no directives for answer")
	}
	if res.Question == nil || res.Question.AnswerText != "Because." {
		t.Fatalf("expected question.answerText, got %+v", res.Question)
	}
}
