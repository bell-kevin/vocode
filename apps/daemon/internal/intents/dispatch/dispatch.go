package dispatch

import (
	"fmt"

	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch/command"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch/edit"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch/navigation"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch/undo"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Handler routes executable planner intents to per-kind handlers and produces protocol directives
// for the extension.
type Handler struct {
	engine *edit.Engine
}

func NewHandler(editEngine *edit.Engine) *Handler {
	return &Handler{engine: editEngine}
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
func (h *Handler) DispatchIntent(next intents.Intent, editCtx edit.EditExecutionContext) (DispatchResult, error) {
	if err := intents.ValidateIntent(next); err != nil {
		return DispatchResult{}, err
	}
	switch next.Kind {
	case intents.IntentKindEdit:
		res, err := h.engine.DispatchEdit(editCtx, *next.Edit)
		if err != nil {
			return DispatchResult{}, fmt.Errorf("intent edit: %w", err)
		}
		return DispatchResult{EditDirective: &res}, nil
	case intents.IntentKindCommand:
		res, err := command.DispatchCommand(*next.Command)
		if err != nil {
			return DispatchResult{}, fmt.Errorf("intent command: %w", err)
		}
		return DispatchResult{CommandDirective: &res}, nil
	case intents.IntentKindNavigate:
		res, err := navigation.DispatchNavigation(*next.Navigate)
		if err != nil {
			return DispatchResult{}, fmt.Errorf("intent navigate: %w", err)
		}
		return DispatchResult{NavigationDirective: &res}, nil
	case intents.IntentKindUndo:
		res, err := undo.DispatchUndo(*next.Undo)
		if err != nil {
			return DispatchResult{}, fmt.Errorf("intent undo: %w", err)
		}
		return DispatchResult{UndoDirective: &res}, nil
	case intents.IntentKindDone:
		return DispatchResult{}, fmt.Errorf("done is not executable")
	case intents.IntentKindRequestContext:
		return DispatchResult{}, fmt.Errorf("request_context is not executable")
	default:
		return DispatchResult{}, fmt.Errorf("unknown intent kind %q", next.Kind)
	}
}
