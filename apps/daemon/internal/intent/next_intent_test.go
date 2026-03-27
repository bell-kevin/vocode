package intent

import "testing"

func TestValidateNextIntentDone(t *testing.T) {
	t.Parallel()
	if err := ValidateNextIntent(NextIntent{Kind: NextIntentKindDone}); err != nil {
		t.Fatalf("expected done intent to be valid: %v", err)
	}
}
