package executor_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/gather"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch/edit"
	"vocoding.net/vocode/v2/apps/daemon/internal/symbols"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/executor"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// seqTurnClient returns scripted [agent.TurnResult] values in order.
type seqTurnClient struct {
	replies []agent.TurnResult
	i       int
}

func (s *seqTurnClient) NextTurn(ctx context.Context, in agentcontext.TurnContext) (agent.TurnResult, error) {
	_ = ctx
	_ = in
	if s.i >= len(s.replies) {
		return agent.TurnResult{}, context.Canceled
	}
	r := s.replies[s.i]
	s.i++
	return r, nil
}

func newTestExecutor(t *testing.T, model agent.ModelClient) *executor.Executor {
	t.Helper()
	sym := symbols.NewTreeSitterResolver()
	h := dispatch.NewHandler(edit.NewEngine())
	return executor.New(agent.New(model), h, gather.NewProvider(sym), executor.Options{
		MaxAgentTurns:            8,
		MaxIntentRetries:         2,
		MaxContextRounds:         4,
		MaxContextBytes:          50_000,
		MaxConsecutiveContextReq: 6,
		MaxIntentsPerBatch:       16,
		Symbols:                  sym,
	})
}

func TestExecuteIrrelevant(t *testing.T) {
	t.Parallel()
	ex := newTestExecutor(t, &seqTurnClient{replies: []agent.TurnResult{
		{Kind: agent.TurnIrrelevant, IrrelevantReason: "not a coding request"},
	}})
	params := protocol.VoiceTranscriptParams{Text: "hello weather"}
	res, dirs, _, pending, ok, reason := ex.Execute(params, agentcontext.Gathered{}, nil, nil, nil, nil)
	if !ok || !res.Success || len(dirs) != 0 || pending != nil {
		t.Fatalf("got ok=%v success=%v dirs=%d pending=%v reason=%q", ok, res.Success, len(dirs), pending, reason)
	}
	if res.Summary != "not a coding request" {
		t.Fatalf("summary %q", res.Summary)
	}
	if res.TranscriptOutcome != "irrelevant" {
		t.Fatalf("transcriptOutcome %q", res.TranscriptOutcome)
	}
}

func TestExecuteDone(t *testing.T) {
	t.Parallel()
	ex := newTestExecutor(t, &seqTurnClient{replies: []agent.TurnResult{
		{Kind: agent.TurnFinish, FinishSummary: "all set"},
	}})
	params := protocol.VoiceTranscriptParams{Text: "thanks"}
	res, dirs, _, pending, ok, reason := ex.Execute(params, agentcontext.Gathered{}, nil, nil, nil, nil)
	if !ok || !res.Success || len(dirs) != 0 || pending != nil {
		t.Fatalf("got ok=%v success=%v pending=%v reason=%q res=%+v", ok, res.Success, pending != nil, reason, res)
	}
	if res.Summary != "all set" {
		t.Fatalf("summary %q", res.Summary)
	}
	if res.TranscriptOutcome != "" {
		t.Fatalf("expected empty transcriptOutcome, got %q", res.TranscriptOutcome)
	}
}

func TestExecuteGatherContextThenCommand(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "a.go")
	if err := os.WriteFile(srcPath, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ex := newTestExecutor(t, &seqTurnClient{replies: []agent.TurnResult{
		{Kind: agent.TurnGatherContext, GatherContext: &agentcontext.GatherContextSpec{
			Kind: agentcontext.GatherContextKindFileExcerpt,
			Path: "a.go",
		}},
		{Kind: agent.TurnIntents, Intents: []intents.Intent{
			{
				Kind:    intents.IntentKindCommand,
				Command: echoCommandForTest(),
			},
		}},
	}})
	params := protocol.VoiceTranscriptParams{Text: "run echo", WorkspaceRoot: dir}
	res, dirs, _, pending, ok, reason := ex.Execute(params, agentcontext.Gathered{}, nil, nil, nil, nil)
	if !ok || !res.Success || len(dirs) != 1 || pending == nil {
		t.Fatalf("ok=%v success=%v dirs=%d pending=%v reason=%q", ok, res.Success, len(dirs), pending != nil, reason)
	}
	if dirs[0].Kind != "command" {
		t.Fatalf("kind %q", dirs[0].Kind)
	}
}

func TestExecuteMultiExecutableBatch(t *testing.T) {
	t.Parallel()
	ex := newTestExecutor(t, &seqTurnClient{replies: []agent.TurnResult{
		{Kind: agent.TurnIntents, Intents: []intents.Intent{
			{
				Kind:    intents.IntentKindCommand,
				Command: echoCommandForTest(),
			},
			{
				Kind:    intents.IntentKindCommand,
				Command: echoCommandForTest(),
			},
		}},
	}})
	params := protocol.VoiceTranscriptParams{Text: "twice", WorkspaceRoot: t.TempDir()}
	res, dirs, _, pending, ok, reason := ex.Execute(params, agentcontext.Gathered{}, nil, nil, nil, nil)
	if !ok || !res.Success || len(dirs) != 2 || pending == nil {
		t.Fatalf("ok=%v success=%v dirs=%d pending=%v reason=%q", ok, res.Success, len(dirs), pending != nil, reason)
	}
}

func TestExecuteRetryAfterDispatchFailure(t *testing.T) {
	t.Parallel()
	ex := newTestExecutor(t, &seqTurnClient{replies: []agent.TurnResult{
		{Kind: agent.TurnIntents, Intents: []intents.Intent{
			{
				Kind:    intents.IntentKindCommand,
				Command: &intents.CommandIntent{Command: "disallowed-cmd", Args: []string{}},
			},
		}},
		{Kind: agent.TurnIntents, Intents: []intents.Intent{
			{
				Kind:    intents.IntentKindCommand,
				Command: echoCommandForTest(),
			},
		}},
	}})
	// First command fails daemon allowlist; second succeeds after retry.
	params := protocol.VoiceTranscriptParams{Text: "fix it", WorkspaceRoot: t.TempDir()}
	res, dirs, _, pending, ok, reason := ex.Execute(params, agentcontext.Gathered{}, nil, nil, nil, nil)
	if !ok || !res.Success || len(dirs) != 1 {
		t.Fatalf("ok=%v success=%v dirs=%d reason=%q", ok, res.Success, len(dirs), reason)
	}
	if dirs[0].Kind != "command" {
		t.Fatalf("expected command after retry, got %q", dirs[0].Kind)
	}
	if pending == nil {
		t.Fatal("expected pending batch")
	}
}

func echoCommandForTest() *intents.CommandIntent {
	if runtime.GOOS == "windows" {
		return &intents.CommandIntent{Command: "cmd.exe", Args: []string{"/c", "echo", "ok"}}
	}
	return &intents.CommandIntent{Command: "echo", Args: []string{"ok"}}
}
