package intents

import "testing"

func TestValidateIntentDone(t *testing.T) {
	t.Parallel()
	if err := ValidateIntent(Intent{Kind: IntentKindDone}); err != nil {
		t.Fatalf("expected done intent to be valid: %v", err)
	}
}

func TestValidateIntentUndo(t *testing.T) {
	t.Parallel()
	err := ValidateIntent(Intent{
		Kind: IntentKindUndo,
		Undo: &UndoIntent{Scope: UndoScopeLastTranscript},
	})
	if err != nil {
		t.Fatalf("expected undo intent to be valid: %v", err)
	}
}
