package intents

import (
	"fmt"
	"strings"
)

type IntentKind string

const (
	IntentKindEdit           IntentKind = "edit"
	IntentKindCommand        IntentKind = "command"
	IntentKindNavigate       IntentKind = "navigate"
	IntentKindUndo           IntentKind = "undo"
	IntentKindRequestContext IntentKind = "request_context"
	IntentKindDone           IntentKind = "done"
)

type Intent struct {
	Kind           IntentKind            `json:"kind"`
	Edit           *EditIntent           `json:"edit,omitempty"`
	Command        *CommandIntent        `json:"command,omitempty"`
	Navigate       *NavigationIntent     `json:"navigate,omitempty"`
	Undo           *UndoIntent           `json:"undo,omitempty"`
	RequestContext *RequestContextIntent `json:"requestContext,omitempty"`
}

func ValidateIntent(a Intent) error {
	switch a.Kind {
	case IntentKindEdit:
		if a.Edit == nil {
			return fmt.Errorf("intent: kind %q requires edit", a.Kind)
		}
		return ValidateEditIntent(*a.Edit)
	case IntentKindCommand:
		if a.Command == nil {
			return fmt.Errorf("intent: kind %q requires command", a.Kind)
		}
		if strings.TrimSpace(a.Command.Command) == "" {
			return fmt.Errorf("intent: command.command is empty")
		}
		return nil
	case IntentKindNavigate:
		if a.Navigate == nil {
			return fmt.Errorf("intent: kind %q requires navigate", a.Kind)
		}
		return ValidateNavigationIntent(*a.Navigate)
	case IntentKindUndo:
		if a.Undo == nil {
			return fmt.Errorf("intent: kind %q requires undo", a.Kind)
		}
		return ValidateUndoIntent(*a.Undo)
	case IntentKindRequestContext:
		if a.RequestContext == nil {
			return fmt.Errorf("intent: kind %q requires requestContext", a.Kind)
		}
		switch a.RequestContext.Kind {
		case RequestContextKindSymbols, RequestContextKindFileExcerpt, RequestContextKindUsages:
			return nil
		default:
			return fmt.Errorf("intent: unknown requestContext kind %q", a.RequestContext.Kind)
		}
	case IntentKindDone:
		return nil
	default:
		return fmt.Errorf("intent: unknown kind %q", a.Kind)
	}
}
