package intent

import (
	"fmt"
	"strings"
)

type NextIntentKind string

const (
	NextIntentKindEdit           NextIntentKind = "edit"
	NextIntentKindCommand        NextIntentKind = "command"
	NextIntentKindNavigate       NextIntentKind = "navigate"
	NextIntentKindUndo           NextIntentKind = "undo"
	NextIntentKindRequestContext NextIntentKind = "request_context"
	NextIntentKindDone           NextIntentKind = "done"
)

type NextIntent struct {
	Kind           NextIntentKind        `json:"kind"`
	Edit           *EditIntent           `json:"edit,omitempty"`
	Command        *CommandIntent        `json:"command,omitempty"`
	Navigate       *NavigationIntent     `json:"navigate,omitempty"`
	Undo           *UndoIntent           `json:"undo,omitempty"`
	RequestContext *RequestContextIntent `json:"requestContext,omitempty"`
}

func ValidateNextIntent(a NextIntent) error {
	switch a.Kind {
	case NextIntentKindEdit:
		if a.Edit == nil {
			return fmt.Errorf("next intent: kind %q requires edit", a.Kind)
		}
		return ValidateEditIntent(*a.Edit)
	case NextIntentKindCommand:
		if a.Command == nil {
			return fmt.Errorf("next intent: kind %q requires command", a.Kind)
		}
		if strings.TrimSpace(a.Command.Command) == "" {
			return fmt.Errorf("next intent: command.command is empty")
		}
		return nil
	case NextIntentKindNavigate:
		if a.Navigate == nil {
			return fmt.Errorf("next intent: kind %q requires navigate", a.Kind)
		}
		return ValidateNavigationIntent(*a.Navigate)
	case NextIntentKindUndo:
		if a.Undo == nil {
			return fmt.Errorf("next intent: kind %q requires undo", a.Kind)
		}
		return ValidateUndoIntent(*a.Undo)
	case NextIntentKindRequestContext:
		if a.RequestContext == nil {
			return fmt.Errorf("next intent: kind %q requires requestContext", a.Kind)
		}
		switch a.RequestContext.Kind {
		case RequestContextKindSymbols, RequestContextKindFileExcerpt, RequestContextKindUsages:
			return nil
		default:
			return fmt.Errorf("next intent: unknown requestContext kind %q", a.RequestContext.Kind)
		}
	case NextIntentKindDone:
		return nil
	default:
		return fmt.Errorf("next intent: unknown kind %q", a.Kind)
	}
}
