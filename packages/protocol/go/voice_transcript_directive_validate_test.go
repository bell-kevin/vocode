package protocol

import "testing"

func TestVoiceTranscriptDirectiveUndoValidates(t *testing.T) {
	t.Parallel()
	d := VoiceTranscriptDirective{
		Kind:          "undo",
		UndoDirective: &UndoDirective{Scope: "last_transcript"},
	}
	if err := d.Validate(); err != nil {
		t.Fatalf("expected valid undo directive: %v", err)
	}
}

func TestVoiceTranscriptDirectiveUndoRejectsBadScope(t *testing.T) {
	t.Parallel()
	d := VoiceTranscriptDirective{
		Kind:          "undo",
		UndoDirective: &UndoDirective{Scope: "nope"},
	}
	if err := d.Validate(); err == nil {
		t.Fatal("expected invalid scope error")
	}
}
