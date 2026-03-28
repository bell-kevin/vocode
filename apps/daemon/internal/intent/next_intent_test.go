package intent

import "testing"

func TestValidateNextIntentDone(t *testing.T) {
	t.Parallel()
	if err := ValidateNextIntent(NextIntent{Kind: NextIntentKindDone}); err != nil {
		t.Fatalf("expected done intent to be valid: %v", err)
	}
}

func TestValidateNextIntentUndo(t *testing.T) {
	t.Parallel()
	err := ValidateNextIntent(NextIntent{
		Kind: NextIntentKindUndo,
		Undo: &UndoIntent{Scope: UndoScopeLastTranscript},
	})
	if err != nil {
		t.Fatalf("expected undo intent to be valid: %v", err)
	}
}
