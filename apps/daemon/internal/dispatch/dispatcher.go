package dispatch

import (
	"fmt"

	"vocoding.net/vocode/v2/apps/daemon/internal/commandexec"
	"vocoding.net/vocode/v2/apps/daemon/internal/edits"
	"vocoding.net/vocode/v2/apps/daemon/internal/intent"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

type Dispatcher struct {
	edits    *edits.Service
	commands *commandexec.Service
}

func NewDispatcher(editsService *edits.Service, commandService *commandexec.Service) *Dispatcher {
	return &Dispatcher{edits: editsService, commands: commandService}
}

type StepResult struct {
	EditResult    *protocol.EditApplyResult
	CommandParams *protocol.CommandRunParams
	Navigation    *intent.NavigationIntent
}

func (d *Dispatcher) ExecuteNextIntent(next intent.NextIntent, editCtx edits.EditExecutionContext) (StepResult, error) {
	if err := intent.ValidateNextIntent(next); err != nil {
		return StepResult{}, err
	}
	switch next.Kind {
	case intent.NextIntentKindEdit:
		res, err := d.edits.ApplyIntent(editCtx, *next.Edit)
		if err != nil {
			return StepResult{}, fmt.Errorf("next intent edit: %w", err)
		}
		return StepResult{EditResult: &res}, nil
	case intent.NextIntentKindRunCommand:
		params := next.RunCommand.CommandParams()
		if d.commands != nil {
			if err := d.commands.Validate(params); err != nil {
				return StepResult{}, fmt.Errorf("next intent run_command: %w", err)
			}
		}
		return StepResult{CommandParams: &params}, nil
	case intent.NextIntentKindNavigate:
		return StepResult{Navigation: next.Navigate}, nil
	case intent.NextIntentKindDone:
		return StepResult{}, fmt.Errorf("done is not executable")
	case intent.NextIntentKindRequestContext:
		return StepResult{}, fmt.Errorf("request_context is not executable")
	default:
		return StepResult{}, fmt.Errorf("unknown next intent kind %q", next.Kind)
	}
}
