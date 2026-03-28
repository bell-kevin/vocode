package dispatch

import (
	"fmt"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch/command"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch/edit"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch/navigation"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch/requestcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch/undo"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Handler routes planner intents: control intents (done / request_context) vs executables
// (edit / command / navigate / undo → protocol directives).
type Handler struct {
	engine  *edit.Engine
	request *requestcontext.Provider
}

func NewHandler(editEngine *edit.Engine, request *requestcontext.Provider) *Handler {
	return &Handler{engine: editEngine, request: request}
}

// DispatchResult holds at most one populated directive pointer from a successful executable dispatch.
type DispatchResult struct {
	EditDirective       *protocol.EditDirective
	CommandDirective    *protocol.CommandDirective
	NavigationDirective *protocol.NavigationDirective
	UndoDirective       *protocol.UndoDirective
}

// HandleOutcomeKind classifies the result of [Handler.Handle].
type HandleOutcomeKind int8

const (
	OutcomeDone HandleOutcomeKind = iota
	OutcomeRequestContextFulfilled
	OutcomeExecutableDispatched
)

// HandleOutcome is returned by [Handler.Handle] for control vs executable paths.
type HandleOutcome struct {
	Kind            HandleOutcomeKind
	PlanningContext agent.PlanningContext // set when Kind == OutcomeRequestContextFulfilled
	Dispatch        DispatchResult        // set when Kind == OutcomeExecutableDispatched
}

// HandleInput carries transcript + planner state for one [Handler.Handle] call.
type HandleInput struct {
	Params  protocol.VoiceTranscriptParams
	TurnCtx agent.PlanningContext
	Intent  intents.Intent
	EditCtx edit.EditExecutionContext
}

// Handle validates the union and dispatches control intents vs executables.
func (h *Handler) Handle(in HandleInput) (HandleOutcome, error) {
	if err := in.Intent.Validate(); err != nil {
		return HandleOutcome{}, err
	}
	if c := in.Intent.Control; c != nil {
		switch c.Kind {
		case intents.ControlIntentKindDone:
			return HandleOutcome{Kind: OutcomeDone}, nil
		case intents.ControlIntentKindRequestContext:
			if h.request == nil {
				return HandleOutcome{}, fmt.Errorf("request_context: provider not configured")
			}
			updated, err := h.request.Fulfill(in.Params, in.TurnCtx, c.RequestContext)
			if err != nil {
				return HandleOutcome{}, err
			}
			return HandleOutcome{Kind: OutcomeRequestContextFulfilled, PlanningContext: updated}, nil
		default:
			return HandleOutcome{}, fmt.Errorf("unknown control intent kind %q", c.Kind)
		}
	}
	ex := in.Intent.Executable
	if ex == nil {
		return HandleOutcome{}, fmt.Errorf("planner intent: missing executable")
	}
	dr, err := h.dispatchExecutable(ex, in.EditCtx)
	if err != nil {
		return HandleOutcome{}, err
	}
	return HandleOutcome{Kind: OutcomeExecutableDispatched, Dispatch: dr}, nil
}

func (h *Handler) dispatchExecutable(ex *intents.ExecutableIntent, editCtx edit.EditExecutionContext) (DispatchResult, error) {
	switch ex.Kind {
	case intents.ExecutableIntentKindEdit:
		res, err := h.engine.DispatchEdit(editCtx, *ex.Edit)
		if err != nil {
			return DispatchResult{}, fmt.Errorf("intent edit: %w", err)
		}
		return DispatchResult{EditDirective: &res}, nil
	case intents.ExecutableIntentKindCommand:
		res, err := command.DispatchCommand(*ex.Command)
		if err != nil {
			return DispatchResult{}, fmt.Errorf("intent command: %w", err)
		}
		return DispatchResult{CommandDirective: &res}, nil
	case intents.ExecutableIntentKindNavigate:
		res, err := navigation.DispatchNavigation(*ex.Navigate)
		if err != nil {
			return DispatchResult{}, fmt.Errorf("intent navigate: %w", err)
		}
		return DispatchResult{NavigationDirective: &res}, nil
	case intents.ExecutableIntentKindUndo:
		res, err := undo.DispatchUndo(*ex.Undo)
		if err != nil {
			return DispatchResult{}, fmt.Errorf("intent undo: %w", err)
		}
		return DispatchResult{UndoDirective: &res}, nil
	default:
		return DispatchResult{}, fmt.Errorf("unknown executable intent kind %q", ex.Kind)
	}
}
