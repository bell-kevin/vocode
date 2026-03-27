package actionplan

import "fmt"

// NextActionKind is the iterative planner response discriminant.
type NextActionKind string

const (
	NextActionKindEdit          NextActionKind = "edit"
	NextActionKindRunCommand    NextActionKind = "run_command"
	NextActionKindNavigate      NextActionKind = "navigate"
	NextActionKindRequestContext NextActionKind = "request_context"
	NextActionKindDone          NextActionKind = "done"
)

type ContextRequestKind string

const (
	ContextRequestKindSymbols     ContextRequestKind = "request_symbols"
	ContextRequestKindFileExcerpt ContextRequestKind = "request_file_excerpt"
	ContextRequestKindUsages      ContextRequestKind = "request_usages"
)

type ContextRequest struct {
	Kind ContextRequestKind `json:"kind"`
	// Minimal, generic fields for now; concrete payload structs can follow.
	Path      string `json:"path,omitempty"`
	Query     string `json:"query,omitempty"`
	SymbolID  string `json:"symbolId,omitempty"`
	MaxResult int    `json:"maxResult,omitempty"`
}

// NextAction is one model turn output in iterative planning mode.
type NextAction struct {
	Kind NextActionKind `json:"kind"`

	Edit           *EditIntent       `json:"edit,omitempty"`
	RunCommand     *CommandIntent    `json:"runCommand,omitempty"`
	Navigate       *NavigationIntent `json:"navigate,omitempty"`
	RequestContext *ContextRequest   `json:"requestContext,omitempty"`
}

func ValidateNextAction(a NextAction) error {
	switch a.Kind {
	case NextActionKindEdit:
		if a.Edit == nil {
			return fmt.Errorf("next action: kind %q requires edit", a.Kind)
		}
		return ValidateEditIntent(*a.Edit)
	case NextActionKindRunCommand:
		if a.RunCommand == nil {
			return fmt.Errorf("next action: kind %q requires runCommand", a.Kind)
		}
		return nil
	case NextActionKindNavigate:
		if a.Navigate == nil {
			return fmt.Errorf("next action: kind %q requires navigate", a.Kind)
		}
		return ValidateNavigationIntent(*a.Navigate)
	case NextActionKindRequestContext:
		if a.RequestContext == nil {
			return fmt.Errorf("next action: kind %q requires requestContext", a.Kind)
		}
		switch a.RequestContext.Kind {
		case ContextRequestKindSymbols, ContextRequestKindFileExcerpt, ContextRequestKindUsages:
			return nil
		default:
			return fmt.Errorf("next action: unknown requestContext kind %q", a.RequestContext.Kind)
		}
	case NextActionKindDone:
		return nil
	default:
		return fmt.Errorf("next action: unknown kind %q", a.Kind)
	}
}

func NextActionToStep(a NextAction) (Step, bool, error) {
	if err := ValidateNextAction(a); err != nil {
		return Step{}, false, err
	}
	switch a.Kind {
	case NextActionKindEdit:
		return Step{Kind: StepKindEdit, Edit: a.Edit}, false, nil
	case NextActionKindRunCommand:
		return Step{Kind: StepKindRunCommand, RunCommand: a.RunCommand}, false, nil
	case NextActionKindNavigate:
		return Step{Kind: StepKindNavigate, Navigate: a.Navigate}, false, nil
	case NextActionKindDone:
		return Step{}, true, nil
	case NextActionKindRequestContext:
		return Step{}, false, fmt.Errorf("request_context not yet executable")
	default:
		return Step{}, false, fmt.Errorf("unknown next action kind %q", a.Kind)
	}
}
