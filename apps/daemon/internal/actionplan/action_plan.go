package actionplan

import (
	"fmt"
	"strings"
)

// StepKind discriminates one entry in an [ActionPlan].
type StepKind string

const (
	StepKindEdit       StepKind = "edit"
	StepKindRunCommand StepKind = "run_command"
	StepKindNavigate   StepKind = "navigate"
)

// Step is one concrete action in the plan: either apply a structured edit, or
// run a shell command. It is the union of [EditIntent] and [CommandIntent]
// with an explicit kind.
type Step struct {
	Kind StepKind `json:"kind"`

	Edit       *EditIntent       `json:"edit,omitempty"`
	RunCommand *CommandIntent    `json:"runCommand,omitempty"`
	Navigate   *NavigationIntent `json:"navigate,omitempty"`
}

// ActionPlan is structured model output: an ordered list of edits and/or
// commands to run. The daemon supplies file/context snapshots for edit steps
// via edits.EditExecutionContext.
//
// JSON shape matches packages/protocol/schema/action-plan.schema.json.
type ActionPlan struct {
	Steps []Step `json:"steps"`
}

// ValidateStep checks a single step is self-consistent.
func ValidateStep(s Step) error {
	switch s.Kind {
	case StepKindEdit:
		if s.RunCommand != nil {
			return fmt.Errorf("step: kind %q must not include runCommand", s.Kind)
		}
		if s.Navigate != nil {
			return fmt.Errorf("step: kind %q must not include navigate", s.Kind)
		}
		if s.Edit == nil {
			return fmt.Errorf("step: kind %q requires edit", s.Kind)
		}
		return ValidateEditIntent(*s.Edit)
	case StepKindRunCommand:
		if s.Edit != nil {
			return fmt.Errorf("step: kind %q must not include edit", s.Kind)
		}
		if s.Navigate != nil {
			return fmt.Errorf("step: kind %q must not include navigate", s.Kind)
		}
		if s.RunCommand == nil {
			return fmt.Errorf("step: kind %q requires runCommand", s.Kind)
		}
		if strings.TrimSpace(s.RunCommand.Command) == "" {
			return fmt.Errorf("step: runCommand.command is empty")
		}
		return nil
	case StepKindNavigate:
		if s.Edit != nil {
			return fmt.Errorf("step: kind %q must not include edit", s.Kind)
		}
		if s.RunCommand != nil {
			return fmt.Errorf("step: kind %q must not include runCommand", s.Kind)
		}
		if s.Navigate == nil {
			return fmt.Errorf("step: kind %q requires navigate", s.Kind)
		}
		return ValidateNavigationIntent(*s.Navigate)
	default:
		return fmt.Errorf("step: unknown kind %q", s.Kind)
	}
}

// ValidateActionPlan validates every step. Empty steps is allowed (no-op plan).
func ValidateActionPlan(p ActionPlan) error {
	for i := range p.Steps {
		if err := ValidateStep(p.Steps[i]); err != nil {
			return fmt.Errorf("action plan: step %d: %w", i, err)
		}
	}
	return nil
}
