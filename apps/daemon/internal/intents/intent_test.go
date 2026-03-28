package intents

import (
	"encoding/json"
	"testing"
)

func TestValidateIntentDone(t *testing.T) {
	t.Parallel()
	if err := ControlDone().Validate(); err != nil {
		t.Fatalf("expected done to be valid: %v", err)
	}
}

func TestValidateIntentUndo(t *testing.T) {
	t.Parallel()
	err := FromExecutable(ExecutableIntent{
		Kind: ExecutableIntentKindUndo,
		Undo: &UndoIntent{Scope: UndoScopeLastTranscript},
	}).Validate()
	if err != nil {
		t.Fatalf("expected undo to be valid: %v", err)
	}
}

func TestIntentJSONRoundTrip(t *testing.T) {
	t.Parallel()
	cases := []Intent{
		ControlDone(),
		FromExecutable(ExecutableIntent{
			Kind: ExecutableIntentKindCommand,
			Command: &CommandIntent{
				Command: "echo",
				Args:    []string{"hi"},
			},
		}),
	}
	for _, want := range cases {
		data, err := json.Marshal(want)
		if err != nil {
			t.Fatalf("marshal %+v: %v", want, err)
		}
		var got Intent
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal %s: %v", string(data), err)
		}
		if err := got.Validate(); err != nil {
			t.Fatalf("validate after round-trip: %v", err)
		}
		if got.Summary() != want.Summary() {
			t.Fatalf("summary: got %q want %q (json=%s)", got.Summary(), want.Summary(), string(data))
		}
	}
}
