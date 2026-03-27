package intent

import (
	"fmt"
	"strings"
)

type NextIntentKind string

const (
	NextIntentKindEdit           NextIntentKind = "edit"
	NextIntentKindRunCommand     NextIntentKind = "run_command"
	NextIntentKindNavigate       NextIntentKind = "navigate"
	NextIntentKindRequestContext NextIntentKind = "request_context"
	NextIntentKindDone           NextIntentKind = "done"
)

type NextIntent struct {
	Kind NextIntentKind `json:"kind"`
	Edit           *EditIntent           `json:"edit,omitempty"`
	RunCommand     *CommandIntent        `json:"runCommand,omitempty"`
	Navigate       *NavigationIntent     `json:"navigate,omitempty"`
	RequestContext *RequestContextIntent `json:"requestContext,omitempty"`
}

func ValidateNextIntent(a NextIntent) error {
	switch a.Kind {
	case NextIntentKindEdit:
		if a.Edit == nil { return fmt.Errorf("next intent: kind %q requires edit", a.Kind) }
		return ValidateEditIntent(*a.Edit)
	case NextIntentKindRunCommand:
		if a.RunCommand == nil { return fmt.Errorf("next intent: kind %q requires runCommand", a.Kind) }
		if strings.TrimSpace(a.RunCommand.Command) == "" { return fmt.Errorf("next intent: runCommand.command is empty") }
		return nil
	case NextIntentKindNavigate:
		if a.Navigate == nil { return fmt.Errorf("next intent: kind %q requires navigate", a.Kind) }
		return ValidateNavigationIntent(*a.Navigate)
	case NextIntentKindRequestContext:
		if a.RequestContext == nil { return fmt.Errorf("next intent: kind %q requires requestContext", a.Kind) }
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
