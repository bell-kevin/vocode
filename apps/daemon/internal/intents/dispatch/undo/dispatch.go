package undo

import (
	"fmt"

	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// DispatchUndo maps a planner undo intent to the wire payload for the extension (host applies).
func DispatchUndo(u intents.UndoIntent) (protocol.UndoDirective, error) {
	if err := intents.ValidateUndoIntent(u); err != nil {
		return protocol.UndoDirective{}, fmt.Errorf("undo intent: %w", err)
	}
	return protocol.UndoDirective{Scope: string(u.Scope)}, nil
}
