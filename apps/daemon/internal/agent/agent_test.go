package agent_test

import (
	"context"
	"testing"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agent/stub"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
)

func TestIntentStubFlow(t *testing.T) {
	t.Parallel()

	a := agent.New(stub.New())
	in := agent.ModelInput{Transcript: "hello"}

	for i := 0; i < 4; i++ {
		next, err := a.NextIntent(context.Background(), in)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if err := intents.ValidateIntent(next); err != nil {
			t.Fatalf("invalid next intent: %v", err)
		}
		if next.Kind == intents.IntentKindDone {
			t.Fatal("unexpected done before 4th step")
		}
		in.CompletedActions = append(in.CompletedActions, next)
	}

	final, err := a.NextIntent(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if final.Kind != intents.IntentKindDone {
		t.Fatalf("expected done, got %q", final.Kind)
	}
}
