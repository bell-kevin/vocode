package intents

import (
	"fmt"
	"strings"
)

// UndoScope selects how the host should undo recent voice-driven work.
type UndoScope string

const (
	UndoScopeLastEdit       UndoScope = "last_edit"
	UndoScopeLastTranscript UndoScope = "last_transcript"
)

// UndoIntent is emitted when the user asks to revert prior changes (host applies).
type UndoIntent struct {
	Scope UndoScope `json:"scope"`
}

// ValidateUndoIntent checks wire-level scope values only.
func ValidateUndoIntent(u UndoIntent) error {
	switch UndoScope(strings.TrimSpace(string(u.Scope))) {
	case UndoScopeLastEdit, UndoScopeLastTranscript:
		return nil
	default:
		return fmt.Errorf("next intent: undo.scope must be last_edit or last_transcript, got %q", u.Scope)
	}
}
