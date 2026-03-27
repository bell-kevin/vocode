package agent_test

import (
	"context"
	"testing"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agent/stub"
	"vocoding.net/vocode/v2/apps/daemon/internal/intent"
)

func TestNextIntentStubFlow(t *testing.T) {
	t.Parallel()

	a := agent.New(stub.New())
	in := agent.ModelInput{Transcript: "hello"}

	for i := 0; i < 4; i++ {
		next, err := a.NextIntent(context.Background(), in)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if err := intent.ValidateNextIntent(next); err != nil {
			t.Fatalf("invalid next action: %v", err)
		}
		if next.Kind == intent.NextIntentKindDone {
			t.Fatal("unexpected done before 4th step")
		}
		in.CompletedActions = append(in.CompletedActions, next)
	}

	final, err := a.NextIntent(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if final.Kind != intent.NextIntentKindDone {
		t.Fatalf("expected done, got %q", final.Kind)
	}
}
