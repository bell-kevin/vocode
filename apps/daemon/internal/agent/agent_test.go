package agent_test

import (
	"context"
	"testing"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agent/stub"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func TestHandleTranscriptRejectsEmpty(t *testing.T) {
	t.Parallel()
	a := agent.New(stub.New())
	r := a.HandleTranscript(context.Background(), protocol.VoiceTranscriptParams{Text: "   "})
	if r.Valid {
		t.Fatal("expected empty transcript to be invalid")
	}
}

func TestHandleTranscriptStubReturnsPlan(t *testing.T) {
	t.Parallel()
	a := agent.New(stub.New())
	r := a.HandleTranscript(context.Background(), protocol.VoiceTranscriptParams{Text: "hello"})
	if !r.Valid {
		t.Fatal("expected non-empty transcript to be valid")
	}
	if r.Err != nil {
		t.Fatalf("unexpected err: %v", r.Err)
	}
	if r.Plan == nil {
		t.Fatal("expected stub plan")
	}
	if len(r.Plan.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(r.Plan.Steps))
	}
	if r.Plan.Steps[0].Kind != agent.StepKindEdit {
		t.Fatalf("expected stub edit step first, got %q", r.Plan.Steps[0].Kind)
	}
	if r.Plan.Steps[0].Edit == nil || r.Plan.Steps[0].Edit.Kind != agent.EditIntentReplaceCurrentFunctionBody {
		t.Fatalf("expected replace_current_function_body edit, got %+v", r.Plan.Steps[0].Edit)
	}
	if r.Plan.Steps[1].Kind != agent.StepKindRunCommand {
		t.Fatalf("expected stub run_command step second, got %q", r.Plan.Steps[1].Kind)
	}
}

func TestHandleTranscriptModelError(t *testing.T) {
	t.Parallel()
	a := agent.New(modelClientFunc(func(ctx context.Context, in agent.ModelInput) (agent.ActionPlan, error) {
		return agent.ActionPlan{}, errTestModel
	}))
	r := a.HandleTranscript(context.Background(), protocol.VoiceTranscriptParams{Text: "x"})
	if !r.Valid {
		t.Fatal("expected transcript accepted before model error")
	}
	if r.Err == nil {
		t.Fatal("expected model error")
	}
	if r.Plan != nil {
		t.Fatal("expected no plan on error")
	}
}

var errTestModel = &testModelError{}

type testModelError struct{}

func (*testModelError) Error() string { return "test model error" }

type modelClientFunc func(context.Context, agent.ModelInput) (agent.ActionPlan, error)

func (f modelClientFunc) Plan(ctx context.Context, in agent.ModelInput) (agent.ActionPlan, error) {
	return f(ctx, in)
}
