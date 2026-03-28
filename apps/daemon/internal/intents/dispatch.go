package intents

import (
	"fmt"

	"vocoding.net/vocode/v2/apps/daemon/internal/intent"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/command"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/edits"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/navigation"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/undo"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Handler routes executable planner intents to per-kind handlers and produces protocol directives
// for the extension.
type Handler struct {
	edits *edits.Engine
}

func NewHandler(editEngine *edits.Engine) *Handler {
	return &Handler{edits: editEngine}
}

// DispatchResult holds at most one populated directive pointer from a successful intent dispatch.
type DispatchResult struct {
	EditDirective       *protocol.EditDirective
	CommandDirective    *protocol.CommandDirective
	NavigationDirective *protocol.NavigationDirective
	UndoDirective       *protocol.UndoDirective
}

// DispatchIntent handles only executable intents (edit/command/navigate/undo).
// Control-flow intents (done/request_context) are handled by the transcript executor.
func (h *Handler) DispatchIntent(next intent.NextIntent, editCtx edits.EditExecutionContext) (DispatchResult, error) {
	if err := intent.ValidateNextIntent(next); err != nil {
		return DispatchResult{}, err
	}
	switch next.Kind {
	case intent.NextIntentKindEdit:
		res, err := h.edits.DispatchEdit(editCtx, *next.Edit)
		if err != nil {
			return DispatchResult{}, fmt.Errorf("next intent edit: %w", err)
		}
		return DispatchResult{EditDirective: &res}, nil
	case intent.NextIntentKindCommand:
		res, err := command.DispatchCommand(*next.Command)
		if err != nil {
			return DispatchResult{}, fmt.Errorf("next intent command: %w", err)
		}
		return DispatchResult{CommandDirective: &res}, nil
	case intent.NextIntentKindNavigate:
		res, err := navigation.DispatchNavigation(*next.Navigate)
		if err != nil {
			return DispatchResult{}, fmt.Errorf("next intent navigate: %w", err)
		}
		return DispatchResult{NavigationDirective: &res}, nil
	case intent.NextIntentKindUndo:
		res, err := undo.DispatchUndo(*next.Undo)
		if err != nil {
			return DispatchResult{}, fmt.Errorf("next intent undo: %w", err)
		}
		return DispatchResult{UndoDirective: &res}, nil
	case intent.NextIntentKindDone:
		return DispatchResult{}, fmt.Errorf("done is not executable")
	case intent.NextIntentKindRequestContext:
		return DispatchResult{}, fmt.Errorf("request_context is not executable")
	default:
		return DispatchResult{}, fmt.Errorf("unknown next intent kind %q", next.Kind)
	}
}
