package dispatch

import (
	"fmt"

	"vocoding.net/vocode/v2/apps/daemon/internal/commandexec"
	"vocoding.net/vocode/v2/apps/daemon/internal/edits"
	"vocoding.net/vocode/v2/apps/daemon/internal/intent"
	"vocoding.net/vocode/v2/apps/daemon/internal/navigation"
	"vocoding.net/vocode/v2/apps/daemon/internal/undo"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type Dispatcher struct {
	edits    *edits.Service
	commands *commandexec.Service
	nav      *navigation.Service
	undo     *undo.Service
}

func NewDispatcher(
	editsService *edits.Service,
	commandService *commandexec.Service,
	navigationService *navigation.Service,
	undoService *undo.Service,
) *Dispatcher {
	return &Dispatcher{
		edits:    editsService,
		commands: commandService,
		nav:      navigationService,
		undo:     undoService,
	}
}

type DispatchResult struct {
	EditDirective       *protocol.EditDirective
	CommandDirective    *protocol.CommandDirective
	NavigationDirective *protocol.NavigationDirective
	UndoDirective       *protocol.UndoDirective
}

// DispatchExecutableIntent handles only executable intents (edit/command/navigate/undo).
// Control-flow intents (done/request_context) are handled by the intent loop.
func (d *Dispatcher) DispatchExecutableIntent(next intent.NextIntent, editCtx edits.EditExecutionContext) (DispatchResult, error) {
	if err := intent.ValidateNextIntent(next); err != nil {
		return DispatchResult{}, err
	}
	switch next.Kind {
	case intent.NextIntentKindEdit:
		res, err := d.edits.DispatchIntent(editCtx, *next.Edit)
		if err != nil {
			return DispatchResult{}, fmt.Errorf("next intent edit: %w", err)
		}
		return DispatchResult{EditDirective: &res}, nil
	case intent.NextIntentKindCommand:
		res, err := d.commands.DispatchIntent(*next.Command)
		if err != nil {
			return DispatchResult{}, fmt.Errorf("next intent command: %w", err)
		}
		return DispatchResult{CommandDirective: &res}, nil
	case intent.NextIntentKindNavigate:
		res, err := d.nav.DispatchIntent(*next.Navigate)
		if err != nil {
			return DispatchResult{}, fmt.Errorf("next intent navigate: %w", err)
		}
		return DispatchResult{NavigationDirective: &res}, nil
	case intent.NextIntentKindUndo:
		res, err := d.undo.DispatchIntent(*next.Undo)
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
