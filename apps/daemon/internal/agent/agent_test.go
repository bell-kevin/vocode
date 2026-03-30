package agent_test

import (
	"context"
	"testing"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/agent/stub"
	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
)

func TestStubBatchTurn(t *testing.T) {
	t.Parallel()

	a := agent.New(stub.New())
	in := agentcontext.TurnContext{TranscriptText: "hello"}

	turn, err := a.NextTurn(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if err := turn.Validate(); err != nil {
		t.Fatalf("invalid turn: %v", err)
	}
	if turn.Kind != agent.TurnIntents {
		t.Fatalf("expected TurnIntents, got %q", turn.Kind)
	}
	if len(turn.Intents) != 4 {
		t.Fatalf("expected 4 intents, got %d", len(turn.Intents))
	}

	in.IntentApplyHistory = []agentcontext.IntentApplyRecord{{BatchOrdinal: 1, IndexInBatch: 0}}
	turn2, err := a.NextTurn(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if turn2.Kind != agent.TurnDone {
		t.Fatalf("expected TurnDone after apply history, got %q", turn2.Kind)
	}
}
